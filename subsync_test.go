package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShiftSRT(t *testing.T) {
	in := "1\n00:00:01,000 --> 00:00:03,500\nHi\n\n2\n01:59:59,900 --> 02:00:00,400\nBye\n"
	got := shiftSRT(in, 1250)
	if !strings.Contains(got, "00:00:02,250 --> 00:00:04,750") || !strings.Contains(got, "02:00:01,150 --> 02:00:01,650") {
		t.Fatalf("later: %q", got)
	}
	if got := shiftSRT(in, -2000); !strings.Contains(got, "00:00:00,000 --> 00:00:01,500") {
		t.Fatalf("earlier, clamped at 0: %q", got)
	}
}

// The lines alass printed for a real case: WEBRip subtitles on a BluRay copy of Dune (2021).
func TestParseAlassOutput(t *testing.T) {
	s := parseAlassOutput([]string{
		"extracting audio from reference file 'Dune (2021).mkv'...",
		"Guessing framerate ratio...",
		"info: 'reference file FPS/input file FPS' ratio is 25/23.976",
		"shifted block of 1001 subtitles with length 2:21:17.599 by 0:00:01.687",
	})
	if s.FPSFix != "25/23.976" || s.ShiftMs != 1687 || s.Parts != 1 || s.Unchanged {
		t.Fatalf("got %+v", s)
	}
	if got := s.String(); got != "Fixed subtitles made for a different frame rate, moved them 1.7 s later." {
		t.Fatalf("summary: %q", got)
	}
	cut := parseAlassOutput([]string{
		"info: 'reference file FPS/input file FPS' ratio is 1/1",
		"shifted block of 300 subtitles with length 0:40:00.000 by -0:00:02.500",
		"shifted block of 700 subtitles with length 1:20:00.000 by 0:00:10.000",
	})
	if cut.FPSFix != "" || cut.ShiftMs != -2500 || cut.Parts != 2 {
		t.Fatalf("cut: %+v", cut)
	}
	if same := parseAlassOutput([]string{"shifted block of 10 subtitles with length 0:01:00.000 by 0:00:00.000"}); !same.Unchanged {
		t.Fatalf("unchanged: %+v", same)
	}
}

// Fix sync on a movie that already has subtitles inside: a manual shift needs only FFmpeg.
func TestFixSyncManualShift(t *testing.T) {
	ffmpeg, ffprobe := toolFFmpeg.find(), toolFFprobe.find()
	if ffmpeg == "" || ffprobe == "" {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	if out, err := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=25:duration=6",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest", src).CombinedOutput(); err != nil {
		t.Fatalf("clip: %v %s", err, out)
	}
	srt := filepath.Join(dir, "in.srt")
	os.WriteFile(srt, []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o644)
	movie := filepath.Join(dir, "Movie (2024).mkv")
	if out, err := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", src, "-i", srt,
		"-map", "0", "-map", "1:0", "-c", "copy", "-c:s", "srt",
		"-metadata:s:s:0", "language=eng", "-metadata:s:s:0", "title=English", movie).CombinedOutput(); err != nil {
		t.Fatalf("movie: %v %s", err, out)
	}

	res, err := (&Preparer{}).FixSync(movie, false, 1500, false, func(float64, string) {})
	if err != nil {
		t.Fatalf("FixSync: %v", err)
	}
	if res.Output != movie || res.SyncNote != "Subtitles moved 1.5 s later." {
		t.Fatalf("result: %+v", res)
	}
	// The original is kept next to it (Recycle Bin was off), and the new subtitles start at 2.5 s.
	if !fileExists(filepath.Join(dir, "Movie (2024) (before sync).mkv")) {
		t.Fatal("original not kept")
	}
	out, _ := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-i", movie, "-map", "0:s:0", "-f", "srt", "-").Output()
	if !strings.Contains(string(out), "00:00:02,500 --> 00:00:03,500") {
		t.Fatalf("subtitles not shifted: %q", out)
	}
	if lang, title := subTrackTags(ffprobe, movie); lang != "eng" || title != "English" {
		t.Fatalf("tags lost: %q %q", lang, title)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "*.partial.mkv")); len(m) != 0 {
		t.Fatalf("leftovers: %v", m)
	}
}
