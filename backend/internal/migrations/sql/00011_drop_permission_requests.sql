-- +goose Up

DROP TABLE IF EXISTS permission_requests;

-- +goose Down

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

CREATE UNIQUE INDEX IF NOT EXISTS ux_permission_requests_pending_scope
ON permission_requests (
    user_id,
    requested_role_template_id,
    requested_scope_type,
    COALESCE(requested_scope_id, '')
)
WHERE status = 'pending';
