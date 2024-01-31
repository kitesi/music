package lastfm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

const (
	SUGGEST_API_END_POINT = "https://www.last.fm/player/station/user/%s/recommended"
)

func SuggestSetup() *cobra.Command {
	args := LastfmSuggestArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "suggest [username]",
		Short: "Suggest songs using lastfm's station",
		Long:  "Suggest songs using lastfm's station. This only requires a username, no authentication. You should either provide a username in the first argument, or have it set in the credentials file.",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, positional []string) {
			username := ""

			if len(positional) == 1 {
				username = positional[0]
			}

			if err := suggestRunner(&args, username); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}

				fmt.Fprintf(os.Stderr, "\nIf this was a request error, go to the url on a browser and check (first replace %%s with your username):\n%s\n", SUGGEST_API_END_POINT)
			}
		},
	}

	lastfmCommand.Flags().BoolVar(&args.debug, "debug", false, "set debug mode")
	lastfmCommand.Flags().BoolVar(&args.printUrls, "print-urls", true, "print the urls along with the song name")
	lastfmCommand.Flags().IntVarP(&args.limit, "limit", "l", 10, "limit the number of songs to suggest")
	lastfmCommand.Flags().StringVarP(&args.musicPath, "music-path", "m", "", "the music path to use")
	lastfmCommand.Flags().StringVarP(&args.format, "format", "f", "bestaudio[ext=m4a]", "the format to install")
	lastfmCommand.Flags().BoolVarP(&args.install, "install", "i", false, "install the files to the music path under the 'Suggestions' folder")

	return lastfmCommand
}

func installSong(args *LastfmSuggestArgs, link string, artist string, title string) error {
	ytdlArgs := []string{"--extract-audio", "--format", args.format, "--add-metadata", "--postprocessor-args", fmt.Sprintf("-metadata title=\"%s\" -metadata artist=\"%s\"", title, artist), "--output", path.Join(args.musicPath, "Suggestions", fmt.Sprintf("%s - %s.%%(ext)s", artist, title)), link}

	cmd := exec.Command("youtube-dl", ytdlArgs...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func suggestRunner(args *LastfmSuggestArgs, username string) error {
	if username == "" {
		credentials, err := setupOrGetCredentials()

		if err != nil {
			if credentials.Username != "" {
				fmt.Printf("Recieved error, but ignoring as I have a username: %s\n", err.Error())
			} else {
				return errors.New("Could not find a username in credentials file, and none was provided. Credentials error: " + err.Error())
			}
		}

		if credentials.Username == "" {
			return errors.New("Could not find a username in credentials file, and none was provided")
		}

		username = credentials.Username
	}

	resp, err := http.Get(fmt.Sprintf(SUGGEST_API_END_POINT, username))

	if err != nil {
		return fmt.Errorf("Could not get url - %s", err.Error())
	}

	if resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("Could not read response body - %s", err.Error())
	}

	var resultJson GetLastfmSuggestionsResponse
	err = json.Unmarshal(body, &resultJson)

	if err != nil {
		return fmt.Errorf("Could not read parse json - %s", err.Error())
	}

	if args.limit == -1 {
		args.limit = len(resultJson.Playlist)
	}

	if args.install && args.musicPath == "" {
		defaultMusicPath, err := utils.GetDefaultMusicPath()

		if err != nil {
			return err
		}

		args.musicPath = defaultMusicPath
	}

	var installWaitGroup sync.WaitGroup

	for i := 0; i < args.limit; i++ {
		item := resultJson.Playlist[i]
		albumArtist := ""

		for i, artist := range item.Artists {
			if artist.Name != "" {
				if i != 0 {
					albumArtist += ", " + artist.Name
				}

				albumArtist += artist.Name
			}
		}

		fmt.Printf("%s - %s", albumArtist, item.Name)
		playUrl := ""

		if args.printUrls || args.install {
			for _, playlink := range item.Playlinks {
				if playlink.Url != "" {
					playUrl = playlink.Url
					break
				}
			}
		}

		if args.printUrls && playUrl != "" {
			fmt.Printf(" (%s)", playUrl)
		}

		fmt.Println()

		if args.install && playUrl != "" {
			installWaitGroup.Add(1)

			go func() {
				defer installWaitGroup.Done()
				installSong(args, playUrl, albumArtist, item.Name)
			}()
		}
	}

	installWaitGroup.Wait()
	return nil
}
