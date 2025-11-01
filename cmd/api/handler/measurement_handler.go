// Package handler provides handlers for accessing API endpoints.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sensor/cmd/api/models"
	"sensor/cmd/api/pagination"
	"sensor/cmd/api/service"
	"sensor/cmd/api/storage"

	"strconv"
)

type MeasurementHandler struct {
	service  *service.MeasurementService
	infoLog  *log.Logger
	errorLog *log.Logger
	storage  *storage.SQLStorage
}

func NewMeasurementHandler(service *service.MeasurementService, infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage) *MeasurementHandler {
	return &MeasurementHandler{service: service, infoLog: infoLog, errorLog: errorLog, storage: storage}
}

func (app *MeasurementHandler) CreateMeasurement(w http.ResponseWriter, r *http.Request) {
	var req service.CreateMeasurementReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	response, err := app.service.CreateMeasurement(&req)
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
func (app *MeasurementHandler) GetAllMeasurements(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]string)
	queryParams := r.URL.Query()
	for key, values := range queryParams {
		if len(values) > 0 && len(key) > 7 && key[:7] == "filter[" {
			field := key[7 : len(key)-1]
			filters[field] = values[0]
		}
	}
	readings, err := app.service.GetAllMeasurements(filters)
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(readings)
	if err != nil {
		app.errorLog.Println("Failed to encode JSON:", err)
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
}
*/

type pageResponse struct {
	Items      []models.Measurement `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

func (app *MeasurementHandler) GetAllMeasurements(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var cur *pagination.MeasurementCursor
	if tok := r.URL.Query().Get("cursor"); tok != "" {
		c, err := pagination.Decode(tok)
		if err != nil {
			http.Error(w, "bad cursor", http.StatusBadRequest)
			return
		}
		cur = &c
	}

	// fetch
	items, err := app.storage.GetMeasurementsPage(limit, cur)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = pagination.Encode(pagination.MeasurementCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pageResponse{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
