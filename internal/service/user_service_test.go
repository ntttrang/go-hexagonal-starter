package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
	"github.com/nttttranggo-hexagonal-starter/internal/service"
)

type mockRepo struct {
	users   map[uuid.UUID]*domain.User
	byEmail map[string]*domain.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:   make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *mockRepo) Create(_ context.Context, user *domain.User) error {
	if _, ok := m.byEmail[user.Email]; ok {
		return domain.ErrConflict
	}
	cp := *user
	m.users[user.ID] = &cp
	m.byEmail[user.Email] = &cp
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *mockRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *mockRepo) List(_ context.Context, limit, offset int) ([]domain.User, error) {
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, *u)
	}
	if offset >= len(out) {
		return []domain.User{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (m *mockRepo) Update(_ context.Context, user *domain.User) error {
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

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	u, ok := m.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.byEmail, u.Email)
	delete(m.users, id)
	return nil
}

type mockTokens struct{}

func (m *mockTokens) Issue(claims domain.TokenClaims, _ time.Duration) (string, error) {
	return "token-" + claims.UserID.String(), nil
}

func (m *mockTokens) Parse(token string) (*domain.TokenClaims, error) {
	return nil, errors.New("not implemented")
}

func TestRegisterAndAuthenticate(t *testing.T) {
	svc := service.NewUserService(newMockRepo(), &mockTokens{}, time.Hour, nil)

	user, err := svc.Register(context.Background(), domain.RegisterInput{
		Email:    "Alice@Example.com",
		Name:     "Alice",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)

	_, err = svc.Register(context.Background(), domain.RegisterInput{
		Email:    "alice@example.com",
		Name:     "Alice",
		Password: "password123",
	})
	assert.ErrorIs(t, err, domain.ErrConflict)

	token, got, err := svc.Authenticate(context.Background(), "alice@example.com", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, user.ID, got.ID)

	_, _, err = svc.Authenticate(context.Background(), "alice@example.com", "wrong")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestRegisterValidation(t *testing.T) {
	svc := service.NewUserService(newMockRepo(), &mockTokens{}, time.Hour, nil)

	_, err := svc.Register(context.Background(), domain.RegisterInput{
		Email: "bad", Name: "A", Password: "short",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestUpdateAndDelete(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewUserService(repo, &mockTokens{}, time.Hour, nil)

	user, err := svc.Register(context.Background(), domain.RegisterInput{
		Email: "bob@example.com", Name: "Bob", Password: "password123",
	})
	require.NoError(t, err)

	newName := "Bobby"
	updated, err := svc.Update(context.Background(), user.ID, domain.UpdateUserInput{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Bobby", updated.Name)

	err = svc.Delete(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), user.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestPasswordHashing(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("password123")))
}
