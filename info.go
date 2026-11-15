package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MovieInfo is what the details view shows about one file, read with ffprobe.
type MovieInfo struct {
	Minutes    int      `json:"minutes"`
	Resolution string   `json:"resolution"` // "1080p", "4K", "720p", ...
	Video      string   `json:"video"`      // "HEVC 10-bit", "H.264", ...
	Audio      []string `json:"audio"`      // "English 5.1", "French stereo", ...
	Subtitles  []string `json:"subtitles"`  // languages of the subtitle tracks inside the file
}

type probeStream struct {
	CodecType string            `json:"codec_type"`
	CodecName string            `json:"codec_name"`
	Profile   string            `json:"profile"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	PixFmt    string            `json:"pix_fmt"`
	Channels  int               `json:"channels"`
	Tags      map[string]string `json:"tags"`
	Dispo     map[string]int    `json:"disposition"`
}

type probeResult struct {
	Streams []probeStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func movieInfo(p string) (MovieInfo, error) {
	var info MovieInfo
	ffprobe := toolFFprobe.find()
	if ffprobe == "" {
		return info, fmt.Errorf("FFmpeg is not installed")
	}
	out, err := hiddenCmd(ffprobe, "-v", "error", "-print_format", "json",
		"-show_entries", "format=duration:stream=codec_type,codec_name,profile,width,height,pix_fmt,channels:stream_tags=language,title:stream_disposition=attached_pic",
		p).Output()
	if err != nil {
		return info, fmt.Errorf("couldn't read this movie's details")
	}
	return parseProbe(out)
}

func parseProbe(out []byte) (MovieInfo, error) {
	var info MovieInfo
	var r probeResult
	if err := json.Unmarshal(out, &r); err != nil {
		return info, err
	}
	if d, err := strconv.ParseFloat(r.Format.Duration, 64); err == nil {
		info.Minutes = int(d/60 + 0.5)
	}
	info.Audio, info.Subtitles = []string{}, []string{}
	for _, s := range r.Streams {
		switch s.CodecType {
		case "video":
			if info.Video != "" || s.Dispo["attached_pic"] == 1 {
				continue
			}
			info.Resolution = resolutionName(s.Width, s.Height)
			info.Video = codecName(s.CodecName)
			if strings.Contains(s.PixFmt, "10") || strings.Contains(s.PixFmt, "12") {
				info.Video += " 10-bit"
			}
		case "audio":
			a := strings.TrimSpace(langName(s.Tags["language"]) + " " + channelName(s.Channels))
			if a == "" {
				a = strings.ToUpper(s.CodecName)
			}
			a = strings.ToUpper(a[:1]) + a[1:]
			info.Audio = append(info.Audio, a)
		case "subtitle":
			l := langName(s.Tags["language"])
			if l == "" {
				l = "Unknown language"
			}
			info.Subtitles = append(info.Subtitles, l)
		}
	}
	return info, nil
}

func resolutionName(w, h int) string {
	switch {
	case w >= 3200 || h >= 2000:
		return "4K"
	case w >= 1800 || h >= 1000:
		return "1080p"
	case w >= 1200 || h >= 700:
		return "720p"
	case w > 0:
		return "SD"
	}
	return ""
}

func codecName(c string) string {
	switch c {
	case "hevc":
		return "HEVC"
	case "h264":
		return "H.264"
	case "av1":
		return "AV1"
	case "vp9":
		return "VP9"
	case "mpeg4":
		return "MPEG-4"
	}
	return strings.ToUpper(c)
}

func channelName(n int) string {
	switch {
	case n == 1:
		return "mono"
	case n == 2:
		return "stereo"
	case n == 6:
		return "5.1 surround"
	case n == 8:
		return "7.1 surround"
	case n > 0:
		return fmt.Sprintf("%d ch", n)
	}
	return ""
}

// langName turns an ISO 639-2 code from the file ("eng", "fre") into a name.
func langName(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || code == "und" {
		return ""
	}
	for _, l := range langTokens {
		if l.code == code {
			return l.name
		}
	}
	if alt, ok := map[string]string{"fra": "French", "deu": "German", "ger": "German", "spa": "Spanish", "ita": "Italian", "por": "Portuguese", "ara": "Arabic", "jpn": "Japanese", "kor": "Korean", "chi": "Chinese", "zho": "Chinese", "rus": "Russian", "dut": "Dutch", "nld": "Dutch", "tur": "Turkish", "hin": "Hindi"}[code]; ok {
		return alt
	}
	return strings.ToUpper(code)
}

// backdropDataURL returns a wide 16:9 frame of the movie for the details view,
// taken at the same moment as the poster and cached next to it.
func backdropDataURL(video string) (string, error) {
	key, err := posterKey(video)
	if err != nil {
		return "", err
	}
	out := filepath.Join(posterCacheDir(), key+"-wide.jpg")
	if !fileExists(out) {
		ffmpeg, ffprobe := toolFFmpeg.find(), toolFFprobe.find()
		if ffmpeg == "" || ffprobe == "" {
			return "", fmt.Errorf("FFmpeg is not installed")
		}
		posterSlots <- struct{}{}
		defer func() { <-posterSlots }()
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", err
		}
		at := 60.0
		if d := probeDuration(ffprobe, video); d > 0 {
			at = d * 0.15
		}
		tmp := out + ".tmp.jpg"
		err = hiddenCmd(ffmpeg, "-hide_banner", "-loglevel", "error", "-y",
			"-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", video,
			"-vf", "thumbnail=48,scale=960:540:force_original_aspect_ratio=increase,crop=960:540",
			"-frames:v", "1", "-q:v", "5", tmp).Run()
		if err != nil || !fileExists(tmp) {
			os.Remove(tmp)
			return "", fmt.Errorf("could not take a frame from this movie")
		}
		if err := os.Rename(tmp, out); err != nil {
			return "", err
		}
	}
	b, err := os.ReadFile(out)
	if err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b), nil
}
