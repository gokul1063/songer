# Songer – Development TODO

Fresh architecture plan for the terminal-based music player.

---

1. System Overview

---

Songer is a terminal-based music player written in Go.

Core capabilities:

- YouTube search
- Download songs
- Play songs
- Local library scanning
- Playlist system
- Listening history
- Terminal UI

External tools used:

mpv
yt-dlp
curl

---

2. Project Structure

---

.
├── cmd
│ └── songer
│ └── main.go
│
├── internal
│ ├── player
│ ├── youtube
│ ├── search
│ ├── scanner
│ ├── storage
│ ├── database
│ ├── ui
│ ├── logger
│ └── config
│
├── models
├── scripts
├── logs
├── storage

Principles

- Modular packages
- Clear separation of concerns
- Service-oriented design
- Independent internal modules

---

3. Backend Features

---

YouTube Search

Requirements

- search YouTube using curl
- parse results
- retrieve metadata

Metadata fields

title
author
share link
video id
suggestions

Song Download

Tool

yt-dlp

Tasks

download audio
store locally
extract metadata

Playback System

Tool

mpv

Capabilities

play
pause
resume
stop
seek
progress tracking

Local Library Scanner

Capabilities

scan directories
detect audio files

Supported formats

mp3
flac
wav
m4a
opus

Database System

Data to store

Listening History
Playlists
Liked Songs
Song Metadata

---

4. Frontend (TUI)

---

Terminal user interface.

Requirements

alternate screen buffer
real-time updates

Player UI

Displays

song title
author
youtube link
tags

Progress Bar

Shows

current playback time
total duration

Search UI

Modes

YouTube search
Local search

---

5. Storage

---

Song storage directory

~/.local/share/songer/songs

Database file

~/.local/share/songer/database.db

Logs

logs/app.log

---

6. Configuration

---

Config file

~/.config/songer/config.json

Config fields

music_directory
scan_directories
download_quality
ui_settings

---

7. Logging

---

Requirements

debug logs
error logs
playback logs

---

8. Development Roadmap

---

Phase 1

project skeleton
config loader
logger
models

Phase 2

mpv player integration
local song playback

Phase 3

YouTube search
yt-dlp download

Phase 4

database system
playlists
history

Phase 5

terminal UI
search interface
player interface

Phase 6

recommendation system
song suggestions
