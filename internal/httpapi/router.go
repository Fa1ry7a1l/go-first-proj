// Package httpapi contains HTTP routing and handlers for the Gophermart API.
package httpapi

import (
	"net/http"
)

// NewRouter creates the HTTP handler tree for the service.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handlePing)
	return mux
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}
