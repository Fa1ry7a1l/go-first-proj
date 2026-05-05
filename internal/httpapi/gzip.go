package httpapi

import (
	"compress/gzip"
	"net/http"
	"strings"
)

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = reader
		}

		next.ServeHTTP(w, r)
	})
}
