package config

import "testing"

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
			cfg := &Config{Env: "production", Site: SiteConfig{URL: tt.siteURL}}

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
