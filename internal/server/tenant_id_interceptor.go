package server

import (
	"context"

	"connectrpc.com/connect"

	"github.com/pj-hoakari/go-service-template/internal/tenantctx"
)

// newTenantPublicIDInterceptor injects the authenticated tenant's public ID
// from the verified internal JWT into the request context, where handlers and
// use cases read it via the tenantctx package. Services are tenant-scoped by
// default, so the interceptor fails closed: a request whose token carries no
// tenant public ID is rejected unless its RPC is listed in
// tenantIDNotRequired. Use-case boundaries still guard with tenantctx.Ensure.
func newTenantPublicIDInterceptor(validator JWTValidator) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if tenantIDNotRequired(req.Spec().Procedure) {
				return next(ctx, req)
			}

			claims, err := validator.Claims(ctx, req.Header().Get("Authorization"))

			tenantPublicID, ok := tenantPublicIDFromClaims(claims)
			if err != nil || !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			return next(tenantctx.WithTenantPublicID(ctx, tenantPublicID), req)
		}
	})
}

// tenantIDNotRequired reports whether an RPC may be called without a
// tenant-carrying token. All RPCs require a tenant public ID by default; a
// service exposing tenant-independent RPCs (self-signup, service-to-service
// calls, PUBLIC endpoints) lists their procedure names here, e.g.
// greetv1connect.GreetServiceGreetProcedure.
func tenantIDNotRequired(_ string) bool {
	return false
}
