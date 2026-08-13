package content

import "testing"

func TestVersionAvailabilityEvidenceIsComplete(t *testing.T) {
	featureIDs := make(map[string]struct{}, len(versionAvailability))
	for _, entry := range versionAvailability {
		if entry.FeatureID == "" || entry.FeatureName == "" || entry.SourcePath == "" || entry.IntroducedBy == "" ||
			entry.VerifiedVersion == "" || entry.VerificationMethod == "" || entry.Confidence == "" {
			t.Errorf("version availability entry %#v is incomplete", entry)
		}
		if _, exists := featureIDs[entry.FeatureID]; exists {
			t.Errorf("duplicate version availability feature id %q", entry.FeatureID)
		}
		featureIDs[entry.FeatureID] = struct{}{}
	}
}

func TestVersionAvailabilityBadgeLabelsFollowConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence VersionAvailabilityConfidence
		want       string
	}{
		{"available", VersionAvailabilityAvailable, "Available in v1.2.3"},
		{"introduced", VersionAvailabilityIntroduced, "Introduced in v1.2.3"},
		{"present by", VersionAvailabilityPresentBy, "Present by v1.2.3"},
		{"insufficient", VersionAvailabilityInsufficient, ""},
		{"unknown", VersionAvailabilityConfidence("unknown"), ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := VersionAvailability{VerifiedVersion: "v1.2.3", Confidence: test.confidence}
			if got := entry.BadgeLabel(); got != test.want {
				t.Errorf("BadgeLabel() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestVersionAvailabilityRequiresExplicitRecord(t *testing.T) {
	if entries := VersionAvailabilityForSource("config.md"); len(entries) != 0 {
		t.Errorf("unrecorded source returned version availability: %#v", entries)
	}
	entries := VersionAvailabilityForSource("workflow-overlays.md")
	if len(entries) != 1 || entries[0].FeatureID != "machine-local-workflow-overlays" {
		t.Errorf("recorded source returned version availability %#v", entries)
	}
}
