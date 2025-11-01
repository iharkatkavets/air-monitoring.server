package main

import (
	"context"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sensor/cmd/api/db"
	"sensor/cmd/api/service"
	"sensor/cmd/api/storage"
	"syscall"
	"time"
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
}

func (app *application) serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	// Server errors channel
	errCh := make(chan error, 1)

	// Start server
	go func() {
		app.infoLog.Printf("Starting %s server on %s (v%s)", app.config.env, srv.Addr, app.version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Wait for ctx cancellation or server error
	select {
	case <-ctx.Done():
		// Begin graceful shutdown
		app.infoLog.Println("Shutdown signal received, shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			// If graceful shutdown times out, force close
			app.errorLog.Printf("graceful shutdown failed: %v; forcing close", err)
			_ = srv.Close()
		}

		// Ensure the serve goroutine returns
		return <-errCh

	case err := <-errCh:
		// Server crashed on startup or runtime
		return err
	}
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
	if err := store.InitDB(); err != nil {
		errorLog.Println(err)
		log.Fatal(err)
	}

	m := storage.NewMigrations(database, infoLog)
	if err := m.Run(); err != nil {
		errorLog.Println(err)
		log.Fatal(err)
	}

	svc := service.NewMeasurementService(store)

	app := &application{
		config:   cfg,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
		service:  svc,
		storage:  store,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.serve(ctx); err != nil {
		app.errorLog.Println(err)
		errorLog.Fatal(err)
	}

	if err := database.Close(); err != nil {
		errorLog.Printf("db close error: %v", err)
	}

	infoLog.Println("Shutdown complete")
}
