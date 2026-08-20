package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tripleagent/server/internal/httpapi"
	"tripleagent/server/internal/room"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	rooms := room.NewRegistry()
	defer rooms.Close()

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: httpapi.NewWithOptions(rooms, httpapi.Options{
			Logger:         logger,
			AllowedOrigins: cfg.AllowedOrigins,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// net/http does not wait for hijacked WebSocket connections during
	// Shutdown. Closing the registry closes every room-owned session.
	server.RegisterOnShutdown(rooms.Close)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("Triple Agent server listening", "addr", cfg.Addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("server shutdown requested")
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("server shutdown complete")
	return nil
}
