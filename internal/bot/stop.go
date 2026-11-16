/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

// stopHandler handles the /stop command.
func stopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if err := webapp.StopPlayback(c, chatID); err != nil {
		_, replyErr := utils.ReplyBotText(c, m, tr(chatID, err.Error()), nil)
		return replyErr
	}
	_, _ = utils.ReplyBotText(c, m, fmt.Sprintf(tr(m.ChatId, "<b>Stream ended by</b> %s"), firstName(c, m)), replyOpts)
	return nil
}
