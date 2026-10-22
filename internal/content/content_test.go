package content

import (
	"context"
	"strings"
	"testing"

	"detent.build/internal/ctxkeys"
	"detent.build/internal/docs"
)

func TestVersionMatchesBare(t *testing.T) {
	if got := strings.TrimPrefix(Version, "v"); got != VersionBare {
		t.Errorf("Version without prefix = %q, want VersionBare %q", got, VersionBare)
	}
}

func TestPlansMatchHostedCapacityCatalog(t *testing.T) {
	want := []struct {
		name           string
		price          string
		concurrentWork int
	}{
		{"Free", "0", 2},
		{"Starter", "49", 8},
		{"Team", "99", 20},
		{"Growth", "249", 50},
		{"Scale", "", 0},
	}

	if len(Plans) != len(want) {
		t.Fatalf("Plans has %d tiers, want %d", len(Plans), len(want))
	}
	for i, w := range want {
		p := Plans[i]
		if p.Name != w.name || p.Price != w.price || p.ConcurrentWork != w.concurrentWork {
			t.Errorf("Plans[%d] = %s $%s %d, want %s $%s %d", i, p.Name, p.Price, p.ConcurrentWork, w.name, w.price, w.concurrentWork)
		}
		if p.ConcurrentWork > MaxPlanConcurrentWork {
			t.Errorf("%s concurrent work %d exceeds MaxPlanConcurrentWork %d", p.Name, p.ConcurrentWork, MaxPlanConcurrentWork)
		}
	}
	if Plans[0].HostedArtifact {
		t.Error("Free plan claims hosted artifacts; only paid tiers add them")
	}
}

func TestCloudLinksAvoidMissingLoginRoute(t *testing.T) {
	for _, link := range []string{CloudURL, CloudSignUpURL} {
		if !strings.HasPrefix(link, "https://cloud.detent.build/") {
			t.Errorf("Cloud link %q is not on cloud.detent.build", link)
		}
		if strings.Contains(link, "/login") {
			t.Errorf("Cloud link %q uses /login, which cloud.detent.build does not serve", link)
		}
	}
}

func TestVersionFromCtx(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"release watcher value", context.WithValue(context.Background(), ctxkeys.Version, "v1.2.3"), "v1.2.3"},
		{"compiled fallback", context.Background(), Version},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionFromCtx(tt.ctx); got != tt.want {
				t.Errorf("VersionFromCtx() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocUsesPublishedPathOrPinnedBlob(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		want       string
	}{
		{"published", "concepts.md", "/docs/concepts"},
		{"excluded", "development.md", DocsBase + "development.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Doc(tt.sourcePath); got != tt.want {
				t.Errorf("Doc(%q) = %q, want %q", tt.sourcePath, got, tt.want)
			}
		})
	}
}

func TestPinnedDocumentationLinksUseVendoredCommit(t *testing.T) {
	for _, link := range []string{DocsBase} {
		if !strings.Contains(link, docs.CommitSHA) {
			t.Errorf("documentation link %q does not contain pinned commit %s", link, docs.CommitSHA)
		}
	}
}

func TestLineageNamesTheCurrentArchitecture(t *testing.T) {
	var lineage strings.Builder
	lineage.WriteString(LineageHeading)
	lineage.WriteString(LineageSummary)
	for _, fragment := range LineageSymphony {
		lineage.WriteString(fragment.Text)
	}
	for _, fragment := range LineageHistory {
		lineage.WriteString(fragment.Text)
	}
	for _, divergence := range LineageDivergences {
		lineage.WriteString(divergence.Title)
		for _, fragment := range divergence.Body {
			lineage.WriteString(fragment.Text)
		}
	}

	for _, want := range []string{
		"native board",
		"Change Requests",
		"GitHub is an optional integration",
		"Runners you enroll",
	} {
		if !strings.Contains(lineage.String(), want) {
			t.Errorf("lineage does not contain %q", want)
		}
	}
}
