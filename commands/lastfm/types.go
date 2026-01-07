package lastfm

import "time"

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

type PostMultipleScrobbleResponse struct {
	Scrobbles struct {
		Scrobble []struct {
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

type GetRecentTracksResponse struct {
	RecentTracks struct {
		Track []struct {
			Artist struct {
				Text string `json:"#text"`
			}
			Name string
			Date struct {
				Uts  string `json:"uts"`
				Text string `json:"#text"`
			}
		}
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

type CurrentTrackInfo struct {
	Track        string
	Artist       string
	Album        string
	LastPosition float64
	StartTime    time.Time
	Length       float64
}

type LastfmWatchArgs struct {
	interval       int
	minTrackLength int
	minListenTime  int
	debug          bool
	logDbFile      string
	source         string
}

type LastfmSuggestArgs struct {
	debug     bool
	printUrls bool
	install   bool
	limit     int
	musicPath string
	format    string
}

type LastfmRecentArgs struct {
	debug    bool
	limit    int
	username string
	json     bool
}

type LastfmImportArgs struct {
	debug bool
}
