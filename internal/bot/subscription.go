package bot

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func sourceChannelKeyboard(chatID int64, verify bool) *td.ReplyMarkupInlineKeyboard {
	rows := [][]td.InlineKeyboardButton{{{Text: protectionText(chatID, "قناة السورس", "Source channel"), Type: &td.InlineKeyboardButtonTypeUrl{Url: config.SupportChannel}}}}
	if verify {
		rows = append(rows, []td.InlineKeyboardButton{menuButton(protectionText(chatID, "تحقق من الاشتراك", "Check subscription"), "subscription:check", td.ButtonStyleSuccess{})})
	}
	return &td.ReplyMarkupInlineKeyboard{Rows: rows}
}

func subscribedStatus(status td.ChatMemberStatus) bool {
	switch s := status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator, *td.ChatMemberStatusMember:
		return true
	case *td.ChatMemberStatusRestricted:
		return s.IsMember
	}
	return false
}

// Fail closed when Telegram cannot confirm membership; never cache a user's status.
func sourceSubscribed(c *td.Client, userID int64) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	username := strings.TrimPrefix(strings.TrimRight(config.SupportChannel, "/"), "https://t.me/")
	chat, err := c.SearchPublicChat(username)
	if err != nil {
		return false, err
	}
	member, err := c.GetChatMember(chat.Id, &td.MessageSenderUser{UserId: userID})
	if err != nil {
		return false, err
	}
	return member != nil && subscribedStatus(member.Status), nil
}

func subscriptionNotice(chatID int64, failed bool) string {
	if failed {
		return protectionText(chatID, "تعذر التحقق من الاشتراك الآن. حاول مجدداً بعد قليل.", "Unable to verify your subscription. Please try again shortly.")
	}
	return protectionText(chatID, "اشترك في قناة السورس أولاً، ثم اضغط تحقق من الاشتراك لاستخدام البوت.", "Join the source channel, then check your subscription to use the bot.")
}

func requireSourceSubscription(c *td.Client, m *td.Message) bool {
	ok, err := sourceSubscribed(c, m.SenderID())
	if ok {
		return true
	}
	_, _ = utils.ReplyBotText(c, m, subscriptionNotice(m.ChatId, err != nil), &td.SendMessageOpts{ReplyMarkup: sourceChannelKeyboard(m.ChatId, true)})
	return false
}

func subscriptionCallbackGuard(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	ok, err := sourceSubscribed(c, cb.SenderUserId)
	if cb.DataString() == "subscription:check" {
		text := subscriptionNotice(cb.ChatId, err != nil)
		if ok {
			text = protectionText(cb.ChatId, "تم التحقق من اشتراكك. أرسل الأمر الذي تريده الآن.", "Subscription verified. Send your command now.")
		}
		_ = cb.Answer(c, text, &td.CallbackAnswerOpts{ShowAlert: true})
		return td.EndGroups
	}
	if ok {
		return td.ContinueGroups
	}
	_ = cb.Answer(c, subscriptionNotice(cb.ChatId, err != nil), &td.CallbackAnswerOpts{ShowAlert: true})
	_, _ = utils.SendBotText(c, cb.ChatId, subscriptionNotice(cb.ChatId, err != nil), &td.SendMessageOpts{ReplyMarkup: sourceChannelKeyboard(cb.ChatId, true)})
	return td.EndGroups
}
