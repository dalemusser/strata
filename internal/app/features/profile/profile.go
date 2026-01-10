// internal/app/features/profile/profile.go
package profile

import (
	"net/http"

	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	userstore "github.com/dalemusser/strata/internal/app/store/users"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/authutil"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/strata/internal/domain/models"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides profile handlers.
type Handler struct {
	userStore *userstore.Store
	errLog    *errorsfeature.ErrorLogger
	logger    *zap.Logger
}

// NewHandler creates a new profile Handler.
func NewHandler(db *mongo.Database, errLog *errorsfeature.ErrorLogger, logger *zap.Logger) *Handler {
	return &Handler{
		userStore: userstore.New(db),
		errLog:    errLog,
		logger:    logger,
	}
}

// ProfileVM is the view model for the profile page.
type ProfileVM struct {
	viewdata.BaseVM
	User    *models.User
	Success string
	Error   string
}

// Routes returns a chi.Router with profile routes mounted.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.showProfile)
	r.Post("/", h.updateProfile)
	r.Get("/change-password", h.showChangePassword)
	r.Post("/change-password", h.handleChangePassword)
	r.Post("/theme", h.updateTheme)

	return r
}

// showProfile displays the user profile.
func (h *Handler) showProfile(w http.ResponseWriter, r *http.Request) {
	sessionUser, ok := auth.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.userStore.GetByID(r.Context(), sessionUser.UserID())
	if err != nil {
		h.errLog.Log(r, "failed to get user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	vm := ProfileVM{
		BaseVM: viewdata.New(r),
		User:   user,
	}
	vm.Title = "Profile"

	if r.URL.Query().Get("success") == "1" {
		vm.Success = "Profile updated successfully"
	}

	templates.Render(w, r, "profile/show", vm)
}

// updateProfile updates the user profile.
func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	sessionUser, ok := auth.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	if err := h.userStore.UpdateFromInput(r.Context(), sessionUser.UserID(), update); err != nil {
		h.errLog.Log(r, "failed to update user", err)

		user, _ := h.userStore.GetByID(r.Context(), sessionUser.UserID())
		vm := ProfileVM{
			BaseVM: viewdata.New(r),
			User:   user,
			Error:  "Failed to update profile",
		}
		templates.Render(w, r, "profile/show", vm)
		return
	}

	http.Redirect(w, r, "/profile?success=1", http.StatusSeeOther)
}

// ChangePasswordVM is the view model for the change password page.
type ChangePasswordVM struct {
	viewdata.BaseVM
	Required bool
	Success  string
	Error    string
}

// showChangePassword displays the change password form.
func (h *Handler) showChangePassword(w http.ResponseWriter, r *http.Request) {
	vm := ChangePasswordVM{
		BaseVM:   viewdata.New(r),
		Required: r.URL.Query().Get("required") == "1",
	}
	vm.Title = "Change Password"

	templates.Render(w, r, "profile/change_password", vm)
}

// handleChangePassword processes the password change.
func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	sessionUser, ok := auth.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate new password
	if newPassword != confirmPassword {
		vm := ChangePasswordVM{
			BaseVM: viewdata.New(r),
			Error:  "Passwords do not match",
		}
		templates.Render(w, r, "profile/change_password", vm)
		return
	}

	if err := authutil.ValidatePassword(newPassword); err != nil {
		vm := ChangePasswordVM{
			BaseVM: viewdata.New(r),
			Error:  err.Error(),
		}
		templates.Render(w, r, "profile/change_password", vm)
		return
	}

	// Get user and verify current password
	user, err := h.userStore.GetByID(r.Context(), sessionUser.UserID())
	if err != nil {
		h.errLog.Log(r, "failed to get user", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Only check current password if user has one (not temp password)
	if user.PasswordHash != nil && (user.PasswordTemp == nil || !*user.PasswordTemp) {
		if !authutil.CheckPassword(currentPassword, *user.PasswordHash) {
			vm := ChangePasswordVM{
				BaseVM: viewdata.New(r),
				Error:  "Current password is incorrect",
			}
			templates.Render(w, r, "profile/change_password", vm)
			return
		}
	}

	// Hash new password
	hash, err := authutil.HashPassword(newPassword)
	if err != nil {
		h.errLog.Log(r, "failed to hash password", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update password
	tempFalse := false
	if err := h.userStore.UpdateFromInput(r.Context(), sessionUser.UserID(), userstore.UpdateInput{
		PasswordHash: &hash,
		PasswordTemp: &tempFalse,
	}); err != nil {
		h.errLog.Log(r, "failed to update password", err)
		vm := ChangePasswordVM{
			BaseVM: viewdata.New(r),
			Error:  "Failed to update password",
		}
		templates.Render(w, r, "profile/change_password", vm)
		return
	}

	http.Redirect(w, r, "/profile?success=1", http.StatusSeeOther)
}

// updateTheme updates the user's theme preference.
func (h *Handler) updateTheme(w http.ResponseWriter, r *http.Request) {
	sessionUser, ok := auth.CurrentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	theme := r.FormValue("theme")
	if theme != "light" && theme != "dark" && theme != "system" {
		theme = "system"
	}

	if err := h.userStore.UpdateFromInput(r.Context(), sessionUser.UserID(), userstore.UpdateInput{
		ThemePreference: &theme,
	}); err != nil {
		h.errLog.Log(r, "failed to update theme", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
