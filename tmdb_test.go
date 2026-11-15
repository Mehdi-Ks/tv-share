package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeTMDB answers like TMDB for two movies, and counts the calls it gets.
func fakeTMDB(t *testing.T) *int32 {
	t.Helper()
	t.Setenv("LOCALAPPDATA", t.TempDir()) // poster/TMDB cache
	t.Setenv("APPDATA", t.TempDir())      // settings and chosen matches
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Query().Get("api_key") != "testkey" {
			w.WriteHeader(401)
			return
		}
		switch {
		case r.URL.Path == "/3/search/movie":
			q := r.URL.Query().Get("query")
			res := []map[string]any{}
			if strings.Contains(strings.ToLower(q), "coyote") {
				res = append(res,
					map[string]any{"id": 11, "title": "Coyote Ugly", "release_date": "2000-08-04", "poster_path": "/ugly.jpg"},
					map[string]any{"id": 22, "title": "Coyote vs. Acme", "release_date": "2026-08-28", "poster_path": "/acme.jpg"})
			}
			json.NewEncoder(w).Encode(map[string]any{"results": res})
		case r.URL.Path == "/3/movie/22" || r.URL.Path == "/3/movie/11":
			id := strings.TrimPrefix(r.URL.Path, "/3/movie/")
			json.NewEncoder(w).Encode(map[string]any{
				"id": map[string]int{"22": 22, "11": 11}[id], "title": map[string]string{"22": "Coyote vs. Acme", "11": "Coyote Ugly"}[id],
				"release_date": "2026-08-28", "overview": "A lawsuit.", "vote_average": 7.4, "vote_count": 120,
				"poster_path": "/acme.jpg", "backdrop_path": "/wide.jpg", "genres": []map[string]string{{"name": "Comedy"}},
			})
		case r.URL.Path == "/3/configuration":
			w.Write([]byte(`{"images":{}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	img := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte("\xff\xd8fakejpeg"))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(img.Close)
	oldAPI, oldImg := tmdbAPI, tmdbImages
	tmdbAPI, tmdbImages = srv.URL+"/3", img.URL
	t.Cleanup(func() { tmdbAPI, tmdbImages = oldAPI, oldImg })
	return &calls
}

func TestTMDBMatchCacheAndOverride(t *testing.T) {
	calls := fakeTMDB(t)
	tm := newTMDB()
	tm.setKey("testkey")

	// Picks the title that fits (not TMDB's first result), and fetches details.
	m, ok, err := tm.meta("Coyote Vs. Acme (2026)")
	if err != nil || !ok || m.ID != 22 || m.Overview != "A lawsuit." || len(m.Genres) != 1 || m.Poster != "/acme.jpg" {
		t.Fatalf("meta = %+v %v %v", m, ok, err)
	}
	// Second time comes from the cache: no new calls.
	before := atomic.LoadInt32(calls)
	if m2, ok, _ := tm.meta("Coyote Vs. Acme (2026)"); !ok || m2.ID != 22 || m2.Backdrop != "/wide.jpg" {
		t.Fatalf("cached meta = %+v", m2)
	}
	if atomic.LoadInt32(calls) != before {
		t.Fatalf("cache missed")
	}
	// Images are cached too.
	if u, err := tm.image(m.Poster, "w342"); err != nil || !strings.HasPrefix(u, "data:image/jpeg;base64,") {
		t.Fatalf("image: %v", err)
	}
	before = atomic.LoadInt32(calls)
	tm.image(m.Poster, "w342")
	if atomic.LoadInt32(calls) != before {
		t.Fatalf("image cache missed")
	}

	// "Not this movie?": choosing another one sticks.
	if err := tm.matches.set("Coyote Vs. Acme (2026)", 11); err != nil {
		t.Fatal(err)
	}
	if m, ok, _ := tm.meta("Coyote Vs. Acme (2026)"); !ok || m.ID != 11 || !m.Manual {
		t.Fatalf("override = %+v", m)
	}
	// Saved to disk: a fresh start remembers it.
	if id, ok := loadMatches().get("Coyote Vs. Acme (2026)"); !ok || id != 11 {
		t.Fatalf("saved match = %d %v", id, ok)
	}
	// "Use a frame from the movie" turns TMDB off for this title.
	tm.matches.set("Coyote Vs. Acme (2026)", -1)
	if _, ok, _ := tm.meta("Coyote Vs. Acme (2026)"); ok {
		t.Fatal("frame choice ignored")
	}

	// Nothing found is fine, and remembered.
	if _, ok, err := tm.meta("Home Video (2019)"); ok || err != nil {
		t.Fatalf("not found: %v %v", ok, err)
	}

	// Candidates for the picker.
	c, err := tm.candidates("Coyote Vs. Acme (2026)", "")
	if err != nil || len(c) != 2 || c[1].ID != 22 || c[1].Poster == "" {
		t.Fatalf("candidates = %+v %v", c, err)
	}
}

func TestTMDBKeyChecks(t *testing.T) {
	fakeTMDB(t)
	if err := checkTMDBKey("testkey"); err != nil {
		t.Fatalf("good key rejected: %v", err)
	}
	if err := checkTMDBKey("wrong"); err == nil || !strings.Contains(err.Error(), "didn't accept") {
		t.Fatalf("bad key: %v", err)
	}
	tm := newTMDB()
	if _, ok, err := tm.meta("Anything (2020)"); ok || err == nil {
		t.Fatal("lookup without a key should fail quietly")
	}
}

func TestNormTitle(t *testing.T) {
	if normTitle("Coyote Vs. Acme") != normTitle("Coyote vs Acme") {
		t.Fatal("punctuation should not matter")
	}
	if n, y := splitTitleYear("Les Baronnes (2025)"); n != "Les Baronnes" || y != "2025" {
		t.Fatalf("split: %q %q", n, y)
	}
}
