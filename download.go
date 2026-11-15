package main

// When winget isn't available (or fails), TV Share downloads the two helpers itself,
// from the same official sources winget uses, and checks each file against the
// SHA-256 checksum those sites publish. Only the needed .exe files are unpacked,
// into %LOCALAPPDATA%\TV Share\tools.

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	rcloneBase     = "https://downloads.rclone.org"
	ffmpegZipURL   = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	ffmpegHashURL  = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip.sha256"
	downloadClient = &http.Client{Timeout: 30 * time.Minute}
)

// toolsDir is where downloaded helpers live.
func toolsDir() string {
	dir, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "TV Share", "tools")
}

func fetchText(url string) (string, error) {
	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s answered %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(b), err
}

// downloadFile saves url to dst, reporting progress (0..1), and returns its SHA-256.
func downloadFile(url, dst string, progress func(float64)) (string, error) {
	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("couldn't download (are you online?)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("the download site answered %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	total := float64(resp.ContentLength)
	var done float64
	buf := make([]byte, 256<<10)
	last := time.Time{}
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return "", err
			}
			h.Write(buf[:n])
			done += float64(n)
			if total > 0 && time.Since(last) > 250*time.Millisecond {
				progress(done / total)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", fmt.Errorf("the download was interrupted, try again")
		}
	}
	progress(1)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// unzipFiles copies the zip entries whose base name is in want into dir.
// Entry paths are never used for the destination, so a bad archive can't write elsewhere.
func unzipFiles(zipPath, dir string, want ...string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	need := map[string]bool{}
	for _, w := range want {
		need[strings.ToLower(w)] = true
	}
	for _, f := range zr.File {
		base := strings.ToLower(filepath.Base(strings.ReplaceAll(f.Name, "\\", "/")))
		if !need[base] || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		tmp := filepath.Join(dir, base+".part")
		out, err := os.Create(tmp)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, io.LimitReader(rc, 1<<30))
		rc.Close()
		out.Close()
		if err != nil {
			os.Remove(tmp)
			return err
		}
		if err := os.Rename(tmp, filepath.Join(dir, base)); err != nil {
			return err
		}
		delete(need, base)
	}
	if len(need) > 0 {
		return fmt.Errorf("the download didn't contain the expected files")
	}
	return nil
}

// downloadRclone fetches the current rclone release for this PC.
func downloadRclone(report func(pct float64, msg string)) error {
	report(0, "Checking the latest rclone…")
	v, err := fetchText(rcloneBase + "/version.txt") // "rclone v1.76.0"
	if err != nil {
		return fmt.Errorf("couldn't reach rclone.org (are you online?)")
	}
	f := strings.Fields(v)
	if len(f) < 2 || !strings.HasPrefix(f[1], "v") {
		return fmt.Errorf("rclone.org gave an unexpected answer")
	}
	ver := f[1]
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	name := fmt.Sprintf("rclone-%s-windows-%s.zip", ver, arch)
	sums, err := fetchText(fmt.Sprintf("%s/%s/SHA256SUMS", rcloneBase, ver))
	if err != nil {
		return fmt.Errorf("couldn't get rclone's checksums")
	}
	want := ""
	for _, line := range strings.Split(sums, "\n") {
		if p := strings.Fields(line); len(p) == 2 && p[1] == name {
			want = strings.ToLower(p[0])
		}
	}
	if want == "" {
		return fmt.Errorf("rclone's checksum for this PC is missing")
	}
	return fetchVerifyUnzip(fmt.Sprintf("%s/%s/%s", rcloneBase, ver, name), want, "rclone",
		report, "rclone.exe")
}

// downloadFFmpeg fetches the FFmpeg "essentials" build (the one winget's Gyan.FFmpeg also installs).
func downloadFFmpeg(report func(pct float64, msg string)) error {
	report(0, "Checking the latest FFmpeg…")
	h, err := fetchText(ffmpegHashURL)
	if err != nil {
		return fmt.Errorf("couldn't reach gyan.dev (are you online?)")
	}
	want := strings.ToLower(strings.TrimSpace(strings.Fields(h + " x")[0]))
	if len(want) != 64 {
		return fmt.Errorf("gyan.dev gave an unexpected checksum")
	}
	return fetchVerifyUnzip(ffmpegZipURL, want, "FFmpeg", report, "ffmpeg.exe", "ffprobe.exe")
}

func fetchVerifyUnzip(url, wantHash, label string, report func(float64, string), files ...string) error {
	dir := toolsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	zipPath := filepath.Join(dir, label+".download.zip")
	defer os.Remove(zipPath)
	got, err := downloadFile(url, zipPath, func(p float64) {
		report(p, fmt.Sprintf("Downloading %s… %d%%", label, int(p*100)))
	})
	if err != nil {
		return fmt.Errorf("%s: %v", label, err)
	}
	if got != wantHash {
		return fmt.Errorf("%s: the downloaded file doesn't match its official checksum, so it wasn't used. Try again later", label)
	}
	report(1, "Unpacking "+label+"…")
	return unzipFiles(zipPath, dir, files...)
}
