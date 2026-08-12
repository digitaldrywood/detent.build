package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"detent.build/internal/docsregistry"
)

type provenanceManifest struct {
	Schema       int              `json:"schema"`
	Repository   string           `json:"repository"`
	Tag          string           `json:"tag"`
	TagObjectSHA string           `json:"tag_object_sha"`
	CommitSHA    string           `json:"commit_sha"`
	SyncedAt     string           `json:"synced_at"`
	Files        []provenanceFile `json:"files"`
}

type provenanceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// TestVendoredContentMatchesManifest is an offline tamper-evidence check. It
// detects local edits to the vendored bytes, but does not prove those bytes
// came from upstream; make docs-sync performs that networked provenance check.
func TestVendoredContentMatchesManifest(t *testing.T) {
	contents, err := fs.ReadFile(embedded, "manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest provenanceManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Schema != 1 {
		t.Errorf("manifest schema = %d, want 1", manifest.Schema)
	}
	if manifest.Repository != SourceRepository {
		t.Errorf("manifest repository = %q, want %q", manifest.Repository, SourceRepository)
	}
	if manifest.Tag != ReleaseTag {
		t.Errorf("manifest tag = %q, want %q", manifest.Tag, ReleaseTag)
	}
	if manifest.TagObjectSHA != TagObjectSHA {
		t.Errorf("manifest tag object = %q, want %q", manifest.TagObjectSHA, TagObjectSHA)
	}
	if manifest.CommitSHA != CommitSHA {
		t.Errorf("manifest commit = %q, want %q", manifest.CommitSHA, CommitSHA)
	}
	if _, err := time.Parse(time.RFC3339, manifest.SyncedAt); err != nil {
		t.Errorf("manifest sync timestamp %q is not RFC 3339: %v", manifest.SyncedAt, err)
	}

	want := make(map[string]string, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.Path == "" || path.Clean(file.Path) != file.Path || path.IsAbs(file.Path) {
			t.Errorf("manifest contains unsafe path %q", file.Path)
			continue
		}
		if _, exists := want[file.Path]; exists {
			t.Errorf("manifest contains duplicate path %q", file.Path)
			continue
		}
		want[file.Path] = file.SHA256
	}

	var got []string
	err = fs.WalkDir(Files, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		got = append(got, filePath)
		contents, readErr := fs.ReadFile(Files, filePath)
		if readErr != nil {
			return readErr
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		expected, exists := want[filePath]
		if !exists {
			t.Errorf("vendored file %q is missing from the manifest", filePath)
			return nil
		}
		if actual != expected {
			t.Errorf("vendored file %q has SHA-256 %s, want %s", filePath, actual, expected)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk vendored files: %v", err)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Errorf("vendored tree has %d files, manifest has %d", len(got), len(want))
	}
	for _, filePath := range got {
		delete(want, filePath)
	}
	for missing := range want {
		t.Errorf("manifest file %q is missing from the vendored tree", missing)
	}
}

func TestPublishedRegistryMatchesCuratedInformationArchitecture(t *testing.T) {
	groups := Published.Groups()
	wantGroups := []struct {
		title string
		count int
	}{
		{"detent.build guides", 1},
		{"Get started", 4},
		{"Operate Detent", 10},
		{"Reference and contribute", 3},
	}
	if len(groups) != len(wantGroups) {
		t.Fatalf("group count = %d, want %d", len(groups), len(wantGroups))
	}
	for index, want := range wantGroups {
		if groups[index].Title != want.title {
			t.Errorf("group %d title = %q, want %q", index, groups[index].Title, want.title)
		}
		if len(groups[index].Pages) != want.count {
			t.Errorf("group %q page count = %d, want %d", want.title, len(groups[index].Pages), want.count)
		}
	}

	excluded := []string{
		"comparison.md",
		"development.md",
		"parity-audit.md",
		"redesign-contrast.md",
		"release.md",
		"structured-workpad-signaling-migration.md",
		"thread-resume-spike.md",
		"workflow-layout-migration.md",
		"templates/detent.label.yaml",
	}
	for _, sourcePath := range excluded {
		if publicPath, ok := PublicPath(sourcePath); ok {
			t.Errorf("excluded source %q is published at %q", sourcePath, publicPath)
		}
	}
}

func TestPublishedPagesAreRenderedWithPinnedSources(t *testing.T) {
	for _, page := range Published.Pages() {
		t.Run(page.SourcePath, func(t *testing.T) {
			if page.HTML == "" {
				t.Error("rendered HTML is empty")
			}
			if !strings.Contains(page.HTML, "<h1") {
				t.Error("rendered HTML does not contain the document heading")
			}
			if page.Origin == OriginUpstream && !strings.Contains(page.SourceURL, CommitSHA) {
				t.Errorf("upstream source URL %q does not contain commit %s", page.SourceURL, CommitSHA)
			}
			if page.Origin == OriginSite && !strings.HasPrefix(page.SourceURL, SiteRepository+"/blob/main/docs/site/hype/") {
				t.Errorf("site source URL %q does not point to the Hype source", page.SourceURL)
			}
			if got, ok := Published.Page(page.PublicPath); !ok || got.SourcePath != page.SourcePath {
				t.Errorf("public path %q does not resolve to %q", page.PublicPath, page.SourcePath)
			}
		})
	}
}

func TestMarkdownRenderingOmitsRawHTMLAndDangerousLinks(t *testing.T) {
	source := []byte("# Safety\n\n<script>alert('unsafe')</script>\n\n[unsafe](javascript:alert('unsafe'))")
	rendered, err := renderMarkdown(markdownRenderer(), source, "safety.md", map[string]string{}, Files)
	if err != nil {
		t.Fatalf("render Markdown: %v", err)
	}
	for _, forbidden := range []string{"<script", "javascript:", "alert('unsafe')"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("rendered HTML contains unsafe content %q: %s", forbidden, rendered)
		}
	}
}

func TestRelativeLinkDestinationsResolveFromSourceDocument(t *testing.T) {
	tests := []struct {
		name        string
		sourcePath  string
		destination string
		publicPaths map[string]string
		want        string
	}{
		{"published sibling with fragment", "getting-started.md", "concepts.md#review-gate", Published.bySource, "/docs/concepts#review-gate"},
		{"published nested Markdown with query", "bootstrap.md", "examples/non-code-artifact/README.md?plain=1#run-the-demo", map[string]string{"examples/non-code-artifact/README.md": "/docs/non-code-artifact"}, "/docs/non-code-artifact?plain=1#run-the-demo"},
		{"unpublished vendored Markdown", "getting-started.md", "workflow-layout-migration.md", Published.bySource, RepositoryBlobURL("docs/workflow-layout-migration.md")},
		{"repository root Markdown", "getting-started.md", "../README.md#documentation", Published.bySource, RepositoryBlobURL("README.md") + "#documentation"},
		{"repository root non-Markdown", "config.md", "../config.annotated.yaml", Published.bySource, RepositoryBlobURL("config.annotated.yaml")},
		{"vendored directory", "bootstrap.md", "examples/non-code-artifact", Published.bySource, RepositoryTreeURL("docs/examples/non-code-artifact")},
		{"vendored non-Markdown file", "getting-started.md", "templates/detent.label.yaml", Published.bySource, RepositoryBlobURL("docs/templates/detent.label.yaml")},
		{"nested source sibling", "examples/non-code-artifact/README.md", "WORKFLOW.md", Published.bySource, RepositoryBlobURL("docs/examples/non-code-artifact/WORKFLOW.md")},
		{"nested source ancestor", "examples/non-code-artifact/README.md", "../../concepts.md", Published.bySource, "/docs/concepts"},
		{"percent-encoded published path", "getting-started.md", "workflow%2Doverlays.md#machine-local-workflow-overlays", Published.bySource, "/docs/workflow-overlays#machine-local-workflow-overlays"},
		{"external URL", "getting-started.md", "https://example.com/guide.md?plain=1#top", Published.bySource, "https://example.com/guide.md?plain=1#top"},
		{"absolute site path", "getting-started.md", "/docs/concepts", Published.bySource, "/docs/concepts"},
		{"fragment only", "getting-started.md", "#quick-start", Published.bySource, "#quick-start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveDestination(tt.sourcePath, tt.destination, tt.publicPaths, Files); got != tt.want {
				t.Errorf("resolveDestination(%q, %q) = %q, want %q", tt.sourcePath, tt.destination, got, tt.want)
			}
		})
	}
}

func TestMarkdownRewritesImagesAndPreservesAutolinks(t *testing.T) {
	source := []byte(strings.Join([]string{
		"![Detent mark](brand/detent-mark.svg?raw=1)",
		"<https://example.com/guide.md?plain=1#top>",
	}, "\n\n"))
	rendered, err := renderMarkdown(markdownRenderer(), source, "getting-started.md", Published.bySource, Files)
	if err != nil {
		t.Fatalf("render Markdown: %v", err)
	}
	for _, want := range []string{
		`src="` + RepositoryBlobURL("docs/brand/detent-mark.svg") + `?raw=1"`,
		`href="https://example.com/guide.md?plain=1#top"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered HTML is missing %s: %s", want, rendered)
		}
	}
}

func TestHeadingIDsMatchInboundGitHubFragments(t *testing.T) {
	tests := []struct {
		sourcePath string
		fragment   string
	}{
		{"bootstrap.md", "bootstrap-on-a-new-machine-humans-and-ai-agents"},
		{"getting-started.md", "quick-start"},
		{"multi-project.md", "running-multiple-instances"},
	}

	for _, tt := range tests {
		t.Run(tt.sourcePath+"#"+tt.fragment, func(t *testing.T) {
			source, err := fs.ReadFile(Files, tt.sourcePath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			rendered, err := renderMarkdown(markdownRenderer(), source, tt.sourcePath, Published.bySource, Files)
			if err != nil {
				t.Fatalf("render Markdown: %v", err)
			}
			if !strings.Contains(rendered, `id="`+tt.fragment+`"`) {
				t.Errorf("rendered heading does not contain inbound GitHub fragment %q", tt.fragment)
			}
		})
	}

	repeatedRealHeading := []byte("# Quick Start\n\n## Quick Start\n")
	rendered, err := renderMarkdown(markdownRenderer(), repeatedRealHeading, "getting-started.md", Published.bySource, Files)
	if err != nil {
		t.Fatalf("render duplicate headings: %v", err)
	}
	for _, id := range []string{"quick-start", "quick-start-1"} {
		if !strings.Contains(rendered, `id="`+id+`"`) {
			t.Errorf("duplicate heading is missing GitHub-compatible ID %q", id)
		}
	}
}

func TestPublishedDocumentationHasNoUnresolvedRelativeLinks(t *testing.T) {
	attribute := regexp.MustCompile(`(?:href|src)="([^"]+)"`)

	for _, page := range Published.Pages() {
		t.Run(page.SourcePath, func(t *testing.T) {
			for _, match := range attribute.FindAllStringSubmatch(page.HTML, -1) {
				destination := match[1]
				parsed, err := url.Parse(destination)
				if err != nil {
					t.Errorf("parse rendered destination %q: %v", destination, err)
					continue
				}
				if !parsed.IsAbs() && !strings.HasPrefix(parsed.Path, "/") && path.Ext(parsed.Path) == ".md" {
					t.Errorf("rendered documentation contains unresolved relative Markdown link %q", destination)
				}
				if strings.HasPrefix(destination, SourceRepository+"/") {
					if !strings.Contains(destination, "/"+CommitSHA+"/") {
						t.Errorf("GitHub destination is not pinned to commit %s: %q", CommitSHA, destination)
					}
					if strings.Contains(destination, "/main/") {
						t.Errorf("GitHub destination points at main: %q", destination)
					}
				}
			}
			for _, forbidden := range []string{"www.detent.build", "http://detent.build"} {
				if strings.Contains(page.HTML, forbidden) {
					t.Errorf("rendered documentation contains forbidden URL %q", forbidden)
				}
			}
		})
	}
}

func TestSiteAuthoredDocumentationUsesDedicatedPrefix(t *testing.T) {
	siteEntry := docsregistry.Page{
		Group:      "Site",
		Title:      "Guide",
		SourcePath: "guide.md",
		PublicPath: SitePathPrefix + "guide",
		Origin:     OriginSite,
	}
	if err := validateRegistration(siteEntry); err != nil {
		t.Fatalf("valid site registration: %v", err)
	}
	siteEntry.PublicPath = "/docs/guide"
	if err := validateRegistration(siteEntry); err == nil {
		t.Error("site registration outside the dedicated prefix passed validation")
	}

	upstreamEntry := siteEntry
	upstreamEntry.PublicPath = SitePathPrefix + "upstream"
	upstreamEntry.Origin = OriginUpstream
	if err := validateRegistration(upstreamEntry); err == nil {
		t.Error("upstream registration inside the site prefix passed validation")
	}
}

func TestCatalogLoadFailsWhenRegisteredSourceIsMissing(t *testing.T) {
	_, err := loadCatalog(fstest.MapFS{}, fstest.MapFS{}, docsregistry.Registry{Pages: []docsregistry.Page{{
		Group:      "Get started",
		Title:      "Missing",
		SourcePath: "missing.md",
		PublicPath: "/docs/missing",
		Origin:     OriginUpstream,
	}}})
	if err == nil {
		t.Fatal("loadCatalog succeeded for a missing source")
	}
}

func TestCatalogLoadsAliasAndTombstoneRoutes(t *testing.T) {
	registry := docsregistry.Registry{
		Pages: []docsregistry.Page{{
			Group:      "Reference",
			Title:      "Renamed",
			SourcePath: "renamed.md",
			PublicPath: "/docs/renamed",
			Origin:     OriginUpstream,
		}},
		Aliases:    []docsregistry.Alias{{PublicPath: "/docs/old-name", CanonicalPath: "/docs/renamed"}},
		Tombstones: []docsregistry.Tombstone{{PublicPath: "/docs/withdrawn"}},
	}
	catalog, err := loadCatalog(fstest.MapFS{
		"renamed.md": {Data: []byte("# Renamed\n")},
	}, fstest.MapFS{}, registry)
	if err != nil {
		t.Fatalf("loadCatalog() error = %v", err)
	}

	aliases := catalog.Aliases()
	if len(aliases) != 1 || aliases[0].PublicPath != "/docs/old-name" || aliases[0].Page.PublicPath != "/docs/renamed" {
		t.Errorf("aliases = %#v", aliases)
	}
	tombstones := catalog.Tombstones()
	if len(tombstones) != 1 || tombstones[0].PublicPath != "/docs/withdrawn" {
		t.Errorf("tombstones = %#v", tombstones)
	}
	if page, ok := catalog.Page("/docs/old-name"); !ok || page.PublicPath != "/docs/renamed" {
		t.Errorf("alias lookup = %#v, %v", page, ok)
	}
	if paths := catalog.Paths(); len(paths) != 1 || paths[0] != "/docs/renamed" {
		t.Errorf("indexable paths = %#v", paths)
	}
}

func TestCatalogRejectsAliasWithoutCanonicalPage(t *testing.T) {
	_, err := loadCatalog(fstest.MapFS{}, fstest.MapFS{}, docsregistry.Registry{
		Aliases: []docsregistry.Alias{{PublicPath: "/docs/old-name", CanonicalPath: "/docs/missing"}},
	})
	if err == nil {
		t.Fatal("loadCatalog succeeded for an alias without a canonical page")
	}
}
