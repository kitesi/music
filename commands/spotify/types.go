package spotify

type SpotifyImportArgs struct {
	debug     bool
	musicPath string
}

type SpotifyAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type SpotifyPlaylistTracksResponse struct {
	Total int                          `json:"total"`
	Next  string                       `json:"next"`
	Items []SpotifyPlaylistTrackObject `json:"items"`
}

type SpotifyAlbumTracksResponse struct {
	Total int                  `json:"total"`
	Next  string               `json:"next"`
	Items []SpotifyTrackObject `json:"items"`
}

type SpotifyPlaylistTrackObject struct {
	// track can be either TrackObject or EpisodeObject, we will ignore it if it's an episode
	Track SpotifyTrackObject `json:"track"`
}

type SpotifyTrackObject struct {
	Type       string                `json:"type"`
	Album      SpotifyAlbumObject    `json:"album"`
	Artists    []SpotifyArtistObject `json:"artists"`
	DurationMs int                   `json:"duration_ms"`
	Id         string                `json:"id"`
	Name       string                `json:"name"`
}

type SpotifyAlbumObject struct {
	Name string `json:"name"`
}

type SpotifyArtistObject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SimilarityInfo struct {
	path       string
	similarity float64
}

type MatchedSpotifyToLocal struct {
	spotify string
	local   string
}
