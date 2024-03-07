package lastfm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

func RecentSetup() *cobra.Command {
	args := LastfmRecentArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "recent",
		Short: "Get recent songs",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, positional []string) {
			if err := recentRunner(&args); err != nil {
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

	lastfmCommand.Flags().StringVarP(&args.username, "username", "u", "", "Lastfm username to look up")
	lastfmCommand.Flags().IntVarP(&args.limit, "limit", "l", 50, "Number of songs to get")
	lastfmCommand.Flags().BoolVarP(&args.debug, "debug", "d", config.Debug, "set debug mode")
	lastfmCommand.Flags().BoolVarP(&args.json, "json", "j", config.Debug, "set debug mode")
	return lastfmCommand
}

func recentRunner(args *LastfmRecentArgs) error {
	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("method", "user.getRecentTracks")
	params.Set("user", credentials.Username)
	params.Set("api_key", credentials.ApiKey)
	params.Set("limit", fmt.Sprint(args.limit))
	params.Set("api_sig", generateSignature(params, credentials.ApiSecret))
	params.Set("format", "json")

	resp, err := http.PostForm(API_END_POINT, params)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if args.json {
		fmt.Println(string(body))
	} else {
		var resultJson GetRecentTracksResponse
		err = json.Unmarshal(body, &resultJson)

		if err != nil {
			return err
		}

		for _, track := range resultJson.RecentTracks.Track {
			fmt.Println(track.Artist.Text + " - " + track.Name + " - " + track.Date.Text)
		}
	}

	return nil
}
