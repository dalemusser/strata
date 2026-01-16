// internal/app/features/logout/logout.go
package logout

import (
	"net/http"

	"github.com/dalemusser/strata/internal/app/store/activity"
	"github.com/dalemusser/strata/internal/app/store/sessions"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/auditlog"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler provides logout handlers.
type Handler struct {
	sessionMgr    *auth.SessionManager
	auditLogger   *auditlog.Logger
	sessionsStore *sessions.Store
	activityStore *activity.Store
	logger        *zap.Logger
}

// NewHandler creates a new logout Handler.
func NewHandler(
	sessionMgr *auth.SessionManager,
	auditLogger *auditlog.Logger,
	sessionsStore *sessions.Store,
	activityStore *activity.Store,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		sessionMgr:    sessionMgr,
		auditLogger:   auditLogger,
		sessionsStore: sessionsStore,
		activityStore: activityStore,
		logger:        logger,
	}
}

// Routes returns a chi.Router with logout routes mounted.
func Routes(h *Handler, sessionMgr *auth.SessionManager) http.Handler {
	r := chi.NewRouter()
	r.Use(sessionMgr.RequireAuth)
	r.Post("/", h.handleLogout)
	r.Get("/", h.handleLogout) // Allow GET for simple logout links
	return r
}

// handleLogout terminates the session.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if user, ok := auth.CurrentUser(r); ok {
		h.auditLogger.Logout(r.Context(), r, user.ID)

		// Close session in MongoDB tracking (preserves for audit, records duration)
		if token := user.SessionToken(); token != "" {
			// Get session ID before closing for activity recording
			sess, _ := h.sessionsStore.GetByToken(r.Context(), token)

			if err := h.sessionsStore.Close(r.Context(), token, sessions.EndReasonLogout); err != nil {
				h.logger.Warn("failed to close session in store", zap.Error(err))
			} else if h.activityStore != nil && sess != nil {
				// Record logout activity event
				if err := h.activityStore.RecordLogout(r.Context(), user.UserID(), sess.ID); err != nil {
					h.logger.Warn("failed to record logout activity", zap.Error(err))
				}
			}
		}
	}

	h.sessionMgr.DestroySession(w, r)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
