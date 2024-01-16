package lastfm

type Credentials struct {
	ApiKey     string
	ApiSecret  string
	SessionKey string
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

type GetSessionKeyResponse struct {
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

type LastfmArgs struct {
	interval int
	debug    bool
}
