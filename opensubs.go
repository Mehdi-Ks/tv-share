package main

// Subtitles from OpenSubtitles.com, only when the user clicks "Find subtitles online".
// A search sends the movie file's fingerprint (the OpenSubtitles hash: file size plus the first
// and last 64 KB) and its title and year; nothing else leaves the PC. Results made for this exact
// file come first: they're already in sync.
//
// Release builds get an API key at build time (build.ps1 reads build/opensubtitles.key, never
// committed). People can also paste their own key, and sign in with a free OpenSubtitles account
// for more downloads per day; only the login token is kept, never the password.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var osBuiltInKey = "" // set by the release build, see build.ps1

var (
	osAPI     = "https://api.opensubtitles.com/api/v1"
	osHTTP    = &http.Client{Timeout: 20 * time.Second}
	osSignup  = "https://www.opensubtitles.com/en/users/sign_up"
	osKeyPage = "https://www.opensubtitles.com/en/consumers"
)

// osHash is OpenSubtitles' file fingerprint: size + 64-bit little-endian sum of the first and last 64 KB.
func osHash(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	const chunk = 65536
	size := st.Size()
	if size < chunk*2 {
		return "", fmt.Errorf("file too small")
	}
	h := uint64(size)
	buf := make([]byte, chunk)
	for _, off := range []int64{0, size - chunk} {
		if _, err := f.ReadAt(buf, off); err != nil {
			return "", err
		}
		for i := 0; i < chunk; i += 8 {
			h += binary.LittleEndian.Uint64(buf[i:])
		}
	}
	return fmt.Sprintf("%016x", h), nil
}

// OpenSubtitles uses two-letter codes (and a few regional ones).
var osLangCodes = map[string]string{
	"eng": "en", "fre": "fr", "spa": "es", "ger": "de", "ita": "it", "por": "pt-PT", "ara": "ar", "dut": "nl",
	"tur": "tr", "pol": "pl", "rus": "ru", "ukr": "uk", "rum": "ro", "gre": "el", "swe": "sv", "nor": "no",
	"dan": "da", "fin": "fi", "cze": "cs", "slo": "sk", "slv": "sl", "hrv": "hr", "srp": "sr", "bul": "bg",
	"hun": "hu", "est": "et", "lav": "lv", "lit": "lt", "ice": "is", "heb": "he", "hin": "hi", "ind": "id",
	"may": "ms", "tha": "th", "vie": "vi", "chi": "zh-CN", "jpn": "ja", "kor": "ko", "per": "fa",
}

func osLangToOurs(code string) string {
	code = strings.ToLower(code)
	for ours, theirs := range osLangCodes {
		if strings.EqualFold(theirs, code) || strings.HasPrefix(code, theirs+"-") {
			return ours
		}
	}
	if code == "pt-br" || code == "pb" {
		return "por"
	}
	return ""
}

// OSResult is one subtitle choice in "Find subtitles online".
type OSResult struct {
	FileID    int    `json:"fileId"`
	Language  string `json:"language"`  // our 3-letter code, "" if unknown
	LangLabel string `json:"langLabel"` // "English"
	Release   string `json:"release"`   // the release it was made for
	Exact     bool   `json:"exact"`     // made for this exact file (hash match)
	Downloads int    `json:"downloads"`
	HI        bool   `json:"hi"` // for the hard of hearing
}

// OSStatus is what Settings shows.
type OSStatus struct {
	Available bool   `json:"available"` // a key is built in or was entered
	BuiltIn   bool   `json:"builtIn"`
	OwnKey    bool   `json:"ownKey"`
	User      string `json:"user"`      // signed-in username, "" if not signed in
	Remaining int    `json:"remaining"` // downloads left today, -1 if unknown
	ResetIn   string `json:"resetIn"`   // e.g. "5 hours", "" if unknown
}

type osClient struct {
	key, token, base, user string
}

func (c *osClient) baseURL() string {
	if c.base != "" {
		return c.base
	}
	return osAPI
}

func (c *osClient) do(method, path string, q url.Values, body any, out any) (int, error) {
	u := c.baseURL() + path
	if len(q) > 0 {
		u += "?" + strings.ReplaceAll(q.Encode(), "%20", "+") // Encode sorts keys, as OpenSubtitles asks
	}
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Api-Key", c.key)
	req.Header.Set("User-Agent", "TVShare v"+appVersion)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := osHTTP.Do(req)
	if err != nil {
		return 0, fmt.Errorf("couldn't reach OpenSubtitles (are you online?)")
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		return resp.StatusCode, osError(resp.StatusCode, b)
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			return resp.StatusCode, fmt.Errorf("OpenSubtitles gave an unexpected answer")
		}
	}
	return resp.StatusCode, nil
}

// errOSLimit means today's downloads are used up.
type errOSLimit struct{ resetIn string }

func (e errOSLimit) Error() string {
	if e.resetIn != "" {
		return "You've used today's free subtitle downloads. More in " + e.resetIn + ", or sign in to OpenSubtitles for more."
	}
	return "You've used today's free subtitle downloads. Sign in to OpenSubtitles for more."
}

func osError(status int, body []byte) error {
	var m struct {
		Message string `json:"message"`
		Errors  []string
		Reset   string `json:"reset_time"`
	}
	_ = json.Unmarshal(body, &m)
	switch {
	case status == 401 && strings.Contains(strings.ToLower(m.Message), "invalid"):
		return fmt.Errorf("OpenSubtitles didn't accept that username or password")
	case status == 401 || status == 403:
		return fmt.Errorf("OpenSubtitles didn't accept the key")
	case status == 406 || status == 429 && strings.Contains(strings.ToLower(m.Message), "download"):
		return errOSLimit{resetIn: m.Reset}
	case status == 429:
		return fmt.Errorf("OpenSubtitles is busy, try again in a few seconds")
	}
	if m.Message != "" {
		return fmt.Errorf("OpenSubtitles: %s", m.Message)
	}
	return fmt.Errorf("OpenSubtitles answered %d", status)
}

// search lists subtitles for a movie file, exact matches first.
// search lists subtitles for a movie, or for one episode when season and episode are set.
func (c *osClient) search(video, title, year, lang string, season, episode int) ([]OSResult, error) {
	q := url.Values{}
	if l := osLangCodes[lang]; l != "" {
		q.Set("languages", strings.ToLower(l))
	}
	hash, _ := osHash(video)
	if hash != "" {
		q.Set("moviehash", hash)
	}
	q.Set("query", strings.ToLower(title))
	if year != "" && episode == 0 {
		q.Set("year", year)
	}
	if episode > 0 {
		q.Set("season_number", strconv.Itoa(season))
		q.Set("episode_number", strconv.Itoa(episode))
	}
	var r struct {
		Data []struct {
			Attributes struct {
				Language      string `json:"language"`
				DownloadCount int    `json:"download_count"`
				HI            bool   `json:"hearing_impaired"`
				Release       string `json:"release"`
				HashMatch     bool   `json:"moviehash_match"`
				Files         []struct {
					FileID int `json:"file_id"`
				} `json:"files"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if _, err := c.do("GET", "/subtitles", q, nil, &r); err != nil {
		return nil, err
	}
	out := []OSResult{}
	for _, d := range r.Data {
		a := d.Attributes
		if len(a.Files) == 0 {
			continue
		}
		ours := osLangToOurs(a.Language)
		label := langName(ours)
		if label == "" {
			label = strings.ToUpper(a.Language)
		}
		out = append(out, OSResult{FileID: a.Files[0].FileID, Language: ours, LangLabel: label,
			Release: a.Release, Exact: a.HashMatch, Downloads: a.DownloadCount, HI: a.HI})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Exact != out[j].Exact {
			return out[i].Exact
		}
		return out[i].Downloads > out[j].Downloads
	})
	if len(out) > 20 {
		out = out[:20]
	}
	return out, nil
}

// download saves one subtitle file next to the movie and returns its path and downloads left.
func (c *osClient) download(fileID int, video, lang string) (string, int, error) {
	var r struct {
		Link      string `json:"link"`
		Remaining int    `json:"remaining"`
		Reset     string `json:"reset_time"`
	}
	if _, err := c.do("POST", "/download", nil, map[string]any{"file_id": fileID, "sub_format": "srt"}, &r); err != nil {
		return "", -1, err
	}
	if r.Link == "" {
		return "", r.Remaining, errOSLimit{resetIn: r.Reset}
	}
	resp, err := osHTTP.Get(r.Link)
	if err != nil {
		return "", r.Remaining, fmt.Errorf("couldn't download the subtitles")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", r.Remaining, fmt.Errorf("couldn't download the subtitles (%d)", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil || len(b) == 0 {
		return "", r.Remaining, fmt.Errorf("couldn't download the subtitles")
	}
	base := strings.TrimSuffix(video, filepath.Ext(video))
	suffix := lang
	if suffix == "" {
		suffix = "online"
	}
	out := base + "." + suffix + ".srt"
	for i := 2; fileExists(out); i++ {
		out = fmt.Sprintf("%s.%s.%d.srt", base, suffix, i)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		return "", r.Remaining, err
	}
	return out, r.Remaining, nil
}

// login signs in with a free OpenSubtitles account; only the token is kept.
func (c *osClient) login(user, pass string) error {
	var r struct {
		Token   string `json:"token"`
		BaseURL string `json:"base_url"`
	}
	if _, err := c.do("POST", "/login", nil, map[string]string{"username": user, "password": pass}, &r); err != nil {
		return err
	}
	if r.Token == "" {
		return fmt.Errorf("OpenSubtitles didn't accept that username or password")
	}
	c.token, c.user = r.Token, user
	c.base = ""
	if r.BaseURL != "" {
		b := r.BaseURL
		if !strings.HasPrefix(b, "http") {
			b = "https://" + b
		}
		c.base = strings.TrimSuffix(b, "/") + "/api/v1"
	}
	return nil
}

func (c *osClient) logout() {
	if c.token != "" {
		_, _ = c.do("DELETE", "/logout", nil, nil, nil)
	}
	c.token, c.user, c.base = "", "", ""
}

// remaining asks how many downloads are left today (signed-in users only).
func (c *osClient) remaining() (int, string) {
	if c.token == "" {
		return -1, ""
	}
	var r struct {
		Data struct {
			Remaining int    `json:"remaining_downloads"`
			Reset     string `json:"reset_time"`
		} `json:"data"`
	}
	if _, err := c.do("GET", "/infos/user", nil, nil, &r); err != nil {
		return -1, ""
	}
	return r.Data.Remaining, r.Data.Reset
}

// checkOSKey asks OpenSubtitles whether a key works. Searches answer even with a wrong key
// (they're cached), so this asks to download a file that doesn't exist: a good key gets
// "invalid file" (406, nothing is downloaded or counted), a wrong one gets 401/403.
func checkOSKey(key string) error {
	c := &osClient{key: strings.TrimSpace(key)}
	status, err := c.do("POST", "/download", nil, map[string]any{"file_id": 1}, nil)
	if status == 406 {
		return nil
	}
	if err == nil {
		return nil
	}
	return err
}
