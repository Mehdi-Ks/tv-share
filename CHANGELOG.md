# Changelog

## 1.0.1 (2026-11-15)

- **"Watch on your TV" now explains the most common VLC problem.** If a movie won't start, or VLC jumps straight to
  another file, VLC is resuming it from a saved position at the very end. Long-press the movie and choose
  *Play from start*. Found while testing on a real TV.
- New **"More fixes for common problems"** link to the troubleshooting guide.
- **Windows installer** (`TVShare-Setup.exe`): per-user install with no admin prompt, Start menu and desktop
  shortcuts, and an uninstall entry in *Installed apps*. English and French.
- The icon-drawing script (`build/make_icon.py`) is now in the repo, so everything in the release can be rebuilt from source.

## 1.0.0 (2026-11-15)

First release: share a folder with the TV over the home network (rclone DLNA, read-only), build subtitles into
movie files with FFmpeg, one-time helper setup with winget.
