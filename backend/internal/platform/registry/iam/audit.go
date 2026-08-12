package iam

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strconv"
)

type auditActor struct {
	realm   string
	subject string
}

type auditActorKey struct{}

// WithAuditActor binds the verified operator to IAM mutations. PostgreSQL IAM
// writes consume it inside their own transaction; callers cannot supply actor
// fields in the request body.
func WithAuditActor(ctx context.Context, realm, subject string) context.Context {
	return context.WithValue(ctx, auditActorKey{}, auditActor{realm: realm, subject: subject})
}

func auditIAMMutation(ctx context.Context, tx *sql.Tx, tenant Tenant, action, resourceType string, resourceID int64, details any) error {
	actor, ok := ctx.Value(auditActorKey{}).(auditActor)
	if !ok || actor.realm == "" || actor.subject == "" {
		return nil
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,app_id,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,'SUCCEEDED',$9)`, base64.RawURLEncoding.EncodeToString(random), actor.realm, tenant.AppID, tenant.MerchantID, actor.subject, action, resourceType, strconv.FormatInt(resourceID, 10), payload)
	return err
}

func auditIAMSubjectMutation(ctx context.Context, tx *sql.Tx, tenant Tenant, action, subject string, details any) error {
	actor, ok := ctx.Value(auditActorKey{}).(auditActor)
	if !ok || actor.realm == "" || actor.subject == "" {
		return nil
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,app_id,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
		VALUES($1,$2,$3,$4,$5,$6,'platform.subject',$7,'SUCCEEDED',$8)`, base64.RawURLEncoding.EncodeToString(random), actor.realm, tenant.AppID, tenant.MerchantID, actor.subject, action, subject, payload)
	return err
}
