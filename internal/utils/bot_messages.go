package utils

import (
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func QuotedHTML(text string) string {
	if strings.TrimSpace(text) == "" {
		text = "…"
	}
	return "<blockquote>" + text + "</blockquote>"
}

func botTextHTML(c *td.Client, text, mode string, entities []td.TextEntity) (string, error) {
	formatted, err := c.GetFormattedText(text, entities, mode)
	if err != nil {
		return "", err
	}
	return formatted.Format("HTML"), nil
}

func ReplyBotText(c *td.Client, m *td.Message, text string, opts *td.SendMessageOpts) (*td.Message, error) {
	copy := td.SendMessageOpts{}
	if opts != nil {
		copy = *opts
	}
	rendered, err := botTextHTML(c, text, copy.ParseMode, copy.Entities)
	if err != nil {
		return nil, err
	}
	copy.ParseMode = "HTML"
	copy.Entities = nil
	if keyboard, ok := copy.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard); ok {
		copy.ReplyMarkup = keyboard
		return ReplyControlMessage(c, m, rendered, &copy)
	}
	return m.ReplyText(c, QuotedHTML(rendered), &copy)
}
func SendBotText(c *td.Client, chatID int64, text string, opts *td.SendMessageOpts) (*td.Message, error) {
	copy := td.SendMessageOpts{}
	if opts != nil {
		copy = *opts
	}
	rendered, err := botTextHTML(c, text, copy.ParseMode, copy.Entities)
	if err != nil {
		return nil, err
	}
	copy.ParseMode = "HTML"
	copy.Entities = nil
	if keyboard, ok := copy.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard); ok {
		copy.ReplyMarkup = nil
		return c.SendRichMessageHTML(chatID, richControlHTML(rendered, keyboard, ButtonThemeForChat(chatID)), &td.SendRichTextOpts{SendMessageOpts: copy, IsRtl: richRTL(rendered)})
	}
	return c.SendMessage(chatID, QuotedHTML(rendered), &copy)
}
func EditBotText(c *td.Client, m *td.Message, text string, opts *td.EditMessageTextOpts) (*td.Message, error) {
	copy := td.EditMessageTextOpts{}
	if opts != nil {
		copy = *opts
	}
	rendered, err := botTextHTML(c, text, copy.ParseMode, copy.Entities)
	if err != nil {
		return nil, err
	}
	copy.ParseMode = "HTML"
	copy.Entities = nil
	if _, ok := copy.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard); ok {
		return EditControlMessage(c, m.ChatId, m.Id, rendered, &copy)
	}
	return m.EditText(c, QuotedHTML(rendered), &copy)
}
func EditCallbackBotText(c *td.Client, cb *td.UpdateNewCallbackQuery, text string, opts *td.EditMessageTextOpts) (*td.Message, error) {
	copy := td.EditMessageTextOpts{}
	if opts != nil {
		copy = *opts
	}
	rendered, err := botTextHTML(c, text, copy.ParseMode, copy.Entities)
	if err != nil {
		return nil, err
	}
	copy.ParseMode = "HTML"
	copy.Entities = nil
	if _, ok := copy.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard); ok {
		return EditControlMessage(c, cb.ChatId, cb.MessageId, rendered, &copy)
	}
	return cb.EditMessageText(c, QuotedHTML(rendered), &copy)
}
