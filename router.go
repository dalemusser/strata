package main

import (
	"net/http"
	"os"
)

// setupRouter initializes routes and middleware for the application
func setupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", handleCompressedResponse(handleIndex(fs)))

	// API routes
	mux.Handle("/api/hello", handleCompressedResponse(http.HandlerFunc(helloHandler)))

	// Authentication routes
	mux.Handle("/login", handleCompressedResponse(http.HandlerFunc(loginHandler)))
	mux.Handle("/authenticate", handleCompressedResponse(http.HandlerFunc(authenticate)))
	mux.Handle("/auth/callback", handleCompressedResponse(http.HandlerFunc(callbackHandler)))

	// Dashboard route (secured)
	mux.Handle("/dashboard", handleCompressedResponse(http.HandlerFunc(dashboardHandler)))

	return mux
}

// handleIndex dynamically serves index.html if it exists, otherwise serves a login page
func handleIndex(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("./static/index.html"); err == nil {
			// Serve index.html if it exists
			next.ServeHTTP(w, r)
		} else {
			// Serve login page if index.html doesn't exist
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

// helloHandler is a basic API handler
func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, this is a compressed API response"))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Render the login page (assumed to exist in your templates)
	renderTemplate(w, "login.html", nil)
}
