package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"detent.build/internal/config"
	"detent.build/internal/content"
	"detent.build/internal/middleware"

	"github.com/labstack/echo/v4"
)

func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()

	cfg := &config.Config{
		Port: "3000",
		Env:  "test",
		Site: config.SiteConfig{Name: "detent.build", URL: "https://detent.build"},
	}

	e := echo.New()
	middleware.Setup(e, cfg)
	New(cfg).RegisterRoutes(e)
	return e
}

func get(t *testing.T, e *echo.Echo, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestPagesRender(t *testing.T) {
	e := newTestServer(t)

	tests := []struct {
		path string
		want string
	}{
		{"/", "The gates decide when it lands."},
		{"/how-it-works", "The mechanism, stop by stop."},
		{"/why-detent", "You are not steering an agent."},
		{"/dashboard", "The board stays honest."},
		{"/install", "No service to stand up."},
		{"/open-source", "No control plane."},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := get(t, e, tt.path, nil)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("body does not contain %q", tt.want)
			}
		})
	}
}

func TestInstallPlatforms(t *testing.T) {
	e := newTestServer(t)

	for _, target := range content.InstallTargets {
		t.Run(target.Key, func(t *testing.T) {
			rec := get(t, e, "/install/"+target.Key, nil)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if !strings.Contains(rec.Body.String(), target.Primary.Cmd) {
				t.Errorf("body does not contain the primary command %q", target.Primary.Cmd)
			}
		})
	}
}

// An HTMX tab click must swap the whole tab strip, not just the panel, or the
// selected tab stays wrong after the swap.
func TestInstallPlatformHTMXSwapsTabStrip(t *testing.T) {
	e := newTestServer(t)

	rec := get(t, e, "/install/windows", map[string]string{"HX-Request": "true"})
	body := rec.Body.String()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX response returned a full page instead of a fragment")
	}
	if !strings.Contains(body, `id="install-tabs"`) {
		t.Error("HTMX response does not include the tab strip, so the selected tab cannot update")
	}
	if !strings.Contains(body, `href="/install/windows" hx-get="/install/windows" hx-target="#install-tabs" hx-swap="outerHTML" hx-push-url="true" aria-selected="true"`) {
		t.Error("the Windows tab is not marked selected in the swapped fragment")
	}
}

func TestInstallPlatformUnknown(t *testing.T) {
	e := newTestServer(t)

	rec := get(t, e, "/install/plan9", nil)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHealth(t *testing.T) {
	e := newTestServer(t)

	rec := get(t, e, "/health", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("body = %q, want the ok payload", got)
	}
}

// detent.build has no www host and the domain is HSTS-preloaded, so no page
// may emit a www hostname or an http:// URL for the production host.
func TestNoWWWAndNoPlainHTTPForProductionHost(t *testing.T) {
	e := newTestServer(t)

	paths := []string{"/", "/how-it-works", "/why-detent", "/dashboard", "/install", "/open-source", "/sitemap.xml"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body := get(t, e, path, nil).Body.String()

			if strings.Contains(body, "www.detent.build") {
				t.Error("page references www.detent.build, which does not resolve")
			}
			if strings.Contains(body, "http://detent.build") {
				t.Error("page emits a plain-http URL for an HSTS-preloaded host")
			}
		})
	}
}

func TestSitemapUsesCanonicalApexURLs(t *testing.T) {
	e := newTestServer(t)

	rec := get(t, e, "/sitemap.xml", nil)
	body := rec.Body.String()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{
		"<loc>https://detent.build/</loc>",
		"<loc>https://detent.build/how-it-works</loc>",
		"<loc>https://detent.build/open-source</loc>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sitemap is missing %s", want)
		}
	}
	if strings.Contains(body, "detent.build//") {
		t.Error("sitemap contains a doubled slash from a trailing-slash site URL")
	}
}

// The site URL is configured with no trailing slash in production, but a
// stray one must not corrupt the sitemap.
func TestSitemapTolerantOfTrailingSlashConfig(t *testing.T) {
	cfg := &config.Config{
		Port: "3000",
		Env:  "test",
		Site: config.SiteConfig{Name: "detent.build", URL: "https://detent.build/"},
	}
	e := echo.New()
	middleware.Setup(e, cfg)
	New(cfg).RegisterRoutes(e)

	body := get(t, e, "/sitemap.xml", nil).Body.String()

	if strings.Contains(body, "detent.build//") {
		t.Error("trailing slash in SITE_URL produced a doubled slash")
	}
}

// Traefik terminates TLS and already redirects. An app-level redirect would
// loop, so no route may answer with a 3xx.
func TestNoAppLevelRedirects(t *testing.T) {
	e := newTestServer(t)

	for _, path := range []string{"/", "/install", "/install/linux", "/health", "/sitemap.xml"} {
		t.Run(path, func(t *testing.T) {
			if code := get(t, e, path, nil).Code; code >= 300 && code < 400 {
				t.Errorf("status = %d; the app must not redirect, Traefik already does", code)
			}
		})
	}
}

func TestDefaultPortIsThreeThousand(t *testing.T) {
	t.Setenv("PORT", "")

	if got := config.Load().Port; got != "3000" {
		t.Errorf("default port = %q, want 3000; the Dokploy domain entry is bound to 3000", got)
	}
}
