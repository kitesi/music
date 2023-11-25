# Music CLI

-   [About](#about)
-   [Installation](#installation)
-   [Requirements](#requirements)
-   [Usage](#usage)
    -   [Playing Music](#playing-music)
    -   [Tags](#tags)
    -   [Installing music](#installing-music)
    -   [Auto Completion](#auto-completion)

## About

This is a simple command line tool to help with music-related tasks.
The primary usage is the querying it provides which allows you to quickly
select the songs you want to play. This is **not** a music player, it does
not provide a TUI or GUI, and it uses VLC internally.

For playlists and grouping of songs, it has a tag system. The tags and data are stored in your `$MUSIC_PATH/tags.json`.

This program does not support piracy; you should have the rights to all your files.

## Requirements

-   VLC
-   youtube-dl (if you plan on installing music)
-   jq (if you want to use auto-completion for tags)

## Installation

## Usage

Each command takes in a music-path argument, which defaults to `$HOME/Music`.
I recommend a folder structure of:

```text
~/Music/
    Artist1/
        x.mp3
        y.m4a
    Category/
        z.mp3
```

But this is not a necessary, as any file in your music path will be considered.

Files should follow some basic file naming rules: no new lines, no crazy characters, etc.

### Playing Music

You can play music with the `play` command which will take in
any amount of positional arguments, these are called terms.

A term can have a "!" prefix, meaning it's a negation term, and anything
that matches that term fails.

If no term is provided, the program will spawn VLC with the directory
and `--recursive=expand`.

Otherwise, a song will have to match at least one of the terms and none
of the negation terms.

A term can have required sections and one-of sections, specified with "#" and
"," respectively.

When querying, the string that's tested is the lowercase full path to the file
minus your music path.

For example, `~/Music/Jaxson/Make Time For Me.m4a` would use
`jaxson/make time for me.m4a`.

Example of usage:

```shell
music play tonight monday#mornings care,bear,say make#you,me#believe \!joe
```

There are four terms here:

-   `tonight`
-   `monday#mornings`
-   `care,bear,say`
-   `make#you,me#believe`
-   `\!joe`

A song will have to match one of those terms and not have the substring "joe".

To match the first term, a song simply needs to have the word "tonight" in the path.

To match the second term, a song needs to have the words "monday" and "mornings" in its path (not necessarily next to each other).

To match the third term, a song needs to have any of the following words: "care", "bear" or "say" in its path.

To match the fourth term, a song needs to have "make", either "you" or "me", and "believe" in its path.

To match the fifth term, a song simply needs to not have the word "joe" in its path. The backlash is there because `!` is a special character in bash.

When combining these, the string is split by `#` first, and then `,`.

### Tags

Tags are a way to group music. You can use it for playlists, genres or whatever. Tags will be stored in `$MUSIC_PATH/tags.json`

You can view your tags with `music tags`. If you want to see the songs in a tag
use `music tags <tag>`.

If you want to delete a tag use `--delete` or `-d`. Edit a tag or the `tags.json`
with `--edit` or `-e`.

The intended way to add songs to a tag is to query the songs with `music play`
and then using `--add-to-tag | -a <tag>` or `--set-to-tag | -s <tag>`.

### Installing music

`music install "https://www.youtube.com/watch?v=K4DyBUG242c" ncs` => download from youtube

The first positional argument is the link to download or a youtube video id. The
second is the child folder name of your music path to download to. The folder
name can be pretty loose in comparison to the real name. It's case-insensitive
and replaces spaces with dashes (-).

For example, if you had a folder named "Kite Hughes", you would use "kite-hughes".

### Other Cool Features

You can use `music play --live` to get a live query search of your songs.
I personally bind this command to a keybinding of `Ctrl+Alt+m`

### Auto Completion

This tool uses [cobra](https://github.com/spf13/cobra) which provides a
completion command you can use to generate completions.

It works fine, but it doesn't have reactive completion to a few things:
tags, music subdirectory install, format, and sort-type.

The cobra provided completion is also a bit more descriptive than I personally
like, which is why I use my own personal bash completion. It can be found in
`./completion.bash`

Note: I have `m` as an alias for `music` and `mx` as an alias for `music play`

Note: You need `jq` if you want completions on `--add-to-tag|-a` or
`--set-to-tag` or `music tags [tag]`

It's also more static/hard-coded, so a bit more error-prone/inaccurate.

### Configuration

There is no configuration file currently. I would suggest setting up an alias
with your desired options.
