package storage

import (
	"context"
	"database/sql"
	"time"
)

type SensorItem struct {
	SensorID   string
	SensorName string
	LastSeen   time.Time
}

type SensorWithMeasurements struct {
	SensorID     string
	SensorName   string
	LastSeen     time.Time
	Measurements []string
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
	defer rows.Close()

	result := []SensorItem{}
	for rows.Next() {
		var item SensorItem
		var timestamp int64
		if err := rows.Scan(
			&item.SensorID, &item.SensorName, &timestamp,
		); err != nil {
			s.errorLog.Printf("Failed to scan sensor row: %v", err)
			return nil, err
		}
		item.LastSeen = time.Unix(timestamp, 0).UTC()
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		s.errorLog.Printf("Row iteration error: %v", err)
		return nil, err
	}

	return result, nil
}

func (s *SQLStorage) UpdateSensorMeasurement(
	ctx context.Context,
	sensorID string,
	measurement string,
) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO sensor_measurement (sensor_id, name)
         VALUES (?, ?)
         ON CONFLICT(sensor_id, name) DO NOTHING`,
		sensorID, measurement)

	if err != nil {
		s.errorLog.Printf("Failed to add measurement %s for sensor %s: %v",
			measurement, sensorID, err)
		return err
	}
	return nil
}

func (s *SQLStorage) GetAllSensorsWithMeasurements(ctx context.Context) ([]SensorWithMeasurements, error) {
	const query = `
        SELECT s.sensor_id, s.sensor_name, s.last_seen_unix, sm.name
        FROM sensor s
        LEFT JOIN sensor_measurement sm ON sm.sensor_id = s.sensor_id
        ORDER BY s.sensor_id
    `

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		s.errorLog.Printf("Failed to fetch sensors with measurements: %v", err)
		return nil, err
	}
	defer rows.Close()

	type agg struct {
		item SensorWithMeasurements
	}
	sensors := make(map[string]*agg)

	for rows.Next() {
		var (
			sensorID    string
			sensorName  string
			timestamp   int64
			measurement sql.NullString
		)

		if err := rows.Scan(&sensorID, &sensorName, &timestamp, &measurement); err != nil {
			s.errorLog.Printf("Failed to scan sensor+measurement row: %v", err)
			return nil, err
		}

		entry, ok := sensors[sensorID]
		if !ok {
			entry = &agg{
				item: SensorWithMeasurements{
					SensorID:   sensorID,
					SensorName: sensorName,
					LastSeen:   time.Unix(timestamp, 0).UTC(),
				},
			}
			sensors[sensorID] = entry
		}
		if measurement.Valid {
			entry.item.Measurements = append(entry.item.Measurements, measurement.String)
		}
	}

	if err := rows.Err(); err != nil {
		s.errorLog.Printf("Row iteration error: %v", err)
		return nil, err
	}

	result := make([]SensorWithMeasurements, 0, len(sensors))
	for _, v := range sensors {
		result = append(result, v.item)
	}
	return result, nil
}
