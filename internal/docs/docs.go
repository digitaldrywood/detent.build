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

	"detent.build/internal/docsregistry"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	SourceRepository = "https://github.com/digitaldrywood/detent"
	ReleaseTag       = "v0.57.0"
	TagObjectSHA     = "10c9b2a531089e8bac7a3fcd42593b257863ec8d"
	CommitSHA        = "1543929187369eca2703abd2a655cf86e9e5d83e"
)

//go:embed vendor manifest.json
var embedded embed.FS

// Files contains the byte-identical documentation tree rooted at the pinned release's docs directory.
var Files = mustSub(embedded, "vendor")

type Origin = docsregistry.Origin

const (
	OriginUpstream = docsregistry.OriginUpstream
	OriginSite     = docsregistry.OriginSite
	SitePathPrefix = "/docs/site/"
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

type Alias struct {
	PublicPath string
	Page       Page
}

type Tombstone struct {
	PublicPath string
}

type Catalog struct {
	groups     []Group
	pages      []Page
	aliases    []Alias
	tombstones []Tombstone
	byPath     map[string]Page
	bySource   map[string]string
}

var ExternalReferences = []ExternalReference{
	{"Release process", RepositoryBlobURL("docs/release.md")},
	{"Development", RepositoryBlobURL("docs/development.md")},
	{"Contribution guide", RepositoryBlobURL("CONTRIBUTING.md")},
	{"Comparison", RepositoryBlobURL("docs/comparison.md")},
}

var Published = mustLoadCatalog(Files, docsregistry.Current)

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

func (c *Catalog) Aliases() []Alias {
	return append([]Alias(nil), c.aliases...)
}

func (c *Catalog) Tombstones() []Tombstone {
	return append([]Tombstone(nil), c.tombstones...)
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
	for _, entry := range docsregistry.Current.Pages {
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

func NewCatalog(source fs.FS, registry docsregistry.Registry) (*Catalog, error) {
	return loadCatalog(source, registry)
}

func mustLoadCatalog(source fs.FS, registry docsregistry.Registry) *Catalog {
	catalog, err := loadCatalog(source, registry)
	if err != nil {
		panic(err)
	}
	return catalog
}

func loadCatalog(source fs.FS, registry docsregistry.Registry) (*Catalog, error) {
	catalog := &Catalog{
		byPath:   make(map[string]Page, len(registry.Pages)+len(registry.Aliases)),
		bySource: make(map[string]string, len(registry.Pages)),
	}
	for _, entry := range registry.Pages {
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
	for _, entry := range registry.Pages {
		sourceBytes, err := fs.ReadFile(source, entry.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("read documentation source %q: %w", entry.SourcePath, err)
		}
		rendered, err := renderMarkdown(markdown, sourceBytes, entry.SourcePath, catalog.bySource)
		if err != nil {
			return nil, fmt.Errorf("render documentation source %q: %w", entry.SourcePath, err)
		}
		page := Page{
			Group:      entry.Group,
			Title:      entry.Title,
			SourcePath: entry.SourcePath,
			PublicPath: entry.PublicPath,
			SourceURL:  RepositoryBlobURL(path.Join("docs", entry.SourcePath)),
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
	for _, entry := range registry.Aliases {
		if err := validateRetiredPath(entry.PublicPath); err != nil {
			return nil, fmt.Errorf("invalid documentation alias: %w", err)
		}
		if _, exists := catalog.byPath[entry.PublicPath]; exists {
			return nil, fmt.Errorf("duplicate documentation public path %q", entry.PublicPath)
		}
		page, exists := catalog.byPath[entry.CanonicalPath]
		if !exists {
			return nil, fmt.Errorf("documentation alias %q points to unknown canonical path %q", entry.PublicPath, entry.CanonicalPath)
		}
		catalog.aliases = append(catalog.aliases, Alias{PublicPath: entry.PublicPath, Page: page})
		catalog.byPath[entry.PublicPath] = page
	}
	for _, entry := range registry.Tombstones {
		if err := validateRetiredPath(entry.PublicPath); err != nil {
			return nil, fmt.Errorf("invalid documentation tombstone: %w", err)
		}
		if _, exists := catalog.byPath[entry.PublicPath]; exists {
			return nil, fmt.Errorf("duplicate documentation public path %q", entry.PublicPath)
		}
		for _, tombstone := range catalog.tombstones {
			if tombstone.PublicPath == entry.PublicPath {
				return nil, fmt.Errorf("duplicate documentation public path %q", entry.PublicPath)
			}
		}
		catalog.tombstones = append(catalog.tombstones, Tombstone{PublicPath: entry.PublicPath})
	}
	return catalog, nil
}

func markdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

func validateRegistration(entry docsregistry.Page) error {
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

func validateRetiredPath(publicPath string) error {
	if publicPath == "" || path.Clean(publicPath) != publicPath || !strings.HasPrefix(publicPath, "/docs/") {
		return fmt.Errorf("invalid documentation public path %q", publicPath)
	}
	return nil
}

func renderMarkdown(markdown goldmark.Markdown, source []byte, sourcePath string, publicPaths map[string]string) (string, error) {
	reader := text.NewReader(source)
	document := markdown.Parser().Parse(reader)
	if err := rewriteLinks(document, sourcePath, publicPaths); err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := markdown.Renderer().Render(&output, source, document); err != nil {
		return "", err
	}
	return output.String(), nil
}

func rewriteLinks(document ast.Node, sourcePath string, publicPaths map[string]string) error {
	return ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch link := node.(type) {
		case *ast.Link:
			link.Destination = []byte(resolveDestination(sourcePath, string(link.Destination), publicPaths))
		case *ast.Image:
			link.Destination = []byte(resolveDestination(sourcePath, string(link.Destination), publicPaths))
		}
		return ast.WalkContinue, nil
	})
}

func resolveDestination(sourcePath, destination string, publicPaths map[string]string) string {
	parsed, err := url.Parse(destination)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || strings.HasPrefix(destination, "/") {
		return destination
	}
	target := path.Clean(path.Join(path.Dir(path.Join("docs", sourcePath)), parsed.Path))
	if target == ".." || strings.HasPrefix(target, "../") {
		return destination
	}
	if strings.HasPrefix(target, "docs/") {
		if publicPath, ok := publicPaths[strings.TrimPrefix(target, "docs/")]; ok {
			return withURLSuffix(publicPath, parsed)
		}
	}
	return withURLSuffix(RepositoryBlobURL(target), parsed)
}

func withURLSuffix(base string, parsed *url.URL) string {
	if parsed.RawQuery != "" {
		base += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		base += "#" + parsed.Fragment
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
