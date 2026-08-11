package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/mock/gomock"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
	"github.com/pj-hoakari/go-service-template/internal/tenantctx"
)

func TestTenantPublicIDInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		claims    jwks.InternalJWTClaims
		claimsErr error
		wantID    string
		wantCode  connect.Code
	}{
		{
			name:      "injects tenant ID from access token",
			claims:    jwks.InternalJWTClaims{TokenUse: internalTokenUseAccess, TenantPublicID: "a1b2c3d4e5f60718"},
			claimsErr: nil,
			wantID:    "a1b2c3d4e5f60718",
		},
		{
			name:      "trims surrounding whitespace",
			claims:    jwks.InternalJWTClaims{TokenUse: internalTokenUseAccess, TenantPublicID: " a1b2c3d4e5f60718 "},
			claimsErr: nil,
			wantID:    "a1b2c3d4e5f60718",
		},
		{
			name:      "rejects tenant-independent token",
			claims:    jwks.InternalJWTClaims{TokenUse: internalTokenUseAccess},
			claimsErr: nil,
			wantCode:  connect.CodeUnauthenticated,
		},
		{
			name:      "rejects non-access token use",
			claims:    jwks.InternalJWTClaims{TokenUse: "service", TenantPublicID: "a1b2c3d4e5f60718"},
			claimsErr: nil,
			wantCode:  connect.CodeUnauthenticated,
		},
		{
			name:      "rejects invalid token",
			claims:    jwks.InternalJWTClaims{},
			claimsErr: errors.New("invalid internal JWT"),
			wantCode:  connect.CodeUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			controller := gomock.NewController(t)
			validator := NewMockJWTValidator(controller)
			validator.EXPECT().Claims(gomock.Any(), "Bearer test-token").Return(tt.claims, tt.claimsErr)

			nextCalled := false

			var gotID string

			next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
				nextCalled = true
				gotID, _ = tenantctx.TenantPublicIDFromContext(ctx)

				return connect.NewResponse(&greetv1.GreetResponse{}), nil
			})

			req := connect.NewRequest(&greetv1.GreetRequest{Name: "Ada"})
			req.Header().Set("Authorization", "Bearer test-token")

			_, err := newTenantPublicIDInterceptor(validator).WrapUnary(next)(context.Background(), req)

			if tt.wantCode != 0 {
				if connect.CodeOf(err) != tt.wantCode {
					t.Fatalf("interceptor error code = %v, want %v", connect.CodeOf(err), tt.wantCode)
				}

				if nextCalled {
					t.Fatal("next handler was called for a rejected request")
				}

				return
			}

			if err != nil {
				t.Fatalf("interceptor error = %v", err)
			}

			if gotID != tt.wantID {
				t.Errorf("tenant ID = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}
