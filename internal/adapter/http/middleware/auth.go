package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

const (
	// ContextUserIDKey is the gin context key for the authenticated user ID.
	ContextUserIDKey = "user_id"
	// ContextUserEmailKey is the gin context key for the authenticated user email.
	ContextUserEmailKey = "user_email"
)

// Auth returns JWT authentication middleware.
func Auth(tokens domain.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokens.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserEmailKey, claims.Email)
		c.Next()
	}
}
