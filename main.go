/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package main

import (
	"ashokshau/tg-web/internal/bot"
	"ashokshau/tg-web/internal/cleanup"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/i18n"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/voicechat"
	"ashokshau/tg-web/internal/webapp"
	"os"
	"time"

	"github.com/AshokShau/gotdbot"
)

//go:generate go run github.com/AshokShau/gotdbot/scripts/tools

func main() {
	if err := config.LoadEnv(); err != nil {
		panic(err)
	}

	if err := db.InitDatabase(); err != nil {
		panic("failed to connect database: " + err.Error())
	}

	tdDir := "td"
	_ = os.Remove(tdDir)

	libPath := "./libtdjson.so.1.8.70"
	manager := gotdbot.NewClientManager(libPath)
	clientConfig := gotdbot.DefaultClientConfig()
	clientConfig.AutoRetry = &gotdbot.AutoRetry{
		ChatNotFound: true,
		MaxFloodWait: 5 * time.Second,
	}

	clientConfig.UpdateWorkers = 8

	clientConfig.DatabaseDirectory = tdDir
	client, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.Token, clientConfig)
	if err != nil {
		panic("failed to register client: " + err.Error())
	}

	if config.DlBotToken != "" {
		dlClientConfig := gotdbot.DefaultClientConfig()
		dlClientConfig.AutoRetry = &gotdbot.AutoRetry{
			ChatNotFound: true,
		}
		dlClientConfig.DatabaseDirectory = tdDir + "_dl"
		_ = os.Remove(dlClientConfig.DatabaseDirectory)

		dlClient, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.DlBotToken, dlClientConfig)
		if err != nil {
			client.Logger.Warnf("failed to register dl client: %s", err.Error())
			downloader.DlBot = client
		} else {
			downloader.DlBot = dlClient
			dlClient.Logger.Infof("dl client registered")
		}
	}

	if config.VoiceChatEnabled {
		assistant, err := voicechat.New()
		if err != nil {
			panic("cannot initialize voice assistant")
		}
		stopAssistant, err := assistant.Start(config.VoiceChatPython, config.VoiceChatScript)
		if err != nil {
			panic(err)
		}
		defer stopAssistant()
		voicechat.Default = assistant
	}

	cleanupScheduler, stopCleanup := cleanup.Start(client)
	bot.LoadModules(client, cleanupScheduler.CancelChat)
	language, _ := db.Instance.GetLanguage(config.LoggerId)
	_, _ = utils.SendBotText(client, config.LoggerId, i18n.Text(language, "The bot has started!"), nil)
	webapp.RegisterRoutes(client, cleanupScheduler.CancelChat)

	manager.Idle()
	stopCleanup()
	_ = db.Instance.Close()
	client.Logger.Info("The bot is shutting down...")
	_ = os.Remove(config.DownloadsDir)
}
