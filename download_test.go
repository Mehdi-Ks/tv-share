package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func TestDownloadRcloneAndFFmpeg(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	rc := makeZip(t, map[string]string{"rclone-v9.9.9-windows-amd64/rclone.exe": "RCLONE", "rclone-v9.9.9-windows-amd64/README.txt": "x"})
	ff := makeZip(t, map[string]string{"ffmpeg-9-essentials_build/bin/ffmpeg.exe": "FFMPEG", "ffmpeg-9-essentials_build/bin/ffprobe.exe": "FFPROBE",
		"../../evil.exe": "nope"})
	badHash := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version.txt":
			fmt.Fprint(w, "rclone v9.9.9\n")
		case "/v9.9.9/SHA256SUMS":
			fmt.Fprintf(w, "-----BEGIN PGP SIGNED MESSAGE-----\n%s  rclone-v9.9.9-windows-amd64.zip\n%s  rclone-v9.9.9-windows-arm64.zip\n", sum(rc), sum(rc))
		case "/v9.9.9/rclone-v9.9.9-windows-amd64.zip", "/v9.9.9/rclone-v9.9.9-windows-arm64.zip":
			w.Write(rc)
		case "/ff.zip":
			w.Write(ff)
		case "/ff.zip.sha256":
			if badHash {
				fmt.Fprint(w, strings.Repeat("0", 64))
			} else {
				fmt.Fprint(w, sum(ff))
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	oldR, oldZ, oldH := rcloneBase, ffmpegZipURL, ffmpegHashURL
	rcloneBase, ffmpegZipURL, ffmpegHashURL = srv.URL, srv.URL+"/ff.zip", srv.URL+"/ff.zip.sha256"
	defer func() { rcloneBase, ffmpegZipURL, ffmpegHashURL = oldR, oldZ, oldH }()

	var msgs []string
	report := func(p float64, m string) { msgs = append(msgs, m) }
	if err := downloadRclone(report); err != nil {
		t.Fatalf("rclone: %v", err)
	}
	if err := downloadFFmpeg(report); err != nil {
		t.Fatalf("ffmpeg: %v", err)
	}
	for name, body := range map[string]string{"rclone.exe": "RCLONE", "ffmpeg.exe": "FFMPEG", "ffprobe.exe": "FFPROBE"} {
		b, err := os.ReadFile(filepath.Join(toolsDir(), name))
		if err != nil || string(b) != body {
			t.Fatalf("%s: %q %v", name, b, err)
		}
	}
	// Nothing written outside the tools folder, and no leftovers.
	if fileExists(filepath.Join(filepath.Dir(filepath.Dir(toolsDir())), "evil.exe")) || fileExists(filepath.Join(toolsDir(), "evil.exe")) {
		t.Fatal("unexpected file extracted")
	}
	if m, _ := filepath.Glob(filepath.Join(toolsDir(), "*.zip")); len(m) != 0 {
		t.Fatalf("leftover downloads: %v", m)
	}
	if !strings.Contains(strings.Join(msgs, "|"), "Downloading FFmpeg") {
		t.Fatalf("no progress messages: %v", msgs)
	}
	// The app now finds them.
	if toolRclone.find() == "" || toolFFprobe.find() == "" {
		t.Fatal("downloaded tools not found")
	}

	// A file that doesn't match its checksum is refused.
	os.Remove(filepath.Join(toolsDir(), "ffmpeg.exe"))
	badHash = true
	if err := downloadFFmpeg(report); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("bad checksum accepted: %v", err)
	}
	if fileExists(filepath.Join(toolsDir(), "ffmpeg.exe")) {
		t.Fatal("unverified ffmpeg.exe was kept")
	}
}
