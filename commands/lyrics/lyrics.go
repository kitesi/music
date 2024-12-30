package lyrics

import (
	"fmt"
	"os"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

type LyricsArgs struct {
	debug bool
}

const (
	geniusAccessToken = ""
	geniusAPIURL      = "https://api.genius.com"
)

func Setup() *cobra.Command {
	args := LyricsArgs{}

	lyricsCommand := &cobra.Command{
		Use:   "lyrics",
		Short: "Get lyrics for the current song",
		Long:  "Get lyrics for the current song",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := lyricsRunner(&args); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
	}

	config, err := utils.GetConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}

	lyricsCommand.Flags().BoolVar(&args.debug, "debug", config.Debug, "set debug mode")

	return lyricsCommand
}

func lyricsRunner(args *LyricsArgs) error {
	songMetadata, err := utils.GetCurrentPlayingSong()

	if err != nil {
		return err
	}

	fmt.Println(songMetadata)

	return nil
}
