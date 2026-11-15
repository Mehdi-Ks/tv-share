# Changelog

## 1.1.0 (2026-11-15)

A new look: a dark, cinematic wall of posters.

- **Posters**: each movie gets a poster made from a frame of the movie itself (picked past the opening credits,
  skipping black frames). The whole frame is shown, centred, over a blurred copy of itself, so nothing is cut off. Made on your PC with FFmpeg, cached, about 16 KB each; nothing is looked
  up online. Movies without a frame get a stylish title cover.
- **New layout**: full-window poster grid, a slim top bar with **Share with TV** switch and search (Ctrl+F),
  and a **?** panel with the "Watch on your TV" steps, the VLC *Play from start* tip and your share details.
- **Subtitles at a glance**: CC (built in) or SRT (can be added) on each poster, **+ Subtitles** under the title,
  language choice on hover, and a progress ring on the poster while it works.
- Warm amber accent, dark only, Windows' own Bahnschrift and Segoe UI fonts (nothing downloaded), new matching app icon.
- Polish from a UI/UX and motion review: pressed-state feedback on every button, interruptible toasts with faster
  exits, dialogs that scale in, hover effects only for a mouse, and calmer motion when Windows' "reduce motion" is on.

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
