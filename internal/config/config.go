package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type SiteConfig struct {
	Name           string
	URL            string
	DefaultOGImage string
}

type Config struct {
	Port string
	Env  string
	Site SiteConfig
	// TailscaleHostname is the MagicDNS name of the host, e.g.
	// corys-mac-studio.tail60aac7.ts.net. When set, startup logs a second
	// clickable URL so the dev server can be opened from any device on the
	// tailnet. Development convenience only; leave it unset in production.
	TailscaleHostname string
}

func Load() *Config {
	return &Config{
		Port:              getEnvOrDefault("PORT", "3000"),
		Env:               getEnvOrDefault("ENV", "development"),
		TailscaleHostname: strings.TrimSpace(os.Getenv("TAILSCALE_HOSTNAME")),
		Site: SiteConfig{
			Name:           getEnvOrDefault("SITE_NAME", "detent.build"),
			URL:            getEnvOrDefault("SITE_URL", "http://localhost:3000"),
			DefaultOGImage: getEnvOrDefault("DEFAULT_OG_IMAGE", "/static/images/og-default.png"),
		},
	}
}

// IsProduction reports whether the binary is running under the production
// environment. Used to decide whether an unbindable port is fatal.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// Validate rejects a production configuration that would emit unreachable
// URLs. detent.build has no www host, and .build is on the HSTS preload list,
// so an http or www SITE_URL does not merely look wrong — every canonical,
// og:url, and sitemap entry built from it points somewhere a crawler cannot
// follow. Failing at boot beats shipping a site full of dead canonicals.
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}

	u := strings.TrimSuffix(c.Site.URL, "/")
	switch {
	case u == "":
		return errors.New("SITE_URL must be set in production")
	case strings.HasPrefix(u, "http://"):
		return fmt.Errorf("SITE_URL %q must use https: .build is HSTS-preloaded, so http URLs are unreachable", u)
	case !strings.HasPrefix(u, "https://"):
		return fmt.Errorf("SITE_URL %q must be an absolute https URL", u)
	case strings.Contains(u, "://www."):
		return fmt.Errorf("SITE_URL %q uses a www host, which has no DNS record", u)
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
