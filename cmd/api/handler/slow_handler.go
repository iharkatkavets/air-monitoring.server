// Package handler provides handlers for accessing API endpoints.
package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type SlowHandler struct {
	infoLog *log.Logger
}

func NewSlowHandler(infoLog *log.Logger) *SlowHandler {
	return &SlowHandler{infoLog: infoLog}
}

func (h *SlowHandler) MakeItSlow(w http.ResponseWriter, r *http.Request) {
	secondsStr := chi.URLParam(r, "seconds")
	delay := time.Second * 8
	if len(secondsStr) != 0 {
		if s, err := strconv.ParseInt(secondsStr, 10, 32); err == nil {
			delay = time.Second * time.Duration(s)
		}
	}
	h.infoLog.Println("Slow response started")
	time.Sleep(delay)
	end := time.Now()
	h.infoLog.Printf("Slow response completed after %d at %v", delay, end)
	fmt.Fprintf(w, "Slow response completed after %d at %v", delay, end)
}
