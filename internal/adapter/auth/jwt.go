package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

// JWTIssuer implements domain.TokenIssuer using HMAC-SHA256 JWTs.
type JWTIssuer struct {
	secret []byte
	issuer string
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// NewJWTIssuer creates a JWT token issuer.
func NewJWTIssuer(secret, issuer string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer}
}

// Issue creates a signed JWT for the given claims.
func (j *JWTIssuer) Issue(c domain.TokenClaims, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Email: c.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.UserID.String(),
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	return token.SignedString(j.secret)
}

// Parse validates a JWT and returns domain claims.
func (j *JWTIssuer) Parse(tokenStr string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return &domain.TokenClaims{UserID: userID, Email: c.Email}, nil
}
