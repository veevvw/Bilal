package webapp

import (
	"ashokshau/tg-web/internal/voicechat"
	td "github.com/AshokShau/gotdbot"
	"time"
)

// Existing assistant memberships are preferred. Private chats use a short-lived,
// one-member link created by this bot; the link never reaches a user message.
func prepareAssistant(bot *td.Client, chatID int64) error {
	p, ok := voicechat.Default.(interface{ Prepare(int64, string) error })
	if !ok || !voicechat.Default.Enabled(chatID) {
		return nil
	}
	if bot == nil {
		return p.Prepare(chatID, "")
	}
	return ensureAssistantJoined(bot, p, chatID)
}

type assistantJoinBot interface {
	GetChatMember(int64, td.MessageSender) (*td.ChatMember, error)
	CreateChatInviteLink(int64, *td.CreateChatInviteLinkOpts) (*td.ChatInviteLink, error)
	RevokeChatInviteLink(int64, string) (*td.ChatInviteLinks, error)
}
type assistantPreparer interface{ Prepare(int64, string) error }

func ensureAssistantJoined(bot assistantJoinBot, p assistantPreparer, chatID int64) error {
	join := func() error {
		link, err := bot.CreateChatInviteLink(chatID, &td.CreateChatInviteLinkOpts{Name: "ShaHm assistant", ExpirationDate: int32(time.Now().Add(5 * time.Minute).Unix()), MemberLimit: 1})
		if err != nil || link == nil || link.InviteLink == "" {
			return &voicechat.BridgeError{Code: "invite_permission"}
		}
		defer func() { _, _ = bot.RevokeChatInviteLink(chatID, link.InviteLink) }()
		return p.Prepare(chatID, link.InviteLink)
	}
	if identity, ok := p.(interface{ AssistantID() (int64, error) }); ok {
		id, err := identity.AssistantID()
		if err != nil {
			return err
		}
		member, err := bot.GetChatMember(chatID, &td.MessageSenderUser{UserId: id})
		if err != nil || member == nil {
			return join()
		}
		switch status := member.Status.(type) {
		case *td.ChatMemberStatusBanned:
			return &voicechat.BridgeError{Code: "assistant_banned"}
		case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator, *td.ChatMemberStatusMember:
		case *td.ChatMemberStatusRestricted:
			if !status.IsMember {
				return join()
			}
		default:
			return join()
		}
	}
	err := p.Prepare(chatID, "")
	if err == nil || !voicechat.NeedsMembership(err) || bot == nil {
		return err
	}
	return join()
}
