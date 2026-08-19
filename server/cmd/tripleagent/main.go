package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"tripleagent/server/internal/httpapi"
	"tripleagent/server/internal/room"
)

func main() {
	rooms := room.NewRegistry()
	defer rooms.Close()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.New(rooms),
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
