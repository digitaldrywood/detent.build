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

func TestFleetMatchesGlobalConfig(t *testing.T) {
	want := []FleetProject{
		{"detent", "1", "0"},
		{"gopher-ai", "1", "3"},
		{"gopher-corp", "1", "3"},
		{"detent.build", "1", "3"},
	}

	if len(Fleet) != len(want) {
		t.Fatalf("Fleet has %d projects, want %d", len(Fleet), len(want))
	}
	for i := range want {
		if Fleet[i] != want[i] {
			t.Errorf("Fleet[%d] = %#v, want %#v", i, Fleet[i], want[i])
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
	for _, link := range []string{DocsBase, NonCodeWorkflowURL} {
		if !strings.Contains(link, docs.CommitSHA) {
			t.Errorf("documentation link %q does not contain pinned commit %s", link, docs.CommitSHA)
		}
	}
}

func TestLineageNamesGitHubStatusSources(t *testing.T) {
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
		"GitHub-native",
		"ProjectV2",
		"boardless issue-field mode",
		"boardless label mode",
		"github_local",
	} {
		if !strings.Contains(lineage.String(), want) {
			t.Errorf("lineage does not contain %q", want)
		}
	}
}
