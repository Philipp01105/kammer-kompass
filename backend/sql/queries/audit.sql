-- name: CreateReviewEvent :one
INSERT INTO review_events (target_type, target_id, actor_user_id, action, old_status, new_status, comment)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, target_type, target_id, actor_user_id, action, old_status, new_status, comment, created_at;

-- name: CreateAuditLog :one
INSERT INTO audit_logs (actor_user_id, action, resource_type, resource_id, scope_type, scope_id, old_value, new_value, ip_hash, user_agent_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, actor_user_id, action, resource_type, resource_id, scope_type, scope_id, old_value, new_value, ip_hash, user_agent_hash, created_at;

