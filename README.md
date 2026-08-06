# songer

A lightning-fast, keyboard-driven music player that streams songs straight from
YouTube. Type a song name, hit enter, and audio starts playing in seconds —
no accounts, no API keys, no payments.

`songer` talks directly to YouTube's own internal (InnerTube) API to search and
to find related songs, plays through **mpv**, and ships a fully
keyboard-controlled **TUI** with a live equalizer, a real-time queue, and async
downloading.

```
```

## Features

- **Instant streaming playback** — no downloads before you can listen. Audio
  starts as soon as the stream buffers (~2–4s on normal connections).
- **Seamless search** — single fast request to YouTube's InnerTube API; results
  ranked by YouTube relevance with title, duration, channel, and views.
- **Auto-play** — builds a 6-song suggestion queue (2 + 2×2) *from the currently
  playing song*, in a background goroutine, so the queue keeps refilling as you
  go (capped at 60, deduped).
- **Keyboard-driven TUI** — 70/30 layout (now playing / up next), animated
  equalizer, moving seekbar, focusable panels, zero mouse.
- **Async downloads** — concurrent yt-dlp downloads via a worker pool, with
  live progress and optional MP3 conversion.
- **Themes** — every color lives in a TOML config (`opencode` theme included).
- **Fast CLI** — search, play, queue, and download are all one command away.

## Requirements

| Dependency | Needed for | Notes |
|-----------|-----------|-------|
| Go ≥ 1.24 | building | |
| [mpv](https://mpv.io) | playback | the player engine |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | downloads only | |
| ffmpeg | MP3 conversion only | `--mp3` |
| node / bun / deno | downloads only | yt-dlp JS runtime for YouTube |

Playback needs only `mpv`. Downloads additionally need `yt-dlp` (+ a JS runtime
for modern yt-dlp) and optionally `ffmpeg`.

## Install

```sh
git clone git@github.com:gokul1063/songer.git
cd songer
go build -o songer .
```

Optionally move it into your `PATH`:

```sh
go install .            # installs as `songer`
# or
sudo mv songer /usr/local/bin/
```

## Quick start

```sh
# search and play the first result
./songer --source "surf curse freak"

# play a specific result
./songer --source "never gonna give you up" --rank 1

# the TUI — full player experience
./songer --source "lofi beats" --tui
```

## CLI usage

```
songer --source "song name" [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source` | — | song name or video to search on YouTube |
| `--limit` | `10` | max number of results to return |
| `--rank` | `1` | play the Nth search result (1-based) |
| `--tui` | off | launch the keyboard-driven TUI |
| `--video` | off | play with video instead of audio-only |
| `--no-play` | off | search only, do not start playback |
| `--autoplay` | off | build + print the suggested queue (2 + 4 = 6) |
| `--download` | off | download the selected result and exit |
| `--download-all` | off | download all results concurrently and exit |
| `--download-dir` | `downloads` | where downloads are saved |
| `--workers` | `3` | concurrent downloads |
| `--mp3` | off | convert downloads to MP3 (needs ffmpeg) |

### Examples

```sh
# search only
./songer --source "the strokes" --no-play

# play the 3rd result
./songer --source "the strokes last nite" --rank 3

# print the autoplay queue
./songer --source "surf curse freak" --autoplay --no-play

# download the first result as an MP3
./songer --source "surf curse freak" --download --mp3

# download all 5 results concurrently (4 workers)
./songer --source "tame impala" --limit 5 --download-all --workers 4

# TUI with video enabled
./songer --source "lofi" --tui --video
```

## The TUI

Launch it with `--tui`. Everything is keys — there is no mouse.

Layout (70 : 30):

- **Main (70%)** — animated equalizer (gradient wave) filling the panel,
  seekbar with a moving thumb + elapsed/remaining time, the song name under
  the bar, and details below (channel • views • play state).
- **List (30%)** — the up-next queue. Played songs are removed from the list as
  they play; autoplay keeps refilling it in the background. Supports scrolling
  (up to 60 entries).
- **Header** — app name, play state, volume, and which panel is focused.
- **Footer** — context-aware key hints.

### Key bindings

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | quit |
| `space` / `p` | play / pause |
| `n` | next song (plays first item in the list) |
| `]` / `[` | +5s / −5s |
| `}` / `{` | +10s / −10s |
| `h` / `l` | −5s / +5s *(main area focused)* |
| `j` / `k` | navigate the queue |
| `ctrl+j` / `ctrl+k` | move the focused song down / up |
| `d` | delete the focused song |
| `enter` | play the selected song |
| `ctrl+h` / `ctrl+l` | focus main / focus list |
| `+` / `-` | volume up / down |
| `?` | toggle the help overlay |

Auto-play: when a song ends, the first item in the list plays next
automatically and is removed from the list.

## Configuration

All UI colors — and no code colors are hardcoded — live in a TOML file.
`songer` looks for `./config.toml`, then `~/.config/songer/config.toml`, and
falls back to built-in defaults.

```toml
[ui]
theme = "opencode"

[player]
volume = 80

[autoplay]
per_node = 2
depth = 2

[themes.opencode]
background = "#0f0f1a"
primary    = "#00ff87"
wave       = "#00e5ff"
thumb      = "#ffcc66"
# ... see config.toml for the full palette
```

A `mono` theme is included; add your own under `[themes.<name>]` and switch
with `[ui] theme = "<name>"`.

## How it works

- **Search & suggestions** — `songer` POSTs to YouTube's internal InnerTube
  endpoints (`/youtubei/v1/search`, `/youtubei/v1/next`) with the public
  web-client key (the same one YouTube's own frontend ships). No API key, no
  auth, no quota. The related-videos endpoint feeds the auto-play queue.
- **Playback** — the TUI drives an `mpv` child process over its JSON IPC
  socket (`--input-ipc-server`), observing `pause`, `time-pos`, `duration`,
  `volume`, and `media-title`, and sending seek/load commands. `--idle=yes`
  keeps mpv alive so the next song can be loaded on end-of-file.
- **Downloads** — `yt-dlp` extracts the best audio; a worker pool downloads
  concurrently and streams progress back on a channel. The JS runtime
  (`node`/`bun`/`deno`) is auto-detected and passed via `--js-runtimes`.

## Project layout

```
songer/
├── main.go               # CLI entrypoint (flag handling, search → play/queue/download/TUI)
├── config.toml           # user-editable themes & defaults
└── pkg/
    ├── search/           # YouTube InnerTube search → []Video
    ├── autoplay/         # related-videos suggestions; BFS queue build (2+4=6)
    ├── play/             # blocking mpv playback (CLI mode) with latency tracking
    ├── mpv/              # mpv JSON-IPC controller (seek/pause/load/observe events)
    ├── download/         # async yt-dlp downloads with worker pool + progress
    ├── tui/              # bubbletea UI (70/30 layout, wave, focus, keys, themes)
    └── config/           # TOML loading, themes
```

## Notes / limitations

- `songer` scrapes YouTube's internal API. YouTube can (and has) changed its
  response schemas — parsing is isolated per endpoint so fixes stay local, but
  treat it as best-effort.
- The related-videos feed currently doesn't expose durations, so queued songs
  may show no length.
- The public web-client key flagged by GitHub secret scanning is a **false
  positive** — it is not a secret.

## Roadmap

- [x] Search engine
- [x] mpv-native streaming playback
- [x] Auto-play suggestion queue
- [x] Async downloads
- [x] Keyboard-driven TUI
- [x] TOML themes
- [ ] Favorites / liked songs
- [ ] Local library & playlists
- [ ] Persisted session history

## Disclaimer

`songer` is an educational/experimental tool. It is not affiliated with or
endorsed by YouTube. Respect YouTube's Terms of Service and applicable laws in
your jurisdiction.
