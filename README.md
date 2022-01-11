# Music CLI

[![NPM version](https://img.shields.io/npm/v/@karizma/music?style=flat-square)](https://www.npmjs.com/package/@karizma/music) [![NPM downloads per week](https://img.shields.io/npm/dw/@karizma/music?color=blue&style=flat-square)](https://www.npmjs.com/package/@karizma/music)

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
  - [Android](#android)
- [Plans](#plans)

## About

Simple music command line tool mainly for quick and robust querying. Works with local audio files. Does not include a tui, or interactive mode. Uses vlc internally.

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

### Configuration

In your config file (differs for OSes; run `music get-config-path`)

The directory:

<!-- prettier-ignore -->
- MacOS: `~/Library/Preferences/music-cli-nodejs`
- Windows: `%APPDATA%\music-cli-nodejs\Config` (example: `C:\Users\USERNAME\AppData\Roaming\music-cli-nodejs\Config`)
- Linux: `~/.config/music-cli-nodejs` (or `$XDG_CONFIG_HOME/music-cli-nodejs`)

And then in that directory, you need to make a config.json

### Folder Structure

Any file in your music folder will be considered when querying.

```text
~/Music/
    Artist1/
        x.mp3
        y.m4a
    Category/
        z.mp3
```

### Playing Music

When filtering, the string that's tested is the full path to the file minus your music path.

For example, `~/Music/Mac Miller/Objects In The Mirror.m4a` would use `Mac Miller/Objects In The Mirror.m4a`.

Filtering:

Songs must match at least one term and not match any negation term.

A term is any of the positional arguments, and a negation term is a term that starts with `!`.

Example: `music blackbear !bad`

both blackbear and bad are terms, but bad is a negation term. This would match any song that has
blackbear in the title (or parent folder) except if it has the word 'bad' in the title (or parent folder).

Should note, in bash `!` is a special character, so you will need to escape it, replacing it with `\!`.

In a term you can use the symbols `#` to specify another text that must be matched, and `,` to specify an alternative text that may be matched.

`music mac#objects` => open all songs that have the words `mac` and `objects` in them

`music sad,bad` => open all songs that have either the word `sad` or `bad` in the title

When combining these, the string is split by `#` first, and then `,`.
`music mac#blue,objects` => open all songs that have the words `mac` in them,
and has either `blue` or `objects` in it

Other examples:

`music` => open all songs

`music sad` => open all songs that have the word `sad` in the title

`music blackbear !bad` => open all songs that have the word `blackbear` in them
but does not have the word `bad`.

Flairs:

`--dry-run | -d` => dry run, show the results, don't actually open vlc

`--dry-paths | -p` => only output all the matching songs, absolute path. This might help with some scripts.

`--limit {num} | -l` => limit the amount of songs played

`--play-new-first | --pnf` => play by newest

`--delete-old-first | --dof` => when used with `--limit`, prioritizes the newest songs from the list

`--new | -n` => `--delete-old-first` and `--play-new-first`

### Installing music

`music install "https://www.youtube.com/watch?v=K4DyBUG242c" ncs` => download from youtube

1st positional argument is the youtube link or id, the second is the folder name.
The folder name can be pretty loose in comparasion to the real name. Essentially
it's case insensitive and it replaces spaces with dashes (-).

Note: this program does not support piracy.

Flairs:

`--format | -f` => specify what format to download with, default is m4a. All the allowed formats are just what ytdl allows. Currently it is `3gp`, `aac`, `flv`, `m4a`, `mp3`, `mp4`, `ogg`, `wav`, `webm`
`--ytdl-args | -y` => specify any ytdl args to add to the command, example: `--ytdl-args "-4"`

### Auto Completion

If you are using bash you can add the following in your `.bashrc`. I think it's pretty
thorough but if you think there should be more you can create an issue.

```bash
_music_completions()
{
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    local install_command=false

    for i in "${COMP_WORDS[@]}"
    do
        if [ "$i" = "install" ] || [ "$i" = "i" ] && [ "${COMP_WORDS[COMP_CWORD]}" != "i" ]; then
            install_command=true
            break
        fi
    done

    if [ "$prev_word" = "install" ] || [ "$prev_word" = "i" ]; then
        COMPREPLY=( $(compgen -W "https://www.youtube.com/watch?v=" -- ${cur_word}) )
    elif [ "$prev_word" = "--format" ] || [ "$prev_word" = "-f" ]; then
        COMPREPLY=( $(compgen -W "3gp aac flv m4a mp3 mp4 ogg wav webm" -- ${cur_word}) )
    elif [ "$install_command" = true ]; then
        # depending how up to date you want this to be, you can set this variable outside of
        # this function (global scope). It's still pretty fast for me so I personally won't
        local SONGS_SUB_DIRS=$(basename -a ~/Music/*/ | sed 's/ /-/g' | awk '{print tolower($0)}' | tr '\n' ' ')
        COMPREPLY=( $(compgen -W "${SONGS_SUB_DIRS[*]}--format --ytdl-args -f -y" -- ${cur_word}) )
    elif [ "$prev_word" = "--sort-type" ]; then
        COMPREPLY=( $(compgen -W "a c m" -- ${cur_word}) )
    elif [ "$prev_word" = "--songs-path" ] || [ "$prev_word" = "--vlc-path" ]; then
        COMPREPLY=()
    else
        local generic_options="install play get-config-path --help --version --dry-paths --play-new-first --delete-old-first --persist --vlc-path --sort-type --songs-path --dry-run --limit --new --pnf --dof --no-persist -h -n -d -l -p"
        COMPREPLY=( $(compgen -W "${generic_options}" -- ${cur_word}) )
    fi

    return 0
}
```

### Android

If you want to use this on android, you can, but it's not as great.
Basically you can use the filtering aspect of this program, and copy all the
files to a directory, and then play that directory on vlc.

You can also make a playlist with just that directory, which is what I do.

You can also use the install command.

You will need termux and the vlc app downloaded.

1. Install nodejs `pkg install nodejs`
2. Copy `android-termux-mx` to `/data/data/com.termux/files/usr/bin`
3. Make mx executable with `chmod +x /data/data/com.termux/files/usr/bin/mx`

```bash
pkg install nodejs
curl https://raw.githubusercontent.com/karizma/music-cli/main/android-termux-mx > /data/data/com.termux/files/usr/bin/mx
chmod +x /data/data/com.termux/files/usr/bin/mx
```

## Plans

<!-- prettier-ignore -->
- Faster, (currently uses a lot of sync functions)
- Tests
- Config
  - Colors
  - Symbols (`!`, `#`, `,`)
- Better documentation
- Flairs
  - `--old | -o` sort by old
  - `--delete-new-first | --dnf` prioritize old songs
- Playlists?
- Tags?
  - Maybe using file metadata, having to add it manually would be a pain
- Command: `music ls`
