module detent.build

// Declared floor is 1.25 because the pinned nixpkgs archive in nixpacks.toml
// ships go_1_25 and the Nixpacks build phase has no network to fetch a newer
// toolchain. Local development on 1.26 builds this fine.
go 1.25

require (
	github.com/Oudwins/tailwind-merge-go v0.2.1
	github.com/a-h/templ v0.3.1001
	github.com/labstack/echo/v4 v4.14.0
	github.com/lmittmann/tint v1.1.2
	github.com/templui/templui v1.13.0
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/time v0.14.0 // indirect
)
