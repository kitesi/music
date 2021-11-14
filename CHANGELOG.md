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
