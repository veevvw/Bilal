package webapp

import (
	"ashokshau/tg-web/internal/voicechat"
	"errors"
	td "github.com/AshokShau/gotdbot"
	"testing"
)

type joinBotFake struct {
	status                 td.ChatMemberStatus
	lookupErr, errorInvite error
	created, revoked       int
	target                 int64
}

func (b *joinBotFake) GetChatMember(chat int64, sender td.MessageSender) (*td.ChatMember, error) {
	b.target = chat
	if b.lookupErr != nil {
		return nil, b.lookupErr
	}
	return &td.ChatMember{Status: b.status}, nil
}
func (b *joinBotFake) CreateChatInviteLink(chat int64, opts *td.CreateChatInviteLinkOpts) (*td.ChatInviteLink, error) {
	b.created++
	b.target = chat
	if opts.MemberLimit != 1 || opts.ExpirationDate == 0 || opts.CreatesJoinRequest {
		panic("unsafe invite settings")
	}
	if b.errorInvite != nil {
		return nil, b.errorInvite
	}
	return &td.ChatInviteLink{InviteLink: "https://t.me/+private-test"}, nil
}
func (b *joinBotFake) RevokeChatInviteLink(chat int64, link string) (*td.ChatInviteLinks, error) {
	b.revoked++
	return nil, nil
}

type joinBridgeFake struct {
	invites []string
	failure error
}

func (p *joinBridgeFake) AssistantID() (int64, error) { return 77, nil }
func (p *joinBridgeFake) Prepare(chat int64, invite string) error {
	p.invites = append(p.invites, invite)
	return p.failure
}

func TestMissingAssistantReceivesBotInviteBeforePlayback(t *testing.T) {
	for _, status := range []td.ChatMemberStatus{&td.ChatMemberStatusLeft{}, &td.ChatMemberStatusRestricted{IsMember: false}} {
		bot := &joinBotFake{status: status}
		bridge := &joinBridgeFake{}
		if err := ensureAssistantJoined(bot, bridge, -100123); err != nil {
			t.Fatal(err)
		}
		if bot.created != 1 || bot.revoked != 1 || bot.target != -100123 || len(bridge.invites) != 1 || bridge.invites[0] == "" {
			t.Fatal("invite not supplied before readiness", bot, bridge)
		}
	}
}
func TestMemberSkipsInviteAndBannedAccountIsNotReadded(t *testing.T) {
	bot := &joinBotFake{status: &td.ChatMemberStatusMember{}}
	bridge := &joinBridgeFake{}
	if err := ensureAssistantJoined(bot, bridge, -100); err != nil || bot.created != 0 || len(bridge.invites) != 1 || bridge.invites[0] != "" {
		t.Fatal("existing member invited again")
	}
	bot = &joinBotFake{status: &td.ChatMemberStatusBanned{}}
	bridge = &joinBridgeFake{}
	if err := ensureAssistantJoined(bot, bridge, -100); err == nil || bot.created != 0 || len(bridge.invites) != 0 {
		t.Fatal("banned assistant readded")
	}
}
func TestInviteFailureDoesNotContinueAndLinksAreRevoked(t *testing.T) {
	bot := &joinBotFake{lookupErr: errors.New("unknown member"), errorInvite: errors.New("invite forbidden")}
	bridge := &joinBridgeFake{}
	if err := ensureAssistantJoined(bot, bridge, -100); err == nil || len(bridge.invites) != 0 {
		t.Fatal("continued without an invitation")
	}
	bot = &joinBotFake{status: &td.ChatMemberStatusLeft{}}
	bridge = &joinBridgeFake{failure: &voicechat.BridgeError{Code: "join_pending"}}
	if err := ensureAssistantJoined(bot, bridge, -100); err == nil || bot.revoked != 1 {
		t.Fatal("join failure ignored or invite leaked")
	}
}
