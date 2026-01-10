// internal/app/bootstrap/routes.go
package bootstrap

import (
	"net/http"

	auditlogfeature "github.com/dalemusser/strata/internal/app/features/auditlog"
	authgooglefeature "github.com/dalemusser/strata/internal/app/features/authgoogle"
	dashboardfeature "github.com/dalemusser/strata/internal/app/features/dashboard"
	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	healthfeature "github.com/dalemusser/strata/internal/app/features/health"
	homefeature "github.com/dalemusser/strata/internal/app/features/home"
	loginfeature "github.com/dalemusser/strata/internal/app/features/login"
	logoutfeature "github.com/dalemusser/strata/internal/app/features/logout"
	pagesfeature "github.com/dalemusser/strata/internal/app/features/pages"
	profilefeature "github.com/dalemusser/strata/internal/app/features/profile"
	settingsfeature "github.com/dalemusser/strata/internal/app/features/settings"
	systemusersfeature "github.com/dalemusser/strata/internal/app/features/systemusers"
	appresources "github.com/dalemusser/strata/internal/app/resources"
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

	// Initialize viewdata with storage for logo URLs.
	viewdata.Init(deps.FileStorage)

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

	// CORS middleware: must be early in the chain to handle preflight requests.
	// Only active when enable_cors=true in config.
	r.Use(middleware.CORSFromConfig(coreCfg))

	// Global auth middleware: loads SessionUser into context if logged in.
	// This makes the current user available to all handlers via auth.CurrentUser(r).
	r.Use(sessionMgr.LoadSessionUser)

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

	// Authentication
	googleEnabled := appCfg.GoogleClientID != "" && appCfg.GoogleClientSecret != ""
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
	profileHandler := profilefeature.NewHandler(deps.MongoDatabase, errLog, logger)
	r.Route("/profile", func(sr chi.Router) {
		sr.Use(sessionMgr.RequireRole("admin"))
		sr.Mount("/", profilefeature.Routes(profileHandler))
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

	// Site Settings (admin only)
	settingsHandler := settingsfeature.NewHandler(deps.MongoDatabase, deps.FileStorage, errLog, logger)
	r.Route("/settings", func(sr chi.Router) {
		sr.Use(sessionMgr.RequireRole("admin"))
		settingsHandler.MountRoutes(sr)
	})

	return r, nil
}
