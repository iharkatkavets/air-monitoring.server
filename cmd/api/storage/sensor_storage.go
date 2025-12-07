package storage

import (
	"context"
	"time"
)

type SensorItem struct {
	SensorID   string
	SensorName string
	LastSeen   time.Time
}

type SensorStorage interface {
	UpsertSensor(ctx context.Context, sensorID, sensorName *string, timestamp time.Time)
	GetAllSensors(ctx context.Context) ([]SensorItem, error)
}

func (s *SQLStorage) UpsertSensor(ctx context.Context, sensorID, sensorName *string, timestamp time.Time) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO sensor (sensor_id, sensor_name, last_seen_unix)
        VALUES (?, ?, ?)
        ON CONFLICT(sensor_id) DO UPDATE SET last_seen_unix=excluded.last_seen_unix 
        `,
		sensorID, sensorName, timestamp.UTC().Unix())

	if err != nil {
		s.errorLog.Printf("Failed to add or update sensor id %s name %s %s", *sensorID, *sensorName, err)
		return err
	}
	return nil
}

func (s *SQLStorage) GetAllSensors(ctx context.Context) ([]SensorItem, error) {
	query := `SELECT sensor_id, sensor_name, last_seen_unix from sensor`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		s.errorLog.Printf("Failed to fetch sensors %s", err)
		return nil, err
	}

	result := []SensorItem{}
	for rows.Next() {
		var item SensorItem
		var timestamp int64
		if err := rows.Scan(
			&item.SensorID, &item.SensorName, &timestamp,
		); err != nil {
			s.errorLog.Printf("Failed to scan %s", err)
			return nil, err
		}
		item.LastSeen = time.Unix(timestamp, 0).UTC()
		result = append(result, item)
	}

	return result, nil
}
