// internal/app/features/auditlog/auditlog.go
package auditlog

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"net/http"
	"strconv"
	"time"

	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	"github.com/dalemusser/strata/internal/app/store/audit"
	userstore "github.com/dalemusser/strata/internal/app/store/users"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/timezones"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/strata/internal/domain/models"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides audit log handlers.
type Handler struct {
	auditStore *audit.Store
	userStore  *userstore.Store
	errLog     *errorsfeature.ErrorLogger
	logger     *zap.Logger
}

// NewHandler creates a new audit log Handler.
func NewHandler(
	db *mongo.Database,
	errLog *errorsfeature.ErrorLogger,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		auditStore: audit.New(db),
		userStore:  userstore.New(db),
		errLog:     errLog,
		logger:     logger,
	}
}

// EventDisplay represents an audit event for display.
type EventDisplay struct {
	audit.Event
	UserName    string
	UserLoginID string
	ActorName   string
}

// ListVM is the view model for the audit log list.
type ListVM struct {
	viewdata.BaseVM
	Events         []EventDisplay
	Filter         FilterParams
	TotalCount     int64
	Page           int
	PrevPage       int
	NextPage       int
	PageSize       int
	TotalPages     int
	TimezoneGroups []timezones.ZoneGroup
}

// FilterParams represents the filter parameters.
type FilterParams struct {
	Category  string
	EventType string
	UserID    string
	StartDate string
	EndDate   string
}

// Routes returns a chi.Router with audit log routes mounted.
func Routes(h *Handler, sessionMgr *auth.SessionManager) http.Handler {
	r := chi.NewRouter()
	r.Use(sessionMgr.RequireRole("admin"))

	r.Get("/", h.list)

	return r
}

// list displays the audit log with filtering and pagination.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse pagination
	page := 1
	if p := q.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 50

	// Parse filters
	filter := audit.QueryFilter{
		Category:  q.Get("category"),
		EventType: q.Get("event_type"),
		Limit:     int64(pageSize),
		Offset:    int64((page - 1) * pageSize),
	}

	if userID := q.Get("user_id"); userID != "" {
		if objID, err := primitive.ObjectIDFromHex(userID); err == nil {
			filter.UserID = &objID
		}
	}

	if startDate := q.Get("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartTime = &t
		}
	}

	if endDate := q.Get("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndTime = &endOfDay
		}
	}

	// Query events
	events, err := h.auditStore.Query(r.Context(), filter)
	if err != nil {
		h.errLog.Log(r, "failed to query audit events", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	totalCount, err := h.auditStore.CountByFilter(r.Context(), filter)
	if err != nil {
		h.errLog.Log(r, "failed to count audit events", err)
		totalCount = 0
	}

	// Build user lookup map
	userIDs := make(map[primitive.ObjectID]bool)
	for _, e := range events {
		if e.UserID != nil {
			userIDs[*e.UserID] = true
		}
		if e.ActorID != nil {
			userIDs[*e.ActorID] = true
		}
	}

	userMap := make(map[primitive.ObjectID]*models.User)
	for id := range userIDs {
		if user, err := h.userStore.GetByID(r.Context(), id); err == nil {
			userMap[id] = user
		}
	}

	// Build display events
	displayEvents := make([]EventDisplay, len(events))
	for i, e := range events {
		display := EventDisplay{Event: e}
		if e.UserID != nil {
			if user, ok := userMap[*e.UserID]; ok {
				display.UserName = user.FullName
				if user.LoginID != nil {
					display.UserLoginID = *user.LoginID
				}
			}
		}
		if e.ActorID != nil {
			if user, ok := userMap[*e.ActorID]; ok {
				display.ActorName = user.FullName
			}
		}
		displayEvents[i] = display
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	// Get timezone groups for selector
	tzGroups, _ := timezones.Groups()

	vm := ListVM{
		BaseVM: viewdata.New(r),
		Events: displayEvents,
		Filter: FilterParams{
			Category:  q.Get("category"),
			EventType: q.Get("event_type"),
			UserID:    q.Get("user_id"),
			StartDate: q.Get("start_date"),
			EndDate:   q.Get("end_date"),
		},
		TotalCount:     totalCount,
		Page:           page,
		PrevPage:       page - 1,
		NextPage:       page + 1,
		PageSize:       pageSize,
		TotalPages:     totalPages,
		TimezoneGroups: tzGroups,
	}
	vm.Title = "Audit Log"

	templates.Render(w, r, "auditlog/list", vm)
}
