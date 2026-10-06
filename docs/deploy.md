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

### Hourly develop releases

`.github/workflows/ci.yml` runs at minute 17 of each UTC hour and keeps manual
dispatch. GitHub schedules use the default branch; keep `develop` as the default.
Manual dispatches from another branch do nothing. Pushes and pull requests do
not trigger this workflow, and its jobs must not become landing requirements.
The runner's pre-landing gate remains `make check && CGO_ENABLED=0 go build -o
/dev/null ./cmd/server`.

Each run pins one develop commit for every job. `check` runs the complete local
gate, template formatting, and CI helper tests. `browser` exercises the built
binary, every published docs page, CSS, and HTMX install navigation in Chromium,
and runs the HTTP smoke locally. A failure or skipped prerequisite prevents
promotion.

On green, `promote` fetches main again and skips if main already contains the
tested commit. Otherwise it creates a merge commit whose tree must exactly
match the tested develop tree and pushes normally. Newer develop commits wait
for another run; main-only changes, conflicts, concurrent incompatible updates,
and branch protection rejections fail visibly. There is no force push or
protection bypass. Dokploy continues deploying from main. After a 90-second
build allowance, `smoke` retries the production checks up to 15 times. This
verifies production responses; it does not attest Dokploy's deployed commit.

Before rollout, the repository/release operator must configure these Actions
secrets:

- `DETENT_PROMOTION_TOKEN`: a release identity with repository contents write
  access that main's existing protections allow. Use a dedicated app token or
  fine-grained token; do not relax protections. This also supplies a normal push
  event for Dokploy. The workflow's default token stays read-only.
- `DETENT_CI_INTAKE_URL` and `DETENT_CI_INTAKE_TOKEN`: an HTTPS receiver connected
  to this **native Detent Cloud project**. The worker cannot provision or verify
  this receiver. The checked-in OSS `/api/v1/intake` documentation only supports
  GitHub trackers; it is not evidence of a Cloud API. Do not substitute GitHub
  issues or point the receiver at a guessed Cloud route.

The receiver integration contract is explicit in `scripts/ci/report.mjs`:
authenticated POST JSON with `project_id`, `summary`, `details`, a stable
job-specific `fingerprint`, `delivery_id`, `state: Todo`, and `priority: 2`.
It must create one native Todo/High issue per failed job or **append a comment**
to the open issue with that fingerprint, preserving its current state. Repeated
delivery ids must not duplicate comments. Return JSON containing a canonical
`work_item_id` and `action` (`created`, `commented`, or `duplicate`), only after
durable success. This is a required adapter contract, not a claimed built-in
Cloud endpoint. Receiver availability and conformance must be verified before
enabling promotion.

The reporter reads the current run attempt's failed jobs and actual logs, and
sends the run/job URLs, tested SHA, and the last 16000 characters of output.
Skipped jobs do not create issues. Every failure is attempted even if another
delivery fails; intake failures make the reporter red. Full job output and
browser traces remain in Actions. Promotion requires all three secrets; missing
configuration cannot silently release.

### Live acceptance (release operator)

After integration and receiver setup, leave `CI_FAILURE_JOB` unset and record
an actual **scheduled** green run with a new develop SHA. Confirm its promotion
commit on main, Dokploy deployment, and successful `smoke` job. Record the run,
tested SHA, promotion, and smoke links in the native issue through the release
owner; a manual run or local fixture is not scheduled-run evidence.

Set the repository Actions variable `CI_FAILURE_JOB=check` for the next
scheduled run. The check job fails deliberately after validation with a clear
log line. Confirm no main promotion, and a Todo/High native issue containing
that run URL, SHA, and failing output. A second scheduled failure must comment
on the same open issue, without duplicating it or resetting its state. Remove
the variable immediately after verification, and record those links in the
native acceptance issue. Secrets, receiver setup, and both live runs are
post-integration work; do not claim them from a source worker's local tests.

### HTTP smoke

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
