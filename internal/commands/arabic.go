package commands

import (
	"regexp"
	"strings"
	"unicode"
)

// Arabic is the single command vocabulary used by routing and help text.
var Arabic = map[string]string{
	"start": "ابدأ", "help": "مساعدة", "play": "تشغيل", "fplay": "تشغيل_فوري",
	"pause": "مؤقت", "resume": "استئناف", "stop": "ايقاف", "skip": "تخطي",
	"queue": "الانتظار", "seek": "تقديم", "loop": "تكرار", "remove": "حذف_من_الانتظار",
	"settings": "الاعدادات", "player": "المشغل", "language": "اللغة", "ping": "بنج",
	"privacy": "الخصوصية", "stats": "الاحصائيات", "reload": "تحديث", "auth": "تخويل",
	"removeAuth": "الغاء_تخويل", "auths": "المخولين", "broadcast": "اذاعة",
	"stop_broadcast": "ايقاف_الاذاعة", "av": "الاتصالات", "logger": "السجل", "sh": "تنفيذ",
	"createplaylist": "انشاء_قائمة", "deleteplaylist": "حذف_قائمة", "addtoplaylist": "اضف_للقائمة",
	"removefromplaylist": "حذف_من_القائمة", "playlistinfo": "معلومات_القائمة", "myplaylists": "قوائمي",
	"protection": "حماية", "colors": "الوان", "ban": "حظر", "unban": "الغاء_حظر",
	"mute": "كتم", "unmute": "الغاء_كتم", "warn": "تحذير", "clearwarns": "مسح_التحذيرات",
	"blockword": "منع_كلمة", "unblockword": "سماح_كلمة", "blockedwords": "الكلمات_المحظورة",
	"commands": "الاوامر", "games": "العاب", "kattweet": "كت", "daily": "هديتي", "points": "نقاطي", "leaderboard": "الترتيب", "quiz": "مسابقة", "math": "حساب", "riddle": "لغز",
	"questions": "اسئلة", "challenge": "تحديات",
	"yout": "يوت",
}

var alternatives = map[string]string{
	"وقف": "stop", "توقف": "stop", "ايقاف_مؤقت": "pause", "اكمال": "resume",
	"تشغيل_اجباري": "fplay", "بدء": "start", "اوامر": "commands", "الحماية": "protection",
	"الالعاب": "games", "توب": "leaderboard", "جوائز": "games", "الجوائز": "games",
	"الالوان": "colors", "احصائيات": "stats", "قائمة_الانتظار": "queue",
	"الغاز": "riddle", "الالغاز": "riddle", "حسابات": "math", "مسابقات": "quiz", "سؤال": "questions", "تحدي": "challenge",
}

func Normalize(word string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || r == 'ـ' {
			return -1
		}
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ':
			return 'ا'
		case 'ى':
			return 'ي'
		}
		return unicode.ToLower(r)
	}, word)
}

func Resolve(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}
	word := Normalize(strings.TrimPrefix(fields[0], "/"))
	for canonical, alias := range Arabic {
		if word == Normalize(alias) {
			return canonical, AcceptsText(canonical, text)
		}
	}
	canonical, ok := alternatives[word]
	return canonical, ok && AcceptsText(canonical, text)
}

// Only commands with actual parameters accept text after their name.
func AcceptsText(name, text string) bool {
	if len(strings.Fields(text)) <= 1 {
		return true
	}
	switch strings.ToLower(name) {
	case "blockedwords", "colors", "logger", "play", "p", "fplay", "fp", "yout", "seek", "loop", "remove", "language", "lang", "protection", "auth", "addauth", "removeauth", "rmauth", "ban", "unban", "mute", "unmute", "warn", "clearwarns", "blockword", "unblockword", "broadcast", "gcast", "sh", "start", "createplaylist", "cplist", "deleteplaylist", "addtoplaylist", "addtoplist", "removefromplaylist", "rmplist", "playlistinfo", "plistinfo":
		return true
	}
	return false
}

var slashCommand = regexp.MustCompile(`/[A-Za-z][A-Za-z0-9_]*`)

// Display changes command examples, leaving URL paths intact.
func Display(text, language string) string {
	if language != "ar" {
		return text
	}
	indices := slashCommand.FindAllStringIndex(text, -1)
	var out strings.Builder
	last := 0
	for _, span := range indices {
		if span[0] > 0 && strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.:/", rune(text[span[0]-1])) {
			continue
		}
		name := text[span[0]+1 : span[1]]
		alias, ok := Arabic[name]
		if !ok {
			continue
		}
		out.WriteString(text[last:span[0]])
		out.WriteString(alias)
		last = span[1]
	}
	out.WriteString(text[last:])
	return out.String()
}
