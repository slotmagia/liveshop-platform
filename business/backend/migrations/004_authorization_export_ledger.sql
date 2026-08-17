-- One-time handoff evidence for removing browser-user authorization from
-- Platform. The authorizationexport command writes one immutable canonical
-- bundle before it is allowed to drop the retired authorization tables.
CREATE TABLE IF NOT EXISTS platform_authorization_export_ledger (
    singleton_id TINYINT NOT NULL,
    schema_version INT NOT NULL,
    payload_sha256 CHAR(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
    row_count BIGINT NOT NULL,
    payload JSON NOT NULL,
    exported_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    finalized_at DATETIME(3) NULL,
    PRIMARY KEY (singleton_id),
    CONSTRAINT ck_authorization_export_singleton CHECK (singleton_id = 1),
    CONSTRAINT ck_authorization_export_schema CHECK (schema_version = 1),
    CONSTRAINT ck_authorization_export_rows CHECK (row_count >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
