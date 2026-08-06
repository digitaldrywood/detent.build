package config

import (
	"strings"
	"testing"
)

func TestValidateRejectsUnreachableProductionURLs(t *testing.T) {
	tests := []struct {
		name    string
		siteURL string
		wantErr bool
	}{
		{"apex https", "https://detent.build", false},
		{"apex https with trailing slash", "https://detent.build/", false},
		{"plain http", "http://detent.build", true},
		{"www host", "https://www.detent.build", true},
		{"www over http", "http://www.detent.build", true},
		{"relative", "/", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Env:             "production",
				SiteURLExplicit: tt.siteURL != "",
				Site:            SiteConfig{URL: tt.siteURL},
			}

			err := cfg.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("SITE_URL %q was accepted in production but is unreachable", tt.siteURL)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SITE_URL %q was rejected: %v", tt.siteURL, err)
			}
		})
	}
}

// Development runs on http://localhost and must not be held to the production
// rules, or `make dev` stops working.
func TestValidateIgnoresDevelopment(t *testing.T) {
	cfg := &Config{Env: "development", Site: SiteConfig{URL: "http://localhost:3000"}}

	if err := cfg.Validate(); err != nil {
		t.Errorf("development config rejected: %v", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"PORT", "ENV", "SITE_NAME", "SITE_URL", "DEFAULT_OG_IMAGE", "TAILSCALE_HOSTNAME"} {
		t.Setenv(key, "")
	}

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("default port = %q, want 3000; the Dokploy domain entry is bound to 3000", cfg.Port)
	}
	if cfg.Site.DefaultOGImage == "" {
		t.Error("DefaultOGImage is empty, so no page would emit a social card")
	}
	if cfg.TailscaleHostname != "" {
		t.Errorf("TailscaleHostname = %q, want empty when unset", cfg.TailscaleHostname)
	}
}

// SITE_URL falling back to its default in production is the failure that
// shipped localhost canonicals. An unset variable must be rejected even though
// the default value itself is well-formed.
func TestValidateRejectsUnsetSiteURLInProduction(t *testing.T) {
	cfg := &Config{
		Env:             "production",
		SiteURLExplicit: false,
		Site:            SiteConfig{URL: "http://localhost:3000"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("an unset SITE_URL was accepted in production")
	}
}

// Validate only runs in production, so an unset ENV disables it entirely.
// Warnings is what makes that visible in the deployment log.
func TestWarningsFlagUnsetEnvAndSiteURL(t *testing.T) {
	for _, key := range []string{"ENV", "SITE_URL", "PORT", "SITE_NAME", "DEFAULT_OG_IMAGE", "TAILSCALE_HOSTNAME"} {
		t.Setenv(key, "")
	}

	got := Load().Warnings()

	if len(got) != 2 {
		t.Fatalf("Warnings() returned %d entries, want 2: %v", len(got), got)
	}
	for _, want := range []string{"ENV is not set", "SITE_URL is not set"} {
		found := false
		for _, w := range got {
			if strings.Contains(w, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("Warnings() does not mention %q: %v", want, got)
		}
	}
}

func TestNoWarningsWhenConfigured(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("SITE_URL", "https://detent.build")

	if got := Load().Warnings(); len(got) != 0 {
		t.Errorf("Warnings() = %v, want none for a fully configured environment", got)
	}
}
