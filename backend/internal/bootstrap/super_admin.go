package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// creates the default super admin, password will get logged to stdout

const defaultSuperAdminEmail = "super_admin@local.invalid"
const defaultSuperAdminUsername = "super_admin"

type DefaultSuperAdminCredentials struct {
	Username string
	Password string
}

func EnsureDefaultSuperAdmin(ctx context.Context, db *pgxpool.Pool) (*DefaultSuperAdminCredentials, error) {
	q := sqlc.New(db)

	role, err := q.GetRoleTemplateByName(ctx, "super_admin")
	if err != nil {
		return nil, err
	}

	user, created, password, err := ensureDefaultUser(ctx, q)
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
		Password: password,
	}, nil
}

func ensureDefaultUser(ctx context.Context, q *sqlc.Queries) (sqlc.User, bool, string, error) {
	user, err := q.GetUserByEmail(ctx, defaultSuperAdminEmail)
	if err == nil {
		return user, false, "", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.User{}, false, "", err
	}

	password, err := randomPassword()
	if err != nil {
		return sqlc.User{}, false, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return sqlc.User{}, false, "", err
	}

	user, err = q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        defaultSuperAdminEmail,
		DisplayName:  defaultSuperAdminUsername,
		PasswordHash: string(hash),
	})
	if err != nil {
		return sqlc.User{}, false, "", err
	}
	return user, true, password, nil
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

func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
