package dbUtils

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(filePath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filePath)

	if err != nil {
		return nil, err
	}
	return db, nil
}
