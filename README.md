# TV Share

**Watch the movies on your Windows PC on your TV, with subtitles. No USB stick, no cables, no account.**

![TV Share: your movies as a wall of posters](docs/screenshot.png)

TV Share makes a folder on your PC (your *Videos* folder by default) show up on your TV, over your home Wi-Fi.
Open VLC on the TV, pick your PC, and play. On the PC, your movies appear as a dark, cinematic wall of posters.
Posters are either official ones from [TMDB](https://www.themoviedb.org) (you choose, see *Privacy* below) or a frame taken from the movie itself. The screenshots show Blender's free open movies with their TMDB posters.
Sort by *Recently added*, *A to Z* or size, and click a poster for its details: runtime, picture quality, sound
tracks and subtitles, over a wide frame from the film.

![A movie's details page](docs/screenshot-details.png)

It also fixes the most common annoyance: **subtitles that don't show up on the TV.** TV apps usually ignore separate
`.srt` files when playing over the network. One click builds the subtitles *into* the movie file, with no re-encoding
and no quality loss. It takes about a minute.

## Download

1. Go to [**Releases**](../../releases/latest) and download **`TVShare-Setup.exe`** (4 MB).
2. Run it and click **Install**. No admin rights needed. You get a Start menu and desktop shortcut, and it shows
   up in *Settings → Apps → Installed apps* for uninstalling.
3. The first time the app opens, click **Install helpers**. TV Share sets up two free tools it needs (see below), about 150 MB, from their official sites.
   Then choose the folder with your movies (your *Videos* folder is suggested) and the name you'll pick on the TV.

   ![First run: choose your movie folder and the name on the TV](docs/screenshot-welcome.png)

Prefer no install? Download `TVShare.exe` instead and run it from anywhere (a USB stick, Downloads, ...).

> **"Windows protected your PC"?** The app isn't signed with a paid certificate, so Windows SmartScreen warns about
> new downloads. Click **More info → Run anyway**. The full source code is here if you want to check it or build it yourself.

Needs Windows 10 or 11, 64-bit. The app is in **English and French** (it follows Windows; change it in Settings).
*TV Share est disponible en français : l'app suit la langue de Windows.*

## How to use it

**On the PC**
1. Open TV Share and turn on **Share with TV** (top right).
2. The first time, Windows asks whether **Rclone** may use the network (Rclone is the helper that talks to the TV). Click **Allow**.
3. Keep the app open and the PC awake while you watch.

Keyboard: arrow keys move across the posters, Enter opens a movie, Esc goes back, Ctrl+F searches.

The **?** button opens these steps inside the app. Your PC's name on the TV, the movie folder and the network are in **Settings**.

![The "Watch on your TV" panel](docs/screenshot-help.png)

**On the TV**
1. Install **VLC** from the TV's app store (free). Most smart TVs, Android TV / Google TV, Fire TV and game consoles have it.
2. In VLC, go to **Browse → Local network** and pick your PC's name (shown in the app's Settings under *Name on the TV*).
3. Open a movie.

VLC is the recommended player, but anything that can browse a "media server" (DLNA/UPnP) works: Kodi, Xbox and
PlayStation media players, and many TVs' built-in media apps.

**Subtitles**
- Movies with subtitles already inside show **CC**. Nothing to do.
- Movies that came with `.srt` files (next to the movie or in a `Subs` folder) show **SRT** and **+ Subtitles**.
  Click it, or open the movie (click its poster) to pick another language first. The **Can get subtitles** filter lists them all. The new file is checked (same length, subtitle track present) before anything
  else happens. Then the original goes to the **Recycle Bin**, never deleted outright. You can turn that off in Settings.
- Movie somewhere else (like Downloads)? Drag its folder onto the window to add subtitles. To watch it, move it into
  the shared folder.

## When something doesn't work

| Problem | Fix |
|---|---|
| The TV doesn't list your PC | Make sure both are on the **same Wi-Fi**. If the app says the network is *public*, click **Open network settings** and set it to **Private** (only on your home network). |
| The PC shows up but nothing loads | Close and reopen VLC on the TV (or *Force stop* it in the TV's app settings), then try again. |
| A movie won't start, or VLC jumps straight to another file in the folder | VLC is resuming the movie from a saved position at its very end, so it "finishes" instantly and moves on. In VLC, **long-press the movie and choose *Play from start***. (The saved position can exist even when the movie isn't listed in History.) Last resort: on the TV, *Settings → Apps → VLC → Clear data* (resets VLC's history and settings; *Clear cache* is not enough). |
| Playback stops after a while | The PC went to sleep. Keep it plugged in and set sleep to *Never* while on power. |
| "Port already in use" | Another copy of the share is running. Close it, or restart the PC. |
| Subtitles still don't show | In VLC, open the subtitle menu during playback and pick the track. |

## Privacy and safety

- The TV can **only read** the folder you choose. Nothing on your PC can be changed or deleted from the network.
- Sharing only happens while the app is open and **Share with TV** is on. It works on your local network only,
  with nothing sent to the internet.
- The only downloads are the two helpers below: from Windows' package manager (winget) when the PC has it,
  otherwise straight from rclone.org and gyan.dev, checked against the checksums those sites publish.
- **Posters are your choice.** *Movie frames*: made on your PC from a frame of each movie, nothing looked up online.
  *Official posters*: each movie's title and year are sent to TMDB to fetch its poster, plot and rating; nothing
  else is sent. You choose on first run and can change it in Settings. Both are cached in
  `%LOCALAPPDATA%\TV Share` (TMDB data is refreshed every 3 months).
- Settings are stored in `%APPDATA%\TV Share\settings.json`. To remove the app, uninstall it from *Installed apps*
  (or delete `TVShare.exe` if you used the portable version), then delete that folder if you want your settings gone too.

## What's under the hood

TV Share is a small (~11 MB) app built with [Wails](https://wails.io) (Go, with the window drawn by Windows' built-in
WebView2), so it stays light even on older PCs. It drives two well-known open-source tools:

- [**rclone**](https://rclone.org) (MIT) runs the media server (`rclone serve dlna`, read-only).
- [**FFmpeg**](https://ffmpeg.org) (LGPL/GPL) copies video and audio unchanged into an MKV with the subtitles added,
  and takes the poster frames.

The look uses fonts that ship with Windows (Bahnschrift and Segoe UI), so nothing is downloaded at runtime.

Both are installed with winget (`Rclone.Rclone`, `Gyan.FFmpeg`) when it's available; otherwise TV Share downloads
the official builds itself (rclone's release zip and gyan.dev's FFmpeg "essentials" build), verifies their SHA-256
checksums and keeps them in `%LOCALAPPDATA%\TV Share\tools`. If you'd rather not use winget, put `rclone.exe`,
`ffmpeg.exe` and `ffprobe.exe` next to `TVShare.exe` and the app will use them.

## Credits

Movie posters and information come from [TMDB](https://www.themoviedb.org).
This product uses the TMDB API but is not endorsed or certified by TMDB.

<img src="frontend/dist/img/tmdb.svg" alt="TMDB logo" height="14">

## Build it yourself

You need [Go](https://go.dev) (version in `go.mod`) and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
git clone https://github.com/Mehdi-Ks/tv-share.git
cd tv-share
go test ./...
wails build -clean -trimpath
```

The app ends up in `build/bin/TVShare.exe`. Official posters need a TMDB key: get a free one at
[themoviedb.org](https://www.themoviedb.org/settings/api) and paste it in the app's Settings, or put it in
`build/tmdb.key` (one line, never committed) and build with `build.ps1`, which builds it into the app and also
makes the installer. To build the installer too, install [NSIS](https://nsis.sourceforge.io)
(`winget install NSIS.NSIS`) and run `makensis /DVERSION=1.6.1 build/installer/tvshare.nsi`, which writes
`build/bin/TVShare-Setup.exe`. The window is plain HTML, CSS and JavaScript in `frontend/dist`
(no Node.js or bundler needed). `wails dev` runs it with live reload.

The app icon is drawn by `build/make_icon.py` (Python + Pillow). See [CHANGELOG.md](CHANGELOG.md) for what changed in each version.

## Translations

All text is in English in the code; `frontend/dist/i18n.js` swaps in other languages as text reaches the screen.
To add a language, copy the French block in that file, translate the right-hand side, and add it to `LANGS`.

## Contributing

Bug reports, translations and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to set up,
test and send a change. Questions and "my TV doesn't see my PC" go in [Discussions](../../discussions).

## License

[MIT](LICENSE). Free and open source: use it, change it, share it.
