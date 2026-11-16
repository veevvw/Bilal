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
	"sync"

	td "github.com/AshokShau/gotdbot"
)

var skipVotes = struct {
	sync.Mutex
	m map[int64]map[int64]bool
}{m: make(map[int64]map[int64]bool)}

// skipHandler handles the /skip command (Supports Admin instant skip & Member vote-to-skip).
func skipHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "The bot is not streaming."), nil)
		return nil
	}

	// If the user is an admin, skip immediately without voting
	if adminMode(c, m) {
		resetSkipVotes(chatID)
		return webapp.PlayNext(c, chatID)
	}

	userID := m.SenderID()
	if userID == 0 {
		userID = m.ChatId
	}

	skipVotes.Lock()
	if skipVotes.m[chatID] == nil {
		skipVotes.m[chatID] = make(map[int64]bool)
	}

	if skipVotes.m[chatID][userID] {
		skipVotes.Unlock()
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "You have already voted to skip this track."), nil)
		return nil
	}

	skipVotes.m[chatID][userID] = true
	currentVotes := len(skipVotes.m[chatID])
	skipVotes.Unlock()

	requiredVotes := 3 // الحد المطلوب لتخطي الأغنية (يمكنك تعديله حسب رغبتك)

	if currentVotes >= requiredVotes {
		resetSkipVotes(chatID)
		_, _ = utils.ReplyBotText(c, m, tr(m.ChatId, "<b>Required votes reached. Skipping track...</b>"), replyOpts)
		return webapp.PlayNext(c, chatID)
	}

	markup := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: fmt.Sprintf("👍 Vote to Skip (%d/%d)", currentVotes, requiredVotes),
					Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("vote:skip")},
				},
			},
		},
	}

	_, _ = utils.ReplyBotText(c, m, fmt.Sprintf(tr(m.ChatId, "<b>Member requested to skip the track.</b> Current votes: <b>%d/%d</b>\nClick the button below to vote."), currentVotes, requiredVotes), &td.SendMessageOpts{
		ReplyMarkup: markup,
		ParseMode:   "HTML",
	})

	return nil
}

// voteSkipCallbackHandler handles the callback when a user clicks the vote button
func voteSkipCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	chatID := cb.ChatId
	userID := cb.SenderUserId

	if !cache.ChatCache.IsActive(chatID) {
		return cb.Answer(c, tr(cb.ChatId, "There is no active playback."), &td.CallbackAnswerOpts{ShowAlert: true})
	}

	skipVotes.Lock()
	if skipVotes.m[chatID] == nil {
		skipVotes.m[chatID] = make(map[int64]bool)
	}

	if skipVotes.m[chatID][userID] {
		skipVotes.Unlock()
		return cb.Answer(c, tr(cb.ChatId, "You have already voted!"), &td.CallbackAnswerOpts{ShowAlert: true})
	}

	skipVotes.m[chatID][userID] = true
	currentVotes := len(skipVotes.m[chatID])
	skipVotes.Unlock()

	requiredVotes := 3

	if currentVotes >= requiredVotes {
		resetSkipVotes(chatID)
		_ = cb.Answer(c, tr(cb.ChatId, "Required votes reached! Skipping track..."), &td.CallbackAnswerOpts{ShowAlert: true})
		_, _ = utils.SendBotText(c, chatID, tr(cb.ChatId, "<b>Track skipped successfully via member votes.</b>"), &td.SendMessageOpts{ParseMode: "HTML"})
		return webapp.PlayNext(c, chatID)
	}

	return cb.Answer(c, fmt.Sprintf(tr(cb.ChatId, "Vote recorded successfully! (%d/%d)"), currentVotes, requiredVotes), nil)
}

func resetSkipVotes(chatID int64) {
	skipVotes.Lock()
	delete(skipVotes.m, chatID)
	skipVotes.Unlock()
}
