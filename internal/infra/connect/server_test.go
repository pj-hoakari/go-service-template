package connect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	connectrpc "connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
	"github.com/pj-hoakari/go-service-template/internal/jwtgen"
	"github.com/pj-hoakari/go-service-template/internal/tenantctx"
)

// newTestJWKSValidator builds a validator backed by an httptest JWKS endpoint
// serving keys, mirroring the API Gateway publishing its signing keys.
func newTestJWKSValidator(t *testing.T, keys jwtgen.JWKS) *jwks.JWKSValidator {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(keys); err != nil {
			t.Errorf("encode JWKS: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	return jwks.NewJWKSValidator(server.URL, internalJWTIssuer, internalJWTAudience)
}

// newTestHandlerWithJWKS builds the production handler wired to a test JWKS
// validator trusting keys.
func newTestHandlerWithJWKS(t *testing.T, keys jwtgen.JWKS) http.Handler {
	t.Helper()

	handler, err := NewHandlerWithValidator(application.NewGreetService(nopGreetingRepository{}), newTestJWKSValidator(t, keys))
	if err != nil {
		t.Fatalf("NewHandlerWithValidator() error = %v", err)
	}

	return handler
}

// mintInternalJWT issues an internal JWT signed by a fresh key, returning the
// Authorization header value and the JWKS document publishing the key. An
// empty tenantPublicID omits the tenant_id claim.
func mintInternalJWT(t *testing.T, tokenUse, scope, tenantPublicID string) (string, jwtgen.JWKS) {
	t.Helper()

	output, err := jwtgen.Generate(jwtgen.Config{
		Issuer:         internalJWTIssuer,
		Audience:       internalJWTAudience,
		TokenUse:       tokenUse,
		TenantPublicID: tenantPublicID,
		Scope:          scope,
		KeyID:          "test-key",
		TTL:            time.Hour,
	})
	if err != nil {
		t.Fatalf("generate internal JWT: %v", err)
	}

	return "Bearer " + output.Token, output.JWKS
}

func TestGreetServiceAuthz(t *testing.T) {
	t.Parallel()

	authorization, keys := mintInternalJWT(t, internalTokenUseAccess, "greeting.read", "a1b2c3d4e5f60718")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, keys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	t.Run("rejects missing bearer token", func(t *testing.T) {
		_, err := client.Greet(context.Background(), connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"}))
		if connectrpc.CodeOf(err) != connectrpc.CodeUnauthenticated {
			t.Fatalf("Greet() error code = %v, want %v", connectrpc.CodeOf(err), connectrpc.CodeUnauthenticated)
		}
	})

	t.Run("accepts internal JWT with required scope", func(t *testing.T) {
		req := connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
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

	authorization, keys := mintInternalJWT(t, internalTokenUseAccess, "greeting.write", "a1b2c3d4e5f60718")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, keys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	req := connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
	req.Header().Set("Authorization", authorization)

	_, err := client.Greet(context.Background(), req)
	if got, want := connectrpc.CodeOf(err), connectrpc.CodePermissionDenied; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}
}

func TestGreetServiceAuthzRejectsUnknownSigningKey(t *testing.T) {
	t.Parallel()

	// The handler trusts a different JWKS than the one that signed this token.
	_, trustedKeys := mintInternalJWT(t, internalTokenUseAccess, "greeting.read", "a1b2c3d4e5f60718")
	foreignAuthorization, _ := mintInternalJWT(t, internalTokenUseAccess, "greeting.read", "a1b2c3d4e5f60718")
	httpServer := httptest.NewServer(newTestHandlerWithJWKS(t, trustedKeys))
	t.Cleanup(httpServer.Close)
	client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

	req := connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
	req.Header().Set("Authorization", foreignAuthorization)

	_, err := client.Greet(context.Background(), req)
	if got, want := connectrpc.CodeOf(err), connectrpc.CodeUnauthenticated; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}
}

// tenantEchoService echoes the tenant public ID found in the request context,
// letting the test observe the interceptor end to end without changing the
// real greet service.
type tenantEchoService struct {
	greetv1connect.UnimplementedGreetServiceHandler
}

func (tenantEchoService) Greet(ctx context.Context, _ *connectrpc.Request[greetv1.GreetRequest]) (*connectrpc.Response[greetv1.GreetResponse], error) {
	tenantPublicID, _ := tenantctx.TenantPublicIDFromContext(ctx)

	return connectrpc.NewResponse(&greetv1.GreetResponse{Greeting: tenantPublicID}), nil
}

func TestGreetServiceInjectsTenantPublicID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		tenantPublicID string
		want           string
		wantCode       connectrpc.Code
	}{
		{
			name:           "handler reads tenant ID injected from tenant_id claim",
			tenantPublicID: "a1b2c3d4e5f60718",
			want:           "a1b2c3d4e5f60718",
		},
		{
			name:           "rejects token without tenant_id claim",
			tenantPublicID: "",
			wantCode:       connectrpc.CodeUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorization, keys := mintInternalJWT(t, internalTokenUseAccess, "greeting.read", tt.tenantPublicID)
			validator := newTestJWKSValidator(t, keys)

			mux := http.NewServeMux()
			path, handler := greetv1connect.NewGreetServiceHandlerWithAuthz(
				tenantEchoService{},
				newGreetAuthzVerifier(validator),
				connectrpc.WithInterceptors(newTenantPublicIDInterceptor(validator)),
			)
			mux.Handle(path, handler)

			httpServer := httptest.NewServer(mux)
			t.Cleanup(httpServer.Close)
			client := greetv1connect.NewGreetServiceClient(httpServer.Client(), httpServer.URL)

			req := connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
			req.Header().Set("Authorization", authorization)

			res, err := client.Greet(context.Background(), req)

			if tt.wantCode != 0 {
				if connectrpc.CodeOf(err) != tt.wantCode {
					t.Fatalf("Greet() error code = %v, want %v", connectrpc.CodeOf(err), tt.wantCode)
				}

				return
			}

			if err != nil {
				t.Fatalf("Greet() error = %v", err)
			}

			if got := res.Msg.GetGreeting(); got != tt.want {
				t.Errorf("tenant ID echoed by handler = %q, want %q", got, tt.want)
			}
		})
	}
}
