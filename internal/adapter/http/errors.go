package httpadapter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

func mapError(c *gin.Context, log *slog.Logger, err error) {
	status := http.StatusInternalServerError
	msg := "internal server error"

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		msg = "not found"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		msg = "conflict"
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		msg = "invalid input"
	case errors.Is(err, domain.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		msg = "invalid credentials"
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
		msg = "unauthorized"
	}

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	attrs := []any{
		"error", err.Error(),
		"status", status,
		"method", c.Request.Method,
		"path", path,
		"request_id", c.GetString("request_id"),
	}

	if log != nil {
		switch {
		case status >= 500:
			log.ErrorContext(c.Request.Context(), "request failed", attrs...)
		case status == http.StatusUnauthorized || status == http.StatusConflict:
			log.WarnContext(c.Request.Context(), "request rejected", attrs...)
		default:
			log.InfoContext(c.Request.Context(), "request rejected", attrs...)
		}
	}

	c.JSON(status, ErrorResponse{Error: msg})
}
