ALTER TABLE platform_permission_catalog
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS release_version TEXT;

CREATE INDEX IF NOT EXISTS idx_platform_permission_catalog_active
    ON platform_permission_catalog (module_id, active, permission_code);

COMMENT ON COLUMN platform_permission_catalog.active IS
    'True only when the owning permission is declared by the module active release.';

COMMENT ON COLUMN platform_permission_catalog.release_version IS
    'Last module release version that activated this permission definition.';
