package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

var videoExts = map[string]bool{".mp4": true, ".mkv": true, ".avi": true, ".m4v": true, ".mov": true, ".webm": true, ".wmv": true, ".ts": true}

const minVideoBytes = 30 << 20 // ignore clips under 30 MB (samples, trailers, screen grabs)

type SubOption struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Lang  string `json:"lang"` // ISO 639-2 code, "" if unknown
	SDH   bool   `json:"sdh"`
}

type Movie struct {
	Path       string      `json:"path"`
	Title      string      `json:"title"`
	SizeMB     int64       `json:"sizeMB"`
	HasSubs    bool        `json:"hasSubs"` // subtitles already inside the file
	Subs       []SubOption `json:"subs"`    // separate subtitle files found next to it
	DefaultSub int         `json:"defaultSub"`
	Outside    bool        `json:"outside"` // not in the shared folder (dropped in from elsewhere)
}

// scanMovies lists the videos in root (and its sub-folders, up to 3 levels).
func scanMovies(root, prefLang string) []Movie {
	var paths []string
	rootDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "$")) {
				return fs.SkipDir
			}
			if strings.Count(filepath.Clean(p), string(os.PathSeparator))-rootDepth > 3 {
				return fs.SkipDir
			}
			return nil
		}
		if isVideo(p) {
			if info, err := d.Info(); err == nil && info.Size() >= minVideoBytes && !strings.HasSuffix(strings.ToLower(p), ".partial.mkv") {
				paths = append(paths, p)
			}
		}
		return nil
	})
	return describeMovies(paths, root, prefLang)
}

func isVideo(p string) bool { return videoExts[strings.ToLower(filepath.Ext(p))] }

func describeMovies(paths []string, root, prefLang string) []Movie {
	movies := make([]Movie, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			movies[i] = describeMovie(p, root, prefLang)
		}(i, p)
	}
	wg.Wait()
	sort.Slice(movies, func(a, b int) bool {
		return strings.ToLower(movies[a].Title) < strings.ToLower(movies[b].Title)
	})
	return movies
}

func describeMovie(p, root, prefLang string) Movie {
	m := Movie{Path: p, Title: movieTitle(p, root)}
	if st, err := os.Stat(p); err == nil {
		m.SizeMB = st.Size() >> 20
	}
	m.HasSubs = hasEmbeddedSubs(p)
	m.Subs = findSubs(p)
	sortSubs(m.Subs, prefLang)
	m.DefaultSub = 0
	return m
}

func hasEmbeddedSubs(p string) bool {
	ffprobe := toolFFprobe.find()
	if ffprobe == "" {
		return false
	}
	out, err := hiddenCmd(ffprobe, "-v", "error", "-select_streams", "s",
		"-show_entries", "stream=index", "-of", "csv=p=0", p).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// --- Titles -------------------------------------------------------------------

var (
	reTitleYearParen = regexp.MustCompile(`^(.*?\(\d{4}\))`)
	reYear           = regexp.MustCompile(`^(.*?)[ ._\-\[(]+((?:19|20)\d{2})(?:[ ._\-\])]|$)`)
	reBadFileChars   = regexp.MustCompile(`[<>:"/\\|?*]`)
)

// movieTitle turns "Some.Movie.2026.1080p.WEBRip-[GROUP].mp4" (or its folder) into "Some Movie (2026)".
func movieTitle(p, root string) string {
	dir := filepath.Dir(p)
	fromFile := cleanTitle(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
	// A movie alone in its own folder: the folder name is usually the cleaner title,
	// but only if it looks like one ("Movie (2026)"), not "New folder" or "Downloads".
	if filepath.Clean(dir) != filepath.Clean(root) && countVideos(dir) == 1 {
		if fromDir := cleanTitle(filepath.Base(dir)); hasYear(fromDir) || !hasYear(fromFile) && looksLikeTitle(fromDir) {
			return fromDir
		}
	}
	return fromFile
}

var reYearSuffix = regexp.MustCompile(`\((?:19|20)\d{2}\)$`)

func hasYear(t string) bool { return reYearSuffix.MatchString(t) }

func looksLikeTitle(t string) bool {
	switch strings.ToLower(t) {
	case "new folder", "nouveau dossier", "downloads", "téléchargements", "videos", "vidéos", "movies", "films", "video", "film", "temp":
		return false
	}
	return t != ""
}

func cleanTitle(name string) string {
	if m := reTitleYearParen.FindStringSubmatch(name); m != nil {
		return strings.TrimSpace(m[1])
	}
	spaced := strings.NewReplacer(".", " ", "_", " ").Replace(name)
	if m := reYear.FindStringSubmatch(spaced); m != nil && strings.TrimSpace(m[1]) != "" {
		return strings.TrimSpace(m[1]) + " (" + m[2] + ")"
	}
	if i := strings.Index(spaced, " ["); i > 0 {
		spaced = spaced[:i]
	}
	return strings.TrimSpace(spaced)
}

func countVideos(dir string) int {
	n := 0
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && isVideo(e.Name()) {
			if info, err := e.Info(); err == nil && info.Size() >= minVideoBytes {
				n++
			}
		}
	}
	return n
}

// --- Subtitle files -------------------------------------------------------------

type lang struct{ code, name string }

var langTokens = map[string]lang{}

func init() {
	add := func(code, name string, tokens ...string) {
		for _, t := range append(tokens, code, strings.ToLower(name)) {
			langTokens[t] = lang{code, name}
		}
	}
	add("eng", "English", "en")
	add("fre", "French", "fra", "fr", "francais", "français")
	add("spa", "Spanish", "es", "esp", "español", "espanol", "castellano")
	add("ger", "German", "deu", "de", "deutsch")
	add("ita", "Italian", "it", "italiano")
	add("por", "Portuguese", "pt", "portugues", "português")
	add("ara", "Arabic", "ar")
	add("dut", "Dutch", "nld", "nl")
	add("tur", "Turkish", "tr")
	add("pol", "Polish", "pl")
	add("rus", "Russian", "ru")
	add("ukr", "Ukrainian", "uk")
	add("rum", "Romanian", "ron", "ro")
	add("gre", "Greek", "ell", "el")
	add("swe", "Swedish", "sv")
	add("nor", "Norwegian", "nob", "no", "nb")
	add("dan", "Danish", "da")
	add("fin", "Finnish", "fi")
	add("cze", "Czech", "ces", "cs")
	add("slo", "Slovak", "slk", "sk")
	add("slv", "Slovenian", "sl")
	add("hrv", "Croatian", "hr")
	add("srp", "Serbian", "sr")
	add("bul", "Bulgarian", "bg")
	add("hun", "Hungarian", "hu")
	add("est", "Estonian", "et")
	add("lav", "Latvian", "lv")
	add("lit", "Lithuanian", "lt")
	add("ice", "Icelandic", "isl", "is")
	add("heb", "Hebrew", "he")
	add("hin", "Hindi", "hi-in")
	add("ind", "Indonesian", "id")
	add("may", "Malay", "msa", "ms")
	add("tha", "Thai", "th")
	add("vie", "Vietnamese", "vi")
	add("chi", "Chinese", "zho", "zh", "chs", "cht")
	add("jpn", "Japanese", "ja", "jp")
	add("kor", "Korean", "ko")
	add("per", "Persian", "fas", "fa")
}

var reSplitTokens = regexp.MustCompile(`[\s._\-()\[\]]+`)

// findSubs collects .srt files next to the video and in a "Subs"/"Subtitles" folder beside it.
func findSubs(video string) []SubOption {
	dir := filepath.Dir(video)
	base := strings.TrimSuffix(filepath.Base(video), filepath.Ext(video))
	alone := countVideos(dir) == 1
	var subs []SubOption

	consider := func(p string, nextToVideo bool) {
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		if nextToVideo && !alone && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
			return // in a shared folder, only take subtitles named after this video
		}
		label, code, sdh := describeSub(name, base)
		subs = append(subs, SubOption{Path: p, Label: label, Lang: code, SDH: sdh})
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			n := strings.ToLower(e.Name())
			if n == "subs" || n == "sub" || n == "subtitles" || n == "subtitle" {
				inner, _ := os.ReadDir(filepath.Join(dir, e.Name()))
				for _, f := range inner {
					if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".srt") {
						consider(filepath.Join(dir, e.Name(), f.Name()), false)
					}
				}
			}
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".srt") {
			consider(filepath.Join(dir, e.Name()), true)
		}
	}
	return subs
}

func describeSub(name, videoBase string) (label, code string, sdh bool) {
	rest := name
	if strings.HasPrefix(strings.ToLower(name), strings.ToLower(videoBase)) {
		rest = strings.TrimSpace(name[len(videoBase):])
	}
	tokens := reSplitTokens.Split(strings.ToLower(rest), -1)
	var variant []string
	var found *lang
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if t == "sdh" || t == "hi" || t == "cc" {
			sdh = true
			continue
		}
		if l, ok := langTokens[t]; ok && found == nil {
			lc := l
			found = &lc
			continue
		}
		if found == nil && len(t) > 2 {
			variant = append(variant, t)
		}
	}
	if found == nil {
		if rest == "" {
			return "Subtitles (same name as the movie)", "", sdh
		}
		return name, "", sdh
	}
	label = found.name
	if len(variant) > 0 && len(variant) <= 2 {
		label += " · " + titleCase(strings.Join(variant, " "))
	}
	if sdh {
		label += " · for the hard of hearing"
	}
	return label, found.code, sdh
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = strings.ToUpper(string(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

// sortSubs puts the preferred language first (plain before hard-of-hearing), then the rest by name.
func sortSubs(subs []SubOption, pref string) {
	rank := func(s SubOption) int {
		switch {
		case s.Lang == pref && !s.SDH:
			return 0
		case s.Lang == pref:
			return 1
		case s.Lang == "":
			return 3
		default:
			return 2
		}
	}
	sort.SliceStable(subs, func(a, b int) bool {
		ra, rb := rank(subs[a]), rank(subs[b])
		if ra != rb {
			return ra < rb
		}
		return subs[a].Label < subs[b].Label
	})
}

// --- Putting subtitles inside the file ----------------------------------------------

type PrepareResult struct {
	Output   string `json:"output"`
	Recycled bool   `json:"recycled"`
}

type Preparer struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (pr *Preparer) Busy() bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.cancel != nil
}

func (pr *Preparer) Cancel() {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if pr.cancel != nil {
		pr.cancel()
	}
}

// Prepare copies video + audio unchanged into an MKV with the chosen subtitles inside,
// checks the result, then (optionally) sends the original to the Recycle Bin.
func (pr *Preparer) Prepare(video string, sub SubOption, title string, recycle bool, progress func(float64)) (PrepareResult, error) {
	var res PrepareResult
	ffmpeg, ffprobe := toolFFmpeg.find(), toolFFprobe.find()
	if ffmpeg == "" || ffprobe == "" {
		return res, fmt.Errorf("FFmpeg is not installed yet")
	}
	if !fileExists(video) || !fileExists(sub.Path) {
		return res, fmt.Errorf("the movie or subtitle file is gone")
	}

	pr.mu.Lock()
	if pr.cancel != nil {
		pr.mu.Unlock()
		return res, fmt.Errorf("another movie is already being prepared")
	}
	ctx, cancel := context.WithCancel(context.Background())
	pr.cancel = cancel
	pr.mu.Unlock()
	defer func() {
		pr.mu.Lock()
		pr.cancel = nil
		pr.mu.Unlock()
		cancel()
	}()

	dir := filepath.Dir(video)
	name := strings.TrimSpace(reBadFileChars.ReplaceAllString(title, ""))
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(video), filepath.Ext(video))
	}
	out := filepath.Join(dir, name+".mkv")
	if strings.EqualFold(out, video) || fileExists(out) {
		out = filepath.Join(dir, name+" (with subtitles).mkv")
	}
	tmp := strings.TrimSuffix(out, ".mkv") + ".partial.mkv"
	defer os.Remove(tmp)

	duration := probeDuration(ffprobe, video)

	enc := "UTF-8"
	if b, err := os.ReadFile(sub.Path); err == nil && !utf8.Valid(b) {
		enc = "CP1252"
	}
	langCode := sub.Lang
	if langCode == "" {
		langCode = "und"
	}
	subTitle := sub.Label
	if sub.Lang == "" {
		subTitle = "Subtitles"
	}

	cmd := hiddenCmd(ffmpeg, "-hide_banner", "-nostats", "-loglevel", "error", "-y",
		"-i", video,
		"-sub_charenc", enc, "-i", sub.Path,
		"-map", "0:v", "-map", "0:a?", "-map", "1:0",
		"-c", "copy", "-c:s", "srt",
		"-metadata:s:s:0", "language="+langCode,
		"-metadata:s:s:0", "title="+subTitle,
		"-disposition:s:0", "default",
		"-progress", "pipe:1",
		tmp)
	stdout, _ := cmd.StdoutPipe()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return res, err
	}
	attachToJob(cmd.Process)
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "out_time_us=") && duration > 0 {
			if us, err := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_us="), 64); err == nil {
				progress(math.Min(99, us/1e6/duration*100))
			}
		}
	}
	err := cmd.Wait()
	if ctx.Err() != nil {
		return res, fmt.Errorf("cancelled")
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return res, fmt.Errorf("FFmpeg could not build the file: %s", lastLine(msg))
	}

	// Check: same length and a subtitle track, before touching the original.
	if d := probeDuration(ffprobe, tmp); duration > 0 && math.Abs(d-duration) > 2 {
		return res, fmt.Errorf("the new file is %.0f s long instead of %.0f s; the original is untouched", d, duration)
	}
	if !hasEmbeddedSubs(tmp) {
		return res, fmt.Errorf("the new file has no subtitle track; the original is untouched")
	}
	if err := os.Rename(tmp, out); err != nil {
		return res, err
	}
	res.Output = out
	progress(100)

	if recycle {
		if err := moveToRecycleBin(video); err == nil {
			res.Recycled = true
		}
	}
	return res, nil
}

func probeDuration(ffprobe, p string) float64 {
	out, err := hiddenCmd(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", p).Output()
	if err != nil {
		return 0
	}
	d, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return d
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
