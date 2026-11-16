package db

import (
	"context"
	"testing"
)

func TestLanguageDefaultsAndSupportedCache(t *testing.T) {
	db := newTestDB()
	for _, tc := range []struct{ stored, want string }{{"en", "en"}, {"ar", "ar"}, {"fr", "ar"}, {"", "ar"}} {
		db.langCache.Set(toKey(100), tc.stored)
		got, err := db.GetLanguage(100)
		if err != nil || got != tc.want {
			t.Fatalf("stored %q: got %q, %v", tc.stored, got, err)
		}
	}
	if got, err := db.GetLanguage(200); err != nil || got != "ar" {
		t.Fatalf("new chat: %q, %v", got, err)
	}
	if err := db.SetLanguage(context.Background(), 100, "fr"); err == nil {
		t.Fatal("unsupported language accepted")
	}
}
