package handler

import (
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"detent.build/internal/config"
	"detent.build/internal/content"
	"detent.build/internal/docs"
	"detent.build/internal/middleware"

	"github.com/labstack/echo/v4"
)

// newTestServerWithSiteURL builds the server through config.Load() rather than
// a hand-written Config, so the tests exercise the same defaults production
// gets and cannot silently drift from them.
func newTestServerWithSiteURL(t *testing.T, siteURL string) *echo.Echo {
	t.Helper()

	t.Setenv("PORT", "")
	t.Setenv("ENV", "test")
	t.Setenv("SITE_NAME", "detent.build")
	t.Setenv("SITE_URL", siteURL)
	t.Setenv("DEFAULT_OG_IMAGE", "")

	cfg := config.Load()

	e := echo.New()
	middleware.Setup(e, cfg, nil)
	New(cfg).RegisterRoutes(e)
	return e
}

func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()
	return newTestServerWithSiteURL(t, "https://detent.build")
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
		{"/docs", "Documentation, pinned to the source."},
		{"/docs/getting-started", "Quick Start"},
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

func TestPublishedDocumentationRoutesShowPinnedSource(t *testing.T) {
	e := newTestServer(t)

	for _, page := range docs.Published.Pages() {
		t.Run(page.PublicPath, func(t *testing.T) {
			rec := get(t, e, page.PublicPath, nil)
			body := rec.Body.String()
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			for _, want := range []string{docs.ReleaseTag, docs.ShortCommit(), page.SourceURL, "Mirrored upstream"} {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
		})
	}
}

func TestUnregisteredDocumentationIsNotServed(t *testing.T) {
	e := newTestServer(t)

	for _, path := range []string{
		"/docs/comparison",
		"/docs/development",
		"/docs/parity-audit",
		"/docs/getting-started.md",
		"/docs/templates/detent.label.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			if code := get(t, e, path, nil).Code; code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", code, http.StatusNotFound)
			}
		})
	}
}

func TestDocsNavigationUsesLocalIndex(t *testing.T) {
	body := get(t, newTestServer(t), "/docs", nil).Body.String()
	if !strings.Contains(body, `href="/docs"`) {
		t.Error("documentation navigation does not link to the local index")
	}
	if strings.Contains(body, `Docs ↗`) {
		t.Error("documentation navigation is still marked as external")
	}
}

func TestWorkflowStatesArePresentedAsConfigurable(t *testing.T) {
	e := newTestServer(t)

	tests := []struct {
		path      string
		want      []string
		forbidden []string
	}{
		{
			path: "/",
			want: []string{
				html.EscapeString(content.BoardCaption),
				content.NonCodeWorkflowURL,
				`content="` + content.OpenGraphDefaultImageAlt + `"`,
			},
			forbidden: []string{"one item at every stop", "The six board lanes", "the gates decide when it lands. The six"},
		},
		{
			path:      "/dashboard",
			want:      []string{html.EscapeString(content.DashboardLaneHeading), content.NonCodeWorkflowURL},
			forbidden: []string{"The same six stops, live."},
		},
		{
			path:      "/how-it-works",
			want:      []string{content.HowItWorksStateHeading, html.EscapeString(content.HowItWorksStateDetail)},
			forbidden: []string{"Six stops, and the catches between them."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			body := get(t, e, tt.path, nil).Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(body, forbidden) {
					t.Errorf("body contains fixed-state wording %q", forbidden)
				}
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
	if !strings.Contains(body, `hx-get="/install/windows"`) {
		t.Error("the swapped fragment does not contain the Windows link")
	}
	// These are navigation links, not an ARIA tabs widget: aria-current marks
	// the selected one, and only one link may carry it.
	if got := strings.Count(body, `aria-current="page"`); got != 1 {
		t.Errorf("aria-current appears %d times in the swapped fragment, want exactly 1", got)
	}
	if !strings.Contains(body, `href="/install/windows" hx-get="/install/windows" hx-target="#install-tabs" hx-swap="outerHTML" hx-push-url="true" aria-current="page"`) {
		t.Error("the Windows link is not the one marked aria-current after the swap")
	}
	if strings.Contains(body, `role="tab"`) {
		t.Error("role=tab is back without the tabpanel, roving tabindex, and arrow-key model it promises")
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

	paths := []string{"/", "/how-it-works", "/why-detent", "/dashboard", "/install", "/open-source", "/docs", "/docs/getting-started", "/sitemap.xml"}

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

// Canonicals and og:url resolve from SITE_URL rather than being hard-coded,
// and the card image must be absolute or crawlers cannot fetch it.
func TestCanonicalAndOpenGraphAreAbsoluteApexURLs(t *testing.T) {
	e := newTestServer(t)

	tests := []struct {
		path string
		want string
	}{
		{"/", "https://detent.build/"},
		{"/how-it-works", "https://detent.build/how-it-works"},
		{"/why-detent", "https://detent.build/why-detent"},
		{"/dashboard", "https://detent.build/dashboard"},
		{"/install", "https://detent.build/install"},
		{"/open-source", "https://detent.build/open-source"},
		{"/docs", "https://detent.build/docs"},
		{"/docs/getting-started", "https://detent.build/docs/getting-started"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			body := get(t, e, tt.path, nil).Body.String()

			if want := `<link rel="canonical" href="` + tt.want + `">`; !strings.Contains(body, want) {
				t.Errorf("missing canonical %s", tt.want)
			}
			if want := `<meta property="og:url" content="` + tt.want + `">`; !strings.Contains(body, want) {
				t.Errorf("missing og:url %s", tt.want)
			}
			if want := `<meta property="og:image" content="https://detent.build/static/images/og-default.png">`; !strings.Contains(body, want) {
				t.Error("og:image is missing or not an absolute URL")
			}
		})
	}
}

// Every install tab is the same page with a different tab selected, so they
// must all point at /install rather than competing for the index.
func TestInstallTabsShareOneCanonical(t *testing.T) {
	e := newTestServer(t)

	for _, target := range content.InstallTargets {
		t.Run(target.Key, func(t *testing.T) {
			body := get(t, e, "/install/"+target.Key, nil).Body.String()

			if !strings.Contains(body, `<link rel="canonical" href="https://detent.build/install">`) {
				t.Errorf("/install/%s does not canonicalize to /install", target.Key)
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
		"<loc>https://detent.build/docs</loc>",
		"<loc>https://detent.build/docs/getting-started</loc>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sitemap is missing %s", want)
		}
	}
	if strings.Contains(body, "detent.build//") {
		t.Error("sitemap contains a doubled slash from a trailing-slash site URL")
	}
}

func TestSitemapIncludesEveryIndexablePageRoute(t *testing.T) {
	e := newTestServer(t)
	body := get(t, e, "/sitemap.xml", nil).Body.String()

	nonIndexableRoutes := map[string]struct{}{
		// Health is a machine-readable availability endpoint, not a page.
		"/health": {},
		// Robots directives are crawler configuration, not page content.
		"/robots.txt": {},
		// The sitemap describes pages but is not itself an indexable page.
		"/sitemap.xml": {},
		// Install platform routes are HTMX partials canonicalized to /install.
		"/install/:platform": {},
		// Static files support pages but are not independently indexable pages.
		"/static*": {},
		// templUI assets support interactions but are not page content.
		"/assets*": {},
	}

	for _, route := range e.Routes() {
		if route.Method != http.MethodGet {
			continue
		}
		if _, excluded := nonIndexableRoutes[route.Path]; excluded {
			continue
		}

		want := "<loc>https://detent.build" + route.Path + "</loc>"
		if !strings.Contains(body, want) {
			t.Errorf("registered page route %q is missing from the sitemap", route.Path)
		}
	}
}

// The site URL is configured with no trailing slash in production, but a
// stray one must not corrupt the sitemap.
func TestSitemapTolerantOfTrailingSlashConfig(t *testing.T) {
	e := newTestServerWithSiteURL(t, "https://detent.build/")

	body := get(t, e, "/sitemap.xml", nil).Body.String()

	if strings.Contains(body, "detent.build//") {
		t.Error("trailing slash in SITE_URL produced a doubled slash")
	}
}

// A trailing slash in SITE_URL must not leak into canonicals or the card URL
// either — a doubled slash is a different URL to a crawler.
func TestCanonicalTolerantOfTrailingSlashConfig(t *testing.T) {
	e := newTestServerWithSiteURL(t, "https://detent.build/")

	body := get(t, e, "/how-it-works", nil).Body.String()

	if strings.Contains(body, "detent.build//") {
		t.Error("trailing slash in SITE_URL produced a doubled slash in page metadata")
	}
	if !strings.Contains(body, `<link rel="canonical" href="https://detent.build/how-it-works">`) {
		t.Error("canonical is wrong when SITE_URL carries a trailing slash")
	}
}

// Traefik terminates TLS and already redirects. An app-level redirect would
// loop, so no route may answer with a 3xx.
func TestNoAppLevelRedirects(t *testing.T) {
	e := newTestServer(t)

	for _, path := range []string{"/", "/install", "/install/linux", "/docs", "/docs/getting-started", "/health", "/sitemap.xml"} {
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
