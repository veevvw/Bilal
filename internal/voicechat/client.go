package voicechat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// Backend connects the existing queue to Telegram group calls. Private rooms
// continue to use the browser player.
type Backend interface {
	Enabled(int64) bool
	Play(int64, string) error
	Pause(int64) error
	Resume(int64) error
	Seek(int64, float64, bool) error
	Stop(int64) error
}

var Default Backend = &Client{}

type Client struct {
	Active bool
	URL    string
	Token  string
	HTTP   *http.Client
}

type command struct {
	Action   string  `json:"action"`
	ChatID   int64   `json:"chat_id"`
	Source   string  `json:"source,omitempty"`
	Position float64 `json:"position,omitempty"`
	Paused   bool    `json:"paused,omitempty"`
	Invite   string  `json:"invite,omitempty"`
}

type BridgeError struct{ Code string }

func (e *BridgeError) Error() string { return ErrorMessage(e.Code) }
func NeedsMembership(err error) bool {
	var e *BridgeError
	return errors.As(err, &e) && e.Code == "assistant_missing"
}
func (c *Client) Prepare(chatID int64, invite string) error {
	return c.call(command{Action: "prepare", ChatID: chatID, Invite: invite})
}

func (c *Client) AssistantID() (int64, error) {
	req, err := http.NewRequest(http.MethodGet, c.URL+"/health", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	response, err := c.HTTP.Do(req)
	if err != nil {
		return 0, errors.New("The voice assistant is unavailable. Restart the bot and try again.")
	}
	defer response.Body.Close()
	var info struct {
		AssistantID int64 `json:"assistant_id"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&info) != nil || info.AssistantID <= 0 {
		return 0, errors.New("The voice assistant is unavailable. Restart the bot and try again.")
	}
	return info.AssistantID, nil
}

func New() (*Client, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	return &Client{Active: true, URL: "http://127.0.0.1:9091", Token: hex.EncodeToString(secret), HTTP: &http.Client{Timeout: 60 * time.Second}}, nil
}

func (c *Client) Enabled(chatID int64) bool { return c.Active && chatID < 0 }
func (c *Client) Play(chatID int64, source string) error {
	return c.call(command{Action: "play", ChatID: chatID, Source: source})
}
func (c *Client) Pause(chatID int64) error  { return c.call(command{Action: "pause", ChatID: chatID}) }
func (c *Client) Resume(chatID int64) error { return c.call(command{Action: "resume", ChatID: chatID}) }
func (c *Client) Seek(chatID int64, position float64, paused bool) error {
	return c.call(command{Action: "seek", ChatID: chatID, Position: position, Paused: paused})
}
func (c *Client) Stop(chatID int64) error { return c.call(command{Action: "stop", ChatID: chatID}) }

func (c *Client) call(value command) error {
	if !c.Enabled(value.ChatID) {
		return nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/command", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("The voice assistant is unavailable. Restart the bot and try again.")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	var failure struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&failure)
	return &BridgeError{Code: failure.Code}
}

// Only stable, translated messages reach Telegram; Python errors can include
// private paths or session details and are never forwarded.
func ErrorMessage(code string) string {
	switch code {
	case "assistant_missing":
		return "The assistant could not join this chat. Give the bot invite-member permission or add the assistant manually."
	case "join_pending":
		return "Approve the assistant's join request, then try playing again."
	case "invite_permission":
		return "Give the bot permission to invite members so it can add the assistant automatically."
	case "assistant_banned":
		return "The assistant is banned from this chat. Unban it before playing music."
	case "call_missing":
		return "Start a voice chat in this group, then try /play again."
	case "permission":
		return "Allow the assistant to speak in the voice chat or promote it to an administrator."
	case "session":
		return "The assistant session is invalid. Update STRING_SESSION and restart the bot."
	case "media_missing":
		return "The audio file is unavailable. Try playing the song again."
	case "flood":
		return "Telegram temporarily limited the assistant. Wait a little and try again."
	default:
		return "Unable to stream into the voice chat. Check the assistant and try again."
	}
}
