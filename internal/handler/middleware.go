package handler

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf(
			"INFO: started %s %s",
			r.Method,
			r.RequestURI,
		)

		next.ServeHTTP(w, r)

		log.Printf(
			"INFO: completed %s %s in %v",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}
