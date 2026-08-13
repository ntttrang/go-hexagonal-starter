package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TokenClaims represents authenticated identity embedded in a token.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}

// TokenIssuer is the outbound port for issuing and validating tokens.
type TokenIssuer interface {
	Issue(claims TokenClaims, ttl time.Duration) (string, error)
	Parse(token string) (*TokenClaims, error)
}

// UserService is the inbound application port for user use cases.
type UserService interface {
	Register(ctx context.Context, in RegisterInput) (*User, error)
	Authenticate(ctx context.Context, email, password string) (token string, user *User, err error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateUserInput) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
