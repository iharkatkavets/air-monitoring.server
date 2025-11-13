// Package storage is responsible for storing data
package storage

import (
	"sensor/cmd/api/models"
	"sensor/cmd/api/pagination"
	"time"

	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	CreateMeasurement(m *models.Measurement) error
	GetLastID() (int64, error)
	GetMeasurementsAfterID(ctx context.Context, afterID int64, limit int) ([]models.Measurement, int64, error)
}

func (s *SQLStorage) CreateMeasurement(m *models.Measurement) error {
	query := `INSERT INTO measurement 
    (sensor, parameter, value, unit, timestamp_unix, created_at_unix) VALUES 
    (?, ?, ?, ?, ?, ?)`
	result, err := s.DB.Exec(query, m.Sensor, m.Parameter, m.Value, m.Unit, m.Timestamp.UTC().Unix(), m.CreatedAt.UTC().Unix())
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

func (s *SQLStorage) GetMeasurementsPage(limit int, after *pagination.MeasurementCursor) ([]models.Measurement, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	args := []any{}
	where := ""

	// Forward pagination: everything "after" the cursor in a DESC order
	if after != nil {
		where = "WHERE (created_at_unix < ? OR (created_at_unix = ? AND id < ?))"
		cursorUnix := after.CreatedAt.UTC().Unix()
		args = append(args, cursorUnix, cursorUnix, after.ID)
	}

	// Always keep the ORDER BY stable and matching the index
	q := `
		SELECT id, sensor, parameter, value, unit, timestamp_unix, created_at_unix
		FROM measurement
		` + where + `
		ORDER BY created_at_unix DESC, id DESC
		LIMIT ?
	`
	args = append(args, limit+1)

	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Measurement{}
	for rows.Next() {
		var m models.Measurement
		var tsUnix, createdAtUnix int64
		if err := rows.Scan(
			&m.ID, &m.Sensor, &m.Parameter, &m.Value, &m.Unit, &tsUnix, &createdAtUnix,
		); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(tsUnix, 0)
		m.CreatedAt = time.Unix(createdAtUnix, 0)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLStorage) GetLastID() (int64, error) {
	q := `SELECT id FROM measurement ORDER BY id DESC LIMIT 1`
	row := s.DB.QueryRow(q)

	var id int64
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (s *SQLStorage) GetMeasurementsAfterID(ctx context.Context, afterID int64, limit int) ([]models.Measurement, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := `
        SELECT id, sensor, COALESCE(parameter, ''), value, unit, timestamp_unix, created_at_unix
        FROM measurement
        WHERE id > ?
        ORDER BY id ASC
        LIMIT ?
    `
	rows, err := s.DB.QueryContext(ctx, q, afterID, limit+1)
	if err != nil {
		return nil, afterID, err
	}
	defer rows.Close()

	out := make([]models.Measurement, 0, limit)
	outID := afterID
	for rows.Next() {
		var m models.Measurement
		var tsUnix, createdAtUnix int64
		if err := rows.Scan(&m.ID, &m.Sensor, &m.Parameter, &m.Value, &m.Unit, &tsUnix, &createdAtUnix); err != nil {
			return nil, afterID, err
		}
		m.Timestamp = time.Unix(tsUnix, 0)
		m.CreatedAt = time.Unix(createdAtUnix, 0)
		out = append(out, m)
		outID = m.ID
	}
	if err := rows.Err(); err != nil {
		return nil, afterID, err
	}

	return out, outID, nil
}
