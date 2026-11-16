package utils

import (
	"strings"
	"testing"

	td "github.com/AshokShau/gotdbot"
)

func TestRichButtonsKeepCallbacksColorsAndEscapeValues(t *testing.T) {
	keyboard := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{
		{Text: `<تشغيل>`, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(`play_"<&`)}, Style: td.ButtonStylePrimary{}},
		{Text: "Stop", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("play_stop")}, Style: td.ButtonStyleDanger{}},
		{Text: "Open", Type: &td.InlineKeyboardButtonTypeUrl{Url: `https://t.me/example?x=1&y=2`}},
	}}}
	got := richControlHTML("<b>ShaHm</b>\nPlaying", keyboard, "green")
	for _, want := range []string{`<blockquote><b>ShaHm</b><br>Playing</blockquote>`, `style="success"`, `style="danger"`, `&lt;تشغيل&gt;`, `data="play_&#34;&lt;&amp;"`, `url="https://t.me/example?x=1&amp;y=2"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if keyboard.Rows[0][0].Text != `<تشغيل>` {
		t.Fatal("shared keyboard mutated")
	}
	if strings.Index(got, "</blockquote>") > strings.Index(got, "<tg-button-row>") {
		t.Fatal("controls nested inside quote")
	}
}
func TestQuotedPlainTextDoesNotBecomeMarkup(t *testing.T) {
	got, err := botTextHTML(&td.Client{}, `<script>alert("x")</script> & أغنية`, "", nil)
	if err != nil || strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("unsafe plain text %q, %v", got, err)
	}
	if !richRTL(got) || richRTL("English") {
		t.Fatal("wrong text direction")
	}
}
