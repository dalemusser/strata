package main

import (
	"log"
	"net"
	"net/http"
)

// Check if the host is a non-routable IP or localhost and switch to HTTP automatically
func startServer(config *Config, logger *log.Logger) *http.Server {
	logger.Printf("Starting server at %s:%s...", config.Host, config.Port)

	// Check if the host is a numeric IP or non-routable address
	if isLocalNetwork(config.Host) {
		logger.Println("Non-routable or numeric IP detected. Starting HTTP server...")
		config.UseTLS = false
	}

	// Initialize the router
	r := setupRouter()

	server := &http.Server{
		Addr:    config.Host + ":" + config.Port,
		Handler: r,
	}

	// Start the server with or without TLS
	if config.UseTLS {
		logger.Println("Starting HTTPS server...")
		tlsConfig, err := getTLSConfig(config)
		if err != nil {
			logger.Fatalf("Failed to configure TLS: %v", err)
			return nil
		}
		server.TLSConfig = tlsConfig

		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil {
				logger.Fatalf("Could not start HTTPS server: %v", err)
			}
		}()
	} else {
		logger.Println("Starting HTTP server...")
		go func() {
			if err := http.ListenAndServe(config.Host+":"+config.Port, r); err != nil {
				logger.Fatalf("Could not start HTTP server: %v", err)
			}
		}()
	}

	return server
}

// isLocalNetwork checks if the provided host is a numeric IP or non-routable IP address
func isLocalNetwork(host string) bool {
	// Check if it's localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Check if it's a numeric IP address
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check if it's a non-routable IP (e.g., 192.168.x.x, 10.x.x.x, etc.)
	if ip.IsPrivate() || ip.IsLoopback() {
		return true
	}

	// Assume it's a public routable address
	return false
}
