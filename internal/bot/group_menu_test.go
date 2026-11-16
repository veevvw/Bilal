package bot

import (
	"ashokshau/tg-web/internal/moderation"
	td "github.com/AshokShau/gotdbot"
	"strings"
	"testing"
)

func checkGroupKeyboard(t *testing.T, k *td.ReplyMarkupInlineKeyboard) {
	t.Helper()
	if len(k.Rows) == 0 || len(k.Rows[0]) != 1 {
		t.Fatal("heading must be a single button")
	}
	for i, row := range k.Rows {
		want := 1
		if (i+1)%2 == 0 {
			want = 2
		}
		if len(row) != want {
			t.Fatalf("row %d has %d buttons", i, len(row))
		}
		for _, button := range row {
			if callback, ok := button.Type.(*td.InlineKeyboardButtonTypeCallback); ok {
				if len(callback.Data) > 64 {
					t.Fatal("callback exceeds Telegram limit")
				}
			}
		}
	}
}

func TestAlternatingRowsForAnyButtonCount(t *testing.T) {
	for n := 0; n < 25; n++ {
		buttons := make([]td.InlineKeyboardButton, n)
		for i := range buttons {
			buttons[i] = menuButton("button", "game:menu", td.ButtonStylePrimary{})
		}
		checkGroupKeyboard(t, groupKeyboard(menuButton("title", "hub:home", td.ButtonStylePrimary{}), buttons...))
	}
}
func TestUnifiedMenuAndProtectionLayout(t *testing.T) {
	_, k := groupHome(-1001234567890)
	checkGroupKeyboard(t, k)
	for _, page := range []string{"home", "content", "media", "spam", "penalty", "words"} {
		_, k = protectionPage(-1001234567890, moderation.Default(), page)
		checkGroupKeyboard(t, k)
	}
	_, k = gamesPanel(-1001234567890, true)
	checkGroupKeyboard(t, k)
	data := ""
	for _, row := range k.Rows {
		for _, b := range row {
			if callback, ok := b.Type.(*td.InlineKeyboardButtonTypeCallback); ok {
				data += string(callback.Data) + " "
			}
		}
	}
	for _, want := range []string{"game:kat", "game:quiz", "game:daily", "game:top", "hub:protection"} {
		if !strings.Contains(data, want) {
			t.Errorf("missing %s", want)
		}
	}
}
