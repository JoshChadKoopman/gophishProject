package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/gophish/gophish/logger"
)

// ErrMethodNotAllowed is the standard message for unsupported HTTP methods.
const ErrMethodNotAllowed = "Method not allowed"

// ErrInvalidJSON is the standard message for malformed JSON request bodies.
const ErrInvalidJSON = "Invalid JSON"

// ErrPermissionDenied is the standard message for permission-denied responses.
const ErrPermissionDenied = "Permission denied"

// JSONResponse attempts to set the status code, c, and marshal the given interface, d, into a response that
// is written to the given ResponseWriter.
func JSONResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Error(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}
