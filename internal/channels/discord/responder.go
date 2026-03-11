package discord

import "context"

// DiscordResponder implements commands.Responder for the Discord channel.
type DiscordResponder struct {
	channel *Channel
}

// Reply sends text back to a Discord channel. Handles chunking for >2000 char messages.
func (r *DiscordResponder) Reply(_ context.Context, chatID string, text string) error {
	return r.channel.sendChunkedText(chatID, text)
}
