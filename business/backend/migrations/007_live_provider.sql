-- Platform-owned live Provider catalogue. Current state and every immutable
-- version are written in one transaction; credentials are AES-GCM ciphertext.

CREATE TABLE IF NOT EXISTS live_provider_catalogue (
    app_id      BIGINT      NOT NULL,
    revision    BIGINT      NOT NULL DEFAULT 0,
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id),
    CONSTRAINT ck_live_provider_catalogue_app CHECK (app_id > 0),
    CONSTRAINT ck_live_provider_catalogue_revision CHECK (revision >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS live_provider (
    id                       BIGINT       NOT NULL AUTO_INCREMENT,
    app_id                   BIGINT       NOT NULL,
    code                     VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    kind                     VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    driver                   VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    config_json              JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    ttl_seconds              BIGINT       NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    is_default               TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    health_status            VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT 'UNKNOWN',
    health_message           VARCHAR(512) NOT NULL DEFAULT '',
    health_checked_at        DATETIME(3)  NULL,
    version                  BIGINT       NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_live_provider_app_code (app_id, code),
    KEY idx_live_provider_app_state (app_id, lifecycle, is_default, id),
    CONSTRAINT ck_live_provider_app CHECK (app_id > 0),
    CONSTRAINT ck_live_provider_kind CHECK (kind IN ('RTMP','RTC')),
    CONSTRAINT ck_live_provider_driver CHECK (driver IN ('STATIC','SRS','CLOUD','AGORA','AGORA_MEDIA_GATEWAY')),
    CONSTRAINT ck_live_provider_lifecycle CHECK (lifecycle IN ('ACTIVE','RETIRED')),
    CONSTRAINT ck_live_provider_health CHECK (health_status IN ('UNKNOWN','HEALTHY','UNHEALTHY')),
    CONSTRAINT ck_live_provider_version CHECK (version > 0),
    CONSTRAINT ck_live_provider_ttl CHECK (ttl_seconds BETWEEN 60 AND 2592000),
    CONSTRAINT ck_live_provider_default_enabled CHECK (is_default = 0 OR (enabled = 1 AND lifecycle = 'ACTIVE'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS live_provider_version (
    app_id                   BIGINT       NOT NULL,
    provider_code            VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    version                  BIGINT       NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    kind                     VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    driver                   VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    config_json              JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    ttl_seconds              BIGINT       NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    is_default               TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    health_status            VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    health_message           VARCHAR(512) NOT NULL DEFAULT '',
    health_checked_at        DATETIME(3)  NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, provider_code, version),
    CONSTRAINT ck_live_provider_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS live_provider_command (
    app_id          BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE ascii_bin NOT NULL,
    action          VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    provider_code   VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, command_key),
    CONSTRAINT ck_live_provider_command_action CHECK (action IN ('UPSERT','RETIRE')),
    CONSTRAINT ck_live_provider_command_version CHECK (result_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
