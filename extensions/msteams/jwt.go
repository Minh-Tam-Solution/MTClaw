package msteams

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	botFrameworkOpenIDURL = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	botFrameworkIssuer    = "https://api.botframework.com"
	jwksCacheTTL          = 24 * time.Hour
)

// jwksCache holds fetched RSA public keys from the Bot Framework JWKS endpoint.
// Keys rotate infrequently — 24-hour TTL is appropriate.
type jwksCache struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey // kid → RSA public key
	expiry  time.Time
	httpCli *http.Client
}

func newJWKSCache() *jwksCache {
	return &jwksCache{
		keys:    make(map[string]*rsa.PublicKey),
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

// openIDConfig is the minimal subset of Bot Framework OpenID metadata we need.
type openIDConfig struct {
	JWKSURI string `json:"jwks_uri"`
}

// jwksDocument is the Bot Framework JWKS response.
type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

// jwk is a single JSON Web Key (RSA public key fields).
type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"` // base64url-encoded modulus
	E   string `json:"e"` // base64url-encoded exponent
}

// GetKey returns the RSA public key for the given key ID, refreshing the cache if needed.
func (c *jwksCache) GetKey(kid string) (*rsa.PublicKey, error) {
	// Fast path: read-lock if still valid and key is present
	c.mu.RLock()
	if time.Now().Before(c.expiry) {
		if k, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return k, nil
		}
	}
	c.mu.RUnlock()

	// Slow path: refresh
	if err := c.refresh(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("msteams: JWT key id %q not found in Bot Framework JWKS", kid)
	}
	return k, nil
}

// refresh fetches the Bot Framework OpenID metadata → JWKS → RSA keys.
func (c *jwksCache) refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-checked: another goroutine may have refreshed while we waited for the lock
	if time.Now().Before(c.expiry) {
		return nil
	}

	jwksURI, err := c.fetchJWKSURI()
	if err != nil {
		return err
	}

	keys, err := c.fetchKeys(jwksURI)
	if err != nil {
		return err
	}

	c.keys = keys
	c.expiry = time.Now().Add(jwksCacheTTL)
	return nil
}

// fetchJWKSURI fetches the Bot Framework OpenID metadata and returns the jwks_uri.
func (c *jwksCache) fetchJWKSURI() (string, error) {
	resp, err := c.httpCli.Get(botFrameworkOpenIDURL)
	if err != nil {
		return "", fmt.Errorf("msteams: fetch OpenID metadata: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("msteams: read OpenID metadata: %w", err)
	}

	var meta openIDConfig
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("msteams: parse OpenID metadata: %w", err)
	}
	if meta.JWKSURI == "" {
		return "", fmt.Errorf("msteams: missing jwks_uri in OpenID metadata")
	}
	return meta.JWKSURI, nil
}

// fetchKeys fetches the JWKS from jwksURI and builds a kid→RSAPublicKey map.
func (c *jwksCache) fetchKeys(jwksURI string) (map[string]*rsa.PublicKey, error) {
	resp, err := c.httpCli.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("msteams: fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil, fmt.Errorf("msteams: read JWKS: %w", err)
	}

	var doc jwksDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("msteams: parse JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Use != "sig" || k.N == "" || k.E == "" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			continue // skip malformed keys
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("msteams: no valid RSA sig keys found in JWKS")
	}
	return keys, nil
}

// parseRSAPublicKey builds an *rsa.PublicKey from base64url-encoded N and E values.
func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("msteams: decode RSA N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("msteams: decode RSA E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// jwtKeyResolver wraps jwksCache for use as a jwt.Keyfunc.
type jwtKeyResolver struct {
	cache *jwksCache
}

func (r *jwtKeyResolver) Keyfunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("msteams: unexpected JWT signing method: %v", token.Header["alg"])
	}
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("msteams: JWT missing kid header")
	}
	return r.cache.GetKey(kid)
}

// ValidateBotFrameworkJWT verifies an inbound Bot Framework JWT.
// Validates: RSA signature, iss == botFrameworkIssuer, aud == appID, expiry.
func ValidateBotFrameworkJWT(tokenStr, appID string, keyCache *jwksCache) error {
	resolver := &jwtKeyResolver{cache: keyCache}

	token, err := jwt.Parse(tokenStr, resolver.Keyfunc,
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(botFrameworkIssuer),
		jwt.WithAudience(appID),
	)
	if err != nil {
		return fmt.Errorf("msteams: JWT validation failed: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("msteams: JWT is not valid")
	}
	return nil
}
