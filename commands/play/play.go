package play

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/djherbis/times"
	"github.com/spf13/cobra"
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

func Setup(rootCmd *cobra.Command) {
	var args = PlayArgs{}

	var playCmd = &cobra.Command{
		Use:   "play [terms..]",
		Short: "play music",
		Long:  "play music",
		Run: func(cmd *cobra.Command, terms []string) {
			run(args, terms)
		},
	}

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

	playCmd.Flags().IntVarP(&args.limit, "limit", "l", -1, "dry run")
	playCmd.Flags().IntVar(&args.skip, "skip", 0, "songs to skip from the start")

	rootCmd.AddCommand(playCmd)

}

func run(args PlayArgs, terms []string) {
	log.SetFlags(0)

	if args.sortType != "a" && args.sortType != "c" && args.sortType != "m" {
		log.Fatal("Error: invalid --sort-type, expected value of 'a'|'c'|'m'")
	}

	if args.live {
		return
	}

	if args.musicPath == "" {
		dirname, err := os.UserHomeDir()

		if err != nil {
			log.Fatal(err)
		}

		args.musicPath = filepath.Join(dirname, "Music")
	}

	if len(terms) == 0 && args.limit != 0 && !args.dryPaths && !args.playNewFirst && !args.new && !args.editor && len(args.tags) == 0 {
		fmt.Println("Playing all songs")
		runVLC(args, []string{"--recursive=expand", args.musicPath})
		return
	}

	songs := getSongs(args, terms)

	if args.addToTag != "" {
		changeSongsInTag(args.musicPath, args.addToTag, songs, true)
	}

	if args.setToTag != "" {
		changeSongsInTag(args.musicPath, args.setToTag, songs, false)
	}

	if args.dryPaths {
		for _, s := range songs {
			fmt.Println(s.path)
		}

		return
	}

	vlcArgs := []string{}
	isPlayingAll := args.limit == -1 && len(terms) == 0 && len(args.tags) == 0 && !args.editor

	if isPlayingAll {
		fmt.Println("Playing all songs")
	} else {
		fmt.Printf("Playing [%d]\n", len(songs))
	}

	for _, s := range songs {
		if !isPlayingAll {
			fmt.Printf("- %s\n", strings.Replace(s.path, args.musicPath+"/", "", 1))
		}

		vlcArgs = append(vlcArgs, s.path)
	}

	runVLC(args, vlcArgs)
}

func getSongs(args PlayArgs, terms []string) []Song {
	songs := []Song{}
	canEndEarly := !args.new && !args.skipOldFirst && !args.playNewFirst

	storedTags := getStoredTags(args.musicPath)

	if args.limit > 0 && args.skip > 0 {
		args.limit += args.skip
	}

	var walk = func(fileName string, dirEntry fs.DirEntry, err error) error {
		if dirEntry.IsDir() {
			return nil
		}

		if canEndEarly && len(songs) == args.limit {
			return nil
		}

		if doesSongPass(args, storedTags, terms, strings.ToLower(fileName)) {
			stat, statErr := times.Stat(fileName)

			if statErr != nil {
				fmt.Println(statErr)
				return nil
			}

			songs = append(songs, Song{stat, fileName})
		}

		return nil
	}

	filepath.WalkDir(args.musicPath, walk)

	if len(songs) == 0 {
		return songs
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

	if args.editor {
		return nil
	}

	return songs
}

func runVLC(args PlayArgs, vlcArgs []string) {
	if args.dryRun {
		return
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

	if err != nil {
		log.Fatal(err)
	}
}
