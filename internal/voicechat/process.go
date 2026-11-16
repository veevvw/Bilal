package voicechat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// Start launches the local, authenticated assistant bridge. Sessions travel in
// the child's environment, never command-line arguments or browser responses.
func (c *Client) Start(python, script string) (func(), error) {
	cmd := exec.Command(python, script)
	cmd.Env = append(os.Environ(), "VOICE_BRIDGE_TOKEN="+c.Token, "PYTHONUNBUFFERED=1")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, errors.New("cannot start the voice assistant; install voicechat/requirements.txt and check VOICE_CHAT_PYTHON")
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	stop := func() {
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return nil, errors.New("the voice assistant could not sign in; check STRING_SESSION, API_ID and API_HASH")
		case <-deadline.C:
			stop()
			return nil, errors.New("the voice assistant did not become ready within 90 seconds")
		case <-tick.C:
			if c.ready() {
				return stop, nil
			}
		}
	}
}

func (c *Client) ready() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.URL+"/health", nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	response, err := c.HTTP.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	var status struct {
		Ready bool `json:"ready"`
	}
	return response.StatusCode == http.StatusOK && json.NewDecoder(response.Body).Decode(&status) == nil && status.Ready
}
