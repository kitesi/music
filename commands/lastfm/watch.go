package lastfm

import (
	"bufio"
	"crypto/md5"
	"syscall"

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
	"os/signal"
	"path"
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
	REAL_TIME_ERROR_MARGIN   = 10.0
)

func WatchSetup() *cobra.Command {
	args := LastfmWatchArgs{}

	lastfmCommand := &cobra.Command{
		Use:   "watch",
		Short: "Scrobble tracks to last.fm",
		Long:  "Watch for tracks playing in VLC and scrobble them to last.fm",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := watchRunner(&args); err != nil {
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

	if resp.StatusCode > 299 {
		return "", errors.New(resp.Status)
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

	if resp.StatusCode > 299 {
		return Session{}, errors.New(resp.Status)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return Session{}, err
	}

	var resultJson GetSessionResponse
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

	if resp.StatusCode > 299 {
		return PostScrobbleResponse{}, errors.New(resp.Status)
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

// playerctl is only on linux so we can just use xdg-open
func open(url string) error {
	return exec.Command("xdg-open", url).Run()
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
		case "username":
			credentials.Username = value
		default:
			return Credentials{}, errors.New("Unknown key in credentials file: " + parts[0])
		}
	}

	credentialsFile.Close()

	if err := scanner.Err(); err != nil {
		return credentials, errors.New("Error with reading credentials file: " + err.Error())
	}

	if credentials.ApiKey == "" {
		return credentials, errors.New("API key not found in credentials file")
	}

	if credentials.ApiSecret == "" {
		return credentials, errors.New("API secret not found in credentials file")
	}

	if credentials.SessionKey == "" {
		authToken, err := getAuthToken(credentials.ApiKey, credentials.ApiSecret)

		if err != nil {
			return credentials, errors.New("Error getting auth token: " + err.Error())
		}

		fmt.Println("Attempting to open up in browser...")
		err = open("http://www.last.fm/api/auth/?api_key=" + credentials.ApiKey + "&token=" + authToken)

		if err != nil {
			return credentials, errors.New("Error opening browser: " + err.Error())
		}

		fmt.Println("Press enter when you have accepted...")
		fmt.Scanln()

		session, err := getSession(credentials.ApiKey, credentials.ApiSecret, authToken)

		if err != nil {
			return credentials, errors.New("Error getting session key: " + err.Error())
		}

		credentials.SessionKey = session.Key
		credentials.Username = session.Name
		credentialsFile, err := os.OpenFile(credentialsPath, os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			return credentials, errors.New("Error with opening credentials file: " + err.Error())
		}

		_, err = credentialsFile.WriteString("\nsession_key=" + credentials.SessionKey + "\nusername=" + credentials.Username)

		if err != nil {
			return credentials, errors.New("Error with writing to credentials file: " + err.Error())
		}
	}

	return credentials, nil
}

func getCurrentPosition() (float64, error) {
	positionCmd := exec.Command("playerctl", "-p", "vlc", "position")
	positionOutput, err := positionCmd.Output()

	if err != nil || string(positionOutput) == "No player could handle this command" {
		return 0.0, errors.New("playerctl - no player could handle this command")
	}

	position, err := strconv.ParseFloat(strings.TrimSpace(string(positionOutput)), 64)

	if err != nil {
		return 0.0, errors.New("playerctl - could not parse position")
	}

	return position, nil
}

func attemptScrobble(credentials Credentials, currentTrack *CurrentTrackInfo, currentPosition float64, delay int, stdOut *log.Logger, stdErr *log.Logger) {
	paddedLastPosition := currentTrack.LastPosition + float64(delay) - currentPosition
	timeConditionPassed := -1.0

	if paddedLastPosition > currentTrack.Length/2.0 {
		timeConditionPassed = currentTrack.Length / 2.0
	} else if paddedLastPosition > MIN_LISTEN_TIME {
		timeConditionPassed = MIN_LISTEN_TIME
	}

	realTimePassed := float64(time.Now().Unix()-currentTrack.StartTime) / time.Second.Seconds()
	listenStats := fmt.Sprintf("listened for %.2f, real: %.2f, half len: %.2f, min: %d", paddedLastPosition, realTimePassed, currentTrack.Length/2.0, MIN_LISTEN_TIME)

	if timeConditionPassed == -1.0 {
		stdOut.Printf("└── not scrobbling because it did not pass either listen condition (%s)", listenStats)
	} else if realTimePassed > timeConditionPassed-REAL_TIME_ERROR_MARGIN {
		reason := ""

		if paddedLastPosition > currentTrack.Length/2.0 {
			reason = "it is over half way through"
		} else {
			reason = "it has been listened to for over the minimum listen time"
		}

		stdOut.Printf("└── scrobbling because %s (%s)", reason, listenStats)

		scrobbleResponse, err := scrobble(credentials, currentTrack.Artist, currentTrack.Track, currentTrack.StartTime)

		if err != nil {
			stdErr.Printf("└── last.fm api error - %s", err.Error())
		}

		if scrobbleResponse.Scrobbles.Attr.Ignored == 1 {
			stdErr.Printf("└── last.fm ignored this scrobble - %s", scrobbleResponse.Scrobbles.Scrobble.IgnoredMessage.Text)
		}

	} else {
		stdOut.Printf("└── not scrobbling because while it did pass the time condition, the real time did not pass (%s)", listenStats)
	}
}

func watchForTracks(credentials Credentials, currentTrack *CurrentTrackInfo, delay int, stdOut *log.Logger, stdErr *log.Logger) {
	waitTime := time.Duration(delay) * time.Second

	for {
		position, err := getCurrentPosition()

		if err != nil {
			if currentTrack.Track != "" {
				attemptScrobble(credentials, currentTrack, 0, delay, stdOut, stdErr)
				currentTrack.Track = ""
				currentTrack.Artist = ""
			}

			stdErr.Println(err)
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
			stdErr.Println("playerctl - could get metadata but not the necessary fields")
			time.Sleep(waitTime)
			continue
		}

		artist := metadata["xesam:artist"]
		track := metadata["xesam:title"]

		if ((artist != currentTrack.Artist || track != currentTrack.Track) || position < currentTrack.LastPosition) && currentTrack.Track != "" && currentTrack.Length != -1.0 {
			attemptScrobble(credentials, currentTrack, position, delay, stdOut, stdErr)
		}

		if artist != currentTrack.Artist || track != currentTrack.Track {
			currentTrack.Track = track
			currentTrack.Artist = artist
			currentTrack.LastPosition = position
			currentTrack.StartTime = time.Now().Unix()

			stdOut.Printf("new song detected - %s - %s", artist, track)
			length, err := strconv.ParseFloat(metadata["vlc:time"], 64)

			if err != nil {
				stdErr.Printf("└── playerctl - could not parse length of")
				currentTrack.Length = -1.0
			} else if length < MIN_TRACK_LEN {
				stdOut.Printf("└── skipping track because it is too short")
				currentTrack.Length = -1.0
			} else {
				currentTrack.Length = length
			}
		} else {
			if position < currentTrack.LastPosition {
				stdErr.Printf("└── playerctl - position went backwards")
				currentTrack.StartTime = time.Now().Unix()
			}

			currentTrack.LastPosition = position
		}

		time.Sleep(waitTime)
	}
}

type CurrentTrackInfo struct {
	Track        string
	Artist       string
	LastPosition float64
	StartTime    int64
	Length       float64
}

func watchRunner(args *LastfmWatchArgs) error {
	_, err := exec.LookPath("playerctl")

	if err != nil {
		return errors.New("playerctl is not installed, this program only works on linux")
	}

	gracefulExit := make(chan os.Signal, 1)
	signal.Notify(gracefulExit, syscall.SIGINT, syscall.SIGTERM)

	lockFileName := path.Join(os.TempDir(), "music-lastfm.lock")
	_, err = os.Stat(lockFileName)

	if err == nil {
		return fmt.Errorf("server is already running - if it is not, delete %s and try again", lockFileName)
	}

	credentials, err := setupOrGetCredentials()

	if err != nil {
		return err
	}

	lockFile, err := os.Create(lockFileName)

	if err != nil {
		return errors.New("could not create lock file")
	}

	currentTrack := CurrentTrackInfo{}
	stdOutLog := log.New(os.Stdout, "info : ", log.LstdFlags)
	stdErrLog := log.New(os.Stderr, "error: ", log.LstdFlags)

	if !args.debug {
		stdErrLog.SetOutput(io.Discard)
	}

	go func() {
		<-gracefulExit
		lockFile.Close()
		os.Remove(lockFileName)

		if currentTrack.Track != "" {
			position, err := getCurrentPosition()

			if err != nil {
				stdErrLog.Println("could not get position when gracefully exiting")
			} else {
				attemptScrobble(credentials, &currentTrack, position, args.interval, stdOutLog, stdErrLog)
			}
		} else {
			stdOutLog.Println("did not find any track when gracefully exiting")
		}

		os.Exit(0)
	}()

	watchForTracks(credentials, &currentTrack, args.interval, stdOutLog, stdErrLog)
	return nil
}
