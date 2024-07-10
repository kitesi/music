package spotify

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

const (
	REDIRECT_URI = "http://localhost:8080/callback"
	AUTH_URL     = "https://accounts.spotify.com/authorize"
	TOKEN_URL    = "https://accounts.spotify.com/api/token"
	API_BASE_URL = "https://api.spotify.com/v1"
)

func ImportSetup() *cobra.Command {
	args := SpotifyImportArgs{}

	spotifyCommand := &cobra.Command{
		Use:   "import <playlist> <tag>",
		Short: "import a spotify playlist to a tag",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := importRunner(positional[0], positional[1], &args); err != nil {
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

	spotifyCommand.Flags().BoolVarP(&args.debug, "debug", "d", config.Debug, "set debug mode")
	spotifyCommand.Flags().StringVarP(&args.musicPath, "music-path", "m", config.MusicPath, "the music path to use")
	return spotifyCommand
}

func open(link string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", link).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", link).Start()
	case "darwin":
		return exec.Command("open", link).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func openAuthTokenUrl(clientId string, clientSecret string, state string) error {
	params := url.Values{}

	params.Set("client_id", clientId)
	params.Set("response_type", "code")
	params.Set("redirect_uri", REDIRECT_URI)
	params.Set("client_id", clientId)
	params.Set("scope", "user-read-private user-read-email")
	params.Set("state", state)

	err := open(AUTH_URL + "?" + params.Encode())

	if err != nil {
		return err
	}

	return nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// https://stackoverflow.com/a/22892986/
func generateRandomState(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func handleCallback(w http.ResponseWriter, r *http.Request, authCodeChan chan<- string, realState string, server *http.Server) {
	query := r.URL.Query()

	code := query.Get("code")
	state := query.Get("state")

	if state != realState {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	authCodeChan <- code
	w.Write([]byte("You can close this tab now"))

	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down server: %v", err)
		}
	}()
}

func exchangeCodeForToken(clientId string, clientSecret string, code string) (SpotifyAuthTokenResponse, error) {
	params := url.Values{}

	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", REDIRECT_URI)

	req, err := http.NewRequest("POST", TOKEN_URL, strings.NewReader(params.Encode()))

	if err != nil {
		return SpotifyAuthTokenResponse{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientId, clientSecret)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return SpotifyAuthTokenResponse{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SpotifyAuthTokenResponse{}, errors.New("Invalid status code: " + resp.Status)
	}

	var tokenResponse SpotifyAuthTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return SpotifyAuthTokenResponse{}, err
	}

	return tokenResponse, nil
}

func setupOrGetCredentials(credentialsPath string) (Credentials, error) {
	rand.Seed(time.Now().UnixNano())
	_, err := os.Stat(credentialsPath)

	var credentialsFile *os.File

	if os.IsNotExist(err) {
		credentialsFile, err = os.Create(credentialsPath)

		if err != nil {
			return Credentials{}, errors.New("Error with creating credentials file: " + err.Error())
		}

		fmt.Println("Created credentials file at " + credentialsPath)
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
		case "client_id":
			credentials.ClientId = value
		case "client_secret":
			credentials.ClientSecret = value
		case "access_token":
			credentials.AccessToken = value
		case "refresh_token":
			credentials.RefreshToken = value
		default:
			return Credentials{}, errors.New("Unknown key in credentials file: " + parts[0])
		}
	}

	credentialsFile.Close()

	if err := scanner.Err(); err != nil {
		return credentials, errors.New("Error with reading credentials file: " + err.Error())
	}

	if credentials.ClientId == "" {
		return credentials, errors.New("Client id not found in credentials file")
	}

	if credentials.ClientSecret == "" {
		return credentials, errors.New("Client secret not found in credentials file")
	}

	if credentials.AccessToken == "" {
		authCodeChan := make(chan string)
		state := generateRandomState(16)
		server := &http.Server{Addr: ":8080"}

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			handleCallback(w, r, authCodeChan, state, server)
		})

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Error starting server: %v", err)
			}
		}()

		err := openAuthTokenUrl(credentials.ClientId, credentials.ClientSecret, state)

		if err != nil {
			return credentials, errors.New("Error getting auth token: " + err.Error())
		}

		code := <-authCodeChan
		authResponse, err := exchangeCodeForToken(credentials.ClientId, credentials.ClientSecret, code)

		if err != nil {
			return credentials, errors.New("Error exchanging code for token: " + err.Error())
		}

		credentials.AccessToken = authResponse.AccessToken
		credentials.RefreshToken = authResponse.RefreshToken
		credentialsFile, err := os.OpenFile(credentialsPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return credentials, errors.New("Error with opening credentials file: " + err.Error())
		}

		_, err = credentialsFile.WriteString("\naccess_token=" + credentials.AccessToken + "\nrefresh_token=" + credentials.RefreshToken)
		if err != nil {
			return credentials, errors.New("Error with writing to credentials file: " + err.Error())
		}
	}

	return credentials, nil
}

func updateLocalPlaylistToMatch(playlistResponse SpotifyPlaylistTracksResponse, localTagName string, args *SpotifyImportArgs) {
	playlistSongs := playlistResponse.Items

	// for i, playlistSong := range playlistSongs {
	// 	fmt.Printf("%d: %s - %s\n", i, playlistSong.Track.Name, playlistSong.Track.Artists[0].Name)
	// }

	err := filepath.WalkDir(args.musicPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)

		if ext != ".mp3" && ext != ".flac" && ext != ".m4a" && ext != ".ogg" {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)

		if err != nil {
			fmt.Println("Error opening file: ", path)
			return nil
		}

		defer file.Close()
		metadata, err := tag.ReadFrom(file)

		if err != nil {
			fmt.Println("Error reading metadata from file: ", path)
			return nil
		}

		title := strings.ToLower(metadata.Title())
		artist := strings.ToLower(metadata.Artist())

		// find if in playlist songs and remove from playlists songs if so
		for i, playlistSong := range playlistSongs {
			playlistSongName := strings.ToLower(playlistSong.Track.Name)
			playlistSongArtist := strings.ToLower(playlistSong.Track.Artists[0].Name)

			if title == playlistSongName && artist == playlistSongArtist {
				playlistSongs = append(playlistSongs[:i], playlistSongs[i+1:]...)
			}
		}

		return nil
	})

	for _, playlistSong := range playlistSongs {
		fmt.Printf("Song not found in local library: %s - %s\n", playlistSong.Track.Name, playlistSong.Track.Artists[0].Name)
	}

	if err != nil {
		fmt.Println("Error walking path: ", err)
	}
}

func refreshToken(creds *Credentials, credsPath string) error {
	params := url.Values{}

	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", creds.RefreshToken)
	params.Set("client_id", creds.ClientId)

	req, err := http.NewRequest("POST", TOKEN_URL, strings.NewReader(params.Encode()))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(creds.ClientId, creds.ClientSecret)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Invalid status code: " + resp.Status)
	}

	var tokenResponse SpotifyAuthTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return err
	}

	fmt.Println(tokenResponse)

	creds.AccessToken = tokenResponse.AccessToken
	creds.RefreshToken = tokenResponse.RefreshToken

	// replace access token in file
	credentialsFile, err := os.OpenFile(credsPath, os.O_RDWR, 0644)

	if err != nil {
		return err
	}

	defer credentialsFile.Close()
	scanner := bufio.NewScanner(credentialsFile)

	var lines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "access_token=") {
			line = "access_token=" + creds.AccessToken
		} else if strings.HasPrefix(line, "refresh_token=") {
			line = "refresh_token=" + creds.RefreshToken
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	credentialsFile.Truncate(0)
	credentialsFile.Seek(0, 0)

	for _, line := range lines {
		_, err := credentialsFile.WriteString(line + "\n")

		if err != nil {
			return err
		}
	}

	return nil
}

func importRunner(playlist string, tagName string, args *SpotifyImportArgs) error {
	cacheDir, err := os.UserCacheDir()

	if err != nil {
		return errors.New("Error getting cache dir: " + err.Error())
	}

	credentialsPath := path.Join(cacheDir, ".music-spotify-credentials")
	creds, err := setupOrGetCredentials(credentialsPath)

	if err != nil {
		return err
	}

	playlistId := strings.TrimPrefix(playlist, "https://open.spotify.com/playlist/")
	fmt.Println("playlistId: ", playlistId)

	params := url.Values{}
	params.Set("limit", "50")

	req, err := http.NewRequest("GET", API_BASE_URL+"/playlists/"+playlistId+"/tracks?"+params.Encode(), nil)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)

	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("Access token expired, refreshing")
		err := refreshToken(&creds, credentialsPath)

		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Invalid status code: " + resp.Status)
	}

	var playlistResponse SpotifyPlaylistTracksResponse

	if err := json.NewDecoder(resp.Body).Decode(&playlistResponse); err != nil {
		return err
	}

	// TODO: currently assumes there are less than 50 songs in the playlist

	updateLocalPlaylistToMatch(playlistResponse, tagName, args)

	return nil
}
