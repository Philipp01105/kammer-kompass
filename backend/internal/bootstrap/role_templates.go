package bootstrap

import (
	"context"

	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncRoleTemplates inserts role templates into the database.
func SyncRoleTemplates(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, template := range rbac.RoleTemplateDefinitions() {
		mask := int64(template.AllowMask)
		if _, err := tx.Exec(ctx, `
INSERT INTO role_templates (name, description, allow_mask)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  allow_mask = EXCLUDED.allow_mask,
  updated_at = now()
`, template.Name, template.Description, mask); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
UPDATE user_role_assignments
SET allow_mask = $2
WHERE role_template_id = (
  SELECT id
  FROM role_templates
  WHERE name = $1
)
`, template.Name, mask); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
