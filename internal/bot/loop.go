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
	"fmt"
	"strconv"

	td "github.com/AshokShau/gotdbot"
)

func loopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, err := utils.ReplyBotText(c, m, tr(m.ChatId, "There is no active playback."), nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, err := utils.ReplyBotText(c, m, tr(m.ChatId, "<b>Loop Control</b>\n\n<b>Usage:</b> <code>/loop [count]</code>\n0 to disable looping\n1-10 to set the number of repeats"), &td.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	argsInt, err := strconv.Atoi(args)
	if err != nil {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "Invalid loop value. Please provide a number between 0 and 10."), nil)
		return nil
	}

	if argsInt < 0 || argsInt > 10 {
		_, err = utils.ReplyBotText(c, m, tr(m.ChatId, "Loop count must be between 0 and 10."), nil)
		return err
	}

	cache.ChatCache.SetLoopCount(chatID, argsInt)

	var action string
	if argsInt == 0 {
		action = tr(m.ChatId, "Looping has been disabled")
	} else {
		action = fmt.Sprintf(tr(m.ChatId, "Looping has been set to %d time(s)"), argsInt)
	}

	_, err = utils.ReplyBotText(c, m, fmt.Sprintf(tr(m.ChatId, "%s.\nChanged by: %s"), action, firstName(c, m)), nil)
	return err
}
