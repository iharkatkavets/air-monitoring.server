package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sensor/cmd/api/settings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLStorage struct {
	DB       *sql.DB
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewSQLStorage(db *sql.DB, infoLog *log.Logger, errorLog *log.Logger) *SQLStorage {
	s := &SQLStorage{DB: db, infoLog: infoLog, errorLog: errorLog}
	return s
}

func (s *SQLStorage) InitDB(ctx context.Context) error {
	if err := s.createTables(); err != nil {
		return err
	}
	if err := s.createIndexByIDAndCreatedAtUnix(); err != nil {
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
	if err := s.createSettingTable(); err != nil {
		return err
	}
	if err := s.createSensorTable(); err != nil {
		return err
	}
	if err := s.createSensorMeasurementTable(); err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createMeasurementTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS measurement (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        sensor_name TEXT NOT NULL,
        sensor_id TEXT,
        measurement TEXT NOT NULL,
        parameter TEXT,
        value REAL NOT NULL,
        unit TEXT,
        timestamp_unix INTEGER NOT NULL,
        created_at_unix INTEGER NOT NULL
    );
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createSettingTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS setting (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        parameter TEXT,
        updated_at_unix INTEGER NOT NULL
    )
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createSensorTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS sensor (
        sensor_id TEXT PRIMARY KEY,
        sensor_name TEXT NOT NULL,
        last_seen_unix INTEGER NOT NULL
    )
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createSensorMeasurementTable() error {
	sqlCreate := `
    CREATE TABLE IF NOT EXISTS sensor_measurement (
        sensor_id TEXT NOT NULL,
        name TEXT NOT NULL,
        PRIMARY KEY (sensor_id, name),
        FOREIGN KEY (sensor_id) REFERENCES sensor(sensor_id) ON DELETE CASCADE
    )
    `
	_, err := s.DB.Exec(sqlCreate)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) createIndexByIDAndCreatedAtUnix() error {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('measurement') 
		WHERE name = 'created_at_unix'
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		s.errorLog.Println("Can't create index because colum 'created_at_unix' doesn't exist")
		return errors.New("can't create index because column 'created_at_unix' doesn't exist")
	}

	_, err = s.DB.Exec(`
        CREATE INDEX IF NOT EXISTS idx_measurement_created_at_unix_id
        ON measurement(created_at_unix DESC, id DESC);
	`)
	return err
}

func (s *SQLStorage) EnsureDefaultSettings(ctx context.Context, defaults map[string]string) error {
	for key, value := range defaults {
		query := `
		INSERT INTO setting (key, value, updated_at_unix)
		VALUES (?, ?, strftime('%s', 'now'))
		ON CONFLICT(key) DO NOTHING;
		`
		if _, err := s.DB.ExecContext(ctx, query, key, value); err != nil {
			return err
		}
	}
	return nil
}
