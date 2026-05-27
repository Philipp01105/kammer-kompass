-- +goose Up
CREATE TABLE IF NOT EXISTS permission_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_type TEXT NOT NULL CHECK (request_type IN ('registration', 'role_request')),
    requested_role_template_id UUID NOT NULL REFERENCES role_templates(id),
    requested_scope_type TEXT NOT NULL CHECK (requested_scope_type IN ('global', 'state', 'ihk')),
    requested_scope_id TEXT,
    proof_note TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    decision_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS trg_permission_requests_updated_at ON permission_requests;
CREATE TRIGGER trg_permission_requests_updated_at
BEFORE UPDATE ON permission_requests
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_permission_requests_status
ON permission_requests(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_permission_requests_user_id
ON permission_requests(user_id);

-- +goose Down
DROP TABLE IF EXISTS permission_requests;
