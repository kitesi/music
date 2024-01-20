package lastfm

type Credentials struct {
	ApiKey     string
	ApiSecret  string
	SessionKey string
	Username   string
}

type Session struct {
	Name       string
	Key        string
	Subscriber int
}

type GetAuthTokenResponse struct {
	Token   string
	Message string
	Error   int
}

type GetSessionResponse struct {
	Session Session
	Message string
	Error   int
}

type PostScrobbleResponse struct {
	Scrobbles struct {
		Scrobble struct {
			Artist struct {
				Text      string `json:"#text"`
				Corrected string
			}
			Album struct {
				Text      string `json:"#text"`
				Corrected string
			}
			Track struct {
				Text      string `json:"#text"`
				Corrected string
			}
			AlbumArtist struct {
				Text      string `json:"#text"`
				Corrected string
			}
			IgnoredMessage struct {
				Code string
				Text string `json:"#text"`
			}
			Timestamp string
		}
		Attr struct {
			Ignored  int
			Accepted int
		} `json:"@attr"`
	}
}

// same interface for /mix /library
type GetLastfmSuggestionsResponse struct {
	Playlist []struct {
		Name     string
		Url      string
		Duration int
		Artists  []struct {
			Name string
		}
		Playlinks []struct {
			Url       string
			Id        string
			Source    string
			Affiliate string
		}
	}
}

type LastfmWatchArgs struct {
	interval int
	debug    bool
}

type LastfmSuggestArgs struct {
	debug     bool
	printUrls bool
	install   bool
	limit     int
	musicPath string
	format    string
}
