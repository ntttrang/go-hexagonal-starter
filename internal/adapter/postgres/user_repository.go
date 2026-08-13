package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/nttttranggo-hexagonal-starter/internal/domain"
)

const tracerName = "github.com/nttttranggo-hexagonal-starter/internal/adapter/postgres"

// UserRepository implements domain.UserRepository with pgx.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a Postgres-backed user repository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const q = `
		INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	ctx, span := startDBSpan(ctx, "INSERT", q)
	defer span.End()

	_, err := r.pool.Exec(ctx, q,
		user.ID, user.Email, user.Name, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			recordDBResult(span, domain.ErrConflict)
			return domain.ErrConflict
		}
		recordDBResult(span, err)
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByID fetches a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE id = $1`

	return r.scanOne(ctx, "SELECT", q, id)
}

// GetByEmail fetches a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE email = $1`

	return r.scanOne(ctx, "SELECT", q, email)
}

// List returns a page of users ordered by created_at desc.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	const q = `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	ctx, span := startDBSpan(ctx, "SELECT", q)
	defer span.End()

	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		recordDBResult(span, err)
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			recordDBResult(span, err)
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		recordDBResult(span, err)
		return nil, fmt.Errorf("rows: %w", err)
	}
	span.SetAttributes(attribute.Int("db.rows_affected", len(users)))
	return users, nil
}

// Update persists mutable user fields.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const q = `
		UPDATE users
		SET email = $2, name = $3, updated_at = $4
		WHERE id = $1`

	ctx, span := startDBSpan(ctx, "UPDATE", q)
	defer span.End()

	tag, err := r.pool.Exec(ctx, q, user.ID, user.Email, user.Name, user.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			recordDBResult(span, domain.ErrConflict)
			return domain.ErrConflict
		}
		recordDBResult(span, err)
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		recordDBResult(span, domain.ErrNotFound)
		return domain.ErrNotFound
	}
	span.SetAttributes(attribute.Int64("db.rows_affected", tag.RowsAffected()))
	return nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM users WHERE id = $1`

	ctx, span := startDBSpan(ctx, "DELETE", q)
	defer span.End()

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		recordDBResult(span, err)
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		recordDBResult(span, domain.ErrNotFound)
		return domain.ErrNotFound
	}
	span.SetAttributes(attribute.Int64("db.rows_affected", tag.RowsAffected()))
	return nil
}

// Ping checks database connectivity.
func (r *UserRepository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ctx, span := startDBSpan(ctx, "PING", "SELECT 1")
	defer span.End()

	err := r.pool.Ping(ctx)
	recordDBResult(span, err)
	return err
}

func (r *UserRepository) scanOne(ctx context.Context, op, q string, arg any) (*domain.User, error) {
	ctx, span := startDBSpan(ctx, op, q)
	defer span.End()

	var u domain.User
	err := r.pool.QueryRow(ctx, q, arg).Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			recordDBResult(span, domain.ErrNotFound)
			return nil, domain.ErrNotFound
		}
		recordDBResult(span, err)
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &u, nil
}

func startDBSpan(ctx context.Context, operation, statement string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, "db."+operation+" users",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBOperation(operation),
			attribute.String("db.collection.name", "users"),
			semconv.DBStatement(truncateSQL(statement)),
		),
	)
}

func recordDBResult(span trace.Span, err error) {
	if err == nil {
		return
	}
	// Expected domain outcomes are not treated as span failures.
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
		span.SetAttributes(attribute.String("db.result", err.Error()))
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func truncateSQL(q string) string {
	const max = 256
	if len(q) <= max {
		return q
	}
	return q[:max] + "…"
}
