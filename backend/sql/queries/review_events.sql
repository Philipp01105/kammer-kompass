-- name: ListReviewEvents :many
SELECT id, target_type, target_id, actor_user_id, action, old_status, new_status, comment, created_at
FROM review_events
WHERE target_type = $1
  AND target_id = $2
ORDER BY created_at ASC, id ASC;

