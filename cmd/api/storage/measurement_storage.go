// Package storage is responsible for storing data
package storage

import (
	"sensor/cmd/api/models"
	"sensor/cmd/api/pagination"
	"time"

	"context"

	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	CreateMeasurement(ctx context.Context, m *models.Measurement) (MeasurementRecord, error)
	GetMeasurementsAfterID(ctx context.Context, afterID int64, limit int) ([]MeasurementRecord, int64, error)
}

type MeasurementRecord struct {
	ID        int64     `json:"id"`
	Sensor    string    `json:"sensor"`
	Parameter *string   `json:"parameter,omitempty"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SQLStorage) CreateMeasurement(ctx context.Context, m *models.Measurement) (MeasurementRecord, error) {
	result, err := s.DB.ExecContext(ctx,
		`INSERT INTO measurement (sensor, parameter, value, unit, timestamp_unix, created_at_unix) 
        VALUES (?, ?, ?, ?, ?, ?)`,
		m.Sensor, m.Parameter, m.Value, m.Unit, m.Timestamp.UTC().Unix(), m.CreatedAt.UTC().Unix())
	if err != nil {
		return MeasurementRecord{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return MeasurementRecord{}, err
	}
	return MeasurementRecord{
		ID:        id,
		Sensor:    m.Sensor,
		Parameter: m.Parameter,
		Value:     m.Value,
		Unit:      m.Unit,
		Timestamp: m.Timestamp,
		CreatedAt: m.CreatedAt,
	}, nil
}

func (s *SQLStorage) GetMeasurementsPage(limit int, after *pagination.MeasurementCursor) ([]MeasurementRecord, error) {
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

	out := []MeasurementRecord{}
	for rows.Next() {
		var m MeasurementRecord
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
