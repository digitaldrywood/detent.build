package meta

import (
	"context"
	"strings"

	"detent.build/internal/config"
	"detent.build/internal/ctxkeys"
)

func SiteFromCtx(ctx context.Context) config.SiteConfig {
	if cfg, ok := ctx.Value(ctxkeys.SiteConfig).(config.SiteConfig); ok {
		return cfg
	}
	return config.SiteConfig{Name: "detent.build"}
}

func SiteNameFromCtx(ctx context.Context) string {
	return SiteFromCtx(ctx).Name
}

func SiteURLFromCtx(ctx context.Context) string {
	return SiteFromCtx(ctx).URL
}

// AbsoluteURL resolves a site-root path against the configured site URL.
// Open Graph and Twitter card images must be absolute, and the production
// host is HSTS-preloaded, so a relative or http path is not merely untidy —
// crawlers cannot fetch it.
func AbsoluteURL(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	base := strings.TrimSuffix(SiteURLFromCtx(ctx), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// OGImageFromCtx returns the absolute card image for a page, falling back to
// the site-wide default when the page does not set one.
func OGImageFromCtx(ctx context.Context, m PageMeta) string {
	image := m.OGImage
	if image == "" {
		image = SiteFromCtx(ctx).DefaultOGImage
	}
	return AbsoluteURL(ctx, image)
}
