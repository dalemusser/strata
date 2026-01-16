// internal/app/features/heartbeat/heartbeat.go
package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dalemusser/strata/internal/app/store/activity"
	"github.com/dalemusser/strata/internal/app/store/sessions"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles heartbeat requests for activity tracking.
type Handler struct {
	Sessions   *sessions.Store
	Activity   *activity.Store
	SessionMgr *auth.SessionManager
	Log        *zap.Logger
}

// NewHandler creates a new heartbeat handler.
func NewHandler(sessStore *sessions.Store, activityStore *activity.Store, sessionMgr *auth.SessionManager, logger *zap.Logger) *Handler {
	return &Handler{
		Sessions:   sessStore,
		Activity:   activityStore,
		SessionMgr: sessionMgr,
		Log:        logger,
	}
}

// Routes returns a chi.Router with heartbeat routes mounted.
func Routes(h *Handler, sessionMgr *auth.SessionManager) http.Handler {
	r := chi.NewRouter()
	r.Use(sessionMgr.RequireAuth)
	r.Post("/", h.ServeHeartbeat)
	return r
}

// heartbeatRequest is the JSON body for the heartbeat endpoint.
type heartbeatRequest struct {
	Page string `json:"page"`
}

// ServeHeartbeat handles POST /api/heartbeat.
// Updates the LastActivity timestamp for the user's current session.
// If the session was closed due to inactivity, creates a new one.
// Returns 401 if the session has been terminated by an admin.
func (h *Handler) ServeHeartbeat(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r)
	if !ok {
		w.WriteHeader(http.StatusOK) // Silent fail - not authenticated
		return
	}

	sessionToken := user.SessionToken()
	if sessionToken == "" {
		w.WriteHeader(http.StatusOK) // Silent fail - no session token
		return
	}

	// Parse request body to get current page
	var req heartbeatRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error, page is optional
	}

	// Check if the session is still valid (not terminated by admin)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dbSession, err := h.Sessions.GetByToken(ctx, sessionToken)
	if err != nil || dbSession == nil {
		// Session was terminated or doesn't exist - tell client to logout
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Update last activity time and current page (using token-based lookup)

	result, err := h.Sessions.UpdateCurrentPage(ctx, sessionToken, req.Page)
	if err != nil {
		h.Log.Warn("failed to update session activity",
			zap.Error(err),
			zap.String("page", req.Page))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Record page view event if page changed (but not on first heartbeat when PreviousPage is empty)
	if result.Updated && req.Page != "" && result.PreviousPage != "" && req.Page != result.PreviousPage && h.Activity != nil {
		userOID := user.UserID()
		// Look up session to get its ID for activity recording
		sessionDoc, _ := h.Sessions.GetByToken(ctx, sessionToken)
		if sessionDoc != nil {
			if err := h.Activity.RecordPageView(ctx, userOID, sessionDoc.ID, req.Page); err != nil {
				h.Log.Warn("failed to record page view",
					zap.Error(err),
					zap.String("page", req.Page))
			}
		}
	}

	// If session wasn't updated (already closed), create a new one
	if !result.Updated {
		userOID := user.UserID()
		if userOID.IsZero() {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Generate new session token
		newToken, tokenErr := auth.GenerateSessionToken()
		if tokenErr != nil {
			h.Log.Warn("failed to generate session token",
				zap.Error(tokenErr),
				zap.String("user_id", user.ID))
			w.WriteHeader(http.StatusOK)
			return
		}

		// Create new activity session
		now := time.Now()
		newSess := sessions.Session{
			Token:        newToken,
			UserID:       userOID,
			IPAddress:    clientIP(r),
			UserAgent:    r.UserAgent(),
			LoginAt:      now,
			LastActivity: now,
			ExpiresAt:    now.Add(24 * 30 * time.Hour), // 30 days
		}
		if err := h.Sessions.Create(ctx, newSess); err != nil {
			h.Log.Warn("failed to create new activity session after timeout",
				zap.Error(err),
				zap.String("user_id", user.ID))
			w.WriteHeader(http.StatusOK)
			return
		}

		// Update cookie with new session token via session manager
		sess, err := h.SessionMgr.GetSession(r)
		if err == nil {
			sess.Values["session_token"] = newToken
			if err := sess.Save(r, w); err != nil {
				h.Log.Warn("failed to save session with new session_token",
					zap.Error(err))
			}
		}

		h.Log.Info("created new activity session after inactivity timeout",
			zap.String("user_id", user.ID))
	}

	w.WriteHeader(http.StatusOK)
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// The first IP in the list is the client IP
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
