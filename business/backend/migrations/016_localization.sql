-- Platform localization: MT provider singleton, source snapshots from module
-- events, published overlays. Platform never JOINs Catalog/Live/Trade tables.

CREATE TABLE IF NOT EXISTS i18n_config (
    id                       BIGINT       NOT NULL PRIMARY KEY,
    provider                 VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT 'noop',
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    api_key_set              TINYINT(1)   NOT NULL DEFAULT 0,
    version                  BIGINT       NOT NULL DEFAULT 0,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT ck_i18n_config_id CHECK (id = 1),
    CONSTRAINT ck_i18n_config_provider CHECK (provider IN ('noop','deepl','google')),
    CONSTRAINT ck_i18n_config_version CHECK (version >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO i18n_config(id, provider, version) VALUES(1, 'noop', 0)
ON DUPLICATE KEY UPDATE id = id;

CREATE TABLE IF NOT EXISTS i18n_command (
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    action          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_kind   VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id     VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (command_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS i18n_source (
    entity_type     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    entity_id       VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    merchant_id     BIGINT       NOT NULL,
    shop_id         BIGINT       NOT NULL,
    source_text     TEXT         NOT NULL,
    source_version  BIGINT       NOT NULL,
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (entity_type, entity_id, merchant_id, shop_id),
    KEY idx_i18n_source_type (entity_type, merchant_id, shop_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS i18n_text (
    entity_type              VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    entity_id                VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    locale                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    merchant_id              BIGINT       NOT NULL,
    shop_id                  BIGINT       NOT NULL,
    value                    TEXT         NOT NULL,
    status                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    text_source              VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    source_version_at_write  BIGINT       NOT NULL,
    version                  BIGINT       NOT NULL,
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (entity_type, entity_id, locale, merchant_id, shop_id),
    KEY idx_i18n_text_published (entity_type, locale, status, merchant_id, shop_id),
    CONSTRAINT ck_i18n_text_status CHECK (status IN ('machine','published')),
    CONSTRAINT ck_i18n_text_source CHECK (text_source IN ('human','machine')),
    CONSTRAINT ck_i18n_text_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS i18n_text_history (
    id                       BIGINT       NOT NULL AUTO_INCREMENT,
    entity_type              VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    entity_id                VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    locale                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    merchant_id              BIGINT       NOT NULL,
    shop_id                  BIGINT       NOT NULL,
    value                    TEXT         NOT NULL,
    status                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    text_source              VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    source_version_at_write  BIGINT       NOT NULL,
    version                  BIGINT       NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_i18n_text_history_row (entity_type, entity_id, locale, merchant_id, shop_id, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
