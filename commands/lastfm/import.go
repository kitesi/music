package lastfm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

func ImportSetup() *cobra.Command {
	args := LastfmImportArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "import <file>",
		Short: "Import songs from a json file exported by the recent command",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := importRunner(positional[0], &args); err != nil {
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

	lastfmCommand.Flags().BoolVarP(&args.debug, "debug", "d", config.Debug, "set debug mode")
	return lastfmCommand
}

func importRunner(file string, args *LastfmImportArgs) error {
	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	// read file contents
	fileContents, err := os.ReadFile(file)

	if err != nil {
		return err
	}

	var resultJson GetRecentTracksResponse
	err = json.Unmarshal(fileContents, &resultJson)

	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("method", "track.scrobble")
	params.Set("api_key", credentials.ApiKey)
	params.Set("sk", credentials.SessionKey)

	for i, track := range resultJson.RecentTracks.Track {
		params.Set(fmt.Sprintf("artist[%d]", i), track.Artist.Text)
		params.Set(fmt.Sprintf("track[%d]", i), track.Name)
		params.Set(fmt.Sprintf("timestamp[%d]", i), track.Date.Uts)
	}

	params.Set("api_sig", generateSignature(params, credentials.ApiSecret))
	params.Set("format", "json")

	resp, err := http.PostForm(API_END_POINT, params)

	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf(resp.Status)
	}

	fmt.Println("Scrobbled", len(resultJson.RecentTracks.Track), "tracks (hopefully)")
	// TODO: take care of the response (does not match PostScrobbleResponse since Scrobble is an array i guess lol)

	return nil
}
