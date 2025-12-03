// Package models provides models for input models
package models

import (
	"errors"
	"fmt"
	"time"
)

type MeasurementValue struct {
	Measurement string  `json:"measurement"`
	Parameter   *string `json:"parameter,omitempty"`
	Value       float64 `json:"value"`
	Unit        *string `json:"unit,omitempty"`
}

type CreateMeasurementReq struct {
	Sensor    *string   `json:"sensor"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Single measurement
	Measurement *string `json:"measurement,omitempty"`
	Parameter   *string `json:"parameter,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Unit        *string `json:"unit,omitempty"`
	// Multi measurements
	Measurements []MeasurementValue `json:"measurements,omitempty"`
}

var ErrBadPayload = errors.New("invalid measurement payload")

func (r *CreateMeasurementReq) ExtractValues() ([]MeasurementValue, error) {
	if len(r.Measurements) > 0 {
		if r.Measurement != nil || r.Parameter != nil || r.Unit != nil {
			return nil, fmt.Errorf("%w: cannot mix single fields with 'measurements' array", ErrBadPayload)
		}
		for i, m := range r.Measurements {
			if len(m.Measurement) == 0 {
				return nil, fmt.Errorf("%w: measurements[%d].measurement is required", ErrBadPayload, i)
			}
		}
		return r.Measurements, nil
	}

	if r.Measurement == nil {
		return nil, fmt.Errorf("%w: measurement, parameter, value and unit are required for single payload", ErrBadPayload)
	}

	return []MeasurementValue{
		{
			Measurement: *r.Measurement,
			Parameter:   r.Parameter,
			Value:       r.Value,
			Unit:        r.Unit,
		},
	}, nil
}
