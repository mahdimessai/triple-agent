package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/app"
	"tripleagent/server/internal/room"
	"tripleagent/server/internal/transport"
)

func main() {
	admissions := admission.NewStore()
	rooms := room.NewManager()
	lobbies := app.NewLobbies(admissions, rooms)
	sessions := app.NewSessions(rooms)

	handler := transport.NewHandlerWithServices(lobbies, sessions)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Triple Agent server listening on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
