// Package main is the composition root for the go-hexagonal-starter API.
//
//	@title			Go Hexagonal Starter API
//	@version		1.0
//	@description	Production-ready Go microservice using Hexagonal Architecture.
//	@host			localhost:8085
//	@BasePath		/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	_ "github.com/nttttranggo-hexagonal-starter/api/docs"
	authadapter "github.com/nttttranggo-hexagonal-starter/internal/adapter/auth"
	httpadapter "github.com/nttttranggo-hexagonal-starter/internal/adapter/http"
	"github.com/nttttranggo-hexagonal-starter/internal/adapter/postgres"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/config"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/database"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/logger"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/metrics"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/tracing"
	"github.com/nttttranggo-hexagonal-starter/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load .env when present. Existing environment variables (shell, Docker
	// Compose, etc.) take precedence over file values; .env only fills gaps.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.NewWithOptions(logger.Options{
		Level:   cfg.LogLevel,
		Service: cfg.ServiceName,
		Env:     cfg.Env,
	})
	log.Info("starting api", "env", cfg.Env, "port", cfg.Port, "tracing", cfg.TracingEnabled())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	shutdownTracing, err := tracing.Init(ctx, tracing.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		Environment:    cfg.Env,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		SampleRatio:    cfg.TraceSampleRatio,
	})
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			log.Error("tracing shutdown", "error", err)
		}
	}()

	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(cfg, log); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	m := metrics.New()
	go collectDBPoolMetrics(pool, m)

	tokens := authadapter.NewJWTIssuer(cfg.JWTSecret, cfg.JWTIssuer)
	repo := postgres.NewUserRepository(pool)
	userSvc := service.NewUserService(repo, tokens, cfg.JWTExpiry, m)

	router := httpadapter.NewRouter(httpadapter.Dependencies{
		Log:         log,
		Metrics:     m,
		Tokens:      tokens,
		Auth:        httpadapter.NewAuthHandler(userSvc, log),
		Users:       httpadapter.NewUserHandler(userSvc, log),
		Health:      httpadapter.NewHealthHandler(pool),
		Env:         cfg.Env,
		ServiceName: cfg.ServiceName,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("server stopped gracefully")
	return nil
}

func collectDBPoolMetrics(pool *pgxpool.Pool, m *metrics.Metrics) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		m.ObserveDBPool(pool.Stat())
		<-ticker.C
	}
}
