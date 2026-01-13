package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subscription-service/internal/config"
	"subscription-service/internal/db"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	_ "subscription-service/docs"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Subscription Service API
// @version 1.0
// @description API Server for Subscription Management.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/gsrlabs
// @contact.email gsrnode@mail.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8090
// @BasePath /
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("INFO: starting application")

	cfg, err := config.Load("config/config.yml")
	if err != nil {
		log.Fatalf("ERROR: load config: %v", err)
	}

	// 1️⃣ DB
	database, err := db.Connect(ctx, cfg)
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

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Post("/subscriptions", subHandler.Create)
	r.Get("/subscriptions/{id}", subHandler.Get)
	r.Put("/subscriptions/{id}", subHandler.Update)
	r.Delete("/subscriptions/{id}", subHandler.Delete)
	r.Get("/subscriptions", subHandler.List)
	r.Get("/subscriptions/summary", subHandler.Summary)

	// 6️⃣ HTTP server

	server := &http.Server{
		Addr:    ":" + cfg.App.Port,
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
