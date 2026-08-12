CREATE TABLE IF NOT EXISTS platform_registry_state (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    revision BIGINT NOT NULL CHECK (revision > 0),
    releases JSONB NOT NULL,
    active JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO platform_registry_state (id, revision, releases, active)
VALUES (1, 1, '{}'::jsonb, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;
