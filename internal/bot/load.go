/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/commands"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"strings"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
)

var startTime = time.Now()

func LoadModules(c *gotdbot.Client, cancelCleanup func(int64)) {
	utils.ButtonThemeForChat = func(chatID int64) string { return db.Instance.GetButtonTheme(chatID) }
	handlers := map[string]func(*gotdbot.Client, *gotdbot.Message) error{}
	register := func(name string, handler func(*gotdbot.Client, *gotdbot.Message) error) {
		guarded := func(client *gotdbot.Client, m *gotdbot.Message) error {
			if !commands.AcceptsText(name, m.Text()) {
				return gotdbot.EndGroups
			}
			if !requireSourceSubscription(client, m) {
				return gotdbot.EndGroups
			}
			return handler(client, m)
		}
		c.OnCommand(name, guarded)
		handlers[name] = guarded
	}
	register("protection", protectionHandler)
	register("yout", youtHandler)
	register("commands", groupCommandsHandler)
	for command, action := range map[string]string{"games": "menu", "kattweet": "kat", "daily": "daily", "points": "points", "leaderboard": "top", "quiz": "quiz", "math": "math", "riddle": "riddle", "questions": "questions", "challenge": "challenge"} {
		register(command, gamesCommand(action))
	}
	register("colors", colorsHandler)
	for _, action := range []string{"ban", "unban", "mute", "unmute", "warn", "clearwarns"} {
		register(action, moderationCommand(action))
	}
	register("blockword", blockedWordCommand("add"))
	register("unblockword", blockedWordCommand("remove"))
	register("blockedwords", blockedWordCommand("list"))

	register("language", languageHandler)
	register("lang", languageHandler)
	register("reload", reloadAdminCacheHandler)
	register("authList", authListHandler)
	register("auths", authListHandler)
	register("auth", addAuthHandler)
	register("addAuth", addAuthHandler)
	register("removeAuth", removeAuthHandler)
	register("rmAuth", removeAuthHandler)
	register("broadcast", broadcastHandler)
	register("gCast", broadcastHandler)
	register("stop_gcast", cancelBroadcastHandler)
	register("stop_broadcast", cancelBroadcastHandler)
	register("av", activeVcHandler)
	register("activevc", activeVcHandler)
	register("logger", loggerHandler)
	register("privacy", privacyHandler)
	register("loop", loopHandler)
	register("pause", pauseHandler)
	register("resume", resumeHandler)
	register("cplist", createPlaylistHandler)
	register("createplaylist", createPlaylistHandler)
	register("deleteplaylist", deletePlaylistHandler)
	register("queue", queueHandler)
	register("seek", seekHandler)
	register("sh", shellCommand)
	register("skip", skipHandler)
	register("stop", stopHandler)
	register("end", stopHandler)
	register("start", startHandler)
	register("help", helpHandler)
	register("ping", pingHandler)
	register("play", playHandler)
	register("p", playHandler)
	register("fplay", fPlayHandler)
	register("fp", fPlayHandler)
	register("remove", removeHandler)
	register("settings", settingsHandler)
	register("player", webAppCommandHandler)
	register("addtoplaylist", addToPlaylistHandler)
	register("addtoplist", addToPlaylistHandler)
	register("removefromplaylist", removeFromPlaylistHandler)
	register("rmplist", removeFromPlaylistHandler)
	register("plistinfo", playlistInfoHandler)
	register("playlistinfo", playlistInfoHandler)
	register("myplaylists", myPlaylistsHandler)
	register("myplist", myPlaylistsHandler)
	register("stats", statsHandler)
	c.OnMessageGroup(protectionNewMessage, nil, -20)
	c.OnMessageGroup(gameReplyHandler, gameReplyFilter, -15)
	c.OnUpdateMessageSendSucceeded(func(_ *gotdbot.Client, u *gotdbot.UpdateMessageSendSucceeded) error {
		if u.Message != nil {
			gameEngine.MoveMessage(u.Message.ChatId, u.OldMessageId, u.Message.Id)
		}
		return nil
	}, nil)
	c.OnUpdateMessageContent(protectionEditedMessage, nil)
	c.AddUpdateChatMemberHandlerGroup(protectionMember, nil, -20)
	c.OnMessageGroup(func(client *gotdbot.Client, m *gotdbot.Message) error {
		name, ok := commands.Resolve(m.Text())
		if !ok || m.IsOutgoing {
			return gotdbot.ContinueGroups
		}
		if handler := handlers[name]; handler != nil {
			if err := handler(client, m); err != nil {
				return err
			}
			return gotdbot.EndGroups
		}
		return gotdbot.ContinueGroups
	}, func(m *gotdbot.Message) bool {
		if m.EditDate > 0 || m.IsChannelPost {
			return false
		}
		_, ok := commands.Resolve(m.Text())
		return ok
	}, -10)
	c.OnUpdateNewCallbackQuery(protectionCallback, callbackquery.Prefix("protect:"))
	c.OnUpdateNewCallbackQuery(groupHubCallback, callbackquery.Prefix("hub:"))
	c.OnUpdateNewCallbackQuery(gameCallback, callbackquery.Prefix("game:"))
	c.OnUpdateNewCallbackQuery(startLanguageCallback, callbackquery.Prefix("language:"))
	c.OnUpdateNewCallbackQuery(memberProtectionCallback, callbackquery.Prefix("member:"))
	c.OnUpdateNewCallbackQuery(colorsCallback, callbackquery.Prefix("colors:"))
	c.AddUpdateNewCallbackQueryHandlerGroup(subscriptionCallbackGuard, nil, -30)

	c.AddUpdateNewCallbackQueryHandlerGroup(recordTelegramChatContext, nil, -1)
	c.OnUpdateNewCallbackQuery(webAppVerifyCallbackHandler, callbackquery.Prefix("webapp_verify"))
	c.OnUpdateNewCallbackQuery(helpCallbackHandler, callbackquery.Prefix("help_"))
	c.OnUpdateNewCallbackQuery(playCallbackHandler, callbackquery.Prefix("play_"))
	c.OnUpdateNewCallbackQuery(func(client *gotdbot.Client, cb *gotdbot.UpdateNewCallbackQuery) error {
		return settingsCallbackHandler(client, cb, cancelCleanup)
	}, callbackquery.Prefix("settings_"))

	c.OnUpdateChatMember(handleParticipant, nil)
	c.OnUpdateNewGuestQuery(guestHandler, nil)
	c.OnUpdateNewInlineQuery(inlineQueryHandler, nil)
	c.OnUpdateNewWidgetQuery(menuWidgetQueryHandler, func(q *gotdbot.UpdateNewWidgetQuery) bool {
		return string(q.Data) == "menu:faq" || string(q.Data) == "menu:faq:ar" || string(q.Data) == "menu:faq:en" || strings.HasPrefix(string(q.Data), "menu:language:")
	})
	c.OnUpdateNewChosenInlineResult(chosenInlineResultHandler, nil)
	c.OnUpdateNewInlineCallbackQuery(inlineCallbackHandler, nil)

	c.Logger.Debug("Handlers loaded successfully")
}
