package httpadapter

import (
	"time"

	"github.com/google/uuid"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"alice@example.com"`
	Name     string `json:"name" binding:"required,min=1,max=100" example:"Alice"`
	Password string `json:"password" binding:"required,min=8" example:"secret123"`
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"alice@example.com"`
	Password string `json:"password" binding:"required" example:"secret123"`
}

// UpdateUserRequest is the body for PUT /users/:id.
type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty" example:"Alice Updated"`
	Email *string `json:"email,omitempty" example:"alice.new@example.com"`
}

// UserResponse is the public user representation (no password).
type UserResponse struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string    `json:"email" example:"alice@example.com"`
	Name      string    `json:"name" example:"Alice"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthResponse is returned after successful login.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ErrorResponse is a standard API error body.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid input"`
}

// HealthResponse is returned by health endpoints.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func toUserResponses(users []domain.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, toUserResponse(&users[i]))
	}
	return out
}
