package cmd

import (
	"log"

	"github.com/kitesi/music/commands/play"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "music",
}

func Execute() {
	play.Setup(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
