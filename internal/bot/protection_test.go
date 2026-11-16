package bot

import (
	td "github.com/AshokShau/gotdbot"
	"testing"
)

func TestProtectionRequiresActualModeratorRights(t *testing.T) {
	for _, test := range []struct {
		name           string
		status         td.ChatMemberStatus
		restrict, want bool
	}{
		{"member", &td.ChatMemberStatusMember{}, false, false},
		{"banned", &td.ChatMemberStatusBanned{}, true, false},
		{"owner", &td.ChatMemberStatusCreator{}, true, true},
		{"no rights", &td.ChatMemberStatusAdministrator{}, false, false},
		{"delete only", &td.ChatMemberStatusAdministrator{Rights: &td.ChatAdministratorRights{CanDeleteMessages: true}}, false, true},
		{"delete cannot punish", &td.ChatMemberStatusAdministrator{Rights: &td.ChatAdministratorRights{CanDeleteMessages: true}}, true, false},
		{"full moderator", &td.ChatMemberStatusAdministrator{Rights: &td.ChatAdministratorRights{CanDeleteMessages: true, CanRestrictMembers: true}}, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := hasProtectionRights(test.status, test.restrict); got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}
func TestCaptionAndHiddenLinksAreModerated(t *testing.T) {
	m := &td.Message{ChatId: -42, Id: 9, SenderId: &td.MessageSenderUser{UserId: 7}, Content: &td.MessagePhoto{Caption: &td.FormattedText{Text: "اضغط هنا", Entities: []td.TextEntity{{Type: &td.TextEntityTypeTextUrl{Url: "https://example.com"}}}}}}
	got := moderationMessage(m, true)
	if !got.Media || !got.HasLink || !got.Edited || got.SenderID != 7 || got.Text != " اضغط هنا" {
		t.Fatalf("caption metadata lost: %+v", got)
	}
}
