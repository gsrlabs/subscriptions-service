package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"subscription-service/internal/db"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("INFO: starting application")

	database, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("ERROR: failed to connect to database: %v", err)
	}
	defer database.Pool.Close()

	r := chi.NewRouter()

	server := &http.Server{
		Addr:    ":" + getEnv("APP_PORT", "8080"),
		Handler: r,
	}

	go func() {
		log.Printf("INFO: HTTP server started on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ERROR: server failed: %v", err)
		}
	}()

	waitForShutdown(ctx, server)
}

func waitForShutdown(ctx context.Context, server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Printf("INFO: shutting down application")

	ctxShutdown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Printf("ERROR: server shutdown failed: %v", err)
	}

	log.Printf("INFO: application stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
