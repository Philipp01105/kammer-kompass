-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, display_name, password_hash, is_verified, is_active, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, display_name, password_hash, is_verified, is_active, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, display_name, password_hash, is_verified, is_active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, display_name, password_hash, is_verified, is_active, created_at, updated_at
FROM users
WHERE (
  sqlc.narg('query')::text IS NULL
  OR email ILIKE ('%' || sqlc.narg('query') || '%')
  OR display_name ILIKE ('%' || sqlc.narg('query') || '%')
)
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at < sqlc.narg('cursor_created_at') OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id')::uuid))
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

