package middleware

import (
	"context"

	"detent.build/internal/config"
	"detent.build/internal/ctxkeys"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Setup installs the standard middleware stack. version may be nil, in which
// case pages fall back to the compiled-in release constant.
func Setup(e *echo.Echo, cfg *config.Config, version VersionProvider) {
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(RequestLogger())
	e.Use(SiteConfigMiddleware(cfg.Site))
	if version != nil {
		e.Use(VersionMiddleware(version))
	}
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
		HSTSMaxAge:         31536000,
		// script-src keeps 'unsafe-inline' for the two small inline scripts: the
		// pre-paint theme read in base.templ, which cannot move to an external
		// file without a flash, and the theme toggle in nav.templ.
		//
		// style-src stays 'self' — no page renders a <style> block or a style
		// attribute. The exception is narrowed to style-src-attr, which templUI's
		// copybutton needs because it toggles element.style at runtime for its
		// copied-state icon swap. Browser-verified: with a blanket
		// style-src 'self' the component logs a CSP violation.
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
			"style-src 'self'; style-src-attr 'unsafe-inline'; " +
			"img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; " +
			"base-uri 'self'; form-action 'self'; frame-ancestors 'self'",
		ReferrerPolicy: "strict-origin-when-cross-origin",
	}))
}

func SiteConfigMiddleware(site config.SiteConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), ctxkeys.SiteConfig, site)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// VersionProvider supplies the current Detent release tag.
type VersionProvider interface {
	Current() string
}

// VersionMiddleware puts the live release tag on the request context so
// templates render the current version rather than one frozen at build time.
func VersionMiddleware(p VersionProvider) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), ctxkeys.Version, p.Current())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
