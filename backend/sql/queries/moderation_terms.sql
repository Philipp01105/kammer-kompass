-- name: ListActiveModerationTerms :many
SELECT id, term, normalized_term, category, severity, is_active, created_by, created_at, updated_at
FROM moderation_terms
WHERE is_active = true
ORDER BY severity DESC, category ASC, normalized_term ASC;

-- name: ListActiveModerationNormalizedTerms :many
SELECT normalized_term
FROM moderation_terms
WHERE is_active = true
ORDER BY normalized_term ASC;

-- name: CreateModerationTerm :one
INSERT INTO moderation_terms (term, normalized_term, category, severity, is_active, created_by)
VALUES ($1, $2, $3, $4, true, $5)
RETURNING id, term, normalized_term, category, severity, is_active, created_by, created_at, updated_at;

-- name: UpdateModerationTerm :one
UPDATE moderation_terms
SET term = COALESCE(sqlc.narg('term'), term),
    normalized_term = COALESCE(sqlc.narg('normalized_term'), normalized_term),
    category = COALESCE(sqlc.narg('category'), category),
    severity = COALESCE(sqlc.narg('severity'), severity),
    is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = sqlc.arg('id')
RETURNING id, term, normalized_term, category, severity, is_active, created_by, created_at, updated_at;

-- name: SoftDeleteModerationTerm :exec
UPDATE moderation_terms
SET is_active = false
WHERE id = $1;
