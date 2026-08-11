// Package connect provides the Connect HTTP transport for this service.
package connect

import (
	"log"
	"net/http"

	connectrpc "connectrpc.com/connect"

	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

func NewHandler(greetService application.GreetUseCases) http.Handler {
	return NewHandlerWithJWKSURL(greetService, jwks.DefaultInternalJWKSURL)
}

func NewHandlerWithJWKSURL(greetService application.GreetUseCases, jwksURL string) http.Handler {
	return NewHandlerWithValidator(greetService, jwks.NewJWKSValidator(jwksURL, internalJWTIssuer, internalJWTAudience))
}

func NewHandlerWithValidator(greetService application.GreetUseCases, validator JWTValidator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	path, handler := greetv1connect.NewGreetServiceHandlerWithAuthz(
		NewService(greetService),
		newGreetAuthzVerifier(validator),
		connectrpc.WithInterceptors(newTenantPublicIDInterceptor(validator)),
	)
	mux.Handle(path, handler)

	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		log.Printf("healthz response write: %v", err)
	}
}
