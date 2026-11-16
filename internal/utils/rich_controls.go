package utils

import (
	"html"
	"strings"
	"unicode"

	td "github.com/AshokShau/gotdbot"
)

// Assigned once while loading modules; the provider reads persisted chat settings.
var ButtonThemeForChat = func(int64) string { return "auto" }

// Keep embedded rich buttons outside the quoted text block.
func QuoteRichHTML(text string) string {
	if index := strings.Index(text, "<tg-button-row"); index >= 0 {
		return QuotedHTML(text[:index]) + text[index:]
	}
	return QuotedHTML(text)
}

func richRTL(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

// RichControlHTML keeps controls inside the rich message instead of attaching
// a separate inline keyboard. Callback payloads and player links stay unchanged.
func RichControlHTML(text string, keyboard *td.ReplyMarkupInlineKeyboard) string {
	return richControlHTML(text, keyboard, "auto")
}

func RichControlHTMLForChat(chatID int64, text string, keyboard *td.ReplyMarkupInlineKeyboard) string {
	return richControlHTML(text, keyboard, ButtonThemeForChat(chatID))
}
func richControlHTML(text string, keyboard *td.ReplyMarkupInlineKeyboard, theme string) string {
	var out strings.Builder
	out.WriteString(QuoteRichHTML(strings.ReplaceAll(text, "\n", "<br>")))
	if keyboard == nil {
		return out.String()
	}
	for _, row := range keyboard.Rows {
		out.WriteString(`<tg-button-row>`)
		for _, button := range row {
			style := "primary"
			switch button.Style.(type) {
			case td.ButtonStyleDanger, *td.ButtonStyleDanger:
				style = "danger"
			case td.ButtonStyleSuccess, *td.ButtonStyleSuccess:
				style = "success"
			case td.ButtonStyleLink, *td.ButtonStyleLink:
				style = "link"
			}
			if style == "primary" {
				switch theme {
				case "green":
					style = "success"
				case "red":
					style = "danger"
				}
			}
			styleAttr := ` style="` + style + `"`
			switch kind := button.Type.(type) {
			case *td.InlineKeyboardButtonTypeCallback:
				out.WriteString(`<tg-button type="callback_data"` + styleAttr + ` data="` + html.EscapeString(string(kind.Data)) + `">`)
			case *td.InlineKeyboardButtonTypeUrl:
				out.WriteString(`<tg-button type="url"` + styleAttr + ` url="` + html.EscapeString(kind.Url) + `">`)
			default:
				continue
			}
			out.WriteString(html.EscapeString(button.Text))
			out.WriteString(`</tg-button>`)
		}
		out.WriteString(`</tg-button-row>`)
	}
	return out.String()
}

func EditControlMessage(c *td.Client, chatID, messageID int64, text string, opts *td.EditMessageTextOpts) (*td.Message, error) {
	var keyboard *td.ReplyMarkupInlineKeyboard
	if opts != nil {
		keyboard, _ = opts.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard)
	}
	// A photo welcome card cannot be edited as rich text. Open its controls
	// in a rich reply while keeping the original photo and caption together.
	if chatID > 0 {
		message, err := c.GetMessage(chatID, messageID)
		if err != nil {
			return nil, err
		}
		if _, ok := message.Content.(*td.MessagePhoto); ok {
			return ReplyControlMessage(c, message, text, &td.SendMessageOpts{ReplyMarkup: keyboard})
		}
	}
	return c.EditMessageTextContent(chatID, &td.InputMessageRichMessage{Message: &td.InputRichMessage{
		Source: &td.RichMessageSourceHtml{Text: richControlHTML(text, keyboard, ButtonThemeForChat(chatID))},
		IsRtl:  richRTL(text),
	}}, messageID, nil)
}

func ReplyControlMessage(c *td.Client, message *td.Message, text string, opts *td.SendMessageOpts) (*td.Message, error) {
	copyOpts := td.SendMessageOpts{}
	if opts != nil {
		copyOpts = *opts
	}
	keyboard, _ := copyOpts.ReplyMarkup.(*td.ReplyMarkupInlineKeyboard)
	copyOpts.ReplyMarkup = nil
	return message.ReplyRichMessageHTML(c, richControlHTML(text, keyboard, ButtonThemeForChat(message.ChatId)), &td.SendRichTextOpts{SendMessageOpts: copyOpts, IsRtl: richRTL(text)})
}
