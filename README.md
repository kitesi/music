# Music CLI

<!-- prettier-ignore -->
- [About](#about)
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Folder Structure](#folder-structure)
- [Usage](#usage)
  - [Playing Music](#playing-music)
  - [Installing music](#installing-music)
- [Plans](#plans)

## About

Simple music command line tool that works with local files.
Does not include a tui, or interactive mode.

## Features

<!-- prettier-ignore -->
- Lot of filtering options
  - Dry run to test what matches
- Install music with youtube-dl

## Requirements

<!-- prettier-ignore -->
- NodeJS (reimplementation in rust in future)
- VLC
- youtube-dl (if you plan on installing music with this cli)

## Installation

```shell
npm install -g @karizma/music
```

## Folder Structure

Your `~/Music` folder should have folders only, and in those folders you should
have the music files:

```text
~/Music/
    Artist1/
        x.mp3
        y.m4a
    Category/
        z.mp3
```

This program does not currently check if it follows it.

## Usage

### Playing Music

When filtering, the string that's compared is the full path to the file minus
`~/Music`.

For example, `~/Music/Mac Miller/Objects In The Mirror.m4a` would use `Mac Miller/Objects In The Mirror.m4a`.

`music` => open all songs

`music sad` => open all songs that have the word `sad` in the title

`music sad,bad` => open all songs that have the word `sad` or `bad` in the title

`music blackbear !bad` => open all songs that have the word `blackbear` in them
but does not have the word `bad`

`music mac#objects` => open all songs that have the words `mac` and `objects` in
them

`music mac#blue,objects` => open all songs that have the words `mac` in them,
and has either `blue` or `objects` in it

`--dry-run | -d` => dry run, show the results, don't actually open vlc

`--new | -n` => sort by new, play by newest as well

`--limit {num} | -l` => limit the amount of songs played. works in combination of `-n`

### Installing music

`music install https://www.youtube.com/watch?v=jsdoi309asd mac-miller` =>
download from youtube

## Plans

<!-- prettier-ignore -->
- Faster, (currently uses a lot of sync functions)
- More checking
- Tests
- Config
  - Folder path
  - Colors
- Playlists?
- Tags?
  - Maybe using file metadata, having to add it manually would be a pain
