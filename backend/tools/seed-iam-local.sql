INSERT INTO platform_department (app_id, merchant_id, department_id, parent_department_id, name, status, version)
VALUES
    (1001, 2001, 10, NULL, 'Headquarters', 'ACTIVE', 1),
    (1001, 2001, 20, 10, 'Live Commerce', 'ACTIVE', 1)
ON CONFLICT (app_id, merchant_id, department_id) DO NOTHING;

INSERT INTO platform_role (app_id, merchant_id, role_id, name, status, is_super_admin, version)
VALUES (1001, 2001, 1, 'Super Administrator', 'ACTIVE', TRUE, 1)
ON CONFLICT (app_id, merchant_id, role_id) DO NOTHING;

INSERT INTO platform_subject_role (app_id, merchant_id, subject, role_id)
VALUES
    (1001, 2001, 'local-user', 1),
    (1001, 2001, 'platform-admin', 1),
    (1001, 2001, 'merchant-admin', 1)
ON CONFLICT DO NOTHING;

INSERT INTO platform_subject_department (app_id, merchant_id, subject, department_id, is_primary)
VALUES (1001, 2001, 'local-user', 10, TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO platform_iam_revision (app_id, merchant_id, revision)
VALUES (1001, 2001, 1)
ON CONFLICT DO NOTHING;

INSERT INTO platform_identity_account (realm,app_id,merchant_id,subject,username,password_hash,status,version)
VALUES
    ('PLATFORM',1001,2001,'platform-admin','admin','$2b$12$1vjQrZh1vrhLpRRJqNQ3WOTqPGmTBoO0guWWygdPLctwsZGmttrYu','ACTIVE',1),
    ('MERCHANT',1001,2001,'merchant-admin','merch@sufeipay.com','$2b$12$Oo0x.9lbWm8zQ5WKrgwL6.ZJTZ1bR2RNL5xoNS1ngvR8h5bc3FZRS','ACTIVE',1)
ON CONFLICT (realm,app_id,merchant_id,subject) DO UPDATE
SET username=EXCLUDED.username,password_hash=EXCLUDED.password_hash,status=EXCLUDED.status,updated_at=NOW();

UPDATE platform_identity_session
SET revoked_at=COALESCE(revoked_at,NOW()), revoke_reason=COALESCE(revoke_reason,'LOCAL_SEED_CREDENTIAL_CHANGED')
WHERE realm='PLATFORM' AND app_id=1001 AND merchant_id=2001 AND subject='platform-admin' AND revoked_at IS NULL;

UPDATE platform_identity_refresh_token
SET status='REVOKED'
WHERE status='ACTIVE' AND session_id IN (
    SELECT session_id FROM platform_identity_session
    WHERE realm='PLATFORM' AND app_id=1001 AND merchant_id=2001 AND subject='platform-admin'
);

UPDATE platform_identity_session
SET revoked_at=COALESCE(revoked_at,NOW()), revoke_reason=COALESCE(revoke_reason,'LOCAL_SEED_CREDENTIAL_CHANGED')
WHERE realm='MERCHANT' AND app_id=1001 AND merchant_id=2001 AND subject='merchant-admin' AND revoked_at IS NULL;

UPDATE platform_identity_refresh_token
SET status='REVOKED'
WHERE status='ACTIVE' AND session_id IN (
    SELECT session_id FROM platform_identity_session
    WHERE realm='MERCHANT' AND app_id=1001 AND merchant_id=2001 AND subject='merchant-admin'
);
