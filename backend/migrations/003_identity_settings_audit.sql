CREATE TABLE IF NOT EXISTS platform_identity_account (
    realm TEXT NOT NULL CHECK (realm IN ('PLATFORM', 'MERCHANT')),
    app_id BIGINT NOT NULL CHECK (app_id > 0),
    merchant_id BIGINT NOT NULL CHECK (merchant_id > 0),
    subject TEXT NOT NULL,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    version BIGINT NOT NULL CHECK (version > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (realm, app_id, merchant_id, subject),
    UNIQUE (realm, app_id, merchant_id, username)
);

CREATE TABLE IF NOT EXISTS platform_identity_session (
    session_id TEXT PRIMARY KEY,
    realm TEXT NOT NULL,
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    subject TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoke_reason TEXT,
    FOREIGN KEY (realm, app_id, merchant_id, subject)
        REFERENCES platform_identity_account (realm, app_id, merchant_id, subject)
);

CREATE TABLE IF NOT EXISTS platform_identity_refresh_token (
    token_hash BYTEA PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES platform_identity_session(session_id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'USED', 'REVOKED')),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_platform_identity_session_subject
    ON platform_identity_session(realm, app_id, merchant_id, subject);
CREATE INDEX IF NOT EXISTS idx_platform_identity_refresh_session
    ON platform_identity_refresh_token(session_id);

CREATE TABLE IF NOT EXISTS platform_setting (
    realm TEXT NOT NULL CHECK (realm IN ('PLATFORM', 'MERCHANT')),
    app_id BIGINT NOT NULL CHECK (app_id > 0),
    merchant_id BIGINT NOT NULL CHECK (merchant_id > 0),
    namespace TEXT NOT NULL,
    value_json JSONB NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (realm, app_id, merchant_id, namespace)
);

CREATE TABLE IF NOT EXISTS platform_audit_event (
    event_id TEXT PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    realm TEXT NOT NULL,
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    actor_subject TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('SUCCEEDED', 'DENIED', 'FAILED')),
    details JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_scope_time
    ON platform_audit_event(realm, app_id, merchant_id, occurred_at DESC);
