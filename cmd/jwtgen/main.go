// Command jwtgen creates an ES256 internal JWT and matching JWKS document.
package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/pj-hoakari/go-service-template/internal/jwtgen"
)

func main() {
	issuer := flag.String("issuer", "api-gateway", "internal JWT issuer")
	audience := flag.String("audience", "go-service-template", "internal JWT audience")
	tokenUse := flag.String("token-use", "access", "token_use claim")
	tenantPublicID := flag.String("tenant-public-id", "", "tenant public ID (16-character hex; tenant_id claim is omitted when empty)")
	scope := flag.String("scope", "greeting.read", "space-delimited scopes")
	kid := flag.String("kid", "test-key", "JWK key ID")
	ttl := flag.Duration("ttl", 2*time.Minute, "token lifetime")

	flag.Parse()

	// A local CLI reports its failures to a person on stderr, so plain text
	// is more useful here than the service's structured output.
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	output, err := jwtgen.Generate(jwtgen.Config{
		Issuer: *issuer, Audience: *audience, TokenUse: *tokenUse,
		TenantPublicID: *tenantPublicID, Scope: *scope, KeyID: *kid, TTL: *ttl,
	})
	if err != nil {
		logger.Error("generate token failed", "error", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(output); err != nil {
		logger.Error("encode output failed", "error", err)
		os.Exit(1)
	}
}
