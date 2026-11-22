package main

// "Play on this PC": open a movie in the PC's own video player, the same as double-clicking it in
// Explorer. TV Share can't see inside that player, so after opening it the page offers VLC as the
// way out if the movie won't play or the subtitles don't show (VLC plays every format TV Share
// handles and shows built-in subtitles).

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const vlcPage = "https://www.videolan.org/vlc/"

// PlayResult tells the page which player opened the movie and whether VLC is there as a fallback.
type PlayResult struct {
	Player string `json:"player"` // "Media Player", "VLC media player"... ("" if unknown)
	IsVLC  bool   `json:"isVLC"`  // the default player already is VLC
	HasVLC bool   `json:"hasVLC"` // VLC is installed
}

var assocQueryString = windows.NewLazySystemDLL("shlwapi.dll").NewProc("AssocQueryStringW")

// defaultPlayer is the name of the app Windows opens for this file type, as Windows shows it
// ("Media Player", "VLC media player"); "" if none or unknown.
func defaultPlayer(ext string) string {
	if assocQueryString.Find() != nil {
		return ""
	}
	const assocStrFriendlyAppName = 4
	e, _ := windows.UTF16PtrFromString(ext)
	n := uint32(512)
	buf := make([]uint16, n)
	if r, _, _ := assocQueryString.Call(0, assocStrFriendlyAppName, uintptr(unsafe.Pointer(e)), 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r != 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf))
}

// vlcExe finds VLC: its registry entry first, then the usual folders.
func vlcExe() string {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		for _, path := range []string{`SOFTWARE\VideoLAN\VLC`, `SOFTWARE\WOW6432Node\VideoLAN\VLC`} {
			k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			dir, _, _ := k.GetStringValue("InstallDir")
			exe, _, _ := k.GetStringValue("")
			k.Close()
			if exe != "" && fileExists(exe) {
				return exe
			}
			if dir != "" && fileExists(filepath.Join(dir, "vlc.exe")) {
				return filepath.Join(dir, "vlc.exe")
			}
		}
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		if d := os.Getenv(env); d != "" {
			for _, sub := range []string{`VideoLAN\VLC\vlc.exe`, `Programs\VideoLAN\VLC\vlc.exe`} {
				if p := filepath.Join(d, sub); fileExists(p) {
					return p
				}
			}
		}
	}
	return ""
}

// PlayOnPC opens the movie in the PC's default video player.
func (a *App) PlayOnPC(p string) (PlayResult, error) {
	res := PlayResult{HasVLC: vlcExe() != ""}
	if !fileExists(p) {
		return res, fmt.Errorf("This movie isn't there any more. Press Refresh.")
	}
	res.Player = defaultPlayer(filepath.Ext(p))
	res.IsVLC = strings.Contains(strings.ToLower(res.Player), "vlc")
	allowSetForeground.Call(uintptr(asfwAny)) // so the player opens in front of TV Share
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(p)
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(p))
	if err := windows.ShellExecute(0, verb, file, nil, dir, windows.SW_SHOWNORMAL); err != nil {
		return res, fmt.Errorf("Windows has no video player for this kind of file")
	}
	return res, nil
}

// PlayInVLC opens the movie in VLC.
func (a *App) PlayInVLC(p string) error {
	exe := vlcExe()
	if exe == "" {
		return fmt.Errorf("VLC isn't installed")
	}
	allowSetForeground.Call(uintptr(asfwAny))
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(exe)
	args, _ := windows.UTF16PtrFromString(`"` + p + `"`)
	if err := windows.ShellExecute(0, verb, file, args, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("couldn't open VLC")
	}
	return nil
}

// GetVLCPage opens VLC's official download page.
func (a *App) GetVLCPage() { a.OpenURL(vlcPage) }
