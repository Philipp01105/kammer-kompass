-- +goose Up
SELECT 1;

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

