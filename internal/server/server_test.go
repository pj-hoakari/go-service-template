package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/jwtgen"
)

// newTestHandlerWithJWKS builds a handler backed by an httptest JWKS endpoint
// serving keys, mirroring the API Gateway publishing its signing keys.
func newTestHandlerWithJWKS(t *testing.T, keys jwtgen.JWKS) http.Handler {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(keys); err != nil {
			t.Errorf("encode JWKS: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	return NewHandlerWithJWKSURL(server.URL)
}

// mintInternalJWT issues an internal JWT signed by a fresh key, returning the
// Authorization header value and the JWKS document publishing the key.
func mintInternalJWT(t *testing.T, tokenUse, scope string) (string, jwtgen.JWKS) {
	t.Helper()

	output, err := jwtgen.Generate(jwtgen.Config{
		Issuer:   internalJWTIssuer,
		Audience: internalJWTAudience,
		TokenUse: tokenUse,
		Scope:    scope,
		KeyID:    "test-key",
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatalf("generate internal JWT: %v", err)
	}

	return "Bearer " + output.Token, output.JWKS
}

func TestGreetServiceAuthz(t *testing.T) {
	t.Parallel()

	authorization, keys := mintInternalJWT(t, internalTokenUseAccess, "greeting.read")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, keys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	t.Run("rejects missing bearer token", func(t *testing.T) {
		_, err := client.Greet(context.Background(), connect.NewRequest(&greetv1.GreetRequest{Name: "Ada"}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("Greet() error code = %v, want %v", connect.CodeOf(err), connect.CodeUnauthenticated)
		}
	})

	t.Run("accepts internal JWT with required scope", func(t *testing.T) {
		req := connect.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
		req.Header().Set("Authorization", authorization)

		res, err := client.Greet(context.Background(), req)
		if err != nil {
			t.Fatalf("Greet() error = %v", err)
		}

		if got, want := res.Msg.GetGreeting(), "Hello, Ada!"; got != want {
			t.Errorf("Greeting = %q, want %q", got, want)
		}
	})
}

func TestGreetServiceAuthzRejectsMissingScope(t *testing.T) {
	t.Parallel()

	authorization, keys := mintInternalJWT(t, internalTokenUseAccess, "greeting.write")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, keys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	req := connect.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
	req.Header().Set("Authorization", authorization)

	_, err := client.Greet(context.Background(), req)
	if got, want := connect.CodeOf(err), connect.CodePermissionDenied; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}
}

func TestGreetServiceAuthzRejectsUnknownSigningKey(t *testing.T) {
	t.Parallel()

	// The handler trusts a different JWKS than the one that signed this token.
	_, trustedKeys := mintInternalJWT(t, internalTokenUseAccess, "greeting.read")
	foreignAuthorization, _ := mintInternalJWT(t, internalTokenUseAccess, "greeting.read")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, trustedKeys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	req := connect.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
	req.Header().Set("Authorization", foreignAuthorization)

	_, err := client.Greet(context.Background(), req)
	if got, want := connect.CodeOf(err), connect.CodeUnauthenticated; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}
}
