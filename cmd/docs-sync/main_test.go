package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"detent.build/internal/docsregistry"
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

	current.Files[0].Generated = true
	if got := preservedSyncTime(path, current); got == previous.SyncedAt {
		t.Errorf("preservedSyncTime() reused %q for changed provenance", got)
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

func TestSyncVerifiedDocsGeneratedSnapshot(t *testing.T) {
	for _, tt := range []struct {
		name      string
		committed bool
		legacy    bool
		recipe    string
		wantErr   string
	}{
		{name: "publishes config absent from git"},
		{name: "regenerates committed config", committed: true},
		{name: "regenerates legacy config input", committed: true, legacy: true},
		{name: "failed generation preserves publication", committed: true, recipe: `if err := os.WriteFile(output, []byte("partial"), 0644); err != nil { panic(err) }; fmt.Fprintln(os.Stdout, "fixture-generation-stdout"); fmt.Fprintln(os.Stderr, "fixture-generation-failed"); os.Exit(1)`, wantErr: "fixture-generation-failed"},
		{name: "missing output preserves publication", recipe: `return`, wantErr: "inspect upstream document docs/config.md"},
		{name: "missing output rejects stale copy", committed: true, recipe: `return`, wantErr: "inspect upstream document docs/config.md"},
		{name: "changed authored content preserves publication", recipe: `if err := os.WriteFile(filepath.Join(*root, "docs", "doc.md"), []byte("changed"), 0644); err != nil { panic(err) }`, wantErr: "git object mismatch for docs/doc.md"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			upstream, sourceCommit, sourceTag := generatedDocsFixture(t, tt.committed, tt.legacy, tt.recipe)
			docsDir := t.TempDir()
			writeSnapshot(t, docsDir, "old")
			previous := manifest{Schema: 1, Repository: sourceRepository, Tag: releaseTag,
				TagObjectSHA: sourceTag, CommitSHA: sourceCommit, SyncedAt: "2026-08-08T12:00:00Z",
				Files: []manifestFile{{Path: "config.md", SHA256: "old"}, {Path: "doc.md", SHA256: "old"}},
			}
			previousBytes, err := json.Marshal(previous)
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(docsDir, "manifest.json")
			if err := os.WriteFile(manifestPath, previousBytes, 0o644); err != nil {
				t.Fatal(err)
			}
			registry := docsregistry.Registry{Pages: []docsregistry.Page{
				{SourcePath: "config.md", PublicPath: "/docs/configuration", Origin: docsregistry.OriginUpstream},
			}}
			err = syncVerifiedDocs(upstream, docsDir, t.TempDir(), sourceTag, sourceCommit, registry)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("syncVerifiedDocs() error = %v, want %q", err, tt.wantErr)
				}
				if tt.wantErr == "fixture-generation-failed" && !strings.Contains(err.Error(), "fixture-generation-stdout") {
					t.Errorf("generator stdout missing from error: %v", err)
				}
				assertSnapshotFile(t, docsDir, "vendor/doc.md", "old")
				assertSnapshotFile(t, docsDir, "manifest.json", string(previousBytes))
				entries, err := os.ReadDir(filepath.Join(docsDir, "vendor"))
				if err != nil || len(entries) != 1 {
					t.Fatalf("prior vendor inventory changed: %v, %v", entries, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, exists, err := readManifest(manifestPath)
			if err != nil || !exists {
				t.Fatalf("read published manifest: exists=%v, error=%v", exists, err)
			}
			if got.Repository != sourceRepository || got.Tag != releaseTag || got.TagObjectSHA != sourceTag || got.CommitSHA != sourceCommit {
				t.Errorf("source provenance changed: %#v", got)
			}
			want := map[string]string{
				"config.md": "# Configuration\ntyped defaults\n",
				"doc.md":    "authored documentation\n",
			}
			if len(got.Files) != len(want) {
				t.Fatalf("manifest files = %#v, want exactly authored and generated documents", got.Files)
			}
			for _, file := range got.Files {
				contents, ok := want[file.Path]
				if !ok {
					t.Fatalf("unexpected published file %q", file.Path)
				}
				if file.Generated != (file.Path == "config.md") {
					t.Errorf("manifest generated flag for %s = %v", file.Path, file.Generated)
				}
				assertSnapshotFile(t, docsDir, "vendor/"+file.Path, contents)
				digest := sha256.Sum256([]byte(contents))
				if file.SHA256 != hex.EncodeToString(digest[:]) {
					t.Errorf("manifest digest for %s = %s, want digest of generated content", file.Path, file.SHA256)
				}
			}
			if _, err := os.Stat(filepath.Join(docsDir, "vendor", "config.md.in")); !os.IsNotExist(err) {
				t.Errorf("config template published: %v", err)
			}
		})
	}
}

// The generator consumes inputs outside docs/ and uses -root to locate them.
// No Makefile or capability generator is available in this fixture.
func generatedDocsFixture(t *testing.T, committed, legacy bool, recipe string) (string, string, string) {
	t.Helper()
	inputs := t.TempDir()
	files := map[string]string{
		"docs/doc.md":                  "authored documentation\n",
		"docs/config.md.in":            "# Configuration\n",
		"internal/config/defaults.txt": "typed defaults\n",
		"go.mod":                       "module fixture\n\ngo 1.25\n",
		"internal/config/cmd/configdoc/main.go": `package main
import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
)
func main() {
    root := flag.String("root", "", "source root")
    flag.Parse()
    if *root == "" { panic("missing -root") }
    if os.Getenv("GOTOOLCHAIN") != "go1.26.6" { panic("unexpected toolchain") }
    template, err := os.ReadFile(filepath.Join(*root, "docs", "config.md.in"))
    if err != nil { panic(err) }
    defaults, err := os.ReadFile(filepath.Join(*root, "internal", "config", "defaults.txt"))
    if err != nil { panic(err) }
    output := filepath.Join(*root, "docs", "config.md")
` + recipe + `
    if err := os.WriteFile(output, append(template, defaults...), 0644); err != nil { panic(err) }
    fmt.Println("generated config")
}
`,
	}
	if committed {
		files["docs/config.md"] = "stale committed config\n"
	}
	if legacy {
		delete(files, "docs/config.md.in")
		files["docs/config.md"] = "# Configuration\n"
		files["internal/config/cmd/configdoc/main.go"] = strings.ReplaceAll(files["internal/config/cmd/configdoc/main.go"], `"config.md.in"`, `"config.md"`)
	}
	for path, contents := range files {
		destination := filepath.Join(inputs, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	upstream := filepath.Join(t.TempDir(), "upstream.git")
	if _, err := gitBytes("", nil, "init", "--bare", "--quiet", upstream); err != nil {
		t.Fatal(err)
	}
	if _, err := gitBytes(upstream, nil, "--work-tree="+inputs, "add", "--all"); err != nil {
		t.Fatal(err)
	}
	tree, err := gitText(upstream, nil, "write-tree")
	if err != nil {
		t.Fatal(err)
	}
	commit, err := gitText(upstream, nil, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "commit-tree", tree, "-m", "authored source fixture")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := gitText(upstream, []byte(fmt.Sprintf("object %s\ntype commit\ntag fixture\ntagger Fixture <fixture@example.test> 1786190400 +0000\n\nFixture\n", commit)), "mktag")
	if err != nil {
		t.Fatal(err)
	}
	return upstream, commit, tag
}

func TestGenerateConfigDocumentationRestoresDeletedLegacyInput(t *testing.T) {
	upstream, commit, _ := generatedDocsFixture(t, true, true, "")
	sourceRoot := t.TempDir()
	if _, err := gitBytes(upstream, nil, "--work-tree="+sourceRoot, "checkout", "--force", commit, "--", "."); err != nil {
		t.Fatal(err)
	}
	tree, err := gitBytes(upstream, nil, "ls-tree", "-r", "-z", commit, "--", "docs")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseTree(tree)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(sourceRoot, "docs", "config.md")
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	if err := generateConfigDocumentation(upstream, sourceRoot, entries); err != nil {
		t.Fatal(err)
	}
	assertSnapshotFile(t, sourceRoot, "docs/config.md", "# Configuration\ntyped defaults\n")
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

func TestRecoverRemovesInactiveSyncDirectories(t *testing.T) {
	docsDir := t.TempDir()
	writeSnapshot(t, docsDir, "current")
	stale := []string{
		filepath.Join(docsDir, ".docs-preparing-test"),
		filepath.Join(docsDir, ".docs-staging-test"),
	}
	for _, path := range stale {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := recoverInterruptedPublications(docsDir); err != nil {
		t.Fatalf("recoverInterruptedPublications() error = %v", err)
	}
	assertSnapshot(t, docsDir, "current")
	for _, path := range stale {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("inactive directory still exists after recovery: %s: %v", path, err)
		}
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

func TestInventoryChangesClassifiesAddDeleteAndProbableRename(t *testing.T) {
	previous := []manifestFile{
		{Path: "removed.md", SHA256: "removed"},
		{Path: "old-name.md", SHA256: "same"},
		{Path: "unchanged.md", SHA256: "unchanged"},
	}
	current := []manifestFile{
		{Path: "added.md", SHA256: "added"},
		{Path: "new-name.md", SHA256: "same"},
		{Path: "unchanged.md", SHA256: "changed-in-place"},
	}

	changes, err := inventoryChanges(previous, current, nil)
	if err != nil {
		t.Fatalf("inventoryChanges() error = %v", err)
	}
	want := map[string]struct{}{
		inventoryKey(docsregistry.ChangeAdded, "", "added.md"):                        {},
		inventoryKey(docsregistry.ChangeDeleted, "removed.md", ""):                    {},
		inventoryKey(docsregistry.ChangeProbableRename, "old-name.md", "new-name.md"): {},
	}
	if len(changes) != len(want) {
		t.Fatalf("changes = %#v", changes)
	}
	for _, change := range changes {
		key := inventoryKey(change.Kind, change.PreviousPath, change.CurrentPath)
		if _, exists := want[key]; !exists {
			t.Errorf("unexpected change %#v", change)
		}
	}
}

func TestInventoryChangesUsesOperatorRenameClassification(t *testing.T) {
	decision := docsregistry.InventoryDecision{
		Kind:         docsregistry.ChangeProbableRename,
		PreviousPath: "old-name.md",
		CurrentPath:  "new-name.md",
	}
	changes, err := inventoryChanges(
		[]manifestFile{{Path: "old-name.md", SHA256: "old"}},
		[]manifestFile{{Path: "new-name.md", SHA256: "new"}},
		[]docsregistry.InventoryDecision{decision},
	)
	if err != nil {
		t.Fatalf("inventoryChanges() error = %v", err)
	}
	if len(changes) != 1 || changes[0].Kind != docsregistry.ChangeProbableRename {
		t.Errorf("changes = %#v", changes)
	}
}

func TestValidateInventoryChangesRejectsUnclassifiedChange(t *testing.T) {
	err := validateInventoryChanges([]inventoryChange{{
		Kind:        docsregistry.ChangeAdded,
		CurrentPath: "added.md",
	}}, nil, docsregistry.Registry{})
	if err == nil || !strings.Contains(err.Error(), "unclassified") {
		t.Fatalf("validateInventoryChanges() error = %v, want unclassified change", err)
	}
}

func TestValidateInventoryDecisionRoutePolicies(t *testing.T) {
	registry := docsregistry.Registry{
		Pages: []docsregistry.Page{
			{SourcePath: "stable-new.md", PublicPath: "/docs/stable", Origin: docsregistry.OriginUpstream},
			{SourcePath: "alias-new.md", PublicPath: "/docs/new", Origin: docsregistry.OriginUpstream},
		},
		Aliases:    []docsregistry.Alias{{PublicPath: "/docs/old", CanonicalPath: "/docs/new"}},
		Tombstones: []docsregistry.Tombstone{{PublicPath: "/docs/removed"}},
	}
	decisions := []docsregistry.InventoryDecision{
		{
			Kind:         docsregistry.ChangeProbableRename,
			PreviousPath: "stable-old.md",
			CurrentPath:  "stable-new.md",
			Resolution:   docsregistry.ResolutionStable,
			PublicPath:   "/docs/stable",
		},
		{
			Kind:          docsregistry.ChangeProbableRename,
			PreviousPath:  "alias-old.md",
			CurrentPath:   "alias-new.md",
			Resolution:    docsregistry.ResolutionAlias,
			PublicPath:    "/docs/old",
			CanonicalPath: "/docs/new",
		},
		{
			Kind:         docsregistry.ChangeDeleted,
			PreviousPath: "removed.md",
			Resolution:   docsregistry.ResolutionTombstone,
			PublicPath:   "/docs/removed",
		},
	}
	for _, decision := range decisions {
		if err := validateInventoryDecision(decision, registry); err != nil {
			t.Errorf("validateInventoryDecision(%s) error = %v", decision.Resolution, err)
		}
	}
}

func TestValidatePublishedSourcesRejectsOrphanedPage(t *testing.T) {
	err := validatePublishedSources(nil, docsregistry.Registry{Pages: []docsregistry.Page{{
		SourcePath: "missing.md",
		PublicPath: "/docs/missing",
		Origin:     docsregistry.OriginUpstream,
	}}})
	if err == nil || !strings.Contains(err.Error(), "no incoming vendored source") {
		t.Fatalf("validatePublishedSources() error = %v", err)
	}
}

func TestValidatePublishedSourcesIgnoresSiteAuthoredPage(t *testing.T) {
	err := validatePublishedSources(nil, docsregistry.Registry{Pages: []docsregistry.Page{{
		SourcePath: "site.md",
		PublicPath: "/docs/site/guide",
		Origin:     docsregistry.OriginSite,
	}}})
	if err != nil {
		t.Fatalf("validatePublishedSources() error = %v", err)
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
		assertSnapshotFile(t, root, path, want)
	}
}

func assertSnapshotFile(t *testing.T, root, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(contents); got != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
