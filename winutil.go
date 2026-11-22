package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000

// hiddenCmd builds a command that never flashes a console window.
func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd
}

var allowSetForeground = windows.NewLazySystemDLL("user32.dll").NewProc("AllowSetForegroundWindow")

const asfwAny = ^uint32(0) // ASFW_ANY

// windowsDisplayName is the account's full name ("Mehdi K."), or "" when Windows has none.
func windowsDisplayName() string {
	proc := windows.NewLazySystemDLL("secur32.dll").NewProc("GetUserNameExW")
	if proc.Find() != nil {
		return ""
	}
	const nameDisplay = 3
	n := uint32(256)
	buf := make([]uint16, n)
	if r, _, _ := proc.Call(nameDisplay, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r == 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf[:n]))
}

// lowPriority runs a long job (copying a movie, taking frames) below normal priority.
func lowPriority(cmd *exec.Cmd) {
	cmd.SysProcAttr.CreationFlags |= 0x00004000 // BELOW_NORMAL_PRIORITY_CLASS
}

// openShell opens a file or folder the way a double-click in Explorer does, in a visible window.
func openShell(p string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(p)
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(p))
	return windows.ShellExecute(0, verb, file, nil, dir, windows.SW_SHOWNORMAL)
}

// showInExplorer opens a folder with the file selected. Explorer needs the exact form
// /select,"C:\pathile" as written; Go's usual argument quoting breaks it for some names.
func showInExplorer(p string) {
	// We're the app the user just clicked, so we may hand "come to the front" to Explorer;
	// without this its window opens behind ours and the click seems to do nothing.
	allowSetForeground.Call(uintptr(asfwAny))
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + p + `"`}
	_ = cmd.Start()
}

// --- Child processes die with the app, even if it crashes -----------------

var jobHandle windows.Handle

func initJob() {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, err = windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		windows.CloseHandle(h)
		return
	}
	jobHandle = h
}

func attachToJob(p *os.Process) {
	if jobHandle == 0 || p == nil {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	_ = windows.AssignProcessToJobObject(jobHandle, h)
}

// --- Recycle Bin -----------------------------------------------------------

type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// moveToRecycleBin sends a file to the Recycle Bin (recoverable), never a hard delete.
func moveToRecycleBin(path string) error {
	const (
		foDelete          = 0x3
		fofSilent         = 0x4
		fofNoConfirmation = 0x10
		fofAllowUndo      = 0x40
		fofNoErrorUI      = 0x400
	)
	from, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0) // list must end with two NULs
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	r, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	if r != 0 {
		return fmt.Errorf("could not move to Recycle Bin (code %d)", r)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("moving to Recycle Bin was cancelled")
	}
	return nil
}

// --- Start with Windows ------------------------------------------------------

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValue = "TV Share"

func setLaunchAtLogin(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		err = k.DeleteValue(runValue)
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValue, `"`+exe+`" --minimized`)
}

// --- Network -----------------------------------------------------------------

// localIP returns the address other devices on the home network use to reach this PC.
func localIP() string {
	conn, err := net.Dial("udp", "192.0.2.1:9") // no packet is sent; just picks the outgoing interface
	if err == nil {
		defer conn.Close()
		if a, ok := conn.LocalAddr().(*net.UDPAddr); ok && a.IP.IsPrivate() {
			return a.IP.String()
		}
	}
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && n.IP.IsPrivate() &&
				!strings.HasPrefix(i.Name, "vEthernet") {
				return n.IP.String()
			}
		}
	}
	return ""
}

type NetworkInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"` // Public, Private, DomainAuthenticated, or "" if unknown
	IP       string `json:"ip"`
}

func networkInfo() NetworkInfo {
	info := NetworkInfo{IP: localIP()}
	out, err := hiddenCmd("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`$p = Get-NetConnectionProfile | Where-Object { $_.IPv4Connectivity -ne 'Disconnected' } | Select-Object -First 1; if ($p) { "$($p.Name)|$($p.NetworkCategory)" }`).Output()
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
		if len(parts) == 2 {
			info.Name, info.Category = parts[0], parts[1]
		}
	}
	return info
}

// fileAdded is when a file arrived on this PC: the later of its creation time here and its
// last change. Copied or downloaded movies keep an old "modified" date, so creation matters.
func fileAdded(st os.FileInfo) int64 {
	t := st.ModTime().UnixMilli()
	if d, ok := st.Sys().(*syscall.Win32FileAttributeData); ok {
		if c := d.CreationTime.Nanoseconds() / 1e6; c > t {
			t = c
		}
	}
	return t
}
