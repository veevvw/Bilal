package branding

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"
)

//go:embed assets/start-shahm.png
var banner []byte
var once sync.Once
var bannerPath string
var bannerErr error

func BannerFile() (string, error) {
	once.Do(func() {
		var directory string
		directory, bannerErr = os.MkdirTemp("", "shahm-branding-")
		if bannerErr != nil {
			return
		}
		bannerPath = filepath.Join(directory, "start-shahm.png")
		bannerErr = os.WriteFile(bannerPath, banner, 0600)
	})
	return bannerPath, bannerErr
}
