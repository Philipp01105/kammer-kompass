-- +goose Up
CREATE TABLE IF NOT EXISTS role_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    allow_mask BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS trg_role_templates_updated_at ON role_templates;
CREATE TRIGGER trg_role_templates_updated_at
BEFORE UPDATE ON role_templates
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    role_template_id UUID NOT NULL REFERENCES role_templates(id),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('global', 'state', 'ihk')),
    scope_id TEXT,
    allow_mask BIGINT NOT NULL,
    deny_mask BIGINT NOT NULL DEFAULT 0,
    granted_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_id ON user_role_assignments(user_id);

-- +goose Down
DROP TABLE IF EXISTS user_role_assignments;
DROP TABLE IF EXISTS role_templates;

