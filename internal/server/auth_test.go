package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/mock/gomock"

	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

func TestAuthorizeInternalJWT(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)
	validator := NewMockJWTValidator(controller)
	validator.EXPECT().Claims(gomock.Any(), "Bearer test-token").Return(jwks.InternalJWTClaims{
		TokenUse: internalTokenUseAccess,
		Scope:    "greeting.read",
	}, nil)

	err := authorizeInternalJWT(context.Background(), validator, "Bearer test-token", []string{"greeting.read"})
	if err != nil {
		t.Fatalf("authorizeInternalJWT() error = %v", err)
	}
}

func TestAuthorizeInternalJWTRejectsInvalidToken(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)
	validator := NewMockJWTValidator(controller)
	validator.EXPECT().Claims(gomock.Any(), "Bearer bad-token").Return(jwks.InternalJWTClaims{}, errors.New("invalid internal JWT"))

	err := authorizeInternalJWT(context.Background(), validator, "Bearer bad-token", nil)
	if got, want := connect.CodeOf(err), connect.CodeUnauthenticated; got != want {
		t.Errorf("error code = %v, want %v", got, want)
	}
}

func TestAuthorizeInternalJWTRejectsTokenUse(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)
	validator := NewMockJWTValidator(controller)
	validator.EXPECT().Claims(gomock.Any(), "Bearer service-token").Return(jwks.InternalJWTClaims{TokenUse: "service"}, nil)

	err := authorizeInternalJWT(context.Background(), validator, "Bearer service-token", nil)
	if got, want := connect.CodeOf(err), connect.CodeUnauthenticated; got != want {
		t.Errorf("error code = %v, want %v", got, want)
	}
}

func TestAuthorizeInternalJWTRejectsMissingScope(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)
	validator := NewMockJWTValidator(controller)
	validator.EXPECT().Claims(gomock.Any(), "Bearer test-token").Return(jwks.InternalJWTClaims{
		TokenUse: internalTokenUseAccess,
		Scope:    "greeting.write",
	}, nil)

	err := authorizeInternalJWT(context.Background(), validator, "Bearer test-token", []string{"greeting.read"})
	if got, want := connect.CodeOf(err), connect.CodePermissionDenied; got != want {
		t.Errorf("error code = %v, want %v", got, want)
	}
}
