// internal/app/resources/resources.go
package resources

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed templates/*.gohtml templates/**/*.gohtml
var templatesFS embed.FS

//go:embed assets/*
var assetsFS embed.FS

// Templates returns the embedded templates filesystem.
func Templates() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic("failed to get templates subdirectory: " + err.Error())
	}
	return sub
}

// Assets returns the embedded assets filesystem.
func Assets() fs.FS {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic("failed to get assets subdirectory: " + err.Error())
	}
	return sub
}

// ParseTemplates parses all templates and returns a template set.
func ParseTemplates() (*template.Template, error) {
	return template.ParseFS(templatesFS, "templates/*.gohtml", "templates/**/*.gohtml")
}

// LoadSharedTemplates loads shared templates into the template registry.
// This is called during startup to make layout and common templates available.
func LoadSharedTemplates() {
	// Templates are loaded automatically by the waffle template engine
	// when it boots. This function exists for compatibility but the actual
	// loading happens in BuildHandler when templates.Boot is called.
}

// AssetsHandler returns an http.Handler that serves embedded assets.
// The prefix is stripped from the request path before looking up files.
func AssetsHandler(prefix string) http.Handler {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic("failed to get assets subdirectory: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the prefix from the path
		path := strings.TrimPrefix(r.URL.Path, prefix)
		path = strings.TrimPrefix(path, "/")

		r.URL.Path = "/" + path
		fileServer.ServeHTTP(w, r)
	})
}
