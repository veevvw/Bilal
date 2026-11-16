package db

import (
	"ashokshau/tg-web/internal/games"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"strings"
	"time"
)

type GamePlayer struct {
	ChatID int64  `bson:"chat_id"`
	UserID int64  `bson:"user_id"`
	Name   string `bson:"name"`
	Points int    `bson:"points"`
	Wins   int    `bson:"wins"`
}

func (db *Database) GamesEnabled(chatID int64) (bool, error) {
	chat, err := db.getChat(chatID)
	if err != nil {
		return false, err
	}
	return chat == nil || !chat.GamesDisabled, nil
}
func (db *Database) SetGamesEnabled(chatID int64, enabled bool) error {
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{"games_disabled": !enabled}}, options.UpdateOne().SetUpsert(true))
	db.chatCache.Delete(toKey(chatID))
	return err
}

// Reward claims and increments share an atomic player-document update.
func (db *Database) AwardGamePoints(chatID, userID int64, name, claim string, points int, win bool) (bool, error) {
	if chatID >= 0 || userID <= 0 || claim == "" || len(claim) > 100 || points < 1 || points > 30 {
		return false, errors.New("invalid reward")
	}
	ctx, cancel := db.ctx()
	defer cancel()
	filter := bson.M{"_id": toKey(chatID) + ":" + toKey(userID), "claims": bson.M{"$ne": claim}}
	wins := 0
	if win {
		wins = 1
	}
	name = strings.TrimSpace(name)
	if len([]rune(name)) > 64 {
		name = string([]rune(name)[:64])
	}
	set := bson.M{"chat_id": chatID, "user_id": userID, "name": name, "updated_at": time.Now()}
	parts := strings.SplitN(claim, ":", 2)
	if len(parts) != 2 {
		return false, errors.New("invalid reward claim")
	}
	switch parts[0] {
	case "daily", "box", "kat":
		if _, err := time.Parse("2006-01-02", parts[1]); err != nil {
			return false, errors.New("invalid reward date")
		}
		field := parts[0] + "_claim"
		filter = bson.M{"_id": toKey(chatID) + ":" + toKey(userID), field: bson.M{"$ne": parts[1]}}
		set[field] = parts[1]
	case "win":
		if !win || points != games.AnswerReward {
			return false, errors.New("invalid answer reward")
		}
	default:
		return false, errors.New("invalid reward kind")
	}
	update := bson.M{"$inc": bson.M{"points": points, "wins": wins}, "$set": set, "$push": bson.M{"claims": bson.M{"$each": []string{claim}, "$slice": -128}}}
	_, err := db.DB.Collection("game_players").UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	return err == nil, err
}
func (db *Database) GetGamePlayer(chatID, userID int64) (GamePlayer, error) {
	ctx, cancel := db.ctx()
	defer cancel()
	p := GamePlayer{ChatID: chatID, UserID: userID}
	err := db.DB.Collection("game_players").FindOne(ctx, bson.M{"_id": toKey(chatID) + ":" + toKey(userID)}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return p, nil
	}
	return p, err
}
func (db *Database) GameLeaderboard(chatID int64) ([]GamePlayer, error) {
	ctx, cancel := db.ctx()
	defer cancel()
	cursor, err := db.DB.Collection("game_players").Find(ctx, bson.M{"chat_id": chatID}, options.Find().SetSort(bson.D{{Key: "points", Value: -1}, {Key: "wins", Value: -1}, {Key: "user_id", Value: 1}}).SetLimit(10))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var players []GamePlayer
	err = cursor.All(ctx, &players)
	return players, err
}
