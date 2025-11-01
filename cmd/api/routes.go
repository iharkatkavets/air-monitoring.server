package main

import (
	"net/http"
	"sensor/cmd/api/handler"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	measurementHandler := handler.NewMeasurementHandler(app.service, app.infoLog, app.errorLog, app.storage)

	mux.Get("/health", handler.HealthCheck)
	mux.Route("/api/measurements", func(r chi.Router) {
		r.Get("/", measurementHandler.List)
		r.Post("/", measurementHandler.Create)
		r.Get("/stream", measurementHandler.Stream)
	})

	return mux
}
