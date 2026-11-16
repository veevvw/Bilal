package games

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOneWinnerAndOneAttempt(t *testing.T) {
	e := New()
	now := time.Unix(10000, 0)
	r, err := e.Start(-1, "math", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Answer(-1, r.ID, 1, (r.Question.Correct+1)%4, now); !errors.Is(err, ErrWrong) {
		t.Fatal(err)
	}
	if _, err = e.Answer(-1, r.ID, 1, r.Question.Correct, now); !errors.Is(err, ErrAttempted) {
		t.Fatal("second attempt allowed")
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for user := int64(2); user <= 50; user++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if _, err := e.Answer(-1, r.ID, id, r.Question.Correct, now); err == nil {
				winners.Add(1)
			}
		}(user)
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("%d winners", winners.Load())
	}
}
func TestRoundIsolationExpiryAndStorageRetry(t *testing.T) {
	e := New()
	now := time.Unix(10000, 0)
	r, _ := e.Start(-1, "quiz", now)
	if _, err := e.Answer(-2, r.ID, 4, r.Question.Correct, now); !errors.Is(err, ErrExpired) {
		t.Fatal("cross-chat answer accepted")
	}
	if _, err := e.Answer(-1, r.ID, 4, r.Question.Correct, now.Add(time.Minute)); !errors.Is(err, ErrExpired) {
		t.Fatal("expired answer accepted")
	}
	if _, err := e.Answer(-1, r.ID, 4, r.Question.Correct, now); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Answer(-1, r.ID, 4, r.Question.Correct, now); err != nil {
		t.Fatal("database retry blocked")
	}
	if _, err := e.Start(-1, "math", now.Add(20*time.Second)); !errors.Is(err, ErrActive) {
		t.Fatal("reserved winner replaced before reward saved")
	}
	e.Finish(-1, r.ID, 4)
	if _, err := e.Answer(-1, r.ID, 4, r.Question.Correct, now); !errors.Is(err, ErrWon) {
		t.Fatal("completed win replayed")
	}
	e.Stop(-1)
	if _, err := e.Answer(-1, r.ID, 5, r.Question.Correct, now); !errors.Is(err, ErrExpired) {
		t.Fatal("stopped round accepted")
	}
}
func TestQuestionShuffleAndPromptsWithoutCooldown(t *testing.T) {
	for _, kind := range []string{"quiz", "math", "riddle"} {
		for range 50 {
			q := NewQuestion(kind)
			if q.Correct < 0 || q.Correct > 3 || q.Title.AR == "" || q.Title.EN == "" {
				t.Fatal("invalid question")
			}
			seen := map[string]bool{}
			for _, option := range q.Options {
				if option.AR == "" || option.EN == "" || seen[option.AR] {
					t.Fatal("invalid or duplicate option")
				}
				seen[option.AR] = true
			}
		}
	}
	e := New()
	now := time.Unix(10000, 0)
	if _, err := e.Prompt(-1, 4, now); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"kat", "questions", "challenge"} {
		for range 5 {
			if _, err := e.PromptKind(-1, 4, kind, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := e.Prompt(-2, 4, now); err != nil {
		t.Fatal("different group blocked")
	}
	if _, err := e.Prompt(-1, 4, now.Add(21*time.Second)); err != nil {
		t.Fatal(err)
	}
}

func TestRoundsRestartImmediatelyAfterCompletion(t *testing.T) {
	e := New()
	now := time.Unix(10000, 0)
	for _, kind := range []string{"quiz", "math", "riddle"} {
		r, err := e.Start(-1, kind, now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = e.Answer(-1, r.ID, 7, r.Question.Correct, now); err != nil {
			t.Fatal(err)
		}
		e.Finish(-1, r.ID, 7)
	}
}

func TestTextAnswersAndMessageBindings(t *testing.T) {
	e := New()
	now := time.Unix(10000, 0)
	r, _ := e.Start(-1, "riddle", now)
	e.BindMessage(-1, r.ID, 123)
	if _, ok := e.ReplyRound(-2, 123); ok {
		t.Fatal("cross-chat reply accepted")
	}
	if _, ok := e.ReplyRound(-1, 124); ok {
		t.Fatal("unrelated reply accepted")
	}
	answer := r.Question.Options[r.Question.Correct]
	for _, text := range []string{answer.AR, answer.EN} {
		if got := r.Question.Choice(text); got != r.Question.Correct {
			t.Fatalf("correct typed answer rejected: %q", text)
		}
	}
	if r.Question.Choice("not an answer") != -1 {
		t.Fatal("unknown answer accepted")
	}
	if _, err := e.Answer(-1, r.ID, 1, r.Question.Choice(answer.AR), now); err != nil {
		t.Fatal(err)
	}
	e.Finish(-1, r.ID, 1)
	if _, err := e.Answer(-1, r.ID, 1, r.Question.Correct, now); !errors.Is(err, ErrWon) {
		t.Fatal("button replay after typed win accepted")
	}
	e.UnbindMessage(-1, 123)
	if _, ok := e.ReplyRound(-1, 123); ok {
		t.Fatal("edited-away question still accepts replies")
	}
}
func TestTemporaryMessageIDs(t *testing.T) {
	e := New()
	r, _ := e.Start(-1, "math", time.Unix(10000, 0))
	e.MoveMessage(-1, 100, 200)
	e.BindMessage(-1, r.ID, 100)
	if _, ok := e.ReplyRound(-1, 200); !ok {
		t.Fatal("send success before bind not handled")
	}
	e.BindMessage(-1, r.ID, 300)
	e.MoveMessage(-1, 300, 400)
	if _, ok := e.ReplyRound(-1, 400); !ok {
		t.Fatal("send success after bind not handled")
	}
	if _, ok := e.ReplyRound(-1, 300); ok {
		t.Fatal("temporary ID left bound")
	}
}
func TestAnswerNormalizationAndContentBanks(t *testing.T) {
	for input, want := range map[string]string{" الْمِـشْط! ": "مشط", "A comb.": "comb", "١٢": "12", "۱۲": "12", "−١": "-1"} {
		if got := NormalizeAnswer(input); got != want {
			t.Fatalf("%q => %q, want %q", input, got, want)
		}
	}
	if len(Trivia) < 60 || len(Riddles) < 40 || len(KatPrompts) < 80 || len(Challenges) < 40 {
		t.Fatalf("insufficient content: %d %d %d %d", len(Trivia), len(Riddles), len(KatPrompts), len(Challenges))
	}
	for _, bank := range [][]Question{Trivia, Riddles} {
		seen := map[string]bool{}
		for _, q := range bank {
			if seen[q.Title.AR] {
				t.Fatal("duplicate question")
			}
			seen[q.Title.AR] = true
			if q.Choice(q.Options[q.Correct].AR) != q.Correct || q.Choice(q.Options[q.Correct].EN) != q.Correct {
				t.Fatalf("ambiguous answer: %s", q.Title.AR)
			}
		}
	}
}
