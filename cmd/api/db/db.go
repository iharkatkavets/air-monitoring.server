// Package db works with database
package db

import (
	"database/sql"
	"fmt"
	"time"
)

func NewDB(fileName string) (*sql.DB, error) {
	uri := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000", fileName)
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	return db, nil
}
