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
