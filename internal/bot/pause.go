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
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"

	td "github.com/AshokShau/gotdbot"
)

func pauseHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "There is no active playback."), nil)
		return nil
	}

	if _, err := webapp.Manager.Pause(c, chatID); err != nil {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Failed to pause the playback: Please try again later."), nil)
		return nil
	}

	markup := webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID)
	_, err := utils.ReplyControlMessage(c, m, fmt.Sprintf(tr(m.ChatId, "Playback has been paused by %s."), html.EscapeString(firstName(c, m))), &td.SendMessageOpts{ReplyMarkup: markup})
	return err
}

func resumeHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "There is no active playback."), nil)
		return nil
	}

	if _, err := webapp.Manager.Resume(c, chatID); err != nil {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Failed to resume the playback: Please try again later."), nil)
		return nil
	}

	markup := webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID)
	_, err := utils.ReplyControlMessage(c, m, fmt.Sprintf(tr(m.ChatId, "Playback has been resumed by %s."), html.EscapeString(firstName(c, m))), &td.SendMessageOpts{ReplyMarkup: markup})
	return err
}
