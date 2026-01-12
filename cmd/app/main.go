package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subscription-service/internal/db"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/go-chi/chi/v5"
)

// main is the entry point of the application. It orchestrates the initialization
// of the database, repositories, services, and HTTP handlers, and starts the server.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("INFO: starting application")

	// 1️⃣ DB
	database, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("ERROR: failed to connect to database: %v", err)
	}
	defer database.Pool.Close()

	// 2️⃣ Repository
	subRepo := repository.NewSubscriptionRepository(database.Pool)

	// 3️⃣ Service
	subService := service.NewSubscriptionService(subRepo)

	// 4️⃣ Handler
	subHandler := handler.NewSubscriptionHandler(subService)

	// 5️⃣ Router
	r := chi.NewRouter()
	r.Use(handler.LoggingMiddleware)

	r.Post("/subscriptions", subHandler.Create)
	r.Get("/subscriptions/{id}", subHandler.Get)
	r.Put("/subscriptions/{id}", subHandler.Update)
	r.Delete("/subscriptions/{id}", subHandler.Delete)
	r.Get("/subscriptions", subHandler.List)
	r.Get("/subscriptions/summary", subHandler.Summary)

	// 6️⃣ HTTP server
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

// waitForShutdown blocks the main goroutine until a termination signal (SIGINT or SIGTERM) is received,
// then gracefully shuts down the HTTP server with a 5-second timeout.
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

// getEnv retrieves the value of the environment variable named by the key
// or returns a fallback value if the variable is empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
