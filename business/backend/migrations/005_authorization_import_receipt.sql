-- Durable proof that Identity imported the exact immutable export. This is a
-- separate table so upgraded databases converge without altering the original
-- export ledger in place.
CREATE TABLE IF NOT EXISTS platform_authorization_import_receipt (
    singleton_id                   TINYINT      NOT NULL,
    receipt_schema_version         INT          NOT NULL,
    source                         VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    import_id                      VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    payload_sha256                 CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    row_count                      BIGINT       NOT NULL,
    imported_at                    VARCHAR(35)  COLLATE utf8mb4_0900_as_cs NOT NULL,
    target_identity_instance       VARCHAR(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
    target_identity_schema_version INT          NOT NULL,
    key_id                         VARCHAR(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
    receipt_sha256                 CHAR(64)     COLLATE utf8mb4_0900_as_cs NOT NULL,
    receipt                        JSON         NOT NULL,
    acknowledged_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (singleton_id),
    UNIQUE KEY uq_platform_authorization_import_id (import_id),
    CONSTRAINT fk_platform_authorization_import_export
        FOREIGN KEY (singleton_id)
        REFERENCES platform_authorization_export_ledger (singleton_id),
    CONSTRAINT ck_authorization_import_singleton CHECK (singleton_id = 1),
    CONSTRAINT ck_authorization_import_schema CHECK (receipt_schema_version = 1),
    CONSTRAINT ck_authorization_import_rows CHECK (row_count >= 0),
    CONSTRAINT ck_authorization_import_target_schema CHECK (target_identity_schema_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
