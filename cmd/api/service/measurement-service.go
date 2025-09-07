// Package service is a helper for fetching/storing measurements
package service

import (
	"sensor/cmd/api/models"
	"sensor/cmd/api/storage"
	"time"
)

type MeasurementService struct {
	storage storage.Storage
}

func NewMeasurementService(s storage.Storage) *MeasurementService {
	return &MeasurementService{storage: s}
}

// CreateMeasurementReq represents a single measurement request payload
type CreateMeasurementReq struct {
	Timestamp time.Time          `json:"timestamp"`
	Values    []MeasurementValue `json:"values"`
}

// MeasurementValue represents a single measurement entry
type MeasurementValue struct {
	Sensor    string  `json:"sensor"`
	Parameter *string `json:"parameter,omitempty"` // nil for particle_size
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

func (s *MeasurementService) CreateMeasurement(req *CreateMeasurementReq) ([]models.Measurement, error) {
	response := make([]models.Measurement, 0, len(req.Values))
	for _, v := range req.Values {
		var m models.Measurement
		m.Timestamp = req.Timestamp
		m.Parameter = v.Parameter
		m.Value = v.Value
		m.Unit = v.Unit
		m.Sensor = v.Sensor

		if err := s.storage.CreateMeasurement(&m); err != nil {
			return nil, err
		}
		response = append(response, m)
	}

	return response, nil
}

func (s *MeasurementService) GetAllMeasurements(filters map[string]string) ([]models.Measurement, error) {
	readings, err := s.storage.GetAllMeasurements(filters)
	if err != nil {
		return nil, err
	}
	return readings, nil
}
