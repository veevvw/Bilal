package i18n

import (
	"fmt"
	"regexp"
	"slices"
	"testing"
)

func TestDefaultAndEnglish(t *testing.T) {
	for _, language := range []string{"", "ar", "ar-IQ", "unsupported"} {
		if got := Text(language, "Home"); got != "الرئيسية" {
			t.Fatalf("%q: %q", language, got)
		}
	}
	if got := Text("en", "Home"); got != "Home" {
		t.Fatal(got)
	}
	if got := Normalize("en-US"); got != "en" {
		t.Fatal(got)
	}
}

func TestInterpolationPreservesUserContent(t *testing.T) {
	name := "English Song / أغنية عربية <b>Home</b>"
	got := fmt.Sprintf(Text("ar", "Welcome, %s!"), name)
	if got != "مرحبًا، "+name+"!" {
		t.Fatal(got)
	}
}

func TestCatalogPreservesFormatArguments(t *testing.T) {
	format := regexp.MustCompile(`%(?:\[[0-9]+\])?[-+#0 ]*[0-9]*(?:\.[0-9]+)?[a-zA-Z%]`)
	for source, translated := range arabic {
		if !slices.Equal(format.FindAllString(source, -1), format.FindAllString(translated, -1)) {
			t.Errorf("format arguments changed: %q", source)
		}
	}
}
