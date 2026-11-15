# Changelog

## 1.4.0 (2026-11-15)

- **Official posters, plot and rating from TMDB**, if you want them. TV Share asks once (on the first-run screen,
  or with a banner if you already use it): *Use official posters* or *Keep movie frames*. Change it any time in
  Settings. Only each movie's title and year are sent to TMDB; nothing else leaves your PC.
- The movie page shows TMDB's plot, genres and rating, over TMDB's wide backdrop.
- **Not this movie?** If TMDB picked the wrong film, choose the right one from its top results (or search
  another title). TV Share remembers your pick. You can also use a frame from the movie instead, for films TMDB
  doesn't know, like home videos.
- Release builds include a TMDB key, so there is nothing to set up. You can use your own key in Settings instead.
- TMDB data and images are cached on your PC and refreshed every 3 months (TMDB allows 6 at most).

This product uses the TMDB API but is not endorsed or certified by TMDB.

## 1.3.0 (2026-11-15)

- **First-run screen: "Where are your movies?"** After the helpers install, pick the folder your TV will see
  (it says how many movies it found there) and the name you'll look for on the TV. If you've changed any setting
  before, you skip it and your settings are kept; otherwise it shows once.
- **Your share moved to Settings**: the name on the TV, the movie folder and the network are now in Settings,
  next to the other options. The **?** panel is only about watching: the steps and the VLC tip.
- **A friendlier default name**: the first name from your Windows account ("Mehdi's PC") when Windows has
  one, instead of the sign-in name.

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
