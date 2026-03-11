package commands

// CommandMetadata carries command-specific metadata for bus publishing.
// CTO C4: typed struct instead of flat map, platform-specific fields are optional.
type CommandMetadata struct {
	Command         string // "reset", "stop", "stopall", "spec", "review"
	Platform        string // "telegram", "discord", "zalo", "msteams"
	Rail            string // CTO F1: skill routing — "spec-factory", "pr-gate" (optional)
	PRURL           string // /review only: GitHub PR URL (optional)
	LocalKey        string // Telegram forum: "-1001234567890:topic:42" (optional)
	IsForum         string // Telegram: "true"/"false" (optional)
	MessageThreadID string // Telegram forum topic ID (optional)
}

// ToMap converts to bus metadata. PJM-033-1: skips empty fields to avoid
// Telegram-specific metadata leaking into Discord/Zalo bus messages.
func (m CommandMetadata) ToMap() map[string]string {
	result := map[string]string{"command": m.Command, "platform": m.Platform}
	if m.Rail != "" {
		result["rail"] = m.Rail
	}
	if m.PRURL != "" {
		result["pr_url"] = m.PRURL
	}
	if m.LocalKey != "" {
		result["local_key"] = m.LocalKey
	}
	if m.IsForum != "" {
		result["is_forum"] = m.IsForum
	}
	if m.MessageThreadID != "" {
		result["message_thread_id"] = m.MessageThreadID
	}
	return result
}
