package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sensor/cmd/api/db"
	"sensor/cmd/api/service"
	"sensor/cmd/api/settings"
	"sensor/cmd/api/storage"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const version = "1.0.0"

type config struct {
	port int
	db   string
	env  string
}

type application struct {
	config   config
	infoLog  *log.Logger
	errorLog *log.Logger
	version  string
	service  *service.MeasurementService
	storage  *storage.SQLStorage
	settings *settings.SettingsCache
}

func (app *application) serve(ctx context.Context, shutdownTimeout time.Duration) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      0, // for SSE/Long Pooling
	}

	storageCleaner := storage.NewStorageCleaner(app.storage, app.infoLog, app.errorLog)
	storageCleaner.StartCleanupJob(ctx, time.Second*15)

	serverErr := make(chan error, 1)

	go func() {
		app.infoLog.Printf("Starting %s server on %s (v%s)", app.config.env, srv.Addr, app.version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-stop:
		app.infoLog.Println("Shutdown signal received, shutting down server...")
	case <-ctx.Done():
		app.infoLog.Println("Context cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		if closeErr := srv.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	app.infoLog.Println("Server shutdown gracefully")

	return nil
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4001, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application environment {development|production}")
	flag.StringVar(&cfg.db, "db", "api.db", "The path to db file")

	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	database, err := db.NewDB(cfg.db)
	if err != nil {
		errorLog.Println(err)
		log.Fatal(err)
	}
	defer database.Close()

	store := storage.NewSQLStorage(database)
	ctx := context.Background()
	if err := store.InitDB(ctx); err != nil {
		errorLog.Println(err)
		log.Fatal(err)
	}

	m := storage.NewMigrations(database, infoLog)
	if err := m.Run(); err != nil {
		errorLog.Println(err)
		log.Fatal(err)
	}

	svc := service.NewMeasurementService(store)
	var settingsCache settings.SettingsCache
	InitSettings(store, &settingsCache)

	app := &application{
		config:   cfg,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
		service:  svc,
		storage:  store,
		settings: &settingsCache,
	}

	shutdownTimeout := time.Second * 3
	if err := app.serve(ctx, shutdownTimeout); err != nil {
		app.errorLog.Println(err)
		errorLog.Fatal(err)
	}

	if err := database.Close(); err != nil {
		errorLog.Printf("db close error: %v", err)
	}

	infoLog.Println("Shutdown complete")
}

func InitSettings(storage *storage.SQLStorage, obj *settings.SettingsCache) error {
	valStr, ok := settings.DefaultSettings[settings.SettingKeyStoreInterval]
	if !ok {
		return errors.New("no default value for ")
	}
	valDuration, err := time.ParseDuration(valStr)
	if err != nil {
		return err
	}
	obj.SetStoreInteval(valDuration)
	return nil
}
