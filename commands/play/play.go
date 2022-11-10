package play

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/djherbis/times"
	"github.com/spf13/cobra"

	"github.com/kitesi/music/commands/tags"
	stringUtils "github.com/kitesi/music/string-utils"
)

type Song struct {
	stat times.Timespec
	path string
}

type PlayArgs struct {
	dryRun           bool
	dryPaths         bool
	random           bool
	new              bool
	playNewFirst     bool
	skipOldFirst     bool
	persist          bool
	appendToPlaylist bool
	live             bool
	editor           bool
	tags             []string
	addToTag         string
	setToTag         string
	vlcPath          string
	sortType         string
	musicPath        string
	limit            int
	skip             int
}

func addFlags(playCmd *cobra.Command, args *PlayArgs) {
	playCmd.Flags().BoolVarP(&args.dryRun, "dry-run", "d", false, "dry run")
	playCmd.Flags().BoolVarP(&args.dryPaths, "dry-paths", "p", false, "dry paths")
	playCmd.Flags().BoolVarP(&args.random, "random", "z", false, "play by random")
	playCmd.Flags().BoolVarP(&args.new, "new", "n", false, "play by new and skip old first")
	playCmd.Flags().BoolVar(&args.playNewFirst, "play-new-first", false, "play by new")
	playCmd.Flags().BoolVar(&args.skipOldFirst, "skip-old-first", false, "skip old first when there is a limit")
	playCmd.Flags().BoolVarP(&args.persist, "persist", "", false, "persist the command instance")
	playCmd.Flags().BoolVar(&args.appendToPlaylist, "append", false, "append to playlist rather than jumping")
	playCmd.Flags().BoolVar(&args.live, "live", false, "go into live query results mode")
	playCmd.Flags().BoolVarP(&args.editor, "editor", "e", false, "pipe to $EDITOR before playing")

	playCmd.Flags().StringVarP(&args.addToTag, "add-to-tag", "a", "", "add to tag")
	playCmd.Flags().StringVar(&args.setToTag, "set-to-tag", "", "set to tag")
	playCmd.Flags().StringVar(&args.vlcPath, "vlc-path", "vlc", "path to vlc executable to use")
	playCmd.Flags().StringVar(&args.musicPath, "music-path", "", "path to songs")

	playCmd.Flags().StringArrayVarP(&args.tags, "tags", "t", []string{}, "required tags to match")

	playCmd.Flags().StringVarP(&args.sortType, "sort-type", "s", "m", "timestamp to use when sorting by time (a|m|c)")

	playCmd.Flags().IntVarP(&args.limit, "limit", "l", -1, "limit amount of songs played")
	playCmd.Flags().IntVar(&args.skip, "skip", 0, "songs to skip from the start")
}

func generateCommand() (*cobra.Command, *PlayArgs) {
	args := PlayArgs{}

	playCmd := &cobra.Command{
		Use:   "play [terms..]",
		Short: "play music",
		Long:  "play music",
	}

	addFlags(playCmd, &args)
	return playCmd, &args
}

func setDefaultMusicPath(args *PlayArgs) error {
	defaultMusicPath, err := stringUtils.GetDefaultMusicPath()

	if err != nil {
		return err
	}

	args.musicPath = defaultMusicPath
	return nil
}

func Setup(rootCmd *cobra.Command) {
	playCmd, args := generateCommand()

	playCmd.Run = func(_ *cobra.Command, terms []string) {
		if err := playRunner(args, terms); err != nil {
			log.SetFlags(0)
			log.Fatal(err)
		}
	}

	rootCmd.AddCommand(playCmd)
}

func playRunner(args *PlayArgs, terms []string) error {
	if args.live {
		return liveQueryResults()
	}

	if err := setDefaultMusicPath(args); err != nil {
		return err
	}

	if len(terms) == 0 && args.limit != 0 && !args.dryPaths && !args.playNewFirst && !args.new && !args.editor && len(args.tags) == 0 {
		fmt.Println("Playing all songs")
		return runVLC(args, []string{"--recursive=expand", args.musicPath})
	}

	songs, err := getSongs(args, terms)

	if err != nil {
		return err
	}

	if len(songs) == 0 {
		fmt.Println("Didn't match anything")
		return nil
	}

	if args.addToTag != "" {
		if err := tags.ChangeSongsInTag(args.musicPath, args.addToTag, songs, true); err != nil {
			return nil
		}
	}

	if args.setToTag != "" {
		if err := tags.ChangeSongsInTag(args.musicPath, args.setToTag, songs, false); err != nil {
			return err
		}
	}

	if args.dryPaths {
		for _, s := range songs {
			fmt.Println(s)
		}

		return nil
	}

	isPlayingAll := args.limit == -1 && len(terms) == 0 && len(args.tags) == 0 && !args.editor

	if isPlayingAll {
		fmt.Println("Playing all songs")
	} else {
		fmt.Printf("Playing [%d]\n", len(songs))

		for _, s := range songs {
			fmt.Printf("- %s\n", stringUtils.GetBareSongName(s, args.musicPath))
		}
	}

	return runVLC(args, songs)
}

func getSongs(args *PlayArgs, terms []string) ([]string, error) {
	if args.sortType != "a" && args.sortType != "c" && args.sortType != "m" {
		return nil, errors.New("invalid --sort-type, expected value of 'a'|'c'|'m'")
	}

	songs := []Song{}
	canEndEarly := !args.new && !args.skipOldFirst && !args.playNewFirst

	storedTags, err := tags.GetStoredTags(args.musicPath)

	if err != nil {
		return nil, err
	}

	if args.limit > 0 && args.skip > 0 {
		args.limit += args.skip
	}

	var walk = func(fileName string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return nil
		}

		if canEndEarly && len(songs) == args.limit {
			return nil
		}

		if doesSongPass(args, storedTags, terms, strings.ToLower(fileName)) {
			stat, err := times.Stat(fileName)

			if err != nil {
				return err
			}

			songs = append(songs, Song{stat, fileName})
		}

		return nil
	}

	err = filepath.WalkDir(args.musicPath, walk)

	if err != nil {
		return nil, err
	}

	if len(songs) == 0 {
		return []string{}, nil
	}

	if args.new || args.skipOldFirst {
		sortByNew(songs, args.sortType)
	}

	if args.skip > 0 {
		songs = songs[args.skip:]
	}

	if args.limit > 0 && len(songs) > args.limit {
		songs = songs[:args.limit]
	}

	// !new && !skipOldFirst to make sure we don't uselessly sort again
	if args.playNewFirst && !args.new && !args.skipOldFirst {
		sortByNew(songs, args.sortType)
	}

	flatSongs := make([]string, len(songs))

	for i, s := range songs {
		flatSongs[i] = s.path
	}

	if args.editor {
		editedSongs, err := editSongList(flatSongs)

		if err != nil {
			return nil, err
		}

		flatSongs = editedSongs
	}

	return flatSongs, nil
}

func runVLC(args *PlayArgs, vlcArgs []string) error {
	if args.dryRun {
		return nil
	}

	if args.new || args.playNewFirst {
		vlcArgs = append(vlcArgs, "--no-random")
	} else if args.random {
		vlcArgs = append(vlcArgs, "--random")
	}

	if args.appendToPlaylist {
		vlcArgs = append(vlcArgs, "--playlist-enqueue")
	} else {
		vlcArgs = append(vlcArgs, "--no-playlist-enqueue")
	}

	var err error

	if args.persist {
		err = exec.Command("vlc", vlcArgs...).Run()
	} else {
		err = exec.Command("vlc", vlcArgs...).Start()
	}

	return err
}
