package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// getTLSConfig returns a TLS configuration based on environment (Let's Encrypt or manual)
func getTLSConfig(config *Config) (*tls.Config, error) {
	if config.UseLetsEncrypt {
		certManager := autocert.Manager{
			Cache:      autocert.DirCache("certs"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(config.Host),
		}
		return &tls.Config{
			GetCertificate: certManager.GetCertificate,
		}, nil
	}

	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

// startLetsEncryptServer sets up a server with Let's Encrypt
func startLetsEncryptServer(handler http.Handler, config *Config) error {
	certManager := autocert.Manager{
		Cache:      autocert.DirCache("certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(config.Host),
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: handler,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go func() {
		log.Println("Starting HTTP server for Let's Encrypt on port 80")
		err := http.ListenAndServe(":80", certManager.HTTPHandler(nil))
		if err != nil {
			log.Fatalf("HTTP server for Let's Encrypt failed: %v", err)
		}
	}()

	log.Println("Starting HTTPS server with Let's Encrypt")
	return server.ListenAndServeTLS("", "")
}

// startManualTLS starts the server using provided certificates
func startManualTLS(config *Config, handler http.Handler) error {
	if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
		return err
	}

	server := &http.Server{
		Addr:         ":443",
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Starting HTTPS server with provided certificates")
	return server.ListenAndServeTLS(config.CertFile, config.KeyFile)
}

// startHTTPRedirect starts an HTTP server on port 80 that redirects all traffic to HTTPS
func startHTTPRedirect() {
	log.Println("Starting HTTP redirection server on port 80")
	err := http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect HTTP to HTTPS
		http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
	}))

	if err != nil {
		log.Fatalf("HTTP redirection server failed: %v", err)
	}
}
