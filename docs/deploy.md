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
- `nodejs_22`, matching the Node version CI tests with.
- No separate `"npm"` package; it ships inside `nodejs_22`.
- The nixpkgs archive is pinned because the default one lags the Go release `go.mod` declares.
- `go install` runs in the install phase, which has network access; the build
  phase does not.
- `templ` is invoked as `/root/go/bin/templ` because `go install` output is not
  on `PATH` during the build phase.

### The Go version, and why go.mod says 1.27

The pinned archive (nixos-26.05) ships `go_1_27`. A directive newer than the
archive's Go would make the build phase try to download a newer toolchain,
which fails with no network. `go.mod` therefore declares `go 1.27`, and CI sets
up the same Go version.

Move the archive pin, the `go_1_X` package, the `go.mod` directive, and CI's
`go-version` in the same commit.

## Verifying a deploy

### Hourly develop releases

`.github/workflows/ci.yml` runs at minute 17 of each UTC hour and keeps manual
dispatch. GitHub schedules use the default branch; keep `develop` as the default.
Manual dispatches from another branch do nothing. Pushes and pull requests do
not trigger this workflow, and its jobs must not become landing requirements.
The runner's pre-landing gate remains `make check && CGO_ENABLED=0 go build -o
/dev/null ./cmd/server`.

The CSS targets install the locked frontend dependencies with `npm ci
--include=dev` when the local installation is missing or older than either npm
manifest, then invoke the local Tailwind CLI. A fresh landing checkout therefore
needs npm and registry access; it does not need a separate `npm ci` before the
gate. Existing installations are reused on subsequent builds.

Each run pins one develop commit for every job. `check` runs the complete local
gate, template formatting, CI helper tests, and `make vulncheck` (govulncheck
against reachable code, on the latest Go 1.27 patch release). `browser` exercises the built
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
verifies production responses. The subsequent release security audit separately
attests Dokploy's deployed commit before reading any audit source.

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
delivery fails. If logs are unavailable, the issue records that gap and any
available step results; missing logs and intake failures make the reporter red.
Full job output and browser traces remain in Actions. Promotion requires all three secrets; missing
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

### Release security audit

The scheduled/manual `CI` workflow runs `security-audit` after promotion and
production smoke, including on runs where main already contains the tested
develop commit. It never runs on pushes or pull requests and adds no landing
requirement. It neither triggers Dokploy nor waits for repair issues.

`GET https://detent.build/deployment` returns the running binary's full Git
commit from Go build metadata, with `Cache-Control: no-store`. Missing metadata
or a modified source tree returns 503; `/health` remains the availability check.
Nixpacks includes Git and explicitly enables VCS stamping. Preserve the clean
Git checkout and `.git` metadata in the Dokploy build context. Generated templ
and CSS files are ignored and do not dirty the source tree. Never replace this
attestation with an environment variable or the product release shown in the
navigation.

The job polls until the attested SHA equals the promotion commit, or the pinned
main head when promotion was unnecessary. It verifies membership in main and
passes that exact SHA as `identity.commit` and `CI_DEVELOP_SHA` to the controller.
The main promotion can be a merge commit with the tested develop tree; these
commit identities are deliberately retained separately. A stale deployment,
missing metadata, or failed lookup fails as audit infrastructure.

The controller is `digitaldrywood/detent` at reviewed commit
`30d8a2c26b9539b59013b8428f7dd033529b50af` (Detent #1481), built as
`./tools/cifailure`, with `DETENT_RELEASE_AUDIT=true`. Its embedded Cloudflare
bundle and validators are verified by the controller. The checkout used to
build the controller is separate from the Git tree being audited. Updating the
pin requires reviewing that interface and updating the workflow and helper
together. The provider CLI is pinned to `@openai/codex@0.160.0`.

Before enabling the audit, the existing release operator configures:

- `DETENT_MCP_URL` and `DETENT_API_KEY` Actions secrets: the reviewed
  organization MCP connection and an operator credential scoped to native
  project `prj_e7aab744eb84410fafd437f478992bd1`, authorized for audit checkpoints,
  attachments, and the existing native reporter. Worker credentials cannot
  publish these records. These differ from the CI intake adapter credentials.
- `DETENT_RELEASE_AUDIT_OPENAI_API_KEY`: the provider credential. A wrapper
  authenticates each controller-created provider profile using stdin; profiles
  remain in job scratch and are never uploaded as artifacts.
- `DETENT_RELEASE_AUDIT_MODEL`: an Actions variable copied from the project's
  effective selected audit model. There is no workflow model default.
- `DETENT_RELEASE_SERIES`: the native reporter's selected stabilization series,
  in `release/vN.N` form. Keep it stable across deployments in that series to
  preserve existing issue reuse and human holds.

Each invocation allows 40 provider calls, 200000 input tokens, 50000 output
tokens, and 150000 cached input tokens. The controller hydrates prior evidence
from project-scoped Hub checkpoints; this site has no previous reviewer cursor
to migrate or advance. Target execution requires the controller's Linux
Bubblewrap, prlimit and delegated cgroup v2 controls. Missing controls leave
execution-dependent records `needs_validation`; there is no unrestricted
fallback. An ordinary hosted runner may lack those controls.

Confirmed records go through the controller's existing `cifailure` reporter,
which reuses fingerprinted native stabilization work while preserving state,
holds and stronger priority. `needs_validation` and explicitly retained
`incomplete` results do not fail this workflow and never claim clean coverage.
Execution, final publication and reporting failures remain red infrastructure
failures. The existing CI failure reporter includes this job and labels its
diagnostic as infrastructure rather than inferring a vulnerability.

The `release-security-audit` Actions artifact retains the deployment identity,
request, final result and controller stdout/stderr for 30 days; provider profiles
are excluded. Full checkpoints and bounded evidence parts remain with the Hub
reporting owner. The release operator verifies the first scheduled audit after
integration, including the attested SHA against Dokploy's actual deployment,
scoped checkpoint publication, confirmed-finding issue reuse, and infrastructure
delivery. Local fixtures establish source behavior, not a live provider run or
credential availability.

Run `node scripts/ci/test-release-audit-integration.mjs` for the controller
integration check, also invoked by the scheduled workflow's `check` job. It
fetches the exact controller pin into job scratch, adds only a test file, and
uses the site's request builder and result handler around the real stage,
bundle validators, REST transport and native reporter. Controlled provider,
sandbox, checkpoint-storage and HTTPS native-service fixtures establish
publish-before-repair ordering, initial Todo/High creation, fingerprint reuse,
occurrence replay deduplication, and preservation of held state and stronger
priority. Nonconfirmed results create no repairs; provider, checkpoint and
reporting failures remain infrastructure failures. The check needs Git, Node,
Go and access to the pinned source/toolchain dependencies; it uses no live
provider credentials and writes no real tracker issues.

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
