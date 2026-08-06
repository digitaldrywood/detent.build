package middleware

import (
	"context"

	"detent.build/internal/config"
	"detent.build/internal/ctxkeys"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Setup(e *echo.Echo, cfg *config.Config) {
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(SiteConfigMiddleware(cfg.Site))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; base-uri 'self'; form-action 'self'; frame-ancestors 'self'",
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
