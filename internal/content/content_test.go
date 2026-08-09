package content

import (
	"strings"
	"testing"

	"detent.build/internal/docs"
)

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
