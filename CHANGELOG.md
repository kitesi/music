# 1.11.0 (8/16/22)

<!-- prettier-ignore -->
- fix live-search-query
  - no query worked as terms was not set by default
  - if a song's name length was greater than the terminal width, it would mess up the cursor position
  - fix -n not working
- internal: remove setting config values in middleware, and use the default option in the builder (play-music.ts)

# 1.10.0 (8/09/22)

<!-- prettier-ignore -->
- add `--skip <number>` => skip songs from the start, mainly implemented it for using it with `-n` or another
- add tagging system

`--tags | -t <string..>` => this will be an array of tag queries, sorta like the positional terms,
to stop the array use `--` for example `-t sad \!mid 2019 -- -l5`

Worth noting, tags are case-insensitive.

`--add-to-tag | -a <string>` => add all the valid songs to the specified tag. `-d` will not stop this.

`--set-to-tag` => set all the valid songs to the specified tag. If any songs exist in that tag, they will be removed `-d` will not stop this.

# 1.9.0 (7/11/22)

<!-- prettier-ignore -->
- moved `open-graph-scraper` to dependency intead of dev

# 1.8.0 (7/11/22)

<!-- prettier-ignore -->
- add `--live` option, this allows you to type out your query and get live feedback
for the songs it will play
- add `-e | --editor` option to modify song list before playing

  this option will create a temporary file and then execute your ENV's
  default editor and after you finish saving and exiting will read the
  file content and play the songs based of it

  inspired by rangers `:bulkrename`

- add `-e | --editor` option to modify song name before installing
  
  this will fetch the title using open-graph-scraper and then just like before 
  create a temporary file and use your ENV's editor. 

  ".%(ext)s" is added afterwards.

  can not be used in combination with `-n | --name`
- add alias `-s` for `--sort-type`
- fix `--songs-path` argument
- add completions for `mx` alias if you choose to have that

# 1.7.0 (1/23/22)

<!-- prettier-ignore -->
- add `-n | --name` to install command, so you can specify the file name

# 1.6.1 (1/11/22)

<!-- prettier-ignore -->
- add the new bash completion to readme

# 1.6.0 (1/11/22)

<!-- prettier-ignore -->
- change default command to `play` and `p`
- better bash completion

# 1.5.0 (11/28/21)

<!-- prettier-ignore -->
- fix 'playing all' showing when `-l` was specified but no terms were specified
- add config options on command line and global config file
  - `--sort-type` `.sortType`, options on command line are `a`, `c`, `m`. Options on config file are `atimeMs`, `ctimeMs`. and `mtimeMs`
  - `--persist` `.persist` decides if the program should still run. If this option is selected, once the program terminates, so does vlc.
  - `--vlc-path` `.pathToVLC` allows you to set the path to the executable. Default is assuming it's in the global path
- add option `--songs-path`, which allows you to specify which folder to go off of
- better walk function (checks if an item is a file or folder)

# 1.4.0 (11/14/21)

<!-- prettier-ignore -->
- fix `-d` not working when used in the same flair as other shorthand alises. Example: `-dn` would not work. This is because the code simply checked if `-d` or `--dry-run` was in the command line arguments. Now it checks what yargs resolves the value to.
- allow `-n` with all songs / no terms
- make it so no positive search terms with negative search terms defaults to match. For example `music !the` before would match nothing, now it matches every song that doesn't have the word 'the'.

# 1.3.0 (11/2/21)

<!-- prettier-ignore -->
- fix issue where `config.js` was not being installed for some reason, so the program didn't work at all
- add `--dry-paths | -p` on main command so you can get a list of all the songs that match

# 1.2.0 (10/24/21)

<!-- prettier-ignore -->
- add flairs on main command
  - `--play-new-first | --pnf`, play newest songs first
  - `--delete-old-first | --dof`, when filtering, prioritize the newest songs first
- add flairs on install command
  - `--format | -f`, specify what format to download with, default is m4a like before
  - `--ytdl-args | -y`, specify any ytdl args to add to the command, example `--ytdl-args "-4"`
- fix issue when downloading music (folder paths with spaces weren't being replaced with `-`)
- use `.ctimeMs` instead of `.mtimeMs`

# 1.1.0 (10/20/21)

<!-- prettier-ignore -->
- add ability to configure music path
- add completion script to readme

# 1.0.1 (10/10/21)

<!-- prettier-ignore -->
- remove music folder from being included in the song path when filtering
  - if your music folder was `/home/eminem/Music` then `music eminem#sad` would
  essentially be `music sad`
