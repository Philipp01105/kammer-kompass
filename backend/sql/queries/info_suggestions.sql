-- name: CreateInfoSuggestion :one
INSERT INTO info_suggestions (
  ihk_id,
  current_text_snapshot,
  suggested_change,
  public_pending_text,
  reason,
  source_url,
  source_note,
  language_code,
  language_confidence,
  pre_moderation_status,
  moderation_flags,
  public_pending_visible,
  public_pending_created_at,
  submitted_by_user_id,
  submitted_email,
  ip_hash,
  status
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11,
  $12, $13, $14, $15, $16,
  $17
)
RETURNING id, ihk_id, current_text_snapshot, suggested_change, public_pending_text, reason, source_url, source_note,
  language_code, language_confidence, pre_moderation_status, moderation_flags,
  public_pending_visible, public_pending_created_at, public_pending_hidden_at, public_pending_hidden_by, public_pending_hide_reason,
  submitted_by_user_id, submitted_email, ip_hash, status, assigned_to, accepted_by, applied_by, created_at, updated_at;

-- name: ListPendingHintsByIHKID :many
SELECT id, ihk_id, public_pending_text, source_url, source_note, created_at
FROM info_suggestions
WHERE ihk_id = $1
  AND public_pending_visible = true
  AND status IN ('submitted', 'under_review')
ORDER BY created_at DESC
LIMIT $2;

-- name: ListPendingHintsByIHKIDs :many
SELECT h.id, h.ihk_id, h.public_pending_text, h.source_url, h.source_note, h.created_at
FROM unnest(sqlc.arg('ihk_ids')::uuid[]) AS requested(ihk_id)
JOIN LATERAL (
  SELECT id, ihk_id, public_pending_text, source_url, source_note, created_at
  FROM info_suggestions
  WHERE ihk_id = requested.ihk_id
    AND public_pending_visible = true
    AND status IN ('submitted', 'under_review')
  ORDER BY created_at DESC
  LIMIT sqlc.arg('per_ihk_limit')
) h ON true
ORDER BY h.ihk_id, h.created_at DESC;

-- name: ListAdminInfoSuggestions :many
SELECT s.id, s.ihk_id, i.state AS ihk_state, s.public_pending_visible, s.status, s.created_at
FROM info_suggestions s
JOIN ihks i ON i.id = s.ihk_id
JOIN (
  SELECT
    i2.id AS ihk_id,
    COALESCE(bit_or(a.allow_mask), 0)::bigint AS allow_mask,
    COALESCE(bit_or(a.deny_mask), 0)::bigint AS deny_mask
  FROM ihks i2
  JOIN user_role_assignments a ON a.user_id = sqlc.arg('actor_user_id')::uuid
    AND (a.expires_at IS NULL OR a.expires_at > now())
    AND (
      a.scope_type = 'global'
      OR (a.scope_type = 'state' AND a.scope_id = i2.state)
      OR (a.scope_type = 'ihk' AND a.scope_id = i2.id::text)
    )
  GROUP BY i2.id
) scope_mask ON scope_mask.ihk_id = i.id
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('ihk_id')::uuid IS NULL OR ihk_id = sqlc.narg('ihk_id')::uuid)
  AND (sqlc.narg('public_pending_visible')::bool IS NULL OR public_pending_visible = sqlc.narg('public_pending_visible'))
  AND ((scope_mask.allow_mask & ~scope_mask.deny_mask) & sqlc.arg('required_mask')::bigint) = sqlc.arg('required_mask')::bigint
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL OR
    (s.created_at < sqlc.narg('cursor_created_at') OR (s.created_at = sqlc.narg('cursor_created_at') AND s.id < sqlc.narg('cursor_id')::uuid))
  )
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg('limit');

-- name: GetAdminInfoSuggestionByID :one
SELECT
  s.*,
  i.name AS ihk_name,
  i.slug AS ihk_slug,
  i.state AS ihk_state,
  p.current_text AS live_current_text
FROM info_suggestions s
JOIN ihks i ON i.id = s.ihk_id
JOIN ihk_info_pages p ON p.ihk_id = i.id
WHERE s.id = $1;

-- name: LockInfoSuggestionByID :one
SELECT *
FROM info_suggestions
WHERE id = $1
FOR UPDATE;

-- name: UpdateInfoSuggestionStatus :one
UPDATE info_suggestions
SET status = $2,
    public_pending_visible = $3,
    accepted_by = $4,
    applied_by = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: HideInfoSuggestionPending :one
UPDATE info_suggestions
SET public_pending_visible = false,
    public_pending_hidden_at = now(),
    public_pending_hidden_by = $2,
    public_pending_hide_reason = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;
