package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Settings are saved in %APPDATA%\TV Share\settings.json.
type Settings struct {
	Folder          string `json:"folder"`
	Name            string `json:"name"`
	Port            int    `json:"port"`
	AutoStartShare  bool   `json:"autoStartShare"`
	LaunchAtLogin   bool   `json:"launchAtLogin"`
	RecycleOriginal bool   `json:"recycleOriginal"`
	SubLang         string `json:"subLang"`
}

func defaultSettings() Settings {
	folder, err := windows.KnownFolderPath(windows.FOLDERID_Videos, 0)
	if err != nil || folder == "" {
		home, _ := os.UserHomeDir()
		folder = filepath.Join(home, "Videos")
	}
	// "DESKTOP-4F7Q2ZX" is hard to spot on a TV; "Mehdi's PC" is not.
	name := "My PC"
	if u := os.Getenv("USERNAME"); u != "" && len(u) <= 24 {
		name = u + "'s PC"
	}
	return Settings{
		Folder:          folder,
		Name:            name,
		Port:            7879,
		AutoStartShare:  false,
		LaunchAtLogin:   false,
		RecycleOriginal: true,
		SubLang:         "eng",
	}
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "TV Share", "settings.json")
}

func loadSettings() Settings {
	s := defaultSettings()
	b, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	d := defaultSettings()
	if s.Folder == "" {
		s.Folder = d.Folder
	}
	if s.Name == "" {
		s.Name = d.Name
	}
	if s.Port < 1024 || s.Port > 65535 {
		s.Port = d.Port
	}
	if s.SubLang == "" {
		s.SubLang = d.SubLang
	}
	return s
}

func saveSettings(s Settings) error {
	p := settingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(p, b, 0o644)
}
