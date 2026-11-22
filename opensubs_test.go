package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The OpenSubtitles hash of a known pattern, computed independently.
func TestOSHash(t *testing.T) {
	p := filepath.Join(t.TempDir(), "movie.mkv")
	b := make([]byte, 200000)
	for i := range b {
		b[i] = byte(i * 7)
	}
	os.WriteFile(p, b, 0o644)
	h, err := osHash(p)
	if err != nil || len(h) != 16 {
		t.Fatalf("hash %q %v", h, err)
	}
	// Changing a byte in the middle (not read) keeps the hash; changing the start changes it.
	b[100000] ^= 0xff
	os.WriteFile(p, b, 0o644)
	if h2, _ := osHash(p); h2 != h {
		t.Fatal("middle of the file should not matter")
	}
	b[0] ^= 0xff
	os.WriteFile(p, b, 0o644)
	if h3, _ := osHash(p); h3 == h {
		t.Fatal("start of the file should matter")
	}
}

func fakeOS(t *testing.T, limit bool) {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file.srt" && (r.Header.Get("Api-Key") != "oskey" || !strings.HasPrefix(r.Header.Get("User-Agent"), "TVShare v")) {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/api/v1/subtitles":
			q := r.URL.Query()
			if q.Get("moviehash") == "" || q.Get("languages") != "en" || q.Get("query") != "test movie" || q.Get("year") != "2024" {
				t.Errorf("bad search query: %s", r.URL.RawQuery)
			}
			json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"attributes": map[string]any{"language": "en", "download_count": 900, "release": "Test.Movie.WEBRip",
					"moviehash_match": false, "files": []any{map[string]any{"file_id": 11}}}},
				map[string]any{"attributes": map[string]any{"language": "en", "download_count": 40, "release": "Test.Movie.BluRay",
					"moviehash_match": true, "hearing_impaired": true, "files": []any{map[string]any{"file_id": 22}}}},
			}})
		case "/api/v1/download":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["file_id"] == float64(1) {
				w.WriteHeader(406)
				w.Write([]byte(`{"message":"Invalid file_id"}`))
				return
			}
			if limit {
				w.WriteHeader(406)
				w.Write([]byte(`{"message":"You have downloaded your allowed 5 subtitles for 24h","reset_time":"5 hours"}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"link": srv.URL + "/file.srt", "remaining": 4})
		case "/file.srt":
			w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHi\n"))
		case "/api/v1/login":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["password"] != "right" {
				w.WriteHeader(401)
				w.Write([]byte(`{"message":"Error, invalid username/password"}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"token": "tok", "base_url": strings.TrimPrefix(srv.URL, "http://")})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	old := osAPI
	osAPI = srv.URL + "/api/v1"
	t.Cleanup(func() { osAPI = old })
}

func TestOpenSubtitlesSearchDownloadLogin(t *testing.T) {
	fakeOS(t, false)
	dir := t.TempDir()
	video := filepath.Join(dir, "Test.Movie.2024.mkv")
	os.WriteFile(video, make([]byte, 200000), 0o644)
	c := &osClient{key: "oskey"}

	res, err := c.search(video, "Test Movie", "2024", "eng", 0, 0)
	if err != nil || len(res) != 2 {
		t.Fatalf("search: %v %+v", err, res)
	}
	if !res[0].Exact || res[0].FileID != 22 || !res[0].HI || res[0].LangLabel != "English" || res[0].Language != "eng" {
		t.Fatalf("exact match should come first: %+v", res[0])
	}

	p, left, err := c.download(22, video, "eng")
	if err != nil || left != 4 || filepath.Base(p) != "Test.Movie.2024.eng.srt" {
		t.Fatalf("download: %q %d %v", p, left, err)
	}
	// It must be found as a subtitle file of this movie afterwards.
	if subs := findSubs(video); len(subs) != 1 || subs[0].Lang != "eng" {
		t.Fatalf("not picked up: %+v", subs)
	}
	// A second download doesn't overwrite the first.
	if p2, _, _ := c.download(22, video, "eng"); filepath.Base(p2) != "Test.Movie.2024.eng.2.srt" {
		t.Fatalf("second download: %q", p2)
	}

	if err := c.login("me", "wrong"); err == nil || !strings.Contains(err.Error(), "username or password") {
		t.Fatalf("bad login: %v", err)
	}
	if err := c.login("me", "right"); err != nil || c.token != "tok" || !strings.HasSuffix(c.base, "/api/v1") {
		t.Fatalf("login: %v %+v", err, c)
	}
}

func TestOpenSubtitlesLimit(t *testing.T) {
	fakeOS(t, true)
	video := filepath.Join(t.TempDir(), "m.mkv")
	os.WriteFile(video, make([]byte, 200000), 0o644)
	_, _, err := (&osClient{key: "oskey"}).download(22, video, "eng")
	if _, ok := err.(errOSLimit); !ok || !strings.Contains(err.Error(), "5 hours") || !strings.Contains(err.Error(), "sign in") {
		t.Fatalf("limit: %v", err)
	}
	if _, _, err := (&osClient{key: "nope"}).download(22, video, "eng"); err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("bad key: %v", err)
	}
	if err := checkOSKey("oskey"); err != nil {
		t.Fatalf("good key rejected: %v", err)
	}
	if err := checkOSKey("nope"); err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("bad key accepted: %v", err)
	}
}
