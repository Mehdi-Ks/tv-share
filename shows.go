package main

// TV shows and extras. Every video file is still one item (so subtitles, sync, playing and the TV
// work per file), but each one now says what it is:
//   - an episode: "Show (Year) - S01E03 - Title.mkv", "show.1x03.mkv", or "Season 1/03 - Title.mkv"
//   - an extra: anything in a Featurettes / Extras / Deleted Scenes / Trailers... folder
//   - a movie: everything else
// The page groups episodes into one card per show and puts extras on their show's or movie's page.
// When the guess is wrong, the user can switch a file (or a whole show) between show and movie.

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	reSxE        = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])s(\d{1,2})[ ._-]?e(\d{1,3})((?:[ ._-]?e\d{1,3})*)(?:[^0-9]|$)`)
	reNxN        = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(\d{1,2})x(\d{2,3})(?:[^0-9]|$)`)
	reSeasonDir  = regexp.MustCompile(`(?i)^(?:season|saison|series|staffel|temporada|stagione)[ ._-]*(\d{1,2})\b|^s(\d{1,2})$`)
	reSeasonWord = regexp.MustCompile(`(?i)[ ._-]*(?:\(|\[)?\b(?:season|saison|series|s)[ ._-]*\d{1,2}\b.*$`)
	reEpOnly     = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:e|ep|episode|épisode)[ ._-]?(\d{1,3})(?:[^0-9]|$)`)
	reLeadingNum = regexp.MustCompile(`^(\d{1,3})(?:[ ._-]|$)`)
	reQuality    = regexp.MustCompile(`(?i)[ ._-]*[\[(]?\b(2160p|1080p|720p|576p|480p|4k|uhd|web[ ._-]?dl|webrip|web|bluray|blu[ ._-]?ray|bdrip|brrip|hdtv|hdrip|dvdrip|x264|x265|h[ .]?264|h[ .]?265|hevc|10bit|amzn|nf|dsnp|hmax|atvp|repack|proper)\b.*$`)
	reSample     = regexp.MustCompile(`(?i)(?:^|[ ._-])sample(?:[ ._-]|$)`)
)

// Folder names that hold extras (Plex and Jellyfin use the same ones).
var extrasDirs = map[string]bool{
	"featurettes": true, "featurette": true, "extras": true, "extra": true, "bonus": true, "bonuses": true,
	"behind the scenes": true, "deleted scenes": true, "interviews": true, "scenes": true, "shorts": true,
	"trailers": true, "trailer": true, "other": true, "others": true, "making of": true, "bonus features": true,
	"special features": true,
}

const minExtraBytes = 1 << 20 // extras can be short (a 12 MB gag reel), but not tiny samples

// Kind overrides the user chose ("This is a show" / "These are movies"): lower-case path -> "show" or "movie".
var kindOverrides sync.Map

func setKindOverrides(m map[string]string) {
	kindOverrides.Range(func(k, _ any) bool { kindOverrides.Delete(k); return true })
	for k, v := range m {
		kindOverrides.Store(k, v)
	}
}

func kindOverride(p string) string {
	if v, ok := kindOverrides.Load(lowerPath(p)); ok {
		return v.(string)
	}
	return ""
}

func isSample(p string) bool {
	if strings.EqualFold(filepath.Base(filepath.Dir(p)), "sample") {
		return true
	}
	return reSample.MatchString(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
}

// extrasFolder returns the folder that holds the extras folder p is in ("" if p isn't an extra).
func extrasFolder(p, root string) string {
	root = filepath.Clean(root)
	for dir := filepath.Dir(p); dir != root && len(dir) > len(root); dir = filepath.Dir(dir) {
		if extrasDirs[strings.ToLower(strings.TrimSpace(filepath.Base(dir)))] {
			parent := filepath.Dir(dir)
			if strings.EqualFold(filepath.Clean(parent), root) {
				// "Other" or "Shorts" right in the movie folder is the user's own folder, not extras.
				return ""
			}
			return parent
		}
	}
	return ""
}

type episodeInfo struct {
	show    string // "WandaVision (2021)"
	season  int
	episode int
	title   string // "Now in Color", "" if unknown
}

// parseEpisode recognises an episode from its file and folder names. forced says the user marked it
// as a show, so a file without numbers still counts (its episode number is filled in later).
func parseEpisode(p, root string, forced bool) (episodeInfo, bool) {
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	dir := filepath.Dir(p)
	dirName := filepath.Base(dir)
	seasonFromDir := 0
	if m := reSeasonDir.FindStringSubmatch(strings.TrimSpace(dirName)); m != nil {
		seasonFromDir = atoiFirst(m[1], m[2])
	}

	var e episodeInfo
	var before, after string
	if loc := reSxE.FindStringSubmatchIndex(base); loc != nil {
		e.season, _ = strconv.Atoi(base[loc[2]:loc[3]])
		e.episode, _ = strconv.Atoi(base[loc[4]:loc[5]])
		before, after = base[:loc[2]-1], base[loc[7]:] // after a double episode's "E02" too
	} else if loc := reNxN.FindStringSubmatchIndex(base); loc != nil && !looksLikeResolution(base[loc[2]:loc[5]]) {
		e.season, _ = strconv.Atoi(base[loc[2]:loc[3]])
		e.episode, _ = strconv.Atoi(base[loc[4]:loc[5]])
		before, after = base[:loc[2]], base[loc[5]:]
	} else if seasonFromDir > 0 || forced {
		e.season = max(seasonFromDir, 1)
		if loc := reEpOnly.FindStringSubmatchIndex(base); loc != nil {
			e.episode, _ = strconv.Atoi(base[loc[2]:loc[3]])
			before, after = base[:loc[0]], base[loc[3]:]
		} else if m := reLeadingNum.FindStringSubmatch(base); m != nil && !forced {
			e.episode, _ = strconv.Atoi(m[1])
			after = base[len(m[1]):]
		} else if !forced {
			return e, false
		} else {
			after = base
		}
	} else {
		return e, false
	}

	e.title = episodeTitle(after)
	e.show = showName(before, p, root, seasonFromDir > 0)
	if e.show == "" {
		return e, false
	}
	return e, true
}

func atoiFirst(vals ...string) int {
	for _, v := range vals {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

func looksLikeResolution(s string) bool {
	s = strings.ToLower(s)
	return s == "19x10" || s == "12x80" || strings.HasSuffix(s, "x720") || strings.HasSuffix(s, "x1080")
}

// episodeTitle turns " - Now in Color (1080p BluRay x265 Silence)" into "Now in Color".
func episodeTitle(s string) string {
	s = strings.NewReplacer("_", " ").Replace(s)
	if strings.Count(s, ".") >= 2 && !strings.Contains(s, " ") {
		s = strings.ReplaceAll(s, ".", " ")
	}
	s = reQuality.ReplaceAllString(s, "")
	if i := strings.IndexAny(s, "(["); i >= 0 {
		s = s[:i]
	}
	s = strings.Trim(s, " .-_–—")
	if len(s) < 2 {
		return ""
	}
	return s
}

// showName finds the show's name: from the file name before "S01E03", else from the folder
// (skipping a "Season 1" folder), else "".
func showName(before, p, root string, inSeasonDir bool) string {
	if t := cleanShowTitle(before); t != "" {
		return t
	}
	dir := filepath.Dir(p)
	if inSeasonDir {
		dir = filepath.Dir(dir)
	}
	if filepath.Clean(dir) == filepath.Clean(root) || len(dir) < len(filepath.Clean(root)) {
		return ""
	}
	return cleanShowTitle(filepath.Base(dir))
}

// cleanShowTitle: "WandaVision (2021) Season 1 S01 (1080p ...)" -> "WandaVision (2021)", "The.Office.US" -> "The Office US".
func cleanShowTitle(s string) string {
	s = strings.Trim(s, " ._-")
	if s == "" {
		return ""
	}
	if m := reTitleYearParen.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	s = reSeasonWord.ReplaceAllString(s, "")
	s = reQuality.ReplaceAllString(s, "")
	t := cleanTitle(s)
	t = strings.Trim(t, " -")
	if !looksLikeTitle(t) {
		return ""
	}
	return t
}

// classify fills in what a file is. Called for every file the scan finds.
func classify(m *Movie, root string) {
	ov := kindOverride(m.Path)
	if ov != "show" && ov != "movie" {
		if parent := extrasFolder(m.Path, root); parent != "" {
			m.Kind = "extra"
			m.Title = extraTitle(m.Path)
			m.ExtraFor = parent // turned into a show key or movie path by linkExtras
			return
		}
	}
	if ov != "movie" {
		if e, ok := parseEpisode(m.Path, root, ov == "show"); ok {
			m.Kind = "episode"
			m.Show = e.show
			m.Season, m.Episode, m.EpTitle = e.season, e.episode, e.title
			m.ShowKey = showKey(e.show)
			m.Title = e.show
			return
		}
	}
	m.Kind = "movie"
}

func extraTitle(p string) string {
	t := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	t = strings.NewReplacer("_", " ").Replace(t)
	if !strings.Contains(t, " ") {
		t = strings.ReplaceAll(t, ".", " ")
	}
	if s := reQuality.ReplaceAllString(t, ""); strings.TrimSpace(s) != "" {
		t = s
	}
	return strings.Trim(t, " .-")
}

// showKey groups episodes of one show, with or without the year: "WandaVision (2021)" and "WandaVision".
func showKey(show string) string {
	name, _ := splitTitleYear(show)
	return normTitle(name)
}

// organize runs after a scan: one display name per show, episode numbers for files the user
// marked as a show, and each extra linked to its show or movie.
func organize(items []Movie) {
	// One name per show: the one with a year wins ("WandaVision (2021)" over "WandaVision").
	names := map[string]string{}
	for _, m := range items {
		if m.Kind != "episode" {
			continue
		}
		cur := names[m.ShowKey]
		if cur == "" || (!hasYear(cur) && hasYear(m.Show)) {
			names[m.ShowKey] = m.Show
		}
	}
	// Files marked as a show without numbers: number them in file-name order after the others.
	type sk struct {
		key    string
		season int
	}
	maxEp := map[sk]int{}
	var unnumbered []int
	for i, m := range items {
		if m.Kind != "episode" {
			continue
		}
		items[i].Show, items[i].Title = names[m.ShowKey], names[m.ShowKey]
		k := sk{m.ShowKey, m.Season}
		if m.Episode == 0 {
			unnumbered = append(unnumbered, i)
		} else if m.Episode > maxEp[k] {
			maxEp[k] = m.Episode
		}
	}
	sort.SliceStable(unnumbered, func(a, b int) bool {
		return strings.ToLower(items[unnumbered[a]].Path) < strings.ToLower(items[unnumbered[b]].Path)
	})
	for _, i := range unnumbered {
		k := sk{items[i].ShowKey, items[i].Season}
		maxEp[k]++
		items[i].Episode = maxEp[k]
	}

	// Extras: the folder holding the extras folder decides. Episodes there (or in its season
	// folders) -> the show; exactly one movie there -> that movie; otherwise they're just movies.
	showIn := map[string]string{}  // folder -> show key
	moviesIn := map[string][]int{} // folder -> movie indexes
	for i, m := range items {
		d := strings.ToLower(filepath.Dir(m.Path))
		switch m.Kind {
		case "episode":
			showIn[d] = m.ShowKey
			if reSeasonDir.MatchString(strings.TrimSpace(filepath.Base(filepath.Dir(m.Path)))) {
				showIn[strings.ToLower(filepath.Dir(filepath.Dir(m.Path)))] = m.ShowKey
			}
		case "movie":
			moviesIn[d] = append(moviesIn[d], i)
		}
	}
	for i, m := range items {
		if m.Kind != "extra" {
			continue
		}
		d := strings.ToLower(m.ExtraFor)
		if k, ok := showIn[d]; ok {
			items[i].ExtraFor = "show:" + k
		} else if ms := moviesIn[d]; len(ms) == 1 && sameTitle(items[ms[0]].Title, filepath.Base(m.ExtraFor)) {
			// Only a folder that is that movie's own ("Dune (2021)\Extras"), not "Movies\Other".
			items[i].ExtraFor = items[ms[0]].Path
		} else {
			items[i].Kind, items[i].ExtraFor = "movie", ""
			items[i].Title = movieTitle(m.Path, filepath.Dir(m.Path))
		}
	}
}

// sameTitle says whether a folder is named after a movie: "Dune (2021) [1080p]" and "Dune (2021)".
func sameTitle(title, folder string) bool {
	f := cleanShowTitle(folder)
	return f != "" && showKey(f) == showKey(title)
}

func lowerPath(p string) string { return strings.ToLower(filepath.Clean(p)) }
