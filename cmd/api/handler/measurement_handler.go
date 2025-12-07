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
	"sync"
	"time"

	"strconv"

	"github.com/go-chi/chi/v5"
)

type SSEClient struct {
	sensorID string
	ch       chan []models.MeasurementSSE
	done     chan struct{}
}

type MeasurementEvent struct {
	sensorID     string
	measurements []models.MeasurementSSE
}

type SSEBroker struct {
	Notifier       chan MeasurementEvent
	newClients     chan *SSEClient
	closingClients chan *SSEClient
	clients        map[*SSEClient]struct{}
	infoLog        *log.Logger
	errorLog       *log.Logger
}

func NewSSEBroker(infoLog *log.Logger, errorLog *log.Logger) *SSEBroker {
	return &SSEBroker{
		Notifier:       make(chan MeasurementEvent, 64),
		newClients:     make(chan *SSEClient),
		closingClients: make(chan *SSEClient),
		clients:        make(map[*SSEClient]struct{}),
		infoLog:        infoLog,
		errorLog:       errorLog,
	}
}

func (b *SSEBroker) addClient(sensorID string) *SSEClient {
	c := &SSEClient{sensorID: sensorID, ch: make(chan []models.MeasurementSSE, 32), done: make(chan struct{})}
	b.newClients <- c
	return c
}

func (b *SSEBroker) listen() {
	for {
		select {
		case s := <-b.newClients:
			b.clients[s] = struct{}{}
			b.infoLog.Printf("Client added. %d registered clients", len(b.clients))

		case s := <-b.closingClients:
			delete(b.clients, s)
			close(s.ch)
			select {
			case s.done <- struct{}{}:
			default:
			}
			b.infoLog.Printf("Removed client. %d registered clients", len(b.clients))

		case event := <-b.Notifier:
			for c := range b.clients {
				if c.sensorID != event.sensorID {
					continue
				}
				select {
				case c.ch <- event.measurements:
				default:
					delete(b.clients, c)
					close(c.ch)
					select {
					case c.done <- struct{}{}:
					default:
					}
					b.infoLog.Printf("Dropped slow client. %d registered clients", len(b.clients))
				}
			}
		}
	}
}

type MeasurementHandler struct {
	infoLog        *log.Logger
	errorLog       *log.Logger
	storage        *storage.SQLStorage
	settings       *settings.SettingsCache
	prevRecordTime time.Time
	prevRecordMu   sync.Mutex
	broker         *SSEBroker
}

func NewMeasurementHandler(infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage, settings *settings.SettingsCache) *MeasurementHandler {
	prevRecordTime := time.Now().Add(-settings.GetStoreInterval())
	broker := NewSSEBroker(infoLog, errorLog)
	go broker.listen()
	return &MeasurementHandler{
		infoLog:        infoLog,
		errorLog:       errorLog,
		storage:        storage,
		settings:       settings,
		prevRecordTime: prevRecordTime,
		broker:         broker,
	}
}

const pathParamSensorID = "sensor_id"

func (h *MeasurementHandler) Create(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, pathParamSensorID)

	var req models.CreateMeasurementReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorLog.Println(err)
		// improve error response here
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	values, err := req.ExtractValues()
	if err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	currTimestamp := time.Now().UTC()
	storeInterval := h.settings.GetStoreInterval()
	ts := req.Timestamp
	if ts.IsZero() {
		ts = currTimestamp
	}
	h.prevRecordMu.Lock()
	timeSincePrevAdd := currTimestamp.Sub(h.prevRecordTime)
	shouldStore := timeSincePrevAdd >= storeInterval
	if shouldStore {
		h.prevRecordTime = currTimestamp
	}
	h.prevRecordMu.Unlock()

	sseResponse := make([]models.MeasurementSSE, 0, len(values))
	httpResponse := make([]storage.MeasurementRecord, 0, len(values))
	for _, v := range values {
		var m models.MeasurementSSE
		m.SensorID = &sensorID
		m.SensorName = &req.SensorName
		m.Measurement = v.Measurement
		m.Parameter = v.Parameter
		m.Value = v.Value
		m.Unit = v.Unit
		m.Timestamp = ts
		sseResponse = append(sseResponse, m)

		h.storage.UpsertSensor(ctx, &sensorID, &req.SensorName, ts)

		if shouldStore {
			record, err := h.storage.CreateMeasurement(ctx, &sensorID, &req.SensorName, &v, currTimestamp)
			if err != nil {
				h.errorLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httpResponse = append(httpResponse, record)
		}
	}
	h.broker.Notifier <- MeasurementEvent{sensorID: sensorID, measurements: sseResponse}

	if !shouldStore {
		remaining := storeInterval - timeSincePrevAdd
		h.infoLog.Printf(
			"Skip storing. remaining=%s elapsed=%s interval=%s",
			remaining.Round(time.Second),
			timeSincePrevAdd.Round(time.Second),
			storeInterval.Round(time.Second),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"skipped","reason":"interval_not_reached"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(httpResponse); err != nil {
		h.errorLog.Println(err)
	}
}

type pageResponse struct {
	Items      []storage.MeasurementRecord `json:"items"`
	NextCursor string                      `json:"next_cursor,omitempty"`
	HasMore    bool                        `json:"has_more"`
}

func (h *MeasurementHandler) Get(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, pathParamSensorID)
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

	items, err := h.storage.GetMeasurementsPage(sensorID, limit, cur)
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

type sseData struct {
	Items []models.MeasurementSSE `json:"items"`
}

func (h *MeasurementHandler) Stream(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, pathParamSensorID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	c := h.broker.addClient(sensorID)
	defer func() {
		h.broker.closingClients <- c
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			h.infoLog.Println("ctx.Done()")
			return

		case <-c.done:
			h.infoLog.Println("client done")
			return

		case measurements, ok := <-c.ch:
			if !ok {
				h.infoLog.Println("client channel closed")
				return
			}
			if len(measurements) == 0 {
				continue
			}
			b, err := json.Marshal(sseData{Items: measurements})
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
		}
	}
}
