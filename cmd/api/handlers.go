package main

import (
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"sensor/cmd/api/service"
)

func (app *application) CreateMeasurement(w http.ResponseWriter, r *http.Request) {
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

func (app *application) GetAllMeasurements(w http.ResponseWriter, r *http.Request) {
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
