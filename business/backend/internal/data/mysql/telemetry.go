package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
)

type TelemetryRepository struct{ db *sql.DB }

var _ telemetry.Repository = (*TelemetryRepository)(nil)

func NewTelemetryRepository(db *sql.DB) *TelemetryRepository { return &TelemetryRepository{db: db} }

func (r *TelemetryRepository) InsertIgnore(ctx context.Context, items []model.Event) ([]bool, error) {
	inserted := make([]bool, len(items))
	if len(items) == 0 {
		return inserted, nil
	}
	err := transaction(ctx, r.db, 8*time.Second, func(ctx context.Context, tx *sql.Tx) error {
		for i, item := range items {
			result, err := tx.ExecContext(ctx, `INSERT IGNORE INTO platform_telemetry_event(
				merchant_id,shop_id,surface,event_id,event_type,event_name,page,component,action,biz_type,biz_id,
				session_id,anonymous_id,subject,client_ts,occurred_at,received_at,schema_version,live_context,props,state,extra,
				user_agent,ip,referer,ad_touch_id,click_id_type)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				item.MerchantID, item.ShopID, item.Surface, item.EventID, item.EventType, item.EventName,
				item.Page, item.Component, item.Action, item.BizType, item.BizID, item.SessionID, item.AnonymousID,
				item.Subject, item.ClientTs, item.OccurredAt, item.ReceivedAt, item.SchemaVersion,
				jsonOrEmpty(item.LiveContext), jsonOrEmpty(item.Props), jsonOrEmpty(item.State), jsonOrEmpty(item.Extra),
				item.UserAgent, item.IP, item.Referer, item.AdTouchID, item.ClickIDType,
			)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			inserted[i] = affected > 0
		}
		return nil
	})
	return inserted, err
}

func (r *TelemetryRepository) List(ctx context.Context, filter model.Filter) (model.Page, error) {
	where, args := telemetryWhere(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_telemetry_event`+where, args...).Scan(&total); err != nil {
		return model.Page{}, err
	}
	query := `SELECT merchant_id,shop_id,surface,event_id,event_type,event_name,page,component,action,biz_type,biz_id,
		session_id,anonymous_id,subject,client_ts,occurred_at,received_at,schema_version,live_context,props,state,extra,
		user_agent,ip,referer,ad_touch_id,click_id_type,created_at
		FROM platform_telemetry_event` + where + ` ORDER BY client_ts DESC, event_id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, append(append([]any{}, args...), filter.PageSize, filter.Offset())...)
	if err != nil {
		return model.Page{}, err
	}
	defer rows.Close()
	items := make([]model.Event, 0)
	for rows.Next() {
		var item model.Event
		if err := rows.Scan(
			&item.MerchantID, &item.ShopID, &item.Surface, &item.EventID, &item.EventType, &item.EventName,
			&item.Page, &item.Component, &item.Action, &item.BizType, &item.BizID, &item.SessionID, &item.AnonymousID,
			&item.Subject, &item.ClientTs, &item.OccurredAt, &item.ReceivedAt, &item.SchemaVersion,
			&item.LiveContext, &item.Props, &item.State, &item.Extra,
			&item.UserAgent, &item.IP, &item.Referer, &item.AdTouchID, &item.ClickIDType, &item.CreatedAt,
		); err != nil {
			return model.Page{}, err
		}
		items = append(items, item)
	}
	return model.Page{Items: items, Total: total}, rows.Err()
}

func telemetryWhere(filter model.Filter) (string, []any) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	if filter.MerchantID > 0 {
		clauses = append(clauses, "merchant_id=?")
		args = append(args, filter.MerchantID)
	}
	if filter.ShopID > 0 {
		clauses = append(clauses, "shop_id=?")
		args = append(args, filter.ShopID)
	}
	if filter.Surface != "" {
		clauses = append(clauses, "surface=?")
		args = append(args, filter.Surface)
	}
	if filter.EventName != "" {
		clauses = append(clauses, "event_name=?")
		args = append(args, filter.EventName)
	}
	if filter.EventType != "" {
		clauses = append(clauses, "event_type=?")
		args = append(args, filter.EventType)
	}
	if filter.Subject != "" {
		clauses = append(clauses, "subject=?")
		args = append(args, filter.Subject)
	}
	if filter.AnonymousID != "" {
		clauses = append(clauses, "anonymous_id=?")
		args = append(args, filter.AnonymousID)
	}
	if filter.StartMs > 0 {
		clauses = append(clauses, "client_ts>=?")
		args = append(args, filter.StartMs)
	}
	if filter.EndMs > 0 {
		clauses = append(clauses, "client_ts<=?")
		args = append(args, filter.EndMs)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func jsonOrEmpty(value json.RawMessage) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}
	return value
}
