package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestWatcher(t *testing.T, seed string, handler http.HandlerFunc) *Watcher {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	w := New(seed)
	w.url = srv.URL
	w.client = srv.Client()
	return w
}

func TestCurrentReturnsSeedBeforeRefresh(t *testing.T) {
	w := New("v0.55.0")

	if got := w.Current(); got != "v0.55.0" {
		t.Errorf("Current() = %q, want the seed", got)
	}
}

func TestRefreshAdoptsNewerTag(t *testing.T) {
	w := newTestWatcher(t, "v0.55.0", func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte(`{"tag_name":"v0.56.1"}`))
	})

	w.refresh(context.Background())

	if got := w.Current(); got != "v0.56.1" {
		t.Errorf("Current() = %q, want v0.56.1", got)
	}
}

// A failed check must never blank the version or show something malformed —
// the page renders whatever Current returns.
func TestRefreshKeepsCurrentOnFailure(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"server error", func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusInternalServerError)
		}},
		{"rate limited", func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusForbidden)
		}},
		{"malformed json", func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(`not json`))
		}},
		{"empty tag", func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(`{"tag_name":""}`))
		}},
		{"non-semver tag", func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(`{"tag_name":"nightly"}`))
		}},
		{"injection attempt", func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(`{"tag_name":"v1.0.0<script>alert(1)</script>"}`))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newTestWatcher(t, "v0.55.0", tt.handler)

			w.refresh(context.Background())

			if got := w.Current(); got != "v0.55.0" {
				t.Errorf("Current() = %q, want the seed to survive a failed check", got)
			}
		})
	}
}

func TestStartStopsWithContext(t *testing.T) {
	w := newTestWatcher(t, "v0.55.0", func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte(`{"tag_name":"v0.56.0"}`))
	})
	w.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	deadline := time.After(2 * time.Second)
	for w.Current() != "v0.56.0" {
		select {
		case <-deadline:
			t.Fatal("watcher never picked up the new tag")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	cancel()
	time.Sleep(30 * time.Millisecond) // let the goroutine observe cancellation
}
