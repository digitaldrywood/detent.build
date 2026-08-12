// Package docs exposes the pinned Detent documentation as an embedded filesystem.
package docs

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	SourceRepository = "https://github.com/digitaldrywood/detent"
	SiteRepository   = "https://github.com/digitaldrywood/detent.build"
	ReleaseTag       = "v0.57.0"
	TagObjectSHA     = "10c9b2a531089e8bac7a3fcd42593b257863ec8d"
	CommitSHA        = "1543929187369eca2703abd2a655cf86e9e5d83e"
)

//go:embed vendor site manifest.json
var embedded embed.FS

// Files contains the byte-identical documentation tree rooted at the pinned release's docs directory.
var Files = mustSub(embedded, "vendor")

var SiteFiles = mustSub(embedded, "site")

type Origin string

const (
	OriginUpstream Origin = "upstream"
	OriginSite     Origin = "site"
	SitePathPrefix        = "/docs/site/"
)

type Page struct {
	Group      string
	Title      string
	SourcePath string
	PublicPath string
	SourceURL  string
	Origin     Origin
	HTML       string
}

type Group struct {
	Title string
	Pages []Page
}

type ExternalReference struct {
	Title string
	URL   string
}

type registration struct {
	Group      string
	Title      string
	SourcePath string
	PublicPath string
	Origin     Origin
}

type Catalog struct {
	groups   []Group
	pages    []Page
	byPath   map[string]Page
	bySource map[string]string
}

var registry = []registration{
	{"detent.build guides", "Project contracts", "project-contracts.md", "/docs/site/project-contracts", OriginSite},
	{"Get started", "Quick Start", "getting-started.md", "/docs/getting-started", OriginUpstream},
	{"Get started", "Project Onboarding", "ONBOARDING.md", "/docs/project-onboarding", OriginUpstream},
	{"Get started", "Bootstrap a new machine", "bootstrap.md", "/docs/bootstrap", OriginUpstream},
	{"Get started", "Configuration", "config.md", "/docs/configuration", OriginUpstream},
	{"Operate Detent", "Concepts", "concepts.md", "/docs/concepts", OriginUpstream},
	{"Operate Detent", "Dependency workflows", "dependency-workflows.md", "/docs/dependency-workflows", OriginUpstream},
	{"Operate Detent", "Merge train", "merge-train.md", "/docs/merge-train", OriginUpstream},
	{"Operate Detent", "Multi-project operation", "multi-project.md", "/docs/multi-project", OriginUpstream},
	{"Operate Detent", "Machine-local workflow overlays", "workflow-overlays.md", "/docs/workflow-overlays", OriginUpstream},
	{"Operate Detent", "Webhook freshness", "webhook-freshness.md", "/docs/webhook-freshness", OriginUpstream},
	{"Operate Detent", "Scheduled operations", "scheduled-routines.md", "/docs/scheduled-operations", OriginUpstream},
	{"Operate Detent", "Admission criteria", "admission.md", "/docs/admission", OriginUpstream},
	{"Operate Detent", "Efficiency retrospection", "retrospection.md", "/docs/retrospection", OriginUpstream},
	{"Operate Detent", "Dashboard and APIs", "dashboard-api.md", "/docs/dashboard-api", OriginUpstream},
	{"Reference and contribute", "CLI reference", "cli.md", "/docs/cli", OriginUpstream},
	{"Reference and contribute", "Execution seams", "execution-seams.md", "/docs/execution-seams", OriginUpstream},
	{"Reference and contribute", "Local models", "local-models-ollama.md", "/docs/local-models", OriginUpstream},
}

var ExternalReferences = []ExternalReference{
	{"Release process", RepositoryBlobURL("docs/release.md")},
	{"Development", RepositoryBlobURL("docs/development.md")},
	{"Contribution guide", RepositoryBlobURL("CONTRIBUTING.md")},
	{"Comparison", RepositoryBlobURL("docs/comparison.md")},
}

var Published = mustLoadCatalog(Files, SiteFiles, registry)

func (c *Catalog) Groups() []Group {
	groups := make([]Group, len(c.groups))
	for i, group := range c.groups {
		groups[i] = Group{Title: group.Title, Pages: append([]Page(nil), group.Pages...)}
	}
	return groups
}

func (c *Catalog) Pages() []Page {
	return append([]Page(nil), c.pages...)
}

func (c *Catalog) Page(publicPath string) (Page, bool) {
	page, ok := c.byPath[publicPath]
	return page, ok
}

func (c *Catalog) Paths() []string {
	paths := make([]string, 0, len(c.pages))
	for _, page := range c.pages {
		paths = append(paths, page.PublicPath)
	}
	return paths
}

func PublicPath(sourcePath string) (string, bool) {
	for _, entry := range registry {
		if entry.SourcePath == sourcePath {
			return entry.PublicPath, true
		}
	}
	return "", false
}

func RepositoryBlobURL(sourcePath string) string {
	return SourceRepository + "/blob/" + CommitSHA + "/" + strings.TrimPrefix(sourcePath, "/")
}

func RepositoryTreeURL(sourcePath string) string {
	return SourceRepository + "/tree/" + CommitSHA + "/" + strings.TrimPrefix(sourcePath, "/")
}

func ShortCommit() string {
	return CommitSHA[:12]
}

func mustLoadCatalog(upstream, site fs.FS, entries []registration) *Catalog {
	catalog, err := loadCatalog(upstream, site, entries)
	if err != nil {
		panic(err)
	}
	return catalog
}

func loadCatalog(upstream, site fs.FS, entries []registration) (*Catalog, error) {
	catalog := &Catalog{
		byPath:   make(map[string]Page, len(entries)),
		bySource: make(map[string]string, len(entries)),
	}
	for _, entry := range entries {
		if err := validateRegistration(entry); err != nil {
			return nil, err
		}
		if _, exists := catalog.bySource[entry.SourcePath]; exists {
			return nil, fmt.Errorf("duplicate documentation source path %q", entry.SourcePath)
		}
		if _, exists := catalog.byPath[entry.PublicPath]; exists {
			return nil, fmt.Errorf("duplicate documentation public path %q", entry.PublicPath)
		}
		catalog.bySource[entry.SourcePath] = entry.PublicPath
		catalog.byPath[entry.PublicPath] = Page{}
	}

	markdown := markdownRenderer()
	groupIndex := make(map[string]int)
	for _, entry := range entries {
		source := upstream
		if entry.Origin == OriginSite {
			source = site
		}
		sourceBytes, err := fs.ReadFile(source, entry.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("read documentation source %q: %w", entry.SourcePath, err)
		}
		rendered, err := renderMarkdown(markdown, sourceBytes, entry.SourcePath, catalog.bySource, source)
		if err != nil {
			return nil, fmt.Errorf("render documentation source %q: %w", entry.SourcePath, err)
		}
		page := Page{
			Group:      entry.Group,
			Title:      entry.Title,
			SourcePath: entry.SourcePath,
			PublicPath: entry.PublicPath,
			SourceURL:  documentationSourceURL(entry),
			Origin:     entry.Origin,
			HTML:       rendered,
		}
		catalog.pages = append(catalog.pages, page)
		catalog.byPath[page.PublicPath] = page
		index, exists := groupIndex[page.Group]
		if !exists {
			index = len(catalog.groups)
			groupIndex[page.Group] = index
			catalog.groups = append(catalog.groups, Group{Title: page.Group})
		}
		catalog.groups[index].Pages = append(catalog.groups[index].Pages, page)
	}
	return catalog, nil
}

func documentationSourceURL(entry registration) string {
	if entry.Origin == OriginSite {
		return SiteRepository + "/blob/main/" + path.Join("docs/site/hype", entry.SourcePath)
	}
	return RepositoryBlobURL(path.Join("docs", entry.SourcePath))
}

func markdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

func validateRegistration(entry registration) error {
	if entry.Group == "" || entry.Title == "" {
		return fmt.Errorf("documentation registration %q is missing a group or title", entry.SourcePath)
	}
	if entry.SourcePath == "" || path.Clean(entry.SourcePath) != entry.SourcePath || path.IsAbs(entry.SourcePath) {
		return fmt.Errorf("invalid documentation source path %q", entry.SourcePath)
	}
	if path.Ext(entry.SourcePath) != ".md" {
		return fmt.Errorf("documentation source %q is not Markdown", entry.SourcePath)
	}
	if entry.PublicPath == "" || path.Clean(entry.PublicPath) != entry.PublicPath || !strings.HasPrefix(entry.PublicPath, "/docs/") {
		return fmt.Errorf("invalid documentation public path %q", entry.PublicPath)
	}
	if entry.Origin == OriginSite && !strings.HasPrefix(entry.PublicPath, SitePathPrefix) {
		return fmt.Errorf("site-authored documentation path %q must use %q", entry.PublicPath, SitePathPrefix)
	}
	if entry.Origin == OriginUpstream && strings.HasPrefix(entry.PublicPath, SitePathPrefix) {
		return fmt.Errorf("upstream documentation path %q uses the site-authored prefix", entry.PublicPath)
	}
	if entry.Origin != OriginSite && entry.Origin != OriginUpstream {
		return fmt.Errorf("documentation source %q has invalid origin %q", entry.SourcePath, entry.Origin)
	}
	return nil
}

func renderMarkdown(markdown goldmark.Markdown, source []byte, sourcePath string, publicPaths map[string]string, files fs.FS) (string, error) {
	reader := text.NewReader(source)
	document := markdown.Parser().Parse(reader)
	if err := rewriteLinks(document, sourcePath, publicPaths, files); err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := markdown.Renderer().Render(&output, source, document); err != nil {
		return "", err
	}
	return output.String(), nil
}

func rewriteLinks(document ast.Node, sourcePath string, publicPaths map[string]string, files fs.FS) error {
	return ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch link := node.(type) {
		case *ast.Link:
			link.Destination = []byte(resolveDestination(sourcePath, string(link.Destination), publicPaths, files))
		case *ast.Image:
			link.Destination = []byte(resolveDestination(sourcePath, string(link.Destination), publicPaths, files))
		}
		return ast.WalkContinue, nil
	})
}

func resolveDestination(sourcePath, destination string, publicPaths map[string]string, files fs.FS) string {
	parsed, err := url.Parse(destination)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || strings.HasPrefix(destination, "/") {
		return destination
	}
	target := path.Clean(path.Join(path.Dir(path.Join("docs", sourcePath)), parsed.Path))
	if target == ".." || strings.HasPrefix(target, "../") {
		return destination
	}
	if strings.HasPrefix(target, "docs/") {
		vendoredPath := strings.TrimPrefix(target, "docs/")
		if publicPath, ok := publicPaths[vendoredPath]; ok {
			return withURLSuffix(publicPath, parsed)
		}
		if info, statErr := fs.Stat(files, vendoredPath); statErr == nil && info.IsDir() {
			return withURLSuffix(RepositoryTreeURL(target), parsed)
		}
	}
	return withURLSuffix(RepositoryBlobURL(target), parsed)
}

func withURLSuffix(base string, parsed *url.URL) string {
	if parsed.RawQuery != "" {
		base += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		base += "#" + parsed.EscapedFragment()
	}
	return base
}

func mustSub(source fs.FS, path string) fs.FS {
	result, err := fs.Sub(source, path)
	if err != nil {
		panic(err)
	}
	return result
}
