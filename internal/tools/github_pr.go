package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// GitHubClient posts PR comments and sets commit status checks.
// Constructed once at gateway startup (CTO-23: avoid per-message allocation).
type GitHubClient struct {
	token   string
	baseURL string
	client  *http.Client
}

// NewGitHubClient creates a GitHub API client with the given PAT or installation token.
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: "https://api.github.com",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// PostComment posts a comment on a PR (via the issues API).
// GitHub API: POST /repos/{owner}/{repo}/issues/{number}/comments
func (c *GitHubClient) PostComment(ctx context.Context, repo string, prNumber int, body string) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, repo, prNumber)
	payload, _ := json.Marshal(map[string]string{"body": body})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("github: create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("github: post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Warn("github.post_comment_failed",
			"repo", repo, "pr", prNumber, "status", resp.StatusCode, "body", string(respBody))
		return fmt.Errorf("github: post comment: status %d", resp.StatusCode)
	}
	return nil
}

// SetCommitStatus sets a commit status check (success/failure/pending).
// GitHub API: POST /repos/{owner}/{repo}/statuses/{sha}
func (c *GitHubClient) SetCommitStatus(ctx context.Context, repo, sha, state, description string) error {
	url := fmt.Sprintf("%s/repos/%s/statuses/%s", c.baseURL, repo, sha)
	payload, _ := json.Marshal(map[string]string{
		"state":       state,
		"description": description,
		"context":     "mtclaw/pr-gate",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("github: create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("github: set commit status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Warn("github.set_status_failed",
			"repo", repo, "sha", sha, "state", state, "status", resp.StatusCode, "body", string(respBody))
		return fmt.Errorf("github: set commit status: status %d", resp.StatusCode)
	}
	return nil
}

func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}
