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
- Tags
- Install music with youtube-dl

## Requirements

<!-- prettier-ignore -->
- NodeJS
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

Here's the schema:

```json
{
    "path": "~/My-Music", // string to your music path. default is your HomeDir/Music
    "pathToVLC": "~/Downloads/vlc", // path to vlc executablek. default is global vlc,
    "sortType": "aTimeMs", // aTimeMs | mTimeMs | cTimeMs, default is modified
    "persist": false // default = false
}
```

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

For example, `~/Music/Jaxsoe/Make Time For Me.m4a` would use `Jaxson/Make Time For Me.m4a`.

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

`--dry-run | -d` => dry run, show the results, don't actually play any music

`--dry-paths | -p` => only output all the matching songs, absolute path. This might help with some scripts.

`--limit {number} | -l` => limit the amount of songs played

`--play-new-first | --pnf` => play by newest

`--delete-old-first | --dof` => when used with `--limit`, prioritizes the newest songs from the list

`--new | -n` => `--delete-old-first` and `--play-new-first`

`--persist` => persist the instance of vlc through the cli

`--live` => this allows you to type out your query and get live feedback
for the songs it will play

`--editor` => option to modify song list before playing

this option will create a temporary file and then execute your ENV's
default editor and after you finish saving and exiting will read the
file content and play the songs based of it

`--vlc-path <string>` => specifies the path to vlc to use

`--songs-path <string>` => specifies the songs path to use

`--sort-type | -s <a|m|c>` => specifies the what timestamp to use (access, modified, changed)

`--skip <number>` => skip songs from the start, mainly implemented it for using it with `-n` or another
sorting option. If you use it with `-l`, the limit will still be the limit. It won't be limit - skip

For example `. -l5 --skip 2` will still result in 5 songs being the max amount.

`--tags | -t <string..>` => this will be an array of tag queries, sorta like the positional terms,
to stop the array use `--` for example `-t sad \!mid 2019 -- -l5`

Worth noting, tags are case-insensitive.

`--add-to-tag | -a <string>` => add all the valid songs to the specified tag. `-d` will not stop this.
Tags will be stored in `YOUR_MUSIC_PATH/tags.json`

`--set-to-tag` => set all the valid songs to the specified tag. If any songs exist in that tag, they will be removed `-d` will not stop this.

### Installing music

`music install "https://www.youtube.com/watch?v=K4DyBUG242c" ncs` => download from youtube

1st positional argument is the youtube link or id, the second is the folder name.
The folder name can be pretty loose in comparasion to the real name. Essentially
it's case insensitive and it replaces spaces with dashes (-).

Note: this program does not support piracy.

Flairs:

`--format | -f` => specify what format to download with, default is m4a. All the allowed formats are just what ytdl allows. Currently it is `3gp`, `aac`, `flv`, `m4a`, `mp3`, `mp4`, `ogg`, `wav`, `webm`
`--ytdl-args | -y` => specify any ytdl args to add to the command, example: `--ytdl-args "-4"`
`--name | -n` => specify the file name
`--editor | -e` => opens your editor so you can modify the title before installing. is a bit slower since it needs to fetch the title

### Auto Completion

If you are using bash you can add the following in your `.bashrc`. I think it's pretty
thorough but if you think there should be more you can create an issue.

Note: I have `mx` as an alias for `music play`
Note: You need `jq` if you want completions on `--add-to-tag|-a` or `--set-to-tag`

```bash
MUSIC_PLAY_OPTIONS="--help --version --live --editor --skip --tags --add-to-tag --set-to-tag --dry-paths --play-new-first --delete-old-first --persist --vlc-path --sort-type --songs-path --dry-run --limit --new --pnf --dof --no-persist"

_music_completions()
{
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    local is_install_command=false
    local is_play_command=false

    # not the best checking ngl, esp with the aliases "i" and "p", might remove those
    for i in "${COMP_WORDS[@]}"
    do
        if [ "$i" = "install" ] || [ "$i" = "i" ] && [ "${COMP_WORDS[COMP_CWORD]}" != "i" ]; then
            is_install_command=true
            break
        fi

        if [ "$i" = "play" ] || [ "$i" = "p" ] && [ "${COMP_WORDS[COMP_CWORD]}" != "p" ]; then
            is_play_command=true
            break
        fi
    done

    local last_word_is_install=false

    if [ "$is_play_command" = true ]; then
        _music_play_completions
        return 0
    fi

    case "$prev_word" in
        install|i)
            COMPREPLY=( $(compgen -W "https://www.youtube.com/watch?v=" -- ${cur_word}) )
            last_word_is_install=1
            ;;
        --format|-f)
            COMPREPLY=( $(compgen -W "3gp aac flv m4a mp3 mp4 ogg wav webm" -- ${cur_word}) )
            ;;
        *)
            local generic_options="install play get-config-path ${MUSIC_PLAY_OPTIONS}"
            COMPREPLY=( $(compgen -W "${generic_options}" -- ${cur_word}) )
            ;;
    esac

    if [ "$is_install_command" = true ] && [ "$last_word_is_install" = false ] ; then
        # depending how up to date you want this to be, you can set this variable outside of
        # this function (global scope). It's still pretty fast for me so I personally won't
        local SONGS_SUB_DIRS=$(basename -a ~/Music/*/ | sed 's/ /-/g' | awk '{print tolower($0)}' | tr '\n' ' ')
        COMPREPLY=( $(compgen -W "${SONGS_SUB_DIRS[*]}--format --ytdl-args --name --editor" -- ${cur_word}) )
    fi

    return 0
}

_music_play_completions() {
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    case "$prev_word" in
        --sort-type|-s)
            COMPREPLY=( $(compgen -W "a c m" -- ${cur_word}) )
            ;;
        --songs-path)
            COMPREPLY=()
            ;;
        --add-to-tag|--set-to-tag|-a)
            if [ -x "$(which jq)" ]; then
                local tags=$(jq '.[].name' <~/Music/tags.json)
                COMPREPLY=( $(compgen -W "$tags" -- ${cur_word}) )
            else
                COMPREPLY=( $(compgen -W "${MUSIC_PLAY_OPTIONS}" -- ${cur_word}) )
            fi
            ;;
        *)
            COMPREPLY=( $(compgen -W "${MUSIC_PLAY_OPTIONS}" -- ${cur_word}) )
            ;;
    esac

    return 0
}

complete -F _music_completions -o default music
complete -F _music_play_completions -o default mx
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

Now you can run mx like so: `mx jacob`

## Plans

<!-- prettier-ignore -->
- Flairs
  - `--old | -o` sort by old
  - `--delete-new-first | --dnf` prioritize old songs
- Playlists?
