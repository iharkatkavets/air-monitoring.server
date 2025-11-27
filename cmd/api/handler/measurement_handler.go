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

type SSEClient struct {
	ch   chan []models.Measurement
	done chan struct{}
}

type SSEBroker struct {
	Notifier       chan []models.Measurement
	newClients     chan *SSEClient
	closingClients chan *SSEClient
	clients        map[*SSEClient]struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		Notifier:       make(chan []models.Measurement, 64),
		newClients:     make(chan *SSEClient),
		closingClients: make(chan *SSEClient),
		clients:        make(map[*SSEClient]struct{}),
	}
}

func (b *SSEBroker) addClient() *SSEClient {
	c := &SSEClient{ch: make(chan []models.Measurement, 32), done: make(chan struct{})}
	b.newClients <- c
	return c
}

func (b *SSEBroker) listen() {
	for {
		select {
		case s := <-b.newClients:
			b.clients[s] = struct{}{}
			log.Printf("Client added. %d registered clients", len(b.clients))

		case s := <-b.closingClients:
			delete(b.clients, s)
			close(s.ch)
			select {
			case s.done <- struct{}{}:
			default:
			}
			log.Printf("Removed client. %d registered clients", len(b.clients))

		case event := <-b.Notifier:
			for c := range b.clients {
				select {
				case c.ch <- event:
				default:
					delete(b.clients, c)
					close(c.ch)
					select {
					case c.done <- struct{}{}:
					default:
					}
					log.Printf("Dropped slow client. %d registered clients", len(b.clients))
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
	broker         *SSEBroker
}

func NewMeasurementHandler(infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage, settings *settings.SettingsCache) *MeasurementHandler {
	prevRecordTime := time.Now().Add(-settings.GetStoreInterval())
	broker := NewSSEBroker()
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
	h.broker.Notifier <- response

	if timeSincePrevAdd >= storeInterval {
		h.prevRecordTime = currTimestamp
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

type sseData struct {
	Items []models.Measurement `json:"items"`
}

func (h *MeasurementHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	c := h.broker.addClient()
	h.infoLog.Println("A client has connected")
	defer func() {
		h.broker.closingClients <- c
		h.infoLog.Println("A client has disconnected")
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			h.infoLog.Println("ctx.Done()")
			return

		case <-c.done:
			h.infoLog.Println("c.done")
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
