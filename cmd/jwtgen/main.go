// Command jwtgen creates an ES256 internal JWT and matching JWKS document.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/pj-hoakari/go-service-template/internal/jwtgen"
)

func main() {
	issuer := flag.String("issuer", "api-gateway", "internal JWT issuer")
	audience := flag.String("audience", "go-service-template", "internal JWT audience")
	tokenUse := flag.String("token-use", "access", "token_use claim")
	scope := flag.String("scope", "greeting.read", "space-delimited scopes")
	kid := flag.String("kid", "test-key", "JWK key ID")
	ttl := flag.Duration("ttl", 2*time.Minute, "token lifetime")

	flag.Parse()

	output, err := jwtgen.Generate(jwtgen.Config{
		Issuer: *issuer, Audience: *audience, TokenUse: *tokenUse,
		Scope: *scope, KeyID: *kid, TTL: *ttl,
	})
	if err != nil {
		log.Print("jwtgen: ", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(output); err != nil {
		log.Print("jwtgen: encode output: ", err)
		os.Exit(1)
	}
}
