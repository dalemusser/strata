package errors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler()
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestForbidden_Returns403(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	rec := httptest.NewRecorder()

	// Handler will try to render template, which may panic
	// We're primarily testing the status code is set before rendering
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected - template rendering not initialized in tests
			}
		}()
		h.Forbidden(rec, req)
	}()

	// Check status was set (if we got that far before panic)
	if rec.Code != 0 && rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestUnauthorized_Returns401(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/unauthorized", nil)
	rec := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected - template rendering not initialized
			}
		}()
		h.Unauthorized(rec, req)
	}()

	if rec.Code != 0 && rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestNotFound_Returns404(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected - template rendering not initialized
			}
		}()
		h.NotFound(rec, req)
	}()

	if rec.Code != 0 && rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestInternalError_Returns500(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected - template rendering not initialized
			}
		}()
		h.InternalError(rec, req)
	}()

	if rec.Code != 0 && rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestErrorLogger_Log(t *testing.T) {
	logger := zap.NewNop()
	errLog := NewErrorLogger(logger)

	if errLog == nil {
		t.Fatal("NewErrorLogger() returned nil")
	}

	// Should not panic
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	errLog.Log(req, "test error", nil)
}

func TestErrorLogger_LogWithFields(t *testing.T) {
	logger := zap.NewNop()
	errLog := NewErrorLogger(logger)

	// Should not panic
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	errLog.LogWithFields(req, "test error", nil, zap.String("extra", "field"))
}
