package main

import (
	"net/http"
	"sensor/cmd/api/handler"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	measurementHandler := handler.NewMeasurementHandler(app.infoLog, app.errorLog, app.storage, app.settings)
	slowHandler := handler.NewSlowHandler(app.infoLog)
	settingsHandler := handler.NewSettingsHandler(app.infoLog, app.errorLog, app.storage, app.settings)

	mux.Get("/health", handler.HealthCheck)
	mux.Get("/slow", slowHandler.MakeItSlow)
	mux.Get("/slow/{seconds}", slowHandler.MakeItSlow)
	mux.Route("/api/measurements", func(r chi.Router) {
		r.Get("/{sensor}", measurementHandler.List)
		r.Post("/{sensor}", measurementHandler.Create)
		r.Get("/{sensor}/stream", measurementHandler.Stream)
	})
	mux.Route("/api/settings", func(r chi.Router) {
		r.Get("/", settingsHandler.ListSettings)
		r.Get("/{key}", settingsHandler.GetSetting)
		r.Post("/{key}", settingsHandler.UpdateSetting)
	})
	return mux
}
