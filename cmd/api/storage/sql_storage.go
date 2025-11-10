package storage

import (
	"context"
	"database/sql"
	"sensor/cmd/api/settings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLStorage struct {
	DB *sql.DB
}

func NewSQLStorage(db *sql.DB) *SQLStorage {
	s := &SQLStorage{DB: db}
	return s
}

func (s *SQLStorage) InitDB(ctx context.Context) error {
	if err := s.createTables(); err != nil {
		return err
	}
	if err := s.createIndexes(); err != nil {
		return err
	}
	if err := s.EnsureDefaultSettings(ctx, settings.DefaultSettings); err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createTables() error {
	if err := s.createMeasurementTable(); err != nil {
		return err
	}
	if err := s.createSettingsTable(); err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createMeasurementTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS measurement (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        sensor STRING NOT NULL,
        parameter STRING,
        value REAL NOT NULL,
        unit STRING NOT NULL,
        timestamp DATETIME NOT NULL,
        created_at DATETIME NOT NULL
    )
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createSettingsTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS settings (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        parameter TEXT,
        updated_at DATETIME NOT NULL
    )
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createIndexes() error {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('measurement') 
		WHERE name = 'created_at'
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	_, err = s.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_measurement_created_id
		ON measurement(created_at, id);
	`)
	return err
}

func (s *SQLStorage) EnsureDefaultSettings(ctx context.Context, defaults map[string]string) error {
	for key, value := range defaults {
		query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO NOTHING;
		`
		if _, err := s.DB.ExecContext(ctx, query, key, value); err != nil {
			return err
		}
	}
	return nil
}
