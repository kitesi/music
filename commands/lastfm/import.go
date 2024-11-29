package lastfm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

func ImportSetup() *cobra.Command {
	args := LastfmImportArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "import <file> [one of -j or -t]",
		Short: "Note: maybe not the clearest documentation since I don't suspect many will use this. Feel free to create a github issue if you need help! Import songs from a json file exported by the recent command, or by a text file that contains log output from the watch command.",
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
	lastfmCommand.Flags().BoolVarP(&args.json, "json", "j", false, "input file is in json format")
	lastfmCommand.Flags().BoolVarP(&args.text, "text", "t", false, "input file is in text format from watch --debug")
	return lastfmCommand
}

func importJsonFile(fileContents []byte, params *url.Values) (int, error) {
	var resultJson GetRecentTracksResponse
	err := json.Unmarshal(fileContents, &resultJson)

	if err != nil {
		return 0, err
	}

	for i, track := range resultJson.RecentTracks.Track {
		params.Set(fmt.Sprintf("artist[%d]", i), track.Artist.Text)
		params.Set(fmt.Sprintf("track[%d]", i), track.Name)
		params.Set(fmt.Sprintf("timestamp[%d]", i), track.Date.Uts)
	}

	return len(resultJson.RecentTracks.Track), nil
}

/*
In the format of:

info : 2024/11/24 15:45:00 └── scrobbling because it is over half way through (listened for 167.50, real: 130.00, half len: 80.50, min: 240)
info : 2024/11/24 15:45:30 new song detected - Christina Perri - human
info : 2024/11/24 15:45:40 └── not scrobbling because it did not pass either listen condition (listened for -23.80, real: 10.00, half len: 122.00, min: 240)
info : 2024/11/24 15:45:40 new song detected - Kendrick Lamar - Swimming Pools (Drank)
info : 2024/11/24 16:24:20 └── scrobbling because it is over half way through (listened for 246.79, real: 2320.00, half len: 124.00, min: 24j)
info : 2024/11/24 17:06:46 new song detected - AnnenMayKantereit & Giant Rooks - Tom's Diner
info : 2024/11/24 17:08:27 └── not scrobbling because it did not pass either listen condition (listened for 99.75, real: 101.00, half len: 136.50, min: 240)

Basically the idea is the program will extract all the songs that have been attempted to be scrolled to in this text file, and rescrobble it.
There are two reasons for doing this:

1. Maybe you don't have wifi, and thus the scrobble will fail.
2. Maybe you do have wifi but it just failed for some reason. (it's possible that lastfm will send an error which will be logged,
but it's also possible that lastfm will state everything went fine when in reality it didn't)
3. Maybe you were logged in to the wrong account.

I def want to introduce a better method for the first cause though. It should automatically retry scrobbling when it can.
*/

type PossibleRedoScrobble struct {
	Artist string
	Track  string
	Time   string
}

func importTextFile(fileContents []byte, params *url.Values) ([]PossibleRedoScrobble, error) {
	scanner := bufio.NewScanner(bytes.NewReader(fileContents))
	songsToScrobble := []PossibleRedoScrobble{}
	captureRegex := regexp.MustCompile(`info : (.+) (.+) new song detected - (.+) - (.+)`)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "new song detected") {
			if scanner.Scan() {
				nextLine := scanner.Text()

				if strings.Contains(nextLine, "─ scrobbling") {
					matches := captureRegex.FindStringSubmatch(line)

					if len(matches) == 5 {
						layout := "2006/01/02 15:04:05"
						parsedTime, err := time.Parse(layout, matches[1]+" "+matches[2])

						if err != nil {
							return nil, fmt.Errorf("error parsing time: %s from line %s\n", err, line)
						}

						songsToScrobble = append(songsToScrobble, PossibleRedoScrobble{
							Artist: matches[3],
							Track:  matches[4],
							Time:   fmt.Sprintf("%d", parsedTime.Unix()),
						})
					}

				}
			}

		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return songsToScrobble, nil
}

func importRunner(file string, args *LastfmImportArgs) error {
	if args.json && args.text {
		return fmt.Errorf("only one of -j or -t can be used")
	} else if !args.json && !args.text {
		return fmt.Errorf("one of -j or -t must be used")
	}

	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	// read file contents
	fileContents, err := os.ReadFile(file)

	if err != nil {
		return err
	}

	apiKey, _ := credentials.Get("api_key")
	apiSecret, _ := credentials.Get("api_secret")
	sessionKey, _ := credentials.Get("session_key")

	params := url.Values{}
	params.Set("method", "track.scrobble")
	params.Set("api_key", apiKey)
	params.Set("sk", sessionKey)

	amountOfScrobbles := 0

	if args.json {
		amountOfScrobbles, err = importJsonFile(fileContents, &params)

		if err != nil {
			return err
		}
	} else {
		todo, err := importTextFile(fileContents, &params)

		if err != nil {
			return err
		}

		// ask for confirmation
		fmt.Println("The following songs will be scrobbled:")

		for i, song := range todo {
			fmt.Println(i+1, song.Artist, song.Track, song.Time)
			params.Set(fmt.Sprintf("artist[%d]", i), song.Artist)
			params.Set(fmt.Sprintf("track[%d]", i), song.Track)
			params.Set(fmt.Sprintf("timestamp[%d]", i), song.Time)
		}

		// ask for confirmation
		fmt.Print("Do you want to continue? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		if strings.TrimSpace(text) != "y" {
			return nil
		}

		amountOfScrobbles = len(todo)
	}

	fmt.Println(apiSecret)

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

	fmt.Println(resp.StatusCode, string(body))

	fmt.Println("Scrobbled", amountOfScrobbles, "tracks (hopefully)")
	// TODO: take care of the response (does not match PostScrobbleResponse since Scrobble is an array i guess lol)

	return nil
}
