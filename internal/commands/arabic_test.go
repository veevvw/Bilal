package commands

import "testing"

func TestArabicVocabularyRoutesWithoutSlash(t *testing.T) {
	for input, want := range map[string]string{"الاوامر": "commands", "الأوامر": "commands", "اوامر": "commands", "كت": "kattweet", "العاب": "games"} {
		if got, ok := Resolve(input); !ok || got != want {
			t.Fatalf("%q -> %q", input, got)
		}
	}
	for canonical, alias := range Arabic {
		for _, prefix := range []string{"", "/", "  "} {
			got, ok := Resolve(prefix + alias)
			if !ok || got != canonical {
				t.Errorf("%q resolves to %q, %v", alias, got, ok)
			}
		}
	}
	for _, input := range []string{"أَيْقاف", "إيقاف", "ايقاف"} {
		if got, ok := Resolve(input); !ok || got != "stop" {
			t.Errorf("alef/diacritic normalization: %q", input)
		}
	}
	for _, input := range []string{"هذه رسالة عادية", "تشغيلها لاحقا", ""} {
		if _, ok := Resolve(input); ok {
			t.Errorf("ordinary text interpreted as a command: %q", input)
		}
	}
}
func TestArabicHelpLeavesLinksIntact(t *testing.T) {
	input := "<code>/play song</code> /pause https://example.com/play https://t.me/bot/web /stop"
	want := "<code>تشغيل song</code> مؤقت https://example.com/play https://t.me/bot/web ايقاف"
	if got := Display(input, "ar"); got != want {
		t.Fatalf("%q", got)
	}
	if Display(input, "en") != input {
		t.Fatal("English commands changed")
	}
}

func TestStandaloneCommandsRejectSentences(t *testing.T) {
	for _, text := range []string{"الالعاب غير جميلة", "كت تويت", "كت مرة اخرى", "الاوامر هنا", "الغاز صعبة", "نقاطي كثيرة", "ايقاف الآن", "/العاب extra"} {
		if _, ok := Resolve(text); ok {
			t.Errorf("sentence routed: %q", text)
		}
	}
	for _, text := range []string{"كت", "  الألعاب  ", "تشغيل اسم الأغنية", "يوت اغنية", "منع_كلمة عبارة ممنوعة"} {
		if _, ok := Resolve(text); !ok {
			t.Errorf("command rejected: %q", text)
		}
	}
	if AcceptsText("games", "/games@bot extra") {
		t.Fatal("slash command accepts sentence")
	}
}
