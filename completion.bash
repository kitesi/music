MUSIC_PATH=~/Music

_music_completions()
{
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    local subcommand=""

    for i in "${COMP_WORDS[@]}"
    do
        if [ "$i" = "install" ] || [ "$i" = "play" ] || [ "$i" = "tags" ]; then
            subcommand="$i"
            break
        fi
    done

    if [ "$subcommand" = "play" ]; then
        _music_play_completions
        return 0
    fi

    case "$subcommand" in
        play)
            _music_play_completions
            ;;
        install)
            _music_install_completions
            ;;
        tags)
            _music_tags_completions
            ;;
        *)
            COMPREPLY=( $(compgen -W "help completion play tags install --help --version" -- ${cur_word}) )
            ;;
    esac

    return 0
}

_music_install_completions() {
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    case "$prev_word" in 
        install)
            COMPREPLY=( $(compgen -W "https://www.youtube.com/watch?v=" -- "$cur_word") )
            ;;
        --format|-f)
            COMPREPLY=( $(compgen -W "3gp aac flv m4a mp3 mp4 ogg wav webm" -- "$cur_word") )
            ;;
        --music-path|-m)
            COMPREPLY=()
            ;;
        *)
            # depending how up to date you want this to be, you can set this variable outside of
            # this function (global scope). It's still pretty fast for me so I personally won't
            local SONGS_SUB_DIRS=$(basename -a $MUSIC_PATH/*/ | sed 's/ /-/g' | awk '{print tolower($0)}' | tr '\n' ' ')
            COMPREPLY=( $(compgen -W "${SONGS_SUB_DIRS[*]}--format --ytdl-args --name --editor --music-path --help" -- "$cur_word") )
            ;;
    esac
    
    return 0
}

_music_tags_completions() {
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    if [ "$prev_word" = "--music-path" ]; then
        COMPREPLY=()
        return 0
    fi

    local options="--editor --help --music-path --delete"
	options+=" $(find $MUSIC_PATH/tags/ -name \*.m3u -exec basename -s '.m3u' {} +)"

    COMPREPLY=( $(compgen -W "$options" -- "$cur_word") )
    return 0
}

_music_play_completions() {
    local generic_options="--help --append --live --editor --skip --random --tags --add-to-tag --set-to-tag --dry-paths --play-new-first --skip-old-first --persist --vlc-path --sort-type --music-path --dry-run --limit --new --no-persist"
    local cur_word="${COMP_WORDS[COMP_CWORD]}"
    local prev_word="${COMP_WORDS[COMP_CWORD - 1]}"

    case "$prev_word" in
        --sort-type|-s)
            COMPREPLY=( $(compgen -W "a c m" -- "$cur_word") ) 
            ;;
        --music-path)
            COMPREPLY=()
            ;; 
        --add-to-tag|--set-to-tag|-a)
			local tags=$(find $MUSIC_PATH/tags/ -name \*.m3u -exec basename -s '.m3u' {} +)
			COMPREPLY=( $(compgen -W "$tags" -- "$cur_word") )
            ;;
        *)
            COMPREPLY=( $(compgen -W "$generic_options" -- "$cur_word") )
            ;;
    esac

    return 0
}

complete -F _music_completions -o default music
complete -F _music_completions -o default m
complete -F _music_play_completions -o default mx
