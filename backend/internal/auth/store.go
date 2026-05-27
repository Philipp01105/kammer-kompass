package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrPermissionRequestAlreadyPending = errors.New("permission request already pending")
var ErrPermissionAlreadyGranted = errors.New("permission already granted")

type Store struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

type RequestableRoleTemplate struct {
	ID          string
	Name        string
	Description *string
	AllowMask   int64
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db, q: sqlc.New(db)}
}

// RegisterUser creates a new user with the "registered_user" role template assignment. The user will be active and verified by default.
func (s *Store) RegisterUser(ctx context.Context, email, displayName, passwordHash string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	row, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailAlreadyExists
		}
		return User{}, err
	}

	role, err := qtx.GetRoleTemplateByName(ctx, "registered_user")
	if err != nil {
		return User{}, err
	}
	_, err = qtx.CreateUserRoleAssignment(ctx, sqlc.CreateUserRoleAssignmentParams{
		UserID:         row.ID,
		RoleTemplateID: role.ID,
		ScopeType:      "global",
		ScopeID:        nil,
		AllowMask:      role.AllowMask,
		DenyMask:       0,
		GrantedBy:      pgtype.UUID{Valid: false},
		ExpiresAt:      pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

// RegisterUserWithPermissionRequest creates a new user with the given role template assignment. The user will be inactive and unverified by default.
func (s *Store) RegisterUserWithPermissionRequest(ctx context.Context, email, displayName, passwordHash, roleTemplateID, scopeType string, scopeID, proofNote *string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	roleID, err := uuid.Parse(roleTemplateID)
	if err != nil {
		return User{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	if _, err := qtx.GetRoleTemplateByID(ctx, pgtype.UUID{Bytes: roleID, Valid: true}); err != nil {
		return User{}, err
	}

	row, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailAlreadyExists
		}
		return User{}, err
	}

	if _, err := qtx.SetUserActiveVerified(ctx, sqlc.SetUserActiveVerifiedParams{
		ID:         row.ID,
		IsActive:   false,
		IsVerified: false,
	}); err != nil {
		return User{}, err
	}

	if _, err := qtx.CreatePermissionRequest(ctx, sqlc.CreatePermissionRequestParams{
		UserID:                  row.ID,
		RequestType:             "registration",
		RequestedRoleTemplateID: pgtype.UUID{Bytes: roleID, Valid: true},
		RequestedScopeType:      scopeType,
		RequestedScopeID:        scopeID,
		ProofNote:               proofNote,
	}); err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	row.IsActive = false
	row.IsVerified = false
	return userFromRow(row), nil
}

// CreatePermissionRequestForUser creates a new permission request for the given user. The user must not already have the requested permissions orr a pending request
func (s *Store) CreatePermissionRequestForUser(ctx context.Context, userID, roleTemplateID, scopeType string, scopeID, proofNote *string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrUserNotFound
	}
	roleID, err := uuid.Parse(roleTemplateID)
	if err != nil {
		return err
	}
	if _, err := s.q.GetUserByID(ctx, pgtype.UUID{Bytes: uid, Valid: true}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	userUUID := pgtype.UUID{Bytes: uid, Valid: true}
	roleUUID := pgtype.UUID{Bytes: roleID, Valid: true}

	role, err := s.q.GetRoleTemplateByID(ctx, roleUUID)
	if err != nil {
		return err
	}

	pending, err := s.q.ExistsPendingPermissionRequest(ctx, sqlc.ExistsPendingPermissionRequestParams{
		UserID:                  userUUID,
		RequestedRoleTemplateID: roleUUID,
		RequestedScopeType:      scopeType,
		RequestedScopeID:        scopeID,
	})
	if err != nil {
		return err
	}
	if pending {
		return ErrPermissionRequestAlreadyPending
	}

	hasExactAssignment, err := s.q.UserHasActiveRoleTemplateAssignment(ctx, sqlc.UserHasActiveRoleTemplateAssignmentParams{
		UserID:         userUUID,
		RoleTemplateID: roleUUID,
		ScopeType:      scopeType,
		ScopeID:        scopeID,
	})
	if err != nil {
		return err
	}
	if hasExactAssignment {
		return ErrPermissionAlreadyGranted
	}

	assignments, err := s.q.ListRoleAssignmentsByUser(ctx, userUUID)
	if err != nil {
		return err
	}
	if rbac.HasAll(rbac.EffectiveMask(rbacAssignments(assignments), permissionRequestScope(scopeType, scopeID)), rbac.Permission(role.AllowMask)) {
		return ErrPermissionAlreadyGranted
	}

	_, err = s.q.CreatePermissionRequest(ctx, sqlc.CreatePermissionRequestParams{
		UserID:                  userUUID,
		RequestType:             "role_request",
		RequestedRoleTemplateID: roleUUID,
		RequestedScopeType:      scopeType,
		RequestedScopeID:        scopeID,
		ProofNote:               proofNote,
	})
	if isUniqueViolation(err, "ux_permission_requests_pending_scope") {
		return ErrPermissionRequestAlreadyPending
	}
	return err
}

// ListRequestableRoleTemplates returns all role templates that users can request.
func (s *Store) ListRequestableRoleTemplates(ctx context.Context) ([]RequestableRoleTemplate, error) {
	rows, err := s.q.ListRoleTemplates(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]RequestableRoleTemplate, 0, len(rows))
	for _, row := range rows {
		if row.Name == "anonymous" || row.Name == "registered_user" || row.Name == "super_admin" {
			continue
		}
		items = append(items, RequestableRoleTemplate{
			ID:          uuid.UUID(row.ID.Bytes).String(),
			Name:        row.Name,
			Description: row.Description,
			AllowMask:   row.AllowMask,
		})
	}
	return items, nil
}

// rbacAssignments converts a list of sqlc.ListRoleAssignmentsByUserRow to a list of rbac.Assignment
func rbacAssignments(rows []sqlc.ListRoleAssignmentsByUserRow) []rbac.Assignment {
	assignments := make([]rbac.Assignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, rbac.Assignment{
			ScopeType: rbac.ScopeType(row.ScopeType),
			ScopeID:   row.ScopeID,
			AllowMask: rbac.Permission(row.AllowMask),
			DenyMask:  rbac.Permission(row.DenyMask),
		})
	}
	return assignments
}

// permissionRequestScope converts a scope type and scope ID to a rbac.ResourceScope
func permissionRequestScope(scopeType string, scopeID *string) rbac.ResourceScope {
	if scopeID == nil {
		return rbac.ResourceScope{}
	}
	switch scopeType {
	case string(rbac.ScopeState):
		return rbac.ResourceScope{State: *scopeID}
	case string(rbac.ScopeIHK):
		return rbac.ResourceScope{IHKID: *scopeID}
	default:
		return rbac.ResourceScope{}
	}
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return userFromRow(row), nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return User{}, ErrUserNotFound
	}
	row, err := s.q.GetUserByID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return userFromRow(row), nil
}

// userFromRow converts a sqlc.User to a User
func userFromRow(r sqlc.User) User {
	var id string
	if r.ID.Valid {
		id = uuid.UUID(r.ID.Bytes).String()
	}
	return User{
		ID:           id,
		Email:        r.Email,
		DisplayName:  r.DisplayName,
		PasswordHash: r.PasswordHash,
		IsVerified:   r.IsVerified,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}
