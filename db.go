package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS expenses (
		id TEXT PRIMARY KEY,
		amount REAL NOT NULL CHECK (amount > 0),
		category TEXT NOT NULL,
		note TEXT,
		spent_on DATE NOT NULL CHECK(spent_on GLOB '____-__-__'),
		created_at DATETIME
	)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}