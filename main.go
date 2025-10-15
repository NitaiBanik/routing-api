package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"routing-api/config"
	"routing-api/handlers"
	"routing-api/middleware"

	"github.com/gorilla/mux"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting routing-api on port %s", cfg.Port)

	handler := handlers.NewHandler(cfg.ApplicationAPIs)

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware())

	router.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	router.PathPrefix("/").HandlerFunc(handler.ProxyRequest)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		log.Printf("APIs: %v", cfg.ApplicationAPIs)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("forced shutdown:", err)
	}

	log.Println("exited")
}
