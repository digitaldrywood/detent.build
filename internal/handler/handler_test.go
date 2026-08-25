package handler

import (
	"encoding/xml"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"detent.build/internal/config"
	"detent.build/internal/content"
	"detent.build/internal/docs"
	"detent.build/internal/docsregistry"
	"detent.build/internal/middleware"
	"detent.build/internal/videos"

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
		{"/", "Manage work, not agents."},
		{"/how-it-works", "The mechanism, stop by stop."},
		{"/why-detent", "You are not steering an agent."},
		{"/dashboard", "The board stays honest."},
		{"/install", "No service to stand up."},
		{"/open-source", "No control plane."},
		{"/videos", "Detent, moving through real work."},
		{"/docs", "Documentation, pinned to the source."},
		{"/docs/getting-started", "Quick Start"},
		{"/docs/site/project-contracts", "Project contracts"},
		{"/docs/site/working-checkout-merge-gate", "The working-checkout merge-gate trap"},
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

func TestVideoPlacementsRenderFromManifest(t *testing.T) {
	e := newTestServer(t)

	tests := []struct {
		path      string
		want      []string
		forbidden []string
	}{
		{
			path: "/",
			want: []string{
				"It ships itself.",
				"I Opened Seven Issues for One Feature (On Purpose)",
				`src="/static/images/videos/WUob6ZTzqqk.webp"`,
			},
			forbidden: []string{"My AI Orchestrator Is Dumb on Purpose"},
		},
		{
			path: "/videos",
			want: []string{
				"I Opened Seven Issues for One Feature (On Purpose)",
				"My AI Orchestrator Is Dumb on Purpose",
				`href="https://www.youtube.com/watch?v=WUob6ZTzqqk"`,
				`href="https://www.youtube.com/watch?v=QvED7RHTKkI"`,
			},
		},
		{
			path: "/docs/configuration",
			want: []string{
				"Related walkthrough",
				"My AI Orchestrator Is Dumb on Purpose",
				`src="/static/images/videos/QvED7RHTKkI.webp"`,
			},
			forbidden: []string{"I Opened Seven Issues for One Feature (On Purpose)"},
		},
		{
			path:      "/docs/getting-started",
			forbidden: []string{"Related walkthrough", "My AI Orchestrator Is Dumb on Purpose"},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			rec := get(t, e, test.path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			body := rec.Body.String()
			for _, want := range test.want {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(body, forbidden) {
					t.Errorf("body contains %q", forbidden)
				}
			}
		})
	}
}

func TestVideoSurfacesNeverLoadThirdPartyResources(t *testing.T) {
	e := newTestServer(t)
	remoteResource := regexp.MustCompile(`(?:src|srcset)="https?://`)
	youTubeLink := regexp.MustCompile(`href="https://www\.youtube\.com/[^"]+"`)
	approvedWatchURLs := make(map[string]struct{})
	for _, entry := range videos.Published.Entries() {
		approvedWatchURLs[`href="`+entry.WatchURL+`"`] = struct{}{}
	}

	for _, path := range []string{"/", "/videos", "/docs/configuration"} {
		t.Run(path, func(t *testing.T) {
			body := get(t, e, path, nil).Body.String()
			if remoteResource.MatchString(body) {
				t.Error("page loads a third-party resource through src or srcset")
			}
			for _, forbidden := range []string{"<iframe", "autoplay", "youtube.com/embed", "youtu.be/", "i.ytimg.com"} {
				if strings.Contains(body, forbidden) {
					t.Errorf("page contains forbidden video integration %q", forbidden)
				}
			}
			for _, match := range youTubeLink.FindAllString(body, -1) {
				if _, approved := approvedWatchURLs[match]; !approved {
					t.Errorf("page contains unapproved YouTube link %q", match)
				}
			}
		})
	}
}

func TestContentSecurityPolicyRemainsExact(t *testing.T) {
	want := "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
		"style-src 'self'; style-src-attr 'unsafe-inline'; " +
		"img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; " +
		"base-uri 'self'; form-action 'self'; frame-ancestors 'self'"

	for _, path := range []string{"/", "/videos", "/docs/configuration"} {
		rec := get(t, newTestServer(t), path, nil)
		if got := rec.Header().Get("Content-Security-Policy"); got != want {
			t.Errorf("%s Content-Security-Policy = %q, want %q", path, got, want)
		}
	}
}

func TestHomeOwnsSymphonyLineage(t *testing.T) {
	e := newTestServer(t)
	rec := get(t, e, "/", nil)
	body := html.UnescapeString(rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{
		`"Manage work, not agents" is OpenAI Symphony's phrase.`,
		"Symphony is an Apache-2.0 ",
		"SPEC.md",
		"plus an Elixir reference implementation that polls a Linear board.",
		"GitHub-native state",
		"boardless issue-field mode",
		"boardless label mode",
		"github_local",
		"Detent began as an Elixir/OTP implementation adapted from Symphony's Linear target to GitHub Projects v2.",
		`href="https://github.com/openai/symphony"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}

	lineageAt := strings.Index(body, "Symphony named the thesis")
	inversionAt := strings.Index(body, "A system, not an agent")
	if lineageAt < 0 || inversionAt < 0 || lineageAt >= inversionAt {
		t.Errorf("lineage section must appear before the inversion: lineage=%d inversion=%d", lineageAt, inversionAt)
	}
}

func TestLineageMachineLiteralsUseMono(t *testing.T) {
	e := newTestServer(t)

	for _, path := range []string{"/", "/open-source"} {
		t.Run(path, func(t *testing.T) {
			rec := get(t, e, path, nil)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			for _, literal := range []string{"SPEC.md", "ProjectV2", "github_local", "detent doctor"} {
				want := `<span class="font-mono text-xs">` + literal + `</span>`
				if !strings.Contains(rec.Body.String(), want) {
					t.Errorf("body does not contain %q", want)
				}
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
			wants := []string{page.SourceURL}
			if page.Origin == docs.OriginSite {
				wants = append(wants, "detent.build guide", "Hype Markdown", "Provenance")
			} else {
				wants = append(wants, docs.ReleaseTag, docs.ShortCommit(), "Mirrored upstream")
			}
			for _, want := range wants {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
		})
	}
}

func TestDocumentationAliasAndTombstoneRoutes(t *testing.T) {
	catalog, err := docs.NewCatalog(fstest.MapFS{
		"renamed.md": {Data: []byte("# Renamed documentation\n")},
	}, docsregistry.Registry{
		Pages: []docsregistry.Page{{
			Group:      "Reference",
			Title:      "Renamed documentation",
			SourcePath: "renamed.md",
			PublicPath: "/docs/renamed",
			Origin:     docsregistry.OriginUpstream,
		}},
		Aliases:    []docsregistry.Alias{{PublicPath: "/docs/old-name", CanonicalPath: "/docs/renamed"}},
		Tombstones: []docsregistry.Tombstone{{PublicPath: "/docs/withdrawn"}},
	})
	if err != nil {
		t.Fatalf("build documentation catalog: %v", err)
	}

	t.Setenv("ENV", "test")
	t.Setenv("SITE_URL", "https://detent.build")
	cfg := config.Load()
	e := echo.New()
	middleware.Setup(e, cfg, nil)
	h := New(cfg)
	h.docs = catalog
	h.RegisterRoutes(e)

	alias := get(t, e, "/docs/old-name", nil)
	if alias.Code != http.StatusOK {
		t.Fatalf("alias status = %d, want %d", alias.Code, http.StatusOK)
	}
	if want := `<link rel="canonical" href="https://detent.build/docs/renamed">`; !strings.Contains(alias.Body.String(), want) {
		t.Errorf("alias is missing canonical %s", want)
	}
	if code := get(t, e, "/docs/withdrawn", nil).Code; code != http.StatusGone {
		t.Errorf("tombstone status = %d, want %d", code, http.StatusGone)
	}
	for _, code := range []int{alias.Code, get(t, e, "/docs/withdrawn", nil).Code} {
		if code >= 300 && code < 400 {
			t.Errorf("retired documentation route redirected with status %d", code)
		}
	}
	sitemap := get(t, e, "/sitemap.xml", nil).Body.String()
	for _, retired := range []string{"/docs/old-name", "/docs/withdrawn"} {
		if strings.Contains(sitemap, retired) {
			t.Errorf("sitemap includes retired path %q", retired)
		}
	}
	assertSitemapIncludesEveryIndexablePageRoute(t, e, catalog)
	assertSitemapContainsOnlyIndexablePageRoutes(t, e, catalog)
}

func TestDocumentationVersionAvailabilityIsExplicitAndDistinctFromSourcePin(t *testing.T) {
	e := newTestServer(t)

	tests := []struct {
		path      string
		featureID string
		prefix    string
		version   string
	}{
		{"/docs/workflow-overlays", "machine-local-workflow-overlays", "Introduced in", "v0.43.0"},
		{"/docs/scheduled-operations", "scheduled-maintenance-routines", "Introduced in", "v0.46.0"},
		{"/docs/webhook-freshness", "per-project-github-webhook-freshness", "Available in", "v0.27.0"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			body := get(t, e, test.path, nil).Body.String()
			for _, want := range []string{`data-version-feature="` + test.featureID + `"`, test.prefix, test.version, "Docs source", docs.ReleaseTag} {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
		})
	}

	unrecorded := get(t, e, "/docs/configuration", nil).Body.String()
	if strings.Contains(unrecorded, "data-version-feature=") {
		t.Error("documentation without an explicit record rendered a version badge")
	}
}

func TestUnregisteredDocumentationIsNotServed(t *testing.T) {
	e := newTestServer(t)

	for _, path := range []string{
		"/docs/comparison",
		"/docs/development",
		"/docs/parity-audit",
		"/docs/getting-started.md",
		"/docs/site/missing",
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

func TestHomeExplainsDefiniteStatesBetweenHeroAndInversion(t *testing.T) {
	body := get(t, newTestServer(t), "/", nil).Body.String()

	previous := strings.Index(body, content.HomeHeadline)
	for _, want := range []string{
		content.DefiniteStatesHeading,
		content.DefiniteStatesWhereQuestion,
		content.DefiniteStatesWaitQuestion,
		content.DefiniteStatesMoveQuestion,
		content.DefiniteStatesLaneRail,
		"A system, not an agent.",
	} {
		position := strings.Index(body, html.EscapeString(want))
		if position == -1 {
			t.Errorf("body does not contain %q", want)
		}
		if position <= previous {
			t.Errorf("%q is out of order on the home page", want)
		}
		previous = position
	}

	for _, want := range []string{
		content.DefiniteStatesWhereAutonomy,
		content.DefiniteStatesWhereDetent,
		content.DefiniteStatesWaitAutonomy,
		content.DefiniteStatesWaitDetent,
		content.DefiniteStatesMoveAutonomy,
		content.DefiniteStatesMoveDetent,
		content.DetentREADMEURL,
		content.Doc("concepts.md"),
	} {
		if !strings.Contains(body, html.EscapeString(want)) {
			t.Errorf("body does not contain %q", want)
		}
	}

	previous = 0
	for _, want := range []string{"02", "03", "04", "05", "06", "07", "08", "09", "10", "11"} {
		marker := `class="text-accent">` + want + `</span>`
		position := strings.Index(body[previous:], marker)
		if position == -1 {
			t.Errorf("home page does not contain section number %s after the previous section", want)
			continue
		}
		previous += position + len(marker)
	}
}

func TestLaneRailDrawsReworkAsReturnLoop(t *testing.T) {
	body := get(t, newTestServer(t), "/how-it-works", nil).Body.String()
	forwardStart := strings.Index(body, `data-lane-rail-path="forward"`)
	returnStart := strings.Index(body, `data-lane-rail-path="return"`)
	captionStart := strings.Index(body, "The notch marks a gate")

	if forwardStart == -1 || returnStart == -1 || captionStart == -1 || forwardStart >= returnStart || returnStart >= captionStart {
		t.Fatal("lane rail paths are missing or out of order")
	}

	forwardPath := body[forwardStart:returnStart]
	returnPath := body[returnStart:captionStart]
	if strings.Contains(forwardPath, "Rework") {
		t.Error("forward lane rail contains Rework")
	}

	previous := -1
	for _, lane := range []string{"Todo", "In Progress", "Human Review", "Merging", "Done"} {
		position := strings.Index(forwardPath, lane)
		if position == -1 {
			t.Errorf("forward lane rail does not contain %q", lane)
		}
		if position <= previous {
			t.Errorf("%q is out of order in the forward lane rail", lane)
		}
		previous = position
	}

	if !strings.Contains(returnPath, "Rework") {
		t.Error("return path does not contain Rework")
	}
	if !strings.Contains(returnPath, `aria-label="Human Review returns through Rework to In Progress"`) {
		t.Error("return path does not describe the Rework loop")
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

	paths := []string{"/", "/how-it-works", "/why-detent", "/dashboard", "/install", "/open-source", "/videos", "/docs", "/docs/getting-started", "/docs/site/project-contracts", "/docs/site/working-checkout-merge-gate", "/sitemap.xml"}

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
		{"/videos", "https://detent.build/videos"},
		{"/docs", "https://detent.build/docs"},
		{"/docs/getting-started", "https://detent.build/docs/getting-started"},
		{"/docs/site/project-contracts", "https://detent.build/docs/site/project-contracts"},
		{"/docs/site/working-checkout-merge-gate", "https://detent.build/docs/site/working-checkout-merge-gate"},
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
		"<loc>https://detent.build/videos</loc>",
		"<loc>https://detent.build/docs</loc>",
		"<loc>https://detent.build/docs/getting-started</loc>",
		"<loc>https://detent.build/docs/site/project-contracts</loc>",
		"<loc>https://detent.build/docs/site/working-checkout-merge-gate</loc>",
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
	assertSitemapIncludesEveryIndexablePageRoute(t, e, docs.Published)
}

func assertSitemapIncludesEveryIndexablePageRoute(t *testing.T, e *echo.Echo, catalog *docs.Catalog) {
	t.Helper()
	body := get(t, e, "/sitemap.xml", nil).Body.String()

	for path := range registeredIndexablePageRoutes(e, catalog) {
		want := "<loc>https://detent.build" + path + "</loc>"
		if !strings.Contains(body, want) {
			t.Errorf("registered page route %q is missing from the sitemap", path)
		}
	}
}

func TestSitemapContainsOnlyIndexablePageRoutes(t *testing.T) {
	e := newTestServer(t)
	assertSitemapContainsOnlyIndexablePageRoutes(t, e, docs.Published)
}

func assertSitemapContainsOnlyIndexablePageRoutes(t *testing.T, e *echo.Echo, catalog *docs.Catalog) {
	t.Helper()
	registered := registeredIndexablePageRoutes(e, catalog)

	for _, path := range renderedSitemapPaths(t, e) {
		if _, ok := registered[path]; !ok {
			t.Errorf("sitemap path %q has no registered indexable GET route", path)
		}
	}
}

func registeredIndexablePageRoutes(e *echo.Echo, catalog *docs.Catalog) map[string]struct{} {
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
	for _, alias := range catalog.Aliases() {
		nonIndexableRoutes[alias.PublicPath] = struct{}{}
	}
	for _, tombstone := range catalog.Tombstones() {
		nonIndexableRoutes[tombstone.PublicPath] = struct{}{}
	}

	registered := make(map[string]struct{})
	for _, route := range e.Routes() {
		if route.Method != http.MethodGet {
			continue
		}
		if _, excluded := nonIndexableRoutes[route.Path]; excluded {
			continue
		}
		registered[route.Path] = struct{}{}
	}
	return registered
}

func renderedSitemapPaths(t *testing.T, e *echo.Echo) []string {
	t.Helper()

	var sitemap struct {
		URLs []struct {
			Location string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(get(t, e, "/sitemap.xml", nil).Body.Bytes(), &sitemap); err != nil {
		t.Fatalf("parse sitemap: %v", err)
	}

	paths := make([]string, 0, len(sitemap.URLs))
	for _, entry := range sitemap.URLs {
		location, err := url.Parse(entry.Location)
		if err != nil {
			t.Fatalf("parse sitemap URL %q: %v", entry.Location, err)
		}
		paths = append(paths, location.Path)
	}
	return paths
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

	for _, path := range []string{"/", "/videos", "/install", "/install/linux", "/docs", "/docs/getting-started", "/docs/site/project-contracts", "/docs/site/working-checkout-merge-gate", "/health", "/sitemap.xml"} {
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
