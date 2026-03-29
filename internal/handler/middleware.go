package handler

import (
	"net/http"

	"github.com/rs/cors"
)

func CORSMiddleware() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
