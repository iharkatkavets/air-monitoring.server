// Package handler provides handlers for accessing API endpoints.
package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type SlowHandler struct {
	infoLog *log.Logger
}

func NewSlowHandler(infoLog *log.Logger) *SlowHandler {
	return &SlowHandler{infoLog: infoLog}
}

func (h *SlowHandler) SlowResponse(w http.ResponseWriter, r *http.Request) {
	h.infoLog.Println("Slow response started")
	time.Sleep(time.Second * 8)
	end := time.Now()
	fmt.Fprintf(w, "Slow response completed at %v", end)
	h.infoLog.Printf("Slow response completed at %v", end)
}
