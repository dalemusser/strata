# CSRF Protection Implementation Guide

This document outlines the steps needed to implement Cross-Site Request Forgery (CSRF) protection in strata.

---

## Background

### What is CSRF?

CSRF is an attack where a malicious website tricks a user's browser into making unwanted requests to a site where the user is authenticated. The browser automatically includes session cookies, so the target site processes the request as legitimate.

### Example Attack

```html
<!-- Attacker hosts this on evil-site.com -->
<form action="https://your-strata-app.com/settings" method="POST">
  <input type="hidden" name="site_name" value="Hacked!">
  <input type="hidden" name="footer_html" value="Malicious content">
</form>
<script>document.forms[0].submit();</script>
```

If an admin visits this page while logged into strata, the form submits with their valid session cookie.

### Solution

Generate a unique, unpredictable token for each session. Include this token in all forms and validate it on every state-changing request (POST, PUT, DELETE). The attacker cannot guess the token, so their forged requests fail.

---

## Implementation Steps

### Step 1: Add Dependency

```bash
go get github.com/gorilla/csrf
```

Add to `go.mod`:
```
github.com/gorilla/csrf v1.7.2
```

---

### Step 2: Add Configuration

**File:** `internal/app/bootstrap/appconfig.go`

Add a CSRF key field to the config:

```go
type AppConfig struct {
    // ... existing fields

    // CSRFKey is a 32-byte key for CSRF token generation.
    // Must be kept secret and consistent across restarts.
    CSRFKey string `env:"STRATA_CSRF_KEY"`
}
```

**File:** `internal/app/bootstrap/config.go`

Add validation:

```go
func (c *AppConfig) Validate() error {
    // ... existing validation

    if c.Env == "production" && len(c.CSRFKey) < 32 {
        return errors.New("STRATA_CSRF_KEY must be at least 32 characters in production")
    }

    return nil
}
```

---

### Step 3: Add CSRF Middleware

**File:** `internal/app/bootstrap/routes.go`

```go
import (
    "github.com/gorilla/csrf"
    // ... other imports
)

func SetupRoutes(config *AppConfig, deps *DBDeps, ...) http.Handler {
    r := chi.NewRouter()

    // ... existing middleware (CORS, logging, etc.)

    // CSRF protection
    csrfKey := []byte(config.CSRFKey)
    if len(csrfKey) < 32 {
        // Use a default key for development only
        csrfKey = []byte("development-csrf-key-not-for-prod")
    }

    csrfMiddleware := csrf.Protect(
        csrfKey,
        csrf.Secure(config.Env == "production"),  // HTTPS only in production
        csrf.Path("/"),                            // Cookie valid for all paths
        csrf.HttpOnly(true),                       // Cookie not accessible via JavaScript
        csrf.SameSite(csrf.SameSiteStrictMode),   // Strict same-site policy
        csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            http.Error(w, "CSRF token invalid", http.StatusForbidden)
        })),
    )

    r.Use(csrfMiddleware)

    // ... rest of route setup
}
```

---

### Step 4: Update BaseVM

**File:** `internal/app/system/viewdata/viewdata.go`

Add CSRF token to the base view model:

```go
import (
    "github.com/gorilla/csrf"
    // ... other imports
)

type BaseVM struct {
    // ... existing fields

    // CSRFToken is the CSRF token for form submissions
    CSRFToken string

    // CSRFTokenField is the HTML input field (convenience helper)
    CSRFTokenField template.HTML
}

func New(r *http.Request) BaseVM {
    return BaseVM{
        // ... existing field initialization

        CSRFToken:      csrf.Token(r),
        CSRFTokenField: csrf.TemplateField(r),
    }
}

func NewBaseVM(r *http.Request, db *mongo.Database, title, backURL string) BaseVM {
    vm := New(r)
    // ... existing initialization
    return vm
}
```

---

### Step 5: Update Templates

#### Option A: Use CSRFTokenField Helper

The `csrf.TemplateField(r)` returns a complete hidden input element:

```html
<form method="POST" action="/settings">
    {{.CSRFTokenField}}
    <!-- rest of form -->
</form>
```

#### Option B: Use CSRFToken Value

For more control over the markup:

```html
<form method="POST" action="/settings">
    <input type="hidden" name="gorilla.csrf.Token" value="{{.CSRFToken}}">
    <!-- rest of form -->
</form>
```

#### Templates to Update

Search for all forms with `method="POST"` (or PUT/DELETE):

| Template File | Forms |
|--------------|-------|
| `login/templates/identify.gohtml` | Login form |
| `login/templates/password.gohtml` | Password form |
| `login/templates/email.gohtml` | Email login form |
| `login/templates/email_code.gohtml` | Code verification form |
| `login/templates/trust.gohtml` | Trust login form |
| `login/templates/reset_password.gohtml` | Password reset form |
| `settings/templates/show.gohtml` | Settings form |
| `pages/templates/edit.gohtml` | Page edit form |
| `systemusers/templates/new.gohtml` | New user form |
| `systemusers/templates/edit.gohtml` | Edit user form |
| `profile/templates/show.gohtml` | Profile form |
| `invitations/templates/new.gohtml` | New invitation form |
| `invitations/templates/accept.gohtml` | Accept invitation form |
| `announcements/templates/new.gohtml` | New announcement form |
| `announcements/templates/edit.gohtml` | Edit announcement form |
| `home/templates/edit.gohtml` | Home page edit form |

---

### Step 6: Handle HTMX Requests

HTMX sends requests via JavaScript, so the token must be included in request headers.

#### Option A: Global HTMX Headers

Add to your base layout template (`layout.gohtml`):

```html
<body hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'>
```

This automatically includes the CSRF token in all HTMX requests.

#### Option B: Per-Request Headers

For specific HTMX elements:

```html
<button hx-post="/api/action"
        hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'>
    Action
</button>
```

#### Configure Gorilla CSRF to Check Headers

By default, gorilla/csrf checks both form values and the `X-CSRF-Token` header, so no additional configuration is needed.

---

### Step 7: Handle AJAX/Fetch Requests

If you have any custom JavaScript making POST requests:

```javascript
fetch('/api/endpoint', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]').content
    },
    body: JSON.stringify(data)
});
```

Add a meta tag to your layout for JavaScript access:

```html
<head>
    <meta name="csrf-token" content="{{.CSRFToken}}">
</head>
```

---

### Step 8: Exempt Safe Routes (Optional)

Some routes don't need CSRF protection (webhooks, API endpoints with token auth):

```go
// In routes.go
r.Group(func(r chi.Router) {
    r.Use(csrf.Protect(key))
    // Protected routes
})

r.Group(func(r chi.Router) {
    // Unprotected routes (webhooks, etc.)
    r.Post("/webhooks/stripe", stripeHandler)
})
```

Or use the `csrf.UnsafeMethods` option to customize which methods require protection.

---

## Testing

### Manual Testing

1. Submit a form normally - should succeed
2. Modify the CSRF token in browser dev tools - should fail with 403
3. Remove the CSRF token field - should fail with 403
4. Submit via curl without token - should fail with 403

### Automated Testing

For tests that make POST requests, you need to include a valid CSRF token.

**File:** `internal/testutil/csrf.go`

```go
package testutil

import (
    "net/http"
    "net/http/httptest"

    "github.com/gorilla/csrf"
)

// WithCSRFToken adds a valid CSRF token to the request for testing.
// This requires the test to use a handler wrapped with CSRF middleware.
func WithCSRFToken(r *http.Request, handler http.Handler) *http.Request {
    // Get a token by making a GET request first
    rec := httptest.NewRecorder()
    getReq := httptest.NewRequest("GET", r.URL.Path, nil)
    handler.ServeHTTP(rec, getReq)

    // Extract token from response cookies
    for _, cookie := range rec.Result().Cookies() {
        r.AddCookie(cookie)
    }

    // Add token to form
    token := csrf.Token(getReq)
    // Add to request...

    return r
}
```

Alternatively, disable CSRF in test environment:

```go
if config.Env == "test" {
    // Skip CSRF middleware in tests
} else {
    r.Use(csrfMiddleware)
}
```

---

## Environment Variables

Add to your deployment configuration:

```bash
# Production - use a secure random 32+ character string
STRATA_CSRF_KEY=your-32-character-secret-key-here

# Generate a secure key:
# openssl rand -base64 32
```

---

## Checklist

- [ ] Add `github.com/gorilla/csrf` dependency
- [ ] Add `STRATA_CSRF_KEY` to AppConfig
- [ ] Add CSRF middleware to routes.go
- [ ] Update BaseVM with CSRFToken fields
- [ ] Update layout.gohtml with HTMX headers
- [ ] Update all POST form templates (15+ files)
- [ ] Add meta tag for JavaScript access (if needed)
- [ ] Update tests or add test bypass
- [ ] Add STRATA_CSRF_KEY to deployment environment
- [ ] Test all forms manually
- [ ] Document the new environment variable

---

## References

- [Gorilla CSRF Documentation](https://github.com/gorilla/csrf)
- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [HTMX and CSRF](https://htmx.org/docs/#security)
