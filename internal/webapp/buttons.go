package webapp

import (
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/i18n"
	"ashokshau/tg-web/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

const playerContextCallbackPrefix = "webapp_verify:"

func WebAppControlButtons(mode, botUsername string, chatID int64) *td.ReplyMarkupInlineKeyboard {
	instance := ""
	if chatID < 0 {
		instance, _ = db.Instance.GetTelegramChatInstance(chatID)
	}
	return webAppControlButtonsForContext(mode, botUsername, chatID, instance)
}

func webAppControlButtonsForContext(mode, botUsername string, chatID int64, instance string) *td.ReplyMarkupInlineKeyboard {
	markup := utils.WebAppControlButtons(mode, botUsername, chatID)
	language := i18n.Default
	if db.Instance != nil {
		language, _ = db.Instance.GetLanguage(chatID)
	}
	for i := range markup.Rows {
		for j := range markup.Rows[i] {
			markup.Rows[i][j].Text = i18n.Text(language, markup.Rows[i][j].Text)
		}
	}
	if chatID >= 0 || instance != "" {
		return markup
	}

	markup.Rows[len(markup.Rows)-1][0].Type = &td.InlineKeyboardButtonTypeCallback{
		Data: []byte(playerContextCallbackPrefix + mode),
	}
	markup.Rows[len(markup.Rows)-1][0].Text = i18n.Text(language, "🔐 Verify group for player")
	return markup
}

func PlayerContextCallbackMode(data string) (string, bool) {
	if data == "webapp_verify" {
		return "webapp", true
	}

	if len(data) < len(playerContextCallbackPrefix) || data[:len(playerContextCallbackPrefix)] != playerContextCallbackPrefix {
		return "", false
	}

	mode := data[len(playerContextCallbackPrefix):]
	switch mode {
	case "", "play", "pause", "resume", "webapp":
		return mode, true
	default:
		log.Warnf("playerContextCallbackMode: invalid mode: %s", mode)
		return "", false
	}
}
