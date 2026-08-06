package config

import (
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

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
