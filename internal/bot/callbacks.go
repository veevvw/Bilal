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
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()
	if strings.Contains(data, "play_close") {
		_ = cb.Answer(c, tr(cb.ChatId, "Closing..."), &td.CallbackAnswerOpts{ShowAlert: false})
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	
	if strings.HasPrefix(data, "vote:skip") {
		return voteSkipCallbackHandler(c, cb)
	}
	if strings.HasPrefix(data, "song:download") {
		return downloadCurrentSongCallback(c, cb)
	}
	// ========================================================

	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
    


	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}
	if strings.Contains(data, "play_stop") {
		if err := webapp.StopPlayback(c, chatID); err != nil {
			return cb.Answer(c, tr(chatID, err.Error()), &td.CallbackAnswerOpts{ShowAlert: true})
		}
		msg := fmt.Sprintf(tr(cb.ChatId, "<b>Playback stopped.</b>\nRequested by: %s"), html.EscapeString(user.FirstName))
		_ = cb.Answer(c, tr(cb.ChatId, "Playback stopped."), nil)
		_, err = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, msg, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	if !cache.ChatCache.IsActive(chatID) {
		text := tr(cb.ChatId, "There is no active playback.")
		_ = cb.Answer(c, text, &td.CallbackAnswerOpts{ShowAlert: false})
		_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, tr(cb.ChatId, "There is no active playback."), nil)
		_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, tr(cb.ChatId, "There is no active playback."), &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		escUser := html.EscapeString(currentTrack.User)
		title := html.EscapeString(currentTrack.Name)
		if strings.TrimSpace(currentTrack.URL) != "" {
			title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(currentTrack.URL), title)
		}
		return fmt.Sprintf(tr(cb.ChatId, "%s <b>%s</b>\n\n<b>Track:</b> %s\n<b>Duration:</b> %s\n<b>Requested by:</b> %s"),
			emoji, status,
			title,
			utils.SecToMin(currentTrack.Duration),
			escUser,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to skip the current track."), nil)
			_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, tr(cb.ChatId, "Unable to skip the current track."), &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, tr(cb.ChatId, "Track skipped."), nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_pause"):
		if _, err = webapp.Manager.Pause(c, chatID); err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to pause playback."), nil)
			_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, tr(cb.ChatId, "Unable to pause playback."), &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, tr(cb.ChatId, "Playback paused."), nil)
		text := buildTrackMessage(tr(cb.ChatId, "Paused"), "⏸") + fmt.Sprintf(tr(cb.ChatId, "\n\nPaused by %s"), html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID)
		_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err = webapp.Manager.Resume(c, chatID); err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to resume playback."), &td.CallbackAnswerOpts{ShowAlert: false})
			_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, tr(cb.ChatId, "Unable to resume playback."), &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, tr(cb.ChatId, "Playback resumed."), &td.CallbackAnswerOpts{ShowAlert: true})
		text := buildTrackMessage(tr(cb.ChatId, "Now Playing"), "▶") + fmt.Sprintf(tr(cb.ChatId, "\n\nResumed by %s"), html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID)
		_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to fetch playlists."), nil)
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist(tr(cb.ChatId, "My Playlist (ShaHm)"), cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, tr(cb.ChatId, "Unable to create playlist."), &td.CallbackAnswerOpts{ShowAlert: true})
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(playlistID, song, cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to add track to playlist."), nil)
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Playlist not found."), nil)
			return nil
		}

		_ = cb.Answer(c, fmt.Sprintf(tr(cb.ChatId, "Track \"%s\" added to playlist \"%s\"."), song.Name, playlist.Name), nil)
		return nil

	case strings.HasPrefix(data, "play_now_"):
		moved := false
		for _, track := range cache.ChatCache.GetQueue(chatID) {
			if track != nil && utils.PlayNowCallbackData(track.TrackID) == data {
				moved = cache.ChatCache.MoveTrackToFront(chatID, track.TrackID)
				break
			}
		}
		if !moved {
			_ = cb.Answer(c, tr(cb.ChatId, "Track not found in queue."), nil)
			return nil
		}

		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, tr(cb.ChatId, "Unable to play the track."), nil)
			return nil
		}

		_ = cb.Answer(c, tr(cb.ChatId, "Playing now."), nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	text := buildTrackMessage(tr(cb.ChatId, "Now Playing"), "▶")
	_, _ = utils.EditControlMessage(c, cb.ChatId, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}
