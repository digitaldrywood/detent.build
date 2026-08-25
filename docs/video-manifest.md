# Video manifest contract

The committed video manifest is `internal/videos/manifest.json`. Video Studio
proposes changes to that file and commits the corresponding reviewed thumbnail
and transcript assets. The site reads the manifest at build time; it never
calls YouTube at runtime.

The top-level `schema_version` is `1`, `youtube_channel_id` must be
`UCfMcoKPPyPDYDIav3-aoYOA`, and `entries` is an array of objects with these
fields:

| Field | Contract |
|---|---|
| `video_id` | Required, unique 11-character YouTube video id. |
| `watch_url` | Required canonical `https://www.youtube.com/watch?v=<video_id>` URL. |
| `title` | Required reviewed title. |
| `summary` | Required sourced site summary. |
| `recorded_on` | Required recording date in `YYYY-MM-DD` form. |
| `duration_seconds` | Required positive whole-second duration. |
| `demonstrated_revision` | Required Detent release or commit demonstrated by the recording. |
| `thumbnail_path` | Required clean path under `/static/images/videos/`. The optimized image must be committed in the same proposal. |
| `transcript` | Required clean path under `/static/videos/transcripts/`, or an `https` transcript URL. |
| `placement` | Required: `featured`, `docs-adjacent`, or `index`. Every entry appears on `/videos`; the key selects its additional surface. |
| `docs_path` | Required only for `docs-adjacent`; a clean canonical path below `/docs/`. |
| `claims` | Required non-empty list of Detent behavior the recording visibly demonstrates. |

At most one entry may use `featured`. An absent declaration is not an entry:
nothing may infer Detent membership from a YouTube title, tag, playlist, or
channel upload. Titles, summaries, demonstrated revisions, and claims require
the same source review as copy in `internal/content`.

The Go validator in `internal/videos/manifest.go` rejects unknown fields,
unsupported schema versions or placements, non-canonical watch URLs, malformed
dates, duplicate ids, unsafe local paths, empty claims, and multiple featured
entries. Repository tests also require every local thumbnail and transcript to
exist.
