-- name: ListRoleAssignmentsByUser :many
SELECT a.scope_type, COALESCE(a.scope_id, '') AS scope_id, a.allow_mask, a.deny_mask
FROM user_role_assignments a
JOIN users u ON u.id = a.user_id
WHERE a.user_id = $1
  AND u.is_active = true
  AND (a.expires_at IS NULL OR a.expires_at > now())
ORDER BY a.created_at ASC;

-- name: UserHasActiveRoleTemplateAssignment :one
SELECT EXISTS (
  SELECT 1
  FROM user_role_assignments
  WHERE user_id = $1
    AND role_template_id = $2
    AND scope_type = $3
    AND COALESCE(scope_id, '') = COALESCE($4, '')
    AND (expires_at IS NULL OR expires_at > now())
);

-- name: GetRoleTemplateByName :one
SELECT id, name, description, allow_mask, created_at, updated_at
FROM role_templates
WHERE name = $1;

-- name: GetRoleTemplateByID :one
SELECT id, name, description, allow_mask, created_at, updated_at
FROM role_templates
WHERE id = $1;

-- name: ListRoleTemplates :many
SELECT id, name, description, allow_mask, created_at, updated_at
FROM role_templates
ORDER BY name ASC;

-- name: CreateUserRoleAssignment :one
INSERT INTO user_role_assignments (
  user_id, role_template_id, scope_type, scope_id, allow_mask, deny_mask, granted_by, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, role_template_id, scope_type, scope_id, allow_mask, deny_mask, granted_by, expires_at, created_at;

-- name: ListUserRoleAssignmentsDetailed :many
SELECT
  a.id,
  a.user_id,
  a.role_template_id,
  t.name AS role_name,
  a.scope_type,
  a.scope_id,
  a.allow_mask,
  a.deny_mask,
  a.granted_by,
  a.expires_at,
  a.created_at
FROM user_role_assignments a
JOIN role_templates t ON t.id = a.role_template_id
WHERE a.user_id = $1
ORDER BY a.created_at DESC;

-- name: DeleteUserRoleAssignment :one
DELETE FROM user_role_assignments
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, role_template_id, scope_type, scope_id, allow_mask, deny_mask, granted_by, expires_at, created_at;

