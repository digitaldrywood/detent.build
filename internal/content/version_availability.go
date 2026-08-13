package content

type VersionAvailabilityConfidence string

const (
	VersionAvailabilityAvailable    VersionAvailabilityConfidence = "available"
	VersionAvailabilityIntroduced   VersionAvailabilityConfidence = "introduced"
	VersionAvailabilityPresentBy    VersionAvailabilityConfidence = "present_by"
	VersionAvailabilityInsufficient VersionAvailabilityConfidence = "insufficient"
)

type VersionAvailability struct {
	FeatureID          string
	FeatureName        string
	SourcePath         string
	IntroducedBy       string
	VerifiedVersion    string
	VerificationMethod string
	Confidence         VersionAvailabilityConfidence
}

func VersionAvailabilityForSource(sourcePath string) []VersionAvailability {
	var entries []VersionAvailability
	for _, entry := range versionAvailability {
		if entry.SourcePath == sourcePath {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (entry VersionAvailability) BadgeLabel() string {
	prefix := entry.BadgePrefix()
	if prefix == "" || entry.VerifiedVersion == "" {
		return ""
	}
	return prefix + " " + entry.VerifiedVersion
}

func (entry VersionAvailability) BadgePrefix() string {
	switch entry.Confidence {
	case VersionAvailabilityAvailable:
		return "Available in"
	case VersionAvailabilityIntroduced:
		return "Introduced in"
	case VersionAvailabilityPresentBy:
		return "Present by"
	default:
		return ""
	}
}
