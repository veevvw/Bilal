package bot

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"html"

	"github.com/AshokShau/gotdbot"
)

func privacyText(botName, policyURL, supportURL string, chatIDs ...int64) string {
	policyLink := tr(firstChatID(chatIDs), "Read the full policy in the player: <b>Profile → Privacy policy</b>.")
	if policyURL != "" {
		policyLink = fmt.Sprintf(tr(firstChatID(chatIDs), `Read the <a href="%s">full privacy policy</a>.`), html.EscapeString(policyURL))
	}
	return fmt.Sprintf(tr(firstChatID(chatIDs), `<b>Privacy · %s</b>

<b>Data we process</b>
Telegram IDs, available profile details and authentication data; commands, searches, submitted media, playlists and room settings. Connection and diagnostic logs may contain IDs, IP addresses and error details.

<b>Shared rooms</b>
Your displayed profile, presence, track requests and room messages are visible to other listeners. Recent room chat stays in server memory (up to 50 messages). Voice audio is relayed through our server to participants; ShaHm does not record it. Participants may make their own copies or recordings.

<b>Storage and services</b>
IDs, playlists, permissions and settings are stored in the database until removed. The player saves preferences and recent tracks on your device. Telegram, hosting, media services, CDNs and Google's STUN service process data needed to deliver their features.

<b>Your choices</b>
Microphone access requires permission. Bot message access is requested separately. Delete playlists in the player, clear site storage to remove local data, or ask the operator for access, correction or deletion. Blocking the bot stops contact but does not erase stored data.

%s
<a href="%s">Contact support</a> and ask for a private conversation with the operator for data requests. Do not post sensitive details in a public group.`), html.EscapeString(botName), policyLink, html.EscapeString(supportURL))
}

func privacyHandler(c *gotdbot.Client, m *gotdbot.Message) error {
	text := privacyText(config.BrandName, config.PrivacyPolicyURL, config.SupportGroup, m.ChatId)
	text += "\n\n" + protectionText(m.ChatId, "<b>الألعاب والجوائز</b>\nتُحفظ معرّفات المجموعة والعضو والاسم الظاهر والنقاط والانتصارات ومفاتيح منع تكرار المكافآت في قاعدة البيانات. تبقى محاولات المسابقات مؤقتاً في ذاكرة السيرفر. لا تحفظ الألعاب نص إجابات كت. النقاط والألقاب ترفيهية دون قيمة مالية.", "<b>Games and rewards</b>\nGroup and user IDs, display names, points, wins and reward claim keys are stored in the database. Quiz attempts stay temporarily in server memory. Kat answer texts are not saved by the games system. Points and titles are for entertainment and have no monetary value.")
	text += "\n\n" + protectionText(m.ChatId, "<b>حماية المحادثات</b>\nعند تفعيل الحماية تُفحص الرسائل والتعليقات والروابط والوسائط وفق قواعد المشرف. تُحفظ الإعدادات والكلمات المحظورة وعدادات التحذير في قاعدة البيانات. تُستخدم بصمات مؤقتة للنصوص في الذاكرة لكشف السبام، دون حفظ سجل محتوى الرسائل في قاعدة البيانات.", "<b>Chat protection</b>\nWhen enabled, messages, captions, links and media are checked against administrator rules. Settings, blocked words and warning counts are stored in the database. Temporary text hashes in server memory detect flooding; message contents are not stored as a moderation log in the database.")
	_, err := utils.ReplyBotText(c, m, text, &gotdbot.SendMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	return err
}
