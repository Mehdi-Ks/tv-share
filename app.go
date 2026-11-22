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

const appVersion = "1.8.0"

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
	tmdb     *TMDB
}

func NewApp() *App {
	a := &App{settings: loadSettings(), share: &Share{}, prep: &Preparer{}, tmdb: newTMDB()}
	a.tmdb.setKey(a.tmdbKey())
	return a
}

// tmdbKey is the user's own key if they set one, otherwise the one built into this release.
func (a *App) tmdbKey() string {
	if k := strings.TrimSpace(a.Settings().TMDBKey); k != "" {
		return k
	}
	return tmdbBuiltInKey
}

// tmdbOn: the user chose official posters and a key is available.
func (a *App) tmdbOn() bool { return a.Settings().TMDB == "on" && a.tmdb.hasKey() }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	initJob()
	go clearOldTMDBCache()
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
	TMDB     TMDBState   `json:"tmdb"`
}

type TMDBState struct {
	Available bool `json:"available"` // a key is built in or was entered
	BuiltIn   bool `json:"builtIn"`   // this release has its own key
	OwnKey    bool `json:"ownKey"`    // the user entered a key
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
		TMDB: TMDBState{
			Available: a.tmdb.hasKey(),
			BuiltIn:   tmdbBuiltInKey != "",
			OwnKey:    strings.TrimSpace(a.Settings().TMDBKey) != "",
		},
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
	direct := func(pct float64, msg string) { runtime.EventsEmit(a.ctx, "setup", msg) }
	// winget first (it keeps the helpers up to date); if it's missing or fails,
	// download them straight from their official sites, checked against their checksums.
	if err := installTool(toolRclone, report("rclone")); err != nil && toolRclone.find() == "" {
		if derr := downloadRclone(direct); derr != nil {
			return depsStatus(), helperError(err, derr)
		}
	}
	if err := installTool(toolFFmpeg, report("FFmpeg")); err != nil && (toolFFmpeg.find() == "" || toolFFprobe.find() == "") {
		if derr := downloadFFmpeg(direct); derr != nil {
			return depsStatus(), helperError(err, derr)
		}
	}
	return depsStatus(), nil
}

// helperError explains a failed setup in plain words.
func helperError(wingetErr, downloadErr error) error {
	if wingetErr == errNoWinget {
		return fmt.Errorf("Couldn't get the helpers: %v. Check your internet connection and try again", downloadErr)
	}
	return fmt.Errorf("Couldn't get the helpers: %v (Windows' installer also failed: %v)", downloadErr, wingetErr)
}

// OpenAppInstallerStore opens the Microsoft Store page of "App Installer" (it includes winget).
func (a *App) OpenAppInstallerStore() {
	runtime.BrowserOpenURL(a.ctx, "ms-windows-store://pdp/?productid=9NBLGGH4NNS1")
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

// PickFolder asks for a folder without saving it (first-run screen).
func (a *App) PickFolder(cur string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose the folder with your movies",
		DefaultDirectory: cur,
	})
}

// CountMovies says how many movies a folder holds, for the first-run screen.
func (a *App) CountMovies(folder string) int {
	if !dirExists(folder) {
		return -1
	}
	return len(findVideos(folder))
}

// FinishSetup saves the first-run choices: the movie folder and the name the TV shows.
func (a *App) FinishSetup(folder, name string) error {
	name = strings.TrimSpace(name)
	if !dirExists(folder) {
		return fmt.Errorf("that folder doesn't exist any more")
	}
	if name == "" {
		return fmt.Errorf("give your PC a name, so you can find it on the TV")
	}
	if len(name) > 40 {
		name = name[:40]
	}
	return a.updateSettings(func(s *Settings) { s.Folder, s.Name, s.SetupDone = folder, name, true })
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
	case "autoSync":
		return a.updateSettings(func(s *Settings) { s.NoAutoSync = !value })
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
	return a.prep.Prepare(video, *sub, m.Title, s.RecycleOriginal, !s.NoAutoSync, a.progressFor(video))
}

// progressFor sends "prepare:progress" events for one movie, only when the number or step changes.
func (a *App) progressFor(video string) func(float64, string) {
	lastPct, lastStage := -1, ""
	return func(pct float64, stage string) {
		if int(pct) == lastPct && stage == lastStage {
			return
		}
		lastPct, lastStage = int(pct), stage
		runtime.EventsEmit(a.ctx, "prepare:progress", map[string]interface{}{"path": video, "percent": pct, "stage": stage})
	}
}

// FixSubtitleSync re-times the subtitles inside a movie. shiftMs == 0 matches them to the sound
// automatically; otherwise it moves them by that much (negative = earlier).
func (a *App) FixSubtitleSync(video string, shiftMs int) (PrepareResult, error) {
	return a.prep.FixSync(video, shiftMs == 0, shiftMs, a.Settings().RecycleOriginal, a.progressFor(video))
}

func (a *App) CancelSubtitles() { a.prep.Cancel() }

// GetPoster returns a poster image (a frame from the movie) as a data: URL,
// or an error when no frame can be taken; the page then draws a title cover.
func (a *App) GetPoster(video string) (string, error) {
	if a.tmdbOn() {
		if m, ok, _ := a.tmdb.meta(a.titleOf(video)); ok && m.Poster != "" {
			if img, err := a.tmdb.image(m.Poster, "w342"); err == nil {
				return img, nil
			}
		}
	}
	return posterDataURL(video)
}

func (a *App) titleOf(video string) string { return movieTitle(video, a.Settings().Folder) }

// GetMovieMeta returns TMDB's plot, genres and rating (ID 0 when off or not found).
func (a *App) GetMovieMeta(video string) (MovieMeta, error) {
	if !a.tmdbOn() {
		return MovieMeta{}, nil
	}
	m, ok, err := a.tmdb.meta(a.titleOf(video))
	if !ok {
		m = MovieMeta{}
		if _, chosen := a.tmdb.matches.get(a.titleOf(video)); chosen {
			m.Manual = true // "use a frame" was chosen for this one
		}
	}
	return m, err
}

// FindTMDBMatches lists TMDB's best guesses for "Not this movie?". query may be empty.
func (a *App) FindTMDBMatches(video, query string) ([]TMDBCandidate, error) {
	return a.tmdb.candidates(a.titleOf(video), query)
}

// SetTMDBMatch: id > 0 uses that TMDB movie, -1 uses a frame from the movie, 0 goes back to automatic.
func (a *App) SetTMDBMatch(video string, id int) error {
	return a.tmdb.matches.set(a.titleOf(video), id)
}

// SetLanguage tells the app which language the window shows ("en", "fr"), for TMDB plots and genres.
func (a *App) SetLanguage(lang string) { a.tmdb.setLanguage(lang) }

// SetTMDBChoice records the answer to "Use official posters?".
func (a *App) SetTMDBChoice(on bool) error {
	v := "off"
	if on {
		v = "on"
	}
	return a.updateSettings(func(s *Settings) { s.TMDB = v })
}

// ---- OpenSubtitles ---------------------------------------------------------------

func (a *App) osClient() *osClient {
	s := a.Settings()
	key := strings.TrimSpace(s.OSKey)
	if key == "" {
		key = osBuiltInKey
	}
	return &osClient{key: key, token: s.OSToken, user: s.OSUser, base: s.OSBase}
}

// GetOpenSubtitlesStatus is what Settings shows: key, sign-in and downloads left today.
func (a *App) GetOpenSubtitlesStatus() OSStatus {
	c := a.osClient()
	st := OSStatus{Available: c.key != "", BuiltIn: osBuiltInKey != "", OwnKey: strings.TrimSpace(a.Settings().OSKey) != "",
		User: c.user, Remaining: -1}
	if c.key != "" && c.token != "" {
		st.Remaining, st.ResetIn = c.remaining()
	}
	return st
}

// FindSubtitlesOnline searches OpenSubtitles for a movie in one language (our 3-letter code).
func (a *App) FindSubtitlesOnline(video, lang string) ([]OSResult, error) {
	c := a.osClient()
	if c.key == "" {
		return nil, fmt.Errorf("Add an OpenSubtitles key in Settings to search for subtitles online")
	}
	name, year := splitTitleYear(a.titleOf(video))
	return c.search(video, name, year, lang)
}

// DownloadSubtitleOnline saves the chosen subtitles next to the movie; the page then adds them as usual.
func (a *App) DownloadSubtitleOnline(video string, fileID int, lang string) (string, error) {
	c := a.osClient()
	if c.key == "" {
		return "", fmt.Errorf("Add an OpenSubtitles key in Settings to search for subtitles online")
	}
	p, _, err := c.download(fileID, video, lang)
	return p, err
}

// SignInOpenSubtitles signs in with the user's own free account. The password is sent to
// OpenSubtitles once and never stored; only the token they give back is kept.
func (a *App) SignInOpenSubtitles(user, pass string) error {
	user = strings.TrimSpace(user)
	if user == "" || pass == "" {
		return fmt.Errorf("Enter your OpenSubtitles username and password")
	}
	c := a.osClient()
	if c.key == "" {
		return fmt.Errorf("Add an OpenSubtitles key first")
	}
	c.token, c.base = "", ""
	if err := c.login(user, pass); err != nil {
		return err
	}
	return a.updateSettings(func(s *Settings) { s.OSUser, s.OSToken, s.OSBase = c.user, c.token, c.base })
}

func (a *App) SignOutOpenSubtitles() error {
	a.osClient().logout()
	return a.updateSettings(func(s *Settings) { s.OSUser, s.OSToken, s.OSBase = "", "", "" })
}

// SetOpenSubtitlesKey saves the user's own key after checking it ("" goes back to the built-in one).
func (a *App) SetOpenSubtitlesKey(key string) error {
	key = strings.TrimSpace(key)
	if key != "" {
		if err := checkOSKey(key); err != nil {
			return err
		}
	}
	return a.updateSettings(func(s *Settings) { s.OSKey = key })
}

// OpenSubtitlesSignupPage and OpenSubtitlesKeyPage open the right OpenSubtitles pages in the browser.
func (a *App) OpenSubtitlesSignupPage() { runtime.BrowserOpenURL(a.ctx, osSignup) }
func (a *App) OpenSubtitlesKeyPage()    { runtime.BrowserOpenURL(a.ctx, osKeyPage) }

// SetTMDBKey saves the user's own key after checking it with TMDB ("" goes back to the built-in one).
func (a *App) SetTMDBKey(key string) error {
	key = strings.TrimSpace(key)
	if key != "" {
		if err := checkTMDBKey(key); err != nil {
			return err
		}
	}
	if err := a.updateSettings(func(s *Settings) { s.TMDBKey = key }); err != nil {
		return err
	}
	a.tmdb.setKey(a.tmdbKey())
	return nil
}

// GetMovieInfo returns runtime, picture, audio and subtitle details for the details view.
func (a *App) GetMovieInfo(video string) (MovieInfo, error) { return movieInfo(video) }

// GetBackdrop returns a wide frame of the movie for the details view.
func (a *App) GetBackdrop(video string) (string, error) {
	if a.tmdbOn() {
		if m, ok, _ := a.tmdb.meta(a.titleOf(video)); ok && m.Backdrop != "" {
			if img, err := a.tmdb.image(m.Backdrop, "w1280"); err == nil {
				return img, nil
			}
		}
	}
	return backdropDataURL(video)
}

// ---- Small helpers for the page ----------------------------------------------------------

func (a *App) OpenFolder(p string) {
	if p == "" {
		p = a.Settings().Folder
	}
	allowSetForeground.Call(uintptr(asfwAny)) // so the folder opens in front of the app
	_ = hiddenCmd("explorer.exe", p).Start()
}

// ShowInFolder opens Explorer with the file selected (or its folder, if the file is gone).
func (a *App) ShowInFolder(p string) {
	if !fileExists(p) {
		a.OpenFolder(filepath.Dir(p))
		return
	}
	showInExplorer(p)
}

func (a *App) OpenNetworkSettings() {
	runtime.BrowserOpenURL(a.ctx, "ms-settings:network-status")
}

func (a *App) OpenURL(u string) {
	if strings.HasPrefix(u, "https://") {
		runtime.BrowserOpenURL(a.ctx, u)
	}
}
