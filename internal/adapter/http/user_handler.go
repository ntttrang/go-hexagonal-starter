package httpadapter

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

// UserHandler handles user CRUD endpoints.
type UserHandler struct {
	users domain.UserService
	log   *slog.Logger
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(users domain.UserService, log *slog.Logger) *UserHandler {
	return &UserHandler{users: users, log: log}
}

// List godoc
// @Summary List users
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Page size" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, err := h.users.List(c.Request.Context(), limit, offset)
	if err != nil {
		mapError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponses(users))
}

// Get godoc
// @Summary Get user by ID
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		mapError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

// Update godoc
// @Summary Update a user
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body UpdateUserRequest true "Update payload"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid input"})
		return
	}
	if req.Name == nil && req.Email == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid input"})
		return
	}

	user, err := h.users.Update(c.Request.Context(), id, domain.UpdateUserInput{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		mapError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

// Delete godoc
// @Summary Delete a user
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		mapError(c, h.log, err)
		return
	}
	c.Status(http.StatusNoContent)
}
