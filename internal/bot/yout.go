package bot

import (
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/utils"
	"errors"
	td "github.com/AshokShau/gotdbot"
	"html"
	"strings"
	"sync"
)

var youtSlots = make(chan struct{}, 3)
var youtUsers sync.Map

func youtHandler(c *td.Client, m *td.Message) error {
	query := strings.TrimSpace(Args(m))
	if query == "" {
		return protectionFailure(c, m, "يوت اسم الأغنية", "/yout song name")
	}
	key := m.SenderID()
	if key == 0 {
		key = m.ChatId
	}
	if _, busy := youtUsers.LoadOrStore(key, true); busy {
		return protectionFailure(c, m, "لديك تنزيل جارٍ.", "Your download is still running.")
	}
	defer youtUsers.Delete(key)
	select {
	case youtSlots <- struct{}{}:
		defer func() { <-youtSlots }()
	default:
		return protectionFailure(c, m, "التنزيل مشغول حالياً. حاول بعد قليل.", "Downloads are busy. Please retry shortly.")
	}
	status, err := utils.ReplyBotText(c, m, protectionText(m.ChatId, "🎵 جارٍ تنزيل الأغنية…", "🎵 Downloading your song…"), nil)
	if err != nil {
		return err
	}
	track, path, err := downloader.DownloadYouTubeAudio(query)
	if err != nil {
		message := tr(m.ChatId, downloader.DownloadErrorMessage(err))
		if errors.Is(err, downloader.ErrAudioTooLong) {
			message = protectionText(m.ChatId, "تتجاوز الأغنية المدة المسموح بها.", "This song exceeds the duration limit.")
		}
		_, e := utils.EditBotText(c, status, message, nil)
		return e
	}
	_, err = m.ReplyAudio(c, &td.InputFileLocal{Path: path}, &td.SendAudioOpts{Title: track.Title, Performer: track.Channel, Duration: track.Duration, Caption: utils.QuotedHTML("🎵 " + html.EscapeString(track.Title) + "\nShaHm"), ParseMode: "HTML"})
	if err != nil {
		_, e := utils.EditBotText(c, status, protectionText(m.ChatId, "تعذر إرسال الأغنية. تحقق من صلاحية إرسال الملفات الصوتية.", "Unable to send the song. Check audio sending permissions."), nil)
		return e
	}
	_ = status.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
	return nil
}
