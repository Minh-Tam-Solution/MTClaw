package telegram

import (
	"context"
	"strconv"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// TelegramResponder implements commands.Responder for the Telegram channel.
// It wraps bot.SendMessage with forum thread routing.
type TelegramResponder struct {
	bot       *telego.Bot
	setThread func(*telego.SendMessageParams)
}

// NewResponder creates a Responder for a specific Telegram message context.
func NewResponder(bot *telego.Bot, setThread func(*telego.SendMessageParams)) *TelegramResponder {
	return &TelegramResponder{bot: bot, setThread: setThread}
}

// Reply sends text to a Telegram chat with proper forum thread routing.
func (r *TelegramResponder) Reply(ctx context.Context, chatID string, text string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return err
	}
	msg := tu.Message(tu.ID(id), text)
	if r.setThread != nil {
		r.setThread(msg)
	}
	_, err = r.bot.SendMessage(ctx, msg)
	return err
}
