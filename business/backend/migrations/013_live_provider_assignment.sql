CREATE TABLE live_provider_assignment (
    merchant_id   BIGINT      NOT NULL,
    provider_code VARCHAR(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
    enabled       TINYINT(1)  NOT NULL DEFAULT 1,
    is_default    TINYINT(1)  NOT NULL DEFAULT 0,
    version       BIGINT      NOT NULL,
    created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (merchant_id, provider_code),
    CONSTRAINT ck_live_provider_assignment_ids CHECK (merchant_id > 0),
    CONSTRAINT ck_live_provider_assignment_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE live_provider_assignment_state (
    merchant_id BIGINT NOT NULL,
    version     BIGINT NOT NULL,
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (merchant_id),
    CONSTRAINT ck_live_provider_assignment_state_ids CHECK (merchant_id > 0),
    CONSTRAINT ck_live_provider_assignment_state_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE live_provider_assignment_command (
    command_key      VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    request_hash     CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    merchant_id      BIGINT       NULL,
    response_version BIGINT       NOT NULL DEFAULT 0,
    response_json    JSON         NULL,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    completed_at     DATETIME(3)  NULL,
    PRIMARY KEY (command_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
