package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/voicechat"
	"errors"
	"reflect"
	"testing"
	"time"
)

type callRecorder struct {
	actions    []string
	pauseError error
	playError  error
	seekPaused bool
}

func (r *callRecorder) Enabled(id int64) bool { return id < 0 }
func (r *callRecorder) Play(id int64, source string) error {
	r.actions = append(r.actions, "play")
	return r.playError
}
func (r *callRecorder) Pause(id int64) error {
	r.actions = append(r.actions, "pause")
	return r.pauseError
}
func (r *callRecorder) Resume(id int64) error { r.actions = append(r.actions, "resume"); return nil }
func (r *callRecorder) Seek(id int64, pos float64, paused bool) error {
	r.actions = append(r.actions, "seek")
	r.seekPaused = paused
	return nil
}
func (r *callRecorder) Stop(id int64) error { r.actions = append(r.actions, "stop"); return nil }

func TestNativeCallControlsAndBrowserDisconnect(t *testing.T) {
	t.Cleanup(db.SetTestDatabase(cache.NewCache[*db.Chats](time.Minute)))
	const chat int64 = -99173181
	previous := voicechat.Default
	recorder := &callRecorder{}
	voicechat.Default = recorder
	manager := &WebAppPlayerManager{rooms: make(map[int64]*RoomState)}
	t.Cleanup(func() { manager.Stop(nil, chat); voicechat.Default = previous; cache.ChatCache.ClearChat(chat) })
	track := &utils.PlayerCache{TrackID: "voice-song", FilePath: "downloads/song.mp3", Duration: 120}
	cache.ChatCache.AddSong(chat, track)
	if err := manager.playTrack(nil, chat, track); err != nil {
		t.Fatal(err)
	}
	manager.CheckListenersCount(nil, chat)
	room := manager.getOrCreate(nil, chat)
	if room.graceTimer != nil {
		t.Fatal("native call depends on browser listeners")
	}
	recorder.pauseError = errors.New("transport failure")
	if _, err := manager.Pause(nil, chat); err == nil || room.Status != "playing" {
		t.Fatal("failed pause changed playback")
	}
	recorder.pauseError = nil
	if _, err := manager.Pause(nil, chat); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SeekRoom(nil, chat, 30); err != nil || !recorder.seekPaused {
		t.Fatal("seek lost paused state")
	}
	if _, err := manager.Resume(nil, chat); err != nil {
		t.Fatal(err)
	}
	if err := manager.Stop(nil, chat); err != nil {
		t.Fatal(err)
	}
	if cache.ChatCache.IsActive(chat) || room.Status != "stopped" {
		t.Fatal("stop did not clear playback")
	}
	if !reflect.DeepEqual(recorder.actions, []string{"play", "pause", "pause", "seek", "resume", "stop"}) {
		t.Fatal(recorder.actions)
	}
}

func TestFailedNativeJoinDoesNotReportPlaying(t *testing.T) {
	t.Cleanup(db.SetTestDatabase(cache.NewCache[*db.Chats](time.Minute)))
	const chat int64 = -99173182
	previous := voicechat.Default
	voicechat.Default = &callRecorder{playError: errors.New("missing group call")}
	manager := &WebAppPlayerManager{rooms: make(map[int64]*RoomState)}
	t.Cleanup(func() { voicechat.Default = previous; cache.ChatCache.ClearChat(chat) })
	track := &utils.PlayerCache{TrackID: "voice-song", FilePath: "song.mp3"}
	cache.ChatCache.AddSong(chat, track)
	if err := manager.playTrack(nil, chat, track); err == nil {
		t.Fatal("failed join reported success")
	}
	if cache.ChatCache.IsActive(chat) || manager.getOrCreate(nil, chat).Status != "stopped" {
		t.Fatal("failed join left a phantom playing track")
	}
}
