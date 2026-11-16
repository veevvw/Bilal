package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/commands"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/i18n"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"context"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func chatLanguage(chatID int64) string {
	if db.Instance != nil {
		if language, err := db.Instance.GetLanguage(chatID); err == nil {
			return i18n.Normalize(language)
		}
	}
	return i18n.Default
}

func tr(chatID int64, source string) string {
	language := chatLanguage(chatID)
	return commands.Display(i18n.Text(language, source), language)
}
func firstChatID(ids []int64) int64 {
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// Copy buttons before translating; the shared English definitions stay immutable.
func localizedKeyboard(chatID int64, keyboard *td.ReplyMarkupInlineKeyboard) *td.ReplyMarkupInlineKeyboard {
	if keyboard == nil {
		return nil
	}
	copy := *keyboard
	copy.Rows = make([][]td.InlineKeyboardButton, len(keyboard.Rows))
	for i, row := range keyboard.Rows {
		copy.Rows[i] = append([]td.InlineKeyboardButton(nil), row...)
		for j := range copy.Rows[i] {
			copy.Rows[i][j].Text = tr(chatID, row[j].Text)
		}
	}
	return &copy
}

// Language follows a conversation, including private chats. Group changes use
// the same administrator checks as the rest of the settings menu.
func languageHandler(c *td.Client, m *td.Message) error {
	if !m.IsPrivate() {
		if err := cache.SettingsAccess(c, m.ChatId, m.SenderID(), "lang"); err != nil {
			_, replyErr := utils.ReplyBotText(c, m, tr(m.ChatId, "you must be a group administrator to change settings"), nil)
			return replyErr
		}
	}
	parts := strings.Fields(m.GetText())
	if len(parts) == 2 && (parts[1] == "ar" || parts[1] == "en") {
		if err := db.Instance.SetLanguage(context.Background(), m.ChatId, parts[1]); err != nil {
			_, err = utils.ReplyBotText(c, m, tr(m.ChatId, "Unable to save settings. Please try again."), nil)
			return err
		}
		webapp.HubInstance.BroadcastChatSettings(c, m.ChatId)
		_, err := utils.ReplyBotText(c, m, tr(m.ChatId, "Language saved. Use /language ar or /language en to change it."), nil)
		return err
	}
	_, err := utils.ReplyBotText(c, m, tr(m.ChatId, "Choose a language: /language ar — العربية | /language en — English"), nil)
	return err
}
