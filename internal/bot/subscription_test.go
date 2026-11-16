package bot

import (
	td "github.com/AshokShau/gotdbot"
	"testing"
)

func TestSourceSubscriptionMembership(t *testing.T) {
	for _, test := range []struct {
		status td.ChatMemberStatus
		want   bool
	}{
		{&td.ChatMemberStatusCreator{}, true}, {&td.ChatMemberStatusAdministrator{}, true},
		{&td.ChatMemberStatusMember{}, true}, {&td.ChatMemberStatusLeft{}, false},
		{&td.ChatMemberStatusBanned{}, false}, {nil, false},
		{&td.ChatMemberStatusRestricted{IsMember: true}, true},
		{&td.ChatMemberStatusRestricted{IsMember: false}, false},
	} {
		if got := subscribedStatus(test.status); got != test.want {
			t.Errorf("%T = %v, want %v", test.status, got, test.want)
		}
	}
}

func TestKatHasOnlySourceChannelButton(t *testing.T) {
	k := sourceChannelKeyboard(-1, false)
	if len(k.Rows) != 1 || len(k.Rows[0]) != 1 {
		t.Fatal("kat must have only the channel button")
	}
	u, ok := k.Rows[0][0].Type.(*td.InlineKeyboardButtonTypeUrl)
	if !ok || u.Url != "https://t.me/qq4q44q" {
		t.Fatal("wrong source channel link")
	}
}
