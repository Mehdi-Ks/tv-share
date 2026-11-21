package main

// Subtitle sync. Out-of-sync subtitles come from three things: a constant delay, a different frame
// rate (subtitles timed for a 25 fps release on a 23.976 fps movie drift further off over time), or
// a different cut. alass (https://github.com/kaegi/alass, GPL-3.0, run as a separate program like
// FFmpeg) listens to the movie's sound and fixes all three. A manual shift covers the rest.

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var toolAlass = tool{exe: "alass-cli.exe"}

// alass 2.0.0 for Windows, from its GitHub release. The project publishes no checksums, so the
// download is pinned to the exact file we checked.
var (
	alassZipURL  = "https://github.com/kaegi/alass/releases/download/v2.0.0/alass-windows64.zip"
	alassZipHash = "e81a72f97f592910e909a2352d6b8c0de0801c51ac1383bad4ebf3f2ecdd2fd8"
)

// ensureAlass downloads alass the first time sync is used.
func ensureAlass(report func(float64, string)) error {
	if toolAlass.find() != "" {
		return nil
	}
	report(0, "Getting the sync helper…")
	// The download reports 0..1; the ring shows 0..100, and the first run of sync shows it.
	return fetchVerifyUnzip(alassZipURL, alassZipHash, "alass", func(f float64, msg string) {
		report(math.Min(99, f*100), "Getting the sync helper…")
	}, "alass-cli.exe")
}

// SyncSummary describes what alass changed, in plain words.
type SyncSummary struct {
	ShiftMs   int    // the main shift applied (positive = later)
	Parts     int    // how many separately shifted parts (more than 1 = a different cut)
	FPSFix    string // e.g. "25/23.976" when a frame-rate mismatch was corrected
	Unchanged bool
}

func (s SyncSummary) String() string {
	if s.Unchanged {
		return "The subtitles already matched the sound."
	}
	var parts []string
	if s.FPSFix != "" {
		parts = append(parts, "fixed subtitles made for a different frame rate")
	}
	if s.ShiftMs != 0 {
		dir := "later"
		if s.ShiftMs < 0 {
			dir = "earlier"
		}
		parts = append(parts, fmt.Sprintf("moved them %.1f s %s", math.Abs(float64(s.ShiftMs))/1000, dir))
	}
	if s.Parts > 1 {
		parts = append(parts, fmt.Sprintf("in %d parts", s.Parts))
	}
	if len(parts) == 0 {
		return "The subtitles were matched to the sound."
	}
	out := strings.Join(parts, ", ")
	return strings.ToUpper(out[:1]) + out[1:] + "."
}

var (
	reAlassRatio = regexp.MustCompile(`ratio is ([\d.]+/[\d.]+)`)
	reAlassShift = regexp.MustCompile(`shifted block of \d+ subtitles with length \S+ by (-?)(\d+):(\d+):(\d+)\.(\d+)`)
	reAlassProg  = regexp.MustCompile(`^\s*(\d+) / (\d+) \[`)
)

func parseAlassOutput(lines []string) SyncSummary {
	var s SyncSummary
	for _, l := range lines {
		if m := reAlassRatio.FindStringSubmatch(l); m != nil {
			a, b, _ := strings.Cut(m[1], "/")
			if fa, err1 := strconv.ParseFloat(a, 64); err1 == nil {
				if fb, err2 := strconv.ParseFloat(b, 64); err2 == nil && math.Abs(fa-fb) > 0.01 {
					s.FPSFix = m[1]
				}
			}
		}
		if m := reAlassShift.FindStringSubmatch(l); m != nil {
			h, _ := strconv.Atoi(m[2])
			mi, _ := strconv.Atoi(m[3])
			sec, _ := strconv.Atoi(m[4])
			frac := (m[5] + "000")[:3]
			ms, _ := strconv.Atoi(frac)
			total := ((h*60+mi)*60+sec)*1000 + ms
			if m[1] == "-" {
				total = -total
			}
			if s.Parts == 0 {
				s.ShiftMs = total
			}
			s.Parts++
		}
	}
	if s.FPSFix == "" && (s.Parts == 0 || (s.Parts == 1 && s.ShiftMs == 0)) {
		s.Unchanged = true
	}
	return s
}

// runAlass writes a synced copy of srtIn to srtOut, timed against the movie's sound.
func runAlass(ctx context.Context, video, srtIn, srtOut string, progress func(float64)) (SyncSummary, error) {
	alass, ffmpeg, ffprobe := toolAlass.find(), toolFFmpeg.find(), toolFFprobe.find()
	if alass == "" || ffmpeg == "" || ffprobe == "" {
		return SyncSummary{}, fmt.Errorf("the sync helper is not installed")
	}
	cmd := hiddenCmd(alass, video, srtIn, srtOut)
	cmd.Env = append(os.Environ(), "ALASS_FFMPEG_PATH="+ffmpeg, "ALASS_FFPROBE_PATH="+ffprobe)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return SyncSummary{}, err
	}
	cmd.Stderr = cmd.Stdout
	lowPriority(cmd)
	if err := cmd.Start(); err != nil {
		return SyncSummary{}, err
	}
	attachToJob(cmd.Process)
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
		case <-stop:
		}
	}()

	var lines []string
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	sc.Split(scanLinesOrCR)
	last := time.Time{}
	for sc.Scan() {
		l := sc.Text()
		if m := reAlassProg.FindStringSubmatch(l); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if total, err := strconv.Atoi(m[2]); err == nil && total > 0 && time.Since(last) > 300*time.Millisecond {
					progress(float64(n) / float64(total))
					last = time.Now()
				}
			}
			continue
		}
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		return SyncSummary{}, fmt.Errorf("cancelled")
	}
	if err != nil || !fileExists(srtOut) {
		return SyncSummary{}, fmt.Errorf("couldn't match the subtitles to the sound")
	}
	return parseAlassOutput(lines), nil
}

var reSRTTime = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})`)

// shiftSRT moves every subtitle by ms milliseconds (negative = earlier), never before 0.
func shiftSRT(text string, ms int) string {
	return reSRTTime.ReplaceAllStringFunc(text, func(t string) string {
		m := reSRTTime.FindStringSubmatch(t)
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		s, _ := strconv.Atoi(m[3])
		f, _ := strconv.Atoi(m[4])
		v := ((h*60+mi)*60+s)*1000 + f + ms
		if v < 0 {
			v = 0
		}
		return fmt.Sprintf("%02d:%02d:%02d,%03d", v/3600000, v/60000%60, v/1000%60, v%1000)
	})
}

// scaled maps a 0..100 progress callback onto the lo..hi part of the overall ring.
func scaled(progress func(float64, string), lo, hi float64, stage string) func(float64) {
	return func(f float64) { progress(lo+(hi-lo)*math.Max(0, math.Min(1, f)), stage) }
}

// begin claims the Preparer for one job (one movie at a time).
func (pr *Preparer) begin() (context.Context, func(), error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if pr.cancel != nil {
		return nil, nil, fmt.Errorf("another movie is already being prepared")
	}
	ctx, cancel := context.WithCancel(context.Background())
	pr.cancel = cancel
	return ctx, func() {
		pr.mu.Lock()
		pr.cancel = nil
		pr.mu.Unlock()
		cancel()
	}, nil
}

// FixSync re-times the subtitles already inside a movie: automatically against the sound (auto),
// and/or by a manual shift. The new file is checked before the original is replaced.
func (pr *Preparer) FixSync(video string, auto bool, shiftMs int, recycle bool, progress func(float64, string)) (PrepareResult, error) {
	var res PrepareResult
	ffmpeg, ffprobe := toolFFmpeg.find(), toolFFprobe.find()
	if ffmpeg == "" || ffprobe == "" {
		return res, fmt.Errorf("FFmpeg is not installed yet")
	}
	if !fileExists(video) {
		return res, fmt.Errorf("the movie or subtitle file is gone")
	}
	if auto {
		if err := ensureAlass(progress); err != nil {
			return res, err
		}
	}
	ctx, done, err := pr.begin()
	if err != nil {
		return res, err
	}
	defer done()

	work, err := os.MkdirTemp("", "tvshare-sync-")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(work)

	// 1. Take the first subtitle track out, with its language and title.
	progress(1, "Reading the subtitles…")
	lang, title := subTrackTags(ffprobe, video)
	srt := filepath.Join(work, "subs.srt")
	if out, err := hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", video,
		"-map", "0:s:0", "-c:s", "srt", srt).CombinedOutput(); err != nil || !fileExists(srt) {
		return res, fmt.Errorf("couldn't read the subtitles in this movie: %s", lastLine(string(out)))
	}

	// 2. Sync to the sound, then apply any manual shift.
	if auto {
		synced := filepath.Join(work, "synced.srt")
		sum, err := runAlass(ctx, video, srt, synced, scaled(progress, 2, 45, "Matching the subtitles to the sound…"))
		if err != nil {
			return res, err
		}
		srt, res.SyncNote = synced, sum.String()
	}
	if shiftMs != 0 {
		b, err := os.ReadFile(srt)
		if err != nil {
			return res, err
		}
		shifted := filepath.Join(work, "shifted.srt")
		if err := os.WriteFile(shifted, []byte(shiftSRT(string(b), shiftMs)), 0o644); err != nil {
			return res, err
		}
		srt = shifted
		dir := "later"
		if shiftMs < 0 {
			dir = "earlier"
		}
		note := fmt.Sprintf("Subtitles moved %.1f s %s.", math.Abs(float64(shiftMs))/1000, dir)
		if res.SyncNote != "" {
			note = res.SyncNote + " " + note
		}
		res.SyncNote = note
	}

	// 3. Build the movie again with the new subtitles in place of the old track.
	base := strings.TrimSuffix(video, filepath.Ext(video))
	tmp := base + ".partial.mkv"
	defer os.Remove(tmp)
	nSubs := countSubTracks(ffprobe, video)
	args := []string{"-hide_banner", "-nostats", "-loglevel", "error", "-y",
		"-i", video, "-sub_charenc", srtEncoding(srt), "-i", srt,
		"-map", "0", "-map", "-0:s:0", "-map", "1:0",
		"-c", "copy", "-c:s", "srt"}
	newIdx := strconv.Itoa(nSubs - 1) // the new track goes after the others
	if lang != "" {
		args = append(args, "-metadata:s:s:"+newIdx, "language="+lang)
	}
	if title != "" {
		args = append(args, "-metadata:s:s:"+newIdx, "title="+title)
	}
	args = append(args, "-disposition:s:"+newIdx, "default", "-progress", "pipe:1", tmp)
	lo := 2.0
	if auto {
		lo = 45
	}
	if err := runFFmpegProgress(ctx, ffmpeg, args, probeDuration(ffprobe, video), lo, progress); err != nil {
		return res, err
	}

	// 4. Check, then swap the files.
	progress(99, "Checking the new file…")
	if d, d0 := probeDuration(ffprobe, tmp), probeDuration(ffprobe, video); d0 > 0 && math.Abs(d-d0) > 2 {
		return res, fmt.Errorf("the new file is %.0f s long instead of %.0f s; the original is untouched", d, d0)
	}
	if !hasEmbeddedSubs(tmp) {
		return res, fmt.Errorf("the new file has no subtitle track; the original is untouched")
	}
	out := base + ".mkv"
	if recycle {
		progress(99, "Moving the original to the Recycle Bin…")
		if err := moveToRecycleBin(video); err != nil {
			return res, fmt.Errorf("couldn't move the original to the Recycle Bin, so nothing was changed")
		}
		res.Recycled = true
	} else {
		keep := base + " (before sync)" + filepath.Ext(video)
		if err := os.Rename(video, keep); err != nil {
			return res, err
		}
	}
	if err := os.Rename(tmp, out); err != nil {
		return res, err
	}
	res.Output = out
	progress(100, "Done")
	return res, nil
}

// runFFmpegProgress runs an FFmpeg job that writes "-progress pipe:1", reporting lo..99.
func runFFmpegProgress(ctx context.Context, ffmpeg string, args []string, duration, lo float64, progress func(float64, string)) error {
	cmd := hiddenCmd(ffmpeg, args...)
	stdout, _ := cmd.StdoutPipe()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	lowPriority(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	attachToJob(cmd.Process)
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
		case <-stop:
		}
	}()
	progress(lo, "Building the subtitles in…")
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "out_time_us=") && duration > 0 {
			if us, err := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_us="), 64); err == nil {
				f := math.Min(1, us/1e6/duration)
				stage := "Building the subtitles in…"
				if f >= 0.9 {
					stage = "Finishing the file…"
				}
				progress(math.Min(99, lo+(99-lo)*f), stage)
			}
		}
	}
	err := cmd.Wait()
	if ctx.Err() != nil {
		return fmt.Errorf("cancelled")
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("FFmpeg could not build the file: %s", lastLine(msg))
	}
	return nil
}

func srtEncoding(p string) string {
	if b, err := os.ReadFile(p); err == nil && !utf8.Valid(b) {
		return "CP1252"
	}
	return "UTF-8"
}

func subTrackTags(ffprobe, video string) (lang, title string) {
	out, err := hiddenCmd(ffprobe, "-v", "error", "-select_streams", "s:0",
		"-show_entries", "stream_tags=language,title", "-of", "default=noprint_wrappers=1", video).Output()
	if err != nil {
		return "", ""
	}
	for _, l := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(l), "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimPrefix(k, "TAG:")) {
		case "language":
			lang = v
		case "title":
			title = v
		}
	}
	return lang, title
}

func countSubTracks(ffprobe, video string) int {
	out, err := hiddenCmd(ffprobe, "-v", "error", "-select_streams", "s",
		"-show_entries", "stream=index", "-of", "csv=p=0", video).Output()
	if err != nil {
		return 1
	}
	n := 0
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return n
}
