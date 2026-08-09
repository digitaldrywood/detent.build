package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestRepositorySiteSourcesRespectPolicy(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSources(root, policy); err != nil {
		t.Fatalf("validate site sources: %v", err)
	}
}

func TestVerifyHypeBuildInfoRequiresPinnedCLI(t *testing.T) {
	tests := []struct {
		name string
		info debug.BuildInfo
		want bool
	}{
		{
			name: "pinned CLI",
			info: debug.BuildInfo{
				Path: hypeCommandPackage,
				Main: debug.Module{Path: hypeModule, Version: "v0.8.0"},
			},
			want: true,
		},
		{
			name: "wrapper dependency",
			info: debug.BuildInfo{
				Path: "example.com/wrapper",
				Main: debug.Module{Path: "example.com/wrapper", Version: "v1.0.0"},
				Deps: []*debug.Module{{Path: hypeModule, Version: "v0.8.0"}},
			},
		},
		{
			name: "wrong version",
			info: debug.BuildInfo{
				Path: hypeCommandPackage,
				Main: debug.Module{Path: hypeModule, Version: "v0.7.0"},
			},
		},
		{
			name: "replaced module",
			info: debug.BuildInfo{
				Path: hypeCommandPackage,
				Main: debug.Module{
					Path:    hypeModule,
					Version: "v0.8.0",
					Replace: &debug.Module{Path: "example.com/fork", Version: "v0.8.0"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isPinnedHypeBuild(test.info, "v0.8.0"); got != test.want {
				t.Errorf("isPinnedHypeBuild() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestValidateSourcesRejectsVendoredReads(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{name: "include", source: `<include src="internal/docs/vendor/config.md"></include>`},
		{name: "code", source: `<code src="internal/docs/vendor/config.md"></code>`},
		{name: "other source", source: `<img src="internal/docs/vendor/brand/detent-mark.svg">`},
		{name: "traversal", source: `<code src="../../../internal/docs/vendor/config.md"></code>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := testRepository(t, test.source)
			if _, err := validateSources(root, sourcePolicy{Inputs: map[string]string{}, CommandAllowlist: map[string]struct{}{}}); err == nil {
				t.Fatal("validateSources accepted a vendored read")
			}
		})
	}
}

func TestValidateSourcesRejectsCommandsOutsideAllowlist(t *testing.T) {
	for _, source := range []string{
		`<cmd exec="git status"></cmd>`,
		`<go run="./cmd/example"></go>`,
	} {
		root := testRepository(t, source)
		if _, err := validateSources(root, sourcePolicy{Inputs: map[string]string{}, CommandAllowlist: map[string]struct{}{}}); err == nil {
			t.Fatalf("validateSources accepted %s", source)
		}
	}
}

func TestValidateSourcesAcceptsMappedFilesAndIncludes(t *testing.T) {
	root := testRepository(t, strings.Join([]string{
		`<code src="detent.yaml"></code>`,
		`<include src="partials/workflow.md"></include>`,
	}, "\n"))
	writeTestFile(t, filepath.Join(root, sourceDirectory, "partials", "workflow.md"), `<code src="examples/WORKFLOW.md"></code>`)
	writeTestFile(t, filepath.Join(root, "detent.yaml"), "schema: 1\n")
	writeTestFile(t, filepath.Join(root, "docs", "site", "examples", "WORKFLOW.md"), "example\n")
	testPolicy := sourcePolicy{
		Inputs: map[string]string{
			"detent.yaml":                   "detent.yaml",
			"partials/examples/WORKFLOW.md": "docs/site/examples/WORKFLOW.md",
		},
		CommandAllowlist: map[string]struct{}{},
	}
	files, err := validateSources(root, testPolicy)
	if err != nil {
		t.Fatalf("validateSources: %v", err)
	}
	for _, want := range []string{"project-contracts.md", "partials/workflow.md", "detent.yaml", "partials/examples/WORKFLOW.md"} {
		if _, exists := files[want]; !exists {
			t.Errorf("staged files are missing %q", want)
		}
	}
}

func TestSanitizedEnvironmentOmitsCredentialsAndBlocksStandardNetworkPaths(t *testing.T) {
	environment := sanitizedEnvironment(t.TempDir())
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			t.Fatalf("invalid environment entry %q", entry)
		}
		values[key] = value
	}
	for _, forbidden := range []string{"GH_TOKEN", "GITHUB_TOKEN", "DETENT_API_TOKEN", "SSH_AUTH_SOCK", "AWS_ACCESS_KEY_ID", "GOOGLE_APPLICATION_CREDENTIALS"} {
		if _, exists := values[forbidden]; exists {
			t.Errorf("sanitized environment contains %s", forbidden)
		}
	}
	for _, proxy := range []string{"ALL_PROXY", "HTTP_PROXY", "HTTPS_PROXY", "all_proxy", "http_proxy", "https_proxy"} {
		if values[proxy] != "http://127.0.0.1:1" {
			t.Errorf("%s = %q", proxy, values[proxy])
		}
	}
	if values["GOPROXY"] != "off" || values["GOSUMDB"] != "off" {
		t.Error("Go network access is not disabled")
	}
}

func TestCleanVirtualPathRejectsRemoteAbsoluteAndEscapingPaths(t *testing.T) {
	for _, reference := range []string{
		"https://example.com/file.md",
		"/etc/passwd",
		"../../../internal/docs/vendor/config.md",
		"..\\..\\internal\\docs\\vendor\\config.md",
	} {
		if resolved, err := cleanVirtualPath("guide.md", reference); err == nil {
			t.Errorf("cleanVirtualPath(%q) = %q", reference, resolved)
		}
	}
}

func TestHypeIsBuildTimeOnly(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{"go.mod", "nixpacks.toml"} {
		contents, err := os.ReadFile(filepath.Join(root, filename))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(contents), hypeModule) || strings.Contains(string(contents), "docs-build") {
			t.Errorf("%s includes the Hype build pipeline", filename)
		}
	}
}

func testRepository(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, sourceDirectory, "project-contracts.md"), source)
	return root
}

func writeTestFile(t *testing.T, filename, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
