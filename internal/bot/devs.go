/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func activeVcHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	activeChats := cache.ChatCache.GetActiveChats()
	var sb strings.Builder
	sb.WriteString(tr(m.ChatId, "<h3>🎵 Active Voice Chats</h3>"))

	if len(activeChats) == 0 {
		sb.WriteString(tr(m.ChatId, "<blockquote><b>🔇 No Active Chats:</b> There are currently no active voice or video chats.</blockquote>"))
	} else {
		sb.WriteString(fmt.Sprintf(tr(m.ChatId, "<p>There are currently <b>%d</b> active voice/video chat(s) running.</p>"), len(activeChats)))
		sb.WriteString("<details>")
		sb.WriteString(tr(m.ChatId, "<summary><b>📊 Click to Show Active Chats</b></summary>"))
		sb.WriteString("<br>")
		sb.WriteString("<table bordered striped>")
		sb.WriteString("<tr>")
		sb.WriteString("<th align='center'><b>#</b></th>")
		sb.WriteString(tr(m.ChatId, "<th align='center'><b>Chat ID</b></th>"))
		sb.WriteString(tr(m.ChatId, "<th align='center'><b>Queue</b></th>"))
		sb.WriteString(tr(m.ChatId, "<th align='left'><b>Now Playing Track Info</b></th>"))
		sb.WriteString("</tr>")

		for i, chatID := range activeChats {
			queueLength := cache.ChatCache.GetQueueLength(chatID)
			currentSong := cache.ChatCache.GetPlayingTrack(chatID)
			var trackLink string
			if currentSong != nil {
				trackName := html.EscapeString(currentSong.Name)
				trackURL := html.EscapeString(currentSong.URL)
				if trackURL == "" {
					trackURL = "https://t.me/qq4q44q"
				}
				durStr := utils.SecToMin(currentSong.Duration)
				trackLink = fmt.Sprintf("<a href='%s'>%s</a> (%s)", trackURL, trackName, durStr)
			} else {
				trackLink = tr(m.ChatId, "<i>🔇 No song playing.</i>")
			}

			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td align='center'>%d</td>", i+1))
			sb.WriteString(fmt.Sprintf("<td align='center'><code>%d</code></td>", chatID))
			sb.WriteString(fmt.Sprintf("<td align='center'>%d</td>", queueLength))
			sb.WriteString(fmt.Sprintf("<td align='left'>%s</td>", trackLink))
			sb.WriteString("</tr>")
		}

		sb.WriteString("</table>")
		sb.WriteString("</details>")
		sb.WriteString("<br>")
	}

	richMessage := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: utils.RichControlHTML(sb.String(), nil)}}
	_, err := m.ReplyRichMessage(c, richMessage, nil)
	return err
}

// Handles the /logger command to toggle logger status
func loggerHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if config.LoggerId == 0 {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Please set LOGGER_ID in .env first."), nil)
		return td.EndGroups
	}

	loggerStatus := db.Instance.GetLoggerStatus()
	args := strings.ToLower(Args(m))
	if len(args) == 0 {
		_, _ = utils.ReplyBotText(c, m, fmt.Sprintf(tr(m.ChatId, "Usage: /logger [enable|disable|on|off]\nCurrent status: %t"), loggerStatus), nil)
		return td.EndGroups
	}

	switch args {
	case "enable", "on":
		_ = db.Instance.SetLoggerStatus(true)
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Logger Enabled"), nil)
	case "disable", "off":
		_ = db.Instance.SetLoggerStatus(false)
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Logger disabled"), nil)
	default:
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Invalid argument. Use 'enable', 'disable', 'on', or 'off'."), nil)
	}

	return td.EndGroups
}
