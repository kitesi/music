# Music CLI

<!-- prettier-ignore -->
- [About](#about) 
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
  - [Folder Structure](#folder-structure)
  - [Playing Music](#playing-music)
  - [Installing music](#installing-music)
  - [Auto Completion](#auto-completion)
- [Plans](#plans)

## About

Simple music command line tool that works with local files. Does not include a tui, or interactive mode.

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

## Usage

### Folder Structure

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

This program does not currently check if the path follows the format.

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

3rd parameter is the youtube link or id, the 4th parameter is the folder name.
The folder name can be pretty loose in comparasion to the real name. Essentially
it's case insensitive and it replaces spaces with dashes (-).

### Auto Completion

If you are using bash you can add something similar to this in your `.bashrc`:

```bash
_music_completions()
{

    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local second_prev_word="${COMP_WORDS[COMP_CWORD - 2]}"

    if [ "$second_prev_word" == "install" ]; then
        local IFS=$'\n'
        COMPREPLY=( $(compgen -W "${SONGS_SUB_DIRS[*]}" -- ${cur_word}) )
    else
        local generic_options="install --help --new --dry-run --limit --new -h -n -d -l"
        COMPREPLY=( $(compgen -W "${generic_options}" -- ${cur_word}) )
    fi


    # if no match was found, fall back to filename completion
    if [ ${#COMPREPLY[@]} -eq 0 ]; then
      COMPREPLY=()
    fi

    return 0
}

complete -F _music_completions music
```

## Plans

<!-- prettier-ignore -->
- Faster, (currently uses a lot of sync functions)
- Tests
- Config
  - Folder path
  - Colors
- Playlists?
- Tags?
  - Maybe using file metadata, having to add it manually would be a pain
