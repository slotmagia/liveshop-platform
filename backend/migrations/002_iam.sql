CREATE TABLE IF NOT EXISTS platform_permission_catalog (
    module_id TEXT NOT NULL,
    permission_code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    resource_code TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (permission_code = resource_code || '.' || action)
);

CREATE TABLE IF NOT EXISTS platform_department (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    department_id BIGINT NOT NULL,
    parent_department_id BIGINT,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    version BIGINT NOT NULL CHECK (version > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (app_id, merchant_id, department_id),
    FOREIGN KEY (app_id, merchant_id, parent_department_id)
        REFERENCES platform_department (app_id, merchant_id, department_id)
        DEFERRABLE INITIALLY DEFERRED,
    CHECK (parent_department_id IS NULL OR parent_department_id <> department_id)
);

CREATE TABLE IF NOT EXISTS platform_role (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    version BIGINT NOT NULL CHECK (version > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (app_id, merchant_id, role_id),
    UNIQUE (app_id, merchant_id, name)
);

CREATE TABLE IF NOT EXISTS platform_role_permission (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    permission_code TEXT NOT NULL REFERENCES platform_permission_catalog(permission_code),
    PRIMARY KEY (app_id, merchant_id, role_id, permission_code),
    FOREIGN KEY (app_id, merchant_id, role_id)
        REFERENCES platform_role (app_id, merchant_id, role_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS platform_role_data_scope (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    resource_code TEXT NOT NULL,
    scope_type TEXT NOT NULL CHECK (scope_type IN ('ALL', 'DEPARTMENT_AND_CHILDREN', 'DEPARTMENT', 'SELF', 'CUSTOM')),
    PRIMARY KEY (app_id, merchant_id, role_id, resource_code),
    FOREIGN KEY (app_id, merchant_id, role_id)
        REFERENCES platform_role (app_id, merchant_id, role_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS platform_role_scope_department (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    resource_code TEXT NOT NULL,
    department_id BIGINT NOT NULL,
    PRIMARY KEY (app_id, merchant_id, role_id, resource_code, department_id),
    FOREIGN KEY (app_id, merchant_id, role_id, resource_code)
        REFERENCES platform_role_data_scope (app_id, merchant_id, role_id, resource_code) ON DELETE CASCADE,
    FOREIGN KEY (app_id, merchant_id, department_id)
        REFERENCES platform_department (app_id, merchant_id, department_id)
);

CREATE TABLE IF NOT EXISTS platform_subject_role (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    subject TEXT NOT NULL,
    role_id BIGINT NOT NULL,
    PRIMARY KEY (app_id, merchant_id, subject, role_id),
    FOREIGN KEY (app_id, merchant_id, role_id)
        REFERENCES platform_role (app_id, merchant_id, role_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS platform_subject_department (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    subject TEXT NOT NULL,
    department_id BIGINT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (app_id, merchant_id, subject, department_id),
    FOREIGN KEY (app_id, merchant_id, department_id)
        REFERENCES platform_department (app_id, merchant_id, department_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_platform_subject_primary_department
    ON platform_subject_department (app_id, merchant_id, subject) WHERE is_primary;

CREATE TABLE IF NOT EXISTS platform_iam_revision (
    app_id BIGINT NOT NULL,
    merchant_id BIGINT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (app_id, merchant_id)
);

INSERT INTO platform_permission_catalog (module_id, permission_code, name, resource_code, action, description)
VALUES ('platform', 'platform.iam.manage', 'Manage roles and organization', 'platform.iam', 'manage', 'Platform IAM administration')
ON CONFLICT (permission_code) DO NOTHING;
