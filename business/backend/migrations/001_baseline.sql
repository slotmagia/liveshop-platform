-- Platform control-plane schema baseline for MySQL 8.
-- The migration runner replays files, so every statement is idempotent.
-- Browser-user identity, sessions, roles, grants, policy and entitlement
-- projections are owned exclusively by liveshop-identity and are not created
-- by a fresh Platform database.

CREATE TABLE IF NOT EXISTS platform_registry_state (
    id         SMALLINT    NOT NULL,
    revision   BIGINT      NOT NULL,
    releases   JSON        NOT NULL,
    active     JSON        NOT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    CONSTRAINT ck_registry_state_singleton CHECK (id = 1),
    CONSTRAINT ck_registry_state_revision CHECK (revision > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO platform_registry_state (id, revision, releases, active)
VALUES (1, 1, JSON_OBJECT(), JSON_OBJECT())
ON DUPLICATE KEY UPDATE id = id;

CREATE TABLE IF NOT EXISTS platform_permission_catalog (
    module_id       VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    permission_code VARCHAR(191) COLLATE utf8mb4_0900_as_cs NOT NULL,
    name            VARCHAR(191) NOT NULL,
    resource_code   VARCHAR(160) COLLATE utf8mb4_0900_as_cs NOT NULL,
    action          VARCHAR(30)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    description     VARCHAR(512) NOT NULL DEFAULT '',
    active          TINYINT(1)   NOT NULL DEFAULT 1,
    release_version VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NULL,
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (permission_code),
    KEY idx_platform_permission_catalog_active (module_id, active, permission_code),
    CONSTRAINT ck_permission_code_shape CHECK (permission_code = CONCAT(resource_code, '.', action))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_setting (
    realm       VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    app_id      BIGINT       NOT NULL,
    merchant_id BIGINT       NOT NULL,
    namespace   VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    value_json  JSON         NOT NULL,
    version     BIGINT       NOT NULL,
    updated_by  VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (realm, app_id, merchant_id, namespace),
    CONSTRAINT ck_setting_realm CHECK (realm IN ('PLATFORM', 'MERCHANT')),
    CONSTRAINT ck_setting_app CHECK (app_id > 0),
    CONSTRAINT ck_setting_merchant CHECK (merchant_id > 0),
    CONSTRAINT ck_setting_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_audit_event (
    event_id      VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    occurred_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    realm         VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    app_id        BIGINT       NOT NULL,
    merchant_id   BIGINT       NOT NULL,
    actor_subject VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    action        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_type VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id   VARCHAR(191) COLLATE utf8mb4_0900_as_cs NOT NULL,
    result        VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    details       JSON         NOT NULL,
    PRIMARY KEY (event_id),
    KEY idx_platform_audit_scope_time (realm, app_id, merchant_id, occurred_at DESC),
    CONSTRAINT ck_audit_result CHECK (result IN ('SUCCEEDED', 'DENIED', 'FAILED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
