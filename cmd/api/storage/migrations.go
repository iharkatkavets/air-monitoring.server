package storage

import (
	"database/sql"
	"fmt"
	"log"
)

type Migrations struct {
	DB     *sql.DB
	Logger *log.Logger // optional
}

func NewMigrations(db *sql.DB, logger *log.Logger) *Migrations {
	return &Migrations{DB: db, Logger: logger}
}

func (m *Migrations) AddCreatedAt() error {
	var count int
	err := m.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('measurement') 
		WHERE name = 'created_at'
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		m.Logger.Println("Adding `created_at` column...")

		// Step 1: Add as NULLABLE
		if _, err = m.DB.Exec(`ALTER TABLE measurement ADD COLUMN created_at DATETIME;`); err != nil {
			return err
		}

		// Step 2: Backfill with timestamp or current time
		if _, err = m.DB.Exec(`UPDATE measurement SET created_at = COALESCE(timestamp, datetime('now'));`); err != nil {
			return fmt.Errorf("backfill created_at: %w", err)
		}

		if _, err := m.DB.Exec(`
        CREATE INDEX IF NOT EXISTS idx_measurement_created_id
        ON measurement(created_at DESC, id DESC);
        `); err != nil {
			log.Fatal(err)
		}
		m.Logger.Println("Column `created_at` added and backfilled.")
	} else {
		m.Logger.Println("Column `created_at` already exists.")
	}
	return nil
}

func (m *Migrations) Run() error {
	if err := m.AddCreatedAt(); err != nil {
		return err
	}
	return nil
}
