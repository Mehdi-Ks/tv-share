package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEpisode(t *testing.T) {
	root := `C:\Videos`
	cases := []struct {
		path       string
		ok         bool
		show       string
		season, ep int
		title      string
	}{
		{`C:\Videos\WandaVision (2021) Season 1 S01 (1080p BluRay x265 HEVC 10bit EAC3 5.1 Silence)\WandaVision (2021) - S01E03 - Now in Color (1080p BluRay x265 Silence).mkv`, true, "WandaVision (2021)", 1, 3, "Now in Color"},
		{`C:\Videos\The.Office.US.S02E03.The.Dundies.720p.WEB.x264.mkv`, true, "The Office US", 2, 3, "The Dundies"},
		{`C:\Videos\Severance\severance.s01e01.1080p.web.h264.mkv`, true, "severance", 1, 1, ""},
		{`C:\Videos\Friends\friends.3x07.the.one.with.the.race.car.bed.avi`, true, "friends", 3, 7, "the one with the race car bed"},
		{`C:\Videos\Breaking Bad\Season 2\03 - Bit by a Dead Bee.mkv`, true, "Breaking Bad", 2, 3, "Bit by a Dead Bee"},
		{`C:\Videos\Dark\Saison 1\Dark - Episode 4.mkv`, true, "Dark", 1, 4, ""},
		{`C:\Videos\Show.S01E01E02.Pilot.mkv`, true, "Show", 1, 1, "Pilot"},
		// Movies must stay movies.
		{`C:\Videos\Dune (2021) [1080p] [BluRay] [x265] [10bit] [5.1] [YTS.MX]\Dune (2021).mkv`, false, "", 0, 0, ""},
		{`C:\Videos\Dune.Part.Two.2024.REPACK.1080p.WEBRip.x265.10bit.AAC5.1-[YTS.MX].mp4`, false, "", 0, 0, ""},
		{`C:\Videos\Blade Runner 2049 (2017)\Blade.Runner.2049.2017.1080p.mkv`, false, "", 0, 0, ""},
		{`C:\Videos\Se7en (1995).mkv`, false, "", 0, 0, ""},
		{`C:\Videos\2001 A Space Odyssey (1968).mkv`, false, "", 0, 0, ""},
		{`C:\Videos\Movie.1920x1080.mkv`, false, "", 0, 0, ""},
	}
	for _, c := range cases {
		e, ok := parseEpisode(c.path, root, false)
		if ok != c.ok || (ok && (e.show != c.show || e.season != c.season || e.episode != c.ep || e.title != c.title)) {
			t.Errorf("%s\n  got ok=%v %+v", filepath.Base(c.path), ok, e)
		}
	}
}

func TestShowsAndExtrasScan(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, minVideoBytes)
	small := make([]byte, 2<<20)
	write := func(rel string, b []byte) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, b, 0o644)
	}
	show := `WandaVision (2021) Season 1 S01 (1080p BluRay x265)`
	write(show+`\WandaVision (2021) - S01E01 - Filmed Before a Live Studio Audience (1080p).mkv`, big)
	write(show+`\WandaVision (2021) - S01E02 - Don't Touch That Dial (1080p).mkv`, big)
	write(show+`\Featurettes\Gag Reel.mkv`, small)
	write(show+`\Featurettes\Sample.mkv`, small)
	write(`Dune (2021)\Dune (2021).mkv`, big)
	write(`Dune (2021)\Extras\Behind.the.Scenes.1080p.mkv`, small)
	write(`Some Movie (2020)\Some Movie (2020).mkv`, big)
	write(`Some Movie (2020)\sample\some-movie-sample.mkv`, big)

	items := scanMovies(root, "eng")
	byKind := map[string][]Movie{}
	for _, m := range items {
		byKind[m.Kind] = append(byKind[m.Kind], m)
	}
	if len(byKind["episode"]) != 2 || len(byKind["movie"]) != 2 || len(byKind["extra"]) != 2 {
		for _, m := range items {
			t.Logf("%s %s %q %q", m.Kind, filepath.Base(m.Path), m.Title, m.ExtraFor)
		}
		t.Fatalf("kinds: %d episodes, %d movies, %d extras", len(byKind["episode"]), len(byKind["movie"]), len(byKind["extra"]))
	}
	for _, x := range byKind["extra"] {
		switch x.Title {
		case "Gag Reel":
			if x.ExtraFor != "show:wandavision" {
				t.Errorf("gag reel belongs to %q", x.ExtraFor)
			}
		case "Behind the Scenes":
			if filepath.Base(x.ExtraFor) != "Dune (2021).mkv" {
				t.Errorf("behind the scenes belongs to %q", x.ExtraFor)
			}
		default:
			t.Errorf("unexpected extra %q", x.Title)
		}
	}
	if e := byKind["episode"][0]; e.Show != "WandaVision (2021)" || e.ShowKey != "wandavision" {
		t.Errorf("episode: %+v", e)
	}
}

func TestKindOverrides(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, minVideoBytes)
	a := filepath.Join(root, "My Show", "first one.mkv")
	b := filepath.Join(root, "My Show", "second one.mkv")
	os.MkdirAll(filepath.Dir(a), 0o755)
	os.WriteFile(a, big, 0o644)
	os.WriteFile(b, big, 0o644)
	ep := filepath.Join(root, "Show.S01E01.mkv")
	os.WriteFile(ep, big, 0o644)

	setKindOverrides(map[string]string{lowerPath(a): "show", lowerPath(b): "show", lowerPath(ep): "movie"})
	defer setKindOverrides(nil)
	for _, m := range scanMovies(root, "eng") {
		switch m.Path {
		case a, b:
			want := map[string]int{a: 1, b: 2}[m.Path]
			if m.Kind != "episode" || m.Show != "My Show" || m.Season != 1 || m.Episode != want {
				t.Errorf("forced show: %+v", m)
			}
		case ep:
			if m.Kind != "movie" {
				t.Errorf("forced movie: %+v", m)
			}
		}
	}
}

func TestSetKindUndoWholeFolder(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir()) // settings go to a temporary folder
	a := &App{settings: Settings{}}
	dir := filepath.Join(t.TempDir(), "Show")
	e1, e2 := filepath.Join(dir, "Show.S01E01.mkv"), filepath.Join(dir, "Show.S01E02.mkv")
	if err := a.SetKind([]string{e1, e2}, "movie"); err != nil {
		t.Fatal(err)
	}
	if kindOverride(e1) != "movie" || kindOverride(e2) != "movie" {
		t.Fatal("not marked as movies")
	}
	if err := a.SetKind([]string{e1}, "show"); err != nil {
		t.Fatal(err)
	}
	if kindOverride(e1) != "" || kindOverride(e2) != "" {
		t.Fatalf("undo should clear the whole folder: %q %q", kindOverride(e1), kindOverride(e2))
	}
	setKindOverrides(nil)
}

func TestTrickyLayouts(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, minVideoBytes)
	small := make([]byte, 2<<20)
	write := func(rel string, b []byte) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, b, 0o644)
	}
	write(`The Office (US)\Season 1\The.Office.US.S01E01.Pilot.mkv`, big)
	write(`The Office (US)\Season 2\The.Office.US.S02E01.The.Dundies.mkv`, big)
	write(`The Office (US)\Specials\The.Office.US.S00E01.Webisode.mkv`, big)
	write(`Severance S01\Severance.S01E01.mkv`, big)
	write(`Severance (2022) Season 2\Severance (2022) - S02E01 - Hello Ms Cobel.mkv`, big)
	write(`Doctor.Who.2005.S01E01.Rose.mkv`, big) // a loose episode right in the folder
	write(`Other\Heat (1995).mkv`, big)           // the user's own folders, not extras
	write(`Shorts\Paperman.mkv`, big)
	write(`Movies\Arrival (2016).mkv`, big)
	write(`Movies\Other\Sicario (2015).mkv`, big) // one movie next to it, but not its folder
	write(`Dark\Season 1\Dark.S01E01.mkv`, big)
	write(`Dark\Season 1\Featurettes\Making of.mkv`, small)
	write(`Season of the Witch (2011).mkv`, big)

	got := map[string]Movie{}
	for _, m := range scanMovies(root, "eng") {
		rel, _ := filepath.Rel(root, m.Path)
		got[rel] = m
	}
	want := map[string]string{ // file -> kind / show key / extra's owner
		`The Office (US)\Season 1\The.Office.US.S01E01.Pilot.mkv`:                  "episode the office us",
		`The Office (US)\Season 2\The.Office.US.S02E01.The.Dundies.mkv`:            "episode the office us",
		`The Office (US)\Specials\The.Office.US.S00E01.Webisode.mkv`:               "episode the office us",
		`Severance S01\Severance.S01E01.mkv`:                                       "episode severance",
		`Severance (2022) Season 2\Severance (2022) - S02E01 - Hello Ms Cobel.mkv`: "episode severance",
		`Doctor.Who.2005.S01E01.Rose.mkv`:                                          "episode doctor who",
		`Other\Heat (1995).mkv`:                                                    "movie ",
		`Shorts\Paperman.mkv`:                                                      "movie ",
		`Movies\Arrival (2016).mkv`:                                                "movie ",
		`Movies\Other\Sicario (2015).mkv`:                                          "movie ",
		`Dark\Season 1\Dark.S01E01.mkv`:                                            "episode dark",
		`Dark\Season 1\Featurettes\Making of.mkv`:                                  "extra show:dark",
		`Season of the Witch (2011).mkv`:                                           "movie ",
	}
	for rel, w := range want {
		m, ok := got[rel]
		if !ok {
			t.Errorf("%s: not found", rel)
			continue
		}
		s := m.Kind + " " + m.ShowKey
		if m.Kind == "extra" {
			s = m.Kind + " " + m.ExtraFor
		}
		if s != w {
			t.Errorf("%s: got %q, want %q", rel, s, w)
		}
	}
	if s := got[`Severance S01\Severance.S01E01.mkv`].Show; s != "Severance (2022)" {
		t.Errorf("both seasons should share one name, got %q", s)
	}
}

func TestSetKindUndoAllSeasons(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	root := t.TempDir()
	a := &App{settings: Settings{Folder: root}}
	e1 := filepath.Join(root, "Show", "Season 1", "Show.S01E01.mkv")
	e2 := filepath.Join(root, "Show", "Season 2", "Show.S02E01.mkv")
	other := filepath.Join(root, "Other Show", "Other.Show.S01E01.mkv")
	if err := a.SetKind([]string{e1, e2, other}, "movie"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetKind([]string{e1}, "show"); err != nil {
		t.Fatal(err)
	}
	if kindOverride(e1) != "" || kindOverride(e2) != "" {
		t.Errorf("undo should bring back every season: %q %q", kindOverride(e1), kindOverride(e2))
	}
	if kindOverride(other) != "movie" {
		t.Errorf("another show must stay as movies")
	}
	setKindOverrides(nil)
}

func TestSeasonFolderNeverNamesAMovie(t *testing.T) {
	root := t.TempDir()
	for rel, want := range map[string]string{
		`Show\Season 2\Show.S02E01.mkv`: "Show S02E01",
		`Show\Specials\Show.S00E01.mkv`: "Show S00E01",
		`Heat (1995)\heat.1080p.mkv`:    "Heat (1995)",
	} {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, make([]byte, minVideoBytes), 0o644)
		if got := movieTitle(p, root); got != want {
			t.Errorf("%s: got %q, want %q", rel, got, want)
		}
	}
}
