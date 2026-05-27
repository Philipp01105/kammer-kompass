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

-- name: ListAdminInfoSuggestions :many
SELECT s.id, s.ihk_id, i.state AS ihk_state, s.public_pending_visible, s.status, s.created_at
FROM info_suggestions s
JOIN ihks i ON i.id = s.ihk_id
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('ihk_id')::uuid IS NULL OR ihk_id = sqlc.narg('ihk_id')::uuid)
  AND (sqlc.narg('public_pending_visible')::bool IS NULL OR public_pending_visible = sqlc.narg('public_pending_visible'))
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
