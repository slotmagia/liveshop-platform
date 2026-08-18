package mysql

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
)

const notifyTransactionTimeout = 5 * time.Second

type NotificationRepository struct{ db *sql.DB }

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

var _ notification.Repository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Project(ctx context.Context, revision uint64, declarations []notifymodel.Declaration) error {
	return transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		seen := make([]string, 0, len(declarations))
		for _, declaration := range declarations {
			variables, err := json.Marshal(declaration.Variables)
			if err != nil {
				return err
			}
			channels, err := json.Marshal(declaration.AllowedChannels)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_notify_event(event_key,module_id,module_name,operation_id,title,variables,allowed_channels,default_dispatch,dispatchable,registry_revision)
				VALUES(?,?,?,?,?,?,?,?,1,?) AS incoming
				ON DUPLICATE KEY UPDATE module_id=incoming.module_id, module_name=incoming.module_name, operation_id=incoming.operation_id, title=incoming.title,
					variables=incoming.variables, allowed_channels=incoming.allowed_channels, default_dispatch=incoming.default_dispatch, dispatchable=1, registry_revision=incoming.registry_revision, updated_at=CURRENT_TIMESTAMP(3)`,
				declaration.EventKey, declaration.ModuleID, declaration.ModuleName, declaration.OperationID, declaration.Title, variables, channels, declaration.DefaultDispatch, revision); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO platform_notify_event_policy(event_key,dispatch_mode,delay_seconds,channel_sms,channel_email,channel_in_app,template_sms,template_email,template_in_app,version,command_key)
				VALUES(?,?,0,?,?,?,?,?,?,1,?)`,
				declaration.EventKey, declaration.DefaultDispatch,
				containsChannel(declaration.AllowedChannels, notifymodel.ChannelSMS),
				containsChannel(declaration.AllowedChannels, notifymodel.ChannelEmail),
				containsChannel(declaration.AllowedChannels, notifymodel.ChannelInApp),
				"", "", "",
				projectionCommandKey(declaration.EventKey)); err != nil {
				return err
			}
			seen = append(seen, declaration.EventKey)
		}
		if len(seen) == 0 {
			_, err := tx.ExecContext(ctx, `UPDATE platform_notify_event SET dispatchable=0, registry_revision=?, updated_at=CURRENT_TIMESTAMP(3) WHERE dispatchable=1`, revision)
			return err
		}
		args := make([]any, 0, len(seen)+1)
		args = append(args, revision)
		placeholders := make([]string, 0, len(seen))
		for _, key := range seen {
			placeholders = append(placeholders, "?")
			args = append(args, key)
		}
		_, err := tx.ExecContext(ctx, `UPDATE platform_notify_event SET dispatchable=0, registry_revision=?, updated_at=CURRENT_TIMESTAMP(3) WHERE event_key NOT IN (`+strings.Join(placeholders, ",")+`)`, args...)
		return err
	})
}

func (r *NotificationRepository) ListEvents(ctx context.Context, filter notifymodel.EventFilter) ([]notifymodel.Event, error) {
	query := `SELECT e.event_key,e.module_id,e.module_name,e.operation_id,e.title,e.variables,e.allowed_channels,e.default_dispatch,e.dispatchable,e.registry_revision,e.updated_at,
		COALESCE(p.dispatch_mode,e.default_dispatch),COALESCE(p.delay_seconds,0),COALESCE(p.channel_sms,0),COALESCE(p.channel_email,0),COALESCE(p.channel_in_app,0),
		COALESCE(p.template_sms,''),COALESCE(p.template_email,''),COALESCE(p.template_in_app,''),COALESCE(p.version,0),COALESCE(p.updated_at,e.updated_at)
		FROM platform_notify_event e LEFT JOIN platform_notify_event_policy p ON p.event_key=e.event_key WHERE e.dispatchable=1`
	args := []any{}
	if filter.Module != "" {
		query += ` AND e.module_id=?`
		args = append(args, filter.Module)
	}
	if filter.Keyword != "" {
		query += ` AND (e.title LIKE ? OR e.event_key LIKE ? OR e.operation_id LIKE ?)`
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword, keyword)
	}
	if filter.Channel == notifymodel.ChannelSMS {
		query += ` AND p.channel_sms=1`
	} else if filter.Channel == notifymodel.ChannelEmail {
		query += ` AND p.channel_email=1`
	} else if filter.Channel == notifymodel.ChannelInApp {
		query += ` AND p.channel_in_app=1`
	}
	query += ` ORDER BY e.module_id,e.operation_id,e.event_key`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notifymodel.Event, 0)
	for rows.Next() {
		item, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetEvent(ctx context.Context, eventKey string) (notifymodel.Event, error) {
	row := r.db.QueryRowContext(ctx, `SELECT e.event_key,e.module_id,e.module_name,e.operation_id,e.title,e.variables,e.allowed_channels,e.default_dispatch,e.dispatchable,e.registry_revision,e.updated_at,
		COALESCE(p.dispatch_mode,e.default_dispatch),COALESCE(p.delay_seconds,0),COALESCE(p.channel_sms,0),COALESCE(p.channel_email,0),COALESCE(p.channel_in_app,0),
		COALESCE(p.template_sms,''),COALESCE(p.template_email,''),COALESCE(p.template_in_app,''),COALESCE(p.version,0),COALESCE(p.updated_at,e.updated_at)
		FROM platform_notify_event e LEFT JOIN platform_notify_event_policy p ON p.event_key=e.event_key WHERE e.event_key=?`, eventKey)
	item, err := scanEvent(row)
	if err == sql.ErrNoRows {
		return notifymodel.Event{}, notifymodel.ErrNotFound
	}
	return item, err
}

func (r *NotificationRepository) ReplacePolicy(ctx context.Context, scope notifymodel.Scope, input notifymodel.ReplacePolicy, requestHash string) (notifymodel.Policy, error) {
	var output notifymodel.Policy
	err := transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayNotifyCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			item, err := loadPolicy(ctx, tx, input.EventKey)
			if err != nil {
				return err
			}
			if item.Version != replay {
				return notifymodel.ErrConflict
			}
			output = item
			return nil
		}
		current, err := lockPolicy(ctx, tx, input.EventKey)
		if err != nil {
			return err
		}
		if current.Version != input.ExpectedVersion {
			return notifymodel.ErrConflict
		}
		next := notifymodel.Policy{EventKey: input.EventKey, DispatchMode: input.DispatchMode, DelaySeconds: input.DelaySeconds, Channels: input.Channels, Version: current.Version + 1}
		sms, email, inApp, smsCode, emailCode, inAppCode := policyColumns(next.Channels)
		if _, err := tx.ExecContext(ctx, `UPDATE platform_notify_event_policy SET dispatch_mode=?, delay_seconds=?, channel_sms=?, channel_email=?, channel_in_app=?, template_sms=?, template_email=?, template_in_app=?, version=?, command_key=?, updated_at=CURRENT_TIMESTAMP(3) WHERE event_key=? AND version=?`,
			next.DispatchMode, next.DelaySeconds, sms, email, inApp, smsCode, emailCode, inAppCode, next.Version, input.CommandKey, input.EventKey, input.ExpectedVersion); err != nil {
			return err
		}
		if err := insertNotifyCommand(ctx, tx, input.CommandKey, requestHash, "POLICY", "policy", input.EventKey, next.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "notify.policy.replace", ResourceType: "platform.notify.event", ResourceID: input.EventKey, Details: map[string]any{"version": next.Version, "mode": next.DispatchMode}}); err != nil {
			return err
		}
		loaded, err := loadPolicy(ctx, tx, input.EventKey)
		output = loaded
		return err
	})
	return output, err
}

func (r *NotificationRepository) ListLibraryTemplates(ctx context.Context, filter notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error) {
	query := `SELECT code,channel,COALESCE(text_template,''),COALESCE(subject,''),COALESCE(body_html,''),COALESCE(title,''),COALESCE(body,''),variables,lifecycle,version,updated_at FROM platform_notify_template_library WHERE 1=1`
	args := []any{}
	if filter.Channel != "" {
		query += ` AND channel=?`
		args = append(args, filter.Channel)
	}
	if filter.Keyword != "" {
		query += ` AND (code LIKE ? OR COALESCE(title,'') LIKE ? OR COALESCE(subject,'') LIKE ?)`
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword, keyword)
	}
	query += ` ORDER BY channel, code`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notifymodel.LibraryTemplate, 0)
	for rows.Next() {
		item, err := scanLibraryTemplate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetLibraryTemplate(ctx context.Context, code string) (notifymodel.LibraryTemplate, error) {
	row := r.db.QueryRowContext(ctx, `SELECT code,channel,COALESCE(text_template,''),COALESCE(subject,''),COALESCE(body_html,''),COALESCE(title,''),COALESCE(body,''),variables,lifecycle,version,updated_at FROM platform_notify_template_library WHERE code=?`, code)
	item, err := scanLibraryTemplate(row)
	if err == sql.ErrNoRows {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrNotFound
	}
	return item, err
}

func (r *NotificationRepository) UpsertLibraryTemplate(ctx context.Context, scope notifymodel.Scope, input notifymodel.UpsertLibraryTemplate, requestHash string) (notifymodel.LibraryTemplate, error) {
	var output notifymodel.LibraryTemplate
	err := transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, found, err := replayNotifyCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			item, err := loadLibraryTemplate(ctx, tx, input.Code)
			output = item
			return err
		}
		current, err := lockLibraryTemplate(ctx, tx, input.Code)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			current = notifymodel.LibraryTemplate{Code: input.Code}
		}
		if current.Version != input.ExpectedVersion {
			return notifymodel.ErrConflict
		}
		if current.Lifecycle == notifymodel.TemplateRetired {
			return notifymodel.ErrConflict
		}
		variables, err := json.Marshal(input.Variables)
		if err != nil {
			return err
		}
		nextVersion := current.Version + 1
		if _, err := tx.ExecContext(ctx, `INSERT INTO platform_notify_template_library(code,channel,text_template,subject,body_html,title,body,variables,lifecycle,version,command_key)
			VALUES(?,?,?,?,?,?,?,?,?,?,?) AS incoming
			ON DUPLICATE KEY UPDATE channel=incoming.channel, text_template=incoming.text_template, subject=incoming.subject, body_html=incoming.body_html, title=incoming.title, body=incoming.body, variables=incoming.variables, lifecycle='ACTIVE', version=incoming.version, command_key=incoming.command_key, updated_at=CURRENT_TIMESTAMP(3)`,
			input.Code, input.Channel, input.TextTemplate, input.Subject, input.BodyHTML, input.Title, input.Body, variables, notifymodel.TemplateActive, nextVersion, input.CommandKey); err != nil {
			return err
		}
		if err := insertNotifyCommand(ctx, tx, input.CommandKey, requestHash, "LIBRARY", "template", input.Code, nextVersion); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "notify.template.upsert", ResourceType: "platform.notify.template", ResourceID: input.Code, Details: map[string]any{"channel": input.Channel, "version": nextVersion}}); err != nil {
			return err
		}
		item, err := loadLibraryTemplate(ctx, tx, input.Code)
		output = item
		return err
	})
	return output, err
}

func (r *NotificationRepository) RetireLibraryTemplate(ctx context.Context, scope notifymodel.Scope, input notifymodel.RetireLibraryTemplate, requestHash string) (notifymodel.LibraryTemplate, error) {
	var output notifymodel.LibraryTemplate
	err := transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, found, err := replayNotifyCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			item, err := loadLibraryTemplate(ctx, tx, input.Code)
			output = item
			return err
		}
		current, err := lockLibraryTemplate(ctx, tx, input.Code)
		if err != nil {
			if err == sql.ErrNoRows {
				return notifymodel.ErrNotFound
			}
			return err
		}
		if current.Version != input.ExpectedVersion {
			return notifymodel.ErrConflict
		}
		nextVersion := current.Version + 1
		if _, err := tx.ExecContext(ctx, `UPDATE platform_notify_template_library SET lifecycle='RETIRED', version=?, command_key=?, updated_at=CURRENT_TIMESTAMP(3) WHERE code=? AND version=?`,
			nextVersion, input.CommandKey, input.Code, input.ExpectedVersion); err != nil {
			return err
		}
		if err := insertNotifyCommand(ctx, tx, input.CommandKey, requestHash, "RETIRE", "template", input.Code, nextVersion); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "notify.template.retire", ResourceType: "platform.notify.template", ResourceID: input.Code, Details: map[string]any{"version": nextVersion}}); err != nil {
			return err
		}
		item, err := loadLibraryTemplate(ctx, tx, input.Code)
		output = item
		return err
	})
	return output, err
}

func (r *NotificationRepository) GetInAppConfig(ctx context.Context) (notifymodel.InAppConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT driver,enabled,version,updated_at FROM platform_notify_inapp_config WHERE id=1`)
	item, err := scanInAppConfig(row)
	if err == sql.ErrNoRows {
		return notifymodel.InAppConfig{Driver: notifymodel.InAppDriver, Enabled: true}, nil
	}
	return item, err
}

func (r *NotificationRepository) ReplaceInAppConfig(ctx context.Context, scope notifymodel.Scope, input notifymodel.ReplaceInAppConfig, requestHash string) (notifymodel.InAppConfig, error) {
	var output notifymodel.InAppConfig
	err := transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, found, err := replayNotifyCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			item, err := loadInAppConfig(ctx, tx)
			output = item
			return err
		}
		current, err := lockInAppConfig(ctx, tx)
		if err != nil {
			return err
		}
		if current.Version != input.ExpectedVersion {
			return notifymodel.ErrConflict
		}
		nextVersion := current.Version + 1
		if _, err := tx.ExecContext(ctx, `UPDATE platform_notify_inapp_config SET enabled=?, version=?, command_key=?, updated_at=CURRENT_TIMESTAMP(3) WHERE id=1 AND version=?`,
			input.Enabled, nextVersion, input.CommandKey, input.ExpectedVersion); err != nil {
			return err
		}
		if err := insertNotifyCommand(ctx, tx, input.CommandKey, requestHash, "INAPP", "inapp", "inbox", nextVersion); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "notify.inapp.replace", ResourceType: "platform.notify.inapp", ResourceID: "inbox", Details: map[string]any{"enabled": input.Enabled, "version": nextVersion}}); err != nil {
			return err
		}
		item, err := loadInAppConfig(ctx, tx)
		output = item
		return err
	})
	return output, err
}

func (r *NotificationRepository) ListDeliveries(ctx context.Context, eventKey string) ([]notifymodel.Delivery, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT delivery_id,delivery_key,event_key,channel,merchant_id,shop_id,status,recipient,variables,request_hash,not_before,last_error,attempt_count,created_at,updated_at
		FROM platform_notify_delivery WHERE event_key=? ORDER BY created_at DESC LIMIT 200`, eventKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notifymodel.Delivery, 0)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetDelivery(ctx context.Context, deliveryID string) (notifymodel.Delivery, error) {
	row := r.db.QueryRowContext(ctx, `SELECT delivery_id,delivery_key,event_key,channel,merchant_id,shop_id,status,recipient,variables,request_hash,not_before,last_error,attempt_count,created_at,updated_at
		FROM platform_notify_delivery WHERE delivery_id=?`, deliveryID)
	item, err := scanDelivery(row)
	if err == sql.ErrNoRows {
		return notifymodel.Delivery{}, notifymodel.ErrNotFound
	}
	return item, err
}

func (r *NotificationRepository) PrepareDeliveries(ctx context.Context, input notifymodel.DispatchInput, event notifymodel.Event, channels []notifymodel.Channel, requestHash string) ([]notifymodel.DeliveryResult, []notifymodel.Delivery, error) {
	var results []notifymodel.DeliveryResult
	var pending []notifymodel.Delivery
	err := transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		results = make([]notifymodel.DeliveryResult, 0, len(channels))
		pending = make([]notifymodel.Delivery, 0, len(channels))
		for _, channel := range channels {
			existing, err := lockDeliveryByKey(ctx, tx, input.DeliveryKey, channel)
			if err == nil {
				if existing.RequestHash != requestHash {
					return notifymodel.ErrConflict
				}
				results = append(results, notifymodel.DeliveryResult{DeliveryID: existing.DeliveryID, Channel: channel, Status: existing.Status, Deduped: true})
				pending = append(pending, existing)
				continue
			}
			if err != sql.ErrNoRows {
				return err
			}
			status := notifymodel.StatusPending
			if event.Policy.DispatchMode == notifymodel.ModeScheduled {
				status = notifymodel.StatusScheduled
			}
			if event.Policy.DispatchMode == notifymodel.ModeSync {
				status = notifymodel.StatusSending
			}
			recipient, _ := notifymodel.RecipientFor(channel, input.Recipients)
			variables, err := json.Marshal(input.Variables)
			if err != nil {
				return err
			}
			item := notifymodel.Delivery{
				DeliveryID: newDeliveryID(), DeliveryKey: input.DeliveryKey, EventKey: event.EventKey, Channel: channel,
				MerchantID: input.MerchantID, ShopID: input.ShopID, Status: status, Recipient: recipient, Variables: input.Variables,
				RequestHash: requestHash, NotBefore: input.NotBefore, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			}
			var notBefore any
			if !input.NotBefore.IsZero() {
				notBefore = input.NotBefore.UTC()
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_notify_delivery(delivery_id,delivery_key,event_key,channel,merchant_id,shop_id,status,recipient,variables,request_hash,not_before)
				VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
				item.DeliveryID, item.DeliveryKey, item.EventKey, item.Channel, item.MerchantID, item.ShopID, item.Status, item.Recipient, variables, item.RequestHash, notBefore); err != nil {
				return err
			}
			results = append(results, notifymodel.DeliveryResult{DeliveryID: item.DeliveryID, Channel: channel, Status: status})
			pending = append(pending, item)
		}
		return nil
	})
	return results, pending, err
}

func (r *NotificationRepository) MarkSending(ctx context.Context, deliveryID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE platform_notify_delivery SET status='SENDING', updated_at=CURRENT_TIMESTAMP(3) WHERE delivery_id=? AND status IN ('PENDING','SCHEDULED','SENDING','UNKNOWN')`, deliveryID)
	return err
}

func (r *NotificationRepository) CompleteDelivery(ctx context.Context, deliveryID string, status notifymodel.DeliveryStatus, detail string, inbox *notifymodel.InboxMessage) error {
	return transaction(ctx, r.db, notifyTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		var attempt int
		if err := tx.QueryRowContext(ctx, `SELECT attempt_count FROM platform_notify_delivery WHERE delivery_id=? FOR UPDATE`, deliveryID).Scan(&attempt); err != nil {
			if err == sql.ErrNoRows {
				return notifymodel.ErrNotFound
			}
			return err
		}
		attempt++
		if _, err := tx.ExecContext(ctx, `UPDATE platform_notify_delivery SET status=?, last_error=?, attempt_count=?, updated_at=CURRENT_TIMESTAMP(3) WHERE delivery_id=?`, status, truncate(detail, 512), attempt, deliveryID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO platform_notify_attempt(delivery_id,attempt_no,status,detail) VALUES(?,?,?,?)`, deliveryID, attempt, status, truncate(detail, 512)); err != nil {
			return err
		}
		if inbox != nil {
			if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO platform_notify_inbox(merchant_id,shop_id,subject,delivery_id,title,body) VALUES(?,?,?,?,?,?)`,
				inbox.MerchantID, inbox.ShopID, inbox.Subject, inbox.DeliveryID, inbox.Title, inbox.Body); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *NotificationRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]notifymodel.Delivery, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT delivery_id,delivery_key,event_key,channel,merchant_id,shop_id,status,recipient,variables,request_hash,not_before,last_error,attempt_count,created_at,updated_at
		FROM platform_notify_delivery
		WHERE attempt_count < ? AND (
			status='PENDING' OR status='UNKNOWN' OR (status='SCHEDULED' AND (not_before IS NULL OR not_before<=?))
		)
		ORDER BY created_at ASC LIMIT ?`, notifymodel.MaxAttempts, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notifymodel.Delivery, 0)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanEvent(row scanner) (notifymodel.Event, error) {
	var item notifymodel.Event
	var variables, channels []byte
	var sms, email, inApp bool
	var smsCode, emailCode, inAppCode string
	var policyUpdated time.Time
	if err := row.Scan(&item.EventKey, &item.ModuleID, &item.ModuleName, &item.OperationID, &item.Title, &variables, &channels, &item.DefaultDispatch, &item.Dispatchable, &item.RegistryRevision, &item.UpdatedAt,
		&item.Policy.DispatchMode, &item.Policy.DelaySeconds, &sms, &email, &inApp, &smsCode, &emailCode, &inAppCode, &item.Policy.Version, &policyUpdated); err != nil {
		return notifymodel.Event{}, err
	}
	_ = json.Unmarshal(variables, &item.Variables)
	_ = json.Unmarshal(channels, &item.AllowedChannels)
	item.Policy.EventKey = item.EventKey
	item.Policy.Channels = map[notifymodel.Channel]notifymodel.ChannelPolicy{
		notifymodel.ChannelSMS:   {Enabled: sms, TemplateCode: smsCode},
		notifymodel.ChannelEmail: {Enabled: email, TemplateCode: emailCode},
		notifymodel.ChannelInApp: {Enabled: inApp, TemplateCode: inAppCode},
	}
	item.Policy.UpdatedAt = policyUpdated
	if item.Variables == nil {
		item.Variables = []string{}
	}
	return item, nil
}

func scanLibraryTemplate(row scanner) (notifymodel.LibraryTemplate, error) {
	var item notifymodel.LibraryTemplate
	var variables []byte
	if err := row.Scan(&item.Code, &item.Channel, &item.TextTemplate, &item.Subject, &item.BodyHTML, &item.Title, &item.Body, &variables, &item.Lifecycle, &item.Version, &item.UpdatedAt); err != nil {
		return notifymodel.LibraryTemplate{}, err
	}
	_ = json.Unmarshal(variables, &item.Variables)
	if item.Variables == nil {
		item.Variables = []string{}
	}
	return item, nil
}

func scanInAppConfig(row scanner) (notifymodel.InAppConfig, error) {
	var item notifymodel.InAppConfig
	if err := row.Scan(&item.Driver, &item.Enabled, &item.Version, &item.UpdatedAt); err != nil {
		return notifymodel.InAppConfig{}, err
	}
	return item, nil
}

func scanDelivery(row scanner) (notifymodel.Delivery, error) {
	var item notifymodel.Delivery
	var variables []byte
	var notBefore sql.NullTime
	if err := row.Scan(&item.DeliveryID, &item.DeliveryKey, &item.EventKey, &item.Channel, &item.MerchantID, &item.ShopID, &item.Status, &item.Recipient, &variables, &item.RequestHash, &notBefore, &item.LastError, &item.AttemptCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return notifymodel.Delivery{}, err
	}
	_ = json.Unmarshal(variables, &item.Variables)
	if notBefore.Valid {
		item.NotBefore = notBefore.Time
	}
	return item, nil
}

func lockPolicy(ctx context.Context, tx *sql.Tx, eventKey string) (notifymodel.Policy, error) {
	row := tx.QueryRowContext(ctx, `SELECT event_key,dispatch_mode,delay_seconds,channel_sms,channel_email,channel_in_app,template_sms,template_email,template_in_app,version,updated_at FROM platform_notify_event_policy WHERE event_key=? FOR UPDATE`, eventKey)
	var item notifymodel.Policy
	var sms, email, inApp bool
	var smsCode, emailCode, inAppCode string
	if err := row.Scan(&item.EventKey, &item.DispatchMode, &item.DelaySeconds, &sms, &email, &inApp, &smsCode, &emailCode, &inAppCode, &item.Version, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return notifymodel.Policy{}, notifymodel.ErrNotFound
		}
		return notifymodel.Policy{}, err
	}
	item.Channels = map[notifymodel.Channel]notifymodel.ChannelPolicy{
		notifymodel.ChannelSMS:   {Enabled: sms, TemplateCode: smsCode},
		notifymodel.ChannelEmail: {Enabled: email, TemplateCode: emailCode},
		notifymodel.ChannelInApp: {Enabled: inApp, TemplateCode: inAppCode},
	}
	return item, nil
}

func loadPolicy(ctx context.Context, tx *sql.Tx, eventKey string) (notifymodel.Policy, error) {
	return lockPolicy(ctx, tx, eventKey)
}

func lockLibraryTemplate(ctx context.Context, tx *sql.Tx, code string) (notifymodel.LibraryTemplate, error) {
	row := tx.QueryRowContext(ctx, `SELECT code,channel,COALESCE(text_template,''),COALESCE(subject,''),COALESCE(body_html,''),COALESCE(title,''),COALESCE(body,''),variables,lifecycle,version,updated_at FROM platform_notify_template_library WHERE code=? FOR UPDATE`, code)
	return scanLibraryTemplate(row)
}

func loadLibraryTemplate(ctx context.Context, tx *sql.Tx, code string) (notifymodel.LibraryTemplate, error) {
	row := tx.QueryRowContext(ctx, `SELECT code,channel,COALESCE(text_template,''),COALESCE(subject,''),COALESCE(body_html,''),COALESCE(title,''),COALESCE(body,''),variables,lifecycle,version,updated_at FROM platform_notify_template_library WHERE code=?`, code)
	return scanLibraryTemplate(row)
}

func lockInAppConfig(ctx context.Context, tx *sql.Tx) (notifymodel.InAppConfig, error) {
	row := tx.QueryRowContext(ctx, `SELECT driver,enabled,version,updated_at FROM platform_notify_inapp_config WHERE id=1 FOR UPDATE`)
	item, err := scanInAppConfig(row)
	if err == sql.ErrNoRows {
		return notifymodel.InAppConfig{}, notifymodel.ErrNotFound
	}
	return item, err
}

func loadInAppConfig(ctx context.Context, tx *sql.Tx) (notifymodel.InAppConfig, error) {
	row := tx.QueryRowContext(ctx, `SELECT driver,enabled,version,updated_at FROM platform_notify_inapp_config WHERE id=1`)
	item, err := scanInAppConfig(row)
	if err == sql.ErrNoRows {
		return notifymodel.InAppConfig{}, notifymodel.ErrNotFound
	}
	return item, err
}

func policyColumns(channels map[notifymodel.Channel]notifymodel.ChannelPolicy) (sms, email, inApp bool, smsCode, emailCode, inAppCode string) {
	sms = channels[notifymodel.ChannelSMS].Enabled
	email = channels[notifymodel.ChannelEmail].Enabled
	inApp = channels[notifymodel.ChannelInApp].Enabled
	smsCode = channels[notifymodel.ChannelSMS].TemplateCode
	emailCode = channels[notifymodel.ChannelEmail].TemplateCode
	inAppCode = channels[notifymodel.ChannelInApp].TemplateCode
	return
}

func lockDeliveryByKey(ctx context.Context, tx *sql.Tx, deliveryKey string, channel notifymodel.Channel) (notifymodel.Delivery, error) {
	row := tx.QueryRowContext(ctx, `SELECT delivery_id,delivery_key,event_key,channel,merchant_id,shop_id,status,recipient,variables,request_hash,not_before,last_error,attempt_count,created_at,updated_at
		FROM platform_notify_delivery WHERE delivery_key=? AND channel=? FOR UPDATE`, deliveryKey, channel)
	return scanDelivery(row)
}

func replayNotifyCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (int64, bool, error) {
	var storedHash string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,result_version FROM platform_notify_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &version)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if storedHash != requestHash {
		return 0, false, notifymodel.ErrConflict
	}
	return version, true, nil
}

func insertNotifyCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, action, kind, resourceID string, version int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO platform_notify_command(command_key,request_hash,action,resource_kind,resource_id,result_version) VALUES(?,?,?,?,?,?)`,
		commandKey, requestHash, action, kind, resourceID, version)
	return err
}

func containsChannel(channels []notifymodel.Channel, target notifymodel.Channel) bool {
	for _, channel := range channels {
		if channel == target {
			return true
		}
	}
	return false
}

func projectionCommandKey(eventKey string) string {
	sum := sha256.Sum256([]byte(eventKey))
	return "proj-" + hex.EncodeToString(sum[:8])
}

func newDeliveryID() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
