package models

import "time"

type Measurement struct {
	ID        int64     `json:"id"`
	Sensor    string    `json:"sensor"`
	Parameter *string   `json:"parameter,omitempty"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}
