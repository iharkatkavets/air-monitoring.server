// Package handler provides handlers for accessing API endpoints.
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sensor/cmd/api/models"
	"sensor/cmd/api/pagination"
	"sensor/cmd/api/service"
	"sensor/cmd/api/storage"
	"time"

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

func (app *MeasurementHandler) Create(w http.ResponseWriter, r *http.Request) {
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

type pageResponse struct {
	Items      []models.Measurement `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

func (app *MeasurementHandler) List(w http.ResponseWriter, r *http.Request) {
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

type SSEData struct {
	Items []models.Measurement `json:"items"`
}

func (app *MeasurementHandler) Stream(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("A client has connected")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	defer app.infoLog.Println("A client has disconnected")

	ctx := r.Context()

	lastID, err := app.storage.GetLastID()
	if err != nil {
		app.errorLog.Printf("Failed to fetch id from database %v", err.Error())
		http.Error(w, "internal erro", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			measurements, outID, err := app.storage.GetMeasurementsAfterID(ctx, lastID, 100)
			if err != nil {
				app.errorLog.Printf("Failed to fetch measurements %v", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(measurements) == 0 {
				continue
			}
			b, err := json.Marshal(SSEData{Items: measurements})
			if err != nil {
				app.errorLog.Printf("Failed to encode measurements to JSON %v", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := fmt.Fprintf(w, "event: measurements\ndata: %s\n\n", string(b)); err != nil {
				app.errorLog.Printf("Failed to write %v", err.Error())
				return
			}
			flusher.Flush()
			lastID = outID
		}
	}
}
