package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	sourceRepository = "https://github.com/digitaldrywood/detent"
	sourceRemote     = sourceRepository + ".git"
	releaseTag       = "v0.57.0"
	tagObjectSHA     = "10c9b2a531089e8bac7a3fcd42593b257863ec8d"
	commitSHA        = "1543929187369eca2703abd2a655cf86e9e5d83e"
	docsPrefix       = "docs/"
)

type treeEntry struct {
	Mode string
	Type string
	OID  string
	Path string
}

type manifest struct {
	Schema       int            `json:"schema"`
	Repository   string         `json:"repository"`
	Tag          string         `json:"tag"`
	TagObjectSHA string         `json:"tag_object_sha"`
	CommitSHA    string         `json:"commit_sha"`
	SyncedAt     string         `json:"synced_at"`
	Files        []manifestFile `json:"files"`
}

type manifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type publicationState struct {
	HadVendor   bool `json:"had_vendor"`
	HadManifest bool `json:"had_manifest"`
}

type syncLock struct {
	file *os.File
}

func main() {
	if err := syncDocs(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "docs sync failed: %v\n", err)
		os.Exit(1)
	}
}

func syncDocs() error {
	root, err := gitText("", nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	lock, err := acquireSyncLock(root)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	docsDir := filepath.Join(root, "internal", "docs")
	if err := recoverInterruptedPublications(docsDir); err != nil {
		return err
	}

	temporary, err := os.MkdirTemp("", "detent-docs-sync-")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporary) }()

	upstream := filepath.Join(temporary, "upstream.git")
	if _, err := gitBytes("", nil, "init", "--bare", "--quiet", upstream); err != nil {
		return fmt.Errorf("initialize upstream repository: %w", err)
	}
	if _, err := gitBytes(upstream, nil, "remote", "add", "origin", sourceRemote); err != nil {
		return fmt.Errorf("configure upstream repository: %w", err)
	}
	refspec := "+refs/tags/" + releaseTag + ":refs/tags/" + releaseTag
	if _, err := gitBytes(upstream, nil, "fetch", "--depth=1", "--no-tags", "--quiet", "origin", refspec); err != nil {
		return fmt.Errorf("fetch %s: %w", releaseTag, err)
	}

	resolvedTag, err := gitText(upstream, nil, "rev-parse", "refs/tags/"+releaseTag)
	if err != nil {
		return fmt.Errorf("resolve tag object: %w", err)
	}
	if resolvedTag != tagObjectSHA {
		return fmt.Errorf("tag object mismatch: got %s, want %s", resolvedTag, tagObjectSHA)
	}
	tagType, err := gitText(upstream, nil, "cat-file", "-t", resolvedTag)
	if err != nil {
		return fmt.Errorf("inspect tag object: %w", err)
	}
	if tagType != "tag" {
		return fmt.Errorf("%s resolves to %s, want an annotated tag object", releaseTag, tagType)
	}
	resolvedCommit, err := gitText(upstream, nil, "rev-parse", "refs/tags/"+releaseTag+"^{commit}")
	if err != nil {
		return fmt.Errorf("peel tag to commit: %w", err)
	}
	if resolvedCommit != commitSHA {
		return fmt.Errorf("peeled commit mismatch: got %s, want %s", resolvedCommit, commitSHA)
	}

	tree, err := gitBytes(upstream, nil, "ls-tree", "-r", "-z", "--full-tree", resolvedCommit, "--", "docs")
	if err != nil {
		return fmt.Errorf("list documentation tree: %w", err)
	}
	entries, err := parseTree(tree)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("upstream documentation tree is empty")
	}

	stagedRoot, err := os.MkdirTemp(docsDir, ".docs-staging-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagedRoot) }()
	stagedVendor := filepath.Join(stagedRoot, "vendor")
	if err := os.Mkdir(stagedVendor, 0o755); err != nil {
		return fmt.Errorf("create staged vendor directory: %w", err)
	}

	files := make([]manifestFile, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		rel, err := safeRelativePath(entry.Path)
		if err != nil {
			return err
		}
		if _, ok := seen[rel]; ok {
			return fmt.Errorf("duplicate documentation path %q", rel)
		}
		seen[rel] = struct{}{}
		if entry.Type != "blob" {
			return fmt.Errorf("unsupported git object type %q for %s", entry.Type, entry.Path)
		}
		mode, err := fileMode(entry.Mode)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Path, err)
		}

		contents, err := gitBytes(upstream, nil, "cat-file", "blob", entry.OID)
		if err != nil {
			return fmt.Errorf("read upstream blob for %s: %w", entry.Path, err)
		}

		destination := filepath.Join(stagedVendor, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(destination, contents, mode); err != nil {
			return fmt.Errorf("stage %s: %w", rel, err)
		}
		stagedContents, err := os.ReadFile(destination)
		if err != nil {
			return fmt.Errorf("read staged file %s: %w", rel, err)
		}
		verifiedOID, err := gitText(upstream, stagedContents, "hash-object", "--stdin")
		if err != nil {
			return fmt.Errorf("verify staged file %s: %w", rel, err)
		}
		if verifiedOID != entry.OID {
			return fmt.Errorf("git object mismatch for %s: got %s, want %s", entry.Path, verifiedOID, entry.OID)
		}
		digest := sha256.Sum256(stagedContents)
		files = append(files, manifestFile{Path: rel, SHA256: hex.EncodeToString(digest[:])})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	manifestPath := filepath.Join(docsDir, "manifest.json")
	result := manifest{
		Schema:       1,
		Repository:   sourceRepository,
		Tag:          releaseTag,
		TagObjectSHA: resolvedTag,
		CommitSHA:    resolvedCommit,
		Files:        files,
	}
	result.SyncedAt = preservedSyncTime(manifestPath, result)
	manifestBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	stagedManifest := filepath.Join(stagedRoot, "manifest.json")
	if err := os.WriteFile(stagedManifest, manifestBytes, 0o644); err != nil {
		return fmt.Errorf("stage manifest: %w", err)
	}
	if err := publishSnapshot(docsDir, stagedRoot); err != nil {
		return err
	}

	_, _ = fmt.Printf("vendored %d files from %s at %s (%s)\n", len(files), sourceRepository, releaseTag, resolvedCommit)
	return nil
}

func publishSnapshot(docsDir, stagedRoot string) error {
	return publishSnapshotWithRename(docsDir, stagedRoot, os.Rename)
}

func publishSnapshotWithRename(docsDir, stagedRoot string, rename func(string, string) error) error {
	vendorDir := filepath.Join(docsDir, "vendor")
	manifestPath := filepath.Join(docsDir, "manifest.json")
	hadVendor, err := pathExists(vendorDir)
	if err != nil {
		return fmt.Errorf("inspect vendor tree: %w", err)
	}
	hadManifest, err := pathExists(manifestPath)
	if err != nil {
		return fmt.Errorf("inspect manifest: %w", err)
	}
	state := publicationState{
		HadVendor:   hadVendor,
		HadManifest: hadManifest,
	}
	preparingRoot, err := os.MkdirTemp(docsDir, ".docs-preparing-")
	if err != nil {
		return fmt.Errorf("create publication journal: %w", err)
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		_ = os.RemoveAll(preparingRoot)
		return fmt.Errorf("encode publication state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(preparingRoot, "state.json"), stateBytes, 0o644); err != nil {
		_ = os.RemoveAll(preparingRoot)
		return fmt.Errorf("write publication state: %w", err)
	}
	backupName := strings.Replace(filepath.Base(preparingRoot), ".docs-preparing-", ".docs-backup-", 1)
	backupRoot := filepath.Join(docsDir, backupName)
	if err := rename(preparingRoot, backupRoot); err != nil {
		_ = os.RemoveAll(preparingRoot)
		return fmt.Errorf("activate publication journal: %w", err)
	}

	rollback := func(cause error) error {
		rollbackErr := rollbackPublication(docsDir, backupRoot, state, rename)
		if rollbackErr != nil {
			return errors.Join(cause, fmt.Errorf("rollback publication: %w", rollbackErr))
		}
		if cleanupErr := os.RemoveAll(backupRoot); cleanupErr != nil {
			return errors.Join(cause, fmt.Errorf("remove publication backup: %w", cleanupErr))
		}
		return cause
	}

	if state.HadVendor {
		if err := rename(vendorDir, filepath.Join(backupRoot, "vendor")); err != nil {
			return rollback(fmt.Errorf("back up vendor tree: %w", err))
		}
	}
	if state.HadManifest {
		if err := rename(manifestPath, filepath.Join(backupRoot, "manifest.json")); err != nil {
			return rollback(fmt.Errorf("back up manifest: %w", err))
		}
	}
	if err := rename(filepath.Join(stagedRoot, "vendor"), vendorDir); err != nil {
		return rollback(fmt.Errorf("publish vendor tree: %w", err))
	}
	if err := rename(filepath.Join(stagedRoot, "manifest.json"), manifestPath); err != nil {
		return rollback(fmt.Errorf("publish manifest: %w", err))
	}
	cleanupName := strings.Replace(filepath.Base(backupRoot), ".docs-backup-", ".docs-cleanup-", 1)
	cleanupRoot := filepath.Join(docsDir, cleanupName)
	if err := rename(backupRoot, cleanupRoot); err != nil {
		return rollback(fmt.Errorf("commit publication: %w", err))
	}
	if err := os.RemoveAll(cleanupRoot); err != nil {
		return fmt.Errorf("remove publication backup: %w", err)
	}
	return nil
}

func recoverInterruptedPublications(docsDir string) error {
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return fmt.Errorf("inspect documentation directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		journalRoot := filepath.Join(docsDir, entry.Name())
		if strings.HasPrefix(entry.Name(), ".docs-preparing-") ||
			strings.HasPrefix(entry.Name(), ".docs-staging-") {
			if err := os.RemoveAll(journalRoot); err != nil {
				return fmt.Errorf("remove inactive sync directory: %w", err)
			}
			continue
		}
		if strings.HasPrefix(entry.Name(), ".docs-cleanup-") {
			if err := os.RemoveAll(journalRoot); err != nil {
				return fmt.Errorf("remove committed publication backup: %w", err)
			}
			continue
		}
		if !strings.HasPrefix(entry.Name(), ".docs-backup-") {
			continue
		}
		stateBytes, err := os.ReadFile(filepath.Join(journalRoot, "state.json"))
		if err != nil {
			return fmt.Errorf("read interrupted publication state: %w", err)
		}
		var state publicationState
		if err := json.Unmarshal(stateBytes, &state); err != nil {
			return fmt.Errorf("decode interrupted publication state: %w", err)
		}
		if err := rollbackPublication(docsDir, journalRoot, state, os.Rename); err != nil {
			return fmt.Errorf("recover interrupted publication: %w", err)
		}
		if err := os.RemoveAll(journalRoot); err != nil {
			return fmt.Errorf("remove recovered publication backup: %w", err)
		}
	}
	return nil
}

func rollbackPublication(docsDir, backupRoot string, state publicationState, rename func(string, string) error) error {
	artifacts := []struct {
		name string
		had  bool
	}{
		{name: "vendor", had: state.HadVendor},
		{name: "manifest.json", had: state.HadManifest},
	}
	for _, artifact := range artifacts {
		target := filepath.Join(docsDir, artifact.name)
		backup := filepath.Join(backupRoot, artifact.name)
		backupExists, err := pathExists(backup)
		if err != nil {
			return fmt.Errorf("inspect backup %s: %w", artifact.name, err)
		}
		if backupExists {
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("remove partial %s: %w", artifact.name, err)
			}
			if err := rename(backup, target); err != nil {
				return fmt.Errorf("restore %s: %w", artifact.name, err)
			}
			continue
		}
		if !artifact.had {
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("remove new %s: %w", artifact.name, err)
			}
		}
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func acquireSyncLock(root string) (*syncLock, error) {
	lockPath, err := gitText(root, nil, "rev-parse", "--git-path", "detent-docs-sync.lock")
	if err != nil {
		return nil, fmt.Errorf("resolve docs sync lock: %w", err)
	}
	if !filepath.IsAbs(lockPath) {
		lockPath = filepath.Join(root, lockPath)
	}
	lock, err := acquireFileLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("acquire docs sync lock: %w", err)
	}
	return lock, nil
}

func acquireFileLock(path string) (*syncLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &syncLock{file: file}, nil
}

func (l *syncLock) Close() error {
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return errors.Join(unlockErr, l.file.Close())
}

func parseTree(raw []byte) ([]treeEntry, error) {
	records := bytes.Split(raw, []byte{0})
	entries := make([]treeEntry, 0, len(records)-1)
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		metadata, path, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, fmt.Errorf("malformed git tree entry %q", record)
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed git tree metadata %q", metadata)
		}
		entries = append(entries, treeEntry{
			Mode: fields[0],
			Type: fields[1],
			OID:  fields[2],
			Path: string(path),
		})
	}
	return entries, nil
}

func safeRelativePath(path string) (string, error) {
	if !strings.HasPrefix(path, docsPrefix) {
		return "", fmt.Errorf("path %q is outside %s", path, docsPrefix)
	}
	rel := strings.TrimPrefix(path, docsPrefix)
	if rel == "" || rel == ".." || strings.HasPrefix(rel, "../") || strings.ContainsRune(rel, '\x00') ||
		filepath.IsAbs(rel) || filepath.Clean(rel) != rel {
		return "", fmt.Errorf("unsafe documentation path %q", path)
	}
	return rel, nil
}

func fileMode(mode string) (os.FileMode, error) {
	switch mode {
	case "100644":
		return 0o644, nil
	case "100755":
		return 0o755, nil
	default:
		return 0, fmt.Errorf("unsupported git file mode %s", mode)
	}
}

func preservedSyncTime(path string, current manifest) string {
	contents, err := os.ReadFile(path)
	if err == nil {
		var previous manifest
		if json.Unmarshal(contents, &previous) == nil && sameSnapshot(previous, current) {
			if _, parseErr := time.Parse(time.RFC3339, previous.SyncedAt); parseErr == nil {
				return previous.SyncedAt
			}
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func sameSnapshot(a, b manifest) bool {
	if a.Schema != b.Schema || a.Repository != b.Repository || a.Tag != b.Tag ||
		a.TagObjectSHA != b.TagObjectSHA || a.CommitSHA != b.CommitSHA || len(a.Files) != len(b.Files) {
		return false
	}
	for i := range a.Files {
		if a.Files[i] != b.Files[i] {
			return false
		}
	}
	return true
}

func gitText(repo string, input []byte, args ...string) (string, error) {
	output, err := gitBytes(repo, input, args...)
	return strings.TrimSpace(string(output)), err
}

func gitBytes(repo string, input []byte, args ...string) ([]byte, error) {
	if repo != "" {
		args = append([]string{"-C", repo}, args...)
	}
	cmd := exec.Command("git", args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return output, nil
}
