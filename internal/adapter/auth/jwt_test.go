package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nttttranggo-hexagonal-starter/internal/adapter/auth"
	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

func TestJWTIssuer_IssueAndParse(t *testing.T) {
	issuer := auth.NewJWTIssuer("super-secret-key-16+", "test-issuer")
	id := uuid.New()

	token, err := issuer.Issue(domain.TokenClaims{UserID: id, Email: "a@b.com"}, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := issuer.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, id, claims.UserID)
	assert.Equal(t, "a@b.com", claims.Email)

	_, err = issuer.Parse("not-a-token")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}
