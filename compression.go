package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/andybalholm/brotli"
)

// handleCompressedResponse applies gzip or brotli compression based on the client's Accept-Encoding header
func handleCompressedResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding := r.Header.Get("Accept-Encoding")

		// Check for pre-compressed Brotli file (.br) and serve if available
		if strings.Contains(acceptEncoding, "br") {
			brFilePath := r.URL.Path + ".br"
			if _, err := os.Stat(brFilePath); err == nil {
				w.Header().Set("Content-Encoding", "br")
				http.ServeFile(w, r, brFilePath)
				return
			}
		}

		// Check for pre-compressed Gzip file (.gz) and serve if available
		if strings.Contains(acceptEncoding, "gzip") {
			gzFilePath := r.URL.Path + ".gz"
			if _, err := os.Stat(gzFilePath); err == nil {
				w.Header().Set("Content-Encoding", "gzip")
				http.ServeFile(w, r, gzFilePath)
				return
			}
		}

		// If no pre-compressed files exist, dynamically compress the response
		if strings.Contains(acceptEncoding, "br") {
			// Apply Brotli compression
			w.Header().Set("Content-Encoding", "br")
			brWriter := brotli.NewWriter(w)
			defer brWriter.Close()

			next.ServeHTTP(brotliResponseWriter{Writer: brWriter, ResponseWriter: w}, r)
		} else if strings.Contains(acceptEncoding, "gzip") {
			// Apply Gzip compression
			w.Header().Set("Content-Encoding", "gzip")
			gzWriter := gzip.NewWriter(w)
			defer gzWriter.Close()

			next.ServeHTTP(gzipResponseWriter{Writer: gzWriter, ResponseWriter: w}, r)
		} else {
			// If no compression is supported, serve response without compression
			next.ServeHTTP(w, r)
		}
	})
}

// Define gzipResponseWriter to wrap http.ResponseWriter and apply Gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Define brotliResponseWriter to wrap http.ResponseWriter and apply Brotli compression
type brotliResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w brotliResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
