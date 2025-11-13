// Package handler provides handlers for accessing API endpoints.
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sensor/cmd/api/models"
	"sensor/cmd/api/pagination"
	"sensor/cmd/api/settings"
	"sensor/cmd/api/storage"
	"time"

	"strconv"
)

type MeasurementHandler struct {
	infoLog        *log.Logger
	errorLog       *log.Logger
	storage        *storage.SQLStorage
	settings       *settings.SettingsCache
	prevRecordTime time.Time
}

func NewMeasurementHandler(infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage, settings *settings.SettingsCache) *MeasurementHandler {
	prevRecordTime := time.Now().Add(-settings.GetStoreInterval())
	return &MeasurementHandler{infoLog: infoLog, errorLog: errorLog, storage: storage, settings: settings, prevRecordTime: prevRecordTime}
}

type CreateMeasurementReq struct {
	Timestamp time.Time          `json:"timestamp"`
	Values    []MeasurementValue `json:"values"`
}

type MeasurementValue struct {
	Sensor    string  `json:"sensor"`
	Parameter *string `json:"parameter,omitempty"` // nil for particle_size
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

func (h *MeasurementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateMeasurementReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currTimestamp := time.Now().UTC()
	timeSincePrevAdd := currTimestamp.Sub(h.prevRecordTime)
	storeInterval := h.settings.GetStoreInterval()
	if timeSincePrevAdd >= storeInterval {
		h.prevRecordTime = currTimestamp
		response := make([]models.Measurement, 0, len(req.Values))
		for _, v := range req.Values {
			var m models.Measurement
			ts := req.Timestamp
			if ts.IsZero() {
				ts = currTimestamp
			}
			m.Timestamp = ts
			m.Parameter = v.Parameter
			m.Value = v.Value
			m.Unit = v.Unit
			m.Sensor = v.Sensor
			m.CreatedAt = currTimestamp

			if err := h.storage.CreateMeasurement(&m); err != nil {
				h.errorLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			response = append(response, m)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.errorLog.Println(err)
		}
	} else {
		remaining := storeInterval - timeSincePrevAdd
		h.infoLog.Printf("Skip storing. remaining=%s elapsed=%s interval=%s", remaining.Round(time.Second), timeSincePrevAdd.Round(time.Second), storeInterval.Round(time.Second))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		http.NoBody.WriteTo(w)
	}
}

type pageResponse struct {
	Items      []models.Measurement `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

func (h *MeasurementHandler) List(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.storage.GetMeasurementsPage(limit, cur)
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

func (h *MeasurementHandler) Stream(w http.ResponseWriter, r *http.Request) {
	h.infoLog.Println("A client has connected")

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

	defer h.infoLog.Println("A client has disconnected")

	ctx := r.Context()

	lastID, err := h.storage.GetLastID()
	if err != nil {
		h.errorLog.Printf("Failed to fetch id from database %v", err.Error())
		http.Error(w, "internal erro", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			measurements, outID, err := h.storage.GetMeasurementsAfterID(ctx, lastID, 100)
			if err != nil {
				h.errorLog.Printf("Failed to fetch measurements %v", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(measurements) == 0 {
				continue
			}
			b, err := json.Marshal(SSEData{Items: measurements})
			if err != nil {
				h.errorLog.Printf("Failed to encode measurements to JSON %v", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := fmt.Fprintf(w, "event: measurements\ndata: %s\n\n", string(b)); err != nil {
				h.errorLog.Printf("Failed to write %v", err.Error())
				return
			}
			flusher.Flush()
			lastID = outID
		}
	}
}
