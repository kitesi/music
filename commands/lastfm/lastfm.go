package lastfm

import (
	"bufio"
	"crypto/md5"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	API_END_POINT            = "http://ws.audioscrobbler.com/2.0/"
	MIN_TRACK_LEN            = 30
	MIN_LISTEN_TIME          = 4 * 60
	DEFAULT_INTERVAL_SECONDS = 10
)

func Setup() *cobra.Command {
	args := LastfmArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "lastfm",
		Short: "Scrobble tracks to last.fm",
		Long:  "Watch for tracks playing in VLC and scrobble them to last.fm",
		Run: func(cmd *cobra.Command, positional []string) {
			if err := lastfmRunner(&args); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
	}

	lastfmCommand.Flags().IntVarP(&args.interval, "interval", "i", DEFAULT_INTERVAL_SECONDS, "interval in seconds to check for new tracks")
	lastfmCommand.Flags().BoolVar(&args.debug, "debug", false, "set debug mode")

	return lastfmCommand
}

func generateSignature(params url.Values, apiSecret string) string {
	signature := ""

	keys := make([]string, 0, len(params))

	for key := range params {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		signature += key + params.Get(key)
	}

	signature += apiSecret

	hasher := md5.New()
	hasher.Write([]byte(signature))
	signature = hex.EncodeToString(hasher.Sum(nil))

	return signature
}

func getAuthToken(apiKey string, apiSecret string) (string, error) {
	params := url.Values{}
	params.Set("method", "auth.gettoken")
	params.Set("format", "json")
	params.Set("api_key", apiKey)
	params.Set("api_sig", generateSignature(params, apiSecret))

	resp, err := http.PostForm(API_END_POINT, params)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	var resultJson GetAuthTokenResponse
	err = json.Unmarshal(body, &resultJson)

	if err != nil {
		return "", err
	}

	if resultJson.Error != 0 || resultJson.Message != "" {
		return "", fmt.Errorf("(%d) %s", resultJson.Error, resultJson.Message)
	}

	return resultJson.Token, nil
}

func getSession(apiKey string, apiSecret string, token string) (Session, error) {
	params := url.Values{}
	params.Set("method", "auth.getSession")
	params.Set("api_key", apiKey)
	params.Set("token", token)
	params.Set("api_sig", generateSignature(params, apiSecret))
	// have to do this after generating the signature
	params.Set("format", "json")

	resp, err := http.PostForm(API_END_POINT, params)

	if err != nil {
		return Session{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return Session{}, err
	}

	var resultJson GetSessionKeyResponse
	err = json.Unmarshal(body, &resultJson)

	if err != nil {
		return Session{}, err
	}

	if resultJson.Error != 0 || resultJson.Message != "" {
		return Session{}, fmt.Errorf("(%d) %s", resultJson.Error, resultJson.Message)
	}

	return resultJson.Session, nil
}

func scrobble(credentials Credentials, artist string, track string, timestamp int64) (PostScrobbleResponse, error) {
	params := url.Values{}
	params.Set("method", "track.scrobble")
	params.Set("api_key", credentials.ApiKey)
	params.Set("artist", artist)
	params.Set("track", track)
	params.Set("timestamp", fmt.Sprint(timestamp))
	params.Set("sk", credentials.SessionKey)
	params.Set("api_sig", generateSignature(params, credentials.ApiSecret))
	params.Set("format", "json")

	resp, err := http.PostForm(API_END_POINT, params)

	if err != nil {
		return PostScrobbleResponse{}, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return PostScrobbleResponse{}, err
	}

	resultJson := PostScrobbleResponse{}
	err = json.Unmarshal(body, &resultJson)

	if err != nil {
		return PostScrobbleResponse{}, err
	}

	return resultJson, nil
}

// open opens the specified URL in the default browser of the user.
// https://stackoverflow.com/a/39324149/
func open(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Run()
}

func setupOrGetCredentials() (Credentials, error) {
	cacheDir, err := os.UserCacheDir()

	if err != nil {
		return Credentials{}, errors.New("Could not get cache directory: " + err.Error())
	}

	credentialsPath := path.Join(cacheDir, ".lastfm-credentials")
	_, err = os.Stat(credentialsPath)

	var credentialsFile *os.File

	if os.IsNotExist(err) {
		credentialsFile, err = os.Create(credentialsPath)

		if err != nil {
			return Credentials{}, errors.New("Error with creating credentials file: " + err.Error())
		}
	} else if err != nil {
		return Credentials{}, errors.New("Error with stating credentials file: " + err.Error())
	}

	if credentialsFile == nil {
		credentialsFile, err = os.Open(credentialsPath)

		if err != nil {
			return Credentials{}, errors.New("Error with opening credentials file: " + err.Error())
		}
	}

	var credentials Credentials

	scanner := bufio.NewScanner(credentialsFile)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "api_key":
			credentials.ApiKey = value
		case "api_secret":
			credentials.ApiSecret = value
		case "session_key":
			credentials.SessionKey = value
		default:
			return Credentials{}, errors.New("Unknown key in credentials file: " + parts[0])
		}
	}

	credentialsFile.Close()

	if err := scanner.Err(); err != nil {
		return Credentials{}, errors.New("Error with reading credentials file: " + err.Error())
	}

	if credentials.ApiKey == "" {
		return Credentials{}, errors.New("API key not found in credentials file")
	}

	if credentials.ApiSecret == "" {
		return Credentials{}, errors.New("API secret not found in credentials file")
	}

	if credentials.SessionKey == "" {
		authToken, err := getAuthToken(credentials.ApiKey, credentials.ApiSecret)

		if err != nil {
			return Credentials{}, errors.New("Error getting auth token: " + err.Error())
		}

		err = open("http://www.last.fm/api/auth/?api_key=" + credentials.ApiKey + "&token=" + authToken)

		if err != nil {
			return Credentials{}, errors.New("Error opening browser: " + err.Error())
		}

		fmt.Println("Press enter when you have accepted...")
		fmt.Scanln()

		session, err := getSession(credentials.ApiKey, credentials.ApiSecret, authToken)

		if err != nil {
			return Credentials{}, errors.New("Error getting session key: " + err.Error())
		}

		credentials.SessionKey = session.Key
		credentialsFile, err := os.OpenFile(credentialsPath, os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			return Credentials{}, errors.New("Error with opening credentials file: " + err.Error())
		}

		_, err = credentialsFile.WriteString("\nsession_key=" + credentials.SessionKey)

		if err != nil {
			return Credentials{}, errors.New("Error with writing to credentials file: " + err.Error())
		}
	}

	return credentials, nil
}

func watchForTracks(credentials Credentials, delay int, stdOut *log.Logger, stdErr *log.Logger) {
	currentTrack := ""
	currentArtist := ""
	currentTrackLastPosition := 0.0
	waitTime := time.Duration(delay) * time.Second
	var currentTrackFirstTimestamp int64

	for {
		positionCmd := exec.Command("playerctl", "-p", "vlc", "position")
		positionOutput, err := positionCmd.Output()

		if err != nil || string(positionOutput) == "No player could handle this command" {
			stdErr.Println("playerctl - no player could handle this command")
			time.Sleep(waitTime)
			continue
		}

		position, err := strconv.ParseFloat(strings.TrimSpace(string(positionOutput)), 64)

		if err != nil {
			stdErr.Println("playerctl - could not parse position")
			time.Sleep(waitTime)
			continue
		}

		metadataCmd := exec.Command("playerctl", "-p", "vlc", "metadata")
		metadataOutput, err := metadataCmd.Output()

		if err != nil {
			stdErr.Println("playerctl - could not get metadata")
			time.Sleep(waitTime)
			continue
		}

		metadata := make(map[string]string)

		for _, line := range strings.Split(string(metadataOutput), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}

			// split by whitespace
			sections := strings.Fields(line)

			if len(sections) < 3 {
				continue
			}

			_, key, value := sections[0], sections[1], strings.Join(sections[2:], " ")
			metadata[key] = value
		}

		if metadata["xesam:artist"] == "" || metadata["xesam:title"] == "" || metadata["vlc:length"] == "" {
			stdErr.Println("playerctl - could not get metadata")
			time.Sleep(waitTime)
			continue
		}

		length, err := strconv.ParseFloat(metadata["vlc:time"], 64)

		if err != nil {
			stdErr.Println("playerctl - could not parse length")
			time.Sleep(waitTime)
			continue
		}

		artist := metadata["xesam:artist"]
		track := metadata["xesam:title"]

		if length < MIN_TRACK_LEN {
			stdOut.Printf("skipping track (%s - %s) because it is too short\n", artist, track)
			time.Sleep(waitTime)
			continue
		}

		if artist != currentArtist || track != currentTrack {
			if currentTrack != "" {
				paddedLastPosition := currentTrackLastPosition + DEFAULT_INTERVAL_SECONDS - position
				errorSecondsMargin := 10.0

				timeConditionPassed := -1.0

				if paddedLastPosition > length/2.0 {
					timeConditionPassed = length / 2.0
				} else if paddedLastPosition > MIN_LISTEN_TIME {
					timeConditionPassed = MIN_LISTEN_TIME
				}

				if timeConditionPassed == -1.0 {
					stdOut.Printf("not scrobbling %s - %s because it did not pass either listen condition", currentArtist, currentTrack)
				} else if float64(time.Now().Unix()-currentTrackFirstTimestamp)/time.Second.Seconds() > timeConditionPassed-errorSecondsMargin {
					reason := ""

					if paddedLastPosition > length/2.0 {
						reason = "it is over half way through"
					} else {
						reason = "it has been listened to for over the minimum listen time"
					}

					stdOut.Printf("scrobbling %s - %s because %s", currentArtist, currentTrack, reason)

					scrobbleResponse, err := scrobble(credentials, currentArtist, currentTrack, currentTrackFirstTimestamp)

					if err != nil {
						stdErr.Printf("last.fm api error - %s\n", err.Error())
					}

					if scrobbleResponse.Scrobbles.Attr.Ignored == 1 {
						stdErr.Printf("last.fm ignored this scrobble - %s\n", scrobbleResponse.Scrobbles.Scrobble.IgnoredMessage.Text)
					}

				} else {
					stdOut.Printf("not scrobbling %s - %s because while it did pass the time condition, the real time did not pass\n", currentArtist, currentTrack)
				}
			}

			currentTrack = track
			currentArtist = artist
			currentTrackLastPosition = position
			currentTrackFirstTimestamp = time.Now().Unix()

			stdOut.Printf("new song detected - %s - %s\n", artist, track)
		} else {
			currentTrackLastPosition = position
		}

		time.Sleep(waitTime)
	}
}

func lastfmRunner(args *LastfmArgs) error {
	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	_, err = exec.LookPath("playerctl")

	if err != nil {
		return errors.New("playerctl is not installed")
	}

	stdOutLog := log.New(os.Stdout, "info: ", log.LstdFlags)
	stdErrLog := log.New(os.Stderr, "error: ", log.LstdFlags)

	if !args.debug {
		stdErrLog.SetOutput(io.Discard)
	}

	watchForTracks(credentials, args.interval, stdOutLog, stdErrLog)
	return nil
}
