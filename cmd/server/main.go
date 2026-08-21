package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	db, err := dbinfra.Open(ctx, databaseURL)
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
