package lastfm

import (
	"sort"
	"time"
)

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

type SongListenRange struct {
	Start float64
	End   float64
}

type CurrentTrackInfo struct {
	Track    string
	Artist   string
	Album    string
	Duration float64

	StartTime  time.Time
	LastUpdate time.Time

	LastPosition float64

	ListenTime float64
	WallTime   float64 // real time passed since starting to play

	Ranges    []SongListenRange
	SeekCount int

	OpenRangeStart float64
	RangeOpen      bool
}

func (cti *CurrentTrackInfo) ResetMetrics() {
	cti.Duration = 0
	cti.StartTime = time.Time{}
	cti.LastUpdate = time.Time{}
	cti.LastPosition = 0
	cti.ListenTime = 0
	cti.WallTime = 0
	cti.Ranges = []SongListenRange{}
	cti.SeekCount = 0
	cti.OpenRangeStart = 0
	cti.RangeOpen = false
}

func (cti *CurrentTrackInfo) CloseOpenRange(currentPosition float64) {
	if cti.RangeOpen {
		cti.AddListenRange(cti.OpenRangeStart, currentPosition)
		cti.RangeOpen = false
	}
}

func (cti *CurrentTrackInfo) AddListenRange(start, end float64) {
	if end > start {
		cti.Ranges = append(cti.Ranges, SongListenRange{
			Start: start,
			End:   end,
		})
	}
}

func (cti *CurrentTrackInfo) UniqueCoverageAndMaxPosition() (float64, float64) {
	if len(cti.Ranges) == 0 {
		return 0.0, 0.0
	}

	sort.Slice(cti.Ranges, func(i, j int) bool {
		return cti.Ranges[i].Start < cti.Ranges[j].Start
	})

	covered := 0.0
	currentStart := cti.Ranges[0].Start
	currentEnd := cti.Ranges[0].End

	for _, r := range cti.Ranges[1:] {
		if r.Start <= currentEnd {
			currentEnd = max(currentEnd, r.End)
		} else {
			covered += currentEnd - currentStart
			currentStart = r.Start
			currentEnd = r.End
		}
	}

	covered += currentEnd - currentStart
	return covered, currentEnd
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
