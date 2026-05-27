package rbac

import (
	"context"
	"errors"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidUserID = errors.New("invalid user id")

type Service struct {
	q *sqlc.Queries
}

func NewService(q *sqlc.Queries) *Service {
	return &Service{q: q}
}

// ListAssignments returns all Assignments for the given user.
func (s *Service) ListAssignments(ctx context.Context, userID string) ([]Assignment, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	rows, err := s.q.ListRoleAssignmentsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}

	assignments := make([]Assignment, 0, len(rows))
	for _, r := range rows {
		assignments = append(assignments, Assignment{
			ScopeType: ScopeType(r.ScopeType),
			ScopeID:   r.ScopeID,
			AllowMask: Permission(r.AllowMask),
			DenyMask:  Permission(r.DenyMask),
		})
	}
	return assignments, nil
}

// EffectiveMask returns the effective permission mask for the given user and scope.
func (s *Service) EffectiveMask(ctx context.Context, userID string, scope ResourceScope) (Permission, error) {
	assignments, err := s.ListAssignments(ctx, userID)
	if err != nil {
		return 0, err
	}
	return EffectiveMask(assignments, scope), nil
}
