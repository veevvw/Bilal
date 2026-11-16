package voicechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestTransportCommands(t *testing.T) {
	var commands []command
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("missing authentication")
		}
		var value command
		if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
			t.Error(err)
		}
		commands = append(commands, value)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	c := &Client{Active: true, URL: server.URL, Token: "test-secret", HTTP: server.Client()}
	if err := c.Play(-100, "/app/downloads/song.mp3"); err != nil {
		t.Fatal(err)
	}
	if err := c.Pause(-100); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(-100); err != nil {
		t.Fatal(err)
	}
	if err := c.Seek(-100, 35.5, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(-100); err != nil {
		t.Fatal(err)
	}
	if err := c.Play(100, "private-room"); err != nil {
		t.Fatal(err)
	}
	want := []command{{Action: "play", ChatID: -100, Source: "/app/downloads/song.mp3"}, {Action: "pause", ChatID: -100}, {Action: "resume", ChatID: -100}, {Action: "seek", ChatID: -100, Position: 35.5, Paused: true}, {Action: "stop", ChatID: -100}}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands: %+v", commands)
	}
}

func TestBridgeErrorsStaySafeAndActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"call_missing","detail":"private-session-must-not-escape"}`))
	}))
	defer server.Close()
	c := &Client{Active: true, URL: server.URL, HTTP: server.Client()}
	if err := c.Play(-100, "song.mp3"); err == nil || err.Error() != ErrorMessage("call_missing") {
		t.Fatal(err)
	}
	c.Active = false
	c.HTTP = nil
	if err := c.Play(-100, "song.mp3"); err != nil {
		t.Fatal("disabled bridge performed I/O")
	}
}

func TestPrepareKeepsInviteInAuthenticatedLocalRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Fatal("missing auth")
		}
		var value command
		_ = json.NewDecoder(r.Body).Decode(&value)
		if value.Action != "prepare" || value.Invite != "https://t.me/+test" || value.ChatID != -100 {
			t.Fatal("invalid prepare request")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	c := &Client{Active: true, URL: server.URL, HTTP: server.Client(), Token: "test-secret"}
	if err := c.Prepare(-100, "https://t.me/+test"); err != nil {
		t.Fatal(err)
	}
	err := &BridgeError{Code: "assistant_missing"}
	if !NeedsMembership(err) {
		t.Fatal("membership failure not classified")
	}
}
