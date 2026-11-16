package bot

import (
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/moderation"
	"ashokshau/tg-web/internal/utils"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var protectionEngine = moderation.New()

func protectionText(chatID int64, ar, en string) string {
	if chatLanguage(chatID) == "en" {
		return en
	}
	return ar
}

// Protection authority is independent from the music player permission mode.
// Always fetch current membership; authorized music users cannot moderate chats.
func protectionAccess(c *td.Client, chatID, userID int64, restrict bool) error {
	if chatID >= 0 || userID <= 0 {
		return errors.New("administrator_required")
	}
	member, err := c.GetChatMember(chatID, &td.MessageSenderUser{UserId: userID})
	if err != nil || member == nil {
		return errors.New("administrator_required")
	}
	if hasProtectionRights(member.Status, restrict) {
		return nil
	}
	return errors.New("administrator_required")
}
func hasProtectionRights(status td.ChatMemberStatus, restrict bool) bool {
	switch status := status.(type) {
	case *td.ChatMemberStatusCreator:
		return true
	case *td.ChatMemberStatusAdministrator:
		if status.Rights != nil && status.Rights.CanDeleteMessages && (!restrict || status.Rights.CanRestrictMembers) {
			return true
		}
	}
	return false
}
func botProtectionRights(c *td.Client, chatID int64, restrict bool) error {
	return protectionAccess(c, chatID, c.Me().Id, restrict)
}
func protectionTarget(m *td.Message) (int64, string, error) {
	chatID := m.ChatId
	args := Args(m)
	if m.IsPrivate() {
		fields := strings.Fields(args)
		if len(fields) == 0 {
			return 0, "", errors.New("target_required")
		}
		id, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || id >= 0 {
			return 0, "", errors.New("target_required")
		}
		chatID = id
		args = strings.TrimSpace(strings.TrimPrefix(args, fields[0]))
	}
	return chatID, args, nil
}
func protectionFailure(c *td.Client, m *td.Message, ar, en string) error {
	_, err := utils.ReplyBotText(c, m, protectionText(m.ChatId, ar, en), nil)
	return err
}

func protectionPanel(chatID int64, s moderation.Settings) (string, *td.ReplyMarkupInlineKeyboard) {
	return protectionPage(chatID, s, "home")
}
func protectionFieldPage(field string) string {
	switch field {
	case "enabled", "welcome":
		return "home"
	case "photos", "videos", "stickers", "animations", "audios", "voices", "documents", "polls", "media":
		return "media"
	case "flood", "flood_limit", "repeat_limit":
		return "spam"
	case "warn_limit", "mute_minutes", "action":
		return "penalty"
	case "words":
		return "words"
	default:
		return "content"
	}
}
func protectionHandler(c *td.Client, m *td.Message) error {
	chatID, _, err := protectionTarget(m)
	if err != nil {
		return protectionFailure(c, m, "داخل المجموعة اكتب حماية. لإدارة قناة من الخاص: حماية -100رقم_القناة", "Send /protection in a group, or /protection -100CHANNEL_ID in private.")
	}
	if protectionAccess(c, chatID, m.SenderID(), false) != nil {
		return protectionFailure(c, m, "هذه اللوحة للمشرفين الذين يملكون صلاحية حذف الرسائل.", "This panel requires administrator delete-message rights.")
	}
	s, err := db.Instance.GetProtection(chatID)
	if err != nil {
		return err
	}
	text, keyboard := protectionPanel(chatID, s)
	if m.ChatId == chatID && m.ReplyToMessageID() != 0 {
		if message, e := c.GetMessage(chatID, m.ReplyToMessageID()); e == nil {
			if sender, ok := message.SenderId.(*td.MessageSenderUser); ok {
				text += fmt.Sprintf(protectionText(chatID, "\n<b>إدارة العضو:</b> <code>%d</code>", "\n<b>Member:</b> <code>%d</code>"), sender.UserId)
				for _, action := range []struct{ key, ar, en string }{{"warn", "تحذير", "Warn"}, {"mute", "كتم حسب المدة المحددة", "Mute for configured duration"}, {"unmute", "الغاء كتم", "Unmute"}, {"ban", "حظر", "Ban"}, {"unban", "الغاء حظر", "Unban"}, {"clearwarns", "مسح التحذيرات", "Clear warnings"}} {
					keyboard.Rows = append(keyboard.Rows, []td.InlineKeyboardButton{{Text: protectionText(chatID, action.ar, action.en), Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("member:%d:%d:%s", chatID, sender.UserId, action.key))}, Style: td.ButtonStyleDanger{}}})
				}
			}
		}
	}
	if len(keyboard.Rows) > 1 {
		buttons := []td.InlineKeyboardButton{}
		for _, row := range keyboard.Rows[1:] {
			buttons = append(buttons, row...)
		}
		keyboard = groupKeyboard(keyboard.Rows[0][0], buttons...)
	}
	_, err = utils.ReplyControlMessage(c, m, text, &td.SendMessageOpts{ReplyMarkup: keyboard})
	return err
}

func executeMemberAction(c *td.Client, chatID, userID int64, action string) error {
	if userID <= 0 || userID == c.Me().Id {
		return errors.New("protected member")
	}
	member, err := c.GetChatMember(chatID, &td.MessageSenderUser{UserId: userID})
	if err != nil || member == nil || isAdmin(member.Status) {
		return errors.New("protected member")
	}
	sender := &td.MessageSenderUser{UserId: userID}
	switch action {
	case "unban":
		if _, ok := member.Status.(*td.ChatMemberStatusBanned); !ok {
			return errors.New("not banned")
		}
		return c.SetChatMemberStatus(chatID, sender, &td.ChatMemberStatusLeft{})
	case "unmute":
		if _, ok := member.Status.(*td.ChatMemberStatusRestricted); !ok {
			return errors.New("not restricted")
		}
		return c.SetChatMemberStatus(chatID, sender, &td.ChatMemberStatusMember{})
	case "clearwarns":
		return db.Instance.ClearProtectionWarnings(chatID, userID)
	case "ban", "mute", "warn":
		return protectionPenalty(c, chatID, userID, action)
	default:
		return errors.New("invalid action")
	}
}
func memberProtectionCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) != 4 {
		return nil
	}
	chatID, e1 := strconv.ParseInt(parts[1], 10, 64)
	userID, e2 := strconv.ParseInt(parts[2], 10, 64)
	if e1 != nil || e2 != nil || chatID >= 0 || chatID != cb.ChatId {
		return nil
	}
	if protectionAccess(c, chatID, cb.SenderUserId, true) != nil || botProtectionRights(c, chatID, true) != nil {
		return cb.Answer(c, protectionText(chatID, "للمشرف والبوت صلاحيات الحذف والتقييد مطلوبة.", "You and the bot need delete and restriction rights."), &td.CallbackAnswerOpts{ShowAlert: true})
	}
	if err := executeMemberAction(c, chatID, userID, parts[3]); err != nil {
		return cb.Answer(c, protectionText(chatID, "تعذر تنفيذ الإجراء. لا يمكن معاقبة المشرفين.", "Action failed. Administrators cannot be penalized."), &td.CallbackAnswerOpts{ShowAlert: true})
	}
	return cb.Answer(c, protectionText(chatID, "تم تنفيذ الإجراء.", "Action completed."), &td.CallbackAnswerOpts{ShowAlert: true})
}
func protectionCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) < 3 || len(parts) > 5 {
		return nil
	}
	chatID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || chatID >= 0 || (cb.ChatId < 0 && cb.ChatId != chatID) {
		return nil
	}
	answer := func(ar, en string) error {
		return cb.Answer(c, protectionText(cb.ChatId, ar, en), &td.CallbackAnswerOpts{ShowAlert: true})
	}
	s, err := db.Instance.GetProtection(chatID)
	if err != nil {
		return answer("تعذر تحميل الحماية.", "Unable to load protection.")
	}
	needRestrict := (parts[2] == "action" && len(parts) == 4 && parts[3] != "delete") || (parts[2] == "set" && len(parts) == 5 && parts[3] == "enabled" && parts[4] == "true" && s.Action != "delete")
	if parts[2] == "num" && len(parts) == 5 && (parts[3] == "warn_limit" || parts[3] == "mute_minutes") {
		needRestrict = true
	}
	if protectionAccess(c, chatID, cb.SenderUserId, needRestrict) != nil {
		return answer("ليس لديك صلاحيات إدارة الحماية أو تقييد الأعضاء.", "Protection administrator / restriction rights required.")
	}
	page := "home"
	switch parts[2] {
	case "set":
		if len(parts) != 5 {
			return nil
		}
		enabled, e := strconv.ParseBool(parts[4])
		if e != nil {
			return nil
		}
		if enabled && botProtectionRights(c, chatID, (parts[3] == "enabled" && s.Action != "delete")) != nil {
			return answer("ارفع البوت مشرفاً بصلاحية الحذف، والتقييد عند تفعيل العقوبات.", "Give the bot delete rights and restriction rights for penalties.")
		}
		err = db.Instance.SetProtectionField(chatID, parts[3], enabled)
		page = protectionFieldPage(parts[3])
	case "action":
		if len(parts) != 4 || !moderation.ValidAction(parts[3]) {
			return nil
		}
		if parts[3] != "delete" && botProtectionRights(c, chatID, true) != nil {
			return answer("البوت يحتاج صلاحية تقييد الأعضاء.", "The bot needs restriction rights.")
		}
		err = db.Instance.SetProtectionField(chatID, "action", parts[3])
		page = "penalty"
	case "num":
		if len(parts) != 5 {
			return nil
		}
		number, e := strconv.Atoi(parts[4])
		if e != nil {
			return nil
		}
		err = db.Instance.SetProtectionField(chatID, parts[3], number)
		page = protectionFieldPage(parts[3])
	case "page":
		if len(parts) != 4 {
			return nil
		}
		page = parts[3]
	case "refresh":
	default:
		return nil
	}
	if err != nil {
		return answer("تعذر حفظ التغيير. حاول مجدداً.", "Unable to save changes. Please retry.")
	}
	s, err = db.Instance.GetProtection(chatID)
	if err != nil {
		return err
	}
	text, keyboard := protectionPage(chatID, s, page)
	_ = cb.Answer(c, protectionText(cb.ChatId, "تم تحديث الحماية.", "Protection updated."), nil)
	_, err = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: keyboard})
	return err
}

func moderationMessage(m *td.Message, edited bool) moderation.Message {
	value := moderation.Message{ChatID: m.ChatId, SenderID: m.SenderID(), ID: m.Id, Text: m.Text() + " " + m.Caption(), Forward: m.ForwardInfo != nil, Edited: edited}
	entities := append(append([]td.TextEntity(nil), m.Entities()...), m.CaptionEntities()...)
	for _, entity := range entities {
		switch entity.Type.(type) {
		case *td.TextEntityTypeUrl, *td.TextEntityTypeTextUrl:
			value.HasLink = true
		case *td.TextEntityTypeMention:
			value.HasUsername = true
		case *td.TextEntityTypeMentionName:
			value.HasMention = true
		}
	}
	switch m.Content.(type) {
	case *td.MessageAnimation, *td.MessageAudio, *td.MessageDocument, *td.MessagePhoto, *td.MessageSticker, *td.MessageVideo, *td.MessageVideoNote, *td.MessageVoiceNote, *td.MessagePoll, *td.MessageContact, *td.MessageLocation, *td.MessageVenue:
		value.Media = true
	}
	switch m.Content.(type) {
	case *td.MessagePhoto:
		value.Kind = "photos"
	case *td.MessageVideo:
		value.Kind = "videos"
	case *td.MessageSticker:
		value.Kind = "stickers"
	case *td.MessageAnimation:
		value.Kind = "animations"
	case *td.MessageAudio:
		value.Kind = "audios"
	case *td.MessageVoiceNote, *td.MessageVideoNote:
		value.Kind = "voices"
	case *td.MessageDocument:
		value.Kind = "documents"
	case *td.MessagePoll:
		value.Kind = "polls"
	case *td.MessageChatAddMembers, *td.MessageChatDeleteMember, *td.MessageChatJoinByLink, *td.MessageChatJoinByRequest:
		value.Service = true
	}
	if sender, ok := m.SenderId.(*td.MessageSenderChat); ok {
		value.ChannelSender = sender.ChatId != m.ChatId && !m.IsChannelPost
	}
	return value
}
func protectMessage(c *td.Client, m *td.Message, edited bool) error {
	if m == nil || m.ChatId >= 0 || m.IsOutgoing || m.SenderID() == c.Me().Id {
		return td.ContinueGroups
	}
	s, err := db.Instance.GetProtection(m.ChatId)
	if err != nil || !s.Enabled {
		return td.ContinueGroups
	}
	// Only the group's own anonymous administrator identity is exempt.
	if !m.IsChannelPost {
		if sender, ok := m.SenderId.(*td.MessageSenderChat); ok {
			if sender.ChatId == m.ChatId {
				return td.ContinueGroups
			}
		} else {
			member, e := c.GetChatMember(m.ChatId, m.SenderId)
			if e != nil || member == nil || isAdmin(member.Status) {
				return td.ContinueGroups
			}
		}
	}
	value := moderationMessage(m, edited)
	if s.Bots && !m.IsChannelPost {
		if u, e := c.GetUser(m.SenderID()); e == nil {
			_, value.Bot = u.Type.(*td.UserTypeBot)
		}
	}
	reason := protectionEngine.Reason(s, value, time.Now())
	if reason == "" {
		return td.ContinueGroups
	}
	if err = m.Delete(c, &td.DeleteMessagesOpts{Revoke: true}); err != nil {
		c.Logger.Warn("Protection could not delete message", "chat_id", m.ChatId, "reason", reason)
		return td.ContinueGroups
	}
	if !m.IsChannelPost && m.SenderID() > 0 && s.Action != "delete" {
		if botProtectionRights(c, m.ChatId, true) == nil {
			if err = protectionPenalty(c, m.ChatId, m.SenderID(), s.Action); err != nil {
				c.Logger.Warn("Protection penalty failed", "chat_id", m.ChatId, "action", s.Action)
			}
		}
	}
	return td.EndGroups
}
func protectionNewMessage(c *td.Client, m *td.Message) error { return protectMessage(c, m, false) }
func protectionEditedMessage(c *td.Client, u *td.UpdateMessageContent) error {
	m, err := c.GetMessage(u.ChatId, u.MessageId)
	if err != nil {
		return nil
	}
	return protectMessage(c, m, true)
}
func protectionMember(c *td.Client, u *td.UpdateChatMember) error {
	if u.ChatId >= 0 || u.NewChatMember == nil || u.NewChatMember.MemberId == nil || isAdmin(u.NewChatMember.Status) {
		return td.ContinueGroups
	}
	if _, ok := u.NewChatMember.Status.(*td.ChatMemberStatusMember); !ok {
		return td.ContinueGroups
	}
	if u.OldChatMember != nil {
		if _, ok := u.OldChatMember.Status.(*td.ChatMemberStatusMember); ok {
			return td.ContinueGroups
		}
	}
	userID := SenderID(u.NewChatMember.MemberId)
	if userID <= 0 || userID == c.Me().Id {
		return td.ContinueGroups
	}
	s, err := db.Instance.GetProtection(u.ChatId)
	if err != nil || !s.Enabled {
		return td.ContinueGroups
	}
	user, err := c.GetUser(userID)
	if err != nil {
		return td.ContinueGroups
	}
	if _, ok := user.Type.(*td.UserTypeBot); !ok {
		if s.Welcome {
			_, _ = utils.SendBotText(c, u.ChatId, fmt.Sprintf(protectionText(u.ChatId, "أهلاً %s في المجموعة! 🌿\nأرسل الاوامر لفتح الحماية والألعاب، أو كت للترفيه.", "Welcome %s! 🌿\nOpen /commands for protection and games, or /kattweet for a prompt."), html.EscapeString(user.FirstName)), &td.SendMessageOpts{ParseMode: "HTML"})
		}
		return td.ContinueGroups
	}
	if !s.Bots {
		return td.ContinueGroups
	}
	// Delete-only never bans a newly joined bot; it filters that bot's messages.
	if s.Action == "ban" && botProtectionRights(c, u.ChatId, true) == nil {
		_ = protectionPenalty(c, u.ChatId, userID, "ban")
	}
	return td.ContinueGroups
}
func protectionPenalty(c *td.Client, chatID, userID int64, action string) error {
	sender := &td.MessageSenderUser{UserId: userID}
	if member, err := c.GetChatMember(chatID, sender); err != nil || member == nil || isAdmin(member.Status) {
		return errors.New("protected member")
	}
	if action == "warn" {
		settings, e := db.Instance.GetProtection(chatID)
		if e != nil {
			return e
		}
		count, err := db.Instance.AddProtectionWarning(chatID, userID)
		if err != nil {
			return err
		}
		_, _ = utils.SendBotText(c, chatID, fmt.Sprintf(protectionText(chatID, "تحذير للعضو <code>%d</code>: %d/%d.", "Warning for <code>%d</code>: %d/%d."), userID, count, settings.WarnLimit), &td.SendMessageOpts{ParseMode: "HTML"})
		if count < settings.WarnLimit {
			return nil
		}
		action = "mute"
	}
	var err error
	switch action {
	case "mute":
		settings, e := db.Instance.GetProtection(chatID)
		if e != nil {
			return e
		}
		err = c.SetChatMemberStatus(chatID, sender, &td.ChatMemberStatusRestricted{IsMember: true, Permissions: &td.ChatPermissions{}, RestrictedUntilDate: int32(time.Now().Add(time.Duration(settings.MuteMinutes) * time.Minute).Unix())})
	case "ban":
		err = c.SetChatMemberStatus(chatID, sender, &td.ChatMemberStatusBanned{})
	default:
		return nil
	}
	if err == nil {
		_ = db.Instance.ClearProtectionWarnings(chatID, userID)
	}
	return err
}

func moderationCommand(action string) func(*td.Client, *td.Message) error {
	return func(c *td.Client, m *td.Message) error {
		chatID, args, err := protectionTarget(m)
		if err != nil {
			return protectionFailure(c, m, "حدد المجموعة أو القناة أولاً.", "Specify a group or channel first.")
		}
		if protectionAccess(c, chatID, m.SenderID(), true) != nil || botProtectionRights(c, chatID, true) != nil {
			return protectionFailure(c, m, "يحتاج المشرف والبوت إلى صلاحيات الحذف وتقييد الأعضاء.", "Both you and the bot need delete and restriction rights.")
		}
		var userID int64
		if reply := m.ReplyToMessageID(); reply != 0 && m.ChatId == chatID {
			message, e := c.GetMessage(chatID, reply)
			if e == nil {
				if sender, ok := message.SenderId.(*td.MessageSenderUser); ok {
					userID = sender.UserId
				}
			}
		} else {
			userID, _ = strconv.ParseInt(strings.TrimSpace(args), 10, 64)
		}
		if userID <= 0 || userID == c.Me().Id {
			return protectionFailure(c, m, "رد على رسالة العضو أو أرسل رقم حسابه.", "Reply to the member or supply their numeric user ID.")
		}
		err = executeMemberAction(c, chatID, userID, action)
		if err != nil {
			return protectionFailure(c, m, "تعذر تنفيذ الإجراء؛ تحقق من نوع المحادثة وصلاحيات البوت.", "Action failed; check chat type and bot permissions.")
		}
		return protectionFailure(c, m, "تم تنفيذ الإجراء.", "Action completed.")
	}
}
func blockedWordCommand(action string) func(*td.Client, *td.Message) error {
	return func(c *td.Client, m *td.Message) error {
		chatID, args, err := protectionTarget(m)
		if err != nil || protectionAccess(c, chatID, m.SenderID(), false) != nil {
			return protectionFailure(c, m, "هذا الأمر لمشرفي الحماية. في الخاص أضف رقم المجموعة أو القناة.", "Protection administrators only. In private, include the target chat ID.")
		}
		if action == "list" {
			s, e := db.Instance.GetProtection(chatID)
			if e != nil {
				return e
			}
			text := strings.Join(s.BlockedWords, "، ")
			if text == "" {
				text = protectionText(chatID, "لا توجد كلمات محظورة.", "No blocked words.")
			}
			_, err = utils.ReplyBotText(c, m, text, nil)
			return err
		}
		err = db.Instance.ChangeBlockedWord(chatID, args, action == "add")
		if err != nil {
			return protectionFailure(c, m, "أرسل كلمة أو عبارة حتى 80 حرفاً. الحد الأقصى 100 عبارة.", "Supply a word or phrase up to 80 characters. Maximum 100 phrases.")
		}
		return protectionFailure(c, m, "تم تحديث الكلمات المحظورة.", "Blocked words updated.")
	}
}

func colorPanel(chatID int64) (string, *td.ReplyMarkupInlineKeyboard) {
	current := db.Instance.GetButtonTheme(chatID)
	keyboard := &td.ReplyMarkupInlineKeyboard{}
	for _, item := range []struct {
		key, ar, en string
		style       td.ButtonStyle
	}{{"auto", "ألوان حسب الوظيفة", "Semantic colors", td.ButtonStyleDefault{}}, {"blue", "أزرق", "Blue", td.ButtonStylePrimary{}}, {"green", "أخضر", "Green", td.ButtonStyleSuccess{}}, {"red", "أحمر", "Red", td.ButtonStyleDanger{}}} {
		label := protectionText(chatID, item.ar, item.en)
		if current == item.key {
			label = "✓ " + label
		}
		keyboard.Rows = append(keyboard.Rows, []td.InlineKeyboardButton{{Text: label, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("colors:%d:%s", chatID, item.key))}, Style: item.style}})
	}
	buttons := []td.InlineKeyboardButton{}
	for _, row := range keyboard.Rows {
		buttons = append(buttons, row...)
	}
	if chatID < 0 {
		buttons = append(buttons, menuButton(protectionText(chatID, "⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
	}
	keyboard = groupKeyboard(menuButton(protectionText(chatID, "🎨 الألوان", "🎨 Colors"), fmt.Sprintf("colors:%d:%s", chatID, current), td.ButtonStylePrimary{}), buttons...)
	return protectionText(chatID, "<b>ألوان أزرار ShaHm</b>\nاختر اللون الرئيسي لهذه المحادثة. أزرار الحذف والخطر تحتفظ باللون الأحمر.", "<b>ShaHm button colors</b>\nChoose this chat's accent. Destructive actions keep red."), keyboard
}
func colorsHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	if !m.IsPrivate() && protectionAccess(c, chatID, m.SenderID(), false) != nil {
		return protectionFailure(c, m, "تغيير الألوان للمشرفين فقط.", "Only administrators can change colors.")
	}
	text, keyboard := colorPanel(chatID)
	_, err := utils.ReplyControlMessage(c, m, text, &td.SendMessageOpts{ReplyMarkup: keyboard})
	return err
}
func colorsCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) != 3 {
		return nil
	}
	chatID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || chatID != cb.ChatId {
		return nil
	}
	if chatID < 0 && protectionAccess(c, chatID, cb.SenderUserId, false) != nil {
		return cb.Answer(c, protectionText(chatID, "للمشرفين فقط.", "Administrators only."), &td.CallbackAnswerOpts{ShowAlert: true})
	}
	if chatID > 0 && chatID != cb.SenderUserId {
		return nil
	}
	if err = db.Instance.SetButtonTheme(chatID, parts[2]); err != nil {
		return err
	}
	text, keyboard := colorPanel(chatID)
	_ = cb.Answer(c, protectionText(chatID, "تم تغيير اللون.", "Color updated."), nil)
	_, err = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: keyboard})
	return err
}
