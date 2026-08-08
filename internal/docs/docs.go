// Package docs exposes the pinned Detent documentation as an embedded filesystem.
package docs

import (
	"embed"
	"io/fs"
)

const (
	SourceRepository = "https://github.com/digitaldrywood/detent"
	ReleaseTag       = "v0.57.0"
	TagObjectSHA     = "10c9b2a531089e8bac7a3fcd42593b257863ec8d"
	CommitSHA        = "1543929187369eca2703abd2a655cf86e9e5d83e"
)

//go:embed vendor manifest.json
var embedded embed.FS

// Files contains the byte-identical documentation tree rooted at the pinned release's docs directory.
var Files = mustSub(embedded, "vendor")

func mustSub(source fs.FS, path string) fs.FS {
	result, err := fs.Sub(source, path)
	if err != nil {
		panic(err)
	}
	return result
}
