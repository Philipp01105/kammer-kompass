package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Store struct {
	q *sqlc.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{q: sqlc.New(db)}
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
