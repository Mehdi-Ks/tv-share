# Changelog

## 1.8.1 (2026-11-22)

- **Play on this PC.** Double-click a poster, or click **Play on this PC** on a movie's page, to watch it on the
  computer in your usual video player. If that player isn't VLC, the app asks "Won't play, or no subtitles?" and
  offers **Play in VLC**, or **Get VLC (free)** if it isn't installed. VLC plays every format TV Share handles and
  shows the subtitles built into the movie.
- Fixed: messages (like "Subtitles added") were hidden behind an open movie page or Settings.

## 1.8.0 (2026-11-22)

- **Find subtitles online.** On a movie's page, pick a language and search OpenSubtitles.com. Subtitles made for
  your exact file are marked **Made for this file** and listed first: they're in sync from the start. Pick one and
  TV Share downloads it and builds it into the movie as usual (with the sync check).
- **Easy account sign-in.** Without an account, OpenSubtitles allows 5 downloads a day. Settings → *Subtitles from
  OpenSubtitles* has a one-click link to create a free account and a sign-in box for more downloads. Your password
  is sent to OpenSubtitles only and never saved; only their sign-in token is kept. When the daily limit is reached,
  the app says so and points to sign-in.
- **Your own key, if you want:** paste it in Settings, with a link to the page where you get one for free.
- Searching sends the file's fingerprint and the movie's title and year to OpenSubtitles, only when you click.
- Fixed: a long movie page (for example after adding subtitles) had its top cut off and couldn't be scrolled back up.
- Fixed: a wrong OpenSubtitles key pasted in Settings could be accepted.

## 1.7.0 (2026-11-21)

- **Subtitles in sync with the voices.** When you add subtitles, TV Share now listens to the movie's sound and
  lines them up: it fixes a constant delay, subtitles made for a different frame rate (they drift further off as
  the movie goes on), and subtitles made for a different cut. It says what it changed, for example
  "Fixed subtitles made for a different frame rate, moved them 1.7 s later." About 30 seconds more; switch it off
  in Settings.
- **Fix sync** on a movie's page re-times subtitles that are already inside the movie. Or move them yourself:
  **Earlier** / **Later** by 0.5 to 5 seconds. The new file is checked before the original is replaced.
- Uses [alass](https://github.com/kaegi/alass) (GPL-3.0), downloaded the first time sync is used (26 MB, from its
  GitHub release, checked against a pinned fingerprint). It uses TV Share's own FFmpeg.

## 1.6.2 (2026-11-21)

- **Fixed: the movie list stayed on "Looking for movies…"** when a movie had no subtitles at all (none built in
  and no .srt file next to it). The whole list failed to appear, not just that movie. Bug since 1.2.0.

## 1.6.1 (2026-11-15)

- French wording proofread with DeepL Write: clearer subtitle and error messages ("intégrez-les au film",
  "la TV ne peut pas le lire", "Langue préférée pour les sous-titres"), and punctuation fixes.

## 1.6.0 (2026-11-15)

- **TV Share speaks French.** The app follows Windows' language automatically (French or English), and
  *Settings → App language* lets you pick. Everything is translated: setup, the poster wall, the movie page,
  help, settings, progress steps and error messages. Movie titles stay as they are.
- With official posters on, TMDB's plots and genres come in French too.
- Translations live in one file (`frontend/dist/i18n.js`); adding a language means copying the French block.

## 1.5.0 (2026-11-15)

Found by testing on a brand-new Windows PC (Windows Sandbox):

- **Setup works without winget.** Some PCs don't have Windows' package manager (winget), and setup used to stop
  there. Now TV Share downloads the two helpers straight from their official sites (rclone.org and gyan.dev),
  checks each file against the checksum those sites publish, and keeps them in its own folder. winget is still
  used first when it's there. If the download fails, a button opens App Installer in the Microsoft Store.
- **A heads-up before the first share**: Windows Firewall asks about "Rclone", a name nobody recognises. TV Share
  now explains it just before, so people click *Allow* instead of *Cancel*.
- The installer's first and last pages show TV Share's own picture instead of the generic one.
- Fixed: the spinners next to "rclone" and "FFmpeg" during setup appeared in the window's corner instead of on
  their rows (since 1.1.0). The setup time estimate is now honest about slow connections.

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
