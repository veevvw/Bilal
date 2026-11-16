/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/branding"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"html"
	"runtime"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// pingHandler handles the /ping command.
func pingHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)

	start := time.Now()

	msg, err := utils.ReplyBotText(c, m, tr(m.ChatId, "Pinging… please wait…"), nil)
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := getFormattedDuration(time.Since(startTime), m.ChatId)

	response := fmt.Sprintf(
		tr(m.ChatId, "<b>📊 System Performance Metrics</b>\n\n")+
			tr(m.ChatId, "<b>Bot Latency:</b> <code>%d ms</code>\n")+
			tr(m.ChatId, "<b>Uptime:</b> <code>%s</code>\n")+
			tr(m.ChatId, "<b>Go Routines:</b> <code>%d</code>\n"),
		latency, uptime, runtime.NumGoroutine(),
	)

	_, err = utils.EditBotText(c, msg, response, &td.EditMessageTextOpts{ParseMode: "HTML"})
	return err
}

// startHandler handles the /start command.
func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	go storeChatToDB(chatID)

	if m.IsPrivate() {
		return sendPrivateStart(c, m)
	}

	deleteCmd(c, m)

	uptime := getFormattedDuration(time.Since(startTime), m.ChatId)
	htmlText := fmt.Sprintf(
		tr(m.ChatId, "<h3>%s is ready!</h3>\n")+
			tr(m.ChatId, "<p><b>Uptime:</b> <code>%s</code></p>\n")+
			tr(m.ChatId, "<p><i>A feature-rich Web Player music bot for your Telegram groups. Play your favorite tracks seamlessly.</i></p>\n\n")+
			tr(m.ChatId, "<p><tg-button type=\"url\" url=\"%s\">Updates</tg-button> <tg-button type=\"url\" url=\"%s\">Group</tg-button></p>"),
		config.BrandName, uptime, config.SupportChannel, config.SupportGroup,
	)
	_, err := m.ReplyRichMessage(c, &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: utils.RichControlHTML(htmlText, nil)}}, nil)
	return err
}

func sendPrivateStart(c *td.Client, m *td.Message) error {
	photo, err := branding.BannerFile()
	if err != nil {
		return err
	}
	caption := protectionText(m.ChatId,
		"<b>أهلاً، %s!</b>\nأنا بوت %s لتشغيل الموسيقى في مجموعتك.\nاستخدم الأزرار لعرض أوامر الميوزك أو إضافتي إلى مجموعتك.",
		"<b>Welcome, %s!</b>\nI'm %s, your group music bot.\nUse the buttons to view music commands or add me to your group.")
	text := fmt.Sprintf(caption, html.EscapeString(firstName(c, m)), config.BrandName)
	_, err = m.ReplyRichMessage(c, welcomeRichMessage(m.ChatId, photo, text,
		localizedKeyboard(m.ChatId, utils.AddMeMarkup(c.Me().Usernames.EditableUsername))), nil)
	return err
}

// The image stays outside the quote; controls keep the original rich text style.
func welcomeRichMessage(chatID int64, photo, text string, keyboard *td.ReplyMarkupInlineKeyboard) *td.InputRichMessage {
	return &td.InputRichMessage{
		IsRtl: chatLanguage(chatID) == "ar",
		Source: &td.RichMessageSourceHtml{
			Text: `<img src="tg://photo?id=welcome">` + utils.RichControlHTMLForChat(chatID, text, keyboard),
			Media: []td.InputRichMessageMedia{{Id: "welcome", Media: &td.InputMessagePhoto{
				Photo: &td.InputPhoto{Photo: &td.InputFileLocal{Path: photo}},
			}}},
		},
	}
}
