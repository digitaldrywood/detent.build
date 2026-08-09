package handler

import (
	"net/http"

	"detent.build/internal/config"
	"detent.build/internal/docs"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	cfg  *config.Config
	docs *docs.Catalog
}

func New(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg, docs: docs.Published}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.HTTPErrorHandler = ErrorHandler

	e.Static("/static", "static")
	e.Static("/assets", "assets") // templUI component JavaScript
	// No /favicon.ico route: the layout links the SVG mark, and a registered
	// route to a file that does not exist just turns a 404 into a slower 404.
	e.File("/robots.txt", "static/robots.txt")
	e.GET("/sitemap.xml", h.Sitemap)

	e.GET("/health", h.Health)

	e.GET("/", h.Home)
	e.GET("/how-it-works", h.HowItWorks)
	e.GET("/why-detent", h.WhyDetent)
	e.GET("/dashboard", h.Dashboard)
	e.GET("/install", h.Install)
	e.GET("/open-source", h.OpenSource)
	e.GET("/docs", h.DocsIndex)
	for _, page := range h.docs.Pages() {
		e.GET(page.PublicPath, h.DocPage(page))
	}

	// HTMX partial: swap the install commands for one platform without a
	// page load. Falls back to the full page when JavaScript is off, because
	// every tab is a real URL.
	e.GET("/install/:platform", h.InstallPlatform)
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// render writes a templ component as an HTML response.
//
// It writes to c.Response(), not c.Response().Writer. The former is Echo's
// wrapper, which counts bytes and marks the response committed; the latter is
// the raw http.ResponseWriter, and writing to it leaves every access log line
// reporting 0 bytes.
func render(c echo.Context, status int, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(status)
	return component.Render(c.Request().Context(), c.Response())
}
