package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sensor/cmd/api/settings"
	"sensor/cmd/api/storage"
	"time"

	"github.com/go-chi/chi/v5"
)

type SettingInputValue struct {
	Value     string `json:"value"`
	Parameter string `json:"parameter,omitempty"`
}

type SettingResponseValue struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Parameter *string   `json:"parameter,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type allSettingsResponse struct {
	Items []SettingResponseValue `json:"settings"`
}

type SettingsHandler struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	storage  *storage.SQLStorage
}

func NewSettingsHandler(infoLog *log.Logger, errorLog *log.Logger, storage *storage.SQLStorage) *SettingsHandler {
	return &SettingsHandler{infoLog: infoLog, errorLog: errorLog, storage: storage}
}

func (h *SettingsHandler) GetAllSettings(w http.ResponseWriter, r *http.Request) {
	items, err := h.storage.GetAllSettings(r.Context())
	if err != nil {
		h.errorLog.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	out := make([]SettingResponseValue, 0, len(items))
	for _, item := range items {
		out = append(out, SettingResponseValue{
			Key:       item.Key,
			Value:     item.Value,
			UpdatedAt: item.UpdatedAt})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allSettingsResponse{Items: out}); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) GetSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	item, err := h.storage.GetSetting(r.Context(), key)
	if err != nil {
		h.errorLog.Printf("getMaxAge failed with error %v\n", err)
		http.Error(w, "database error", http.StatusInternalServerError)
	}
	resp := SettingResponseValue{Key: key}

	if item != nil {
		resp.Value = item.Value
		resp.UpdatedAt = item.UpdatedAt
	} else {
		if def, ok := settings.DefaultSettings[key]; ok {
			resp.Value = def
			resp.UpdatedAt = time.Now()
		} else {
			h.errorLog.Printf("no default value for the key %s \n", key)
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	defer r.Body.Close()

	var body SettingInputValue
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if key == settings.SettingKeyMaxAge {
		h.updateMaxAge(w, r, key, &body)
	} else if key == settings.SettingKeyStoreInterval {

	}
}

func (h *SettingsHandler) getMaxAge(w http.ResponseWriter, r *http.Request, key string) {
	item, err := h.storage.GetSetting(r.Context(), key)
	if err != nil {
		h.errorLog.Printf("getMaxAge failed with error %v\n", err)
		http.Error(w, "database error", http.StatusInternalServerError)
	}
	resp := SettingResponseValue{Key: key}

	if item != nil {
		resp.Value = item.Value
		resp.UpdatedAt = item.UpdatedAt
	} else {
		if def, ok := settings.DefaultSettings[key]; ok {
			resp.Value = def
			resp.UpdatedAt = time.Now()
		} else {
			h.errorLog.Printf("no default value for the key %s \n", key)
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) getStoreInterval(w http.ResponseWriter, r *http.Request, key string) {
	item, err := h.storage.GetSetting(r.Context(), key)
	if err != nil {
		h.errorLog.Printf("getMaxAge failed with error %v\n", err)
		http.Error(w, "database error", http.StatusInternalServerError)
	}
	resp := SettingResponseValue{Key: key}

	if item != nil {
		resp.Value = item.Value
		resp.UpdatedAt = item.UpdatedAt
	} else {
		if def, ok := settings.DefaultSettings[key]; ok {
			resp.Value = def
			resp.UpdatedAt = time.Now()
		} else {
			h.errorLog.Printf("no default value for the key %s \n", key)
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) updateMaxAge(w http.ResponseWriter, r *http.Request, key string, s *SettingInputValue) {
	item, err := h.storage.UpsertSetting(r.Context(), key, s.Value)
	if err != nil {
		h.errorLog.Println(err)
		http.Error(w, "Internal Server error", http.StatusInternalServerError)
		return
	}
	resp := SettingResponseValue{Key: key}

	if item != nil {
		resp.Value = item.Value
		resp.UpdatedAt = item.UpdatedAt
	} else {
		if def, ok := settings.DefaultSettings[key]; ok {
			resp.Value = def
			resp.UpdatedAt = time.Now()
		} else {
			h.errorLog.Printf("no default value for the key %s \n", key)
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
