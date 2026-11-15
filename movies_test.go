package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		"Some Movie (2026) [1080p] [WEBRip] [x265] [GROUP]":     "Some Movie (2026)",
		"Some.Movie.2026.1080p.WEBRip.x265.10bit.AAC5.1-[GROUP]": "Some Movie (2026)",
		"Les.Baronnes.2025.French.WEBRIP":                        "Les Baronnes (2025)",
		"my_holiday_video":                                       "my holiday video",
		"Blade Runner 2049 (2017)":                               "Blade Runner 2049 (2017)",
		"2001.A.Space.Odyssey.1968.1080p":                        "2001 A Space Odyssey (1968)",
	}
	for in, want := range cases {
		if got := cleanTitle(in); got != want {
			t.Errorf("cleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDescribeSub(t *testing.T) {
	cases := []struct {
		name, label, code string
		sdh               bool
	}{
		{"English", "English", "eng", false},
		{"SDH.eng.HI", "English · for the hard of hearing", "eng", true},
		{"British.eng", "English · British", "eng", false},
		{"Latin American.spa", "Spanish · Latin American", "spa", false},
		{"fre", "French", "fre", false},
		{"Movie.2026.1080p", "Subtitles (same name as the movie)", "", false},
		{"Movie.2026.1080p.fr", "French", "fre", false},
	}
	for _, c := range cases {
		label, code, sdh := describeSub(c.name, "Movie.2026.1080p")
		if label != c.label || code != c.code || sdh != c.sdh {
			t.Errorf("describeSub(%q) = %q,%q,%v want %q,%q,%v", c.name, label, code, sdh, c.label, c.code, c.sdh)
		}
	}
}

func TestSortSubsPrefersPlainPreferredLanguage(t *testing.T) {
	subs := []SubOption{
		{Label: "Arabic", Lang: "ara"},
		{Label: "English · for the hard of hearing", Lang: "eng", SDH: true},
		{Label: "Subtitles (same name as the movie)"},
		{Label: "English", Lang: "eng"},
	}
	sortSubs(subs, "eng")
	if subs[0].Label != "English" || subs[1].SDH != true || subs[3].Lang != "" {
		t.Fatalf("unexpected order: %+v", subs)
	}
}

// End to end on a generated 6-second clip: needs ffmpeg/ffprobe installed.
func TestPrepareEndToEnd(t *testing.T) {
	ffmpeg := toolFFmpeg.find()
	if ffmpeg == "" || toolFFprobe.find() == "" {
		t.Skip("ffmpeg not installed")
	}
	dir := filepath.Join(t.TempDir(), "Test Film (2024) [1080p] [x265]")
	if err := os.MkdirAll(filepath.Join(dir, "Subs"), 0o755); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(dir, "Test.Film.2024.1080p.mp4")
	out, err := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=25:duration=6",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest", video).CombinedOutput()
	if err != nil {
		t.Fatalf("making test clip: %v %s", err, out)
	}
	srt := "1\n00:00:01,000 --> 00:00:03,000\nHello\n\n2\n00:00:03,500 --> 00:00:05,000\nCafé crème\n"
	os.WriteFile(filepath.Join(dir, "Subs", "English.srt"), []byte(srt), 0o644)
	// Windows-1252 file (é = 0xE9) to exercise the encoding fallback
	os.WriteFile(filepath.Join(dir, "Subs", "fre.srt"), []byte("1\n00:00:01,000 --> 00:00:03,000\nCaf\xe9\n"), 0o644)

	subs := findSubs(video)
	sortSubs(subs, "fre")
	if len(subs) != 2 || subs[0].Lang != "fre" {
		t.Fatalf("findSubs: %+v", subs)
	}
	title := movieTitle(video, filepath.Dir(dir))
	if title != "Test Film (2024)" {
		t.Fatalf("title = %q", title)
	}

	// A lone movie in a generically named folder takes its title from the file
	generic := filepath.Join(filepath.Dir(dir), "New folder")
	os.MkdirAll(generic, 0o755)
	lone := filepath.Join(generic, "Les.Baronnes.2025.French.WEBRIP.mp4")
	if f, err := os.Create(lone); err == nil {
		f.Truncate(minVideoBytes + 1) // big enough to count as a movie
		f.Close()
	}
	if got := movieTitle(lone, filepath.Dir(dir)); got != "Les Baronnes (2025)" {
		t.Fatalf("generic folder title = %q", got)
	}

	var last float64
	res, err := (&Preparer{}).Prepare(video, subs[0], title, false, func(p float64) { last = p })
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if filepath.Base(res.Output) != "Test Film (2024).mkv" || last != 100 {
		t.Fatalf("output %q, progress %v", res.Output, last)
	}
	if !hasEmbeddedSubs(res.Output) {
		t.Fatal("no subtitle track in output")
	}
	if !fileExists(video) {
		t.Fatal("original should be kept when recycle=false")
	}
	// Running again must not overwrite the first result
	res2, err := (&Preparer{}).Prepare(video, subs[1], title, false, func(float64) {})
	if err != nil || filepath.Base(res2.Output) != "Test Film (2024) (with subtitles).mkv" {
		t.Fatalf("second run: %v %q", err, res2.Output)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.partial.mkv")); len(matches) != 0 {
		t.Fatalf("leftover temp files: %v", matches)
	}

	// Poster: a 360x540 JPEG frame, cached, returned as a data URL
	url, err := posterDataURL(res.Output)
	if err != nil || len(url) < 1000 || url[:23] != "data:image/jpeg;base64," {
		t.Fatalf("poster: %v (len %d)", err, len(url))
	}
	pf, _ := posterFile(res.Output)
	if out, err := hiddenCmd(toolFFprobe.find(), "-v", "error", "-show_entries", "stream=width,height", "-of", "csv=p=0", pf).Output(); err != nil || string(out[:7]) != "360,540" {
		t.Fatalf("poster size: %q %v", out, err)
	}
	os.Remove(pf)
}
