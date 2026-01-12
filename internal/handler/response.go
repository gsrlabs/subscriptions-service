package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse represents the standard JSON structure for returning API errors.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON sends a JSON response with a specific HTTP status code and marshals the provided payload.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("ERROR: failed to write response: %v", err)
		}
	}
}

// writeError logs the error message and sends a standardized JSON error response to the client.
func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("ERROR: %s", msg)
	writeJSON(w, status, errorResponse{Error: msg})
}
