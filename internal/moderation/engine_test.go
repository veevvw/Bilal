package moderation

import (
	"fmt"
	"testing"
	"time"
)

func TestFiltersAndDisabledProtection(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, test := range []struct {
		name string
		s    Settings
		m    Message
		want string
	}{
		{"disabled", Default(), Message{Text: "https://example.com"}, ""},
		{"url", Settings{Enabled: true, Links: true}, Message{Text: "زوروا example.com"}, "links"},
		{"hidden url", Settings{Enabled: true, Links: true}, Message{Text: "اضغط هنا", HasLink: true}, "links"},
		{"caption url", Settings{Enabled: true, Links: true}, Message{Text: "https://youtu.be/test"}, "links"},
		{"media", Settings{Enabled: true, Media: true}, Message{Media: true}, "media"},
		{"forward", Settings{Enabled: true, Forwards: true}, Message{Forward: true}, "forwards"},
		{"bot", Settings{Enabled: true, Bots: true}, Message{Bot: true}, "bots"},
		{"arabic normalization", Settings{Enabled: true, Words: true, BlockedWords: []string{"إساءة"}}, Message{Text: "هذه إِسَـاءة!"}, "words"},
		{"whole word", Settings{Enabled: true, Words: true, BlockedWords: []string{"حظر"}}, Message{Text: "محظور"}, ""},
		{"plain song search", Settings{Enabled: true, Links: true}, Message{Text: "تشغيل اغنية جميلة"}, ""},
		{"selective photos", Settings{Enabled: true, Photos: true}, Message{Media: true, Kind: "photos"}, "photos"},
		{"photo lock allows video", Settings{Enabled: true, Photos: true}, Message{Media: true, Kind: "videos"}, ""},
		{"selective sticker", Settings{Enabled: true, Stickers: true}, Message{Media: true, Kind: "stickers"}, "stickers"},
		{"hidden mention", Settings{Enabled: true, Mentions: true}, Message{HasMention: true}, "mentions"},
		{"username", Settings{Enabled: true, Usernames: true}, Message{HasUsername: true}, "usernames"},
		{"channel sender", Settings{Enabled: true, Channels: true}, Message{ChannelSender: true}, "channels"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := New().Reason(test.s, test.m, now); got != test.want {
				t.Fatalf("got %q want %q", got, test.want)
			}
		})
	}
	if Default().Enabled || Default().Action != "delete" {
		t.Fatal("protection must require activation and default to delete only")
	}
}

func TestConfigurableLimitsAndValidatedSettings(t *testing.T) {
	e := New()
	s := Settings{Enabled: true, Flood: true, FloodLimit: 3, RepeatLimit: 5}
	now := time.Unix(10000, 0)
	for i := int64(1); i <= 4; i++ {
		got := e.Reason(s, Message{ChatID: -1, SenderID: 1, ID: i, Text: fmt.Sprint(i)}, now)
		if (got == "flood") != (i == 4) {
			t.Fatalf("limit ignored at %d", i)
		}
	}
	for _, test := range []struct {
		field string
		value any
		want  bool
	}{{"mute_minutes", 1440, true}, {"mute_minutes", 0, false}, {"flood_limit", 1000, false}, {"links", true, true}, {"links", "true", false}, {"$set", true, false}, {"warn_limit", 3, true}} {
		if ValidSetting(test.field, test.value) != test.want {
			t.Errorf("invalid setting validation: %s", test.field)
		}
	}
	defaults := (Settings{}).WithDefaults()
	if defaults.WarnLimit != 3 || defaults.MuteMinutes != 10 || defaults.FloodLimit != 5 || defaults.Action != "delete" {
		t.Fatal("old stored settings lost defaults")
	}
}
func TestFloodIsolationDeduplicationAndExpiry(t *testing.T) {
	e := New()
	s := Settings{Enabled: true, Flood: true}
	now := time.Unix(1000, 0)
	for i := int64(1); i <= 6; i++ {
		m := Message{ChatID: -1, SenderID: 1, ID: i, Text: fmt.Sprint(i)}
		got := e.Reason(s, m, now)
		if (got == "flood") != (i == 6) {
			t.Fatalf("message %d: %q", i, got)
		}
	}
	for _, m := range []Message{{ChatID: -1, SenderID: 2, ID: 7, Text: "hello"}, {ChatID: -2, SenderID: 1, ID: 7, Text: "hello"}, {ChatID: -1, SenderID: 1, ID: 7, Edited: true}} {
		if got := e.Reason(s, m, now); got != "" {
			t.Fatalf("unexpected cross-user, cross-chat or edit flood: %q", got)
		}
	}
	if got := e.Reason(s, Message{ChatID: -1, SenderID: 1, ID: 8, Text: "new"}, now.Add(11*time.Second)); got != "" {
		t.Fatal("expired messages counted")
	}
	e = New()
	m := Message{ChatID: -1, SenderID: 1, ID: 1, Text: "repeat"}
	for range 5 {
		if e.Reason(s, m, now) != "" {
			t.Fatal("duplicate update counted")
		}
	}
	m.ID = 2
	if e.Reason(s, m, now) != "" {
		t.Fatal("second repeat blocked too soon")
	}
	m.ID = 3
	if e.Reason(s, m, now) != "flood" {
		t.Fatal("third identical message not blocked")
	}
}
