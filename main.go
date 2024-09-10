package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	config := LoadConfig()

	// Initialize logger
	logger := NewLogger(config)

	// Start the server based on the configuration
	server := startServer(config, logger)

	// Handle graceful shutdown
	gracefulShutdown(server, logger, config)
}

func gracefulShutdown(server *http.Server, logger *log.Logger, config *Config) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited")
}
