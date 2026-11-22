package main

// Official posters and movie info from TMDB (themoviedb.org). Opt-in: nothing is looked up
// unless the user chose "Use official posters". Only the movie's title and year are sent.
//
// This product uses the TMDB API but is not endorsed or certified by TMDB.
//
// Release builds get a key at build time (build.ps1 reads build/tmdb.key, which is never committed):
//   -ldflags "-X main.tmdbBuiltInKey=..."
// Anyone can also paste their own key in Settings, which then takes priority.

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var tmdbBuiltInKey = "" // set by the release build, see build.ps1

// Endpoints are variables so tests can point them at a fake server.
var (
	tmdbAPI    = "https://api.themoviedb.org/3"
	tmdbImages = "https://image.tmdb.org/t/p"
)

// TMDB's terms: cached data may be kept 6 months at most. We refresh well before that.
const tmdbMaxAge = 90 * 24 * time.Hour

var tmdbHTTP = &http.Client{Timeout: 12 * time.Second}

// MovieMeta is what the details page shows from TMDB.
type MovieMeta struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Year     string   `json:"year"`
	Overview string   `json:"overview"`
	Genres   []string `json:"genres"`
	Rating   float64  `json:"rating"` // out of 10
	Votes    int      `json:"votes"`
	URL      string   `json:"url"` // the movie's page on themoviedb.org
	Poster   string   `json:"posterPath"`
	Backdrop string   `json:"backdropPath"`
	Fetched  int64    `json:"fetched"`
	Manual   bool     `json:"manual"`            // the user picked this match
	Seasons  int      `json:"seasons,omitempty"` // TV shows
}

// EpisodeMeta is one episode's name, plot and picture from TMDB.
type EpisodeMeta struct {
	Episode  int     `json:"episode"`
	Name     string  `json:"name"`
	Overview string  `json:"overview"`
	Still    string  `json:"stillPath"`
	Minutes  int     `json:"minutes"`
	Rating   float64 `json:"rating"`
}

type seasonCache struct {
	Fetched  int64         `json:"fetched"`
	Episodes []EpisodeMeta `json:"episodes"`
}

func (c *tmdbClient) season(id, n int) ([]EpisodeMeta, error) {
	var d struct {
		Episodes []struct {
			Episode  int     `json:"episode_number"`
			Name     string  `json:"name"`
			Overview string  `json:"overview"`
			Still    string  `json:"still_path"`
			Runtime  int     `json:"runtime"`
			Vote     float64 `json:"vote_average"`
		} `json:"episodes"`
	}
	if err := c.get("/tv/"+strconv.Itoa(id)+"/season/"+strconv.Itoa(n), url.Values{"language": {c.language()}}, &d); err != nil {
		return nil, err
	}
	out := []EpisodeMeta{}
	for _, e := range d.Episodes {
		out = append(out, EpisodeMeta{Episode: e.Episode, Name: e.Name, Overview: e.Overview, Still: e.Still, Minutes: e.Runtime, Rating: e.Vote})
	}
	return out, nil
}

// TMDBCandidate is one choice in the "Not this movie?" list.
type TMDBCandidate struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Year   string `json:"year"`
	Poster string `json:"poster"` // small data: URL, or ""
}

type tmdbClient struct {
	key  string
	lang string     // TMDB language for plots and genres, e.g. "fr-FR"; "" means English
	mu   sync.Mutex // one lookup at a time per title is enough; keeps us polite with TMDB
}

func (c *tmdbClient) language() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lang == "" {
		return "en-US"
	}
	return c.lang
}

func (c *tmdbClient) get(path string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	req, err := http.NewRequest("GET", tmdbAPI+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	// v4 "read access tokens" are long JWTs; v3 "API keys" are 32 hex characters.
	if strings.HasPrefix(c.key, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+c.key)
	} else {
		q.Set("api_key", c.key)
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Set("Accept", "application/json")
	resp, err := tmdbHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't reach TMDB (are you online?)")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("TMDB didn't accept the key")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("TMDB answered %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

type tmdbSearchResult struct {
	Results []struct {
		ID           int     `json:"id"`
		Title        string  `json:"title"`
		Original     string  `json:"original_title"`
		ReleaseDate  string  `json:"release_date"`
		Name         string  `json:"name"`           // TV shows
		OriginalName string  `json:"original_name"`  // TV shows
		FirstAir     string  `json:"first_air_date"` // TV shows
		PosterPath   string  `json:"poster_path"`
		Popularity   float64 `json:"popularity"`
	} `json:"results"`
}

// A kind of TMDB lookup: movies (""), or TV shows.
const tvKind = "tv"

func (c *tmdbClient) search(name, year string) (tmdbSearchResult, error) {
	return c.searchKind("", name, year)
}

func (c *tmdbClient) searchKind(kind, name, year string) (tmdbSearchResult, error) {
	var r tmdbSearchResult
	q := url.Values{"query": {name}, "include_adult": {"false"}, "language": {c.language()}}
	path, yearParam := "/search/movie", "year"
	if kind == tvKind {
		path, yearParam = "/search/tv", "first_air_date_year"
	}
	if year != "" {
		q.Set(yearParam, year)
	}
	err := c.get(path, q, &r)
	if err == nil && len(r.Results) == 0 && year != "" {
		// Release years differ between countries; try once without the year.
		q.Del(yearParam)
		err = c.get(path, q, &r)
	}
	if kind == tvKind { // the same fields as movies from here on
		for i := range r.Results {
			x := &r.Results[i]
			x.Title, x.Original, x.ReleaseDate = x.Name, x.OriginalName, x.FirstAir
		}
	}
	return r, err
}

func (c *tmdbClient) details(id int) (MovieMeta, error) { return c.detailsKind("", id) }

func (c *tmdbClient) detailsKind(kind string, id int) (MovieMeta, error) {
	var d struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Name        string  `json:"name"`
		ReleaseDate string  `json:"release_date"`
		FirstAir    string  `json:"first_air_date"`
		Seasons     int     `json:"number_of_seasons"`
		Overview    string  `json:"overview"`
		VoteAverage float64 `json:"vote_average"`
		VoteCount   int     `json:"vote_count"`
		PosterPath  string  `json:"poster_path"`
		Backdrop    string  `json:"backdrop_path"`
		Genres      []struct {
			Name string `json:"name"`
		} `json:"genres"`
	}
	kindPath := "movie"
	if kind == tvKind {
		kindPath = "tv"
	}
	if err := c.get("/"+kindPath+"/"+strconv.Itoa(id), url.Values{"language": {c.language()}}, &d); err != nil {
		return MovieMeta{}, err
	}
	if kind == tvKind {
		d.Title, d.ReleaseDate = d.Name, d.FirstAir
	}
	m := MovieMeta{
		ID: d.ID, Title: d.Title, Year: yearOf(d.ReleaseDate), Overview: d.Overview,
		Rating: d.VoteAverage, Votes: d.VoteCount, Poster: d.PosterPath, Backdrop: d.Backdrop,
		URL: "https://www.themoviedb.org/" + kindPath + "/" + strconv.Itoa(d.ID), Fetched: time.Now().Unix(),
		Seasons: d.Seasons,
	}
	for _, g := range d.Genres {
		m.Genres = append(m.Genres, g.Name)
	}
	return m, nil
}

func yearOf(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

// bestMatch picks the search result whose title and year fit best.
func bestMatch(r tmdbSearchResult, name, year string) int {
	best, bestScore := 0, -1.0
	want := normTitle(name)
	for i, x := range r.Results {
		if i >= 10 {
			break
		}
		score := 0.0
		if normTitle(x.Title) == want || normTitle(x.Original) == want {
			score += 4
		}
		if year != "" && yearOf(x.ReleaseDate) == year {
			score += 3
		}
		score += 1.0 / float64(i+1) // TMDB's own ranking as the tie-breaker
		if score > bestScore {
			best, bestScore = x.ID, score
		}
	}
	return best
}

var reNonWord = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func normTitle(t string) string {
	return strings.TrimSpace(reNonWord.ReplaceAllString(strings.ToLower(t), " "))
}

// ---- Cache and the user's own matches ------------------------------------------

func tmdbCacheDir() string { return filepath.Join(filepath.Dir(posterCacheDir()), "tmdb") }

func titleKey(title string) string {
	h := sha1.Sum([]byte(normTitle(title)))
	return hex.EncodeToString(h[:10])
}

// Matches the user chose: title key -> TMDB id, or -1 for "use a frame from the movie".
type tmdbMatches struct {
	mu sync.Mutex
	m  map[string]int
}

func matchesPath() string { return filepath.Join(filepath.Dir(settingsPath()), "tmdb-matches.json") }

func loadMatches() *tmdbMatches {
	t := &tmdbMatches{m: map[string]int{}}
	if b, err := os.ReadFile(matchesPath()); err == nil {
		_ = json.Unmarshal(b, &t.m)
	}
	return t
}

func (t *tmdbMatches) get(title string) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id, ok := t.m[titleKey(title)]
	return id, ok
}

func (t *tmdbMatches) set(title string, id int) error {
	t.mu.Lock()
	if id == 0 {
		delete(t.m, titleKey(title))
	} else {
		t.m[titleKey(title)] = id
	}
	b, _ := json.MarshalIndent(t.m, "", "  ")
	t.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(matchesPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(matchesPath(), b, 0o644)
}

// metaFile is the cache file for a title in one language (plots and genres differ per language).
func metaFile(title, lang string) string {
	name := titleKey(title)
	if lang != "" && lang != "en-US" {
		name += "-" + lang
	}
	return filepath.Join(tmdbCacheDir(), name+".json")
}

func readMeta(title, lang string) (MovieMeta, bool) {
	var m MovieMeta
	b, err := os.ReadFile(metaFile(title, lang))
	if err != nil || json.Unmarshal(b, &m) != nil {
		return m, false
	}
	return m, time.Since(time.Unix(m.Fetched, 0)) < tmdbMaxAge
}

func writeMeta(title, lang string, m MovieMeta) {
	_ = os.MkdirAll(tmdbCacheDir(), 0o755)
	b, _ := json.Marshal(m)
	_ = os.WriteFile(metaFile(title, lang), b, 0o644)
}

// ClearOldTMDBCache removes cached TMDB data and images older than TMDB allows (6 months).
func clearOldTMDBCache() {
	entries, _ := os.ReadDir(tmdbCacheDir())
	for _, e := range entries {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > 170*24*time.Hour {
			_ = os.Remove(filepath.Join(tmdbCacheDir(), e.Name()))
		}
	}
}

// ---- The lookups the app uses -----------------------------------------------------

type TMDB struct {
	client   tmdbClient
	matches  *tmdbMatches
	inflight sync.Map // title key -> *sync.Mutex, so one title is looked up once
}

func newTMDB() *TMDB { return &TMDB{matches: loadMatches()} }

func (t *TMDB) setKey(k string) {
	t.client.mu.Lock()
	t.client.key = strings.TrimSpace(k)
	t.client.mu.Unlock()
}

// setLanguage picks the language for plots and genres ("fr" -> "fr-FR"; anything else is English).
func (t *TMDB) setLanguage(lang string) {
	l := map[string]string{"fr": "fr-FR"}[lang]
	t.client.mu.Lock()
	t.client.lang = l
	t.client.mu.Unlock()
}

func (t *TMDB) hasKey() bool {
	t.client.mu.Lock()
	defer t.client.mu.Unlock()
	return t.client.key != ""
}

// meta returns the TMDB info for a movie title like "Coyote Vs. Acme (2026)", or ok=false
// when the user chose a frame for it, or nothing was found.
func (t *TMDB) meta(title string) (MovieMeta, bool, error) { return t.metaKind("", title) }

// showMeta is meta for a TV show title like "WandaVision (2021)".
func (t *TMDB) showMeta(show string) (MovieMeta, bool, error) { return t.metaKind(tvKind, show) }

// kindTitle keeps TV shows apart from movies of the same name in the cache and the user's matches.
func kindTitle(kind, title string) string {
	if kind == tvKind {
		return "tv: " + title
	}
	return title
}

func (t *TMDB) metaKind(kind, plainTitle string) (MovieMeta, bool, error) {
	title := kindTitle(kind, plainTitle)
	id, chosen := t.matches.get(title)
	if chosen && id < 0 {
		return MovieMeta{}, false, nil
	}
	lk, _ := t.inflight.LoadOrStore(titleKey(title), &sync.Mutex{})
	lk.(*sync.Mutex).Lock()
	defer lk.(*sync.Mutex).Unlock()

	if m, fresh := readMeta(title, t.client.language()); fresh && (!chosen || m.ID == id) {
		return m, m.ID != 0, nil
	}
	if !t.hasKey() {
		return MovieMeta{}, false, fmt.Errorf("no TMDB key")
	}
	if !chosen {
		name, year := splitTitleYear(plainTitle)
		r, err := t.client.searchKind(kind, name, year)
		if err != nil {
			return MovieMeta{}, false, err
		}
		id = bestMatch(r, name, year)
		if id == 0 {
			writeMeta(title, t.client.language(), MovieMeta{Fetched: time.Now().Unix()}) // remember "not found" too
			return MovieMeta{}, false, nil
		}
	}
	m, err := t.client.detailsKind(kind, id)
	if err != nil {
		return MovieMeta{}, false, err
	}
	m.Manual = chosen
	writeMeta(title, t.client.language(), m)
	return m, true, nil
}

// season returns a show's episode names, plots and pictures for one season (cached like the rest).
func (t *TMDB) season(show string, n int) ([]EpisodeMeta, error) {
	m, ok, err := t.showMeta(show)
	if !ok {
		return []EpisodeMeta{}, err
	}
	file := metaFile(kindTitle(tvKind, show)+" #"+strconv.Itoa(m.ID)+" s"+strconv.Itoa(n), t.client.language())
	var c seasonCache
	if b, err := os.ReadFile(file); err == nil && json.Unmarshal(b, &c) == nil && time.Since(time.Unix(c.Fetched, 0)) < tmdbMaxAge {
		return c.Episodes, nil
	}
	eps, err := t.client.season(m.ID, n)
	if err != nil {
		return []EpisodeMeta{}, err
	}
	c = seasonCache{Fetched: time.Now().Unix(), Episodes: eps}
	_ = os.MkdirAll(tmdbCacheDir(), 0o755)
	b, _ := json.Marshal(c)
	_ = os.WriteFile(file, b, 0o644)
	return eps, nil
}

var reTitleYear = regexp.MustCompile(`^(.*?)\s*\((\d{4})\)$`)

func splitTitleYear(t string) (string, string) {
	if m := reTitleYear.FindStringSubmatch(t); m != nil {
		return m[1], m[2]
	}
	return t, ""
}

// image returns a TMDB image as a data: URL, cached on disk. size is TMDB's ("w342", "w1280", ...).
func (t *TMDB) image(path, size string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("no image")
	}
	name := strings.TrimPrefix(strings.ReplaceAll(path, "/", ""), ".")
	file := filepath.Join(tmdbCacheDir(), size+"-"+name)
	b, err := os.ReadFile(file)
	if err != nil {
		resp, err := tmdbHTTP.Get(tmdbImages + "/" + size + path)
		if err != nil {
			return "", fmt.Errorf("couldn't reach TMDB")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("TMDB image: %d", resp.StatusCode)
		}
		b, err = io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return "", err
		}
		_ = os.MkdirAll(tmdbCacheDir(), 0o755)
		_ = os.WriteFile(file, b, 0o644)
	}
	mime := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(path), ".png") {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// candidates is the "Not this movie?" list: TMDB's top 5 for the title, with small posters.
func (t *TMDB) candidates(title, query string) ([]TMDBCandidate, error) {
	return t.candidatesKind("", title, query)
}

func (t *TMDB) candidatesKind(kind, title, query string) ([]TMDBCandidate, error) {
	if !t.hasKey() {
		return nil, fmt.Errorf("no TMDB key")
	}
	name, year := splitTitleYear(title)
	if strings.TrimSpace(query) != "" {
		name, year = splitTitleYear(strings.TrimSpace(query))
	}
	r, err := t.client.searchKind(kind, name, year)
	if err != nil {
		return nil, err
	}
	out := []TMDBCandidate{}
	for _, x := range r.Results {
		if len(out) == 5 {
			break
		}
		c := TMDBCandidate{ID: x.ID, Title: x.Title, Year: yearOf(x.ReleaseDate)}
		if x.PosterPath != "" {
			c.Poster, _ = t.image(x.PosterPath, "w154")
		}
		out = append(out, c)
	}
	return out, nil
}

// checkKey asks TMDB whether a key works.
func checkTMDBKey(key string) error {
	c := &tmdbClient{key: strings.TrimSpace(key)}
	var out map[string]any
	return c.get("/configuration", nil, &out)
}
