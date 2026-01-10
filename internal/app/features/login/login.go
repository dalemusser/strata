// internal/app/features/login/login.go
package login

import (
	"net/http"
	"time"

	errorsfeature "github.com/dalemusser/strata/internal/app/features/errors"
	"github.com/dalemusser/strata/internal/app/store/emailverify"
	"github.com/dalemusser/strata/internal/app/store/sessions"
	userstore "github.com/dalemusser/strata/internal/app/store/users"
	"github.com/dalemusser/strata/internal/app/system/auth"
	"github.com/dalemusser/strata/internal/app/system/auditlog"
	"github.com/dalemusser/strata/internal/app/system/authutil"
	"github.com/dalemusser/strata/internal/app/system/mailer"
	"github.com/dalemusser/strata/internal/app/system/viewdata"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Handler provides login handlers.
type Handler struct {
	userStore         *userstore.Store
	emailVerifyStore  *emailverify.Store
	sessionsStore     *sessions.Store
	sessionMgr        *auth.SessionManager
	errLog            *errorsfeature.ErrorLogger
	mailer            *mailer.Mailer
	auditLogger       *auditlog.Logger
	baseURL           string
	emailVerifyExpiry time.Duration
	googleEnabled     bool
	logger            *zap.Logger
}

// NewHandler creates a new login Handler.
func NewHandler(
	db *mongo.Database,
	sessionMgr *auth.SessionManager,
	errLog *errorsfeature.ErrorLogger,
	m *mailer.Mailer,
	auditLogger *auditlog.Logger,
	sessionsStore *sessions.Store,
	baseURL string,
	emailVerifyExpiry time.Duration,
	googleEnabled bool,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		userStore:         userstore.New(db),
		emailVerifyStore:  emailverify.New(db, emailVerifyExpiry),
		sessionsStore:     sessionsStore,
		sessionMgr:        sessionMgr,
		errLog:            errLog,
		mailer:            m,
		auditLogger:       auditLogger,
		baseURL:           baseURL,
		emailVerifyExpiry: emailVerifyExpiry,
		googleEnabled:     googleEnabled,
		logger:            logger,
	}
}

// LoginVM is the view model for the login page.
type LoginVM struct {
	viewdata.BaseVM
	GoogleEnabled bool
	Error         string
	LoginID       string
}

// Routes returns a chi.Router with login routes mounted.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.showLogin)
	r.Post("/", h.handleLogin)

	// Trust auth (development)
	r.Get("/trust", h.showTrustLogin)
	r.Post("/trust", h.handleTrustLogin)

	// Password auth
	r.Get("/password", h.showPasswordLogin)
	r.Post("/password", h.handlePasswordLogin)

	// Email verification auth
	r.Get("/email", h.showEmailLogin)
	r.Post("/email", h.handleEmailLogin)
	r.Get("/email/verify", h.showEmailVerify)
	r.Post("/email/verify", h.handleEmailVerify)
	r.Get("/email/magic", h.handleMagicLink)

	return r
}

// showLogin displays the login method selection page.
func (h *Handler) showLogin(w http.ResponseWriter, r *http.Request) {
	vm := LoginVM{
		BaseVM:        viewdata.New(r),
		GoogleEnabled: h.googleEnabled,
	}
	vm.Title = "Login"

	templates.Render(w, r, "login/index", vm)
}

// handleLogin redirects to the appropriate login method.
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	method := r.FormValue("method")
	switch method {
	case "trust":
		http.Redirect(w, r, "/login/trust", http.StatusSeeOther)
	case "password":
		http.Redirect(w, r, "/login/password", http.StatusSeeOther)
	case "email":
		http.Redirect(w, r, "/login/email", http.StatusSeeOther)
	case "google":
		http.Redirect(w, r, "/auth/google", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// TrustLoginVM is the view model for trust login.
type TrustLoginVM struct {
	viewdata.BaseVM
	Error   string
	LoginID string
}

// showTrustLogin displays the trust login form.
func (h *Handler) showTrustLogin(w http.ResponseWriter, r *http.Request) {
	vm := TrustLoginVM{
		BaseVM: viewdata.New(r),
	}
	vm.Title = "Trust Login"

	templates.Render(w, r, "login/trust", vm)
}

// handleTrustLogin processes trust login (development only).
func (h *Handler) handleTrustLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	loginID := r.FormValue("login_id")

	user, err := h.userStore.GetByLoginID(r.Context(), loginID)
	if err != nil {
		h.auditLogger.LogAuthEvent(r, nil, "login_failed_user_not_found", false, "user not found")

		vm := TrustLoginVM{
			BaseVM:  viewdata.New(r),
			Error:   "User not found",
			LoginID: loginID,
		}
		templates.Render(w, r, "login/trust", vm)
		return
	}

	if user.Status != "active" {
		h.auditLogger.LogAuthEvent(r, &user.ID, "login_failed_user_disabled", false, "user disabled")

		vm := TrustLoginVM{
			BaseVM:  viewdata.New(r),
			Error:   "Account is disabled",
			LoginID: loginID,
		}
		templates.Render(w, r, "login/trust", vm)
		return
	}

	// Create session
	if err := h.sessionMgr.CreateSession(w, r, user.ID, user.Role); err != nil {
		h.errLog.Log(r, "failed to create session", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.auditLogger.LogAuthEvent(r, &user.ID, "login_success", true, "")

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// PasswordLoginVM is the view model for password login.
type PasswordLoginVM struct {
	viewdata.BaseVM
	Error   string
	LoginID string
}

// showPasswordLogin displays the password login form.
func (h *Handler) showPasswordLogin(w http.ResponseWriter, r *http.Request) {
	vm := PasswordLoginVM{
		BaseVM: viewdata.New(r),
	}
	vm.Title = "Password Login"

	templates.Render(w, r, "login/password", vm)
}

// handlePasswordLogin processes password login.
func (h *Handler) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	loginID := r.FormValue("login_id")
	password := r.FormValue("password")

	user, err := h.userStore.GetByLoginID(r.Context(), loginID)
	if err != nil {
		h.auditLogger.LogAuthEvent(r, nil, "login_failed_user_not_found", false, "user not found")

		vm := PasswordLoginVM{
			BaseVM:  viewdata.New(r),
			Error:   "Invalid credentials",
			LoginID: loginID,
		}
		templates.Render(w, r, "login/password", vm)
		return
	}

	if user.Status != "active" {
		h.auditLogger.LogAuthEvent(r, &user.ID, "login_failed_user_disabled", false, "user disabled")

		vm := PasswordLoginVM{
			BaseVM:  viewdata.New(r),
			Error:   "Account is disabled",
			LoginID: loginID,
		}
		templates.Render(w, r, "login/password", vm)
		return
	}

	if user.PasswordHash == nil || !authutil.CheckPassword(password, *user.PasswordHash) {
		h.auditLogger.LogAuthEvent(r, &user.ID, "login_failed_wrong_password", false, "wrong password")

		vm := PasswordLoginVM{
			BaseVM:  viewdata.New(r),
			Error:   "Invalid credentials",
			LoginID: loginID,
		}
		templates.Render(w, r, "login/password", vm)
		return
	}

	// Create session
	if err := h.sessionMgr.CreateSession(w, r, user.ID, user.Role); err != nil {
		h.errLog.Log(r, "failed to create session", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.auditLogger.LogAuthEvent(r, &user.ID, "login_success", true, "")

	// Check if password change is required
	if user.PasswordTemp != nil && *user.PasswordTemp {
		http.Redirect(w, r, "/profile/change-password?required=1", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// EmailLoginVM is the view model for email login.
type EmailLoginVM struct {
	viewdata.BaseVM
	Error string
	Email string
}

// showEmailLogin displays the email login form.
func (h *Handler) showEmailLogin(w http.ResponseWriter, r *http.Request) {
	vm := EmailLoginVM{
		BaseVM: viewdata.New(r),
	}
	vm.Title = "Email Login"

	templates.Render(w, r, "login/email", vm)
}

// handleEmailLogin sends a verification code to the email.
func (h *Handler) handleEmailLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	user, err := h.userStore.GetByEmail(r.Context(), email)
	if err != nil {
		// Don't reveal if email exists
		h.auditLogger.LogAuthEvent(r, nil, "verification_code_sent", true, "")
		http.Redirect(w, r, "/login/email/verify?email="+email, http.StatusSeeOther)
		return
	}

	if user.Status != "active" {
		h.auditLogger.LogAuthEvent(r, &user.ID, "login_failed_user_disabled", false, "user disabled")
		http.Redirect(w, r, "/login/email/verify?email="+email, http.StatusSeeOther)
		return
	}

	// Generate and send verification code
	verification, err := h.emailVerifyStore.Create(r.Context(), email, user.ID)
	if err != nil {
		h.errLog.Log(r, "failed to create verification", err)
		http.Redirect(w, r, "/login/email/verify?email="+email, http.StatusSeeOther)
		return
	}

	// Send email with code
	if h.mailer != nil {
		err = h.mailer.Send(mailer.Email{
			To:       email,
			Subject:  "Your Login Code",
			TextBody: "Your login code is: " + verification.Code + "\n\nOr click here: " + h.baseURL + "/login/email/magic?token=" + verification.Token,
		})
		if err != nil {
			h.errLog.Log(r, "failed to send verification email", err)
		}
	}

	h.auditLogger.LogAuthEvent(r, &user.ID, "verification_code_sent", true, "")

	http.Redirect(w, r, "/login/email/verify?email="+email, http.StatusSeeOther)
}

// EmailVerifyVM is the view model for email verification.
type EmailVerifyVM struct {
	viewdata.BaseVM
	Error string
	Email string
}

// showEmailVerify displays the email verification form.
func (h *Handler) showEmailVerify(w http.ResponseWriter, r *http.Request) {
	vm := EmailVerifyVM{
		BaseVM: viewdata.New(r),
		Email:  r.URL.Query().Get("email"),
	}
	vm.Title = "Verify Email"

	templates.Render(w, r, "login/email_verify", vm)
}

// handleEmailVerify verifies the code and logs in.
func (h *Handler) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.errLog.Log(r, "failed to parse form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	code := r.FormValue("code")

	verification, err := h.emailVerifyStore.VerifyCode(r.Context(), email, code)
	if err != nil {
		h.auditLogger.LogAuthEvent(r, nil, "verification_code_failed", false, "invalid code")

		vm := EmailVerifyVM{
			BaseVM: viewdata.New(r),
			Error:  "Invalid or expired code",
			Email:  email,
		}
		templates.Render(w, r, "login/email_verify", vm)
		return
	}

	user, err := h.userStore.GetByID(r.Context(), verification.UserID)
	if err != nil || user.Status != "active" {
		vm := EmailVerifyVM{
			BaseVM: viewdata.New(r),
			Error:  "Account not found or disabled",
			Email:  email,
		}
		templates.Render(w, r, "login/email_verify", vm)
		return
	}

	// Mark verification as used
	h.emailVerifyStore.MarkUsed(r.Context(), verification.ID)

	// Create session
	if err := h.sessionMgr.CreateSession(w, r, user.ID, user.Role); err != nil {
		h.errLog.Log(r, "failed to create session", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.auditLogger.LogAuthEvent(r, &user.ID, "login_success", true, "")

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleMagicLink handles magic link login.
func (h *Handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	verification, err := h.emailVerifyStore.VerifyToken(r.Context(), token)
	if err != nil {
		h.auditLogger.LogAuthEvent(r, nil, "magic_link_failed", false, "invalid token")
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusSeeOther)
		return
	}

	user, err := h.userStore.GetByID(r.Context(), verification.UserID)
	if err != nil || user.Status != "active" {
		http.Redirect(w, r, "/login?error=account_disabled", http.StatusSeeOther)
		return
	}

	// Mark verification as used
	h.emailVerifyStore.MarkUsed(r.Context(), verification.ID)

	// Create session
	if err := h.sessionMgr.CreateSession(w, r, user.ID, user.Role); err != nil {
		h.errLog.Log(r, "failed to create session", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.auditLogger.LogAuthEvent(r, &user.ID, "magic_link_used", true, "")

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
