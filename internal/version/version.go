package version

import (
	"fmt"
	"runtime"
)

const (
	// Version is the current version of CharityLens
	Version = "0.1.0"

	// ProjectURL is the project homepage
	ProjectURL = "https://github.com/matthewgall/charitylens"

	// ContactEmail for API usage questions
	ContactEmail = "crawler@charitylens.org"
)

// UserAgent returns a properly formatted User-Agent string for API requests
func UserAgent() string {
	return fmt.Sprintf("charitylens/%s (%s; %s/%s; +%s; %s)",
		Version,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		ProjectURL,
		ContactEmail,
	)
}

// GetVersion returns the current version
func GetVersion() string {
	return Version
}
