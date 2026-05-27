-- name: ListAuditLogs :many
SELECT id, actor_user_id, action, resource_type, resource_id, scope_type, scope_id, old_value, new_value, ip_hash, user_agent_hash, created_at
FROM audit_logs
WHERE (
  sqlc.narg('cursor_created_at')::timestamptz IS NULL OR
  (created_at < sqlc.narg('cursor_created_at') OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id')::uuid))
)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

