package claudecode

import (
	"strings"
	"testing"
)

func TestOutputRedactor_EmptyInput(t *testing.T) {
	r := NewOutputRedactor()
	if got := r.Redact("", false); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestOutputRedactor_NoSecrets(t *testing.T) {
	r := NewOutputRedactor()
	clean := "total 42\ndrwxr-xr-x 2 user user 4096 Mar 1 12:00 src\n-rw-r--r-- 1 user user 1234 main.go"
	got := r.Redact(clean, false)
	if got != clean {
		t.Errorf("clean text should pass through unchanged:\ngot:  %q\nwant: %q", got, clean)
	}
}

func TestOutputRedactor_APIKeys(t *testing.T) {
	r := NewOutputRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"api_key env", "API_KEY=sk-1234567890abcdef"},
		{"api-key env", "api-key=secret123"},
		{"apikey colon", "apikey: mytoken123"},
		{"secret_key", "SECRET_KEY=verysecretvalue"},
		{"access_key", "ACCESS_KEY=AKIAIOSFODNN7EXAMPLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Redact(tt.input, false)
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("should redact %q, got %q", tt.input, got)
			}
		})
	}
}

func TestOutputRedactor_BearerTokens(t *testing.T) {
	r := NewOutputRedactor()
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.signature"
	got := r.Redact(input, false)
	if strings.Contains(got, "eyJ") {
		t.Errorf("bearer token should be redacted: %q", got)
	}
}

func TestOutputRedactor_AWSKeys(t *testing.T) {
	r := NewOutputRedactor()
	input := "Found key: AKIAIOSFODNN7EXAMPLE"
	got := r.Redact(input, false)
	if strings.Contains(got, "AKIAIOSFODNN") {
		t.Errorf("AWS key should be redacted: %q", got)
	}
}

func TestOutputRedactor_GitHubTokens(t *testing.T) {
	r := NewOutputRedactor()
	tests := []struct {
		name  string
		input string
	}{
		{"ghp token", "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"},
		{"ghs token", "secret: ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Redact(tt.input, false)
			if strings.Contains(got, "ghp_") || strings.Contains(got, "ghs_") {
				t.Errorf("GitHub token should be redacted: %q", got)
			}
		})
	}
}

func TestOutputRedactor_PostgresDSN(t *testing.T) {
	r := NewOutputRedactor()
	input := "DSN=postgresql://admin:secretpass@db.example.com:5432/mydb"
	got := r.Redact(input, false)
	if strings.Contains(got, "secretpass") {
		t.Errorf("Postgres password should be redacted: %q", got)
	}
}

func TestOutputRedactor_Password(t *testing.T) {
	r := NewOutputRedactor()
	tests := []string{
		"password=mysecret123",
		"PASSWORD: hunter2",
		"passwd=abc123",
		"pwd=s3cret",
	}
	for _, input := range tests {
		got := r.Redact(input, false)
		if !strings.Contains(got, "[REDACTED]") {
			t.Errorf("password should be redacted in %q, got %q", input, got)
		}
	}
}

func TestOutputRedactor_GoclawEnvVars(t *testing.T) {
	r := NewOutputRedactor()
	tests := []string{
		"MTCLAW_BFLOW_API_KEY=sk-1234",
		"MTCLAW_ENCRYPTION_KEY=abcdef1234567890",
		"MTCLAW_POSTGRES_DSN=postgresql://admin:pass@host/db",
	}
	for _, input := range tests {
		got := r.Redact(input, false)
		if !strings.Contains(got, "[REDACTED]") {
			t.Errorf("GOCLAW env var should be redacted in %q, got %q", input, got)
		}
	}
}

func TestOutputRedactor_JWTTokens(t *testing.T) {
	r := NewOutputRedactor()
	input := "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	got := r.Redact(input, false)
	if strings.Contains(got, "eyJhbGci") {
		t.Errorf("JWT token should be redacted: %q", got)
	}
}

func TestOutputRedactor_PrivateKey(t *testing.T) {
	r := NewOutputRedactor()
	input := "-----BEGIN RSA PRIVATE KEY-----\nMIIE..."
	got := r.Redact(input, false)
	if strings.Contains(got, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("private key header should be redacted: %q", got)
	}
}

func TestOutputRedactor_HeavyRedact_Paths(t *testing.T) {
	r := NewOutputRedactor()

	tests := []struct {
		name  string
		input string
	}{
		{"etc shadow", "reading /etc/shadow for users"},
		{"env file", "loaded .env file"},
		{"aws creds", "~/.aws/credentials found"},
		{"ssh key", "using ~/.ssh/id_rsa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Standard mode: paths pass through
			standard := r.Redact(tt.input, false)
			// Heavy mode: sensitive paths redacted
			heavy := r.Redact(tt.input, true)
			if !strings.Contains(heavy, "[REDACTED]") {
				t.Errorf("heavy redact should scrub %q, got %q", tt.input, heavy)
			}
			_ = standard // standard may or may not redact paths
		})
	}
}

func TestOutputRedactor_HeavyRedact_CleanPaths(t *testing.T) {
	r := NewOutputRedactor()
	input := "reading /home/user/project/main.go"
	got := r.Redact(input, true)
	if strings.Contains(got, "[REDACTED]") {
		t.Errorf("normal path should not be redacted even in heavy mode: %q", got)
	}
}

func TestTruncateOutput(t *testing.T) {
	lines := "line1\nline2\nline3\nline4\nline5"

	tests := []struct {
		name     string
		max      int
		wantEnd  string
		wantLen  int
	}{
		{"all lines", 5, "line5", 5},
		{"more than available", 10, "line5", 5},
		{"truncate to 3", 3, "... [truncated]", 4},
		{"truncate to 1", 1, "... [truncated]", 2},
		{"zero lines", 0, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateOutput(lines, tt.max)
			if tt.max <= 0 {
				if got != "" {
					t.Errorf("zero max should return empty, got %q", got)
				}
				return
			}
			gotLines := strings.Split(got, "\n")
			if len(gotLines) > tt.wantLen {
				t.Errorf("got %d lines, want at most %d", len(gotLines), tt.wantLen)
			}
		})
	}
}

func TestOutputRedactor_PatternCount(t *testing.T) {
	r := NewOutputRedactor()
	if r.PatternCount() < 15 {
		t.Errorf("expected at least 15 redaction patterns, got %d", r.PatternCount())
	}
}
