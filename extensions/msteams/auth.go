package msteams

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// tokenCache holds a cached Bot Framework access token with expiry tracking.
type tokenCache struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

// tokenProvider acquires and caches Bot Framework access tokens.
// Uses client_credentials flow against login.microsoftonline.com.
// CTO-39: uses httpClient with 10s timeout — never http.DefaultClient.
type tokenProvider struct {
	appID         string
	appPassword   string
	tokenEndpoint string // overridable for tests; defaults to Bot Framework endpoint
	cache         tokenCache
	httpClient    *http.Client
}

func newTokenProvider(appID, appPassword string) *tokenProvider {
	return &tokenProvider{
		appID:         appID,
		appPassword:   appPassword,
		tokenEndpoint: "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token",
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Token returns a valid Bot Framework access token, refreshing the cache as needed.
// The 5-minute expiry buffer prevents using a token right before it expires.
func (p *tokenProvider) Token() (string, error) {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	// Return cached token if still valid (5-minute expiry buffer)
	if p.cache.token != "" && time.Now().Before(p.cache.expiry) {
		return p.cache.token, nil
	}

	token, expiresIn, err := p.acquire()
	if err != nil {
		return "", err
	}

	// Store with 5-minute buffer before actual expiry
	buffer := 5 * time.Minute
	expiry := time.Now().Add(time.Duration(expiresIn)*time.Second - buffer)
	p.cache.token = token
	p.cache.expiry = expiry
	return token, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// acquire fetches a new token from the Bot Framework token endpoint.
func (p *tokenProvider) acquire() (token string, expiresIn int, err error) {
	endpoint := p.tokenEndpoint

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.appID)
	data.Set("client_secret", p.appPassword)
	data.Set("scope", "https://api.botframework.com/.default")

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("msteams: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("msteams: token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", 0, fmt.Errorf("msteams: read token response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", 0, fmt.Errorf("msteams: parse token response: %w", err)
	}

	if tr.Error != "" {
		return "", 0, fmt.Errorf("msteams: token error %s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", 0, fmt.Errorf("msteams: empty access_token in response (status %d)", resp.StatusCode)
	}

	return tr.AccessToken, tr.ExpiresIn, nil
}
