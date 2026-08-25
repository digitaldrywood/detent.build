package videos

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	SchemaVersion = 1
	ChannelID     = "UCfMcoKPPyPDYDIav3-aoYOA"

	PlacementFeatured     = "featured"
	PlacementDocsAdjacent = "docs-adjacent"
	PlacementIndex        = "index"
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

//go:embed manifest.json
var manifestJSON []byte

type Manifest struct {
	SchemaVersion    int     `json:"schema_version"`
	YouTubeChannelID string  `json:"youtube_channel_id"`
	Entries          []Entry `json:"entries"`
}

type Entry struct {
	VideoID              string   `json:"video_id"`
	WatchURL             string   `json:"watch_url"`
	Title                string   `json:"title"`
	Summary              string   `json:"summary"`
	RecordedOn           string   `json:"recorded_on"`
	DurationSeconds      int      `json:"duration_seconds"`
	DemonstratedRevision string   `json:"demonstrated_revision"`
	ThumbnailPath        string   `json:"thumbnail_path"`
	Transcript           string   `json:"transcript"`
	Placement            string   `json:"placement"`
	DocsPath             string   `json:"docs_path,omitempty"`
	Claims               []string `json:"claims"`
}

type Catalog struct {
	entries []Entry
}

var Published = mustLoad(manifestJSON)

func Parse(data []byte) (*Catalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode video manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("decode video manifest: trailing content")
	}
	if err := Validate(manifest); err != nil {
		return nil, err
	}
	return &Catalog{entries: cloneEntries(manifest.Entries)}, nil
}

func Validate(manifest Manifest) error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("video manifest schema_version = %d, want %d", manifest.SchemaVersion, SchemaVersion)
	}
	if manifest.YouTubeChannelID != ChannelID {
		return fmt.Errorf("video manifest youtube_channel_id = %q, want %q", manifest.YouTubeChannelID, ChannelID)
	}

	seen := make(map[string]struct{}, len(manifest.Entries))
	featured := 0
	for index, entry := range manifest.Entries {
		if err := validateEntry(entry); err != nil {
			return fmt.Errorf("video manifest entry %d: %w", index, err)
		}
		if _, exists := seen[entry.VideoID]; exists {
			return fmt.Errorf("video manifest entry %d: duplicate video_id %q", index, entry.VideoID)
		}
		seen[entry.VideoID] = struct{}{}
		if entry.Placement == PlacementFeatured {
			featured++
		}
	}
	if featured > 1 {
		return fmt.Errorf("video manifest has %d featured entries, want at most 1", featured)
	}
	return nil
}

func validateEntry(entry Entry) error {
	for name, value := range map[string]string{
		"title":                 entry.Title,
		"summary":               entry.Summary,
		"demonstrated_revision": entry.DemonstratedRevision,
		"thumbnail_path":        entry.ThumbnailPath,
		"transcript":            entry.Transcript,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if !videoIDPattern.MatchString(entry.VideoID) {
		return fmt.Errorf("video_id %q is not an 11-character YouTube id", entry.VideoID)
	}
	if err := validateWatchURL(entry); err != nil {
		return err
	}
	if _, err := time.Parse("2006-01-02", entry.RecordedOn); err != nil {
		return fmt.Errorf("recorded_on %q must use YYYY-MM-DD", entry.RecordedOn)
	}
	if entry.DurationSeconds <= 0 {
		return fmt.Errorf("duration_seconds must be positive")
	}
	if err := validateLocalPath("thumbnail_path", entry.ThumbnailPath, "/static/images/videos/"); err != nil {
		return err
	}
	if err := validateTranscript(entry.Transcript); err != nil {
		return err
	}
	if len(entry.Claims) == 0 {
		return fmt.Errorf("claims must contain at least one sourced claim")
	}
	for index, claim := range entry.Claims {
		if strings.TrimSpace(claim) == "" {
			return fmt.Errorf("claims[%d] is empty", index)
		}
	}

	switch entry.Placement {
	case PlacementFeatured, PlacementIndex:
		if entry.DocsPath != "" {
			return fmt.Errorf("docs_path is only valid for %q placement", PlacementDocsAdjacent)
		}
	case PlacementDocsAdjacent:
		if !strings.HasPrefix(entry.DocsPath, "/docs/") || path.Clean(entry.DocsPath) != entry.DocsPath {
			return fmt.Errorf("docs_path %q must be a clean /docs/ path", entry.DocsPath)
		}
	default:
		return fmt.Errorf("placement %q is not supported", entry.Placement)
	}
	return nil
}

func validateWatchURL(entry Entry) error {
	expected := "https://www.youtube.com/watch?v=" + entry.VideoID
	if entry.WatchURL != expected {
		return fmt.Errorf("watch_url %q must be canonical https://www.youtube.com/watch?v=%s", entry.WatchURL, entry.VideoID)
	}
	return nil
}

func validateTranscript(reference string) error {
	if strings.HasPrefix(reference, "/") {
		return validateLocalPath("transcript", reference, "/static/videos/transcripts/")
	}
	parsed, err := url.Parse(reference)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("transcript %q must be a local static path or an https URL", reference)
	}
	return nil
}

func validateLocalPath(name, value, prefix string) error {
	if !strings.HasPrefix(value, prefix) || path.Clean(value) != value {
		return fmt.Errorf("%s %q must be a clean path under %s", name, value, prefix)
	}
	return nil
}

func (c *Catalog) Entries() []Entry {
	return cloneEntries(c.entries)
}

func (c *Catalog) Featured() (Entry, bool) {
	for _, entry := range c.entries {
		if entry.Placement == PlacementFeatured {
			return cloneEntry(entry), true
		}
	}
	return Entry{}, false
}

func (c *Catalog) ForDocs(publicPath string) []Entry {
	entries := make([]Entry, 0)
	for _, entry := range c.entries {
		if entry.Placement == PlacementDocsAdjacent && entry.DocsPath == publicPath {
			entries = append(entries, cloneEntry(entry))
		}
	}
	return entries
}

func (e Entry) DurationLabel() string {
	return fmt.Sprintf("%d:%02d", e.DurationSeconds/60, e.DurationSeconds%60)
}

func cloneEntries(entries []Entry) []Entry {
	cloned := make([]Entry, len(entries))
	for index, entry := range entries {
		cloned[index] = cloneEntry(entry)
	}
	return cloned
}

func cloneEntry(entry Entry) Entry {
	entry.Claims = append([]string(nil), entry.Claims...)
	return entry
}

func mustLoad(data []byte) *Catalog {
	catalog, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return catalog
}
