// internal/app/features/settings/settings.go
package settings

import (
	"net/http"

	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	settingsstore "github.com/dalemusser/strata/internal/app/store/settings"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/strata/internal/domain/models"
	"github.com/dalemusser/waffle/pantry/storage"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides settings handlers.
type Handler struct {
	settingsStore *settingsstore.Store
	fileStorage   storage.Store
	errLog        *errorsfeature.ErrorLogger
	logger        *zap.Logger
}

// NewHandler creates a new settings Handler.
func NewHandler(
	db *mongo.Database,
	fileStorage storage.Store,
	errLog *errorsfeature.ErrorLogger,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		settingsStore: settingsstore.New(db),
		fileStorage:   fileStorage,
		errLog:        errLog,
		logger:        logger,
	}
}

// SettingsVM is the view model for the settings page.
type SettingsVM struct {
	viewdata.BaseVM
	Settings *models.SiteSettings
	Success  string
	Error    string
}

// MountRoutes mounts settings routes on the given router.
func (h *Handler) MountRoutes(r chi.Router) {
	r.Get("/", h.show)
	r.Post("/", h.update)
	r.Post("/logo", h.uploadLogo)
	r.Delete("/logo", h.deleteLogo)
}

// show displays the settings page.
func (h *Handler) show(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsStore.Get(r.Context())
	if err != nil && err != mongo.ErrNoDocuments {
		h.errLog.Log(r, "failed to get settings", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if settings == nil {
		settings = &models.SiteSettings{
			SiteName: "Strata",
		}
	}

	vm := SettingsVM{
		BaseVM:   viewdata.New(r),
		Settings: settings,
	}
	vm.Title = "Site Settings"

	if r.URL.Query().Get("success") == "1" {
		vm.Success = "Settings updated successfully"
	}

	templates.Render(w, r, "settings/show", vm)
}

// update saves the settings.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	input := settingsstore.UpdateInput{
		SiteName:   r.FormValue("site_name"),
		FooterHTML: r.FormValue("footer_html"),
	}

	if err := h.settingsStore.Upsert(r.Context(), input); err != nil {
		h.errLog.Log(r, "failed to update settings", err)

		settings, _ := h.settingsStore.Get(r.Context())
		vm := SettingsVM{
			BaseVM:   viewdata.New(r),
			Settings: settings,
			Error:    "Failed to save settings",
		}
		templates.Render(w, r, "settings/show", vm)
		return
	}

	http.Redirect(w, r, "/settings?success=1", http.StatusSeeOther)
}

// uploadLogo handles logo upload.
func (h *Handler) uploadLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		h.errLog.Log(r, "failed to parse multipart form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		h.errLog.Log(r, "failed to get uploaded file", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate filename
	filename := "uploads/logo-" + header.Filename

	// Upload to storage
	opts := &storage.PutOptions{
		ContentType: header.Header.Get("Content-Type"),
	}
	if err := h.fileStorage.Put(r.Context(), filename, file, opts); err != nil {
		h.errLog.Log(r, "failed to upload logo", err)
		http.Error(w, "Failed to upload logo", http.StatusInternalServerError)
		return
	}

	// Generate URL for the uploaded file (assuming local storage serves at /uploads/)
	url := "/uploads/" + filename

	// Update settings with new logo URL
	settings, _ := h.settingsStore.Get(r.Context())
	if settings == nil {
		settings = &models.SiteSettings{}
	}

	input := settingsstore.UpdateInput{
		SiteName:   settings.SiteName,
		FooterHTML: settings.FooterHTML,
		LogoURL:    url,
	}

	if err := h.settingsStore.Upsert(r.Context(), input); err != nil {
		h.errLog.Log(r, "failed to save logo URL", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings?success=1", http.StatusSeeOther)
}

// deleteLogo removes the logo.
func (h *Handler) deleteLogo(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.settingsStore.Get(r.Context())
	if settings == nil || settings.LogoPath == "" {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	// Delete from storage
	if err := h.fileStorage.Delete(r.Context(), settings.LogoPath); err != nil {
		h.logger.Warn("failed to delete logo file", zap.Error(err))
		// Continue anyway - file may not exist
	}

	// Clear logo URL in settings
	input := settingsstore.UpdateInput{
		SiteName:   settings.SiteName,
		FooterHTML: settings.FooterHTML,
		LogoURL:    "",
	}

	if err := h.settingsStore.Upsert(r.Context(), input); err != nil {
		h.errLog.Log(r, "failed to clear logo URL", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings?success=1", http.StatusSeeOther)
}
