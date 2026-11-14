package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Share runs "rclone serve dlna": the folder appears on the TV (VLC, Kodi, most
// smart TV media players) as a media server, read-only, no password.
type Share struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	lastErr string
	onExit  func(errMsg string)
}

func (s *Share) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cmd != nil
}

func (s *Share) Start(set Settings) error {
	s.mu.Lock()
	if s.cmd != nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	rclone := toolRclone.find()
	if rclone == "" {
		return fmt.Errorf("rclone is not installed yet")
	}
	if !dirExists(set.Folder) {
		return fmt.Errorf("the folder %q does not exist", set.Folder)
	}
	if portAnswers(set.Port) {
		return fmt.Errorf("port %d is already in use, maybe the share is already running in another window", set.Port)
	}

	cmd := hiddenCmd(rclone, "serve", "dlna", set.Folder,
		"--name", set.Name,
		"--addr", ":"+strconv.Itoa(set.Port),
		"--read-only",
		"--log-level", "NOTICE")
	errPipe, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}
	attachToJob(cmd.Process)

	var errLines []string
	var linesMu sync.Mutex
	go func() {
		sc := bufio.NewScanner(errPipe)
		for sc.Scan() {
			l := sc.Text()
			if strings.Contains(l, "ERROR") || strings.Contains(l, "CRITICAL") || strings.Contains(l, "Fatal") {
				linesMu.Lock()
				errLines = append(errLines, l)
				linesMu.Unlock()
			}
		}
	}()

	s.mu.Lock()
	s.cmd = cmd
	s.lastErr = ""
	s.mu.Unlock()

	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
		s.mu.Lock()
		wasCurrent := s.cmd == cmd
		if wasCurrent {
			s.cmd = nil
		}
		s.mu.Unlock()
		if wasCurrent && s.onExit != nil {
			linesMu.Lock()
			msg := "the share stopped unexpectedly"
			if len(errLines) > 0 {
				msg = friendlyRcloneError(errLines[len(errLines)-1])
			}
			linesMu.Unlock()
			s.onExit(msg)
		}
	}()

	// Wait until it answers, or dies.
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			linesMu.Lock()
			defer linesMu.Unlock()
			if len(errLines) > 0 {
				return fmt.Errorf("%s", friendlyRcloneError(errLines[len(errLines)-1]))
			}
			return fmt.Errorf("the share could not start")
		default:
		}
		if portAnswers(set.Port) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil // slow machine: assume it is coming up
}

func (s *Share) Stop() {
	s.mu.Lock()
	cmd := s.cmd
	s.cmd = nil
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func portAnswers(port int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/rootDesc.xml", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

func friendlyRcloneError(line string) string {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "only one usage") || strings.Contains(l, "address already in use"):
		return "the network port is already in use (is another copy of the share running?)"
	case strings.Contains(l, "access is denied") || strings.Contains(l, "permission"):
		return "Windows blocked the share. Check that the firewall allows it"
	}
	if i := strings.Index(line, "ERROR"); i >= 0 {
		return strings.TrimSpace(strings.TrimLeft(line[i+5:], " :"))
	}
	return line
}
