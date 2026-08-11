package server

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

const (
	internalJWTIssuer   = "api-gateway"
	internalJWTAudience = "go-service-template"
)

// internalTokenUseAccess is the token_use claim every authenticated RPC
// expects in this template. A service that issues several token kinds (for
// example user access vs service-to-service tokens) can vary the expected
// token_use per RPC by switching on the procedure name
// (e.g. greetv1connect.GreetServiceGreetProcedure).
const internalTokenUseAccess = "access"

// newGreetAuthzVerifier adapts the internal JWT validator to the Verifier
// generated from authz policy annotations. RPCs marked AUTH_LEVEL_PUBLIC skip
// verification; every other RPC requires a valid internal JWT whose scopes
// cover the policy's required scopes.
func newGreetAuthzVerifier(validator JWTValidator) greetv1connect.Verifier {
	return greetv1connect.VerifierFunc(func(ctx context.Context, policy greetv1connect.AuthPolicy) error {
		if policy.Level == greetv1connect.AuthLevelPublic {
			return nil
		}

		callInfo, ok := connect.CallInfoForHandlerContext(ctx)
		if !ok {
			return connect.NewError(connect.CodeUnauthenticated, nil)
		}

		return authorizeInternalJWT(ctx, validator, callInfo.RequestHeader().Get("Authorization"), policy.RequiredScopes)
	})
}

func authorizeInternalJWT(ctx context.Context, validator JWTValidator, authorization string, requiredScopes []string) error {
	claims, err := validator.Claims(ctx, authorization)
	if err != nil || claims.TokenUse != internalTokenUseAccess {
		return connect.NewError(connect.CodeUnauthenticated, nil)
	}

	for _, requiredScope := range requiredScopes {
		if !hasScope(claims.Scope, requiredScope) {
			return connect.NewError(connect.CodePermissionDenied, nil)
		}
	}

	return nil
}

// tenantPublicIDFromClaims extracts the tenant's 16-character hexadecimal
// public ID from the tenant_id JWT claim. The ID is only meaningful on access
// tokens; tenant-independent tokens omit the claim, so ok reports whether a
// usable tenant ID is present.
func tenantPublicIDFromClaims(claims jwks.InternalJWTClaims) (string, bool) {
	tenantPublicID := strings.TrimSpace(claims.TenantPublicID)

	return tenantPublicID, claims.TokenUse == internalTokenUseAccess && tenantPublicID != ""
}

func hasScope(scope, requiredScope string) bool {
	for _, grantedScope := range strings.Fields(scope) {
		if grantedScope == requiredScope {
			return true
		}
	}

	return false
}
