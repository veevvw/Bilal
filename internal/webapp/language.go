package webapp

import (
	"ashokshau/tg-web/internal/commands"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/i18n"
)

func roomText(chatID int64, source string) string {
	language := i18n.Default
	if db.Instance != nil {
		language, _ = db.Instance.GetLanguage(chatID)
	}
	return commands.Display(i18n.Text(language, source), language)
}
