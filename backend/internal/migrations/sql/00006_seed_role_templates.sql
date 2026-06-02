-- +goose Up
-- Seed default role templates. Values match AllPermissions masks in rbac/masks.go.
-- SyncRoleTemplates() keeps these in sync at runtime; this migration ensures a
-- migration-only setup has valid role data without requiring Go bootstrap code.
INSERT INTO role_templates (name, description, allow_mask) VALUES
  ('anonymous',      'Public read + submit suggestions/proposals',  131171),
  ('registered_user','Like anonymous but trackable',                131171),
  ('contributor',    'Can read/comment on suggestions',             137315),
  ('reviewer',       'Can triage/accept/reject info suggestions',   537328163),
  ('writer',         'Can apply accepted info suggestions',         537075363),
  ('regional_lead',  'Regional reviewer + writer',                  537394851),
  ('admin',          'Manage system incl proposals/terms/users',    2147479467),
  ('super_admin',    'Full technical access',                       8589934591)
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  allow_mask  = EXCLUDED.allow_mask,
  updated_at  = now();

-- +goose Down
DELETE FROM role_templates
WHERE name IN (
  'anonymous',
  'registered_user',
  'contributor',
  'reviewer',
  'writer',
  'regional_lead',
  'admin',
  'super_admin'
);

