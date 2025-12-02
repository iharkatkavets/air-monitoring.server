// Package models provides models for input models
package models

import (
	"time"
)

type MeasurementSSE struct {
	SensorID    *string   `json:"sensor_id,omitempty"`
	Measurement string    `json:"measurement"`
	Parameter   *string   `json:"parameter,omitempty"`
	Value       float64   `json:"value"`
	Unit        *string   `json:"unit,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
