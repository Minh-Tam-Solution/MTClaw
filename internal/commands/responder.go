package commands

import "context"

// Responder abstracts sending text back to the user.
// Each channel implements this — Telegram wraps telego, Discord wraps discordgo.
// CTO C1: Do NOT pass raw *telego.SendMessageParams or *discordgo.Session into shared code.
type Responder interface {
	Reply(ctx context.Context, chatID string, text string) error
}
