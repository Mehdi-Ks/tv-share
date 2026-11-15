package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Posters are a frame taken from the movie itself (no internet, nothing leaves the PC),
// cropped to a 2:3 poster shape and cached in %LOCALAPPDATA%\TV Share\posters.

// posterSlots keeps at most two FFmpeg runs at once, so old PCs stay responsive.
var posterSlots = make(chan struct{}, 2)

func posterCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "TV Share", "posters")
}

// posterKey changes whenever the file is replaced or edited.
func posterKey(p string) (string, error) {
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	// "v2": bump when the poster style changes, so old cached posters are remade.
	h := sha1.Sum([]byte(fmt.Sprintf("v2|%s|%d|%d", p, st.Size(), st.ModTime().UnixNano())))
	return hex.EncodeToString(h[:12]), nil
}

// posterFile returns the cached poster for a movie, making it first if needed.
func posterFile(video string) (string, error) {
	key, err := posterKey(video)
	if err != nil {
		return "", err
	}
	out := filepath.Join(posterCacheDir(), key+".jpg")
	if fileExists(out) {
		return out, nil
	}
	ffmpeg, ffprobe := toolFFmpeg.find(), toolFFprobe.find()
	if ffmpeg == "" || ffprobe == "" {
		return "", fmt.Errorf("FFmpeg is not installed")
	}

	posterSlots <- struct{}{}
	defer func() { <-posterSlots }()
	if fileExists(out) { // made by another call while we waited
		return out, nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}

	// Seek to ~15% of the runtime (past logos and opening credits), then let FFmpeg's
	// thumbnail filter pick the most representative of the next 48 frames: this skips
	// black or blurry frames. The whole frame is kept, centred on a 2:3 poster, over a
	// blurred and darkened copy of itself, so nobody at the edge of the shot gets cut off.
	at := 60.0
	if d := probeDuration(ffprobe, video); d > 0 {
		at = d * 0.15
	}
	tmp := out + ".tmp.jpg"
	pc := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y",
		"-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", video,
		"-filter_complex", "thumbnail=48,split[a][b];"+
			"[a]scale=360:540:force_original_aspect_ratio=increase,crop=360:540,boxblur=18:2,eq=brightness=-0.14:saturation=0.9[bg];"+
			"[b]scale=360:-2[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2",
		"-frames:v", "1", "-q:v", "4", tmp)
	lowPriority(pc)
	err = pc.Run()
	if err != nil || !fileExists(tmp) {
		os.Remove(tmp)
		return "", fmt.Errorf("could not take a frame from this movie")
	}
	if err := os.Rename(tmp, out); err != nil {
		return "", err
	}
	return out, nil
}

// posterDataURL returns the poster as a data: URL the page can show directly.
func posterDataURL(video string) (string, error) {
	p, err := posterFile(video)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b), nil
}
