package spotify

import (
	"fmt"
	"os"

	"github.com/kitesi/music/commands/tags"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

type SetOriginArgs struct {
	debug     bool
	musicPath string
}

func SetOriginSetup() *cobra.Command {
	args := SetOriginArgs{}
	config, err := utils.GetConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}

	command := &cobra.Command{
		Use:   "set-origin <tag> [origin]",
		Short: "Set the spotify playlist/album that should be associated with a tag. If no origin is provided, delete any association with that tag",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := setupOriginRunner(positional, &args); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
	}

	command.Flags().BoolVarP(&args.debug, "debug", "d", false, "Print debug information")
	command.Flags().StringVarP(&args.musicPath, "music-path", "m", config.MusicPath, "the music path to use")
	return command
}

func setupOriginRunner(positional []string, args *SetOriginArgs) error {
	tag := positional[0]
	origin := ""

	if len(positional) == 2 {
		origin = positional[1]
	}

	tags, err := tags.GetStoredTags(args.musicPath)

	if err != nil {
		return err
	}

	if _, ok := tags[tag]; !ok {
		return fmt.Errorf("tag %s does not exist", tag)
	}

	// ignore error and use default
	config, _ := utils.GetConfig()

	if config.TagPlaylistAssociations == nil {
		config.TagPlaylistAssociations = make(map[string]string)
	}

	if origin == "" {
		if config.TagPlaylistAssociations[tag] == "" {
			return fmt.Errorf("no association for tag %s", tag)
		} else {
			delete(config.TagPlaylistAssociations, tag)
		}
	} else {
		config.TagPlaylistAssociations[tag] = origin
	}

	if err := utils.WriteConfig(config); err != nil {
		return err
	}

	return nil
}
