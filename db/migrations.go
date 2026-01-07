package dbUtils

import (
	"database/sql"
)

type Migration struct {
	Version int
	Up      string
}

var migrations = []Migration{
	{
		Version: 1,
		Up: `
		create table plays (
			id integer primary key autoincrement,
			fulfilled boolean not null,
			title text not null,
			artist text not null,
			time timestamp not null
		);
		`,
	},
	{
		Version: 2,
		Up: `
		alter table plays add column album text;
		alter table plays add column playedFor integer not null default 180;
		alter table plays add column length integer not null default 180;

		create table plays_temp (
			id integer primary key autoincrement,
			fulfilled boolean not null,
			album text,
			artist text not null,
			title text not null,
			time timestamp not null,
			playedFor integer not null,
			length integer not null
		);

		insert into plays_temp (
			id, fulfilled, album, artist, title, time, playedFor, length
		) select id, fulfilled, album, artist, title, time, playedFor, length from plays;

		drop table plays;
		alter table plays_temp rename to plays;
		`,
	},
	{
		Version: 3,
		Up: `
		alter table plays add column source text;
		`,
	},
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow(`
			select coalesce(max(version),0) from schema_migrations;
	`).Scan(&v)
	return v, err
}

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`
			create table if not exists schema_migrations (
			version integer primary key
		);
	`)

	if err != nil {
		return err
	}

	currentVersion, err := getCurrentVersion(db)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}

		if _, err := tx.Exec(m.Up); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec("insert into schema_migrations (version) values (?)", m.Version); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
