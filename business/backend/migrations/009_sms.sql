-- Platform-owned SMS catalogue: drivers stay in code; channels, regions and
-- merchant grants are versioned. Secrets are AES-GCM ciphertext. Physical
-- delete is replaced by RETIRED. app_id=0 is the Admin global partition.

CREATE TABLE IF NOT EXISTS sms_catalogue (
    app_id      BIGINT      NOT NULL DEFAULT 0,
    revision    BIGINT      NOT NULL DEFAULT 0,
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id),
    CONSTRAINT ck_sms_catalogue_app CHECK (app_id >= 0),
    CONSTRAINT ck_sms_catalogue_revision CHECK (revision >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_region (
    id          BIGINT       NOT NULL AUTO_INCREMENT,
    app_id      BIGINT       NOT NULL DEFAULT 0,
    code        VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    dial_code   VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    iso2        CHAR(2)      COLLATE utf8mb4_0900_as_cs NOT NULL,
    emoji       VARCHAR(16)  NOT NULL DEFAULT '',
    sort_order  INT          NOT NULL DEFAULT 0,
    enabled     TINYINT(1)   NOT NULL,
    lifecycle   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    version     BIGINT       NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_sms_region_app_code (app_id, code),
    UNIQUE KEY uk_sms_region_app_dial (app_id, dial_code),
    KEY idx_sms_region_app_state (app_id, lifecycle, sort_order, id),
    CONSTRAINT ck_sms_region_app CHECK (app_id >= 0),
    CONSTRAINT ck_sms_region_lifecycle CHECK (lifecycle IN ('ACTIVE','RETIRED')),
    CONSTRAINT ck_sms_region_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_region_version (
    app_id      BIGINT       NOT NULL,
    region_code VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    version     BIGINT       NOT NULL,
    dial_code   VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    iso2        CHAR(2)      COLLATE utf8mb4_0900_as_cs NOT NULL,
    emoji       VARCHAR(16)  NOT NULL DEFAULT '',
    sort_order  INT          NOT NULL,
    enabled     TINYINT(1)   NOT NULL,
    lifecycle   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, region_code, version),
    CONSTRAINT ck_sms_region_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_channel (
    id                       BIGINT       NOT NULL AUTO_INCREMENT,
    app_id                   BIGINT       NOT NULL DEFAULT 0,
    code                     VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    region                   VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    priority                 INT          NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    version                  BIGINT       NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_sms_channel_app_code (app_id, code),
    KEY idx_sms_channel_app_state (app_id, lifecycle, enabled, id),
    CONSTRAINT ck_sms_channel_app CHECK (app_id >= 0),
    CONSTRAINT ck_sms_channel_driver CHECK (driver IN ('mock','aliyun','yunpian','twilio')),
    CONSTRAINT ck_sms_channel_lifecycle CHECK (lifecycle IN ('ACTIVE','RETIRED')),
    CONSTRAINT ck_sms_channel_version CHECK (version > 0),
    CONSTRAINT ck_sms_channel_priority CHECK (priority BETWEEN 0 AND 1000)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_channel_version (
    app_id                   BIGINT       NOT NULL,
    channel_code             VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    version                  BIGINT       NOT NULL,
    name                     VARCHAR(120) NOT NULL,
    driver                   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    region                   VARCHAR(8)   COLLATE utf8mb4_0900_as_cs NOT NULL,
    priority                 INT          NOT NULL,
    enabled                  TINYINT(1)   NOT NULL,
    lifecycle                VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    public_config            JSON         NOT NULL,
    credential_ciphertext    MEDIUMBLOB   NULL,
    credential_key_id        VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
    credential_masks         JSON         NOT NULL,
    created_at               DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, channel_code, version),
    CONSTRAINT ck_sms_channel_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_merchant_grant (
    id             BIGINT      NOT NULL AUTO_INCREMENT,
    app_id         BIGINT      NOT NULL DEFAULT 0,
    grant_app_id   BIGINT      NOT NULL,
    commercial_id  BIGINT      NOT NULL,
    dial_codes     JSON        NOT NULL,
    unrestricted   TINYINT(1)  NOT NULL,
    version        BIGINT      NOT NULL,
    created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_sms_grant_tenant (app_id, grant_app_id, commercial_id),
    CONSTRAINT ck_sms_grant_app CHECK (app_id >= 0),
    CONSTRAINT ck_sms_grant_tenant CHECK (grant_app_id > 0 AND commercial_id > 0),
    CONSTRAINT ck_sms_grant_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_merchant_grant_version (
    app_id         BIGINT      NOT NULL,
    grant_app_id   BIGINT      NOT NULL,
    commercial_id  BIGINT      NOT NULL,
    version        BIGINT      NOT NULL,
    dial_codes     JSON        NOT NULL,
    unrestricted   TINYINT(1)  NOT NULL,
    created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, grant_app_id, commercial_id, version),
    CONSTRAINT ck_sms_grant_version_positive CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sms_command (
    app_id          BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE ascii_bin NOT NULL,
    action          VARCHAR(24)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_kind   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (app_id, command_key),
    CONSTRAINT ck_sms_command_action CHECK (action IN ('UPSERT_CHANNEL','ENABLE_CHANNEL','RETIRE_CHANNEL','UPSERT_REGION','ENABLE_REGION','RETIRE_REGION','PUT_GRANT')),
    CONSTRAINT ck_sms_command_kind CHECK (resource_kind IN ('CHANNEL','REGION','GRANT')),
    CONSTRAINT ck_sms_command_version CHECK (result_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO sms_catalogue(app_id, revision) VALUES (0, 0)
ON DUPLICATE KEY UPDATE app_id=VALUES(app_id);

INSERT INTO sms_region(app_id, code, dial_code, name, iso2, emoji, sort_order, enabled, lifecycle, version)
SELECT 0, t.code, t.dial_code, t.name, t.iso2, t.emoji, t.sort_order, 1, 'ACTIVE', 1
FROM (
    SELECT '86' code, '+86' dial_code, '中国大陆' name, 'CN' iso2, '🇨🇳' emoji, 1 sort_order UNION ALL
    SELECT '852', '+852', '中国香港', 'HK', '🇭🇰', 2 UNION ALL
    SELECT '853', '+853', '中国澳门', 'MO', '🇲🇴', 3 UNION ALL
    SELECT '886', '+886', '中国台湾', 'TW', '🇹🇼', 4 UNION ALL
    SELECT '1', '+1', '美国/加拿大', 'US', '🇺🇸', 5 UNION ALL
    SELECT '44', '+44', '英国', 'GB', '🇬🇧', 6 UNION ALL
    SELECT '81', '+81', '日本', 'JP', '🇯🇵', 7 UNION ALL
    SELECT '82', '+82', '韩国', 'KR', '🇰🇷', 8 UNION ALL
    SELECT '65', '+65', '新加坡', 'SG', '🇸🇬', 9 UNION ALL
    SELECT '60', '+60', '马来西亚', 'MY', '🇲🇾', 10 UNION ALL
    SELECT '66', '+66', '泰国', 'TH', '🇹🇭', 11 UNION ALL
    SELECT '84', '+84', '越南', 'VN', '🇻🇳', 12 UNION ALL
    SELECT '63', '+63', '菲律宾', 'PH', '🇵🇭', 13 UNION ALL
    SELECT '62', '+62', '印度尼西亚', 'ID', '🇮🇩', 14 UNION ALL
    SELECT '91', '+91', '印度', 'IN', '🇮🇳', 15 UNION ALL
    SELECT '61', '+61', '澳大利亚', 'AU', '🇦🇺', 16 UNION ALL
    SELECT '49', '+49', '德国', 'DE', '🇩🇪', 17 UNION ALL
    SELECT '33', '+33', '法国', 'FR', '🇫🇷', 18 UNION ALL
    SELECT '7', '+7', '俄罗斯', 'RU', '🇷🇺', 19 UNION ALL
    SELECT '971', '+971', '阿联酋', 'AE', '🇦🇪', 20
) t
WHERE NOT EXISTS (SELECT 1 FROM sms_region WHERE app_id=0 LIMIT 1);

INSERT INTO sms_region_version(app_id, region_code, version, dial_code, name, iso2, emoji, sort_order, enabled, lifecycle)
SELECT app_id, code, version, dial_code, name, iso2, emoji, sort_order, enabled, lifecycle
FROM sms_region
WHERE app_id=0
  AND NOT EXISTS (SELECT 1 FROM sms_region_version WHERE app_id=0 LIMIT 1);
