package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const appVersion = "1.0.1"

// App holds everything the window can call. Each exported method is available
// in the page as window.go.main.App.<Method>().
type App struct {
	ctx      context.Context
	mu       sync.Mutex
	settings Settings
	share    *Share
	prep     *Preparer
	extra    []string // movie folders dropped in from outside the shared folder
	net      NetworkInfo
	netOnce  sync.Once
}

func NewApp() *App {
	return &App{settings: loadSettings(), share: &Share{}, prep: &Preparer{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	initJob()
	a.share.onExit = func(msg string) {
		runtime.EventsEmit(a.ctx, "share:stopped", msg)
	}
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		runtime.EventsEmit(a.ctx, "drop", a.handleDrop(paths))
	})
	go func() {
		a.refreshNetwork()
		runtime.EventsEmit(a.ctx, "network", a.net)
	}()
	if a.settings.AutoStartShare && depsStatus().Ready {
		go func() {
			if err := a.share.Start(a.Settings()); err != nil {
				runtime.EventsEmit(a.ctx, "share:stopped", err.Error())
			} else {
				runtime.EventsEmit(a.ctx, "share:started", nil)
			}
		}()
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.prep.Cancel()
	a.share.Stop()
}

func (a *App) onSecondInstance(data options.SecondInstanceData) {
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
}

func (a *App) refreshNetwork() {
	n := networkInfo()
	a.mu.Lock()
	a.net = n
	a.mu.Unlock()
}

// ---- State --------------------------------------------------------------------

type State struct {
	Version  string      `json:"version"`
	Deps     DepsStatus  `json:"deps"`
	Sharing  bool        `json:"sharing"`
	Settings Settings    `json:"settings"`
	Network  NetworkInfo `json:"network"`
	Busy     bool        `json:"busy"`
}

func (a *App) Settings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

func (a *App) GetState() State {
	a.mu.Lock()
	n := a.net
	a.mu.Unlock()
	if n.IP == "" {
		n.IP = localIP()
	}
	return State{
		Version:  appVersion,
		Deps:     depsStatus(),
		Sharing:  a.share.Running(),
		Settings: a.Settings(),
		Network:  n,
		Busy:     a.prep.Busy(),
	}
}

func (a *App) RefreshNetwork() NetworkInfo {
	a.refreshNetwork()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.net
}

// ---- Setup ----------------------------------------------------------------------

// InstallDeps installs rclone and FFmpeg with winget, reporting progress as "setup" events.
func (a *App) InstallDeps() (DepsStatus, error) {
	report := func(tool string) func(string) {
		return func(line string) {
			runtime.EventsEmit(a.ctx, "setup", tool+": "+line)
		}
	}
	if err := installTool(toolRclone, report("rclone")); err != nil {
		return depsStatus(), err
	}
	if err := installTool(toolFFmpeg, report("FFmpeg")); err != nil {
		return depsStatus(), err
	}
	return depsStatus(), nil
}

// ---- Sharing --------------------------------------------------------------------

func (a *App) StartShare() error {
	err := a.share.Start(a.Settings())
	go func() {
		a.refreshNetwork()
		runtime.EventsEmit(a.ctx, "network", a.net)
	}()
	return err
}

func (a *App) StopShare() {
	a.share.Stop()
}

// ---- Settings ---------------------------------------------------------------------

func (a *App) ChooseFolder() (string, error) {
	cur := a.Settings().Folder
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose the folder your TV will see",
		DefaultDirectory: cur,
	})
	if err != nil || dir == "" {
		return cur, err
	}
	return dir, a.updateSettings(func(s *Settings) { s.Folder = dir })
}

func (a *App) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("the name can't be empty")
	}
	if len(name) > 40 {
		name = name[:40]
	}
	return a.updateSettings(func(s *Settings) { s.Name = name })
}

func (a *App) SetOption(key string, value bool) error {
	switch key {
	case "autoStartShare":
		return a.updateSettings(func(s *Settings) { s.AutoStartShare = value })
	case "recycleOriginal":
		return a.updateSettings(func(s *Settings) { s.RecycleOriginal = value })
	case "launchAtLogin":
		if err := setLaunchAtLogin(value); err != nil {
			return err
		}
		return a.updateSettings(func(s *Settings) { s.LaunchAtLogin = value })
	}
	return fmt.Errorf("unknown setting %q", key)
}

func (a *App) SetSubLang(code string) error {
	return a.updateSettings(func(s *Settings) { s.SubLang = code })
}

// Languages lists the subtitle languages the app recognises, for the settings menu.
func (a *App) Languages() [][2]string {
	seen := map[string]bool{}
	var out [][2]string
	for _, l := range langTokens {
		if !seen[l.code] {
			seen[l.code] = true
			out = append(out, [2]string{l.code, l.name})
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j][1] < out[j-1][1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (a *App) updateSettings(f func(*Settings)) error {
	a.mu.Lock()
	f(&a.settings)
	s := a.settings
	a.mu.Unlock()
	return saveSettings(s)
}

// ---- Movies -----------------------------------------------------------------------

func (a *App) ListMovies() []Movie {
	s := a.Settings()
	movies := scanMovies(s.Folder, s.SubLang)

	a.mu.Lock()
	extra := append([]string(nil), a.extra...)
	a.mu.Unlock()
	if len(extra) > 0 {
		seen := map[string]bool{}
		for _, m := range movies {
			seen[strings.ToLower(m.Path)] = true
		}
		for _, dir := range extra {
			for _, m := range scanMovies(dir, s.SubLang) {
				if !seen[strings.ToLower(m.Path)] {
					m.Outside = !isInside(m.Path, s.Folder)
					movies = append(movies, m)
					seen[strings.ToLower(m.Path)] = true
				}
			}
		}
	}
	return movies
}

type DropResult struct {
	Added   int    `json:"added"`
	Message string `json:"message"`
}

func (a *App) handleDrop(paths []string) DropResult {
	var dirs []string
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.IsDir() {
			dirs = append(dirs, p)
		} else if isVideo(p) {
			dirs = append(dirs, filepath.Dir(p))
		}
	}
	if len(dirs) == 0 {
		return DropResult{Message: "Drop a movie, or a folder with a movie inside."}
	}
	a.mu.Lock()
	a.extra = append(a.extra, dirs...)
	a.mu.Unlock()
	return DropResult{Added: len(dirs)}
}

func isInside(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// AddSubtitles puts the chosen subtitles inside the movie. Progress arrives as
// "prepare:progress" events; the call returns when it is finished.
func (a *App) AddSubtitles(video, subPath string) (PrepareResult, error) {
	s := a.Settings()
	m := describeMovie(video, s.Folder, s.SubLang)
	var sub *SubOption
	for i := range m.Subs {
		if m.Subs[i].Path == subPath {
			sub = &m.Subs[i]
		}
	}
	if sub == nil {
		return PrepareResult{}, fmt.Errorf("that subtitle file is gone")
	}
	return a.prep.Prepare(video, *sub, m.Title, s.RecycleOriginal, func(pct float64) {
		runtime.EventsEmit(a.ctx, "prepare:progress", map[string]interface{}{"path": video, "percent": pct})
	})
}

func (a *App) CancelSubtitles() { a.prep.Cancel() }

// ---- Small helpers for the page ----------------------------------------------------------

func (a *App) OpenFolder(p string) {
	if p == "" {
		p = a.Settings().Folder
	}
	_ = hiddenCmd("explorer.exe", p).Start()
}

func (a *App) ShowInFolder(p string) {
	_ = hiddenCmd("explorer.exe", "/select,", p).Start()
}

func (a *App) OpenNetworkSettings() {
	runtime.BrowserOpenURL(a.ctx, "ms-settings:network-status")
}

func (a *App) OpenURL(u string) {
	if strings.HasPrefix(u, "https://") {
		runtime.BrowserOpenURL(a.ctx, u)
	}
}
