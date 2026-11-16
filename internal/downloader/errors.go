package downloader

import "errors"

var ErrYouTubeLoginRequired = errors.New("YouTube requires authentication")

// DownloadErrorMessage returns a safe user-facing reason without subprocess
// output, cookies, proxy credentials, signed media URLs or filesystem paths.
func DownloadErrorMessage(err error) string {
	if errors.Is(err, ErrYouTubeLoginRequired) {
		return "YouTube requires a valid login cookie file on this server. Ask the bot owner to update YTDLP_COOKIES_FILE."
	}
	return "Download failed: Please try again later."
}
