// Package rag provides an HTTP client for the Bflow AI-Platform RAG API.
// Sprint 6: Context Drift Layer B — SOUL-Aware RAG Routing (US-034).
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Result represents a single RAG search result.
type Result struct {
	Content  string   `json:"content"`
	Metadata Metadata `json:"metadata"`
	Score    float64  `json:"score"`
}

// Metadata contains source information for a RAG result.
type Metadata struct {
	Source string `json:"source"`
}

// Response represents the RAG API response.
type Response struct {
	Results    []Result `json:"results"`
	TotalHits  int      `json:"total_hits"`
	TokensUsed int      `json:"tokens_used"`
}

// Client queries the Bflow AI-Platform RAG endpoint.
type Client struct {
	baseURL  string
	apiKey   string
	tenantID string
	http     *http.Client
}

// NewClient creates a RAG client using Bflow AI-Platform credentials.
// If baseURL is empty, defaults to MTCLAW_BFLOW_BASE_URL env or http://ai-platform:8120.
// If apiKey is empty, reads from MTCLAW_BFLOW_API_KEY env.
// If tenantID is empty, reads from BFLOW_TENANT_ID env.
func NewClient(baseURL, apiKey, tenantID string) *Client {
	if baseURL == "" {
		baseURL = os.Getenv("MTCLAW_BFLOW_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "http://ai-platform:8120"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if apiKey == "" {
		apiKey = os.Getenv("MTCLAW_BFLOW_API_KEY")
	}
	if tenantID == "" {
		tenantID = os.Getenv("BFLOW_TENANT_ID")
	}

	return &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		tenantID: tenantID,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// queryRequest is the JSON body sent to the RAG endpoint.
type queryRequest struct {
	Query      string `json:"query"`
	Collection string `json:"collection"`
	TopK       int    `json:"top_k"`
	MaxTokens  int    `json:"max_tokens"`
}

// Query searches the RAG collection for relevant documents.
// Returns nil results (not error) if collection is empty or query is blank.
// Graceful degradation: returns (nil, err) on failure — caller should proceed without RAG.
func (c *Client) Query(ctx context.Context, query string, collection string, maxTokens int) (*Response, error) {
	if query == "" || collection == "" {
		return nil, nil
	}

	body, err := json.Marshal(queryRequest{
		Query:      query,
		Collection: collection,
		TopK:       5,
		MaxTokens:  maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("rag: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api/v1/rag/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rag: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rag: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("rag: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("rag: decode response: %w", err)
	}

	return &result, nil
}

// QueryMultiple queries multiple collections and merges results by score.
// Applies maxTokens budget across all collections.
func (c *Client) QueryMultiple(ctx context.Context, query string, collections []string, maxTokens int) (*Response, error) {
	if len(collections) == 0 || query == "" {
		return nil, nil
	}

	var allResults []Result
	totalHits := 0
	totalTokens := 0

	for _, coll := range collections {
		resp, err := c.Query(ctx, query, coll, maxTokens)
		if err != nil {
			// Graceful: skip failed collections, continue with others
			continue
		}
		if resp != nil {
			allResults = append(allResults, resp.Results...)
			totalHits += resp.TotalHits
			totalTokens += resp.TokensUsed
		}
	}

	if len(allResults) == 0 {
		return nil, nil
	}

	// Sort by score descending, truncate to budget
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Truncate to fit within maxTokens (rough estimate: 1 token ≈ 4 chars)
	var kept []Result
	charBudget := maxTokens * 4
	used := 0
	for _, r := range allResults {
		if used+len(r.Content) > charBudget {
			break
		}
		kept = append(kept, r)
		used += len(r.Content)
	}

	return &Response{
		Results:    kept,
		TotalHits:  totalHits,
		TokensUsed: totalTokens,
	}, nil
}

// CollectionMap maps SOUL agent_key to RAG collection(s).
// Hardcoded for Sprint 6 (MTS collections). Sprint 9+: configurable per tenant.
var CollectionMap = map[string][]string{
	"dev":       {"engineering"},
	"coder":     {"engineering"},
	"architect": {"engineering"},
	"reviewer":  {"engineering"},
	"devops":    {"engineering"},
	"tester":    {"engineering"},
	"itadmin":   {"engineering"},
	"sales":     {"sales"},
	"cs":        {"engineering", "sales"},
	"assistant": {"engineering", "sales"},
	"pm":        {"engineering", "sales"},
	"writer":    {"engineering"},
}
