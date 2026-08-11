package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pj-hoakari/go-service-template/internal/application"
	connectinfra "github.com/pj-hoakari/go-service-template/internal/infra/connect"
	dbinfra "github.com/pj-hoakari/go-service-template/internal/infra/db"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

const (
	defaultAddr       = ":8080"
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := getenv("SERVER_ADDR", defaultAddr)
	jwksURL := getenv("INTERNAL_JWKS_URL", jwks.DefaultInternalJWKSURL)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	db, err := openDatabase(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("go-service-template: close database: %v", err)
		}
	}()

	greetService := application.NewGreetService(dbinfra.NewPostgresGreetingRepository(db))
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           connectinfra.NewHandlerWithJWKSURL(greetService, jwksURL),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serveErr := make(chan error, 1)

	go func() {
		log.Printf("go-service-template: server listening on %s", addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err

			return
		}

		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Print("go-service-template: server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return httpServer.Shutdown(shutdownCtx)
	}
}

func openDatabase(ctx context.Context, databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping database: %w; close database: %v", err, closeErr)
		}

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
