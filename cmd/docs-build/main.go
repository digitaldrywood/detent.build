package main

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	hypeModule         = "github.com/gopherguides/hype"
	hypeCommandPackage = hypeModule + "/cmd/hype"
	sourceDirectory    = "docs/site/hype"
	outputDirectory    = "internal/docs/site"
	versionFile        = "docs/site/hype.version"
)

type document struct {
	Source string
	Output string
}

type sourcePolicy struct {
	Inputs           map[string]string
	CommandAllowlist map[string]struct{}
}

var documents = []document{
	{Source: "project-contracts.md", Output: "project-contracts.md"},
}

var policy = sourcePolicy{
	Inputs: map[string]string{
		"detent.yaml":                   "detent.yaml",
		"partials/examples/WORKFLOW.md": "docs/site/examples/WORKFLOW.md",
	},
	CommandAllowlist: map[string]struct{}{},
}

func main() {
	var hypeBinary string
	var check bool
	flag.StringVar(&hypeBinary, "hype", "", "path to the pinned Hype binary")
	flag.BoolVar(&check, "check", false, "verify committed output without changing it")
	flag.Parse()

	if hypeBinary == "" {
		fatal(errors.New("-hype is required"))
	}
	root, err := repositoryRoot()
	if err != nil {
		fatal(err)
	}
	if err := build(root, hypeBinary, check); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "site documentation build failed: %v\n", err)
	os.Exit(1)
}

func build(root, hypeBinary string, check bool) error {
	version, err := pinnedVersion(root)
	if err != nil {
		return err
	}
	if err := verifyHypeBinary(hypeBinary, version); err != nil {
		return err
	}
	files, err := validateSources(root, policy)
	if err != nil {
		return err
	}

	stage, err := os.MkdirTemp("", "detent-site-docs-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	if err := stageFiles(stage, files); err != nil {
		return err
	}

	for _, doc := range documents {
		generated, err := exportDocument(stage, hypeBinary, version, doc)
		if err != nil {
			return err
		}
		destination := filepath.Join(root, outputDirectory, filepath.FromSlash(doc.Output))
		if check {
			if err := compareOutput(destination, generated); err != nil {
				return err
			}
			continue
		}
		if err := writeOutput(destination, generated); err != nil {
			return err
		}
	}
	return nil
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		contents, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil && bytes.HasPrefix(contents, []byte("module detent.build\n")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository root not found")
		}
		dir = parent
	}
}

func pinnedVersion(root string) (string, error) {
	contents, err := os.ReadFile(filepath.Join(root, versionFile))
	if err != nil {
		return "", fmt.Errorf("read Hype version: %w", err)
	}
	version := strings.TrimSpace(string(contents))
	if version == "" || !strings.HasPrefix(version, "v") || strings.ContainsAny(version, " \t\r\n") {
		return "", fmt.Errorf("invalid Hype version %q", version)
	}
	return version, nil
}

func verifyHypeBinary(filename, version string) error {
	info, err := buildinfo.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("inspect Hype binary: %w", err)
	}
	if !isPinnedHypeBuild(*info, version) {
		return fmt.Errorf("hype binary is not %s from %s %s", hypeCommandPackage, hypeModule, version)
	}
	return nil
}

func isPinnedHypeBuild(info debug.BuildInfo, version string) bool {
	return info.Path == hypeCommandPackage && info.Main.Path == hypeModule && info.Main.Version == version && info.Main.Replace == nil
}

func validateSources(root string, policy sourcePolicy) (map[string]string, error) {
	sourceRoot := filepath.Join(root, sourceDirectory)
	sourceFiles := make(map[string]string)
	err := filepath.WalkDir(sourceRoot, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || filepath.Ext(filename) != ".md" {
			return fmt.Errorf("hype source tree contains unsupported file %q", filename)
		}
		relative, err := filepath.Rel(sourceRoot, filename)
		if err != nil {
			return err
		}
		sourceFiles[filepath.ToSlash(relative)] = filename
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk Hype sources: %w", err)
	}
	if len(sourceFiles) == 0 {
		return nil, errors.New("hype source tree is empty")
	}

	staged := make(map[string]string, len(sourceFiles)+len(policy.Inputs))
	for virtual, filename := range sourceFiles {
		staged[virtual] = filename
	}
	for virtual, filename := range policy.Inputs {
		clean, err := cleanVirtualPath("", virtual)
		if err != nil || clean != virtual {
			return nil, fmt.Errorf("invalid staged input path %q", virtual)
		}
		realPath := filepath.Join(root, filepath.FromSlash(filename))
		if err := validateRegularFile(root, realPath); err != nil {
			return nil, fmt.Errorf("validate staged input %q: %w", filename, err)
		}
		staged[virtual] = realPath
	}

	paths := make([]string, 0, len(sourceFiles))
	for virtual := range sourceFiles {
		paths = append(paths, virtual)
	}
	sort.Strings(paths)
	for _, virtual := range paths {
		contents, err := os.ReadFile(sourceFiles[virtual])
		if err != nil {
			return nil, fmt.Errorf("read Hype source %q: %w", virtual, err)
		}
		if err := validateSource(virtual, contents, sourceFiles, policy); err != nil {
			return nil, err
		}
	}
	return staged, nil
}

func validateRegularFile(root, filename string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("path resolves outside the repository")
	}
	if filepath.ToSlash(relative) == "internal/docs/vendor" || strings.HasPrefix(filepath.ToSlash(relative), "internal/docs/vendor/") {
		return errors.New("path resolves into the vendored upstream tree")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("path is not a regular file")
	}
	return nil
}

func validateSource(filename string, contents []byte, sourceFiles map[string]string, policy sourcePolicy) error {
	tokenizer := html.NewTokenizer(bytes.NewReader(contents))
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return fmt.Errorf("parse Hype source %q: %w", filename, tokenizer.Err())
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			tag := strings.ToLower(token.Data)
			attributes := make(map[string]string, len(token.Attr))
			for _, attribute := range token.Attr {
				attributes[strings.ToLower(attribute.Key)] = attribute.Val
			}
			if err := validateTag(filename, tag, attributes, sourceFiles, policy); err != nil {
				return err
			}
		}
	}
}

func validateTag(filename, tag string, attributes map[string]string, sourceFiles map[string]string, policy sourcePolicy) error {
	if tag == "cmd" || tag == "go" {
		command := commandKey(tag, attributes)
		if _, allowed := policy.CommandAllowlist[command]; !allowed {
			return fmt.Errorf("hype source %q uses command %q outside the allowlist", filename, command)
		}
	}
	src, hasSource := attributes["src"]
	if !hasSource {
		return nil
	}
	resolved, err := cleanVirtualPath(filename, src)
	if err != nil {
		return fmt.Errorf("hype source %q has unsafe %s src %q: %w", filename, tag, src, err)
	}
	switch tag {
	case "include":
		if _, exists := sourceFiles[resolved]; !exists {
			return fmt.Errorf("hype source %q includes unapproved path %q", filename, resolved)
		}
	case "code":
		if _, exists := policy.Inputs[resolved]; !exists {
			return fmt.Errorf("hype source %q reads unapproved path %q", filename, resolved)
		}
	case "cmd", "go":
		return fmt.Errorf("hype source %q uses executable src %q", filename, resolved)
	default:
		return fmt.Errorf("hype source %q uses unapproved src on <%s>", filename, tag)
	}
	return nil
}

func commandKey(tag string, attributes map[string]string) string {
	if tag == "cmd" {
		return "cmd:" + strings.Join(strings.Fields(attributes["exec"]), " ")
	}
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, key+"="+strings.Join(strings.Fields(attributes[key]), " "))
	}
	return "go:" + strings.Join(parts, ",")
}

func cleanVirtualPath(source, reference string) (string, error) {
	if reference == "" || strings.Contains(reference, "\\") || strings.HasPrefix(reference, "~") {
		return "", errors.New("path is empty or platform-dependent")
	}
	parsed, err := url.Parse(reference)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("path must be a local relative file")
	}
	if path.IsAbs(parsed.Path) {
		return "", errors.New("absolute paths are not allowed")
	}
	resolved := path.Clean(path.Join(path.Dir(source), parsed.Path))
	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", errors.New("path escapes the staged source tree")
	}
	return resolved, nil
}

func stageFiles(stage string, files map[string]string) error {
	paths := make([]string, 0, len(files))
	for virtual := range files {
		paths = append(paths, virtual)
	}
	sort.Strings(paths)
	for _, virtual := range paths {
		contents, err := os.ReadFile(files[virtual])
		if err != nil {
			return fmt.Errorf("read staged input %q: %w", virtual, err)
		}
		destination := filepath.Join(stage, filepath.FromSlash(virtual))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create staged directory: %w", err)
		}
		if err := os.WriteFile(destination, contents, 0o644); err != nil {
			return fmt.Errorf("write staged input %q: %w", virtual, err)
		}
	}
	return nil
}

func exportDocument(stage, hypeBinary, version string, doc document) ([]byte, error) {
	if _, err := os.Stat(filepath.Join(stage, filepath.FromSlash(doc.Source))); err != nil {
		return nil, fmt.Errorf("source document %q is not staged: %w", doc.Source, err)
	}
	runtimeDir := filepath.Join(stage, ".runtime")
	for _, directory := range []string{"home", "bin", "tmp"} {
		if err := os.MkdirAll(filepath.Join(runtimeDir, directory), 0o700); err != nil {
			return nil, fmt.Errorf("create sanitized runtime: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, hypeBinary, "export", "-format", "markdown", "-f", doc.Source)
	cmd.Dir = stage
	cmd.Env = sanitizedEnvironment(runtimeDir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("export %q with Hype: %s", doc.Source, message)
	}
	header := fmt.Sprintf("<!-- Generated by %s %s from %s/%s. DO NOT EDIT. -->\n\n> Provenance: generated from site-authored sources with `%s` `%s`.\n\n", hypeModule, version, sourceDirectory, doc.Source, hypeModule, version)
	generated := append([]byte(header), bytes.Trim(stdout.Bytes(), "\r\n")...)
	return append(generated, '\n'), nil
}

func sanitizedEnvironment(runtimeDir string) []string {
	blockedProxy := "http://127.0.0.1:1"
	return []string{
		"ALL_PROXY=" + blockedProxy,
		"GOPROXY=off",
		"GOSUMDB=off",
		"HOME=" + filepath.Join(runtimeDir, "home"),
		"HTTP_PROXY=" + blockedProxy,
		"HTTPS_PROXY=" + blockedProxy,
		"LANG=C",
		"LC_ALL=C",
		"NO_PROXY=",
		"PATH=" + filepath.Join(runtimeDir, "bin"),
		"TEMP=" + filepath.Join(runtimeDir, "tmp"),
		"TMP=" + filepath.Join(runtimeDir, "tmp"),
		"TMPDIR=" + filepath.Join(runtimeDir, "tmp"),
		"TZ=UTC",
		"all_proxy=" + blockedProxy,
		"http_proxy=" + blockedProxy,
		"https_proxy=" + blockedProxy,
		"no_proxy=",
	}
}

func compareOutput(filename string, generated []byte) error {
	committed, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read committed output %q: %w", filename, err)
	}
	if !bytes.Equal(committed, generated) {
		return fmt.Errorf("generated output %q is stale; run make docs-build", filename)
	}
	return nil
}

func writeOutput(filename string, generated []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".docs-build-")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := temporary.Write(generated); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write generated output: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set generated output mode: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close generated output: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("publish generated output: %w", err)
	}
	return nil
}
