package games

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Text struct{ AR, EN string }

const AnswerReward = 1

func (t Text) In(lang string) string {
	if lang == "en" {
		return t.EN
	}
	return t.AR
}

type Question struct {
	Title   Text
	Options [4]Text
	Correct int
}
type Round struct {
	ID       string
	ChatID   int64
	Question Question
	Expires  time.Time
	Winner   int64
	Awarded  bool
	attempts map[int64]bool
	messages map[int64]bool
	remaps   map[int64]int64
}
type Engine struct {
	mu     sync.Mutex
	rounds map[int64]*Round
}

var (
	ErrActive    = errors.New("round active")
	ErrExpired   = errors.New("round expired")
	ErrAttempted = errors.New("already attempted")
	ErrWrong     = errors.New("wrong answer")
	ErrWon       = errors.New("already won")
)

func New() *Engine {
	return &Engine{rounds: map[int64]*Round{}}
}
func Random(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic("secure randomness unavailable")
	}
	return int(v.Int64())
}
func clone(r *Round) Round {
	copy := *r
	copy.attempts = nil
	copy.messages = nil
	copy.remaps = nil
	return copy
}
func (e *Engine) Start(chatID int64, kind string, now time.Time) (Round, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.rounds[chatID]; r != nil && !r.Awarded && now.Before(r.Expires) {
		return clone(r), ErrActive
	}
	for id, r := range e.rounds {
		if now.Sub(r.Expires) > time.Hour {
			delete(e.rounds, id)
		}
	}
	q := NewQuestion(kind)
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Round{}, err
	}
	r := &Round{ID: hex.EncodeToString(raw[:]), ChatID: chatID, Question: q, Expires: now.Add(time.Minute), attempts: map[int64]bool{}, messages: map[int64]bool{}, remaps: map[int64]int64{}}
	e.rounds[chatID] = r
	return clone(r), nil
}

// Reserve a single winner. Failed storage can be retried by that winner only.
func (e *Engine) Answer(chatID int64, id string, userID int64, choice int, now time.Time) (Round, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.rounds[chatID]
	if r == nil || r.ID != id || !now.Before(r.Expires) {
		return Round{}, ErrExpired
	}
	if r.Winner != 0 {
		if r.Winner == userID && !r.Awarded {
			return clone(r), nil
		}
		return clone(r), ErrWon
	}
	if r.attempts[userID] {
		return clone(r), ErrAttempted
	}
	r.attempts[userID] = true
	if choice < 0 || choice >= 4 {
		return clone(r), ErrWrong
	}
	if choice != r.Question.Correct {
		return clone(r), ErrWrong
	}
	r.Winner = userID
	return clone(r), nil
}
func (e *Engine) Finish(chatID int64, id string, userID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.rounds[chatID]; r != nil && r.ID == id && r.Winner == userID {
		r.Awarded = true
	}
}
func (e *Engine) Stop(chatID int64) { e.mu.Lock(); defer e.mu.Unlock(); delete(e.rounds, chatID) }

func (e *Engine) BindMessage(chatID int64, id string, messageID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.rounds[chatID]; r != nil && r.ID == id {
		if mapped := r.remaps[messageID]; mapped != 0 {
			messageID = mapped
		}
		r.messages[messageID] = true
	}
}
func (e *Engine) MoveMessage(chatID, oldID, newID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.rounds[chatID]; r != nil {
		if r.messages[oldID] {
			delete(r.messages, oldID)
			r.messages[newID] = true
		}
		if len(r.remaps) < 256 {
			r.remaps[oldID] = newID
		}
	}
}
func (e *Engine) UnbindMessage(chatID, messageID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r := e.rounds[chatID]; r != nil {
		delete(r.messages, messageID)
	}
}
func (e *Engine) ReplyRound(chatID, messageID int64) (Round, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.rounds[chatID]
	if r == nil || !r.messages[messageID] {
		return Round{}, false
	}
	return clone(r), true
}
func NormalizeAnswer(text string) string {
	numeric := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		if r >= '٠' && r <= '٩' {
			return '0' + r - '٠'
		}
		if r >= '۰' && r <= '۹' {
			return '0' + r - '۰'
		}
		if r == '−' {
			return '-'
		}
		return r
	}, text)
	if n, err := strconv.ParseInt(numeric, 10, 64); err == nil {
		return strconv.FormatInt(n, 10)
	}
	text = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Cf, r) || r == 'ـ' {
			return -1
		}
		if r >= '٠' && r <= '٩' {
			return '0' + r - '٠'
		}
		if r >= '۰' && r <= '۹' {
			return '0' + r - '۰'
		}
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ':
			return 'ا'
		case 'ى':
			return 'ي'
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return ' '
		}
		return unicode.ToLower(r)
	}, text)), " ")
	for _, prefix := range []string{"ال", "the ", "an ", "a "} {
		text = strings.TrimPrefix(text, prefix)
	}
	return text
}
func (q Question) Choice(text string) int {
	answer := NormalizeAnswer(text)
	if answer == "" {
		return -1
	}
	for i, option := range q.Options {
		if answer == NormalizeAnswer(option.AR) || answer == NormalizeAnswer(option.EN) {
			return i
		}
	}
	return -1
}
func (e *Engine) Prompt(chatID, userID int64, now time.Time) (Text, error) {
	return e.PromptKind(chatID, userID, "kat", now)
}
func (e *Engine) PromptKind(chatID, userID int64, kind string, now time.Time) (Text, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	bank := KatPrompts
	if kind == "challenge" {
		bank = Challenges
	}
	return bank[Random(len(bank))], nil
}
func NewQuestion(kind string) Question {
	var q Question
	if kind == "math" {
		a, b := Random(40)+1, Random(20)+1
		n, op := a+b, "+"
		switch Random(3) {
		case 1:
			if a < b {
				a, b = b, a
			}
			n = a - b
			op = "−"
		case 2:
			a, b = Random(12)+1, Random(12)+1
			n = a * b
			op = "×"
		}
		q = Question{Title: Text{fmt.Sprintf("كم يساوي %d %s %d؟", a, op, b), fmt.Sprintf("What is %d %s %d?", a, op, b)}}
		for i, v := range []int{n, n + 2, n - 1, n + 5} {
			q.Options[i] = Text{fmt.Sprint(v), fmt.Sprint(v)}
		}
	} else if kind == "riddle" {
		q = Riddles[Random(len(Riddles))]
	} else {
		q = Trivia[Random(len(Trivia))]
	}
	for i := 3; i > 0; i-- {
		j := Random(i + 1)
		q.Options[i], q.Options[j] = q.Options[j], q.Options[i]
		if q.Correct == i {
			q.Correct = j
		} else if q.Correct == j {
			q.Correct = i
		}
	}
	return q
}

var Trivia = []Question{
	{Text{"ما عاصمة العراق؟", "What is the capital of Iraq?"}, [4]Text{{"بغداد", "Baghdad"}, {"البصرة", "Basra"}, {"أربيل", "Erbil"}, {"الموصل", "Mosul"}}, 0},
	{Text{"كم عدد أيام السنة الكبيسة؟", "How many days are in a leap year?"}, [4]Text{{"366", "366"}, {"365", "365"}, {"364", "364"}, {"360", "360"}}, 0},
	{Text{"أي كوكب يعرف بالكوكب الأحمر؟", "Which planet is the Red Planet?"}, [4]Text{{"المريخ", "Mars"}, {"الزهرة", "Venus"}, {"المشتري", "Jupiter"}, {"عطارد", "Mercury"}}, 0},
	{Text{"ما أكبر محيط على الأرض؟", "What is Earth's largest ocean?"}, [4]Text{{"الهادئ", "Pacific"}, {"الأطلسي", "Atlantic"}, {"الهندي", "Indian"}, {"المتجمد الشمالي", "Arctic"}}, 0},
	{Text{"كم ضلعاً للمثلث؟", "How many sides does a triangle have?"}, [4]Text{{"3", "3"}, {"4", "4"}, {"5", "5"}, {"6", "6"}}, 0},
	{Text{"ما الغاز الذي تستخدمه النباتات للبناء الضوئي؟", "Which gas do plants use for photosynthesis?"}, [4]Text{{"ثاني أكسيد الكربون", "Carbon dioxide"}, {"الهيليوم", "Helium"}, {"الهيدروجين", "Hydrogen"}, {"النيون", "Neon"}}, 0},
}
var Riddles = []Question{
	{Text{"له أسنان ولا يعض، ما هو؟", "It has teeth but cannot bite. What is it?"}, [4]Text{{"المشط", "A comb"}, {"القلم", "A pen"}, {"الباب", "A door"}, {"الكوب", "A cup"}}, 0},
	{Text{"كلما أخذت منه كبر، ما هو؟", "The more you take away, the larger it gets. What is it?"}, [4]Text{{"الحفرة", "A hole"}, {"الكتاب", "A book"}, {"الشجرة", "A tree"}, {"الكأس", "A glass"}}, 0},
	{Text{"يمشي بلا أرجل ويخبرك بالوقت، ما هو؟", "It runs without legs and tells time. What is it?"}, [4]Text{{"الساعة", "A clock"}, {"المرآة", "A mirror"}, {"القمر", "The moon"}, {"الصندوق", "A box"}}, 0},
	{Text{"له أوراق وليس نباتاً، وتقرأه، ما هو؟", "It has leaves, is not a plant, and you read it. What is it?"}, [4]Text{{"الكتاب", "A book"}, {"القلم", "A pen"}, {"الكرسي", "A chair"}, {"الحجر", "A stone"}}, 0},
}
var KatPrompts = []Text{
	{"كت: ما أجمل شيء حدث لك هذا الأسبوع؟", "Kat: What was the nicest part of your week?"},
	{"كت: صف مزاجك الآن بثلاث كلمات.", "Kat: Describe your mood in three words."},
	{"كت: لو سافرت إلى أي مدينة، ماذا ستختار؟", "Kat: Which city would you like to visit?"},
	{"كت: ما الأغنية التي تحسن مزاجك؟", "Kat: Which song improves your mood?"},
	{"كت: اذكر صفة جميلة في أحد أعضاء المجموعة دون إحراجه.", "Kat: Say something kind about a group member without embarrassing them."},
	{"كت: لو تعلمت مهارة جديدة اليوم، فما هي؟", "Kat: Which new skill would you learn today?"},
	{"كت: اختر قوة خيالية ليوم واحد، وما أول شيء ستفعله؟", "Kat: Choose a superpower for a day. What would you do first?"},
	{"كت: احك موقفاً مضحكاً دون كشف معلومات خاصة.", "Kat: Tell a funny story without sharing private information."},
	{"كت: ما أفضل نصيحة تلقيتها؟", "Kat: What is the best advice you have received?"},
	{"كت: تحدٍ خفيف — اكتب جملة مفيدة دون استخدام حرف الألف.", "Kat: A light challenge — write a sentence without the letter A."},
	{"كت: تحدٍ خفيف — اذكر ثلاثة أشياء لونها أزرق.", "Kat: A light challenge — name three blue things."},
	{"كت: ما طعامك المفضل؟", "Kat: What is your favorite food?"},
	{"كت: اكتب هدفاً صغيراً تستطيع إنجازه هذا الأسبوع.", "Kat: Name a small goal you can complete this week."},
	{"كت: تحدٍ — اصنع قصة قصيرة من خمس كلمات.", "Kat: A challenge — write a five-word story."},
	{"كت: ما الكتاب أو الفيلم الذي تقترحه على الأصدقاء؟", "Kat: Which book or film would you suggest to friends?"},
	{"كت: اذكر شيئاً تتمنى أن تجربه لأول مرة.", "Kat: Name something you would like to try for the first time."},
	{"كت: هل تفضل البحر أم الجبل؟ ولماذا؟", "Kat: Do you prefer the sea or the mountains, and why?"},
	{"كت: ما العادة التي تود الحفاظ عليها؟", "Kat: Which habit would you like to keep?"},
	{"كت: لو أن يومك عنوان كتاب، ماذا سيكون؟", "Kat: If your day were a book title, what would it be?"},
	{"كت: تحدٍ — اكتب رسالة تشجيع قصيرة للمجموعة.", "Kat: A challenge — write a short encouraging message for the group."},
}
