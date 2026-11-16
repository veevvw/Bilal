package branding

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestBannerIsBundledAndUsable(t *testing.T) {
	image, err := png.DecodeConfig(bytes.NewReader(banner))
	if err != nil || image.Width < 1024 || image.Height < 600 {
		t.Fatal("missing or undersized banner", err)
	}
	path, err := BannerFile()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(file, banner) {
		t.Fatal("banner upload source differs from bundled asset")
	}
}
