package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// sitemapPaths are the canonical, indexable URLs. The install platform tabs
// are deliberately absent: they are the same page with a different tab
// selected, and /install already carries the canonical.
var sitemapPaths = []string{
	"/",
	"/how-it-works",
	"/why-detent",
	"/dashboard",
	"/install",
	"/open-source",
}

// Sitemap emits the sitemap against the configured site URL. detent.build has
// no www host and never will, so every URL here is apex and https.
func (h *Handler) Sitemap(c echo.Context) error {
	base := strings.TrimSuffix(h.cfg.Site.URL, "/")

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range sitemapPaths {
		b.WriteString("  <url><loc>")
		b.WriteString(base)
		if p != "/" {
			b.WriteString(p)
		} else {
			b.WriteString("/")
		}
		b.WriteString("</loc></url>\n")
	}
	b.WriteString("</urlset>\n")

	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", []byte(b.String()))
}
