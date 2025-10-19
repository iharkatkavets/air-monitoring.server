package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"sensor/cmd/api/models"
)

type Storage interface {
	CreateMeasurement(m *models.Measurement) error
	GetAllMeasurements(filters map[string]string) ([]models.Measurement, error)
}

type SQLStorage struct {
	DB *sql.DB
}

func NewSQLStorage(db *sql.DB) *SQLStorage {
	s := &SQLStorage{DB: db}
	s.createTables()
	s.createIndexes()
	return s
}

func (s *SQLStorage) InitDB() error {
	return s.createTables()
}

func (s *SQLStorage) createTables() error {
	createMeasurementTable := `
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
	_, err := s.DB.Exec(createMeasurementTable)
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
