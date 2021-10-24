# 1.2.0 (10/24/21)

<!-- prettier-ignore -->
- add flairs on main command
  - `--play-new-first | --pnf`, play newest songs first
  - `--delete-old-first | --dof`, when filtering, remove the oldest songs first
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
