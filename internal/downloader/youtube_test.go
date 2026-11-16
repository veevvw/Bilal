package downloader

import (
	"ashokshau/tg-web/internal/config"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCookieFileCopiedWithoutChangingOriginal(t *testing.T) {
	source := filepath.Join(t.TempDir(), "youtube.txt")
	want := "# Netscape HTTP Cookie File\n.youtube.com\tTRUE\t/\tTRUE\t9999999999\tSID\ttest-value\n"
	if err := os.WriteFile(source, []byte(want), 0400); err != nil {
		t.Fatal(err)
	}
	copy, err := copyCookies(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(copy) })
	got, err := os.ReadFile(copy)
	if err != nil || string(got) != want || copy == source {
		t.Fatal("cookie copy failed")
	}
	if err := os.WriteFile(copy, []byte("rewritten by downloader"), 0600); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(source)
	if err != nil || string(original) != want {
		t.Fatal("original cookies changed")
	}
}

func TestYouTubeAuthenticationErrors(t *testing.T) {
	for _, message := range []string{"Sign in to confirm you're not a bot", "Sign in to confirm you’re not a bot", "The provided YouTube cookies are no longer valid", "LOGIN REQUIRED"} {
		if !youtubeNeedsLogin(message) {
			t.Fatalf("not recognized: %s", message)
		}
	}
	if youtubeNeedsLogin("HTTP Error 403: Forbidden") {
		t.Fatal("generic HTTP failure misclassified as authentication")
	}
	if DownloadErrorMessage(errors.Join(errors.New("private path"), ErrYouTubeLoginRequired)) == DownloadErrorMessage(errors.New("other")) {
		t.Fatal("authentication reason was hidden")
	}
}

func TestLocalCookiesAndProxyCanBeUsedTogether(t *testing.T) {
	previousCookie, previousProxy := config.YtdlpCookiesFile, config.Proxy
	t.Cleanup(func() { config.YtdlpCookiesFile = previousCookie; config.Proxy = previousProxy })
	config.YtdlpCookiesFile = "/run/secrets/youtube.txt"
	config.Proxy = "http://proxy.invalid:8080"
	params, cookie := newYouTubeData("").buildYtdlpParams("jNQXAC9IVRw", false)
	if cookie != config.YtdlpCookiesFile || !slices.Contains(params, "--cookies") || !slices.Contains(params, "--proxy") {
		t.Fatal("cookies and proxy were not both used")
	}
}
