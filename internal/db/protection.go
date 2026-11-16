package db

import (
	"ashokshau/tg-web/internal/moderation"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (db *Database) GetProtection(chatID int64) (moderation.Settings, error) {
	defaults := moderation.Default()
	if db == nil {
		return defaults, errors.New("database unavailable")
	}
	chat, err := db.getChat(chatID)
	if err != nil {
		return defaults, err
	}
	if chat == nil || chat.Protection == nil {
		return defaults, nil
	}
	s := *chat.Protection
	s.BlockedWords = append([]string(nil), s.BlockedWords...)
	return s.WithDefaults(), nil
}

// SetProtectionField uses individual atomic updates so concurrent panel clicks
// cannot overwrite other rules or the word list.
func (db *Database) SetProtectionField(chatID int64, field string, value any) error {
	if !moderation.ValidSetting(field, value) {
		return errors.New("invalid protection setting")
	}
	ctx, cancel := db.ctx()
	defer cancel()
	if err := db.AddChat(chatID); err != nil {
		return err
	}
	// Initialize the full document once; subsequent updates only touch one field.
	_, err := db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID, "protection": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"protection": moderation.Default()}})
	if err != nil {
		return err
	}
	_, err = db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{"protection." + field: value}})
	db.chatCache.Delete(toKey(chatID))
	return err
}

func (db *Database) ChangeBlockedWord(chatID int64, word string, add bool) error {
	word = moderation.Normalize(word)
	if word == "" || len([]rune(word)) > 80 {
		return errors.New("invalid blocked word")
	}
	s, err := db.GetProtection(chatID)
	if err != nil {
		return err
	}
	if add && len(s.BlockedWords) >= 100 {
		return errors.New("blocked word limit reached")
	}
	if err = db.SetProtectionField(chatID, "words", s.Words); err != nil {
		return err
	}
	op := "$pull"
	if add {
		op = "$addToSet"
	}
	ctx, cancel := db.ctx()
	defer cancel()
	_, err = db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{op: bson.M{"protection.blocked_words": word}})
	db.chatCache.Delete(toKey(chatID))
	return err
}

func (db *Database) SetButtonTheme(chatID int64, theme string) error {
	if theme != "auto" && theme != "blue" && theme != "green" && theme != "red" {
		return errors.New("invalid button theme")
	}
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{"button_theme": theme}}, options.UpdateOne().SetUpsert(true))
	db.chatCache.Delete(toKey(chatID))
	return err
}
func (db *Database) GetButtonTheme(chatID int64) string {
	if db == nil {
		return "auto"
	}
	chat, err := db.getChat(chatID)
	if err != nil || chat == nil || chat.ButtonTheme == "" {
		return "auto"
	}
	return chat.ButtonTheme
}

func (db *Database) AddProtectionWarning(chatID, userID int64) (int, error) {
	ctx, cancel := db.ctx()
	defer cancel()
	var doc struct {
		Count int `bson:"count"`
	}
	err := db.DB.Collection("protection_warnings").FindOneAndUpdate(ctx, bson.M{"_id": toKey(chatID) + ":" + toKey(userID)}, bson.M{"$inc": bson.M{"count": 1}, "$set": bson.M{"updated_at": time.Now()}}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&doc)
	return doc.Count, err
}
func (db *Database) ClearProtectionWarnings(chatID, userID int64) error {
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.DB.Collection("protection_warnings").DeleteOne(ctx, bson.M{"_id": toKey(chatID) + ":" + toKey(userID)})
	return err
}
