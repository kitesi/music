package lastfm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	dbUtils "github.com/kitesi/music/db"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

const MAX_SCROBBLES = 50

func ImportSetup() *cobra.Command {
	args := LastfmImportArgs{}

	config, err := utils.GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}

	lastfmCommand := &cobra.Command{
		Use:   "import [log-db-file]",
		Short: "Import unfulfilled scrobbles from a database file",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, positional []string) {
			logDbFile := config.LastFm.LogDbFile
			if len(positional) == 1 {
				logDbFile = positional[0]
			}

			if logDbFile == "" {
				fmt.Fprintln(os.Stderr, "error: log db file not provided and not set in config")
				return
			}

			if err := importRunner(logDbFile, &args); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
	}

	lastfmCommand.Flags().BoolVarP(&args.debug, "debug", "d", config.Debug, "set debug mode")
	return lastfmCommand
}

func importRunner(filename string, _ *LastfmImportArgs) error {
	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	apiKey, _ := credentials.Get("api_key")
	apiSecret, _ := credentials.Get("api_secret")
	sessionKey, _ := credentials.Get("session_key")

	_, err = os.Stat(filename)

	if os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	db, err := dbUtils.OpenDB(filename)
	songsToScrobble, err := dbUtils.GetUnfulfilledPlays(db)

	if err != nil {
		return err
	}

	defer db.Close()

	cursor := 0

	if len(songsToScrobble) > MAX_SCROBBLES {
		fmt.Printf("There are more than %d songs to scrobble. This program will scrobble %d songs at a time.\n", MAX_SCROBBLES, MAX_SCROBBLES)
	}

	if len(songsToScrobble) == 0 {
		fmt.Println("No songs to scrobble.")
		return nil
	}

	for cursor < len(songsToScrobble) {
		endPosition := cursor + MAX_SCROBBLES

		if endPosition > len(songsToScrobble) {
			endPosition = len(songsToScrobble)
		}

		amountOfScrobbles := endPosition - cursor
		fmt.Printf("The following songs will be scrobbled (%d):\n", amountOfScrobbles)

		params := url.Values{}
		params.Set("method", "track.scrobble")
		params.Set("api_key", apiKey)
		params.Set("sk", sessionKey)

		for i, song := range songsToScrobble[cursor:endPosition] {
			timeStr := strconv.FormatInt(song.StartTime.Unix(), 10)
			fmt.Println(i+1, song.Album, song.Artist, song.Title, song.StartTime)
			params.Set(fmt.Sprintf("album[%d]", i), song.Album)
			params.Set(fmt.Sprintf("artist[%d]", i), song.Artist)
			params.Set(fmt.Sprintf("track[%d]", i), song.Title)
			params.Set(fmt.Sprintf("timestamp[%d]", i), timeStr)
		}

		// ask for confirmation
		fmt.Print("Do you want to continue? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		if strings.TrimSpace(text) != "y" {
			break
		}

		params.Set("api_sig", generateSignature(params, apiSecret))
		params.Set("format", "json")

		resp, err := http.PostForm(API_END_POINT, params)

		if err != nil {
			return err
		}

		if resp.StatusCode > 299 {
			return fmt.Errorf(resp.Status)
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		fmt.Println(resp.StatusCode, string(body))
		fmt.Println("Received response code:", resp.StatusCode)

		var resultJson PostMultipleScrobbleResponse
		err = json.Unmarshal(body, &resultJson)

		if err != nil {
			return err
		}

		fmt.Printf("Scrobbles accepted: %d, ignored: %d\n", resultJson.Scrobbles.Attr.Accepted, resultJson.Scrobbles.Attr.Ignored)

		if resultJson.Scrobbles.Attr.Accepted == amountOfScrobbles {
			fmt.Println("All scrobbles accepted, updating local database...")
			dbUtils.UpdateUnfulfilledPlays(db, songsToScrobble[cursor:endPosition])
		} else {
			fmt.Println("Not all scrobbles were accepted, not updating local database.")
		}

		cursor += amountOfScrobbles
	}

	return nil
}
