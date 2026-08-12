ALTER TABLE platform_identity_account
    ADD COLUMN IF NOT EXISTS failed_login_count INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_platform_identity_account_scope
    ON platform_identity_account(app_id, merchant_id, realm, username);
