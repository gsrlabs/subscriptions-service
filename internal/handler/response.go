package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("ERROR: failed to write response: %v", err)
		}
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("ERROR: %s", msg)
	writeJSON(w, status, errorResponse{Error: msg})
}

