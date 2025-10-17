package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"routing-api/internal/circuit"
	"routing-api/internal/config"
	"routing-api/internal/loadbalancer"
	"routing-api/internal/logger"
	"routing-api/internal/middleware"
	"routing-api/internal/proxy"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		zap.L().Fatal("Failed to load configuration", zap.Error(err))
	}

	if err := logger.Init(cfg.LogLevel); err != nil {
		zap.L().Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer logger.Sync()

	log := logger.Global()
	log.Info("Starting routing API server",
		zap.String("port", cfg.Port),
		zap.String("environment", cfg.Environment),
		zap.Strings("servers", cfg.ApplicationAPIs),
		zap.String("log_level", cfg.LogLevel),
	)

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  cfg.MaxFailures,
		ResetTimeout: cfg.ResetTimeout,
	}

	loadBalancerFactory := loadbalancer.NewLoadBalancerFactory()
	loadBalancer := loadBalancerFactory.CreateLoadBalancer(cfg.BalancerType, cfg.ApplicationAPIs, circuitConfig, log)
	clientProvider := loadbalancer.NewLoadBalancerAdapter(loadBalancer)
	handler := proxy.NewProxyHandler(clientProvider, log)

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

	go handler.StartHealthChecks(ctx, cfg.HealthCheckInterval)

	go func() {
		log.Info("Server starting", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Forced shutdown", zap.Error(err))
	}

	log.Info("Server stopped")
}
