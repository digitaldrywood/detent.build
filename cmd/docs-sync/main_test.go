package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTree(t *testing.T) {
	raw := []byte("100644 blob abc123\tdocs/config.md\x00100755 blob def456\tdocs/examples/run.sh\x00")

	got, err := parseTree(raw)
	if err != nil {
		t.Fatalf("parseTree() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parseTree() returned %d entries, want 2", len(got))
	}
	if got[0].Path != "docs/config.md" || got[0].OID != "abc123" || got[0].Mode != "100644" {
		t.Errorf("first entry = %#v", got[0])
	}
	if got[1].Path != "docs/examples/run.sh" || got[1].OID != "def456" || got[1].Mode != "100755" {
		t.Errorf("second entry = %#v", got[1])
	}
}

func TestParseTreeRejectsMalformedEntry(t *testing.T) {
	if _, err := parseTree([]byte("100644 blob abc123 docs/config.md\x00")); err == nil {
		t.Fatal("parseTree() accepted an entry without a tab separator")
	}
}

func TestSafeRelativePath(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "docs/config.md", want: "config.md", ok: true},
		{path: "docs/examples/demo/README.md", want: "examples/demo/README.md", ok: true},
		{path: "README.md"},
		{path: "docs/"},
		{path: "docs/../README.md"},
		{path: "docs/examples/../../README.md"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := safeRelativePath(tt.path)
			if tt.ok && err != nil {
				t.Fatalf("safeRelativePath() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("safeRelativePath() = %q, want an error", got)
			}
			if got != tt.want {
				t.Errorf("safeRelativePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileMode(t *testing.T) {
	for _, mode := range []string{"100644", "100755"} {
		if _, err := fileMode(mode); err != nil {
			t.Errorf("fileMode(%q) error = %v", mode, err)
		}
	}
	if _, err := fileMode("120000"); err == nil {
		t.Fatal("fileMode() accepted a symbolic link")
	}
}

func TestPreservedSyncTime(t *testing.T) {
	current := manifest{
		Schema:       1,
		Repository:   sourceRepository,
		Tag:          releaseTag,
		TagObjectSHA: tagObjectSHA,
		CommitSHA:    commitSHA,
		Files:        []manifestFile{{Path: "config.md", SHA256: "abc"}},
	}
	previous := current
	previous.SyncedAt = "2026-08-08T12:00:00Z"
	contents, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	if got := preservedSyncTime(path, current); got != previous.SyncedAt {
		t.Errorf("preservedSyncTime() = %q, want %q", got, previous.SyncedAt)
	}

	current.Files[0].SHA256 = "changed"
	if got := preservedSyncTime(path, current); got == previous.SyncedAt {
		t.Errorf("preservedSyncTime() reused %q for changed content", got)
	}
}

func TestPublishSnapshotRollsBackOnRenameFailure(t *testing.T) {
	docsDir := t.TempDir()
	writeSnapshot(t, docsDir, "old")
	stagedRoot := filepath.Join(docsDir, ".docs-staging-test")
	writeSnapshot(t, stagedRoot, "new")

	failPath := filepath.Join(stagedRoot, "manifest.json")
	rename := func(oldPath, newPath string) error {
		if oldPath == failPath {
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}

	if err := publishSnapshotWithRename(docsDir, stagedRoot, rename); err == nil {
		t.Fatal("publishSnapshotWithRename() succeeded despite the injected failure")
	}
	assertSnapshot(t, docsDir, "old")
}

func TestRecoverInterruptedPublication(t *testing.T) {
	docsDir := t.TempDir()
	backupRoot := filepath.Join(docsDir, ".docs-backup-test")
	writeSnapshot(t, backupRoot, "old")
	stateBytes, err := json.Marshal(publicationState{HadVendor: true, HadManifest: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupRoot, "state.json"), stateBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "vendor", "doc.md"), []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := recoverInterruptedPublications(docsDir); err != nil {
		t.Fatalf("recoverInterruptedPublications() error = %v", err)
	}
	assertSnapshot(t, docsDir, "old")
	if _, err := os.Stat(backupRoot); !os.IsNotExist(err) {
		t.Errorf("backup still exists after recovery: %v", err)
	}
}

func TestRecoverInterruptedCleanupKeepsPublishedSnapshot(t *testing.T) {
	docsDir := t.TempDir()
	writeSnapshot(t, docsDir, "new")
	cleanupRoot := filepath.Join(docsDir, ".docs-cleanup-test")
	writeSnapshot(t, cleanupRoot, "old")

	if err := recoverInterruptedPublications(docsDir); err != nil {
		t.Fatalf("recoverInterruptedPublications() error = %v", err)
	}
	assertSnapshot(t, docsDir, "new")
	if _, err := os.Stat(cleanupRoot); !os.IsNotExist(err) {
		t.Errorf("cleanup journal still exists after recovery: %v", err)
	}
}

func TestAcquireFileLockRejectsOverlap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs-sync.lock")
	first, err := acquireFileLock(path)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	if second, err := acquireFileLock(path); err == nil {
		_ = second.Close()
		t.Fatal("acquireFileLock() allowed an overlapping lock")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}
	third, err := acquireFileLock(path)
	if err != nil {
		t.Fatalf("reacquire lock: %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func writeSnapshot(t *testing.T, root, value string) {
	t.Helper()
	vendorDir := filepath.Join(root, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "doc.md"), []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertSnapshot(t *testing.T, root, want string) {
	t.Helper()
	for _, path := range []string{"vendor/doc.md", "manifest.json"} {
		contents, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if got := string(contents); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}
