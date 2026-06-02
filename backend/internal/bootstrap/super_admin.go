package bootstrap

import (
	"context"
	"errors"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// EnsureDefaultSuperAdmin creates the default super admin only when explicitly
// configured by the operator. The bootstrap password is never logged.

const defaultSuperAdminEmail = "super_admin@local.invalid"
const defaultSuperAdminUsername = "super_admin"

type DefaultSuperAdminCredentials struct {
	Username string
}

func EnsureDefaultSuperAdmin(ctx context.Context, db *pgxpool.Pool, password string) (*DefaultSuperAdminCredentials, error) {
	if password == "" {
		return nil, errors.New("bootstrap password is required")
	}
	q := sqlc.New(db)

	role, err := q.GetRoleTemplateByName(ctx, "super_admin")
	if err != nil {
		return nil, err
	}

	user, created, err := ensureDefaultUser(ctx, q, password)
	if err != nil {
		return nil, err
	}

	exists, err := hasGlobalRole(ctx, db, user.ID, role.ID)
	if err != nil {
		return nil, err
	}
	if !exists {
		if _, err := q.CreateUserRoleAssignment(ctx, sqlc.CreateUserRoleAssignmentParams{
			UserID:         user.ID,
			RoleTemplateID: role.ID,
			ScopeType:      "global",
			ScopeID:        nil,
			AllowMask:      role.AllowMask,
			DenyMask:       0,
			GrantedBy:      pgtype.UUID{Valid: false},
			ExpiresAt:      pgtype.Timestamptz{Valid: false},
		}); err != nil {
			return nil, err
		}
	}

	if !created {
		return nil, nil
	}
	return &DefaultSuperAdminCredentials{
		Username: defaultSuperAdminUsername,
	}, nil
}

func ensureDefaultUser(ctx context.Context, q *sqlc.Queries, password string) (sqlc.User, bool, error) {
	user, err := q.GetUserByEmail(ctx, defaultSuperAdminEmail)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.User{}, false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return sqlc.User{}, false, err
	}

	user, err = q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        defaultSuperAdminEmail,
		DisplayName:  defaultSuperAdminUsername,
		PasswordHash: string(hash),
	})
	if err != nil {
		return sqlc.User{}, false, err
	}
	return user, true, nil
}

func hasGlobalRole(ctx context.Context, db *pgxpool.Pool, userID pgtype.UUID, roleID pgtype.UUID) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_role_assignments
			WHERE user_id = $1
			  AND role_template_id = $2
			  AND scope_type = 'global'
		)
	`, userID, roleID).Scan(&exists)
	return exists, err
}
