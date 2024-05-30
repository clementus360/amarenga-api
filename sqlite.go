package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func InitializeDb() (*sql.DB, error) {

	db, err := sql.Open("sqlite3", "./notifications.db")
	if err != nil {
		return nil, err
	}

	// Create jobs table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sessionId TEXT,
		reminderJobID INTEGER,
		startJobID INTEGER,
		interpreterToken TEXT,
		sessionDate TEXT,
		reminderDate TEXT,
		userToken TEXT,
		sessionTimestamp TEXT,
		reminderTimestamp TEXT
	);
	`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	return db, nil
}
