package dbUtils

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Play struct {
	ID        int64
	Album     string
	Artist    string
	Title     string
	StartTime time.Time
}

type InsertIntoPlaysParams struct {
	Scrobbable     bool
	Fulfilled      bool
	Album          string
	Artist         string
	Title          string
	Duration       int
	ListenTime     int
	WallTime       int
	MaxPosition    int
	UniqueCoverage int
	SeekCount      int
	StartTime      time.Time
	Source         string
}

const INSERT_INTO_PLAYS_QUERY = `
	insert into plays 
	(scrobbable, fulfilled, 
	album, artist, title, 
	duration, listen_time, wall_time, max_position, unique_coverage, seek_count,
	started_at, source) 

	values (?, ?, 
	?, ?, ?, 
	?, ?, ?, ?, ?, ?,
	?, ?);
`

func InsertIntoPlays(db *sql.DB, params InsertIntoPlaysParams) error {
	_, err := db.Exec(
		INSERT_INTO_PLAYS_QUERY,
		params.Scrobbable,
		params.Fulfilled,
		params.Album,
		params.Artist,
		params.Title,
		params.Duration,
		params.ListenTime,
		params.WallTime,
		params.MaxPosition,
		params.UniqueCoverage,
		params.SeekCount,
		params.StartTime,
		params.Source,
	)
	return err
}

// for now no pagination, just assume we can fit all unfulfilled plays in memory
const GET_UNFULFILLED_PLAYS_QUERY = `
	select id,album,artist,title,started_at from plays where fulfilled = false and scrobbable = true;
`

func GetUnfulfilledPlays(db *sql.DB) ([]Play, error) {
	rows, err := db.Query(GET_UNFULFILLED_PLAYS_QUERY)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var plays []Play

	for rows.Next() {
		var play Play
		if err := rows.Scan(&play.ID, &play.Album, &play.Artist, &play.Title, &play.StartTime); err != nil {
			return nil, err
		}
		plays = append(plays, play)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return plays, nil
}

const UPDATE_UNFULFILLED_PLAYS_QUERY_HELPER = `
	update plays set fulfilled = true where id in (%s);
`

func UpdateUnfulfilledPlays(db *sql.DB, plays []Play) error {
	if len(plays) == 0 {
		return nil
	}

	placeholders := make([]string, len(plays))
	args := make([]any, len(plays))

	for i, play := range plays {
		placeholders[i] = "?"
		args[i] = play.ID
	}

	query := fmt.Sprintf(UPDATE_UNFULFILLED_PLAYS_QUERY_HELPER, strings.Join(placeholders, ","))
	_, err := db.Exec(query, args...)
	return err
}
