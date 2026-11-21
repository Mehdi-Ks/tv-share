# Contributing to TV Share

Thanks for helping. TV Share is for people who aren't technical, so the best changes are the ones that make
something simpler or clearer for them.

- **Something broken?** [Open a bug report](../../issues/new/choose). Include your TV, the app you play with
  (VLC, Kodi, ...) and your Windows language.
- **A question, or "my TV doesn't see my PC"?** Ask in [Discussions](../../discussions); someone with the same TV
  may already know.
- **An idea?** Start a [Discussion](../../discussions) first, so we can agree on it before you write code.

## Set up

You need Windows 10 or 11, [Go](https://go.dev) (version in `go.mod`) and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
git clone https://github.com/Mehdi-Ks/tv-share.git
cd tv-share
wails dev
```

`wails dev` opens the app with live reload: edit `frontend/dist/*` and the window updates. For helpers, the app
uses rclone and FFmpeg from winget, from `%LOCALAPPDATA%\TV Share\tools`, or next to `TVShare.exe`. On first run
it offers to install them.

**Official posters** need a TMDB key, which release builds include but the source code doesn't. Without one the
app shows frames from the movies instead, which is fine for most work. To test posters, get a free key at
[themoviedb.org](https://www.themoviedb.org/settings/api) and paste it in the app's Settings. Never commit a key.

## Where things are

| File | What it does |
|---|---|
| `frontend/dist/index.html`, `style.css`, `app.js` | The whole window: plain HTML/CSS/JS, no bundler |
| `frontend/dist/i18n.js` | Translations (see below) |
| `app.go` | Everything the window can call (`window.go.main.App.*`) |
| `share.go` | Starts and stops `rclone serve dlna` |
| `movies.go` | Finds movies and subtitle files, builds subtitles into movies |
| `poster.go`, `info.go`, `tmdb.go` | Posters (movie frames or TMDB), file details, TMDB lookups |
| `deps.go`, `download.go` | Installing the helpers (winget, or a checked direct download) |
| `subsync.go` | Subtitle sync: alass against the sound, manual shifts, re-timing subtitles inside a movie |
| `settings.go`, `winutil.go` | Settings file, Windows-specific bits |
| `build/installer/tvshare.nsi` | The installer (NSIS) |

## Before you send a pull request

```bash
gofmt -l .
go vet ./...
go test ./...
```

`gofmt -l .` should print nothing. Every pull request is checked automatically on Windows (formatting, tests
with FFmpeg, and a full build). The built app is attached to the check run, so reviewers can try your change.

A few house rules:

- **Plain words in the app.** Write for someone who has never heard of DLNA. Say what a button does ("Add
  subtitles"), and say how to fix an error, not just what failed.
- **Never delete a user's file outright.** Replaced originals go to the Recycle Bin, after the new file is checked.
- **Nothing online without asking.** Anything that sends data off the PC must be opt-in and explained.
- **Keep it light.** No new runtime downloads, frameworks or bundlers unless there's no other way.
- **New text in the app?** Add its French version to `i18n.js` (or mention it in the pull request and someone
  will translate it).

## Add a language

Every piece of text is written in English; `frontend/dist/i18n.js` swaps it as it reaches the screen.

1. Copy the `FR`, `FR_PATTERNS` and `FR_HTML` blocks and translate the right-hand side. Keep `{placeholders}` and
   HTML tags as they are.
2. Add your language to `LANGS` (for example `de: { words: DE, patterns: DE_PATTERNS, html: DE_HTML }`).
3. Add it to the *App language* list in `index.html`.
4. Run `wails dev`, switch to your language in Settings, and look at every screen: setup, the poster wall, a
   movie's page, help and settings.

A missing translation just stays in English, so a partial translation is still welcome.

## License

By contributing you agree that your work is released under the [MIT License](LICENSE), like the rest of TV Share.
