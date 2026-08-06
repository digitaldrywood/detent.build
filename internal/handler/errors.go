package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"detent.build/templates/pages"

	"github.com/labstack/echo/v4"
)

// ErrorHandler renders HTML errors for browsers and JSON for everything else.
// Echo's default answers every error with a JSON body, which is the wrong
// content type for a mistyped URL on a marketing site.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		status = he.Code
	}

	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "error", err, "path", c.Request().URL.Path, "status", status)
	}

	if !wantsHTML(c.Request()) {
		if jsonErr := c.JSON(status, map[string]string{"error": http.StatusText(status)}); jsonErr != nil {
			slog.Error("write json error response", "error", jsonErr)
		}
		return
	}

	heading, body := errorCopy(status)
	if renderErr := render(c, status, pages.Error(status, heading, body)); renderErr != nil {
		slog.Error("render error page", "error", renderErr)
	}
}

// wantsHTML reports whether the client is a browser navigating, rather than a
// fetch for JSON or an asset.
func wantsHTML(r *http.Request) bool {
	if r.Method == http.MethodHead {
		return true
	}
	accept := r.Header.Get("Accept")
	if accept == "" {
		return true
	}
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return false
	}
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

func errorCopy(status int) (heading, body string) {
	switch status {
	case http.StatusNotFound:
		return "No stop by that name.",
			"That page is not on this site. Nothing is held here — the URL is simply wrong."
	case http.StatusInternalServerError:
		return "The server dropped it.",
			"Something failed on our side. The site is a static binary, so a reload usually gets a different answer."
	default:
		return http.StatusText(status),
			"That request could not be served."
	}
}
