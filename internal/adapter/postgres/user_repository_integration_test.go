//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	pgadapter "github.com/nttttranggo-hexagonal-starter/internal/adapter/postgres"
	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

func TestUserRepository_CRUD(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/go_hexagonal?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping integration test: database not ready: %v", err)
	}

	migrationsDir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err)

	m, err := migrate.New("file://"+migrationsDir, dsn)
	require.NoError(t, err)
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_, _ = m.Close()
		require.NoError(t, err)
	}
	_, _ = m.Close()

	repo := pgadapter.NewUserRepository(pool)
	now := time.Now().UTC().Truncate(time.Millisecond)
	user := &domain.User{
		ID:           uuid.New(),
		Email:        "dave-" + uuid.New().String() + "@example.com",
		Name:         "Dave",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	require.NoError(t, repo.Create(ctx, user))

	got, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, user.ID, got.ID)

	user.Name = "David"
	user.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.Update(ctx, user))

	list, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, list)

	require.NoError(t, repo.Delete(ctx, user.ID))
	_, err = repo.GetByID(ctx, user.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
