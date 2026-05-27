package auth

import "time"

type User struct {
	ID           string
	Email        string
	DisplayName  string
	PasswordHash string
	IsVerified   bool
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
