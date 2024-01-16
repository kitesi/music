package cmd

import (
	"github.com/kitesi/music/commands/lastfm"
	"github.com/kitesi/music/commands/play"
	"github.com/kitesi/music/commands/tags"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "music",
	Version: "1.0.0",
}

func Execute() {
	rootCmd.AddCommand(play.Setup())
	rootCmd.AddCommand(tags.Setup())
	rootCmd.AddCommand(lastfm.Setup())

	rootCmd.AddGroup(&cobra.Group{
		ID:    "generic",
		Title: "Generic Commands",
	})

	rootCmd.SetCompletionCommandGroupID("generic")
	rootCmd.SetHelpCommandGroupID("generic")

	rootCmd.Execute()
}
