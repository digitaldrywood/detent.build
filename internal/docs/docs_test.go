package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"path"
	"sort"
	"testing"
	"time"
)

type provenanceManifest struct {
	Schema       int              `json:"schema"`
	Repository   string           `json:"repository"`
	Tag          string           `json:"tag"`
	TagObjectSHA string           `json:"tag_object_sha"`
	CommitSHA    string           `json:"commit_sha"`
	SyncedAt     string           `json:"synced_at"`
	Files        []provenanceFile `json:"files"`
}

type provenanceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// TestVendoredContentMatchesManifest is an offline tamper-evidence check. It
// detects local edits to the vendored bytes, but does not prove those bytes
// came from upstream; make docs-sync performs that networked provenance check.
func TestVendoredContentMatchesManifest(t *testing.T) {
	contents, err := fs.ReadFile(embedded, "manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest provenanceManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Schema != 1 {
		t.Errorf("manifest schema = %d, want 1", manifest.Schema)
	}
	if manifest.Repository != SourceRepository {
		t.Errorf("manifest repository = %q, want %q", manifest.Repository, SourceRepository)
	}
	if manifest.Tag != ReleaseTag {
		t.Errorf("manifest tag = %q, want %q", manifest.Tag, ReleaseTag)
	}
	if manifest.TagObjectSHA != TagObjectSHA {
		t.Errorf("manifest tag object = %q, want %q", manifest.TagObjectSHA, TagObjectSHA)
	}
	if manifest.CommitSHA != CommitSHA {
		t.Errorf("manifest commit = %q, want %q", manifest.CommitSHA, CommitSHA)
	}
	if _, err := time.Parse(time.RFC3339, manifest.SyncedAt); err != nil {
		t.Errorf("manifest sync timestamp %q is not RFC 3339: %v", manifest.SyncedAt, err)
	}

	want := make(map[string]string, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.Path == "" || path.Clean(file.Path) != file.Path || path.IsAbs(file.Path) {
			t.Errorf("manifest contains unsafe path %q", file.Path)
			continue
		}
		if _, exists := want[file.Path]; exists {
			t.Errorf("manifest contains duplicate path %q", file.Path)
			continue
		}
		want[file.Path] = file.SHA256
	}

	var got []string
	err = fs.WalkDir(Files, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		got = append(got, filePath)
		contents, readErr := fs.ReadFile(Files, filePath)
		if readErr != nil {
			return readErr
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		expected, exists := want[filePath]
		if !exists {
			t.Errorf("vendored file %q is missing from the manifest", filePath)
			return nil
		}
		if actual != expected {
			t.Errorf("vendored file %q has SHA-256 %s, want %s", filePath, actual, expected)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk vendored files: %v", err)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Errorf("vendored tree has %d files, manifest has %d", len(got), len(want))
	}
	for _, filePath := range got {
		delete(want, filePath)
	}
	for missing := range want {
		t.Errorf("manifest file %q is missing from the vendored tree", missing)
	}
}
