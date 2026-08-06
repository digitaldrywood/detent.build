package config

import "os"

type SiteConfig struct {
	Name           string
	URL            string
	DefaultOGImage string
}

type Config struct {
	Port string
	Env  string
	Site SiteConfig
}

func Load() *Config {
	return &Config{
		Port: getEnvOrDefault("PORT", "3000"),
		Env:  getEnvOrDefault("ENV", "development"),
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
