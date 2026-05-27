-- name: CreatePermissionRequest :one
INSERT INTO permission_requests (
  user_id,
  request_type,
  requested_role_template_id,
  requested_scope_type,
  requested_scope_id,
  proof_file_name,
  proof_mime_type,
  proof_content_base64,
  proof_note,
  status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending')
RETURNING *;

-- name: ExistsPendingPermissionRequest :one
SELECT EXISTS (
  SELECT 1
  FROM permission_requests
  WHERE user_id = $1
    AND requested_role_template_id = $2
    AND requested_scope_type = $3
    AND COALESCE(requested_scope_id, '') = COALESCE($4, '')
    AND status = 'pending'
);

-- name: ListPermissionRequests :many
SELECT
  pr.id,
  pr.user_id,
  u.email,
  u.display_name,
  pr.request_type,
  pr.requested_role_template_id,
  rt.name AS requested_role_name,
  pr.requested_scope_type,
  pr.requested_scope_id,
  pr.proof_file_name,
  pr.proof_mime_type,
  pr.proof_note,
  pr.status,
  pr.created_at
FROM permission_requests pr
JOIN users u ON u.id = pr.user_id
JOIN role_templates rt ON rt.id = pr.requested_role_template_id
WHERE (sqlc.narg('status')::text IS NULL OR pr.status = sqlc.narg('status'))
ORDER BY pr.created_at DESC, pr.id DESC
LIMIT sqlc.arg('limit');

-- name: GetPermissionRequestByID :one
SELECT
  pr.id,
  pr.user_id,
  u.email,
  u.display_name,
  pr.request_type,
  pr.requested_role_template_id,
  rt.name AS requested_role_name,
  rt.allow_mask AS requested_allow_mask,
  pr.requested_scope_type,
  pr.requested_scope_id,
  pr.proof_file_name,
  pr.proof_mime_type,
  pr.proof_content_base64,
  pr.proof_note,
  pr.status,
  pr.reviewed_by,
  pr.reviewed_at,
  pr.decision_note,
  pr.created_at,
  pr.updated_at
FROM permission_requests pr
JOIN users u ON u.id = pr.user_id
JOIN role_templates rt ON rt.id = pr.requested_role_template_id
WHERE pr.id = $1;

-- name: LockPermissionRequestByID :one
SELECT *
FROM permission_requests
WHERE id = $1
FOR UPDATE;

-- name: UpdatePermissionRequestDecision :one
UPDATE permission_requests
SET status = $2,
    reviewed_by = $3,
    reviewed_at = now(),
    decision_note = $4
WHERE id = $1
RETURNING *;

-- name: SetUserActiveVerified :one
UPDATE users
SET is_active = $2,
    is_verified = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE id = $1;

-- name: ListPermissionRequestInfoSuggestions :many
SELECT id, ihk_id, status, created_at
FROM info_suggestions
WHERE submitted_by_user_id = $1
ORDER BY created_at DESC
LIMIT 25;
