# Deploying detent.build

The site is a single stateless Go binary. It has no database, no persistent
volume, and no writable state, so a deploy is a rebuild and a container swap
with nothing to migrate.

## Dokploy application settings

| Setting | Value |
|---|---|
| Source | GitHub, `digitaldrywood/detent.build`, branch `main` |
| Build type | Nixpacks (`nixpacks.toml` in the repository root) |
| Container port | `3000` |
| Domain | `detent.build` |
| HTTPS | on, Let's Encrypt |
| Volumes | none |

Container port `3000` is not arbitrary — the Dokploy domain entry is already
bound to it. Changing `PORT` breaks the route.

## Environment

```
ENV=production
SITE_URL=https://detent.build
```

`PORT` is left unset so the binary uses its 3000 default.

`SITE_URL` feeds canonical tags and `sitemap.xml`. It must be apex, `https`,
and carry no trailing slash. Two facts make this load-bearing rather than
cosmetic:

- There is no `www.detent.build` record and there will not be one. A www URL
  anywhere on the site is a dead link.
- `.build` is on the HSTS preload list, so browsers refuse plain HTTP to it. An
  `http://detent.build` absolute URL is unreachable, not merely redirected.

`internal/handler/handler_test.go` asserts both: no rendered page may contain
`www.detent.build` or `http://detent.build`, and the sitemap must emit apex
https URLs even if `SITE_URL` picks up a stray trailing slash.

## TLS

Traefik terminates TLS in front of the container. The app serves plain HTTP on
3000 and performs no redirect of its own — an app-level http-to-https redirect
behind a TLS-terminating proxy loops. `TestNoAppLevelRedirects` guards that no
route answers with a 3xx.

## DNS

A single apex ALIAS from `detent.build` to `dokploy.digitaldrywood.com`. No www
record, no apex-to-www or www-to-apex redirect.

## The Nixpacks build

`nixpacks.toml` carries the constraints inline. The short version:

- No `providers = [...]`; every package is listed in `nixPkgs` explicitly,
  because dual Go+Node providers collide over npm's bash completions.
- `nodejs_20`, not `nodejs_22` — the latter is not in this archive.
- No separate `"npm"` package; it ships inside `nodejs_20`.
- The nixpkgs archive is pinned because the default one only carries Go 1.22.
- `go install` runs in the install phase, which has network access; the build
  phase does not.
- `templ` is invoked as `/root/go/bin/templ` because `go install` output is not
  on `PATH` during the build phase.

### The Go version, and why go.mod says 1.25

The pinned archive ships `go_1_25`. A `go 1.26` directive in `go.mod` would
make the build phase try to download a newer toolchain, which fails with no
network. `go.mod` therefore declares `go 1.25`; nothing in this repository uses
a 1.26 feature, and local development on 1.26 builds it unchanged.

If the pin is ever moved to an archive with Go 1.26, raise the directive in the
same commit.

## Verifying a deploy

```sh
make smoke                                # against https://detent.build
make smoke SMOKE_URL=http://localhost:3000
```

`scripts/smoke.sh` asserts what only a real deployment can break: every route,
the 404, the absence of www and plain-http and localhost references, the
canonical and `og:image` matching the deployed host, the sitemap, the
proxy's http-to-https redirect, and the security headers.

**Run it after every deploy.** The first production deploy passed CI, returned
200 on every route, and reported a healthy container while emitting
`http://localhost:3000` as the canonical on every page — because the Dokploy
application had no environment set, so `ENV` fell back to `development` and
`SITE_URL` to its local default. Nothing in the test suite or the container
health check could see it. This script can.

The binary also logs a warning at startup for each of `ENV` and `SITE_URL` when
they are unset, which is visible in the Dokploy log viewer.

If `/health` answers and `/` does not, the binary is up and the templ or CSS
build step produced nothing — check that the build phase ran
`templ generate` and the Tailwind CLI before `go build`.
