// internal/app/features/systemusers/systemusers.go
package systemusers

import (
	"net/http"

	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	userstore "github.com/dalemusser/strata/internal/app/store/users"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/auditlog"
	"github.com/dalemusser/strata/internal/app/system/authutil"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/strata/internal/domain/models"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides system users management handlers.
type Handler struct {
	userStore   *userstore.Store
	errLog      *errorsfeature.ErrorLogger
	auditLogger *auditlog.Logger
	logger      *zap.Logger
}

// NewHandler creates a new system users Handler.
func NewHandler(
	db *mongo.Database,
	errLog *errorsfeature.ErrorLogger,
	auditLogger *auditlog.Logger,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		userStore:   userstore.New(db),
		errLog:      errLog,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// ListVM is the view model for the users list.
type ListVM struct {
	viewdata.BaseVM
	Users []models.User
}

// Routes returns a chi.Router with system users routes mounted.
func Routes(h *Handler, sessionMgr *auth.SessionManager) http.Handler {
	r := chi.NewRouter()
	r.Use(sessionMgr.RequireRole("admin"))

	r.Get("/", h.list)
	r.Get("/new", h.showNew)
	r.Post("/new", h.create)
	r.Get("/{id}", h.show)
	r.Get("/{id}/edit", h.showEdit)
	r.Post("/{id}", h.update)
	r.Post("/{id}/disable", h.disable)
	r.Post("/{id}/enable", h.enable)
	r.Post("/{id}/reset-password", h.resetPassword)

	return r
}

// list displays all system users.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.userStore.ListAll(r.Context())
	if err != nil {
		h.errLog.Log(r, "failed to list users", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	vm := ListVM{
		BaseVM: viewdata.New(r),
		Users:  users,
	}
	vm.Title = "System Users"

	templates.Render(w, r, "systemusers/list", vm)
}

// NewUserVM is the view model for creating a new user.
type NewUserVM struct {
	viewdata.BaseVM
	FullName   string
	LoginID    string
	Email      string
	AuthMethod string
	Error      string
}

// showNew displays the new user form.
func (h *Handler) showNew(w http.ResponseWriter, r *http.Request) {
	vm := NewUserVM{
		BaseVM:     viewdata.New(r),
		AuthMethod: "trust",
	}
	vm.Title = "New User"

	templates.Render(w, r, "systemusers/new", vm)
}

// create creates a new system user.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.CurrentUser(r)

	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	input := userstore.CreateInput{
		FullName:   r.FormValue("full_name"),
		LoginID:    r.FormValue("login_id"),
		Email:      r.FormValue("email"),
		AuthMethod: r.FormValue("auth_method"),
		Role:       "admin", // All system users are admins in strata base
	}

	// Handle password for password auth
	if input.AuthMethod == "password" {
		password := r.FormValue("password")
		if password == "" {
			vm := NewUserVM{
				BaseVM:     viewdata.New(r),
				FullName:   input.FullName,
				LoginID:    input.LoginID,
				Email:      input.Email,
				AuthMethod: input.AuthMethod,
				Error:      "Password is required for password authentication",
			}
			templates.Render(w, r, "systemusers/new", vm)
			return
		}

		hash, err := authutil.HashPassword(password)
		if err != nil {
			h.errLog.Log(r, "failed to hash password", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		input.PasswordHash = &hash

		// Mark as temporary if checkbox is checked
		if r.FormValue("temp_password") == "on" {
			temp := true
			input.PasswordTemp = &temp
		}
	}

	user, err := h.userStore.CreateFromInput(r.Context(), input)
	if err != nil {
		h.errLog.Log(r, "failed to create user", err)

		vm := NewUserVM{
			BaseVM:     viewdata.New(r),
			FullName:   input.FullName,
			LoginID:    input.LoginID,
			Email:      input.Email,
			AuthMethod: input.AuthMethod,
			Error:      "Failed to create user. Login ID may already be in use.",
		}
		templates.Render(w, r, "systemusers/new", vm)
		return
	}

	actorID := actor.UserID()
	h.auditLogger.LogAdminEvent(r, &actorID, &user.ID, "user_created", nil)

	http.Redirect(w, r, "/system-users", http.StatusSeeOther)
}

// ShowVM is the view model for viewing a user.
type ShowVM struct {
	viewdata.BaseVM
	User *models.User
}

// show displays a single user.
func (h *Handler) show(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user, err := h.userStore.GetByID(r.Context(), objID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.NotFound(w, r)
			return
		}
		h.errLog.Log(r, "failed to get user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	vm := ShowVM{
		BaseVM: viewdata.New(r),
		User:   user,
	}
	vm.Title = user.FullName

	templates.Render(w, r, "systemusers/show", vm)
}

// EditVM is the view model for editing a user.
type EditVM struct {
	viewdata.BaseVM
	User    *models.User
	Success string
	Error   string
}

// showEdit displays the edit user form.
func (h *Handler) showEdit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user, err := h.userStore.GetByID(r.Context(), objID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.NotFound(w, r)
			return
		}
		h.errLog.Log(r, "failed to get user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	vm := EditVM{
		BaseVM: viewdata.New(r),
		User:   user,
	}
	vm.Title = "Edit " + user.FullName

	if r.URL.Query().Get("success") == "1" {
		vm.Success = "User updated successfully"
	}

	templates.Render(w, r, "systemusers/edit", vm)
}

// update updates a user.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.CurrentUser(r)

	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fullName := r.FormValue("full_name")
	email := r.FormValue("email")

	update := userstore.UpdateInput{
		FullName: &fullName,
	}
	if email != "" {
		update.Email = &email
	}

	if err := h.userStore.UpdateFromInput(r.Context(), objID, update); err != nil {
		h.errLog.Log(r, "failed to update user", err)

		user, _ := h.userStore.GetByID(r.Context(), objID)
		vm := EditVM{
			BaseVM: viewdata.New(r),
			User:   user,
			Error:  "Failed to update user",
		}
		templates.Render(w, r, "systemusers/edit", vm)
		return
	}

	actorID := actor.UserID()
	h.auditLogger.LogAdminEvent(r, &actorID, &objID, "user_updated", nil)

	http.Redirect(w, r, "/system-users/"+id+"/edit?success=1", http.StatusSeeOther)
}

// disable disables a user account.
func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.CurrentUser(r)

	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Prevent disabling self
	if actor.UserID() == objID {
		http.Redirect(w, r, "/system-users/"+id+"/edit?error=cannot_disable_self", http.StatusSeeOther)
		return
	}

	status := "disabled"
	if err := h.userStore.UpdateFromInput(r.Context(), objID, userstore.UpdateInput{
		Status: &status,
	}); err != nil {
		h.errLog.Log(r, "failed to disable user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	actorID := actor.UserID()
	h.auditLogger.LogAdminEvent(r, &actorID, &objID, "user_disabled", nil)

	http.Redirect(w, r, "/system-users", http.StatusSeeOther)
}

// enable enables a user account.
func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.CurrentUser(r)

	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	status := "active"
	if err := h.userStore.UpdateFromInput(r.Context(), objID, userstore.UpdateInput{
		Status: &status,
	}); err != nil {
		h.errLog.Log(r, "failed to enable user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	actorID := actor.UserID()
	h.auditLogger.LogAdminEvent(r, &actorID, &objID, "user_enabled", nil)

	http.Redirect(w, r, "/system-users", http.StatusSeeOther)
}

// resetPassword resets a user's password to a temporary one.
func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.CurrentUser(r)

	id := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	newPassword := r.FormValue("new_password")
	if newPassword == "" {
		http.Redirect(w, r, "/system-users/"+id+"/edit?error=password_required", http.StatusSeeOther)
		return
	}

	hash, err := authutil.HashPassword(newPassword)
	if err != nil {
		h.errLog.Log(r, "failed to hash password", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tempTrue := true
	if err := h.userStore.UpdateFromInput(r.Context(), objID, userstore.UpdateInput{
		PasswordHash: &hash,
		PasswordTemp: &tempTrue,
	}); err != nil {
		h.errLog.Log(r, "failed to reset password", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	actorID := actor.UserID()
	h.auditLogger.LogAdminEvent(r, &actorID, &objID, "password_reset", nil)

	http.Redirect(w, r, "/system-users/"+id+"/edit?success=1", http.StatusSeeOther)
}
