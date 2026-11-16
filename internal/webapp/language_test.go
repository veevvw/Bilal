package webapp

import (
	"context"
	"testing"
)

type languageSettingsStub struct {
	chatSettingsStore
	chatID   int64
	language string
}

func (s *languageSettingsStub) SetLanguage(_ context.Context, chatID int64, language string) error {
	s.chatID, s.language = chatID, language
	return nil
}

func TestPersonalRoomLanguageSetting(t *testing.T) {
	for _, language := range []string{"ar", "en"} {
		store := &languageSettingsStub{}
		if err := updateChatSettings(nil, 123, 123, ClientMessage{Language: &language}, store, nil); err != nil {
			t.Fatal(err)
		}
		if store.chatID != 123 || store.language != language {
			t.Fatal("language was not persisted")
		}
	}
	invalid := "fr"
	if err := updateChatSettings(nil, 123, 123, ClientMessage{Language: &invalid}, nil, nil); err == nil {
		t.Fatal("unsupported language accepted")
	}
	english := "en"
	if err := updateChatSettings(nil, 123, 456, ClientMessage{Language: &english}, nil, nil); err == nil {
		t.Fatal("another user's room was changed")
	}
}
