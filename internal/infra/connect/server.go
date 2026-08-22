// Package connect provides the Connect HTTP transport for this service.
package connect

import (
	"fmt"
	"log/slog"
	"net/http"

	connectrpc "connectrpc.com/connect"
	"connectrpc.com/otelconnect"

	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

func NewHandler(greetService application.GreetUseCases) (http.Handler, error) {
	return NewHandlerWithJWKSURL(greetService, jwks.DefaultInternalJWKSURL)
}

func NewHandlerWithJWKSURL(greetService application.GreetUseCases, jwksURL string) (http.Handler, error) {
	return NewHandlerWithValidator(greetService, jwks.NewJWKSValidator(jwksURL, internalJWTIssuer, internalJWTAudience))
}

func NewHandlerWithValidator(greetService application.GreetUseCases, validator JWTValidator) (http.Handler, error) {
	// The caller sits behind the API Gateway, so an incoming trace context is
	// trusted and continued instead of being demoted to a span link.
	tracing, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
	if err != nil {
		return nil, fmt.Errorf("create tracing interceptor: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	path, handler := greetv1connect.NewGreetServiceHandlerWithAuthz(
		NewService(greetService),
		newGreetAuthzVerifier(validator),
		connectrpc.WithInterceptors(tracing, newTenantPublicIDInterceptor(validator)),
	)
	mux.Handle(path, handler)

	return mux, nil
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		slog.ErrorContext(r.Context(), "healthz response write failed", "error", err)
	}
}
