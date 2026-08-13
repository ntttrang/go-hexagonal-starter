package httpadapter

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	users domain.UserService
	log   *slog.Logger
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users domain.UserService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{users: users, log: log}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Registration payload"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid input"})
		return
	}

	user, err := h.users.Register(c.Request.Context(), domain.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		mapError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, toUserResponse(user))
}

// Login godoc
// @Summary Login and obtain a JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login payload"
// @Success 200 {object} AuthResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid input"})
		return
	}

	token, user, err := h.users.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		mapError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, AuthResponse{Token: token, User: toUserResponse(user)})
}
