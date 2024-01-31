package cmd

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/shlex"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

type InstallArgs struct {
	format    string
	ytdlArgs  string
	name      string
	musicPath string
	editor    bool
}

func init() {
	args := InstallArgs{}

	installCmd := &cobra.Command{
		Use:   "install <id> <folder>",
		Short: "Install music from youtube id or url",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := installRunner(&args, positional); err != nil {
				log.SetFlags(0)
				log.Fatal(err)
			}
		},
	}

	installCmd.Flags().StringVarP(&args.format, "format", "f", "m4a", "format to install to")
	installCmd.Flags().StringVarP(&args.ytdlArgs, "ytdl-args", "y", "", "additional arguments to send to youtube-dl")
	installCmd.Flags().StringVarP(&args.name, "name", "n", "", "the file name to install to")
	installCmd.Flags().StringVarP(&args.musicPath, "music-path", "m", "", "the music path to use")

	rootCmd.AddCommand(installCmd)
}

func installRunner(args *InstallArgs, positional []string) error {
	id := positional[0]
	folder := positional[1]

	if args.musicPath == "" {
		defaultMusicPath, err := utils.GetDefaultMusicPath()

		if err != nil {
			return err
		}

		args.musicPath = defaultMusicPath
	}

	possibleFolders, err := os.ReadDir(args.musicPath)

	if err != nil {
		return err
	}

	re := regexp.MustCompile(`\s+`)
	adjustedFolder := formatFolderName(folder, re)
	selectedFolder := ""

	for _, f := range possibleFolders {
		if !f.IsDir() {
			continue
		}

		if formatFolderName(f.Name(), re) == adjustedFolder {
			if selectedFolder != "" {
				return errors.New("folder matches more than one folder")
			}

			selectedFolder = f.Name()
		}
	}

	if selectedFolder == "" {
		return errors.New("invalid folder: " + folder)
	}

	youtubeURL := id

	if !strings.HasPrefix(id, "https://") {
		youtubeURL = "https://www.youtube.com/watch?v=" + id
	}

	outputTemplate := "%(title)s.%(ext)s"

	if args.name != "" {
		outputTemplate = args.name + ".%(ext)s"
	}

	if args.editor {

	}

	finalCmdArgs := []string{
		"--no-playlist", "-f", args.format, "-o", filepath.Join(args.musicPath, selectedFolder, outputTemplate),
	}

	if args.ytdlArgs != "" {
		a, err := shlex.Split(args.ytdlArgs)

		if err != nil {
			return err
		}

		finalCmdArgs = append(finalCmdArgs, a...)
	}

	finalCmdArgs = append(finalCmdArgs, "--", youtubeURL)

	cmd := exec.Command("youtube-dl", finalCmdArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func formatFolderName(folder string, re *regexp.Regexp) string {
	return re.ReplaceAllString(strings.ToLower(folder), "-")
}
