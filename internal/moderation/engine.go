package moderation

import (
	"crypto/sha256"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Settings struct {
	Enabled      bool     `bson:"enabled"`
	Links        bool     `bson:"links"`
	Flood        bool     `bson:"flood"`
	Media        bool     `bson:"media"`
	Forwards     bool     `bson:"forwards"`
	Bots         bool     `bson:"bots"`
	Words        bool     `bson:"words"`
	Photos       bool     `bson:"photos"`
	Videos       bool     `bson:"videos"`
	Stickers     bool     `bson:"stickers"`
	Animations   bool     `bson:"animations"`
	Audios       bool     `bson:"audios"`
	Voices       bool     `bson:"voices"`
	Documents    bool     `bson:"documents"`
	Polls        bool     `bson:"polls"`
	Usernames    bool     `bson:"usernames"`
	Mentions     bool     `bson:"mentions"`
	Channels     bool     `bson:"channels"`
	Services     bool     `bson:"services"`
	Welcome      bool     `bson:"welcome"`
	FloodLimit   int      `bson:"flood_limit"`
	RepeatLimit  int      `bson:"repeat_limit"`
	WarnLimit    int      `bson:"warn_limit"`
	MuteMinutes  int      `bson:"mute_minutes"`
	Action       string   `bson:"action"`
	BlockedWords []string `bson:"blocked_words"`
}

func Default() Settings {
	return Settings{Links: true, Flood: true, Bots: true, Words: true, Action: "delete", FloodLimit: 5, RepeatLimit: 3, WarnLimit: 3, MuteMinutes: 10}
}

func (s Settings) WithDefaults() Settings {
	if !ValidAction(s.Action) {
		s.Action = "delete"
	}
	if s.FloodLimit < 3 || s.FloodLimit > 10 {
		s.FloodLimit = 5
	}
	if s.RepeatLimit < 2 || s.RepeatLimit > 5 {
		s.RepeatLimit = 3
	}
	if s.WarnLimit < 2 || s.WarnLimit > 5 {
		s.WarnLimit = 3
	}
	if s.MuteMinutes < 1 || s.MuteMinutes > 1440 {
		s.MuteMinutes = 10
	}
	return s
}
func ValidSetting(field string, value any) bool {
	switch field {
	case "enabled", "links", "flood", "media", "forwards", "bots", "words", "photos", "videos", "stickers", "animations", "audios", "voices", "documents", "polls", "usernames", "mentions", "channels", "services", "welcome":
		_, ok := value.(bool)
		return ok
	case "action":
		s, ok := value.(string)
		return ok && ValidAction(s)
	case "flood_limit":
		n, ok := value.(int)
		return ok && n >= 3 && n <= 10
	case "repeat_limit":
		n, ok := value.(int)
		return ok && n >= 2 && n <= 5
	case "warn_limit":
		n, ok := value.(int)
		return ok && n >= 2 && n <= 5
	case "mute_minutes":
		n, ok := value.(int)
		return ok && n >= 1 && n <= 1440
	}
	return false
}
func ValidAction(action string) bool {
	return action == "delete" || action == "warn" || action == "mute" || action == "ban"
}
func Normalize(text string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || r == 'ـ' || unicode.Is(unicode.Cf, r) {
			return -1
		}
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ':
			return 'ا'
		case 'ى':
			return 'ي'
		}
		return unicode.ToLower(r)
	}, text)), " ")
}

var linkPattern = regexp.MustCompile(`(?i)(https?://|www\.|(?:t|telegram)\.me/|tg://|(?:[\p{L}\p{N}](?:[\p{L}\p{N}-]*[\p{L}\p{N}])?\.)+(?:com|net|org|io|me|co|ru|xyz|info|app|dev|online)(?:\b|/))`)

type Message struct {
	ChatID, SenderID, ID                            int64
	Text                                            string
	HasLink, Media, Forward, Bot, Edited            bool
	Kind                                            string
	HasUsername, HasMention, ChannelSender, Service bool
}
type event struct {
	at   time.Time
	id   int64
	hash [32]byte
}
type key struct{ chat, user int64 }
type Engine struct {
	mu        sync.Mutex
	recent    map[key][]event
	lastSweep time.Time
}

func New() *Engine { return &Engine{recent: make(map[key][]event)} }

// Reason only classifies content; all Telegram permissions and actions are in the adapter.
func (e *Engine) Reason(s Settings, m Message, now time.Time) string {
	if !s.Enabled {
		return ""
	}
	s = s.WithDefaults()
	if s.Services && m.Service {
		return "services"
	}
	if s.Channels && m.ChannelSender {
		return "channels"
	}
	if s.Usernames && m.HasUsername {
		return "usernames"
	}
	if s.Mentions && m.HasMention {
		return "mentions"
	}
	locks := map[string]bool{"photos": s.Photos, "videos": s.Videos, "stickers": s.Stickers, "animations": s.Animations, "audios": s.Audios, "voices": s.Voices, "documents": s.Documents, "polls": s.Polls}
	if locks[m.Kind] {
		return m.Kind
	}
	text := Normalize(m.Text)
	if s.Links && (m.HasLink || linkPattern.MatchString(text)) {
		return "links"
	}
	if s.Forwards && m.Forward {
		return "forwards"
	}
	if s.Media && m.Media {
		return "media"
	}
	if s.Bots && m.Bot {
		return "bots"
	}
	if s.Words {
		wordText := wordTokens(text)
		for _, word := range s.BlockedWords {
			word = wordTokens(Normalize(word))
			if word != "" && strings.Contains(" "+wordText+" ", " "+word+" ") {
				return "words"
			}
		}
	}
	if !s.Flood || m.Edited {
		return ""
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := now.Add(-10 * time.Second)
	if now.Sub(e.lastSweep) > time.Minute {
		for k, v := range e.recent {
			if len(v) == 0 || v[len(v)-1].at.Before(cutoff) {
				delete(e.recent, k)
			}
		}
		e.lastSweep = now
	}
	k := key{m.ChatID, m.SenderID}
	previous := e.recent[k]
	active := previous[:0]
	hash := sha256.Sum256([]byte(text))
	repeats := 0
	for _, v := range previous {
		if !v.at.Before(cutoff) {
			if v.id == m.ID {
				return ""
			}
			active = append(active, v)
			if text != "" && v.hash == hash {
				repeats++
			}
		}
	}
	active = append(active, event{now, m.ID, hash})
	if len(active) > s.FloodLimit+1 {
		active = active[len(active)-s.FloodLimit-1:]
	}
	e.recent[k] = active
	if len(active) > s.FloodLimit || repeats >= s.RepeatLimit-1 {
		return "flood"
	}
	return ""
}

func wordTokens(text string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, text)), " ")
}
