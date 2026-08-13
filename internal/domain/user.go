package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned on a duplicate or unique-constraint violation.
	ErrConflict = errors.New("conflict")
	// ErrInvalidInput is returned when request data fails validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrUnauthorized is returned when authentication is missing or invalid.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidCredentials is returned when login credentials do not match.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// User is the core domain entity.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RegisterInput holds data required to create a new user.
type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

// UpdateUserInput holds mutable user fields.
type UpdateUserInput struct {
	Name  *string
	Email *string
}

// ValidateRegister validates registration input.
func ValidateRegister(in RegisterInput) error {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	name := strings.TrimSpace(in.Name)
	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidInput
	}
	if name == "" || len(name) > 100 {
		return ErrInvalidInput
	}
	if len(in.Password) < 8 {
		return ErrInvalidInput
	}
	return nil
}

// NormalizeEmail lowercases and trims an email address.
func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
