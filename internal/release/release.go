// Package release keeps the displayed Detent version current without a deploy.
//
// The version appears in the nav chip, the hero footnote, and the open-source
// page. Hand-maintaining a constant means the site quietly misreports the
// current release between deploys, which is the same class of defect as any
// other unsourced claim. This polls the GitHub releases API instead and falls
// back to the compiled-in value whenever the fetch fails.
package release

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"
)

const (
	// LatestReleaseURL is the public, unauthenticated releases endpoint.
	// Polling it a few times a day is far inside the 60 requests/hour
	// anonymous limit.
	LatestReleaseURL = "https://api.github.com/repos/digitaldrywood/detent/releases/latest"

	defaultInterval = 6 * time.Hour
	fetchTimeout    = 10 * time.Second
)

// semver guards against rendering whatever a compromised or confused API
// returns. Only a plain vN.N.N tag is accepted.
var semver = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// Watcher holds the current version and refreshes it in the background.
// The zero value is not usable; call New.
type Watcher struct {
	current  atomic.Value // string
	url      string
	interval time.Duration
	client   *http.Client
}

// New returns a Watcher seeded with the compiled-in version. Until Start runs
// and succeeds, Current returns that seed.
func New(seed string) *Watcher {
	w := &Watcher{
		url:      LatestReleaseURL,
		interval: defaultInterval,
		client:   &http.Client{Timeout: fetchTimeout},
	}
	w.current.Store(seed)
	return w
}

// Current is safe to call from any goroutine.
func (w *Watcher) Current() string {
	v, _ := w.current.Load().(string)
	return v
}

// Start fetches once, then refreshes on an interval until ctx is done. It
// returns immediately; failures are logged and leave the previous value in
// place, so a GitHub outage degrades to a slightly stale version rather than
// an empty one.
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		w.refresh(ctx)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.refresh(ctx)
			}
		}
	}()
}

func (w *Watcher) refresh(ctx context.Context) {
	tag, err := w.fetch(ctx)
	if err != nil {
		slog.Warn("release check failed, keeping current version",
			"error", err, "version", w.Current())
		return
	}
	if tag == w.Current() {
		return
	}
	slog.Info("release version updated", "from", w.Current(), "to", tag)
	w.current.Store(tag)
}

func (w *Watcher) fetch(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "detent.build-site")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get latest release: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Debug("close release response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest release returned %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if !semver.MatchString(payload.TagName) {
		return "", fmt.Errorf("unexpected tag %q", payload.TagName)
	}
	return payload.TagName, nil
}
