package bot

import (
	"ashokshau/tg-web/internal/branding"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"html"

	td "github.com/AshokShau/gotdbot"
)

func newlyActivated(chat *td.Chat, old, next td.ChatMemberStatus) bool {
	active := subscribedStatus
	if kind, ok := chat.Type.(*td.ChatTypeSupergroup); ok && kind.IsChannel {
		active = func(status td.ChatMemberStatus) bool {
			if admin, ok := status.(*td.ChatMemberStatusAdministrator); ok {
				return admin.Rights != nil && admin.Rights.CanPostMessages
			}
			return isStatus[*td.ChatMemberStatusCreator](status)
		}
	}
	return !active(old) && active(next)
}

func sendActivationWelcome(c *td.Client, chat *td.Chat) {
	photo, err := branding.BannerFile()
	if err != nil {
		c.Logger.Warnf("Unable to load activation banner: %v", err)
		return
	}
	text := fmt.Sprintf(protectionText(chat.Id,
		"<b>تم تفعيل %s بنجاح</b>\nأهلاً بكم في %s.\nتابعوا قناة التحديثات لمعرفة أخبار البوت.",
		"<b>%s is now active</b>\nWelcome to %s.\nFollow the updates channel for bot news."), config.BrandName, html.EscapeString(chat.Title))
	keyboard := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{{
		Text: protectionText(chat.Id, "قناة التحديثات", "Updates channel"),
		Type: &td.InlineKeyboardButtonTypeUrl{Url: config.SupportChannel}, Style: td.ButtonStylePrimary{},
	}}}}
	if _, err = c.SendRichMessage(chat.Id, welcomeRichMessage(chat.Id, photo, text, keyboard), nil); err != nil {
		c.Logger.Warnf("Unable to send activation welcome to %d: %v", chat.Id, err)
	}
}

func activationChatType(chat *td.Chat) string {
	if kind, ok := chat.Type.(*td.ChatTypeSupergroup); ok {
		if kind.IsChannel {
			return "قناة"
		}
		return "مجموعة خارقة"
	}
	return "مجموعة عادية (غير خارقة)"
}

func activationChatLink(c *td.Client, chat *td.Chat) string {
	var invite *td.ChatInviteLink
	switch kind := chat.Type.(type) {
	case *td.ChatTypeSupergroup:
		if group, err := c.GetSupergroup(kind.SupergroupId); err == nil && group.Usernames != nil {
			if len(group.Usernames.ActiveUsernames) > 0 {
				return "https://t.me/" + group.Usernames.ActiveUsernames[0]
			}
		}
		if info, err := c.GetSupergroupFullInfo(kind.SupergroupId); err == nil {
			invite = info.InviteLink
		}
	case *td.ChatTypeBasicGroup:
		if info, err := c.GetBasicGroupFullInfo(kind.BasicGroupId); err == nil {
			invite = info.InviteLink
		}
	}
	if invite == nil {
		// Create a separate link; never replace or revoke an existing invite.
		invite, _ = c.CreateChatInviteLink(chat.Id, &td.CreateChatInviteLinkOpts{Name: "ShaHm activation log"})
	}
	if invite != nil {
		return invite.InviteLink
	}
	return ""
}

func sendActivationLog(c *td.Client, chat *td.Chat) {
	if config.LoggerId == 0 {
		c.Logger.Warn("Activation log skipped: LOGGER_ID is not configured")
		return
	}
	owner := "غير متاح (تعذّر الوصول إلى بيانات المالك)"
	if admins, err := c.GetChatAdministrators(chat.Id); err == nil {
		for _, admin := range admins.Administrators {
			if !admin.IsOwner {
				continue
			}
			username := "بدون يوزر"
			if user, err := c.GetUser(admin.UserId); err == nil && user.Usernames != nil && len(user.Usernames.ActiveUsernames) > 0 {
				username = "@" + html.EscapeString(user.Usernames.ActiveUsernames[0])
			}
			owner = fmt.Sprintf("%s\n<b>ايدي المالك:</b> <code>%d</code>", username, admin.UserId)
			break
		}
	}
	linkText := "غير متاح (المحادثة خاصة ولا تتوفر صلاحية إنشاء رابط)"
	if link := activationChatLink(c, chat); link != "" {
		linkText = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link), html.EscapeString(link))
	}
	text := fmt.Sprintf("<b>تم تفعيل البوت في محادثة جديدة</b>\n<b>الاسم:</b> %s\n<b>النوع:</b> %s\n<b>ايدي المحادثة:</b> <code>%d</code>\n<b>الرابط:</b> %s\n<b>يوزر المالك:</b> %s",
		html.EscapeString(chat.Title), activationChatType(chat), chat.Id, linkText, owner)
	if _, err := utils.SendBotText(c, config.LoggerId, text, &td.SendMessageOpts{ParseMode: "HTML"}); err != nil {
		c.Logger.Warnf("Unable to send activation log for %d: %v", chat.Id, err)
	}
}
