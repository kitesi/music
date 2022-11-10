package cmd

import (
	"github.com/kitesi/music/commands/play"
	"github.com/kitesi/music/commands/tags"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "music",
}

func Execute() {
	play.Setup(rootCmd)
	tags.Setup(rootCmd)

	rootCmd.Execute()
}
