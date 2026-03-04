package souls

import (
	"testing"
)

func TestChecksumContent(t *testing.T) {
	hash1 := ChecksumContent("hello world")
	hash2 := ChecksumContent("hello world")
	if hash1 != hash2 {
		t.Errorf("same content should produce same hash")
	}

	hash3 := ChecksumContent("different content")
	if hash1 == hash3 {
		t.Errorf("different content should produce different hash")
	}
}

func TestChecksumContent_NormalizesWhitespace(t *testing.T) {
	base := "# SOUL\nYou are a coder."
	withNewline := base + "\n"
	withCRLF := base + "\r\n"
	withTrailingSpaces := base + "  \n\t"

	baseHash := ChecksumContent(base)
	if ChecksumContent(withNewline) != baseHash {
		t.Errorf("trailing newline should not affect checksum")
	}
	if ChecksumContent(withCRLF) != baseHash {
		t.Errorf("trailing CRLF should not affect checksum")
	}
	if ChecksumContent(withTrailingSpaces) != baseHash {
		t.Errorf("trailing whitespace should not affect checksum")
	}
}

func TestCheckDrift_InSync(t *testing.T) {
	status := CheckDrift("content A", "content A")
	if !status.InSync {
		t.Errorf("expected InSync=true for identical content")
	}
	if status.DriftType != "" {
		t.Errorf("expected empty DriftType, got %q", status.DriftType)
	}
}

func TestCheckDrift_ContentChanged(t *testing.T) {
	status := CheckDrift("version 1", "version 2")
	if status.InSync {
		t.Errorf("expected InSync=false for different content")
	}
	if status.DriftType != DriftContentChanged {
		t.Errorf("expected DriftType=%q, got %q", DriftContentChanged, status.DriftType)
	}
	if status.OldChecksum == status.NewChecksum {
		t.Errorf("checksums should differ for different content")
	}
}

func TestCheckDrift_WhitespaceOnly(t *testing.T) {
	status := CheckDrift("content\n", "content\r\n\t")
	if !status.InSync {
		t.Errorf("trailing whitespace difference should be InSync")
	}
}
