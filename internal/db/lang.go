/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"ashokshau/tg-web/internal/i18n"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetLanguage retrieves the language code for a chat.
func (db *Database) GetLanguage(chatID int64) (string, error) {
	if db == nil || db.langCache == nil {
		return i18n.Default, nil
	}
	key := toKey(chatID)
	if cached, ok := db.langCache.Get(key); ok {
		return i18n.Normalize(cached), nil
	}
	if db.langDB == nil {
		return i18n.Default, nil
	}

	ctx, cancel := db.ctx()
	defer cancel()

	var doc struct {
		Lang string `bson:"lang"`
	}
	err := db.langDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			db.langCache.Set(key, i18n.Default)
			return i18n.Default, nil
		}
		return "", err
	}
	doc.Lang = i18n.Normalize(doc.Lang)
	db.langCache.Set(key, doc.Lang)
	return doc.Lang, nil
}

// SetLanguage sets the language code for a chat.
func (db *Database) SetLanguage(ctx context.Context, chatID int64, langCode string) error {
	if langCode != "ar" && langCode != "en" {
		return errors.New("unsupported language")
	}
	_, err := db.langDB.UpdateOne(ctx,
		bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"lang": langCode}},
		options.UpdateOne().SetUpsert(true),
	)

	if err == nil {
		db.langCache.Set(toKey(chatID), langCode)
	}
	return err
}
