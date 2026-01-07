package dbUtils

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Play struct {
	ID        int64
	Fulfilled bool
	Title     string
	Artist    string
	Time      time.Time
}

type InsertIntoPlaysParams struct {
	Fulfilled bool
	Title     string
	Artist    string
	Album     string
	PlayedFor int
	Length    int
	StartTime time.Time
	Source    string
}

const INSERT_INTO_PLAYS_QUERY = `
	insert into plays (fulfilled, title, artist, album, playedFor, length, time, source) values (?, ?, ?, ?, ?, ?, ?, ?)
`

func InsertIntoPlays(db *sql.DB, params InsertIntoPlaysParams) error {
	_, err := db.Exec(
		INSERT_INTO_PLAYS_QUERY,
		params.Fulfilled,
		params.Title,
		params.Artist,
		params.Album,
		params.PlayedFor,
		params.Length,
		params.StartTime,
		params.Source,
	)
	return err
}

// for now no pagination, just assume we can fit all unfulfilled plays in memory
const GET_UNFULFILLED_PLAYS_QUERY = `
	select * from plays where fulfilled = false;
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
		if err := rows.Scan(&play.ID, &play.Fulfilled, &play.Title, &play.Artist, &play.Time); err != nil {
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
	update plays set fulfilled = true where id in (%s)
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
