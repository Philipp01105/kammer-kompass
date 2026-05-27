-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS ux_permission_requests_pending_scope
ON permission_requests (
    user_id,
    requested_role_template_id,
    requested_scope_type,
    COALESCE(requested_scope_id, '')
)
WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS ux_permission_requests_pending_scope;
