// internal/app/bootstrap/routes.go
package bootstrap

import (
	"context"
	"net/http"
	"time"

	announcementsfeature "github.com/dalemusser/strata/internal/app/features/announcements"
	auditlogfeature "github.com/dalemusser/strata/internal/app/features/auditlog"
	authgooglefeature "github.com/dalemusser/strata/internal/app/features/authgoogle"
	dashboardfeature "github.com/dalemusser/strata/internal/app/features/dashboard"
	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	healthfeature "github.com/dalemusser/strata/internal/app/features/health"
	homefeature "github.com/dalemusser/strata/internal/app/features/home"
	invitationsfeature "github.com/dalemusser/strata/internal/app/features/invitations"
	loginfeature "github.com/dalemusser/strata/internal/app/features/login"
	logoutfeature "github.com/dalemusser/strata/internal/app/features/logout"
	pagesfeature "github.com/dalemusser/strata/internal/app/features/pages"
	profilefeature "github.com/dalemusser/strata/internal/app/features/profile"
	settingsfeature "github.com/dalemusser/strata/internal/app/features/settings"
	systemusersfeature "github.com/dalemusser/strata/internal/app/features/systemusers"
	appresources "github.com/dalemusser/strata/internal/app/resources"
	announcementstore "github.com/dalemusser/strata/internal/app/store/announcement"
	"github.com/dalemusser/strata/internal/app/store/audit"
	"github.com/dalemusser/strata/internal/app/store/oauthstate"
	"github.com/dalemusser/strata/internal/app/store/sessions"
	userstore "github.com/dalemusser/strata/internal/app/store/users"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/auditlog"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/waffle/config"
	"github.com/dalemusser/waffle/middleware"
	"github.com/dalemusser/waffle/pantry/fileserver"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"go.uber.org/zap"
)

// BuildHandler constructs the root HTTP handler (router) for this WAFFLE app.
//
// WAFFLE calls this after configuration, DB connections, schema setup, and
// any Startup hooks have completed. At this point you have access to:
//   - coreCfg: WAFFLE core configuration (ports, env, timeouts, etc.)
//   - appCfg: app-specific configuration defined in AppConfig
//   - deps: any DB or backend clients bundled in DBDeps
//   - logger: the fully configured zap.Logger for this app
//
// This function should:
//  1. Create a router (chi, standard mux, etc.)
//  2. Mount feature routers for different parts of your application
//  3. Add any additional middleware needed for specific routes
//  4. Return the configured router as an http.Handler
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
	// Create the session manager using app config.
	// Secure cookies are enabled in production mode.
	secure := coreCfg.Env == "prod"
	sessionMgr, err := auth.NewSessionManager(appCfg.SessionKey, appCfg.SessionName, appCfg.SessionDomain, secure, logger)
	if err != nil {
		logger.Error("session manager init failed", zap.Error(err))
		return nil, err
	}

	// Set up the UserFetcher so LoadSessionUser fetches fresh user data on each request.
	// This ensures role changes, disabled accounts, and profile updates take effect immediately.
	sessionMgr.SetUserFetcher(userstore.NewFetcher(deps.MongoDatabase, logger))

	// Initialize and boot the template engine once at startup.
	// Dev mode enables template reloading for faster iteration.
	eng := templates.New(coreCfg.Env == "dev")
	if err := eng.Boot(logger); err != nil {
		logger.Error("template engine boot failed", zap.Error(err))
		return nil, err
	}
	templates.UseEngine(eng, logger)

	// Initialize viewdata with storage and database for settings loading.
	viewdata.Init(deps.FileStorage, deps.MongoDatabase)

	// Set up announcement loader for viewdata.
	// This allows BaseVM to include active announcements for banner display.
	annStore := announcementstore.New(deps.MongoDatabase)
	viewdata.SetAnnouncementLoader(func(ctx context.Context) []viewdata.AnnouncementVM {
		announcements, err := annStore.GetActive(ctx)
		if err != nil {
			logger.Warn("failed to load active announcements", zap.Error(err))
			return nil
		}
		result := make([]viewdata.AnnouncementVM, len(announcements))
		for i, ann := range announcements {
			result[i] = viewdata.AnnouncementVM{
				ID:          ann.ID.Hex(),
				Title:       ann.Title,
				Content:     ann.Content,
				Type:        string(ann.Type),
				Dismissible: ann.Dismissible,
			}
		}
		return result
	})

	// Create error logger for handlers.
	errLog := errorsfeature.NewErrorLogger(logger)

	// Create audit store and logger for security event tracking.
	auditStore := audit.New(deps.MongoDatabase)
	auditConfig := auditlog.Config{
		Auth:  appCfg.AuditLogAuth,
		Admin: appCfg.AuditLogAdmin,
	}
	auditLogger := auditlog.New(auditStore, logger, auditConfig)

	// Create sessions store for activity tracking.
	sessionsStore := sessions.New(deps.MongoDatabase)

	r := chi.NewRouter()

	// Request timeout middleware: prevents requests from hanging indefinitely.
	// Requests exceeding 30 seconds will be cancelled and return a 503 Service Unavailable.
	r.Use(chimw.Timeout(30 * time.Second))

	// CORS middleware: must be early in the chain to handle preflight requests.
	// Only active when enable_cors=true in config.
	r.Use(middleware.CORSFromConfig(coreCfg))

	// Global auth middleware: loads SessionUser into context if logged in.
	// This makes the current user available to all handlers via auth.CurrentUser(r).
	r.Use(sessionMgr.LoadSessionUser)

	// CSRF protection middleware: protects POST/PUT/DELETE requests from cross-site request forgery.
	// The CSRF token must be included in forms as a hidden field or in the X-CSRF-Token header.
	csrfMiddleware := csrf.Protect(
		[]byte(appCfg.CSRFKey),
		csrf.Secure(secure),
		csrf.Path("/"),
		csrf.CookieName("csrf_token"),
		csrf.FieldName("csrf_token"),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Warn("CSRF validation failed",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("reason", csrf.FailureReason(r).Error()),
			)
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.Error(w, "CSRF token invalid or missing", http.StatusForbidden)
		})),
	)
	r.Use(csrfMiddleware)

	// Health check endpoint for load balancers and orchestrators
	healthHandler := healthfeature.NewHandler(deps.MongoClient, logger)
	r.Mount("/health", healthfeature.Routes(healthHandler))

	// Static assets with pre-compressed file support (gzip/brotli)
	// /static/* serves files from disk (static directory)
	r.Handle("/static/*", fileserver.Handler("/static", "static"))

	// /assets/* serves embedded assets (bundled into the binary)
	r.Handle("/assets/*", appresources.AssetsHandler("/assets"))

	// Uploaded files (local storage only)
	// When using local storage, serve files from the configured path
	if appCfg.StorageType == "local" || appCfg.StorageType == "" {
		r.Handle(appCfg.StorageLocalURL+"/*", fileserver.Handler(appCfg.StorageLocalURL, appCfg.StorageLocalPath))
	}

	// Public pages
	homeHandler := homefeature.NewHandler(deps.MongoDatabase, logger)
	r.Mount("/", homefeature.Routes(homeHandler))

	// Dynamic content pages (about, contact, terms, privacy)
	pagesHandler := pagesfeature.NewHandler(deps.MongoDatabase, errLog, logger)
	r.Mount("/about", pagesHandler.AboutRouter())
	r.Mount("/contact", pagesHandler.ContactRouter())
	r.Mount("/terms", pagesHandler.TermsRouter())
	r.Mount("/privacy", pagesHandler.PrivacyRouter())
	r.Mount("/pages", pagesfeature.EditRoutes(pagesHandler, sessionMgr))

	// User Invitations (public accept route)
	invitationsHandler := invitationsfeature.NewHandler(
		deps.MongoDatabase,
		sessionMgr,
		sessionsStore,
		errLog,
		deps.Mailer,
		auditLogger,
		appCfg.BaseURL,
		7*24*time.Hour, // 7 days expiry
		logger,
	)
	r.Mount("/invite", invitationsfeature.AcceptRoutes(invitationsHandler))

	// Authentication
	googleEnabled := appCfg.GoogleClientID != "" && appCfg.GoogleClientSecret != ""
	// Trust login is only enabled in dev mode for security - it allows passwordless login
	trustLoginEnabled := coreCfg.Env == "dev"
	loginHandler := loginfeature.NewHandler(
		deps.MongoDatabase,
		sessionMgr,
		errLog,
		deps.Mailer,
		auditLogger,
		sessionsStore,
		appCfg.BaseURL,
		appCfg.EmailVerifyExpiry,
		googleEnabled,
		trustLoginEnabled,
		logger,
	)
	r.Mount("/login", loginfeature.Routes(loginHandler))

	logoutHandler := logoutfeature.NewHandler(sessionMgr, auditLogger, sessionsStore, logger)
	r.Mount("/logout", logoutfeature.Routes(logoutHandler, sessionMgr))

	// Google OAuth (only mount if configured)
	if googleEnabled {
		oauthStateStore := oauthstate.New(deps.MongoDatabase)
		googleHandler := authgooglefeature.NewHandler(
			deps.MongoDatabase,
			sessionMgr,
			errLog,
			auditLogger,
			sessionsStore,
			oauthStateStore,
			appCfg.GoogleClientID,
			appCfg.GoogleClientSecret,
			appCfg.BaseURL,
			logger,
		)
		r.Mount("/auth/google", authgooglefeature.Routes(googleHandler))
		logger.Info("Google OAuth enabled", zap.String("redirect_url", appCfg.BaseURL+"/auth/google/callback"))
	}

	// User profile (any logged-in user)
	profileHandler := profilefeature.NewHandler(deps.MongoDatabase, sessionsStore, errLog, logger)
	r.Route("/profile", func(sr chi.Router) {
		sr.Use(sessionMgr.RequireRole("admin"))
		sr.Mount("/", profilefeature.Routes(profileHandler, sessionMgr))
	})

	// Error pages
	errorsHandler := errorsfeature.NewHandler()
	r.Get("/forbidden", errorsHandler.Forbidden)
	r.Get("/unauthorized", errorsHandler.Unauthorized)

	// Role-based dashboards
	dashboardHandler := dashboardfeature.NewHandler(deps.MongoDatabase, logger)
	r.Mount("/dashboard", dashboardfeature.Routes(dashboardHandler, sessionMgr))

	// System user management (admin only)
	sysUsersHandler := systemusersfeature.NewHandler(deps.MongoDatabase, errLog, auditLogger, logger)
	r.Mount("/system-users", systemusersfeature.Routes(sysUsersHandler, sessionMgr))

	// Audit log (admin only)
	auditLogHandler := auditlogfeature.NewHandler(deps.MongoDatabase, errLog, logger)
	r.Mount("/audit", auditlogfeature.Routes(auditLogHandler, sessionMgr))

	// User Invitations management (admin only)
	r.Mount("/invitations", invitationsfeature.AdminRoutes(invitationsHandler, sessionMgr))

	// Announcements management (admin only)
	announcementsHandler := announcementsfeature.NewHandler(deps.MongoDatabase, errLog, logger)
	r.Mount("/announcements", announcementsfeature.Routes(announcementsHandler, sessionMgr))

	// Site Settings (admin only)
	settingsHandler := settingsfeature.NewHandler(deps.MongoDatabase, deps.FileStorage, errLog, logger)
	r.Route("/settings", func(sr chi.Router) {
		sr.Use(sessionMgr.RequireRole("admin"))
		settingsHandler.MountRoutes(sr)
	})

	return r, nil
}
