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
	CreateMeasurement(ctx context.Context, sensorID *string, sensor *string, m *models.MeasurementValue, timestamp time.Time) (MeasurementRecord, error)
	GetMeasurementsPage(sensorID string, limit int, after *pagination.MeasurementCursor) ([]MeasurementRecord, error)
}

type MeasurementRecord struct {
	ID          int64     `json:"id"`
	Sensor      *string   `json:"sensor"`
	SensorID    *string   `json:"sensor_id"`
	Measurement string    `json:"measurement"`
	Parameter   *string   `json:"parameter,omitempty"`
	Value       float64   `json:"value"`
	Unit        *string   `json:"unit,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *SQLStorage) CreateMeasurement(ctx context.Context, sensorID *string, sensor *string, m *models.MeasurementValue, timestamp time.Time) (MeasurementRecord, error) {
	currTimestamp := time.Now().UTC()
	result, err := s.DB.ExecContext(ctx,
		`INSERT INTO measurement (sensor_id, sensor, measurement, parameter, value, unit, timestamp_unix, created_at_unix) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sensorID, sensor, m.Measurement, m.Parameter, m.Value, m.Unit, timestamp.Unix(), currTimestamp.Unix())
	if err != nil {
		return MeasurementRecord{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return MeasurementRecord{}, err
	}
	return MeasurementRecord{
		ID:          id,
		Sensor:      sensor,
		SensorID:    sensorID,
		Measurement: m.Measurement,
		Parameter:   m.Parameter,
		Value:       m.Value,
		Unit:        m.Unit,
		Timestamp:   timestamp,
		CreatedAt:   currTimestamp,
	}, nil
}

func (s *SQLStorage) GetMeasurementsPage(sensorID string, limit int, after *pagination.MeasurementCursor) ([]MeasurementRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	args := []any{sensorID}
	where := "WHERE sensor_id = ?"

	// Forward pagination: everything "after" the cursor in a DESC order
	if after != nil {
		where += " AND (created_at_unix < ? OR (created_at_unix = ? AND id < ?))"
		cursorUnix := after.CreatedAt.UTC().Unix()
		args = append(args, cursorUnix, cursorUnix, after.ID)
	}

	// Always keep the ORDER BY stable and matching the index
	q := `
		SELECT id, sensor_id, sensor, measurement, parameter, value, unit, timestamp_unix, created_at_unix
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
			&m.ID, &m.SensorID, &m.Sensor, &m.Measurement, &m.Parameter, &m.Value, &m.Unit, &tsUnix, &createdAtUnix,
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
