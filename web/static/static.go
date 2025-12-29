package static

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed css/*.css js/*.js
var staticFS embed.FS

var isDevelopment bool

func init() {
	isDevelopment = os.Getenv("GO_ENV") == "development"
}

// FS returns the static filesystem - either embedded or on-disk based on GO_ENV
func FS() fs.FS {
	if isDevelopment {
		// In development mode, serve files directly from disk
		// This allows hot-reloading of CSS/JS without rebuilding
		return os.DirFS("web/static")
	}

	// In production, serve from embedded filesystem
	return staticFS
}
