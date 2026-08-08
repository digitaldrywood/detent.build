You are working on {{ issue.identifier }}: {{ issue.title }}.
Current Detent status: {{ issue.state }}.

Follow repository instructions, keep changes scoped to the issue, and keep a
single persistent `## Codex Workpad` issue comment updated with the plan,
validation evidence, and final handoff. Every Workpad update must include one
`detent-status` fenced block. Detent reads blocker and human-action
declarations from that block; narrative sentences are never read as blockers.
`status` must be one of `in_progress`, `blocked`, or `complete`.

## Project Contract

This repository is the marketing site for Detent. `docs/conventions.md` is the
committed source of truth for its conventions and `docs/deploy.md` for its
production constraints. Read both before changing anything. A `CLAUDE.md` may
exist in a human's local checkout but is gitignored and will not be present
here, so these rules are restated below rather than referenced.

Stack: Go 1.25, Echo v4, templ, HTMX, Tailwind CSS v4, templUI. No database.
The binary is stateless: it renders templ pages and serves static assets.

**Content is sourced, not written.** `internal/content/content.go` holds every
factual claim on the site, and each claim must trace to the
`digitaldrywood/detent` repository — its `README.md`, `docs/comparison.md`,
`docs/concepts.md`, or the live `detent-orchestration` config. Do not add a
capability claim, a metric, a testimonial, or a logo that the Detent repository
cannot substantiate. The comparison matrix is copied from `docs/comparison.md`
including the rows where Detent loses; keep those rows.

**The design system is imported, not approximated.** The `@theme` block in
`static/css/input.css` is copied from Detent's own `static/css/input.css`. The
teal accent is for interactivity only — links, focus, selected state — and
never for status. Indigo (`--color-brand`) is the brand mark's color and
appears nowhere else. Card radius 6px, chip radius 4px, hairline borders, no
shadows, strict 4px spacing grid.

**Mono means machine.** State names, config, CLI output, and lane labels use
Geist Mono because they are literal strings, not decoration.

**One thing animates.** The 6px pulsing dot per active agent, and the held
card's ring. `prefers-reduced-motion` turns both off. Do not add more
animation.

**No signup, no pricing, no demo booking, no fake social proof.** The product
is free and self-hosted; any of those elements would misrepresent it.

**Deployment constraints, each covered by a test in
`internal/handler/handler_test.go`.** `PORT` stays 3000 because the Dokploy
domain entry is bound to container port 3000. There is no `www.detent.build`
DNS record and there will not be one — no www canonical, no www in the sitemap,
no redirect either way. `SITE_URL` is `https://detent.build` with no trailing
slash. Do not add TLS termination or an http-to-https redirect in Go; Traefik
already does both and an app-level redirect loops. `go.mod` declares `go 1.25`
because the pinned nixpkgs archive in `nixpacks.toml` ships `go_1_25` and the
build phase has no network to fetch a newer toolchain. If you change one of
these, the corresponding test should be the thing that tells you.

**templ gotchas.** A prose line that begins with `for`, `if`, or `switch` is
parsed as a control keyword, not text — rephrase or move the word off the start
of the line. Closing a `<div>` with `}` instead of `</div>` produces a
misleading "expected nodes, but none were found" error pointing at the
enclosing component. Go expressions do not interpolate inside `<script>` tags;
use data attributes or `templ.JSFuncCall`.

**templUI.** Add components with the CLI (`templui add <component>`), which
also fetches the `Script()` template that manual copies miss. Components
needing JavaScript must have their `Script()` called in
`templates/layouts/base.templ`, and `assets/js` must stay served.

## Project CI Quality Gates

Each required stage category, the local command that satisfies it, and the CI
check that enforces it. The workflow is `.github/workflows/ci.yml`.

- Code generation: local `make generate` (`templ generate`); CI check `check`,
  step "Generate templ"
- Stylesheet build: local `make css`; CI check `check`, step "Build CSS"
- Static analysis: local `go vet ./...`; CI check `check`, step "Vet"
- Unit tests with race detector: local `go test -race ./...`; CI check `check`,
  step "Test"
- Server build with CGO disabled: local
  `CGO_ENABLED=0 go build -o /dev/null ./cmd/server`; CI check `check`, step
  "Build"
- Lint: local `golangci-lint run` and `templ fmt templates/ ui/` (both via
  `make lint`); CI check `lint`, golangci-lint-action

`make check` runs generate, css, vet, race tests, and golangci-lint. It does
not run the CGO-disabled server build or `templ fmt`, so the configured
validation gate appends the build explicitly. Run `make lint` when you touch
templ files.

Treat this list as part of the project contract. Whenever you touch CI
configuration or perform a review, verify that every declared stage exists,
runs its mapped project tool, and passes on the current pull request head. Do
not rely on Detent or `detent doctor` to infer required stages or inspect CI
configuration.

Use `in_progress` while implementation or validation is still active:

```detent-status
schema: 1
status: in_progress
blockers: []
human_action: null
```

This project runs in autopilot. Detent watches the pull request gate and
promotes completed work to `Merging` itself, so agents never move an issue to
`Human Review`. A stuck agent belongs in `Blocked` with a concrete blocker in
the `detent-status` block. `Backlog`, `Human Review`, `Done`, and `Cancelled`
are never worked.

Use `complete` only when the pull request is open, marked ready for review,
is not a draft, references the issue, validation is green, and no actionable
review comments remain. If the pull request is a draft, mark it ready and
verify the resulting state before declaring completion:

```sh
gh pr ready <number>
gh pr view <number> --json isDraft --jq '.isDraft' # must be false
```

```detent-status
schema: 1
status: complete
blockers: []
human_action: null
```

For dependency blockers, use this order:

1. Create GitHub's native `blocked_by` dependency relation.

```sh
BLOCKED_NUMBER=<blocked-issue-number>
BLOCKER_NUMBER=<blocker-issue-number>
BLOCKER_ID="$(gh api repos/{owner}/{repo}/issues/$BLOCKER_NUMBER --jq '.id')"
gh api --method POST "repos/{owner}/{repo}/issues/$BLOCKED_NUMBER/dependencies/blocked_by" -F issue_id="$BLOCKER_ID"
```

2. Declare the blocker in the Workpad status block with `status: blocked`.

```detent-status
schema: 1
status: blocked
blockers:
  - ref: "owner/repo#123"
    reason: "waiting for the dependency to merge"
human_action: null
```

3. Legacy fallback during the deprecation window: if native dependencies are
   unavailable and the project has not migrated, keep a machine-readable
   issue-body line such as `Blocked by: #123` or `Depends on: owner/repo#123`.

If meaningful out-of-scope work is discovered, file a separate tracker issue in Backlog with a best-guess `detent-agent` effort block instead of expanding the current work item.

## Required Execution Flow

Use the current Detent state as the source of truth for which section applies.

Before any rebase, capture the branch's effective diff against its merge base
or preserve the pre-rebase ref. After the rebase, compare with `git range-diff`
or an equivalent diff-stat and confirm the same files and hunks remain. If
changes are missing without explanation or conflict resolution dropped hunks,
stop before pushing and move the issue to the configured blocked or exception
state.

### For Todo

1. Move the issue to `In Progress`.
2. Create or update the persistent `## Codex Workpad` comment with the plan,
   acceptance criteria, validation plan, and the `in_progress`
   `detent-status` block shown above.
3. Fetch current `origin/main`, confirm this worktree is based on it, and
   confirm every native dependency relation, `detent-status` blocker, and
   issue-body `Depends on:` reference is merged or otherwise terminal before
   coding.
4. Reproduce or confirm the reported behavior before changing code when the
   issue is a bug.
5. Implement the smallest complete change that satisfies the issue.
6. Run focused tests for touched packages, then run the configured validation
   gate.
7. Commit and push the branch.
8. Open or update a pull request that references the issue.
9. Re-check pull request comments, inline review comments, and CI after the
   latest push.
10. Leave the issue in `In Progress`. Set the Workpad block to
    `status: complete` with `blockers: []` and `human_action: null` only after
    the pull request is open, not a draft, references the issue, the gate and
    current-head CI are green, and no actionable review comments remain.
    Detent auto-promotes directly to `Merging`; never use `Human Review`.

### For In Progress

1. Re-read the issue, pull request, comments, and `## Codex Workpad`, including
   the `detent-status` block.
2. Continue from the current repository and tracker state.
3. If implementation is complete, run the full pre-review gate, then update the
   Workpad block to `status: complete` with `blockers: []` and
   `human_action: null` only when the gate passes. Leave the issue in
   `In Progress` and let Detent promote it to `Merging`.

### For Rework

1. Re-read all human and bot feedback.
2. Move the issue to `In Progress`.
3. Fix the requested changes.
4. Push updates to the pull request.
5. Run the full pre-review gate again.
6. Set the Workpad block back to `status: complete` only when the gate passes,
   and leave the issue in `In Progress` for Detent to promote to `Merging`.

### For Merging

1. Confirm `$go-workflow:ship` is available in the Codex environment. If it is
   unavailable, keep the issue in `Merging` and record the missing ship workflow
   as `human_action` in the `detent-status` block.
2. Invoke and follow `$go-workflow:ship`.
3. Do not call `gh pr merge` directly outside the ship workflow.
4. End with exactly one terminal outcome:
   - pull request merged and issue moved to `Done`;
   - issue moved to `Rework` with an actionable defect;
   - issue remains in `Merging` with a concrete external blocker recorded in
     the `detent-status` block and described in the `## Codex Workpad`.
5. Move the issue to `Done` only after the pull request is merged.
