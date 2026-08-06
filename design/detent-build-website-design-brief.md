# Detent — website design brief

**Domain: detent.build**

Paste this whole document into Claude Design as the source material for concepts. Everything below is drawn from the actual repository (README.md, docs/comparison.md, docs/concepts.md, static/css/input.css, docs/brand/detent-mark.svg), not invented.

---

## 1. What the product is, in one paragraph

Detent is status-driven agentic work orchestration shipped as a single Go binary. You mark a GitHub issue `Todo`; Detent claims it, creates an isolated Git worktree, dispatches a Codex coding agent against a workflow contract you wrote, runs your validation gate, opens a pull request, waits for the review gate you specified, and merges through a serialized merge train — all of it live on a web dashboard and a terminal UI. It runs many issues, and many repositories, in parallel from one host. It is self-hosted, MIT-licensed, free, with no vendor control plane.

The name is the argument: a detent is the catch that holds a moving part at a fixed position until it is deliberately released — the click-stop on a dial, the notch on a ratchet. Detent holds each piece of work at a defined stop on your board and only lets it advance when a gate is cleared.

## 2. The core idea the site has to land

The category around Detent is full of autonomy-first assistants: you talk to an agent, it keeps memory and sessions, picks its own tools, and you steer it and course-correct when it drifts. Detent inverts the interaction model. You write the issue first — scope, acceptance criteria, tests, dependency order, review policy, merge rule — and the board state decides when it is eligible. The runtime executes your contract in an isolated worktree and will not land the work until the gates you encoded pass. You are not steering an agent; you are running your own engineering process at scale.

The product's own framing, which the site should carry: **it is a system, not an agent.** The intelligence stays in your spec; the runtime supplies the discipline. The goal is not to replace engineers or hide work behind opaque behavior — it is to scale the judgment of engineers who already have a high bar. The system does not try to be smarter than you; it tries to be as disciplined as you would be, every time, in parallel.

A concrete contrast worth putting on the page: "add OAuth token rotation" in an autonomy-first tool starts as a prompt and becomes a supervision loop — review the plan, inspect partial edits, redirect when it misses migrations or tests. In Detent it starts as an issue that names the storage change, CLI behavior, migration, rollback, and tests; the worker produces a reviewable PR and does not merge until the gates are green.

## 3. How it actually works (the mechanism, for a diagram or animated section)

1. **You write the contracts.** Each project has a checked-in `detent.yaml` machine contract (tracker bindings, states, lifecycle policy, scheduling, retries, leases, gates) plus a checked-in, portable `WORKFLOW.md` agent instruction contract. Optional gitignored `detent.local.yaml` and `WORKFLOW.local.md` apply machine-specific overrides without touching the shared contracts.
2. **You mark an issue `Todo`.** Detent claims it, creates an isolated Git worktree from your source checkout, dispatches a Codex agent with the contract, and moves the issue to `In Progress`.
3. **The agent works** in its own branch, runs your validation gate, and opens or updates a PR.
4. **Gates decide.** The workflow decides whether promotion to `Merging` waits in `Human Review`, waits in the active lane, requires a current-head automated PR review, or only needs linked PR + green CI + quiet time. Unresolved feedback sends it to `Rework` for another pass.
5. **The merge train is serialized** — one rebase, CI-watch, and merge at a time, so concurrent candidates never invalidate each other's CI — then the issue is `Done`.
6. **One host, many repos.** A `global.yaml` runs multiple projects with weights, priority, pause, and fair scheduling. The dashboard and TUI show live counts, running agents, token / budget / rate-limit state, and board flow.

The status lanes are the spine of the whole product and should be visible in the design: **Todo → In Progress → Human Review → Rework → Merging → Done.** That row of lanes is the most recognizable visual asset Detent has. Use it.

Board flexibility worth one line, not a paragraph: the source of truth can be a GitHub ProjectV2 board, or Detent can run boardless from an issue `Status` field or repository status labels while supplying its own Kanban view.

## 4. Target users

### Primary — the staff/principal engineer or engineering lead who runs delivery

They already have a high bar and a written process: CI gates, review standards, merge rules, dependency ordering. Their bottleneck is not idea generation, it is throughput on well-specified work while keeping the bar intact. They are suspicious of agents that merge things they did not read. They will run this on their own hardware. They read a `detent.yaml` before they read a testimonial.

What they need from the site: the state machine, the gate model, the merge-train guarantee, the isolation story (worktree per issue), and evidence the thing is real. They want to see config, not adjectives.

### Primary — the solo builder or small-team founder-engineer running several repos

One person, five repositories, a queue of work that is clear but unwritten. They want parallelism without babysitting five chat sessions. Cost matters: Detent is free and MIT, and it bills agent work against a ChatGPT plan they already pay for rather than a per-seat SaaS.

What they need: fast time-to-first-run, the single-binary install, multi-project from one host, and budget/rate-limit visibility.

### Secondary — the platform / developer-productivity team

Evaluating agentic delivery for an org. Their questions are governance questions: self-hosted, air-gappable, no vendor control plane, budget caps, multi-instance fleet governance, audit trail in GitHub where reviews already live, open source under MIT.

What they need: a comparison surface, the security/isolation posture, and the "one ID space" argument — a GitHub Project *is* the board, its issues are the work items, its pull requests are the deliverables, its comments and reviews are where every conversation happens. No mapping Linear IDs onto PR numbers, no reading discussion in two places.

### Secondary — the technical product manager or delivery lead

Not writing the config, but living on the board. They care that the board stays honest, that work has visible stops, and that "in progress" means an agent is actually working. The dashboard — charts, trends, timelines, hover detail, live counts — is their surface.

What they need: the dashboard shown large and legible, and the vocabulary of lanes and gates explained without config syntax.

### Explicit anti-persona

People who want an autonomous assistant to talk to. People who want a hosted SaaS with a signup button. People looking for a coding copilot inside an editor. The site should be honest enough that these visitors self-select out on the home page rather than bouncing off the docs.

## 5. Proof points to design around

- One CGO-free Go binary for macOS, Linux, and Windows. `go install`, Homebrew, Winget, Scoop, `.deb`, `.rpm`, or copy a single file. No service to stand up.
- Self-hosted, runs fully local, air-gappable. No Detent vendor control plane.
- Deterministic gated merge train — serialized, one at a time, so what lands is always green.
- Multi-project and multi-instance fleet governance from one host, with weights, priority, pause, and fair scheduling.
- Budget and cost caps, with token / budget / rate-limit state live on the dashboard.
- Pluggable validation gates. Code defaults to `make check` + CI + automated review; a human approval-label gate is available when the workflow asks for one.
- `detent doctor` preflight checks, cross-platform config discovery, GoReleaser pipeline, checksum-verified self-update.
- MIT licensed. Free. You bring the model cost.
- **The strongest proof point of all:** Detent builds itself. `digitaldrywood/detent-orchestration` is Detent's own production config, and it dispatches the agents that build Detent. That repo is copyable as a template. Design a section around this — "it ships itself" is the credibility anchor and it is literally true.

## 6. Objections the site must answer

- "Will it merge garbage?" — Answer with the gate model and the serialized merge train, not with reassurance.
- "What if the agent goes off the rails?" — Isolated worktree per issue, your validation gate, your review policy, `Rework` lane for unresolved feedback.
- "Why Codex and not Claude?" — Detent dispatches agents non-interactively, headless, many at once. A ChatGPT plan covers scripted `codex exec` automation against a subscription you already have; Claude Code moves headless usage to a separate Agent SDK credit as of June 15, 2026. Detent still supports explicit backend routing including Claude Code, per role. This is a real question buyers ask — put it on the site, keep it factual, no vendor sniping.
- "Why GitHub Projects and not Linear?" — One ID space, one place to look. Plus cost (no per-seat charge) and API headroom (5,000 req/hr authenticated REST vs 1,500 against a complexity budget).
- "Is this a toy?" — Version history, CI badge, release channel, installer smoke tests on Ubuntu and Windows, the self-hosting proof.

## 7. Voice and tone

Precise, mechanical, unhurried. The vocabulary of the product is machined-metal and process-control: detent, catch, stop, notch, gate, lane, train, claim, lease, gate cleared. Lean into it.

Write like an engineer explaining a mechanism to another engineer who will check the claim. Concrete nouns over abstractions. Real config, real CLI output, real state names on the page. Numbers where numbers exist.

Avoid entirely: "supercharge", "unleash", "10x your team", "AI-powered", "revolutionize", "seamless", "effortless", "empower", "streamline", "leverage". Avoid the AI-marketing gradient-and-glow aesthetic. Avoid faked social proof, fake logo walls, and invented metrics — this is a young open-source project and pretending otherwise destroys the exact credibility the design is trying to build.

The README has one line of dry humor worth preserving as a tone marker: "You can keep reading by hand too; nobody will revoke your keyboard." One joke like that per page is the right density.

## 8. Visual direction

**Existing brand mark:** a "D" as a solid filled letterform with a counter, in indigo `#3730A3`, with a filled circle sitting in the lower counter — reading as the ball bearing seated in its notch. Square viewBox, works at 88px. There is a mono variant. Do not redesign it; build the palette and geometry around it.

**Existing product palette (the dashboard already ships this — the marketing site should feel like the same system):**

Dark, which is the default and the one to lead with: page `#0b0d10`, surface `#14171c`, elevated `#1c2027`, hairline `#262b33`, text `#edf0f4`, secondary text `#8a93a2`, dim `#808a97`. Status: ok `#34d399`, warn `#fbbf24`, error `#f87171`, info `#60a5fa`. Accent `#2dd4bf` (teal) — and there is a hard rule in the codebase worth honoring on the site: **the accent is for interactivity only — links, focus, selected state. Never for status.**

Light: page `#f7f8fa`, surface `#ffffff`, elevated `#f0f2f5`, hairline `#e4e7ec`, text `#1a1f27`, secondary `#5d6b80`. Status: ok `#046c4e`, warn `#92400e`, error `#b42318`, info `#1d4ed8`, accent `#0f766e`.

Note the tension: the brand mark is indigo, the product accent is teal. Concepts may resolve this either way — indigo as the marketing-surface brand color with teal reserved for interactive product surfaces, or unifying on one. Call the choice out explicitly rather than blurring it.

**Wordmark:** the domain is `detent.build`, which reads as a verb phrase rather than a URL. That is worth exploiting — a mono-set `detent.build` in the nav and footer, with the dot as a deliberate typographic beat, does more work than setting "Detent" in a display face. Concepts should try at least one treatment that uses the full domain as the wordmark.

**Type:** Geist for sans, Geist Mono for mono. Mono is not decoration here — state names, config, CLI, and lane labels should genuinely be mono.

**Geometry:** small radii (card 6px, chip 4px), a strict 4px spacing grid, hairline borders rather than shadows. Dense, instrument-panel, high-information. Think aircraft panel, oscilloscope, machinist's dial — not SaaS landing page. Restraint is the brand.

**Imagery:** the highest-value visual is the real dashboard — lanes with live counts, running agents, budget and rate-limit state, board-flow charts, timelines. Second is a state-machine diagram of the six lanes with the gates drawn as actual catches between them. Third is real terminal output. There should be no stock illustration and no 3D abstract blobs anywhere on this site.

## 9. Sitemap

- **Home** — the full argument, top to bottom.
- **How it works** — the state machine, gates, worktree isolation, merge train, in depth with diagrams.
- **Why Detent** — the interaction-model argument versus autonomy-first agents, plus the comparison matrix against Symphony, Copilot coding agent, Cursor, Hermes, OpenClaw, Hyperagent.
- **Dashboard** — a tour of the operator surface: lanes, agent activity, charts, trends, timelines, budget and rate-limit state, the terminal UI.
- **Install / Quick start** — platform-tabbed install commands, requirements, first run, `detent doctor`.
- **Docs** — either linked out to the repo or rendered; the site should not fake having docs it does not have.
- **Open source** — license, repo, releases, contributing, the self-hosting proof via `detent-orchestration`.

## 10. Home page, section by section

1. **Hero.** A one-line claim plus one line of mechanism, two buttons (install command with copy-to-clipboard, and View on GitHub), and — this is the important part — the six lanes rendered as the hero visual with work items sitting in them. Not a screenshot pasted in a laptop frame. The lanes themselves, live-looking, with one card visibly held at a gate.

   Headline candidates to work from: "Hold the work until the gate clears." / "Your process, running in parallel." / "A stop for every piece of work." / "Status-driven agentic orchestration. One binary." / "You write the issue. The gates decide when it lands."

2. **The inversion.** Autonomy-first assistants versus Detent's contract-first model, ideally as a two-column comparison using the OAuth token rotation example. This is the section that makes someone understand what they are looking at.

3. **How it works.** The six numbered steps from section 3 above, tied to a diagram of the state machine. Worth an interactive or scroll-driven treatment — this is the one place motion earns its place, showing an issue traverse the lanes and stop at a gate.

4. **The merge train.** Its own section, because it is the differentiator nobody else in the matrix has. Serialized rebase → CI-watch → merge, one at a time, so concurrent candidates never invalidate each other's CI.

5. **The operator surface.** The dashboard, large. Live counts, running agents, token/budget/rate-limit state, board flow, plus the terminal UI for people who live in tmux.

6. **Contracts, shown.** Real `detent.yaml` and `WORKFLOW.md` excerpts in a code surface. Developers believe config they can read.

7. **It builds itself.** The `detent-orchestration` proof, with the invitation to copy it as a template.

8. **Made for one host, many repos.** Multi-project scheduling, weights, priority, pause, fair scheduling, fleet governance.

9. **Install.** Platform-tabbed, one command per platform, requirements underneath (Go 1.26+ for source builds, Codex CLI signed in, `gh` assumed, a GitHub token scoped to the tracker mode).

10. **Honest footer.** MIT, GitHub, releases, docs, the Symphony lineage. Detent grew out of OpenAI's Symphony and says so — that lineage is a credibility asset, not something to bury.

## 11. What to avoid

No signup form, no "Book a demo", no pricing table — it is free and self-hosted, and any of those elements would misrepresent the product. No fake enterprise logo wall. No testimonial carousel with invented quotes. No hero video of a person typing. No claim about speed or throughput that the repository cannot substantiate. No autoplaying anything.

## 12. What to produce

Three distinct home-page concepts, meaningfully different in structure and not just in accent color. Suggested axes to differentiate along, though the designer should feel free to find better ones: (a) board-first — the Kanban lanes are the hero and the whole page is laid out as an instrument panel; (b) mechanism-first — the detent metaphor drawn literally, a dial with click-stops that the page scrolls through; (c) document-first — dense, typographic, almost a technical spec, the way a systems tool with a strong README earns trust. Each concept should show hero, the how-it-works section, and the dashboard section at minimum, in dark mode, with a note on how light mode differs.
