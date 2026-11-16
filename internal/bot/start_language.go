package bot

import (
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"context"
	td "github.com/AshokShau/gotdbot"
	"strings"
	"time"
)

func startLanguageCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if cb.ChatId <= 0 || cb.ChatId != cb.SenderUserId {
		return cb.Answer(c, "", nil)
	}
	action := strings.TrimPrefix(cb.DataString(), "language:")
	if action == "menu" {
		keyboard := groupKeyboard(menuButton("🌐 العربية / English", "language:menu", td.ButtonStylePrimary{}), menuButton("العربية", "language:ar", td.ButtonStyleSuccess{}), menuButton("English", "language:en", td.ButtonStylePrimary{}))
		_ = cb.Answer(c, "", nil)
		_, err := utils.EditControlMessage(c, cb.ChatId, cb.MessageId, "<b>🌐 العربية / English</b>", &td.EditMessageTextOpts{ReplyMarkup: keyboard})
		return err
	}
	if action != "ar" && action != "en" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Instance.SetLanguage(ctx, cb.ChatId, action); err != nil {
		return cb.Answer(c, protectionText(cb.ChatId, "تعذر حفظ اللغة.", "Unable to save language."), &td.CallbackAnswerOpts{ShowAlert: true})
	}
	_ = cb.Answer(c, protectionText(cb.ChatId, "تم تغيير اللغة.", "Language changed."), nil)
	_, err := utils.EditControlMessage(c, cb.ChatId, cb.MessageId, "<b>ShaHm</b>", &td.EditMessageTextOpts{ReplyMarkup: localizedKeyboard(cb.ChatId, utils.AddMeMarkup(c.Me().Usernames.EditableUsername))})
	return err
}
