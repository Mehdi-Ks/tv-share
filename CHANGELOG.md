# Changelog

## 1.2.1 (2026-11-15)

- **Settings is easier to close**: a close button (✕) in its corner, and clicking outside it closes it too, as does
  Esc. *Done* is still at the bottom. The same click-outside works for the rename box.
- **Show in folder and Open folder now open in front of the app.** Before, Explorer did open, but behind the
  TV Share window, so the click seemed to do nothing.
- **Adding subtitles names each step**: building the subtitles in, finishing the file, checking the new file,
  moving the original to the Recycle Bin. Before, the progress could seem stuck in the 90s while Windows finished
  writing a big file.
- **The PC stays responsive while subtitles are added**: FFmpeg now runs at low priority (also for posters).

## 1.2.0 (2026-11-15)

- **Movie details**: click a poster (or press Enter) to open a full-window page for that movie, over a wide frame
  from the film: runtime, picture quality, sound tracks, subtitles, and where to find it on the TV. The poster
  glides from the shelf into the page. Adding subtitles now happens here, with the language list in plain sight.
- **Sort and filter**: *Recently added* (the default), *A to Z* or *Biggest first*, and a *Can get subtitles* filter.
- **Keyboard**: arrow keys move across the poster shelf, Enter opens a movie, Esc goes back.
- Search stays visible in small windows, small links are easier to click, screen readers no longer read the whole
  shelf aloud on every refresh, and the help panel keeps keyboard focus inside it while open.
- Calmer: posters no longer cascade in on launch; motion is kept for things you do.

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
