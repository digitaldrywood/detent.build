package handler

import (
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDeploymentCommit(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name, vcs, revision, modified, want string
	}{
		{"clean", "git", sha, "false", sha},
		{"dirty", "git", sha, "true", ""},
		{"missing modification status", "git", sha, "", ""},
		{"short hash", "git", "aaaaaaa", "false", ""},
		{"other vcs", "hg", sha, "false", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs", Value: tc.vcs}, {Key: "vcs.revision", Value: tc.revision}, {Key: "vcs.modified", Value: tc.modified},
			}}
			if got := deploymentCommit(info); got != tc.want {
				t.Fatalf("deploymentCommit = %q, want %q", got, tc.want)
			}
		})
	}
	if deploymentCommit(nil) != "" {
		t.Fatal("missing build info attested a commit")
	}
}

func TestDeployment(t *testing.T) {
	for _, commit := range []string{strings.Repeat("a", 40), ""} {
		t.Run(commit, func(t *testing.T) {
			e := echo.New()
			h := &Handler{commit: commit}
			rec := httptest.NewRecorder()
			if err := h.Deployment(e.NewContext(httptest.NewRequest(http.MethodGet, "/deployment", nil), rec)); err != nil {
				t.Fatal(err)
			}
			want := http.StatusOK
			if commit == "" {
				want = http.StatusServiceUnavailable
			}
			if rec.Code != want || rec.Header().Get(echo.HeaderCacheControl) != "no-store" {
				t.Fatalf("status %d, headers %v", rec.Code, rec.Header())
			}
			if commit != "" && !strings.Contains(rec.Body.String(), commit) {
				t.Fatal("response omitted build commit")
			}
		})
	}
}
