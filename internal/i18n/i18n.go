package i18n

import (
	"embed"
	"encoding/json"
	"strings"
)

// Arabic is the default for new conversations and unsupported language codes.
const Default = "ar"

//go:embed ar.json
var localeFS embed.FS
var arabic = func() map[string]string {
	data, err := localeFS.ReadFile("ar.json")
	if err != nil {
		panic(err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		panic(err)
	}
	return messages
}()

func Normalize(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "en" || strings.HasPrefix(language, "en-") {
		return "en"
	}
	return Default
}

// Text translates authored templates before values are interpolated. User text,
// song titles, links, callback data and protocol values never pass through it.
func Text(language, source string) string {
	if Normalize(language) == "en" {
		return source
	}
	if translated, ok := arabic[source]; ok {
		return translated
	}
	return source
}
