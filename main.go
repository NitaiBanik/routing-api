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
	cfg := config.Load()

	handler := NewProxyHandler(cfg.ApplicationAPIs, cfg.BalancerType)

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware())

	router.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	router.PathPrefix("/").HandlerFunc(handler.ProxyRequest)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("forced shutdown:", err)
	}
}
