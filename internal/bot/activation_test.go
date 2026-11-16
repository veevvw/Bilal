package bot

import (
	"strings"
	"testing"

	td "github.com/AshokShau/gotdbot"
)

func TestActivationTransitions(t *testing.T) {
	group := &td.Chat{Type: &td.ChatTypeBasicGroup{}}
	channel := &td.Chat{Type: &td.ChatTypeSupergroup{IsChannel: true}}
	left := &td.ChatMemberStatusLeft{}
	member := &td.ChatMemberStatusMember{}
	admin := &td.ChatMemberStatusAdministrator{Rights: &td.ChatAdministratorRights{CanPostMessages: true}}
	noPost := &td.ChatMemberStatusAdministrator{Rights: &td.ChatAdministratorRights{}}
	for _, tc := range []struct {
		name      string
		chat      *td.Chat
		old, next td.ChatMemberStatus
		want      bool
	}{
		{"basic group join", group, left, member, true},
		{"promotion does not repeat group welcome", group, member, admin, false},
		{"channel cannot post yet", channel, left, noPost, false},
		{"channel becomes writable", channel, noPost, admin, true},
		{"channel permissions unchanged", channel, admin, admin, false},
		{"rejoin after ban", group, &td.ChatMemberStatusBanned{}, admin, true},
		{"restricted nonmember", group, left, &td.ChatMemberStatusRestricted{IsMember: false}, false},
		{"removal", group, member, left, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := newlyActivated(tc.chat, tc.old, tc.next); got != tc.want {
				t.Fatalf("activation = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWelcomeKeepsPhotoOutsideQuoteAndRichControls(t *testing.T) {
	msg := welcomeRichMessage(1, "banner.png", "<b>Welcome</b>", &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{menuButton("Commands", "help_all", td.ButtonStylePrimary{})}}})
	source := msg.Source.(*td.RichMessageSourceHtml)
	if !strings.HasPrefix(source.Text, `<img src="tg://photo?id=welcome"><blockquote>`) ||
		!strings.Contains(source.Text, `</blockquote><tg-button-row>`) ||
		!strings.Contains(source.Text, `type="callback_data"`) {
		t.Fatalf("unexpected welcome layout: %s", source.Text)
	}
	if len(source.Media) != 1 || source.Media[0].Id != "welcome" {
		t.Fatal("welcome photo not attached to rich message")
	}
}
