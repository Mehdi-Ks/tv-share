package main

// What the page calls for TV shows (see shows.go for how episodes are recognised).

import (
	"fmt"
	"path/filepath"
	"strings"
)

// showOf returns the show an episode file belongs to, or "" for movies and extras.
func (a *App) showOf(video string) string {
	m := Movie{Path: video}
	classify(&m, a.Settings().Folder)
	if m.Kind != "episode" {
		return ""
	}
	return m.Show
}

// GetShowMeta returns TMDB's info for a show (ID 0 when off or not found).
func (a *App) GetShowMeta(show string) (MovieMeta, error) {
	if !a.tmdbOn() {
		return MovieMeta{}, nil
	}
	m, ok, err := a.tmdb.showMeta(show)
	if !ok {
		m = MovieMeta{}
		if _, chosen := a.tmdb.matches.get(kindTitle(tvKind, show)); chosen {
			m.Manual = true // "use a frame" was chosen for this show
		}
	}
	return m, err
}

// GetShowPoster returns the show's TMDB poster, or a frame from one of its episodes.
func (a *App) GetShowPoster(show, episode string) (string, error) {
	if a.tmdbOn() {
		if m, ok, _ := a.tmdb.showMeta(show); ok && m.Poster != "" {
			if img, err := a.tmdb.image(m.Poster, "w342"); err == nil {
				return img, nil
			}
		}
	}
	return posterDataURL(episode)
}

// GetShowBackdrop returns the show's wide TMDB picture, or a frame from one of its episodes.
func (a *App) GetShowBackdrop(show, episode string) (string, error) {
	if a.tmdbOn() {
		if m, ok, _ := a.tmdb.showMeta(show); ok && m.Backdrop != "" {
			if img, err := a.tmdb.image(m.Backdrop, "w1280"); err == nil {
				return img, nil
			}
		}
	}
	return backdropDataURL(episode)
}

// episodeBackdrop is the episode's own TMDB picture, else the show's.
func (a *App) episodeBackdrop(show, video string) (string, error) {
	if a.tmdbOn() {
		m := Movie{Path: video}
		classify(&m, a.Settings().Folder)
		if eps, err := a.tmdb.season(show, m.Season); err == nil {
			for _, e := range eps {
				if e.Episode == m.Episode && e.Still != "" {
					if img, err := a.tmdb.image(e.Still, "original"); err == nil {
						return img, nil
					}
				}
			}
		}
	}
	return a.GetShowBackdrop(show, video)
}

// GetSeason returns TMDB's episode names, plots and picture paths for one season ([] when off).
func (a *App) GetSeason(show string, season int) ([]EpisodeMeta, error) {
	if !a.tmdbOn() {
		return []EpisodeMeta{}, nil
	}
	return a.tmdb.season(show, season)
}

// GetEpisodeStill returns an episode picture from TMDB as a data: URL (path from GetSeason).
func (a *App) GetEpisodeStill(path string) (string, error) {
	if !a.tmdbOn() || !strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		return "", fmt.Errorf("no picture")
	}
	return a.tmdb.image(path, "w300")
}

// FindTMDBShowMatches lists TMDB's best guesses for "Not this show?". query may be empty.
func (a *App) FindTMDBShowMatches(show, query string) ([]TMDBCandidate, error) {
	return a.tmdb.candidatesKind(tvKind, show, query)
}

// SetTMDBShowMatch: id > 0 uses that TMDB show, -1 uses frames, 0 goes back to automatic.
func (a *App) SetTMDBShowMatch(show string, id int) error {
	return a.tmdb.matches.set(kindTitle(tvKind, show), id)
}

// SetKind marks files as a show's episodes ("show"), as movies ("movie"), or back to automatic ("").
func (a *App) SetKind(paths []string, kind string) error {
	if kind != "show" && kind != "movie" && kind != "" {
		return fmt.Errorf("unknown kind %q", kind)
	}
	root := a.Settings().Folder
	var kinds map[string]string
	err := a.updateSettings(func(s *Settings) {
		if s.Kinds == nil {
			s.Kinds = map[string]string{}
		}
		for _, p := range paths {
			lp := lowerPath(p)
			switch {
			case kind == "show" && s.Kinds[lp] == "movie":
				// Undoing "Show these as movies": bring back the whole show (every season), not just
				// this file. Files of the same show, or in the same folder, are switched back.
				dir := filepath.Dir(lp)
				key := ""
				if e, ok := parseEpisode(lp, lowerPath(root), false); ok {
					key = showKey(e.show)
				}
				for k, v := range s.Kinds {
					if v != "movie" {
						continue
					}
					if filepath.Dir(k) == dir {
						delete(s.Kinds, k)
					} else if e, ok := parseEpisode(k, lowerPath(root), false); key != "" && ok && showKey(e.show) == key {
						delete(s.Kinds, k)
					}
				}
			case kind == "":
				delete(s.Kinds, lp)
			default:
				s.Kinds[lp] = kind
			}
		}
		kinds = make(map[string]string, len(s.Kinds))
		for k, v := range s.Kinds {
			kinds[k] = v
		}
	})
	setKindOverrides(kinds)
	return err
}
