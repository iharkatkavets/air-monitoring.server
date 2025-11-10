package main

import (
	"net/http"
	"sensor/cmd/api/handler"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	measurementHandler := handler.NewMeasurementHandler(app.service, app.infoLog, app.errorLog, app.storage)
	slowHandler := handler.NewSlowHandler(app.infoLog)
	settingsHandler := handler.NewSettingsHandler(app.infoLog, app.errorLog, app.storage)

	mux.Get("/health", handler.HealthCheck)
	mux.Get("/slow", slowHandler.SlowResponse)
	mux.Route("/api/measurements", func(r chi.Router) {
		r.Get("/", measurementHandler.List)
		r.Post("/", measurementHandler.Create)
		r.Get("/stream", measurementHandler.Stream)
	})
	mux.Route("/api/settings", func(r chi.Router) {
		r.Get("/", settingsHandler.GetAllSettings)
		r.Get("/{key}", settingsHandler.GetSetting)
		r.Post("/{key}", settingsHandler.UpdateSetting)
	})

	return mux
}
