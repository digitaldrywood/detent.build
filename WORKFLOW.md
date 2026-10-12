You are working on {{ issue.identifier }}: {{ issue.title }}.
Current Detent status: {{ issue.state }}.

This is a Detent Cloud native project. The issue lives on the Detent Hub, not
on GitHub, and Detent itself turns your work into a native Change Request.

## Completion Contract

- Do the work in this workspace on the branch Detent checked out for you.
- Commit your finished change on that branch with a conventional commit
  message. Detent records the Change Request from your commits when the run
  finishes; uncommitted work is lost.
- Never push, never open or update pull requests, never post or edit issue
  comments (including any Workpad comment), and never change issue state or
  labels. Never run `gh` or call the GitHub API for any reason; this project
  works without GitHub issues or pull requests. Detent owns all of that.
- Do not rebase onto or merge other branches unless the issue asks for it.
- If you cannot finish, do not commit a half-done change; explain the blocker
  plainly in your final message.
- End with a short summary: what changed, how you validated it, anything a
  reviewer should check.

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

## Validation

Detent runs `make check && CGO_ENABLED=0 go build -o /dev/null ./cmd/server`
on your commit after the run. Run it yourself before committing, and run
`make lint` when you touch templ files. `make check` runs generate, css, vet,
race tests, and golangci-lint. Fix failures instead of committing around them.
Do not post commit statuses.

## Browser Verification In Workers

This project is a website, so nearly every change is UI-visible and needs a
browser check. Use the browser tooling this worker actually has.

- Use the **`chrome-devtools` MCP server** (`mcp__chrome-devtools__*`) for all
  browser verification. It is configured for Codex workers, is worktree-aware,
  and gives each worktree an isolated Chrome profile.
- Do **not** route browser verification through the `claude-in-chrome` skill or
  `mcp__claude-in-chrome__*` tools. That path needs the Claude Chrome extension,
  which is not available in a Detent worker, and the skill forbids substituting
  chrome-devtools — so taking that path dead-ends the run even
  though working browser tooling was available the whole time.
- Missing browser tooling is only a real blocker if
  `mcp__chrome-devtools__navigate_page` is genuinely absent from the tool list.
  Check before declaring the gate unsatisfiable.
- Screenshots: prefer viewport over full-page, and set `deviceScaleFactor: 1`
  before any full-page capture.

## Required Execution Flow

1. Re-read the issue and any earlier review feedback in it. For rework, fix
   exactly what the reviewer asked for on top of the existing branch.
2. Reproduce or confirm the reported behavior before changing code when the
   issue is a bug.
3. Implement the smallest complete change that satisfies the issue.
4. Run focused tests for touched packages, then the full validation command
   above, and verify UI changes in the browser.
5. Commit on the current branch and stop. Detent opens the Change Request and
   moves the issue to review.
