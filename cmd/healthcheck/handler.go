package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type RequestHandler struct {
	health Health
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		payload, err := h.health.GetHealthResult()
		log.Println("payload:", payload)
		if err != nil {
			h.respond(w, http.StatusInternalServerError, []byte{})
			return
		}
		payloadJSON, _ := json.Marshal(payload)
		status := http.StatusOK
		if !payload.Healthy {
			status = http.StatusServiceUnavailable
		}
		h.respond(w, status, payloadJSON)
	case "/ping":
		payloadJSON, _ := json.Marshal(map[string]string{"status": "UP"})
		h.respond(w, http.StatusOK, payloadJSON)
	default:
		h.respond(w, http.StatusNotFound, []byte{})
	}
}

func (h *RequestHandler) respond(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(payload)
}
