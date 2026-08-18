-- Reusable notification templates and per-channel template selection on policy.
-- Per-event platform_notify_template rows are copied then left unread.

ALTER TABLE platform_notify_event_policy
    ADD COLUMN template_sms VARCHAR(64) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '' AFTER channel_in_app,
    ADD COLUMN template_email VARCHAR(64) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '' AFTER template_sms,
    ADD COLUMN template_in_app VARCHAR(64) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '' AFTER template_email;

CREATE TABLE IF NOT EXISTS platform_notify_template_library (
    code            VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    channel         VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    text_template   TEXT         NULL,
    subject         VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NULL,
    body_html       MEDIUMTEXT   NULL,
    title           VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NULL,
    body            TEXT         NULL,
    variables       JSON         NOT NULL,
    lifecycle       VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    version         BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (code),
    UNIQUE KEY uk_platform_notify_template_library_command (command_key),
    KEY ix_platform_notify_template_library_channel (channel, lifecycle),
    CONSTRAINT ck_platform_notify_template_library_channel CHECK (channel IN ('SMS','EMAIL','IN_APP')),
    CONSTRAINT ck_platform_notify_template_library_lifecycle CHECK (lifecycle IN ('ACTIVE','RETIRED')),
    CONSTRAINT ck_platform_notify_template_library_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO platform_notify_template_library(
    code, channel, text_template, subject, body_html, title, body, variables, lifecycle, version, command_key
)
SELECT
    IF(CHAR_LENGTH(LOWER(CONCAT(t.event_key, '.', LOWER(t.channel)))) <= 64,
        LOWER(CONCAT(t.event_key, '.', LOWER(t.channel))),
        CONCAT('t', LEFT(SHA2(CONCAT(t.event_key, '.', t.channel), 256), 63))) AS code,
    t.channel,
    t.text_template,
    t.subject,
    t.body_html,
    t.title,
    t.body,
    COALESCE(e.variables, JSON_ARRAY()),
    'ACTIVE',
    t.version,
    CONCAT('migr-', t.command_key)
FROM platform_notify_template t
LEFT JOIN platform_notify_event e ON e.event_key = t.event_key
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(3);

UPDATE platform_notify_event_policy p
INNER JOIN platform_notify_template t ON t.event_key = p.event_key AND t.channel = 'SMS'
SET p.template_sms = IF(CHAR_LENGTH(LOWER(CONCAT(t.event_key, '.', LOWER(t.channel)))) <= 64,
    LOWER(CONCAT(t.event_key, '.', LOWER(t.channel))),
    CONCAT('t', LEFT(SHA2(CONCAT(t.event_key, '.', t.channel), 256), 63)));

UPDATE platform_notify_event_policy p
INNER JOIN platform_notify_template t ON t.event_key = p.event_key AND t.channel = 'EMAIL'
SET p.template_email = IF(CHAR_LENGTH(LOWER(CONCAT(t.event_key, '.', LOWER(t.channel)))) <= 64,
    LOWER(CONCAT(t.event_key, '.', LOWER(t.channel))),
    CONCAT('t', LEFT(SHA2(CONCAT(t.event_key, '.', t.channel), 256), 63)));

UPDATE platform_notify_event_policy p
INNER JOIN platform_notify_template t ON t.event_key = p.event_key AND t.channel = 'IN_APP'
SET p.template_in_app = IF(CHAR_LENGTH(LOWER(CONCAT(t.event_key, '.', LOWER(t.channel)))) <= 64,
    LOWER(CONCAT(t.event_key, '.', LOWER(t.channel))),
    CONCAT('t', LEFT(SHA2(CONCAT(t.event_key, '.', t.channel), 256), 63)));

CREATE TABLE IF NOT EXISTS platform_notify_inapp_config (
    id              TINYINT      NOT NULL,
    driver          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled         TINYINT(1)   NOT NULL,
    version         BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_platform_notify_inapp_config_command (command_key),
    CONSTRAINT ck_platform_notify_inapp_config_id CHECK (id = 1),
    CONSTRAINT ck_platform_notify_inapp_config_driver CHECK (driver = 'inbox'),
    CONSTRAINT ck_platform_notify_inapp_config_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT IGNORE INTO platform_notify_inapp_config(id, driver, enabled, version, command_key)
VALUES (1, 'inbox', 1, 1, 'proj-inapp-inbox');
