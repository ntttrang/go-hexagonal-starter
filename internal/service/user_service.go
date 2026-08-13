package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/metrics"
)

// UserService implements domain.UserService.
type UserService struct {
	repo    domain.UserRepository
	tokens  domain.TokenIssuer
	ttl     time.Duration
	metrics *metrics.Metrics
}

// NewUserService constructs a UserService. metrics may be nil.
func NewUserService(repo domain.UserRepository, tokens domain.TokenIssuer, ttl time.Duration, m *metrics.Metrics) *UserService {
	return &UserService{repo: repo, tokens: tokens, ttl: ttl, metrics: m}
}

// Register creates a new user with a hashed password.
func (s *UserService) Register(ctx context.Context, in domain.RegisterInput) (*domain.User, error) {
	if err := domain.ValidateRegister(in); err != nil {
		s.incUserOp("register", "invalid")
		return nil, err
	}

	email := domain.NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.Name)
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		s.incUserOp("register", "error")
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		s.incUserOp("register", "conflict")
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.incUserOp("register", "error")
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		s.incUserOp("register", "error")
		return nil, err
	}
	s.incUserOp("register", "success")
	return user, nil
}

// Authenticate verifies credentials and returns a JWT.
func (s *UserService) Authenticate(ctx context.Context, email, password string) (string, *domain.User, error) {
	email = domain.NormalizeEmail(email)
	if email == "" || password == "" {
		s.incAuth("invalid")
		return "", nil, domain.ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.incAuth("invalid")
			return "", nil, domain.ErrInvalidCredentials
		}
		s.incAuth("error")
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.incAuth("invalid")
		return "", nil, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(domain.TokenClaims{UserID: user.ID, Email: user.Email}, s.ttl)
	if err != nil {
		s.incAuth("error")
		return "", nil, fmt.Errorf("issue token: %w", err)
	}
	s.incAuth("success")
	return token, user, nil
}

// GetByID returns a user by ID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.incUserOp("get", "not_found")
		} else {
			s.incUserOp("get", "error")
		}
		return nil, err
	}
	s.incUserOp("get", "success")
	return user, nil
}

// List returns a page of users.
func (s *UserService) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.incUserOp("list", "error")
		return nil, err
	}
	s.incUserOp("list", "success")
	return users, nil
}

// Update applies partial updates to a user.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.incUserOp("update", "not_found")
		} else {
			s.incUserOp("update", "error")
		}
		return nil, err
	}

	if in.Name != nil {
		name := *in.Name
		if name == "" || len(name) > 100 {
			s.incUserOp("update", "invalid")
			return nil, domain.ErrInvalidInput
		}
		user.Name = name
	}

	if in.Email != nil {
		email := domain.NormalizeEmail(*in.Email)
		if email == "" || !containsAt(email) {
			s.incUserOp("update", "invalid")
			return nil, domain.ErrInvalidInput
		}
		if email != user.Email {
			existing, err := s.repo.GetByEmail(ctx, email)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				s.incUserOp("update", "error")
				return nil, err
			}
			if existing != nil {
				s.incUserOp("update", "conflict")
				return nil, domain.ErrConflict
			}
			user.Email = email
		}
	}

	user.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, user); err != nil {
		s.incUserOp("update", "error")
		return nil, err
	}
	s.incUserOp("update", "success")
	return user, nil
}

// Delete removes a user by ID.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.incUserOp("delete", "not_found")
		} else {
			s.incUserOp("delete", "error")
		}
		return err
	}
	s.incUserOp("delete", "success")
	return nil
}

func (s *UserService) incAuth(result string) {
	if s.metrics != nil {
		s.metrics.IncAuthLogin(result)
	}
}

func (s *UserService) incUserOp(op, result string) {
	if s.metrics != nil {
		s.metrics.IncUserOp(op, result)
	}
}

func containsAt(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return true
		}
	}
	return false
}
