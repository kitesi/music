package lastfm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

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

	return lastfmCommand
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

		if args.printUrls {
			playUrl := ""

			for _, playlink := range item.Playlinks {
				if playlink.Url != "" {
					playUrl = playlink.Url
					break
				}
			}

			if playUrl != "" {
				fmt.Printf(" (%s)", playUrl)
			}
		}

		fmt.Println()
	}

	return nil
}