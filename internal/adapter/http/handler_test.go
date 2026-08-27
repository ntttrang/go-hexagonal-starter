package httpadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/nttttranggo-hexagonal-starter/internal/adapter/auth"
	httpadapter "github.com/nttttranggo-hexagonal-starter/internal/adapter/http"
	"github.com/nttttranggo-hexagonal-starter/internal/domain"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/logger"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/metrics"
	"github.com/nttttranggo-hexagonal-starter/internal/service"
)

type memRepo struct {
	users   map[uuid.UUID]*domain.User
	byEmail map[string]*domain.User
}

func newMemRepo() *memRepo {
	return &memRepo{users: map[uuid.UUID]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (m *memRepo) Create(_ context.Context, user *domain.User) error {
	if _, ok := m.byEmail[user.Email]; ok {
		return domain.ErrConflict
	}
	cp := *user
	m.users[user.ID] = &cp
	m.byEmail[user.Email] = &cp
	return nil
}

func (m *memRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memRepo) List(_ context.Context, _, _ int) ([]domain.User, error) {
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, *u)
	}
	return out, nil
}

func (m *memRepo) Update(_ context.Context, user *domain.User) error {
	existing, ok := m.users[user.ID]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.byEmail, existing.Email)
	cp := *user
	m.users[user.ID] = &cp
	m.byEmail[user.Email] = &cp
	return nil
}

func (m *memRepo) Delete(_ context.Context, id uuid.UUID) error {
	u, ok := m.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.byEmail, u.Email)
	delete(m.users, id)
	return nil
}

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokens := authadapter.NewJWTIssuer("test-secret-at-least-16-chars", "test")
	repo := newMemRepo()
	svc := service.NewUserService(repo, tokens, time.Hour, nil, nil)
	log := logger.New("error")
	m := metrics.New()

	r := httpadapter.NewRouter(httpadapter.Dependencies{
		Log:     log,
		Metrics: m,
		Tokens:  tokens,
		Auth:    httpadapter.NewAuthHandler(svc, log),
		Users:   httpadapter.NewUserHandler(svc, log),
		Health:  httpadapter.NewHealthHandler(nil),
		Env:     "test",
	})
	return r
}

func TestRegisterLoginAndCRUD(t *testing.T) {
	r := setupRouter(t)

	body := map[string]string{
		"email": "carol@example.com", "name": "Carol", "password": "password123",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created httpadapter.UserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "carol@example.com", created.Email)

	loginBody, _ := json.Marshal(map[string]string{
		"email": "carol@example.com", "password": "password123",
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var auth httpadapter.AuthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &auth))
	require.NotEmpty(t, auth.Token)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	updateBody, _ := json.Marshal(map[string]string{"name": "Caroline"})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+created.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestHealthz(t *testing.T) {
	r := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
