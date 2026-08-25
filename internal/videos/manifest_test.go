package videos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedManifestIsValidAndAssetsExist(t *testing.T) {
	entries := Published.Entries()
	if len(entries) == 0 {
		t.Fatal("published video manifest is empty")
	}

	for _, entry := range entries {
		t.Run(entry.VideoID, func(t *testing.T) {
			for _, asset := range []string{entry.ThumbnailPath, entry.Transcript} {
				if !strings.HasPrefix(asset, "/static/") {
					continue
				}
				assetPath := filepath.Join("..", "..", filepath.FromSlash(strings.TrimPrefix(asset, "/")))
				info, err := os.Stat(assetPath)
				if err != nil {
					t.Fatalf("required asset %q: %v", asset, err)
				}
				if info.IsDir() || info.Size() == 0 {
					t.Errorf("required asset %q is not a non-empty file", asset)
				}
			}
		})
	}
}

func TestManifestRejectsMultipleFeaturedEntries(t *testing.T) {
	entries := Published.Entries()
	entries[1].Placement = PlacementFeatured
	entries[1].DocsPath = ""

	err := Validate(Manifest{
		SchemaVersion:    SchemaVersion,
		YouTubeChannelID: ChannelID,
		Entries:          entries,
	})
	if err == nil || !strings.Contains(err.Error(), "at most 1") {
		t.Fatalf("Validate() error = %v, want featured constraint", err)
	}
}

func TestManifestRejectsNonCanonicalWatchURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "plain HTTP short link", url: "http://youtu.be/WUob6ZTzqqk"},
		{name: "unexpected host", url: "https://youtube.com/watch?v=WUob6ZTzqqk"},
		{name: "duplicate video query", url: "https://www.youtube.com/watch?v=WUob6ZTzqqk&v=QvED7RHTKkI"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := Published.Entries()[0]
			entry.WatchURL = test.url

			err := Validate(Manifest{
				SchemaVersion:    SchemaVersion,
				YouTubeChannelID: ChannelID,
				Entries:          []Entry{entry},
			})
			if err == nil || !strings.Contains(err.Error(), "canonical https://www.youtube.com/watch") {
				t.Fatalf("Validate() error = %v, want canonical watch URL constraint", err)
			}
		})
	}
}

func TestManifestRejectsUnexpectedChannel(t *testing.T) {
	err := Validate(Manifest{
		SchemaVersion:    SchemaVersion,
		YouTubeChannelID: "unexpected",
		Entries:          Published.Entries(),
	})
	if err == nil || !strings.Contains(err.Error(), "youtube_channel_id") {
		t.Fatalf("Validate() error = %v, want channel constraint", err)
	}
}

func TestManifestRejectsUnknownFields(t *testing.T) {
	data := strings.Replace(string(manifestJSON), `"schema_version": 1`, `"schema_version": 1, "engagement_metrics": {}`, 1)
	if _, err := Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Parse() error = %v, want unknown-field rejection", err)
	}
}

func TestPublishedPlacementsResolve(t *testing.T) {
	featured, ok := Published.Featured()
	if !ok || featured.VideoID != "WUob6ZTzqqk" {
		t.Fatalf("Featured() = %q, %v", featured.VideoID, ok)
	}

	docsEntries := Published.ForDocs("/docs/configuration")
	if len(docsEntries) != 1 || docsEntries[0].VideoID != "QvED7RHTKkI" {
		t.Fatalf("ForDocs() = %#v", docsEntries)
	}
	if entries := Published.ForDocs("/docs/getting-started"); len(entries) != 0 {
		t.Fatalf("unexpected docs-adjacent entries: %#v", entries)
	}
}
