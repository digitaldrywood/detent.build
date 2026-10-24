package handler

import (
	"net/http"
	"regexp"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

var fullCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// deploymentCommit attests only clean Git builds, never an environment variable
// or the Detent product version displayed by the marketing site.
func deploymentCommit(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}
	var vcs, revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs":
			vcs = setting.Value
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	if vcs != "git" || modified != "false" || !fullCommit.MatchString(revision) {
		return ""
	}
	return revision
}

// Deployment lets release CI identify the binary serving production. Unknown
// build provenance fails closed without affecting the availability health check.
func (h *Handler) Deployment(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	if h.commit == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "clean build commit unavailable"})
	}
	return c.JSON(http.StatusOK, map[string]string{"commit": h.commit})
}
