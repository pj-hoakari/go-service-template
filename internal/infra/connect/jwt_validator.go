package connect

//go:generate go tool mockgen -source=jwt_validator.go -destination=mock_jwt_validator_test.go -package=connect

import (
	"context"

	"github.com/pj-hoakari/go-service-template/internal/jwks"
)

// JWTValidator provides verified internal JWT claims to the Connect transport.
type JWTValidator interface {
	Claims(context.Context, string) (jwks.InternalJWTClaims, error)
}
