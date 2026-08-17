-- Platform-owned object-storage catalogue: drivers stay in code; channels
-- are versioned. Secrets are AES-GCM ciphertext. Physical delete is replaced
-- by RETIRED. app_id=0 is the Admin global partition. Seed one local default.

CREATE TABLE IF NOT EXISTS storage_catalogue (
    app_id      BIGINT      NOT NULL DEFAULT 0,
    revision    BIGINT      NOT NULL DEFAULT 0,
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id),
    CONSTRAINT ck_storage_catalogue_app CHECK (app_id >= 0),
    CONSTRAINT ck_storage_catalogue_revision CHECK (revision >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS storage_channel (
    id                       BIGINT       NOT NULL AUTO_INCREMENT,
    app_id                   BIGINT       NOT NULL DEFAULT 0,
    code                     VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    is_default               TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    version                  BIGINT       NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_storage_channel_app_code (app_id, code),
    KEY idx_storage_channel_app_state (app_id, lifecycle, is_default, enabled, id),
    CONSTRAINT ck_storage_channel_app CHECK (app_id >= 0),
    CONSTRAINT ck_storage_channel_driver CHECK (driver IN ('local','aliyun_oss','cloudflare_r2')),
    CONSTRAINT ck_storage_channel_lifecycle CHECK (lifecycle IN ('ACTIVE','RETIRED')),
    CONSTRAINT ck_storage_channel_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS storage_channel_version (
    app_id                   BIGINT       NOT NULL,
    channel_code             VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    version                  BIGINT       NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    is_default               TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, channel_code, version),
    CONSTRAINT ck_storage_channel_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS storage_command (
    app_id          BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE ascii_bin NOT NULL,
    action          VARCHAR(24)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_kind   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, command_key),
    CONSTRAINT ck_storage_command_action CHECK (action IN ('UPSERT_CHANNEL','ENABLE_CHANNEL','DEFAULT_CHANNEL','RETIRE_CHANNEL')),
    CONSTRAINT ck_storage_command_kind CHECK (resource_kind IN ('CHANNEL')),
    CONSTRAINT ck_storage_command_version CHECK (result_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO storage_catalogue(app_id, revision) VALUES (0, 1)
ON DUPLICATE KEY UPDATE app_id=VALUES(app_id);

INSERT INTO storage_channel(app_id, code, name, driver, enabled, is_default, lifecycle, public_config, credential_key_id, credential_masks, version)
SELECT 0, 'local', '本地磁盘', 'local', 1, 1, 'ACTIVE', '{}', '', '{}', 1
WHERE NOT EXISTS (SELECT 1 FROM storage_channel WHERE app_id=0 AND code='local');

INSERT INTO storage_channel_version(app_id, channel_code, version, name, driver, enabled, is_default, lifecycle, public_config, credential_key_id, credential_masks)
SELECT app_id, code, version, name, driver, enabled, is_default, lifecycle, public_config, credential_key_id, credential_masks
FROM storage_channel
WHERE app_id=0 AND code='local'
  AND NOT EXISTS (SELECT 1 FROM storage_channel_version WHERE app_id=0 AND channel_code='local' AND version=1);
