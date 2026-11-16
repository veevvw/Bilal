package bot

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/moderation"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func menuButton(label, data string, style td.ButtonStyle) td.InlineKeyboardButton {
	return td.InlineKeyboardButton{Text: label, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(data)}, Style: style}
}

// Odd rows have one control; even rows have two.
func groupKeyboard(title td.InlineKeyboardButton, buttons ...td.InlineKeyboardButton) *td.ReplyMarkupInlineKeyboard {
	k := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{title}}}
	for i := 0; i < len(buttons); {
		width := 1
		if (len(k.Rows)+1)%2 == 0 {
			width = 2
		}
		end := min(i+width, len(buttons))
		row := append([]td.InlineKeyboardButton(nil), buttons[i:end]...)
		if width == 2 && len(row) == 1 {
			row = append(row, title)
		}
		k.Rows = append(k.Rows, row)
		i = end
	}
	return k
}
func groupHome(chatID int64) (string, *td.ReplyMarkupInlineKeyboard) {
	t := func(ar, en string) string { return protectionText(chatID, ar, en) }
	text := t("<b>أوامر ShaHm للمجموعة</b>", "<b>ShaHm group commands</b>")
	return text, groupKeyboard(menuButton("✦ ShaHm ✦", "hub:home", td.ButtonStylePrimary{}),
		menuButton(t("🛡 الحماية والأقفال", "🛡 Protection"), "hub:protection", td.ButtonStyleSuccess{}),
		menuButton(t("🎮 الألعاب والجوائز", "🎮 Games & rewards"), "hub:games", td.ButtonStylePrimary{}),
		menuButton(t("👮 أوامر الإدارة", "👮 Moderation guide"), "hub:moderation", td.ButtonStylePrimary{}),
		menuButton(t("🎵 أوامر الموسيقى", "🎵 Music guide"), "hub:music", td.ButtonStylePrimary{}),
		menuButton(t("🎨 ألوان الأزرار", "🎨 Button colors"), "hub:colors", td.ButtonStylePrimary{}),
		menuButton(t("📖 طريقة الاستخدام", "📖 How to use"), "hub:guide", td.ButtonStylePrimary{}),
		td.InlineKeyboardButton{Text: t("📢 قناة المشروع", "📢 Project channel"), Type: &td.InlineKeyboardButtonTypeUrl{Url: config.SupportChannel}, Style: td.ButtonStylePrimary{}},
		td.InlineKeyboardButton{Text: t("💬 مجموعة الدعم", "💬 Support group"), Type: &td.InlineKeyboardButtonTypeUrl{Url: config.SupportGroup}, Style: td.ButtonStyleSuccess{}})
}
func groupCommandsHandler(c *td.Client, m *td.Message) error {
	if m.IsPrivate() {
		return protectionFailure(c, m, "أرسل الاوامر داخل المجموعة لفتح لوحة الحماية والألعاب. ومن الخاص يمكنك إدارة قناة بكتابة: حماية -100رقم_القناة", "Send the commands menu in a group. To manage a channel privately: /protection -100CHANNEL_ID.")
	}
	chat, err := c.GetChat(m.ChatId)
	if err != nil || !utils.IsGroup(chat) {
		return nil
	}
	text, k := groupHome(m.ChatId)
	_, err = utils.ReplyControlMessage(c, m, text, &td.SendMessageOpts{ReplyMarkup: k})
	return err
}
func groupHubCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if cb.ChatId >= 0 {
		return cb.Answer(c, protectionText(cb.ChatId, "افتح الاوامر من المجموعة.", "Open the commands menu in a group."), nil)
	}
	page := strings.TrimPrefix(cb.DataString(), "hub:")
	var text string
	var keyboard *td.ReplyMarkupInlineKeyboard
	switch page {
	case "home":
		text, keyboard = groupHome(cb.ChatId)
	case "protection":
		if protectionAccess(c, cb.ChatId, cb.SenderUserId, false) != nil {
			return cb.Answer(c, protectionText(cb.ChatId, "لوحة الحماية للمشرفين بصلاحية الحذف.", "Protection requires administrator delete rights."), &td.CallbackAnswerOpts{ShowAlert: true})
		}
		s, err := db.Instance.GetProtection(cb.ChatId)
		if err != nil {
			return err
		}
		text, keyboard = protectionPanel(cb.ChatId, s)
	case "games":
		return gamesMenuCallback(c, cb)
	case "colors":
		if protectionAccess(c, cb.ChatId, cb.SenderUserId, false) != nil {
			return cb.Answer(c, protectionText(cb.ChatId, "تعديل الألوان للمشرفين.", "Only administrators can change colors."), &td.CallbackAnswerOpts{ShowAlert: true})
		}
		text, keyboard = colorPanel(cb.ChatId)
	case "moderation":
		text = protectionText(cb.ChatId, "<b>إدارة الأعضاء</b>\nبالرد على رسالة العضو: حظر، كتم، تحذير، الغاء_حظر، الغاء_كتم، مسح_التحذيرات.\nأرسل حماية بالرد على العضو لعرض أزرار إدارته.\nمنع_كلمة العبارة — إضافة كلمة أو عبارة محظورة.\nسماح_كلمة العبارة — حذفها.\nالكلمات_المحظورة — عرض القائمة.\nإجراءات العقوبة للمشرفين الذين يملكون الحذف والتقييد؛ تخويل الموسيقى لا يمنح هذه الصلاحيات.", "<b>Member moderation</b>\nReply with /ban, /mute, /warn, /unban, /unmute or /clearwarns.\nReply with /protection to show member action buttons.\n/blockword phrase, /unblockword phrase, /blockedwords.\nPunishments require delete and restrict rights. Music authorization grants no moderation rights.")
	case "music":
		text = protectionText(cb.ChatId, "<b>الموسيقى</b>\nتشغيل اسم الأغنية\nتشغيل_فوري اسم الأغنية\nمؤقت — استئناف — ايقاف — تخطي\nالانتظار — تكرار عدد — تقديم ثوانٍ\nالمشغل — الاعدادات — اللغة ar أو en\nقوائمي — انشاء_قائمة — اضف_للقائمة.", "<b>Music</b>\n/play song, /fplay song\n/pause, /resume, /stop, /skip\n/queue, /loop count, /seek seconds\n/player, /settings, /language ar or en\n/myplaylists, /createplaylist, /addtoplaylist.")
	case "guide":
		text = protectionText(cb.ChatId, "<b>دليل المجموعة</b>\n🛡 الحماية: الأقفال والعقوبات.\n🎮 الألعاب: ألغاز، معلومات، حساب، كت وتحديات.\n🎵 الموسيقى: التشغيل وقوائم الانتظار.\n🎨 الألوان: تخصيص الأزرار.", "<b>Group guide</b>\n🛡 Protection: locks and penalties.\n🎮 Games: riddles, trivia, math, Kat and challenges.\n🎵 Music: playback and queues.\n🎨 Colors: button themes.")
	default:
		return nil
	}
	if keyboard == nil {
		keyboard = groupKeyboard(menuButton("✦ ShaHm ✦", "hub:home", td.ButtonStylePrimary{}), menuButton(protectionText(cb.ChatId, "🛡 الحماية", "🛡 Protection"), "hub:protection", td.ButtonStyleSuccess{}), menuButton(protectionText(cb.ChatId, "🎮 الألعاب", "🎮 Games"), "hub:games", td.ButtonStylePrimary{}))
	}
	_ = cb.Answer(c, "", nil)
	_, err := utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: keyboard})
	if err == nil {
		gameEngine.UnbindMessage(cb.ChatId, cb.MessageId)
	}
	return err
}

func protectionPage(chatID int64, s moderation.Settings, page string) (string, *td.ReplyMarkupInlineKeyboard) {
	s = s.WithDefaults()
	t := func(ar, en string) string { return protectionText(chatID, ar, en) }
	prefix := fmt.Sprintf("protect:%d:", chatID)
	buttons := []td.InlineKeyboardButton{}
	add := func(label, data string) {
		buttons = append(buttons, menuButton(label, prefix+data, td.ButtonStylePrimary{}))
	}
	lock := func(key, ar, en string, on bool) {
		icon := "🔓 "
		var style td.ButtonStyle = td.ButtonStyleDefault{}
		if on {
			icon = "🔒 "
			style = td.ButtonStyleSuccess{}
		}
		buttons = append(buttons, menuButton(icon+t(ar, en), fmt.Sprintf("%sset:%s:%t", prefix, key, !on), style))
	}
	text := t("<b>حماية ShaHm</b>\nالمشرفون مستثنون في المجموعات. قواعد القناة تشمل منشوراتها.\nفعّل الحماية ثم حدّد الأقفال والإجراء.", "<b>ShaHm protection</b>\nGroup administrators are exempt. Channel rules cover its posts.\nEnable protection, choose locks and select an action.")
	switch page {
	case "home":
		lock("enabled", "تفعيل الحماية", "Enable protection", s.Enabled)
		add(t("🔗 أقفال المحتوى", "🔗 Content locks"), "page:content")
		add(t("🖼 أقفال الوسائط", "🖼 Media locks"), "page:media")
		add(t("⚡ السبام والتكرار", "⚡ Anti-spam"), "page:spam")
		add(t("⚖ العقوبات والتحذيرات", "⚖ Penalties"), "page:penalty")
		add(t("🚫 الكلمات المحظورة", "🚫 Blocked words"), "page:words")
		lock("welcome", "الترحيب بالأعضاء", "Welcome new members", s.Welcome)
	case "content":
		lock("links", "الروابط", "Links", s.Links)
		lock("forwards", "التحويلات", "Forwards", s.Forwards)
		lock("bots", "رسائل البوتات", "Bot messages", s.Bots)
		lock("usernames", "المعرفات @", "Usernames @", s.Usernames)
		lock("mentions", "المنشن للأعضاء", "User mentions", s.Mentions)
		lock("channels", "الإرسال باسم قناة", "Posting as a channel", s.Channels)
		lock("services", "رسائل الدخول والخروج", "Join / leave service messages", s.Services)
	case "media":
		lock("media", "كل الوسائط", "All media", s.Media)
		lock("photos", "الصور", "Photos", s.Photos)
		lock("videos", "الفيديو", "Videos", s.Videos)
		lock("stickers", "الملصقات", "Stickers", s.Stickers)
		lock("animations", "المتحركات GIF", "GIF animations", s.Animations)
		lock("audios", "المقاطع الصوتية", "Audio files", s.Audios)
		lock("voices", "البصمات والدوائر", "Voice / video notes", s.Voices)
		lock("documents", "الملفات", "Documents", s.Documents)
		lock("polls", "الاستطلاعات", "Polls", s.Polls)
	case "spam":
		lock("flood", "منع السبام والتكرار", "Anti-flood / repeats", s.Flood)
		text += fmt.Sprintf(t("\nالحد الحالي: %d رسائل خلال 10 ثوانٍ. تكرار النص: %d مرات.", "\nCurrent limit: %d messages in 10 seconds; identical text: %d times."), s.FloodLimit, s.RepeatLimit)
		for _, n := range []int{3, 5, 8, 10} {
			add(fmt.Sprintf(t("الرسائل: %d", "Messages: %d"), n), fmt.Sprintf("num:flood_limit:%d", n))
		}
		for _, n := range []int{2, 3, 4, 5} {
			add(fmt.Sprintf(t("التكرار: %d", "Repeats: %d"), n), fmt.Sprintf("num:repeat_limit:%d", n))
		}
	case "penalty":
		actionLabel := map[string]string{"delete": t("حذف فقط", "Delete only"), "warn": t("تحذير ثم كتم", "Warn then mute"), "mute": t("كتم", "Mute"), "ban": t("حظر", "Ban")}[s.Action]
		text += fmt.Sprintf(t("\nالإجراء: %s\nالتحذيرات: %d — الكتم: %d دقيقة.\nالحذف فقط لا يطبق عقوبة على العضو.", "\nAction: %s\nWarnings: %d — mute: %d minutes.\nDelete only does not penalize members."), actionLabel, s.WarnLimit, s.MuteMinutes)
		for _, a := range []struct{ key, ar, en string }{{"delete", "حذف فقط", "Delete only"}, {"warn", "تحذير ثم كتم", "Warn then mute"}, {"mute", "كتم", "Mute"}, {"ban", "حظر", "Ban"}} {
			label := t(a.ar, a.en)
			if s.Action == a.key {
				label = "✓ " + label
			}
			buttons = append(buttons, menuButton(label, prefix+"action:"+a.key, td.ButtonStyleDanger{}))
		}
		for _, n := range []int{2, 3, 4, 5} {
			add(fmt.Sprintf(t("التحذيرات: %d", "Warnings: %d"), n), fmt.Sprintf("num:warn_limit:%d", n))
		}
		for _, n := range []int{5, 10, 60, 1440} {
			add(fmt.Sprintf(t("كتم %d دقيقة", "Mute %d min"), n), fmt.Sprintf("num:mute_minutes:%d", n))
		}
	case "words":
		lock("words", "فلتر الكلمات", "Word filter", s.Words)
		text += fmt.Sprintf(t("\nالكلمات المسجلة: %d\nمنع_كلمة العبارة — لإضافة كلمة.\nسماح_كلمة العبارة — لحذف كلمة.\nالكلمات_المحظورة — لعرضها.\nيمكن إدارة القنوات من الخاص بإضافة رقمها بعد الأمر.", "\nStored phrases: %d\n/blockword phrase — add.\n/unblockword phrase — remove.\n/blockedwords — list.\nInclude a channel ID after the command when managing privately."), len(s.BlockedWords))
	default:
		return protectionPage(chatID, s, "home")
	}
	if page != "home" {
		add(t("⬅ لوحة الحماية", "⬅ Protection home"), "page:home")
	}
	buttons = append(buttons, menuButton(t("🎮 الألعاب والجوائز", "🎮 Games & rewards"), "hub:games", td.ButtonStyleSuccess{}), menuButton(t("⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
	return text, groupKeyboard(menuButton(t("🛡 لوحة الحماية", "🛡 Protection panel"), prefix+"page:home", td.ButtonStylePrimary{}), buttons...)
}
