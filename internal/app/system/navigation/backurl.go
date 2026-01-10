// Package navigation provides helpers for safe URL navigation and redirects.
package navigation

import (
	"net/http"
	"strings"

	"github.com/dalemusser/waffle/pantry/query"
	"github.com/dalemusser/waffle/pantry/urlutil"
)

// BackURLOptions configures the behavior of SafeBackURL.
type BackURLOptions struct {
	// AllowedPrefix is the required URL prefix (e.g., "/system-users").
	// If empty, any safe URL is allowed.
	AllowedPrefix string

	// ExcludedSubpaths are subpath patterns to reject (e.g., "/edit", "/delete", "/new").
	// These prevent redirect loops back to action pages.
	ExcludedSubpaths []string

	// Fallback is the default URL if no valid return URL is found.
	Fallback string
}

// SafeBackURL extracts and validates a return URL from the request.
//
// It checks both the query parameter and form value for "return", validates
// the URL is safe (not an open redirect), optionally validates the prefix,
// and excludes specified subpaths to prevent redirect loops.
//
// Example usage:
//
//	url := navigation.SafeBackURL(r, navigation.BackURLOptions{
//	    AllowedPrefix:    "/system-users",
//	    ExcludedSubpaths: []string{"/edit", "/delete", "/new"},
//	    Fallback:         "/system-users",
//	})
func SafeBackURL(r *http.Request, opts BackURLOptions) string {
	// Try query parameter first, then form value
	ret := urlutil.SafeReturn(query.Get(r, "return"), "", "")
	if ret == "" {
		ret = urlutil.SafeReturn(strings.TrimSpace(r.FormValue("return")), "", "")
	}

	// Validate against allowed prefix if specified
	if ret != "" {
		valid := true

		if opts.AllowedPrefix != "" && !strings.HasPrefix(ret, opts.AllowedPrefix) {
			valid = false
		}

		// Check excluded subpaths
		for _, excluded := range opts.ExcludedSubpaths {
			if strings.Contains(ret, excluded) {
				valid = false
				break
			}
		}

		if valid {
			return ret
		}
	}

	return opts.Fallback
}

// Common back URL configurations for reuse across packages.
var (
	// SystemUsersBackURL returns options for system-users pages.
	SystemUsersBackURL = BackURLOptions{
		AllowedPrefix:    "/system-users",
		ExcludedSubpaths: []string{"/edit", "/delete", "/new"},
		Fallback:         "/system-users",
	}

	// PagesBackURL returns options for pages management.
	PagesBackURL = BackURLOptions{
		AllowedPrefix:    "/pages",
		ExcludedSubpaths: []string{"/edit"},
		Fallback:         "/dashboard",
	}
)
