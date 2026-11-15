package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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
	SetupDone       bool   `json:"setupDone"` // the first-run "Where are your movies?" screen was completed
}

func defaultSettings() Settings {
	folder, err := windows.KnownFolderPath(windows.FOLDERID_Videos, 0)
	if err != nil || folder == "" {
		home, _ := os.UserHomeDir()
		folder = filepath.Join(home, "Videos")
	}
	// "DESKTOP-4F7Q2ZX" is hard to spot on a TV; "Mehdi's PC" is not.
	// The account's display name ("Mehdi K.") beats the sign-in name, which is often a brand ("MSI").
	name := "My PC"
	if first := firstName(windowsDisplayName()); first != "" {
		name = first + "'s PC"
	} else if u := os.Getenv("USERNAME"); u != "" && len(u) <= 24 {
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

// firstName keeps the first word of a display name, if it looks like a name.
func firstName(display string) string {
	f := strings.Fields(display)
	if len(f) == 0 || len(f[0]) > 20 || strings.ContainsAny(f[0], `@\/`) {
		return ""
	}
	return f[0]
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
	// Settings saved before 1.3.0 have no "setupDone": those people are already set up.
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) == nil {
		if _, ok := raw["setupDone"]; !ok {
			s.SetupDone = true
		}
	}
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
