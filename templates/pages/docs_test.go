package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"detent.build/internal/content"
)

func TestVersionAvailabilityRendererOmitsInsufficientConfidence(t *testing.T) {
	entries := []content.VersionAvailability{
		{FeatureID: "available", FeatureName: "Available", IntroducedBy: "https://example.com/available", VerifiedVersion: "v1.0.0", Confidence: content.VersionAvailabilityAvailable},
		{FeatureID: "introduced", FeatureName: "Introduced", IntroducedBy: "https://example.com/introduced", VerifiedVersion: "v2.0.0", Confidence: content.VersionAvailabilityIntroduced},
		{FeatureID: "present-by", FeatureName: "Present by", IntroducedBy: "https://example.com/present", VerifiedVersion: "v3.0.0", Confidence: content.VersionAvailabilityPresentBy},
		{FeatureID: "insufficient", FeatureName: "Insufficient", IntroducedBy: "https://example.com/insufficient", VerifiedVersion: "v4.0.0", Confidence: content.VersionAvailabilityInsufficient},
	}

	var output bytes.Buffer
	if err := versionAvailability(entries).Render(context.Background(), &output); err != nil {
		t.Fatalf("render version availability: %v", err)
	}
	rendered := output.String()
	for _, want := range []string{"Available in", "v1.0.0", "Introduced in", "v2.0.0", "Present by", "v3.0.0"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered badges do not contain %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{"data-version-feature=\"insufficient\"", "v4.0.0"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("insufficient-confidence entry rendered %q: %s", forbidden, rendered)
		}
	}
}
