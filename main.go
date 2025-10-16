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
	"routing-api/middleware"

	"github.com/gorilla/mux"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	retryConfig := RetryConfig{
		MaxAttempts: cfg.MaxRetries,
		Delay:       cfg.RetryDelay,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  cfg.MaxFailures,
		ResetTimeout: cfg.ResetTimeout,
	}
	handler := NewProxyHandler(cfg.ApplicationAPIs, cfg.BalancerType, retryConfig, circuitConfig)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handler.StartHealthChecks(ctx, cfg.ApplicationAPIs, cfg.HealthCheckInterval)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("forced shutdown:", err)
	}
}
