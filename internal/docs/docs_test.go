package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"
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
	rendered, err := renderMarkdown(markdownRenderer(), source, "safety.md", map[string]string{})
	if err != nil {
		t.Fatalf("render Markdown: %v", err)
	}
	for _, forbidden := range []string{"<script", "javascript:", "alert('unsafe')"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("rendered HTML contains unsafe content %q: %s", forbidden, rendered)
		}
	}
}

func TestMarkdownLinksResolveThroughRegistryOrPinnedRepository(t *testing.T) {
	source := []byte(strings.Join([]string{
		"[published](concepts.md#review-gates)",
		"[excluded](workflow-layout-migration.md)",
		"[root](../README.md#documentation)",
		"[artifact](templates/detent.label.yaml)",
	}, "\n\n"))
	rendered, err := renderMarkdown(markdownRenderer(), source, "getting-started.md", Published.bySource)
	if err != nil {
		t.Fatalf("render Markdown: %v", err)
	}
	wants := []string{
		`href="/docs/concepts#review-gates"`,
		`href="` + RepositoryBlobURL("docs/workflow-layout-migration.md") + `"`,
		`href="` + RepositoryBlobURL("README.md") + `#documentation"`,
		`href="` + RepositoryBlobURL("docs/templates/detent.label.yaml") + `"`,
	}
	for _, want := range wants {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered HTML is missing %s: %s", want, rendered)
		}
	}
}

func TestSiteAuthoredDocumentationUsesDedicatedPrefix(t *testing.T) {
	siteEntry := registration{
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
	_, err := loadCatalog(fstest.MapFS{}, fstest.MapFS{}, []registration{{
		Group:      "Get started",
		Title:      "Missing",
		SourcePath: "missing.md",
		PublicPath: "/docs/missing",
		Origin:     OriginUpstream,
	}})
	if err == nil {
		t.Fatal("loadCatalog succeeded for a missing source")
	}
}
