-- Notification events are projected from the active Registry snapshot.
-- SMS/email channels stay in their own catalogues; this schema owns events,
-- policy, templates, deliveries, attempts and in-app inbox.

CREATE TABLE IF NOT EXISTS platform_notify_event (
    event_key           VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    module_id           VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    module_name         VARCHAR(128) COLLATE utf8mb4_0900_ai_ci NOT NULL,
    operation_id        VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    title               VARCHAR(128) COLLATE utf8mb4_0900_ai_ci NOT NULL,
    variables           JSON         NOT NULL,
    allowed_channels    JSON         NOT NULL,
    default_dispatch    VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    dispatchable        TINYINT(1)   NOT NULL,
    registry_revision   BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (event_key),
    KEY ix_platform_notify_event_module (module_id, dispatchable),
    CONSTRAINT ck_platform_notify_event_dispatch CHECK (default_dispatch IN ('SYNC','ASYNC','SCHEDULED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_event_policy (
    event_key       VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    dispatch_mode   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    delay_seconds   INT          NOT NULL DEFAULT 0,
    channel_sms     TINYINT(1)   NOT NULL,
    channel_email   TINYINT(1)   NOT NULL,
    channel_in_app  TINYINT(1)   NOT NULL,
    version         BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (event_key),
    UNIQUE KEY uk_platform_notify_event_policy_command (command_key),
    CONSTRAINT ck_platform_notify_event_policy_mode CHECK (dispatch_mode IN ('SYNC','ASYNC','SCHEDULED')),
    CONSTRAINT ck_platform_notify_event_policy_delay CHECK (delay_seconds >= 0 AND delay_seconds <= 2592000),
    CONSTRAINT ck_platform_notify_event_policy_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_template (
    event_key       VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    channel         VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    text_template   TEXT         NULL,
    subject         VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NULL,
    body_html       MEDIUMTEXT   NULL,
    title           VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NULL,
    body            TEXT         NULL,
    version         BIGINT       NOT NULL,
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (event_key, channel),
    UNIQUE KEY uk_platform_notify_template_command (command_key),
    CONSTRAINT ck_platform_notify_template_channel CHECK (channel IN ('SMS','EMAIL','IN_APP')),
    CONSTRAINT ck_platform_notify_template_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_delivery (
    delivery_id     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    delivery_key    VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    event_key       VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    channel         VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    merchant_id     BIGINT       NOT NULL,
    shop_id         BIGINT       NOT NULL,
    status          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    recipient       VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
    variables       JSON         NOT NULL,
    request_hash    CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    not_before      DATETIME(3)  NULL,
    last_error      VARCHAR(512) COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
    attempt_count   INT          NOT NULL DEFAULT 0,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (delivery_id),
    UNIQUE KEY uk_platform_notify_delivery_key_channel (delivery_key, channel),
    KEY ix_platform_notify_delivery_event (event_key, created_at),
    KEY ix_platform_notify_delivery_due (status, not_before, updated_at),
    CONSTRAINT ck_platform_notify_delivery_channel CHECK (channel IN ('SMS','EMAIL','IN_APP')),
    CONSTRAINT ck_platform_notify_delivery_status CHECK (status IN ('PENDING','SCHEDULED','SENDING','SENT','FAILED_PERMANENT','UNKNOWN')),
    CONSTRAINT ck_platform_notify_delivery_scope CHECK (merchant_id >= 0 AND shop_id >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_attempt (
    delivery_id     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    attempt_no      INT          NOT NULL,
    status          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    detail          VARCHAR(512) COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (delivery_id, attempt_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_inbox (
    id              BIGINT       NOT NULL AUTO_INCREMENT,
    merchant_id     BIGINT       NOT NULL,
    shop_id         BIGINT       NOT NULL,
    subject         VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    delivery_id     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    title           VARCHAR(256) COLLATE utf8mb4_0900_ai_ci NOT NULL,
    body            TEXT         NOT NULL,
    read_at         DATETIME(3)  NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_platform_notify_inbox_delivery (merchant_id, shop_id, subject, delivery_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS platform_notify_command (
    command_key     VARCHAR(64)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash    CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    action          VARCHAR(32)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_kind   VARCHAR(16)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    resource_id     VARCHAR(160) COLLATE utf8mb4_0900_as_cs NOT NULL,
    result_version  BIGINT       NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (command_key),
    CONSTRAINT ck_platform_notify_command_version CHECK (result_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
