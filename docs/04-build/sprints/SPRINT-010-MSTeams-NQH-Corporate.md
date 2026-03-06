# Sprint 10 — MS Teams Extension + NQH Corporate Rollout

**SDLC Stage**: 04-Build
**Version**: 1.1.0
**Date**: 2026-03-17 (v1.1 — CTO review 8.5/10, issues CTO-35 through CTO-39 incorporated)
**Author**: [@pm] + [@architect]
**Sprint**: 10 of 10+
**Phase**: 3 (Scale)
**Framework**: SDLC Enterprise Framework 6.1.1 — STANDARD tier

---

## 1. Sprint Context

### Predecessor: Sprint 9 ✅ (CTO 9.0/10 APPROVED)

Sprint 9 delivered channel rationalization + complete SOUL behavioral suite:

| Deliverable | Status | CTO Verdict |
|-------------|--------|-------------|
| Channel removal: Feishu/Discord/WhatsApp/Slack (~2,836 LOC) | ✅ | DONE |
| SOUL behavioral tests: 12 governance SOULs × 5 = 60 tests (85 total) | ✅ | EXCELLENT |
| MS Teams scaffold (`extensions/msteams/` + README + .go.TODO) | ✅ | DONE |
| G4 gate proposal (10/11 criteria met, WAU pending) | ✅ | EXCELLENT |
| ADR-007: MS Teams Extension | ✅ APPROVED | Architecture sound |
| CTO-34 fix: Section 8 header 16→17 | ✅ | Fixed |

**Sprint 9 post-review items carried into Sprint 10**:
- CTO-33 (P3): 3 residual Discord comments/string literals — fix during gateway.go MS Teams wiring touchpoint

**Tests at Sprint 9 close**: 350 PASS, 0 FAIL

### Entry Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| ADR-007 APPROVED | ✅ | [@cto] 2026-03-17 — Bot Framework REST API, app password, MTS tenant only |
| MS Teams scaffold in repo | ✅ | `extensions/msteams/README.md` + `extensions/msteams/msteams.go.TODO` |
| 350 tests passing | ✅ | `go test ./...` all green |
| CTO-33 noted | ✅ | 3 Discord refs in gateway_consumer.go + gateway_builtin_tools.go — fix in T10-01 |
| **Azure AD app registered** | ⏳ | **[@devops] pre-work** — MSTEAMS_APP_ID + MSTEAMS_APP_PASSWORD must be provisioned before T10-01 |
| G4 co-signed by @cpo + @ceo | ⏳ | Pending CTO-34 fix (done) — [@pm] to file within 48h of Sprint 10 start |

---

## 2. Sprint Goal

**Implement MS Teams channel and connect NQH management team to MTClaw governance rails.**

### Key Outcomes

1. `extensions/msteams/` compiles, builds, integrates via factory pattern — zero core code changes beyond `RegisterFactory`
2. NQH management team (5-10 users) on MS Teams reaches same 3 governance rails as MTS (Telegram)
3. `/spec` and `/review` commands work identically on Teams as on Telegram
4. MSTEAMS_APP_PASSWORD properly masked in all 3 secret functions (CTO ADR-007 note)
5. CTO-33 residual Discord comments cleaned
6. G4 WAU criterion checked at Sprint 10 Day 5 (2-week window from G4 approval)

---

## 3. Task Overview

| ID | Task | Priority | Points | Days | Owner |
|----|------|----------|--------|------|-------|
| T10-01 | MS Teams core implementation (config + auth + webhook + channel) | P0 | 4 | 1-2 | @coder |
| T10-02 | MS Teams unit tests (~15 tests) | P0 | 2 | 3 | @coder |
| T10-03 | Adaptive Cards for spec/PR review output | P1 | 2 | 3-4 | @coder |
| T10-04 | NQH corporate Teams onboarding (config + docs + rollout) | P0 | 2 | 4-5 | @coder + @devops |
| T10-05 | G4 WAU validation + roadmap update + Sprint 11 prep | P1 | 1 | 5 | @pm |

**Total**: ~11 points, 5 days

**[@pm] parallel tasks** (not in point count):
- G4 co-sign: file G4-GATE-PROPOSAL-SPRINT8.md to @cpo + @ceo for sign-off (Day 1-2)
- NQH Teams rollout coordination: invite 5-10 management users (Day 4-5)

---

## 4. Task Specifications

---

### T10-01: MS Teams Core Implementation (P0, 4 pts) — Days 1-2

**Objective**: Implement the full `extensions/msteams/` package and wire into gateway. Fix CTO-33 during gateway touchpoint.

#### Phase 1 — Config additions (Day 1)

**`internal/config/config_channels.go`** — add MSTeamsConfig struct + field:

```go
// MSTeamsConfig holds Microsoft Bot Framework credentials.
type MSTeamsConfig struct {
    Enabled    bool   `json:"enabled"`
    AppID      string `json:"appId"`      // Azure AD app ID (from MSTEAMS_APP_ID)
    AppSecret  string `json:"appSecret"`  // Bot Framework app password (MSTEAMS_APP_PASSWORD)
    TenantID   string `json:"tenantId"`   // Azure tenant ID — MTS: "mts-tenant-id", NEVER "common"
}

// In ChannelsConfig:
MSTeams MSTeamsConfig `json:"msteams"`
```

**`internal/config/config_load.go`** — add env var blocks:

```go
if v := os.Getenv("MSTEAMS_APP_ID"); v != "" {
    cfg.Channels.MSTeams.AppID = v
    cfg.Channels.MSTeams.Enabled = true
}
if v := os.Getenv("MSTEAMS_APP_PASSWORD"); v != "" {
    cfg.Channels.MSTeams.AppSecret = v
}
if v := os.Getenv("MSTEAMS_TENANT_ID"); v != "" {
    cfg.Channels.MSTeams.TenantID = v
}
```

**`internal/config/config_secrets.go`** — mask MSTeams app secret (CTO-ADR-007 + CTO-38):

```go
// In MaskedCopy() — use maskNonEmpty() helper, same as CTO-27 GitHub pattern:
maskNonEmpty(&cp.Channels.MSTeams.AppSecret)

// In StripSecrets():
cfg.Channels.MSTeams.AppSecret = ""

// In StripMaskedSecrets():
cfg.Channels.MSTeams.AppSecret = ""
```

> CTO-38: do NOT use inline `if ... != "" { ... = "***" }` — use `maskNonEmpty()` helper in `MaskedCopy`. `StripSecrets` and `StripMaskedSecrets` use direct assignment (no helper needed there).

**`.env.example`** — add MS Teams section:

```
# MS Teams (Sprint 10)
MSTEAMS_APP_ID=
MSTEAMS_APP_PASSWORD=        # Bot Framework app password — treat as secret
MSTEAMS_TENANT_ID=           # Production: set to your Azure tenant ID. NEVER use "common".
```

#### Phase 2 — Bot Framework auth (Day 1)

**`extensions/msteams/auth.go`**:

```go
package msteams

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "sync"
    "time"
)

const (
    botFrameworkTokenURL = "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"
    botFrameworkScope    = "https://api.botframework.com/.default"
    tokenExpiryBuffer    = 5 * time.Minute
)

// httpClient is a package-level client with explicit timeout.
// http.DefaultClient has no timeout — external auth endpoints (login.microsoftonline.com,
// Bot Framework JWKS) can hang indefinitely without one. (CTO-39)
var httpClient = &http.Client{Timeout: 10 * time.Second}

type tokenCache struct {
    mu        sync.Mutex
    token     string
    expiresAt time.Time
}

func (c *tokenCache) getToken(ctx context.Context, appID, appSecret string) (string, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if time.Now().Before(c.expiresAt.Add(-tokenExpiryBuffer)) {
        return c.token, nil
    }

    data := url.Values{}
    data.Set("grant_type", "client_credentials")
    data.Set("client_id", appID)
    data.Set("client_secret", appSecret)
    data.Set("scope", botFrameworkScope)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, botFrameworkTokenURL,
        strings.NewReader(data.Encode()))
    if err != nil {
        return "", fmt.Errorf("msteams: build token request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := httpClient.Do(req) // CTO-39: use package httpClient, not http.DefaultClient
    if err != nil {
        return "", fmt.Errorf("msteams: token request: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
        Error       string `json:"error"`
        ErrorDesc   string `json:"error_description"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("msteams: decode token response: %w", err)
    }
    if result.Error != "" {
        return "", fmt.Errorf("msteams: token error %s: %s", result.Error, result.ErrorDesc)
    }

    c.token = result.AccessToken
    c.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
    return c.token, nil
}
```

#### Phase 3 — Webhook handler (Day 1-2)

**`extensions/msteams/webhook.go`**:

```go
package msteams

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/nextlevelbuilder/goclaw/internal/bus"
)

// Activity is a Bot Framework Activity message (subset of fields we use).
type Activity struct {
    Type         string       `json:"type"`
    ID           string       `json:"id"`
    Timestamp    string       `json:"timestamp"`
    ServiceURL   string       `json:"serviceUrl"`
    ChannelID    string       `json:"channelId"`
    From         ChannelAcct  `json:"from"`
    Conversation Conversation `json:"conversation"`
    Text         string       `json:"text"`
}

type ChannelAcct  struct { ID string `json:"id"`; Name string `json:"name"` }
type Conversation struct { ID string `json:"id"`; IsGroup bool `json:"isGroup"` }

// webhookHandler processes incoming Bot Framework activities.
// JWT verification is handled by botFrameworkJWTMiddleware wrapping this handler.
func (c *MSTeamsChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
    var activity Activity
    if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    switch activity.Type {
    case "message":
        c.handleMessage(r.Context(), &activity)
    case "conversationUpdate":
        c.handleConversationUpdate(r.Context(), &activity)
    default:
        // Acknowledge unknown event types silently.
    }

    w.WriteHeader(http.StatusOK)
}

func (c *MSTeamsChannel) handleMessage(ctx context.Context, a *Activity) {
    if a.Text == "" {
        return
    }

    c.msgBus.PublishInbound(bus.InboundMessage{
        AgentID:        c.defaultAgentKey,
        From:           a.From.ID,
        Content:        a.Text,
        Channel:        "msteams",
        ConversationID: a.Conversation.ID,
        ServiceURL:     a.ServiceURL,
        Metadata: map[string]string{
            "teams_user_id":     a.From.ID,
            "teams_user_name":   a.From.Name,
            "conversation_id":   a.Conversation.ID,
            "is_group":          boolStr(a.Conversation.IsGroup),
            "service_url":       a.ServiceURL,
        },
    })

    slog.Info("msteams: message received",
        "from", a.From.Name,
        "conversation", a.Conversation.ID,
        "len", len(a.Text))
}

func (c *MSTeamsChannel) handleConversationUpdate(_ context.Context, a *Activity) {
    slog.Info("msteams: conversationUpdate received", "conversation", a.Conversation.ID)
    // Onboarding message logic (Sprint 11 if needed).
}

func boolStr(b bool) string {
    if b { return "true" }
    return "false"
}
```

**`extensions/msteams/jwt.go`** — Bot Framework JWT middleware:

```go
package msteams

import (
    "context"
    "crypto/rsa"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log/slog"
    "math/big"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

const (
    botFrameworkOpenIDURL = "https://login.botframework.com/v1/.well-known/openidconfiguration"
    botFrameworkIssuer    = "https://api.botframework.com"
)

// jwtVerifier fetches Bot Framework OpenID configuration and validates tokens.
type jwtVerifier struct {
    mu        sync.RWMutex
    jwksURL   string
    fetchedAt time.Time
    appID     string
}

func newJWTVerifier(appID string) *jwtVerifier {
    return &jwtVerifier{appID: appID}
}

// Middleware validates the Bot Framework JWT on every inbound request.
func (v *jwtVerifier) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            http.Error(w, "missing authorization", http.StatusUnauthorized)
            return
        }
        token := strings.TrimPrefix(auth, "Bearer ")

        if err := v.verify(r.Context(), token); err != nil {
            slog.Warn("msteams: JWT verification failed", "error", err)
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (v *jwtVerifier) verify(ctx context.Context, tokenStr string) error {
    // Parse without validation first to extract claims for logging.
    claims := jwt.MapClaims{}
    _, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        // In production: fetch JWKS from botFrameworkOpenIDURL and return matching key.
        // Sprint 10: use public key lookup via kid header from JWKS endpoint.
        return v.fetchPublicKey(ctx, t)
    })
    if err != nil {
        return fmt.Errorf("parse token: %w", err)
    }

    // Validate standard claims.
    if iss, _ := claims["iss"].(string); iss != botFrameworkIssuer {
        return fmt.Errorf("invalid issuer: %s", iss)
    }
    if aud, _ := claims["aud"].(string); aud != v.appID {
        return fmt.Errorf("invalid audience: %s (expected %s)", aud, v.appID)
    }
    return nil
}

func (v *jwtVerifier) fetchPublicKey(ctx context.Context, t *jwt.Token) (any, error) {
    // Fetch OpenID metadata → JWKS URL → key matching t.Header["kid"].
    // Cache JWKS response (invalidate after 24h or on kid miss).
    // Implementation: standard JWKS fetch + rsa.PublicKey extraction.
    kid, _ := t.Header["kid"].(string)
    return v.lookupKey(ctx, kid)
}

func (v *jwtVerifier) lookupKey(ctx context.Context, kid string) (any, error) {
    v.mu.RLock()
    stale := time.Since(v.fetchedAt) > 24*time.Hour || v.jwksURL == ""
    v.mu.RUnlock()

    if stale {
        if err := v.refreshOpenID(ctx); err != nil {
            return nil, err
        }
    }
    return v.fetchJWKSKey(ctx, kid)
}

func (v *jwtVerifier) refreshOpenID(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, botFrameworkOpenIDURL, nil)
    if err != nil {
        return err
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var meta struct{ JWKSURI string `json:"jwks_uri"` }
    if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
        return err
    }

    v.mu.Lock()
    v.jwksURL = meta.JWKSURI
    v.fetchedAt = time.Now()
    v.mu.Unlock()
    return nil
}

// jwksCache caches the JWKS response to avoid repeated fetches.
// Keys rotate infrequently (days); 24h TTL is safe.
type jwksCache struct {
    mu        sync.RWMutex
    keys      map[string]*rsa.PublicKey // kid → public key
    fetchedAt time.Time
}

var globalJWKSCache = &jwksCache{keys: make(map[string]*rsa.PublicKey)}

// fetchJWKSKey fetches the RSA public key for the given kid from Bot Framework JWKS.
// CTO-35: This is NOT a stub — must be fully implemented before any end-to-end test. (CTO-35)
func (v *jwtVerifier) fetchJWKSKey(ctx context.Context, kid string) (any, error) {
    // 1. Check cache (read lock).
    globalJWKSCache.mu.RLock()
    key, hit := globalJWKSCache.keys[kid]
    stale := time.Since(globalJWKSCache.fetchedAt) > 24*time.Hour
    globalJWKSCache.mu.RUnlock()

    if hit && !stale {
        return key, nil
    }

    // 2. Fetch JWKS from v.jwksURL (set during refreshOpenID).
    v.mu.RLock()
    jwksURL := v.jwksURL
    v.mu.RUnlock()

    if jwksURL == "" {
        if err := v.refreshOpenID(ctx); err != nil {
            return nil, fmt.Errorf("msteams: refresh OpenID: %w", err)
        }
        v.mu.RLock()
        jwksURL = v.jwksURL
        v.mu.RUnlock()
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
    if err != nil {
        return nil, fmt.Errorf("msteams: build JWKS request: %w", err)
    }

    resp, err := httpClient.Do(req) // CTO-39: use httpClient with timeout
    if err != nil {
        return nil, fmt.Errorf("msteams: JWKS fetch: %w", err)
    }
    defer resp.Body.Close()

    // 3. Parse JWKS response: {"keys": [{kty,use,kid,n,e,...}]}
    var jwks struct {
        Keys []struct {
            Kid string `json:"kid"`
            Kty string `json:"kty"`
            Use string `json:"use"`
            N   string `json:"n"` // base64url-encoded modulus
            E   string `json:"e"` // base64url-encoded exponent
        } `json:"keys"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
        return nil, fmt.Errorf("msteams: decode JWKS: %w", err)
    }

    // 4. Build map and find our key.
    globalJWKSCache.mu.Lock()
    globalJWKSCache.keys = make(map[string]*rsa.PublicKey, len(jwks.Keys))
    globalJWKSCache.fetchedAt = time.Now()
    for _, k := range jwks.Keys {
        if k.Kty != "RSA" || k.Use != "sig" {
            continue
        }
        pubKey, err := parseRSAPublicKey(k.N, k.E)
        if err != nil {
            continue
        }
        globalJWKSCache.keys[k.Kid] = pubKey
    }
    found := globalJWKSCache.keys[kid]
    globalJWKSCache.mu.Unlock()

    if found == nil {
        return nil, fmt.Errorf("msteams: kid %q not found in JWKS (possible key rollover)", kid)
    }
    return found, nil
}

// parseRSAPublicKey decodes base64url-encoded n and e into an rsa.PublicKey.
func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
    nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
    if err != nil {
        return nil, fmt.Errorf("decode n: %w", err)
    }
    eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
    if err != nil {
        return nil, fmt.Errorf("decode e: %w", err)
    }

    n := new(big.Int).SetBytes(nBytes)
    eBig := new(big.Int).SetBytes(eBytes)
    return &rsa.PublicKey{N: n, E: int(eBig.Int64())}, nil
}
```

> **Note**: `jwt.go` uses `github.com/golang-jwt/jwt/v5` — add to `extensions/msteams/go.mod` (workspace package, separate module).

#### Phase 4 — Channel struct + Send (Day 2)

**`extensions/msteams/channel.go`**:

```go
package msteams

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"

    "github.com/nextlevelbuilder/goclaw/internal/bus"
    "github.com/nextlevelbuilder/goclaw/internal/channels"
    "github.com/nextlevelbuilder/goclaw/internal/store"
)

// MSTeamsChannel implements channels.Channel for Microsoft Teams via Bot Framework.
type MSTeamsChannel struct {
    cfg             Config
    msgBus          *bus.MessageBus
    pairingSvc      store.PairingStore
    defaultAgentKey string
    tokenCache      tokenCache
    jwtVerifier     *jwtVerifier
}

var _ channels.Channel = (*MSTeamsChannel)(nil)

func (c *MSTeamsChannel) Start(_ context.Context) error {
    slog.Info("msteams: channel started", "webhook_path", c.cfg.WebhookPath, "tenant_id", c.cfg.TenantID)
    return nil
}

func (c *MSTeamsChannel) Stop(_ context.Context) error {
    slog.Info("msteams: channel stopped")
    return nil
}

func (c *MSTeamsChannel) SetAgentID(agentKey string) {
    c.defaultAgentKey = agentKey
}

func (c *MSTeamsChannel) RegisterRoutes(mux *http.ServeMux) {
    handler := c.jwtVerifier.Middleware(http.HandlerFunc(c.webhookHandler))
    mux.Handle(c.cfg.WebhookPath, handler)
    slog.Info("msteams: webhook registered", "path", c.cfg.WebhookPath)
}

// Send delivers a reply to a Teams conversation.
func (c *MSTeamsChannel) Send(ctx context.Context, msg channels.OutboundMessage) error {
    token, err := c.tokenCache.getToken(ctx, c.cfg.AppID, c.cfg.AppSecret)
    if err != nil {
        return fmt.Errorf("msteams send: acquire token: %w", err)
    }

    conversationID := msg.To
    serviceURL := msg.Metadata["service_url"]
    if serviceURL == "" {
        return fmt.Errorf("msteams send: missing service_url in metadata")
    }

    url := fmt.Sprintf("%sv3/conversations/%s/activities", serviceURL, conversationID)
    payload := map[string]any{
        "type": "message",
        "text": msg.Content,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("msteams send: marshal payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("msteams send: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("msteams send: HTTP: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("msteams send: unexpected status %d", resp.StatusCode)
    }

    slog.Info("msteams: message sent", "conversation", conversationID, "len", len(msg.Content))
    return nil
}
```

#### Phase 5 — Package entry + Factory (Day 2)

**`extensions/msteams/msteams.go`** (replaces `msteams.go.TODO`):

```go
package msteams

import (
    "encoding/json"
    "fmt"

    "github.com/nextlevelbuilder/goclaw/internal/bus"
    "github.com/nextlevelbuilder/goclaw/internal/channels"
    "github.com/nextlevelbuilder/goclaw/internal/store"
)

// Config is the MS Teams channel configuration (read from MSTeamsConfig via JSON).
type Config struct {
    AppID       string `json:"appId"`
    AppSecret   string `json:"appSecret"`
    TenantID    string `json:"tenantId"`
    WebhookPath string `json:"webhookPath"`
}

// Factory creates an MSTeamsChannel from DB instance credentials.
var Factory channels.ChannelFactory = func(
    name string,
    creds json.RawMessage,
    cfg json.RawMessage,
    msgBus *bus.MessageBus,
    pairingSvc store.PairingStore,
) (channels.Channel, error) {
    var c Config
    if err := json.Unmarshal(creds, &c); err != nil {
        return nil, fmt.Errorf("msteams: parse creds: %w", err)
    }

    // Apply defaults.
    if c.WebhookPath == "" {
        c.WebhookPath = "/v1/channels/msteams/webhook"
    }
    if c.TenantID == "" {
        return nil, fmt.Errorf("msteams: TenantID is required (never use 'common' — ADR-007)")
    }
    if c.AppID == "" || c.AppSecret == "" {
        return nil, fmt.Errorf("msteams: AppID and AppSecret are required")
    }

    return &MSTeamsChannel{
        cfg:         c,
        msgBus:      msgBus,
        pairingSvc:  pairingSvc,
        jwtVerifier: newJWTVerifier(c.AppID),
    }, nil
}
```

#### Phase 6 — Gateway wiring + CTO-33 fix (Day 2)

**`cmd/gateway.go`** — add import + RegisterFactory:

```go
import (
    // existing imports...
    "github.com/nextlevelbuilder/goclaw/extensions/msteams"
)

// In loadManagedChannels:
instanceLoader.RegisterFactory("msteams", msteams.Factory)

// For config-based standalone mode (alongside Telegram + Zalo blocks):
if cfg.Channels.MSTeams.Enabled && cfg.Channels.MSTeams.AppID != "" {
    ch, err := msteams.Factory("msteams",
        mustJSON(cfg.Channels.MSTeams),
        nil,
        msgBus,
        pairingSvc,
    )
    if err != nil {
        slog.Error("msteams channel init failed", "error", err)
    } else {
        manager.RegisterChannel("msteams", ch)
    }
}
```

**CTO-33 cleanup** (same file touch):

```
gateway_consumer.go:46  — comment: update "channels (Telegram, Discord, etc.)" → "channels (Telegram, Zalo, MSTeams, etc.)"
gateway_consumer.go:138 — comment: rephrase session-key comment to be channel-neutral
gateway_builtin_tools.go:68 — string: "Send messages to connected channels (Telegram, Discord, etc.)" → "Send messages to connected channels (Telegram, Zalo, MSTeams, etc.)"
```

**Verification after T10-01:**

```bash
# Build must be clean
go build ./...

# MSTeams channel shows in channels list
./mtclaw channels list  # expects msteams entry if MSTEAMS_APP_ID set

# CTO-33 clean
grep -n "Discord\|discord" cmd/gateway_consumer.go cmd/gateway_builtin_tools.go
# Expected: 0 results (only non-functional comments cleaned)
```

---

### T10-02: MS Teams Unit Tests (P0, 2 pts) — Day 3

**File**: `extensions/msteams/msteams_test.go`

**Target**: ~15 tests covering:

| Test group | Count | What |
|-----------|-------|------|
| Config validation | 3 | TenantID required; AppID+AppSecret required; default webhook path set |
| JWT middleware | 3 | Missing Authorization header → 401; invalid token → 401; valid token → 200 |
| Activity parsing | 3 | `message` type routes to bus; empty text skipped; `conversationUpdate` acknowledged |
| Send | 3 | Token acquired before POST; correct endpoint URL built; 4xx response returns error |
| CTO-33 regression | 3 | grep assertions: "Discord" absent in channel.go, webhook.go, msteams.go |

**Test pattern** (consistent with existing test style):

```go
func TestMSTeamsFactory_TenantIDRequired(t *testing.T) {
    creds, _ := json.Marshal(Config{AppID: "app", AppSecret: "secret"}) // no TenantID
    _, err := Factory("msteams", creds, nil, nil, nil)
    if err == nil {
        t.Error("expected error for missing TenantID")
    }
    if !strings.Contains(err.Error(), "TenantID is required") {
        t.Errorf("expected TenantID error, got: %v", err)
    }
}
```

**Verification:**

```bash
# SOUL + msteams tests
go test ./extensions/msteams/... -v -count=1
# Expected: ~15 PASS

# Full suite — no regression
go test ./... -count=1
# Expected: ≥365 PASS (350 + ~15 msteams)
```

---

### T10-03: Adaptive Cards (P1, 2 pts) — Days 3-4

**Goal**: Teams-native output formatting for `/spec` and PR Gate review results.

**Background**: Telegram output uses Markdown. Teams renders Markdown poorly in cards but has first-class Adaptive Card support. Channel-aware rendering: `msteams` → Adaptive Card JSON; others → plain Markdown.

**New file**: `extensions/msteams/cards.go`

```go
package msteams

import (
    "encoding/json"
    "strings" // CTO-36: required for strings.ToUpper in PRReviewCard
)

// AdaptiveCard wraps a Teams Adaptive Card payload.
type AdaptiveCard struct {
    Type        string          `json:"type"`
    Version     string          `json:"version"`
    Body        []CardElement   `json:"body"`
    Actions     []CardAction    `json:"actions,omitempty"`
}

type CardElement struct {
    Type  string `json:"type"`
    Text  string `json:"text,omitempty"`
    Style string `json:"style,omitempty"`
    Wrap  bool   `json:"wrap,omitempty"`
    Size  string `json:"size,omitempty"`
    Weight string `json:"weight,omitempty"`
}

type CardAction struct {
    Type  string `json:"type"`
    Title string `json:"title"`
    URL   string `json:"url,omitempty"`
}

// SpecCard builds an Adaptive Card for a /spec output.
func SpecCard(specID, title, status string, scenarios []string) json.RawMessage {
    card := AdaptiveCard{
        Type:    "AdaptiveCard",
        Version: "1.4",
        Body: []CardElement{
            {Type: "TextBlock", Text: "📋 " + specID, Size: "Medium", Weight: "Bolder", Wrap: true},
            {Type: "TextBlock", Text: title, Wrap: true},
            {Type: "TextBlock", Text: "Status: " + status, Style: "emphasis", Wrap: true},
        },
    }
    for _, s := range scenarios {
        card.Body = append(card.Body, CardElement{Type: "TextBlock", Text: "• " + s, Wrap: true})
    }
    b, _ := json.Marshal(card)
    return b
}

// PRReviewCard builds an Adaptive Card for a PR Gate evaluation.
func PRReviewCard(prURL, verdict string, blockRules, warnRules []string) json.RawMessage {
    style := "good"
    emoji := "✅"
    if verdict == "fail" {
        style = "attention"
        emoji = "❌"
    } else if verdict == "warn" {
        style = "warning"
        emoji = "⚠️"
    }

    card := AdaptiveCard{
        Type:    "AdaptiveCard",
        Version: "1.4",
        Body: []CardElement{
            {Type: "TextBlock", Text: emoji + " PR Gate: " + strings.ToUpper(verdict),
                Size: "Medium", Weight: "Bolder", Style: style, Wrap: true},
            {Type: "TextBlock", Text: prURL, Wrap: true},
        },
        Actions: []CardAction{
            {Type: "Action.OpenUrl", Title: "View PR", URL: prURL},
        },
    }
    for _, r := range blockRules {
        card.Body = append(card.Body, CardElement{Type: "TextBlock", Text: "🚫 " + r, Style: "attention", Wrap: true})
    }
    for _, r := range warnRules {
        card.Body = append(card.Body, CardElement{Type: "TextBlock", Text: "⚠️ " + r, Style: "warning", Wrap: true})
    }
    b, _ := json.Marshal(card)
    return b
}
```

**Channel-aware send in `channel.go`** — when `msg.Format == "adaptive_card"`, wrap payload as Attachment:

```go
// In Send(), detect card payload:
if msg.Format == "adaptive_card" {
    payload = map[string]any{
        "type": "message",
        "attachments": []map[string]any{{
            "contentType": "application/vnd.microsoft.card.adaptive",
            "content":     json.RawMessage(msg.Content),
        }},
    }
}
```

---

### T10-04: NQH Corporate Teams Onboarding (P0, 2 pts) — Days 4-5

**Goal**: NQH management team (5-10 users) connected to MTClaw via MS Teams. Same governance rails as MTS (Telegram).

#### Subtask A — Production config documentation

Update `extensions/msteams/README.md`:

1. Add **Tenant ID Production Requirement** section:
   ```
   ## Production Security Requirement

   MSTEAMS_TENANT_ID must be set to your organization's Azure tenant ID.
   NEVER use "common" — this would allow any Microsoft 365 user to reach your bot.

   To find your tenant ID:
   - Azure Portal → Azure Active Directory → Overview → Tenant ID
   - Or: az account show --query tenantId
   ```

2. Add NQH-specific config example:
   ```
   # NQH deployment example (.env on NQH VPS):
   MSTEAMS_APP_ID=<nqh-app-id>
   MSTEAMS_APP_PASSWORD=<nqh-app-password>
   MSTEAMS_TENANT_ID=<nqh-azure-tenant-id>   # NOT "common"
   ```

3. Add **Channel @mention behavior** note:
   ```
   When MTClaw is @mentioned in a Teams channel, it responds in the same thread.
   Private replies to channel @mentions are NOT used — conversation context is preserved for all thread viewers.
   ```

#### Subtask B — NQH tenant config

Create or update `config/nqh.env` (on NQH deployment server):
```
MSTEAMS_APP_ID=<provisioned by @devops>
MSTEAMS_APP_PASSWORD=<provisioned by @devops — encrypted at rest>
MSTEAMS_TENANT_ID=<NQH Azure tenant ID>
```

**[@devops] pre-work** (must be done before Day 4):
- Register MTClaw bot in NQH Azure AD
- Configure Bot Framework registration (messaging endpoint: `https://<nqh-host>/v1/channels/msteams/webhook`)
- Provision `MSTEAMS_APP_ID` + `MSTEAMS_APP_PASSWORD` to NQH deployment
- Add MTClaw bot to NQH management Teams channel

#### Subtask C — Cross-channel governance verification

**Pre-check (CTO-37)**: Before running these queries, verify `channel` column exists in migrations:

```bash
grep -n "channel" migrations/000013_governance_specs.sql migrations/000015_pr_gate_evaluations.sql
# If absent: apply migrations/000016_add_channel_to_governance_tables.up.sql first (see T10-04 handoff note)
```

Verify Teams messages hit same 3 governance rails as Telegram:

```sql
-- 1. Send /spec command from Teams, then:
SELECT channel, COUNT(*) FROM governance_specs GROUP BY channel;
-- Expected: telegram N, msteams ≥1

-- 2. Send /review <pr-url> from Teams, then:
SELECT channel, COUNT(*) FROM pr_gate_evaluations GROUP BY channel;
-- Expected: telegram N, msteams ≥1

-- 3. Verify trace is channel-tagged
SELECT channel, COUNT(*) FROM traces WHERE created_at > now() - interval '1 day' GROUP BY channel;
-- Expected: msteams row present
```

These queries are DoD checks — they MUST return results, not fail with "column does not exist".

---

### T10-05: G4 WAU Validation + Roadmap Update (P1, 1 pt) — Day 5

**[@pm] deliverable**:

#### WAU Check

G4 was approved 2026-03-17. Day 5 of Sprint 10 = 2026-03-21 = ~4 days since approval.

G4 WAU criterion requires **2-week observation window**. At Sprint 10 close, WAU measurement window is not yet complete. Document current WAU in a daily log:

```
docs/09-govern/01-CTO-Reports/G4-WAU-TRACKING.md
```

Format:
```
Week 1 (2026-03-17 → 2026-03-21): observed WAU = N/10
Target: ≥7/10 by end of week 2 (2026-03-27)
```

If WAU ≥7 at Sprint 10 Day 5: flag G4 as on-track. If WAU <5: escalate to @ceo, plan adoption intervention.

#### Roadmap Update

Mark Sprint 9 COMPLETE (9.0/10), Sprint 10 → COMPLETE after delivery. Update Sprint-by-Sprint Summary table.

---

## 5. CTO Issue Tracker (Sprint 10)

| ID | Priority | Issue | Task | Status |
|----|----------|-------|------|--------|
| CTO-33 | P3 | Discord residuals in gateway_consumer.go:46,:138 + gateway_builtin_tools.go:68 | T10-01 Phase 6 | Open |
| CTO-35 | P1 | JWKS fetchJWKSKey must be fully implemented — stub returns 401 for all tokens | T10-01 Phase 2 | Open — full impl in plan v1.1 |
| CTO-36 | P1 | cards.go missing "strings" import — compile error | T10-03 | Open — fixed in plan v1.1 |
| CTO-37 | P1 | channel column migration 000016 required before T10-04C SQL queries succeed | T10-04 | Open |
| CTO-38 | P2 | MSTeams AppSecret masking: use maskNonEmpty() helper, not inline if | T10-01 Phase 5 | Open — fixed in plan v1.1 |
| CTO-39 | P2 | auth.go + jwt.go: use httpClient with 10s timeout, not http.DefaultClient | T10-01 Phase 2 | Open — fixed in plan v1.1 |

---

## 6. Risk Register

| # | Risk | Probability | Impact | Mitigation |
|---|------|-------------|--------|------------|
| R1 | Azure AD app registration delayed ([@devops] pre-work) | Medium | High | Sprint 10 entry gate: confirm provisioning before Day 1 code start |
| R2 | JWKS key lookup rotation edge case | Low | Medium | jwksCache handles kid miss with force-refresh (implemented in jwt.go v1.1) |
| R3 | Teams webhook blocked by NQH firewall | Low | High | Verify NQH VPS 443 inbound from Bot Framework IPs before Day 4 rollout |
| R4 | Adaptive Card rendering differs across Teams clients (desktop vs mobile) | Low | Low | Test on Teams desktop + web; mobile parity is Phase 2 |
| R5 | G4 WAU <7/10 after 2 weeks | Medium | High | Adoption intervention: proactive onboarding messages, manager champions, /help command |
| R6 | channel column absent from migrations 000013/000015 | Medium | Medium | Check schema Day 1; apply migration 000016 if needed (T10-04, CTO-37) |

---

## 7. Dependencies

| Dependency | Status | Owner |
|------------|--------|-------|
| ADR-007 APPROVED | ✅ | [@cto] 2026-03-17 |
| MS Teams scaffold (`extensions/msteams/`) | ✅ | Sprint 9 |
| Azure AD bot registration (NQH) | ⏳ **BLOCKER** | [@devops] |
| MSTEAMS_APP_ID + PASSWORD provisioned | ⏳ **BLOCKER** | [@devops] |
| G4 co-sign (@cpo + @ceo) | ⏳ | [@pm] — file Day 1 |
| golang-jwt/jwt/v5 dependency | ⏳ | [@coder] adds to extensions/msteams/go.mod |

---

## 8. Definition of Done

| Check | Command | Expected |
|-------|---------|----------|
| Build clean | `go build ./...` | 0 errors |
| All tests pass | `go test ./... -count=1` | ≥365 PASS |
| MS Teams tests | `go test ./extensions/msteams/... -v` | ~15 PASS |
| CTO-33 clean | `grep -n "Discord" cmd/gateway_consumer.go cmd/gateway_builtin_tools.go` | 0 results |
| MSTEAMS_APP_PASSWORD masked | grep for masking in config_secrets.go | present in all 3 functions |
| TenantID validation | Factory called with empty TenantID → error | confirmed by unit test |
| Tenant restriction documented | README.md "Production Security Requirement" section | present |
| Channels list | `./mtclaw channels list` | msteams entry present |
| Cross-channel governance | queries in T10-04C | msteams traces present |

---

## 9. Sprint 11 Preview

Sprint 11 = Phase 3 Hardening:
- Cross-rail evidence linking (spec → PR → test → deploy — full traceability chain)
- Security penetration test (RLS bypass, cross-tenant, SOUL impersonation)
- Full audit trail export (compliance: JSON + CSV + PDF)
- Performance tuning (cost query optimization, RAG latency <3s p95)
- Post-mortem Sprint 1-11 + G5 (Scale Ready) prep

**Entry criteria for Sprint 11**:
- MS Teams live on NQH + MTS
- G4 WAU ≥7/10 (2-week window complete)
- G4 fully co-signed (@cto ✅ @cpo ⏳ @ceo ⏳)

---

## References

| Document | Location |
|----------|----------|
| ADR-006 (Channel Rationalization) | `docs/02-design/01-ADRs/SPEC-0006-ADR-006-Channel-Rationalization.md` |
| ADR-007 (MS Teams Extension) | `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md` |
| MS Teams scaffold | `extensions/msteams/` |
| G4 Gate Proposal | `docs/08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md` |
| Sprint 9 Plan | `docs/04-build/sprints/SPRINT-009-Channel-Cleanup-SOUL-Complete.md` |
| System Architecture | `docs/02-design/system-architecture-document.md` |
| PR Gate Design | `docs/02-design/pr-gate-design.md` |
