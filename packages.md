# Songer – Package Documentation

This document describes all Go packages used in the project.

For each package:

- Path
- Responsibility
- Functions with short purpose
- Inputs
- Outputs

---

## Project Structure

.
├── cmd
│ └── songer
│ └── main.go
│
├── internal
│ ├── player
│ │ └── player.go
│ │
│ ├── youtube
│ │ └── youtube.go
│ │
│ ├── search
│ │ └── search.go
│ │
│ ├── scanner
│ │ └── scanner.go
│ │
│ ├── storage
│ │ └── storage.go
│ │
│ ├── database
│ │ └── db.go
│ │
│ ├── ui
│ │ └── ui.go
│ │
│ ├── logger
│ │ └── logger.go
│ │
│ └── config
│ └── config.go
│
├── models
│ └── models.go
│
├── scripts
├── logs
├── storage
├── go.mod
├── go.sum
└── README.md

---

## cmd/songer

Purpose:
Application entrypoint.

main.go

Functions

InitCLI()

Purpose:
Initialize CLI commands and start application.

Inputs:
None

Outputs:
None

---

## internal/player

Purpose:
Control audio playback through mpv.

Functions

PlaySong(path string)

Purpose:
Start playback of a song using mpv.

Inputs:
path string

Outputs:
error

Pause()

Purpose:
Pause current playback.

Inputs:
None

Outputs:
error

Resume()

Purpose:
Resume paused playback.

Inputs:
None

Outputs:
error

Stop()

Purpose:
Stop current playback.

Inputs:
None

Outputs:
error

Seek(seconds int)

Purpose:
Seek playback position.

Inputs:
seconds int

Outputs:
error

GetProgress()

Purpose:
Return current playback position and duration.

Inputs:
None

Outputs:
current int
total int
error

---

## internal/youtube

Purpose:
YouTube search and metadata retrieval.

Functions

Search(query string)

Purpose:
Search YouTube using curl and return video results.

Inputs:
query string

Outputs:
[]models.SearchResult
error

GetSuggestions(videoID string)

Purpose:
Retrieve related/suggested videos.

Inputs:
videoID string

Outputs:
[]models.SearchResult
error

Download(videoURL string, outputDir string)

Purpose:
Download audio using yt-dlp.

Inputs:
videoURL string
outputDir string

Outputs:
string
error

---

## internal/search

Purpose:
Unified search interface across YouTube and local storage.

Functions

SearchYouTube(query string)

Purpose:
Search songs from YouTube.

Inputs:
query string

Outputs:
[]models.SearchResult
error

SearchLocal(query string)

Purpose:
Search songs from local database.

Inputs:
query string

Outputs:
[]models.Song
error

---

## internal/scanner

Purpose:
Scan filesystem for audio files.

Functions

ScanDirectories(paths []string)

Purpose:
Scan directories for supported audio files.

Inputs:
paths []string

Outputs:
[]models.Song
error

IsAudioFile(file string)

Purpose:
Check if file is supported audio format.

Inputs:
file string

Outputs:
bool

---

## internal/storage

Purpose:
Manage local song storage.

Functions

SaveSong(path string)

Purpose:
Register downloaded song in storage.

Inputs:
path string

Outputs:
models.Song
error

GetSong(id string)

Purpose:
Retrieve stored song metadata.

Inputs:
id string

Outputs:
models.Song
error

---

## internal/database

Purpose:
Database management for history, playlists, likes.

Functions

InitDB(path string)

Purpose:
Initialize database connection.

Inputs:
path string

Outputs:
error

SaveHistory(songID string)

Purpose:
Save listening history entry.

Inputs:
songID string

Outputs:
error

CreatePlaylist(name string)

Purpose:
Create a playlist.

Inputs:
name string

Outputs:
models.Playlist
error

AddSongToPlaylist(playlistID string, songID string)

Purpose:
Add song to playlist.

Inputs:
playlistID string
songID string

Outputs:
error

---

## internal/ui

Purpose:
Terminal user interface.

Functions

StartUI()

Purpose:
Start TUI rendering loop.

Inputs:
None

Outputs:
error

RenderPlayer()

Purpose:
Render player interface.

Inputs:
None

Outputs:
None

RenderSearch()

Purpose:
Render search results.

Inputs:
results []models.SearchResult

Outputs:
None

---

## internal/logger

Purpose:
Application logging.

Functions

InitLogger(path string)

Purpose:
Initialize logging system.

Inputs:
path string

Outputs:
error

Info(msg string)

Purpose:
Write informational log entry.

Inputs:
msg string

Outputs:
None

Error(msg string)

Purpose:
Write error log entry.

Inputs:
msg string

Outputs:
None

---

## internal/config

Purpose:
Application configuration loader.

Functions

LoadConfig(path string)

Purpose:
Load configuration file.

Inputs:
path string

Outputs:
Config
error

GetConfig()

Purpose:
Return global configuration instance.

Inputs:
None

Outputs:
Config

---

## models

Purpose:
Application data models.

Structures

Song

Fields:
ID
Title
Author
Path
Duration
Tags

Playlist

Fields:
ID
Name
Songs

SearchResult

Fields:
Title
Author
URL

logger format
[TIMESTAMP] [LEVEL] [FUNCTION] message
2026-03-04T15:32:10Z ERROR LoadConfig config file not found
