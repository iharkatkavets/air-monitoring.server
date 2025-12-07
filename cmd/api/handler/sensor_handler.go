package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sensor/cmd/api/storage"
	"time"
)

type SensorHandler struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	storage  *storage.SQLStorage
}

func NewSensorHandler(infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage) *SensorHandler {
	return &SensorHandler{
		infoLog:  infoLog,
		errorLog: errorLog,
		storage:  storage,
	}
}

type SensorResponse struct {
	SensorID   *string   `json:"sensor_id"`
	SensorName *string   `json:"sensor_name"`
	LastSeen   time.Time `json:"last_seen_time"`
}

func (h *SensorHandler) Get(w http.ResponseWriter, r *http.Request) {
	response := []SensorResponse{}

	sensors, err := h.storage.GetAllSensors(r.Context())
	if err != nil {
		h.errorLog.Println("Failed to fetch sensors")
		http.Error(w, "Failed to fetch sensors", http.StatusInternalServerError)
		return
	}
	for _, sensor := range sensors {
		response = append(response, SensorResponse{
			SensorID:   &sensor.SensorID,
			SensorName: &sensor.SensorName,
			LastSeen:   sensor.LastSeen,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.errorLog.Println("Failed to return sensors list")
	}
}
