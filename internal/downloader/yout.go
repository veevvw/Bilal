package downloader

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrAudioTooLong = errors.New("audio duration limit")
var youtLocks [32]sync.Mutex

// YouTube audio downloads are separate from the voice-chat queue.
func DownloadYouTubeAudio(query string) (utils.GetUrlTrack, string, error) {
	y := newYouTubeData(strings.TrimSpace(query))
	results, err := y.search()
	if err != nil || results == nil || len(results.Results) == 0 {
		if err == nil {
			err = errors.New("no results")
		}
		return utils.GetUrlTrack{}, "", err
	}
	track := results.Results[0]
	key := sha256.Sum256([]byte(track.Id))
	lock := &youtLocks[int(key[0])%len(youtLocks)]
	lock.Lock()
	defer lock.Unlock()
	if config.SongDurationLimit > 0 && track.Duration > config.SongDurationLimit {
		return track, "", ErrAudioTooLong
	}
	path, err := y.downloadWithYtDlp(track.Id, false)
	if err != nil {
		return track, "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m4a" && ext != ".mp3" {
		key := sha256.Sum256([]byte(track.Url + track.Id))
		target := filepath.Join(config.DownloadsDir, fmt.Sprintf("yout-%x.m4a", key[:12]))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err = exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", path, "-vn", "-c:a", "aac", "-b:a", "128k", target).Run(); err != nil {
			return track, "", errors.New("audio conversion failed")
		}
		path = target
	}
	file, err := os.Stat(path)
	if err != nil || file.Size() == 0 {
		return track, "", errors.New("audio file unavailable")
	}
	if config.MaxFileSize > 0 && file.Size() > config.MaxFileSize {
		return track, "", errors.New("audio size limit")
	}
	if track.Duration == 0 {
		track.Duration = utils.GetMediaDuration(path)
	}
	if config.SongDurationLimit > 0 && track.Duration > config.SongDurationLimit {
		return track, "", ErrAudioTooLong
	}
	return track, path, nil
}
