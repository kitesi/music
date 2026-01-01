package lastfm

import (
	"crypto/md5"
	"database/sql"
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

	// import Config from here as SimpleConfig

	dbUtils "github.com/kitesi/music/db"
	"github.com/kitesi/music/simpleconfig"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

const (
	API_END_POINT          = "http://ws.audioscrobbler.com/2.0/"
	REAL_TIME_ERROR_MARGIN = 10.0
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

	config, err := utils.GetConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}

	lastfmCommand.Flags().IntVarP(&args.interval, "interval", "i", config.LastFm.Interval, "interval in seconds to check for new tracks")
	lastfmCommand.Flags().StringVar(&args.logDbFile, "log-db-file", config.LastFm.LogDbFile, "sqlite database file to log scrobbles to")
	lastfmCommand.Flags().IntVar(&args.minTrackLength, "min-track-length", config.LastFm.MinTrackLength, "the minimum track length to scrobble")
	lastfmCommand.Flags().IntVar(&args.minListenTime, "min-listen-length", config.LastFm.MinListenTime, "the minimum listem time to scrobble (as a shorter alternative to half way through the track)")
	lastfmCommand.Flags().BoolVar(&args.debug, "debug", config.Debug, "set debug mode")

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

func scrobble(credentials simpleconfig.Config, artist string, track string, timestamp int64) (PostScrobbleResponse, error) {
	apiKey, _ := credentials.Get("api_key")
	apiSecret, _ := credentials.Get("api_secret")
	sessionKey, _ := credentials.Get("session_key")

	params := url.Values{}
	params.Set("method", "track.scrobble")
	params.Set("api_key", apiKey)
	params.Set("artist", artist)
	params.Set("track", track)
	params.Set("timestamp", fmt.Sprint(timestamp))
	params.Set("sk", sessionKey)
	params.Set("api_sig", generateSignature(params, apiSecret))
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

func setupOrGetCredentials() (simpleconfig.Config, error) {
	cacheDir, err := os.UserCacheDir()

	if err != nil {
		return simpleconfig.Config{}, errors.New("could not get cache directory")
	}

	credentialsPath := path.Join(cacheDir, utils.LASTFM_CREDENTIALS_FILE)
	credentials, err := simpleconfig.NewConfig(credentialsPath, []string{"api_key", "api_secret", "session_key", "username"})

	if err != nil {
		return credentials, err
	}

	apiKey, _ := credentials.Get("api_key")
	apiSecret, _ := credentials.Get("api_secret")
	sessionKey, _ := credentials.Get("session_key")

	if apiKey == "" {
		return credentials, errors.New("API key not found in credentials file")
	}

	if apiSecret == "" {
		return credentials, errors.New("API secret not found in credentials file")
	}

	if sessionKey == "" {
		authToken, err := getAuthToken(apiKey, apiSecret)

		if err != nil {
			return credentials, errors.New("Error getting auth token: " + err.Error())
		}

		fmt.Println("Attempting to open up in browser...")
		err = open("http://www.last.fm/api/auth/?api_key=" + apiKey + "&token=" + authToken)

		if err != nil {
			return credentials, errors.New("Error opening browser: " + err.Error())
		}

		fmt.Println("Press enter when you have accepted...")
		fmt.Scanln()

		session, err := getSession(apiKey, apiSecret, authToken)

		if err != nil {
			return credentials, errors.New("Error getting session key: " + err.Error())
		}

		credentials.Set("session_key", session.Key)
		credentials.Set("username", session.Name)

		err = credentials.WriteConfig()

		if err != nil {
			return credentials, errors.New("Error writing credentials file: " + err.Error())
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

func attemptScrobble(db *sql.DB, credentials simpleconfig.Config, currentTrack *CurrentTrackInfo, args *LastfmWatchArgs, currentPosition float64, stdOut *log.Logger, stdErr *log.Logger) {
	paddedLastPosition := currentTrack.LastPosition + float64(args.interval) - currentPosition
	timeConditionPassed := -1.0

	if paddedLastPosition > currentTrack.Length/2.0 {
		timeConditionPassed = currentTrack.Length / 2.0
	} else if paddedLastPosition > float64(args.minListenTime) {
		timeConditionPassed = float64(args.minListenTime)
	}

	realTimePassed := time.Since(currentTrack.StartTime).Seconds()
	listenStats := fmt.Sprintf("listened for %.2f, real: %.2f, half len: %.2f, min: %d", paddedLastPosition, realTimePassed, currentTrack.Length/2.0, args.minListenTime)

	if timeConditionPassed == -1.0 {
		stdOut.Printf("└── not scrobbling because it did not pass either listen condition (%s)", listenStats)
	} else if realTimePassed > timeConditionPassed-REAL_TIME_ERROR_MARGIN {
		reason := ""

		if paddedLastPosition > currentTrack.Length/2.0 {
			reason = "it is over half way through"
		} else {
			reason = "it has been listened to for over the minimum listen time"
		}

		insertParams := dbUtils.InsertIntoPlaysParams{
			Fulfilled: true,
			Title:     currentTrack.Track,
			Artist:    currentTrack.Artist,
			Time:      currentTrack.StartTime,
		}

		stdOut.Printf("└── scrobbling because %s (%s)", reason, listenStats)
		scrobbleResponse, err := scrobble(credentials, currentTrack.Artist, currentTrack.Track, currentTrack.StartTime.Unix())

		if err != nil {
			insertParams.Fulfilled = false
			stdErr.Printf("└── last.fm api error - %s", err.Error())
		}

		if scrobbleResponse.Scrobbles.Attr.Ignored == 1 {
			stdErr.Printf("└── last.fm ignored this scrobble - %s", scrobbleResponse.Scrobbles.Scrobble.IgnoredMessage.Text)
		}

		if db != nil {
			dbUtils.InsertIntoPlays(db, insertParams)
		}
	} else {
		stdOut.Printf("└── not scrobbling because while it did pass the time condition, the real time did not pass (%s)", listenStats)
	}
}

func watchForTracks(db *sql.DB, credentials simpleconfig.Config, currentTrack *CurrentTrackInfo, args *LastfmWatchArgs, stdOut *log.Logger, stdErr *log.Logger) {
	waitTime := time.Duration(args.interval) * time.Second

	for {
		position, err := getCurrentPosition()

		if err != nil {
			if currentTrack.Track != "" {
				attemptScrobble(db, credentials, currentTrack, args, 0.0, stdOut, stdErr)
				currentTrack.Track = ""
				currentTrack.Artist = ""
			}

			stdErr.Println(err)
			time.Sleep(waitTime)
			continue
		}

		songMetadata, err := utils.GetCurrentPlayingSong()

		if err != nil {
			stdErr.Println(err)
			time.Sleep(waitTime)
			continue
		}

		artist := songMetadata.Artist
		track := songMetadata.Track

		if ((artist != currentTrack.Artist || track != currentTrack.Track) || position < currentTrack.LastPosition) && currentTrack.Track != "" && currentTrack.Length != -1.0 {
			attemptScrobble(db, credentials, currentTrack, args, position, stdOut, stdErr)
		}

		if artist != currentTrack.Artist || track != currentTrack.Track {
			currentTrack.Track = track
			currentTrack.Artist = artist
			currentTrack.LastPosition = position
			currentTrack.StartTime = time.Now()

			stdOut.Printf("new song detected - %s - %s", artist, track)
			length, err := strconv.ParseFloat(songMetadata.Length, 64)

			if err != nil {
				stdErr.Printf("└── playerctl - could not parse length of")
				currentTrack.Length = -1.0
			} else if length < float64(args.minTrackLength) {
				stdOut.Printf("└── skipping track because it is too short")
				currentTrack.Length = -1.0
			} else {
				currentTrack.Length = length
			}
		} else {
			if position < currentTrack.LastPosition {
				stdErr.Printf("└── playerctl - position went backwards")
				currentTrack.StartTime = time.Now()
			}

			currentTrack.LastPosition = position
		}

		time.Sleep(waitTime)
	}
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

	var db *sql.DB

	if args.logDbFile != "" {
		db, err = dbUtils.OpenDB(args.logDbFile)
		if err != nil {
			return fmt.Errorf("could not load log db file")
		}
		defer db.Close()
		dbUtils.RunMigrations(db)
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
				attemptScrobble(db, credentials, &currentTrack, args, position, stdOutLog, stdErrLog)
			}
		} else {
			stdOutLog.Println("did not find any track when gracefully exiting")
		}

		os.Exit(0)
	}()

	watchForTracks(db, credentials, &currentTrack, args, stdOutLog, stdErrLog)
	return nil
}
