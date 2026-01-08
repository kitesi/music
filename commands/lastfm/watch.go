package lastfm

import (
	"crypto/md5"
	"database/sql"
	"math"
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
	API_END_POINT = "http://ws.audioscrobbler.com/2.0/"
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
	lastfmCommand.Flags().StringVar(&args.source, "source", config.LastFm.Source, "source to log scrobbles as (e.g. pc, web, vlc, etc.)")
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

func scrobble(credentials simpleconfig.Config, album string, artist string, track string, timestamp int64) (PostScrobbleResponse, error) {
	apiKey, _ := credentials.Get("api_key")
	apiSecret, _ := credentials.Get("api_secret")
	sessionKey, _ := credentials.Get("session_key")

	params := url.Values{}
	params.Set("method", "track.scrobble")
	params.Set("api_key", apiKey)
	params.Set("album", album)
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

const SEEK_TOLERANCE_SECONDS = 8
const SESSION_RESET_THRESHOLD_SECONDS = 90
const DRIFT_TOLERANCE_SECONDS = 1.5

func attemptScrobble(db *sql.DB, credentials simpleconfig.Config, currentTrack *CurrentTrackInfo, args *LastfmWatchArgs, currentPosition float64, stdOut *log.Logger, stdErr *log.Logger) {
	passingReason := ""
	uniqueCoverage, maxPosition := currentTrack.UniqueCoverageAndMaxPosition()

	if uniqueCoverage > currentTrack.Duration/2.0 {
		passingReason = "it covered over half the track uniquely"
	} else if currentTrack.ListenTime > float64(args.minListenTime) {
		passingReason = "it listened for over the minimum listen time"
	}

	realTimePassed := time.Since(currentTrack.StartTime).Seconds()
	currentTrack.WallTime = realTimePassed
	listenStats := fmt.Sprintf("unique coverage for %.2f, listen time: %.2f, wall time: %.2f, half len: %.2f, min: %d", uniqueCoverage, currentTrack.ListenTime, currentTrack.WallTime, currentTrack.Duration/2.0, args.minListenTime)

	insertParams := dbUtils.InsertIntoPlaysParams{
		Fulfilled:      false,
		Scrobbable:     false,
		Title:          currentTrack.Track,
		Artist:         currentTrack.Artist,
		Album:          currentTrack.Album,
		ListenTime:     int(currentTrack.ListenTime),
		WallTime:       int(currentTrack.WallTime),
		SeekCount:      currentTrack.SeekCount,
		MaxPosition:    int(maxPosition),
		UniqueCoverage: int(uniqueCoverage),
		Duration:       int(currentTrack.Duration),
		StartTime:      currentTrack.StartTime,
		Source:         args.source,
	}

	if passingReason == "" {
		stdOut.Printf("└── not scrobbling because it did not pass either listen condition (%s)", listenStats)
	} else {
		insertParams.Scrobbable = true

		stdOut.Printf("└── scrobbling because %s (%s)", passingReason, listenStats)
		scrobbleResponse, err := scrobble(credentials, currentTrack.Album, currentTrack.Artist, currentTrack.Track, currentTrack.StartTime.Unix())

		if err != nil {
			stdErr.Printf("└── last.fm api error - %s", err.Error())
		} else {
			insertParams.Fulfilled = true
			if scrobbleResponse.Scrobbles.Attr.Ignored == 1 {
				stdErr.Printf("└── last.fm ignored this scrobble - %s", scrobbleResponse.Scrobbles.Scrobble.IgnoredMessage.Text)
			}
		}
	}

	if db != nil {
		if err := dbUtils.InsertIntoPlays(db, insertParams); err != nil {
			stdErr.Printf("└── could not log scrobble to db - %s", err.Error())
		}
	}
}

func watchForTracks(db *sql.DB, credentials simpleconfig.Config, currentTrack *CurrentTrackInfo, args *LastfmWatchArgs, stdOut *log.Logger, stdErr *log.Logger) {
	waitTime := time.Duration(args.interval) * time.Second

	for {
		position, err := getCurrentPosition()

		// if we can't get the position, attempt to scrobble the current track and reset
		if err != nil {
			if currentTrack.Track != "" {
				attemptScrobble(db, credentials, currentTrack, args, 0.0, stdOut, stdErr)
				currentTrack.Track = ""
				currentTrack.Artist = ""
				currentTrack.Album = ""
				currentTrack.ResetMetrics()
			}

			stdErr.Println(err)
			time.Sleep(waitTime)
			continue
		}

		songMetadata, err := utils.GetCurrentPlayingSong()
		now := time.Now()

		if err != nil {
			stdErr.Println(err)
			time.Sleep(waitTime)
			continue
		}

		album := songMetadata.Album
		artist := songMetadata.Artist
		track := songMetadata.Track

		// we've found a new song
		if artist != currentTrack.Artist || track != currentTrack.Track || album != currentTrack.Album {
			if currentTrack.Track != "" && currentTrack.Duration != -1.0 {
				attemptScrobble(db, credentials, currentTrack, args, position, stdOut, stdErr)
			}

			currentTrack.Track = track
			currentTrack.Artist = artist
			currentTrack.Album = album

			currentTrack.ResetMetrics()
			currentTrack.StartTime = now

			stdOut.Printf("new song detected - %s - %s", artist, track)
			length, err := strconv.ParseFloat(songMetadata.Length, 64)

			if err != nil {
				stdErr.Printf("└── playerctl - could not parse length of")
				currentTrack.Duration = -1.0
			} else if length < float64(args.minTrackLength) {
				stdOut.Printf("└── skipping track because it is too short")
				currentTrack.Duration = -1.0
			} else {
				currentTrack.Duration = length
			}
		} else {
			deltaPos := position - currentTrack.LastPosition
			absDeltaPos := math.Abs(deltaPos)
			deltaTime := time.Since(currentTrack.LastUpdate).Seconds()
			expectedPos := currentTrack.LastPosition + deltaTime

			if deltaPos > 0 && math.Abs(position-expectedPos) < DRIFT_TOLERANCE_SECONDS { // natural playback
				currentTrack.ListenTime += deltaTime
				currentTrack.AddListenRange(currentTrack.LastPosition, position)
			} else if absDeltaPos > SESSION_RESET_THRESHOLD_SECONDS { // too much of a jump, reset session
				if currentTrack.Duration != -1.0 {
					attemptScrobble(db, credentials, currentTrack, args, position, stdOut, stdErr)
				}
				currentTrack.ResetMetrics()
				currentTrack.StartTime = now
			} else if absDeltaPos > SEEK_TOLERANCE_SECONDS { // medium seek, intentional repositioning
				currentTrack.SeekCount++
				stdOut.Printf("└── playerctl - position seeked")
			}
		}

		currentTrack.LastPosition = position
		currentTrack.LastUpdate = now
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
		err = dbUtils.RunMigrations(db)

		if err != nil {
			return fmt.Errorf("could not run migrations on log db file: %s", err.Error())
		}
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
