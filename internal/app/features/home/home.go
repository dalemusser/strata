// internal/app/features/home/home.go
package home

import (
	"net/http"

	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides home page handlers.
type Handler struct {
	db     *mongo.Database
	logger *zap.Logger
}

// NewHandler creates a new home Handler.
func NewHandler(db *mongo.Database, logger *zap.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

// HomeVM is the view model for the home page.
type HomeVM struct {
	viewdata.BaseVM
}

// Routes returns a chi.Router with home routes mounted.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.Index)
	return r
}

// Index renders the home page.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	vm := HomeVM{
		BaseVM: viewdata.New(r),
	}
	vm.Title = "Home"

	templates.Render(w, r, "home/index", vm)
}
