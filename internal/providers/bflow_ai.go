package providers

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// bflowTransport is an http.RoundTripper that replaces the standard
// Authorization: Bearer header with Bflow AI-Platform's
// X-API-Key + X-Tenant-ID headers.
type bflowTransport struct {
	base     http.RoundTripper
	apiKey   string
	tenantID string
}

func (t *bflowTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone per http.RoundTripper contract (callers may reuse the request).
	clone := req.Clone(req.Context())

	// Replace standard Bearer auth with Bflow AI-Platform headers.
	clone.Header.Del("Authorization")
	clone.Header.Set("X-API-Key", t.apiKey)
	if t.tenantID != "" {
		clone.Header.Set("X-Tenant-ID", t.tenantID)
	}

	return t.base.RoundTrip(clone)
}

// NewBflowAIProvider creates an OpenAI-compatible provider configured for
// the Bflow AI-Platform with X-API-Key + X-Tenant-ID authentication.
//
// If tenantID is empty, it falls back to the BFLOW_TENANT_ID env var.
// apiBase defaults to http://ai-platform:8120/api/v1 if empty.
// defaultModel defaults to qwen3:14b if empty.
func NewBflowAIProvider(apiKey, apiBase, tenantID, defaultModel string) *OpenAIProvider {
	if apiBase == "" {
		apiBase = "http://ai-platform:8120/api/v1"
	}
	apiBase = strings.TrimRight(apiBase, "/")

	if tenantID == "" {
		tenantID = os.Getenv("BFLOW_TENANT_ID")
	}
	if defaultModel == "" {
		defaultModel = "qwen3:14b"
	}

	prov := NewOpenAIProvider("bflow-ai-platform", apiKey, apiBase, defaultModel)

	// Replace HTTP client with Bflow-specific auth transport.
	prov.client = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &bflowTransport{
			base:     http.DefaultTransport,
			apiKey:   apiKey,
			tenantID: tenantID,
		},
	}

	return prov
}
