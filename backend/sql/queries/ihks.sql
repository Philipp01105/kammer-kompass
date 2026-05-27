-- name: ListPublicIHKs :many
SELECT
  i.id,
  i.name,
  i.slug,
  i.city,
  i.state,
  i.official_url,
  i.is_active,
  i.created_at,
  i.updated_at,
  p.id AS info_page_id,
  p.current_text,
  p.confidence_level,
  p.source_summary,
  p.last_version_id,
  p.locked,
  p.updated_by,
  p.created_at AS info_created_at,
  p.updated_at AS info_updated_at
FROM ihks i
LEFT JOIN ihk_info_pages p ON p.ihk_id = i.id
WHERE i.is_active = true
  AND (sqlc.narg('state')::text IS NULL OR i.state = sqlc.narg('state'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR i.name ILIKE ('%' || sqlc.narg('query') || '%')
    OR i.slug ILIKE ('%' || sqlc.narg('query') || '%')
    OR i.city ILIKE ('%' || sqlc.narg('query') || '%')
    OR i.state ILIKE ('%' || sqlc.narg('query') || '%')
  )
  AND (
    sqlc.narg('cursor_name')::text IS NULL OR
    (i.name > sqlc.narg('cursor_name') OR (i.name = sqlc.narg('cursor_name') AND i.id > sqlc.narg('cursor_id')::uuid))
  )
ORDER BY i.name ASC, i.id ASC
LIMIT sqlc.arg('limit');

-- name: GetPublicIHKBySlug :one
SELECT
  i.id,
  i.name,
  i.slug,
  i.city,
  i.state,
  i.official_url,
  i.is_active,
  i.created_at,
  i.updated_at,
  p.id AS info_page_id,
  p.current_text,
  p.confidence_level,
  p.source_summary,
  p.last_version_id,
  p.locked,
  p.updated_by,
  p.created_at AS info_created_at,
  p.updated_at AS info_updated_at
FROM ihks i
LEFT JOIN ihk_info_pages p ON p.ihk_id = i.id
WHERE i.slug = $1
  AND i.is_active = true;

-- name: GetIHKByID :one
SELECT id, name, slug, city, state, official_url, is_active, created_at, updated_at
FROM ihks
WHERE id = $1;

-- name: ListAdminIHKs :many
SELECT id, name, slug, city, state, official_url, is_active, created_at, updated_at
FROM ihks
WHERE (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state'))
  AND (
    sqlc.narg('query')::text IS NULL
    OR name ILIKE ('%' || sqlc.narg('query') || '%')
    OR slug ILIKE ('%' || sqlc.narg('query') || '%')
    OR city ILIKE ('%' || sqlc.narg('query') || '%')
    OR state ILIKE ('%' || sqlc.narg('query') || '%')
  )
  AND (
    sqlc.narg('cursor_name')::text IS NULL
    OR (name > sqlc.narg('cursor_name') OR (name = sqlc.narg('cursor_name') AND id > sqlc.narg('cursor_id')::uuid))
  )
ORDER BY name ASC, id ASC
LIMIT sqlc.arg('limit');

-- name: UpsertCatalogIHK :one
INSERT INTO ihks (name, slug, city, state, official_url, is_active)
VALUES ($1, $2, $3, $4, $5, true)
ON CONFLICT (slug) DO UPDATE SET
  name = EXCLUDED.name,
  city = EXCLUDED.city,
  state = EXCLUDED.state,
  is_active = true,
  updated_at = now()
RETURNING id, name, slug, city, state, official_url, is_active, created_at, updated_at;

-- name: UpdateIHKOfficialURL :one
UPDATE ihks
SET official_url = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, name, slug, city, state, official_url, is_active, created_at, updated_at;

-- name: EnsureEmptyIHKInfoPage :exec
INSERT INTO ihk_info_pages (ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by)
VALUES ($1, '', 'low', NULL, NULL, false, NULL)
ON CONFLICT (ihk_id) DO NOTHING;

-- name: ExistsIHKBySlugOrNameState :one
SELECT EXISTS (
  SELECT 1
  FROM ihks
  WHERE is_active = true
    AND (
      slug = sqlc.arg('slug')
      OR (lower(name) = lower(sqlc.arg('name')::text) AND state = sqlc.arg('state'))
    )
);

-- name: CreateIHKInfoPage :one
INSERT INTO ihk_info_pages (ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by, created_at, updated_at;

-- name: GetIHKInfoPageByIHKID :one
SELECT id, ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by, created_at, updated_at
FROM ihk_info_pages
WHERE ihk_id = $1;

-- name: LockIHKInfoPageByIHKID :one
SELECT id, ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by, created_at, updated_at
FROM ihk_info_pages
WHERE ihk_id = $1
FOR UPDATE;

-- name: UpdateIHKInfoPage :one
UPDATE ihk_info_pages
SET current_text = $2,
    confidence_level = $3,
    source_summary = $4,
    last_version_id = $5,
    updated_by = $6,
    updated_at = now()
WHERE id = $1
RETURNING id, ihk_id, current_text, confidence_level, source_summary, last_version_id, locked, updated_by, created_at, updated_at;

-- name: GetNextInfoVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) + 1 AS next_version
FROM ihk_info_versions
WHERE ihk_info_page_id = $1;

-- name: CreateIHKInfoVersion :one
INSERT INTO ihk_info_versions (
  ihk_info_page_id, ihk_id, version_number, old_text, new_text, change_summary, changed_by,
  based_on_info_suggestion_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, ihk_info_page_id, ihk_id, version_number, old_text, new_text, change_summary, changed_by,
  based_on_info_suggestion_id, created_at;

-- name: ListIHKInfoVersionsByIHKID :many
SELECT id, ihk_info_page_id, ihk_id, version_number, old_text, new_text, change_summary, changed_by,
  based_on_info_suggestion_id, created_at
FROM ihk_info_versions
WHERE ihk_id = $1
ORDER BY version_number DESC;

-- name: GetIHKInfoVersionByIDForIHK :one
SELECT id, ihk_info_page_id, ihk_id, version_number, old_text, new_text, change_summary, changed_by,
  based_on_info_suggestion_id, created_at
FROM ihk_info_versions
WHERE id = $1 AND ihk_id = $2;
