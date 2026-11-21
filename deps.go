package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The app relies on two free tools, installed on first launch with Windows' own
// package manager (winget): rclone (shares the folder on the network) and
// FFmpeg (puts subtitles inside a movie file).

type tool struct {
	exe      string // file name, e.g. "rclone.exe"
	wingetID string // winget package to install it from
	pkgGlob  string // where winget puts it, under %LOCALAPPDATA%\Microsoft\WinGet\Packages
}

var (
	toolRclone  = tool{"rclone.exe", "Rclone.Rclone", `Rclone.Rclone_*\*\rclone.exe`}
	toolFFmpeg  = tool{"ffmpeg.exe", "Gyan.FFmpeg", `Gyan.FFmpeg_*\*\bin\ffmpeg.exe`}
	toolFFprobe = tool{"ffprobe.exe", "Gyan.FFmpeg", `Gyan.FFmpeg_*\*\bin\ffprobe.exe`}
)

// find returns the full path of a tool, or "" when it is not installed.
func (t tool) find() string {
	// 1. Next to the app (lets people drop the .exe files in by hand)
	if exe, err := os.Executable(); err == nil {
		for _, dir := range []string{filepath.Dir(exe), filepath.Join(filepath.Dir(exe), "tools"), toolsDir()} {
			p := filepath.Join(dir, t.exe)
			if fileExists(p) {
				return p
			}
		}
	}
	// 2. On PATH
	if p, err := exec.LookPath(t.exe); err == nil {
		return p
	}
	// 3. Where winget installs things (PATH changes need a restart, so look directly)
	if t.pkgGlob == "" {
		return ""
	}
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		link := filepath.Join(local, "Microsoft", "WinGet", "Links", t.exe)
		if fileExists(link) {
			return link
		}
		matches, _ := filepath.Glob(filepath.Join(local, "Microsoft", "WinGet", "Packages", t.pkgGlob))
		if len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramW6432")} {
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "WinGet", "Packages", t.pkgGlob))
		if len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

type DepsStatus struct {
	Rclone  bool   `json:"rclone"`
	FFmpeg  bool   `json:"ffmpeg"`
	Winget  bool   `json:"winget"`
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

func depsStatus() DepsStatus {
	s := DepsStatus{
		Rclone: toolRclone.find() != "",
		FFmpeg: toolFFmpeg.find() != "" && toolFFprobe.find() != "",
	}
	_, err := exec.LookPath("winget.exe")
	s.Winget = err == nil
	s.Ready = s.Rclone && s.FFmpeg
	return s
}

var errNoWinget = fmt.Errorf("winget is not available")

// installTool runs winget for one package, reporting each output line.
func installTool(t tool, report func(string)) error {
	if t.find() != "" {
		return nil
	}
	winget, err := exec.LookPath("winget.exe")
	if err != nil {
		return errNoWinget
	}
	cmd := hiddenCmd(winget, "install", "--id", t.wingetID, "-e", "--silent",
		"--accept-source-agreements", "--accept-package-agreements", "--disable-interactivity")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Split(scanLinesOrCR)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// winget draws spinners and progress bars; only pass on real words
		if len(line) > 3 && strings.ContainsAny(line, "abcdefghijklmnopqrstuvwxyz") {
			report(line)
		}
	}
	waitErr := cmd.Wait()
	if t.find() != "" {
		return nil
	}
	if waitErr != nil {
		return fmt.Errorf("installing %s failed (%v)", t.wingetID, waitErr)
	}
	return fmt.Errorf("%s was installed but could not be found; restart the app", t.wingetID)
}

// scanLinesOrCR splits on \n or \r so progress lines arrive as they update.
func scanLinesOrCR(data []byte, atEOF bool) (int, []byte, error) {
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
