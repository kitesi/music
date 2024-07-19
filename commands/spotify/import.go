package spotify

import (
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
	"regexp"
	"runtime"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/dhowden/tag"
	"github.com/kitesi/music/commands/tags"
	"github.com/kitesi/music/simpleconfig"
	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
	"golang.org/x/text/unicode/norm"
)

const (
	REDIRECT_URI = "http://localhost:8080/callback"
	AUTH_URL     = "https://accounts.spotify.com/authorize"
	TOKEN_URL    = "https://accounts.spotify.com/api/token"
	API_BASE_URL = "https://api.spotify.com/v1"
)

func ImportSetup() *cobra.Command {
	args := SpotifyImportArgs{}
	config, err := utils.GetConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}

	spotifyCommand := &cobra.Command{
		Use:   "import <tag> [playlist]",
		Short: "import a spotify playlist to a tag",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := importRunner(positional, &args, config); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
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
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
	<html>
	<head>
		<title>Authorization Complete</title>
	</head>
	<body>
		<p>Authorization complete. This window will close automatically.</p>
		<script type="text/javascript">
			window.close();
		</script>
	</body>
	</html>`))

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

func setupOrGetCredentials(credentialsPath string) (simpleconfig.Config, error) {
	credentials, err := simpleconfig.NewConfig(credentialsPath, []string{"access_token", "refresh_token", "client_id", "client_secret"})

	if err != nil {
		return credentials, err
	}

	clientId, _ := credentials.Get("client_id")
	clientSecret, _ := credentials.Get("client_secret")

	if clientId == "" {
		return credentials, errors.New("No client_id found in credentials file")
	}

	if clientSecret == "" {
		return credentials, errors.New("No client_secret found in credentials file")
	}

	if accessToken, _ := credentials.Get("access_token"); accessToken == "" {
		authResponse, err := openServerAndGetAuthToken(clientId, clientSecret)

		if err != nil {
			return credentials, err
		}

		credentials.Set("access_token", authResponse.AccessToken)
		credentials.Set("refresh_token", authResponse.RefreshToken)

		err = credentials.WriteConfig()

		if err != nil {
			return credentials, errors.New("Error writing credentials to file: " + err.Error())
		}
	}

	return credentials, nil
}

func normalizeString(s string) string {
	return norm.NFC.String(strings.ToLower(s))
}

func getSongId(song SpotifyTrackObject) string {
	return song.Artists[0].Name + " - " + song.Name
}

var featureRegex = regexp.MustCompile(`(?i)\(?(ft\.?|feat\.?)\)?`)

func testFileAgainstPlaylist(fileName string, playlistSongs *[]SpotifyTrackObject, metricCmp strutil.StringMetric, mostSimilar *map[string]SimilarityInfo, onMatch func(int, string)) error {
	file, err := os.Open(fileName)

	if err != nil {
		return errors.New("Error opening file: " + fileName + " - " + err.Error())
	}

	defer file.Close()
	metadata, err := tag.ReadFrom(file)

	if err != nil {
		return errors.New("Error reading metadata from file: " + fileName + " - " + err.Error())
	}

	title := normalizeString(metadata.Title())
	artist := normalizeString(metadata.Artist())

	for i, playlistSong := range *playlistSongs {
		playlistSongName := playlistSong.Name
		playlistSongArtist := playlistSong.Artists[0].Name

		titleSimilarity := strutil.Similarity(title, playlistSongName, metricCmp)
		artistSimilarity := strutil.Similarity(artist, playlistSongArtist, metricCmp)

		// handle "ft" in artist name or song title, usually spotify has it in
		// the title rather than a sep artist as exif does
		if featureRegex.MatchString(playlistSongName) {
			parts := featureRegex.Split(playlistSongName, -1)
			featuring := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")

			if strings.Contains(artist, featuring) {
				playlistSongName = strings.TrimSpace(parts[0])
			}
		}

		if strings.Contains(playlistSongArtist, artist) || strings.Contains(artist, playlistSongArtist) {
			artistSimilarity = 1
		}

		similarity := (titleSimilarity + artistSimilarity) / 2
		songId := getSongId(playlistSong)

		if similarity > (*mostSimilar)[songId].similarity {
			(*mostSimilar)[songId] = SimilarityInfo{path: fileName, similarity: similarity}
		}

		if title == playlistSongName && (artist == playlistSongArtist || artistSimilarity == 1) {
			onMatch(i, fileName)
			delete((*mostSimilar), songId)
		}
	}

	return nil
}

func updateLocalPlaylistToMatch(playlistSongs []SpotifyTrackObject, localTagName string, args *SpotifyImportArgs) error {
	fmt.Println("There are", len(playlistSongs), "songs in playlist")

	for i, playlistSong := range playlistSongs {
		if playlistSong.Type == "episode" {
			fmt.Println("Skipping episode: ", playlistSong.Name)
			playlistSongs = append(playlistSongs[:i], playlistSongs[i+1:]...)
		} else {
			playlistSongs[i].Name = normalizeString(playlistSong.Name)
			playlistSongs[i].Artists[0].Name = normalizeString(playlistSong.Artists[0].Name)
		}
	}

	storedTags, err := tags.GetStoredTags(args.musicPath)

	if err != nil {
		return errors.New("Error getting tags: " + err.Error())
	}

	tagSongs, ok := storedTags[localTagName]

	if !ok {
		fmt.Printf("Tag (%s) not found, create? (y/n): ", localTagName)

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) == "y" {
			tagSongs = []string{}
		} else {
			return nil
		}
	}

	tagSongIndex := 0
	foundTaggedSongs := []MatchedSpotifyToLocal{}
	mostSimilarTagged := map[string]SimilarityInfo{}
	metricCmp := metrics.NewLevenshtein()

	onTaggedMatch := func(i int, fileName string) {
		foundTaggedSongs = append(foundTaggedSongs, MatchedSpotifyToLocal{spotify: getSongId(playlistSongs[i]), local: fileName})
		tagSongs = append(tagSongs[:tagSongIndex], tagSongs[tagSongIndex+1:]...)
		playlistSongs = append(playlistSongs[:i], playlistSongs[i+1:]...)
		tagSongIndex--
	}

	for tagSongIndex < len(tagSongs) && len(playlistSongs) > 0 {
		testFileAgainstPlaylist(tagSongs[tagSongIndex], &playlistSongs, metricCmp, &mostSimilarTagged, onTaggedMatch)
		tagSongIndex++
	}

	if len(foundTaggedSongs) != 0 {
		fmt.Println("\nFound", len(foundTaggedSongs), "songs from the spotify playlist that are already tagged:")
		for _, song := range foundTaggedSongs {
			fmt.Println("- " + song.spotify + " -> " + song.local)
		}
	}

	if len(mostSimilarTagged) != 0 {
		fmt.Println("\nSimilarity of playlist songs to tagged songs:")
		for spotifySong, mostSimilar := range mostSimilarTagged {
			if mostSimilar.similarity > 0.9 {
				fmt.Printf("- Assuming %s -> %s (%.2f%%)\n", mostSimilar.path, spotifySong, mostSimilar.similarity*100)
				foundTaggedSongs = append(foundTaggedSongs, MatchedSpotifyToLocal{spotify: spotifySong, local: mostSimilar.path})

				j := 0
				for j < len(playlistSongs) && getSongId(playlistSongs[j]) != spotifySong {
					j++
				}
				if getSongId(playlistSongs[j]) == spotifySong {
					playlistSongs = append(playlistSongs[:j], playlistSongs[j+1:]...)
				}

				j = 0
				for j < len(tagSongs) && tagSongs[j] != mostSimilar.path {
					j++
				}
				if tagSongs[j] == mostSimilar.path {
					tagSongs = append(tagSongs[:j], tagSongs[j+1:]...)
				}
			} else {
				fmt.Printf("- Skipping %s -> %s (%.2f%%)\n", mostSimilar.path, spotifySong, mostSimilar.similarity*100)
			}
		}
	}

	foundUntaggedSongs := []MatchedSpotifyToLocal{}
	mostSimilarUntagged := map[string]SimilarityInfo{}
	onUntaggedMatch := func(i int, fileName string) {
		foundUntaggedSongs = append(foundUntaggedSongs, MatchedSpotifyToLocal{spotify: getSongId(playlistSongs[i]), local: fileName})
		playlistSongs = append(playlistSongs[:i], playlistSongs[i+1:]...)
	}

	err = filepath.WalkDir(args.musicPath, func(path string, d os.DirEntry, err error) error {
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

		testFileAgainstPlaylist(path, &playlistSongs, metricCmp, &mostSimilarUntagged, onUntaggedMatch)
		return nil
	})

	if len(foundUntaggedSongs) != 0 {
		fmt.Println("\nFound", len(foundUntaggedSongs), "songs from the spotify playlist that are not tagged:")
		for _, song := range foundUntaggedSongs {
			fmt.Println("- " + song.spotify + " -> " + song.local)
		}
	}

	if len(mostSimilarUntagged) != 0 {
		fmt.Println("\nSimilarity of playlist songs to untagged songs:")

		for spotifySong, mostSimilar := range mostSimilarUntagged {
			if mostSimilar.similarity > 0.9 {
				fmt.Printf("- Assuming %s -> %s (%.2f%%)\n", mostSimilar.path, spotifySong, mostSimilar.similarity*100)
				foundUntaggedSongs = append(foundUntaggedSongs, MatchedSpotifyToLocal{spotify: spotifySong, local: mostSimilar.path})

				j := 0

				for j < len(playlistSongs) && getSongId(playlistSongs[j]) != spotifySong {
					j++
				}

				playlistSongs = append(playlistSongs[:j], playlistSongs[j+1:]...)
			} else {
				fmt.Printf("- Skipping %s -> %s (%.2f%%)\n", mostSimilar.path, spotifySong, mostSimilar.similarity*100)
			}
		}
	}

	if len(playlistSongs) != 0 {
		fmt.Println("\nCould not find", len(playlistSongs), "songs from the spotify playlist in the local library:")

		for _, playlistSong := range playlistSongs {
			fmt.Printf("- %s - %s\n", playlistSong.Artists[0].Name, playlistSong.Name)
		}
	}

	if len(tagSongs) != 0 {
		fmt.Println("\nCould not find", len(tagSongs), "songs that are tagged in the spotify playlist:")

		for _, tagSong := range tagSongs {

			fmt.Println("- " + tagSong)
		}
	}

	if err != nil {
		return errors.New("Error walking music path: " + err.Error())
	}

	toAppend := []string{}

	for _, song := range foundUntaggedSongs {
		toAppend = append(toAppend, song.local)
	}

	if len(toAppend) != 0 {
		fmt.Println("\nAdding", len(toAppend), "songs to tag:", localTagName)
		err := tags.ChangeSongsInTag(args.musicPath, localTagName, toAppend, true)

		if err != nil {
			return errors.New("Error changing songs in tag: " + err.Error())
		}
	}

	return nil
}

func openServerAndGetAuthToken(clientId, clientSecret string) (SpotifyAuthTokenResponse, error) {
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

	err := openAuthTokenUrl(clientId, clientSecret, state)

	if err != nil {
		return SpotifyAuthTokenResponse{}, errors.New("Error getting auth token: " + err.Error())
	}

	code := <-authCodeChan
	authResponse, err := exchangeCodeForToken(clientId, clientSecret, code)

	if err != nil {
		return SpotifyAuthTokenResponse{}, errors.New("Error exchanging code for token: " + err.Error())
	}

	return authResponse, nil
}

func refreshToken(creds *simpleconfig.Config, credsPath string) error {
	clientId, _ := creds.Get("client_id")
	clientSecret, _ := creds.Get("client_secret")
	refreshToken, _ := creds.Get("refresh_token")

	if refreshToken == "" {
		fmt.Println("No refresh token found, re-authenticating...")
		authResponse, err := openServerAndGetAuthToken(clientId, clientSecret)

		if err != nil {
			return err
		}

		creds.Set("access_token", authResponse.AccessToken)
		creds.Set("refresh_token", authResponse.RefreshToken)
	} else {

		params := url.Values{}

		params.Set("grant_type", "refresh_token")
		params.Set("refresh_token", refreshToken)
		params.Set("client_id", clientId)

		req, err := http.NewRequest("POST", TOKEN_URL, strings.NewReader(params.Encode()))

		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientId, clientSecret)

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

		creds.Set("access_token", tokenResponse.AccessToken)
		creds.Set("refresh_token", tokenResponse.RefreshToken)
	}

	err := creds.WriteConfig()

	if err != nil {
		return errors.New("Error writing credentials to file: " + err.Error())
	}

	return nil
}

func importRunner(positional []string, args *SpotifyImportArgs, config utils.Config) error {
	tagName := positional[0]
	playlist := config.TagPlaylistAssociations[tagName]

	if len(positional) == 2 {
		playlist = positional[1]
	} else if playlist == "" {
		return errors.New("No playlist associated with tag: " + tagName + ". Please provide a playlist URL")
	}

	cacheDir, err := os.UserCacheDir()

	if err != nil {
		return errors.New("Error getting cache dir: " + err.Error())
	}

	credentialsPath := path.Join(cacheDir, ".music-spotify-credentials")
	creds, err := setupOrGetCredentials(credentialsPath)

	if err != nil {
		return err
	}

	apiSecondParam := "playlists"
	playlistId := playlist

	if strings.HasPrefix(playlist, "https://open.spotify.com/album/") {
		apiSecondParam = "albums"
		playlistId = strings.TrimPrefix(playlist, "https://open.spotify.com/album/")
	} else {
		playlistId = strings.TrimPrefix(playlist, "https://open.spotify.com/playlist/")
	}

	params := url.Values{}
	params.Set("limit", "50")

	requestUrl := API_BASE_URL + "/" + apiSecondParam + "/" + playlistId + "/tracks?" + params.Encode()
	req, err := http.NewRequest("GET", requestUrl, nil)

	if err != nil {
		return err
	}

	accessToken, _ := creds.Get("access_token")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)

	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("Access token expired, refreshing")
		err := refreshToken(&creds, credentialsPath)

		if err != nil {
			return err
		}

		fmt.Println("Done. Retrying now...")
		return importRunner(positional, args, config)
	} else if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("Invalid status code: " + resp.Status)
	}

	defer resp.Body.Close()

	if apiSecondParam == "playlists" {
		var playlistResponse SpotifyPlaylistTracksResponse

		if err := json.NewDecoder(resp.Body).Decode(&playlistResponse); err != nil {
			return err
		}

		// TODO: currently assumes there are less than 50 songs in the playlist

		playlistSongs := []SpotifyTrackObject{}

		for _, item := range playlistResponse.Items {
			playlistSongs = append(playlistSongs, item.Track)
		}

		return updateLocalPlaylistToMatch(playlistSongs, tagName, args)
	} else {
		var albumResponse SpotifyAlbumTracksResponse

		if err := json.NewDecoder(resp.Body).Decode(&albumResponse); err != nil {
			return err
		}

		return updateLocalPlaylistToMatch(albumResponse.Items, tagName, args)
	}
}
