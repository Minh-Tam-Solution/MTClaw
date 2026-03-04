// Package souls provides SOUL lifecycle utilities including drift detection (ADR-004).
package souls

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// DriftType constants.
const (
	DriftContentChanged  = "content_changed"
	DriftVersionMismatch = "version_mismatch"
	DriftMissing         = "missing"
)

// DriftStatus represents the result of a drift check.
type DriftStatus struct {
	AgentKey    string
	InSync      bool
	OldChecksum string
	NewChecksum string
	DriftType   string
}

// ChecksumContent computes SHA256 of SOUL content for drift detection.
// Normalizes whitespace (trailing newlines, carriage returns) before hashing
// to avoid false positives from editor differences.
func ChecksumContent(content string) string {
	normalized := strings.TrimRight(content, "\r\n \t")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
}

// CheckDrift compares stored content vs current content by SHA256 checksum.
func CheckDrift(storedContent, currentContent string) DriftStatus {
	storedHash := ChecksumContent(storedContent)
	currentHash := ChecksumContent(currentContent)
	if storedHash == currentHash {
		return DriftStatus{InSync: true, OldChecksum: storedHash, NewChecksum: currentHash}
	}
	return DriftStatus{
		InSync:      false,
		OldChecksum: storedHash,
		NewChecksum: currentHash,
		DriftType:   DriftContentChanged,
	}
}
