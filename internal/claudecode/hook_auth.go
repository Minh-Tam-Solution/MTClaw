package claudecode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

// HookTimestampWindow is the maximum age of a signed hook request (D5).
const HookTimestampWindow = 30 * time.Second

// SignHook creates an HMAC-SHA256 signature for a hook payload.
// Format: HMAC-SHA256(secret, timestamp + "." + body)
func SignHook(secret, body string, timestamp int64) string {
	msg := strconv.FormatInt(timestamp, 10) + "." + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHook checks an HMAC-SHA256 signature with replay protection.
// Returns nil if valid, error describing the failure otherwise.
func VerifyHook(secret, body, signature string, timestamp int64) error {
	// Check timestamp freshness (replay protection)
	now := time.Now().Unix()
	age := math.Abs(float64(now - timestamp))
	if age > HookTimestampWindow.Seconds() {
		return fmt.Errorf("hook timestamp expired: age=%.0fs, window=%s (reason_code=hook_replay_rejected)", age, HookTimestampWindow)
	}

	// Compute expected signature
	expected := SignHook(secret, body, timestamp)

	// Constant-time comparison
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("hook HMAC mismatch (reason_code=hook_signature_invalid)")
	}

	return nil
}
