package bot

import (
	"ashokshau/tg-web/internal/games"
	td "github.com/AshokShau/gotdbot"
	"testing"
	"time"
)

func TestOnlyQuestionRepliesEnterAnswerHandler(t *testing.T) {
	previous := gameEngine
	gameEngine = games.New()
	defer func() { gameEngine = previous }()
	r, _ := gameEngine.Start(-1, "quiz", time.Now())
	gameEngine.BindMessage(-1, r.ID, 123)
	for _, test := range []struct {
		text               string
		replyChat, replyID int64
		edited             bool
		want               bool
	}{
		{"بغداد", -1, 123, false, true}, {"بغداد", 0, 123, false, true}, {"بغداد", -2, 123, false, false}, {"بغداد", -1, 124, false, false}, {"الغاز", -1, 123, false, false}, {"/stop", -1, 123, false, false}, {"بغداد", -1, 123, true, false},
	} {
		m := &td.Message{ChatId: -1, Content: &td.MessageText{Text: &td.FormattedText{Text: test.text}}, ReplyTo: &td.MessageReplyToMessage{ChatId: test.replyChat, MessageId: test.replyID}}
		if test.edited {
			m.EditDate = 1
		}
		if got := gameReplyFilter(m); got != test.want {
			t.Fatalf("%+v got %v", test, got)
		}
	}
}
