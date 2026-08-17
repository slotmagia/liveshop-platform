-- Platform-owned email catalogue: drivers stay in code; the outbound config
-- is a versioned singleton. Secrets are AES-GCM ciphertext. Templates belong
-- to notification events, not this page. app_id=0 is the Admin global partition.

CREATE TABLE IF NOT EXISTS email_catalogue (
    app_id      BIGINT      NOT NULL DEFAULT 0,
    revision    BIGINT      NOT NULL DEFAULT 0,
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id),
    CONSTRAINT ck_email_catalogue_app CHECK (app_id >= 0),
    CONSTRAINT ck_email_catalogue_revision CHECK (revision >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS email_config (
    id                       BIGINT       NOT NULL AUTO_INCREMENT,
    app_id                   BIGINT       NOT NULL DEFAULT 0,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    version                  BIGINT       NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_email_config_app (app_id),
    CONSTRAINT ck_email_config_app CHECK (app_id >= 0),
    CONSTRAINT ck_email_config_driver CHECK (driver IN ('mock','smtp')),
    CONSTRAINT ck_email_config_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS email_config_version (
    app_id                   BIGINT       NOT NULL,
    version                  BIGINT       NOT NULL,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, version),
    CONSTRAINT ck_email_config_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS email_command (
    app_id          BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    action          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_kind   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id     VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, command_key),
    CONSTRAINT ck_email_command_version CHECK (result_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
