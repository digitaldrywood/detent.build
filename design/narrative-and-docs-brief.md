# detent.build — narrative repositioning and docs build

You are working on detent.build, the product site for Detent (`digitaldrywood/detent`). Two jobs: replace the site's core narrative, and build real documentation on the site. Read this whole document before you touch anything. `design/detent-build-website-design-brief.md` in this repo is still valid source material for facts, users, and mechanism — this document supersedes it only on narrative framing.

Before writing copy, read the Detent repository itself: `README.md`, `docs/comparison.md`, `docs/concepts.md`, `docs/merge-train.md`, and the live config at `digitaldrywood/detent-orchestration`. Every claim you put on the site must be traceable to those, matching the standard already set at the top of `internal/content/content.go`. Do not invent capability. If you cannot source a claim, cut it.

## Part 1 — the narrative problem

The site currently leads with gating. The page title is "Hold the work until the gate clears", the H1 is "You write the issue. The gates decide when it lands.", and the hero paragraph ends on "only once the gates you wrote have cleared." That framing is the weakest slice of the product and it needs to be replaced.

Here is why it underperforms. "The gates decide" describes what the software refuses to do. It is a mechanism, not a promise, and its emotional register is restriction — the product's job, as stated, is to hold things back. A reader who does not already own this problem has no reason to care that something is being blocked, because we have not yet told them what they get. We are naming our own internal control flow instead of naming the reader's pain.

There is a second problem: gating is not actually differentiating. Every CI system gates. Every branch protection rule gates. Leading with gates positions Detent against `required_status_checks`, which is a fight about features rather than outcomes.

**The line to lead with is "manage work, not agents."** This is the organizing idea of the product and it should be the organizing idea of the site. Four words, addressed to the reader, describing what changes about their day. It promises relief from something the reader already hates — babysitting agents, reading transcripts to find out where something is, being the human scheduler for a pool of sessions. Everything else on the site is downstream of that sentence.

Two things about how to use it. First, it is descriptively accurate for Detent in a way it is not for most of the category: the unit you touch is the issue and the board state, not a chat session. You never address an agent. You mark work `Todo` and the runtime does the rest. Build the page so that claim is *demonstrated* — the lane row, the board, the config — rather than merely asserted.

Second, on provenance. This phrasing comes from OpenAI's Symphony, the project Detent grew out of, and the Detent README already credits that lineage openly. Keep crediting it. Adopting the framing while naming where it came from reads as confidence and intellectual honesty; adopting it silently reads as derivative to anyone who knows Symphony, and that is a meaningful slice of our audience. The strongest construction is to lead with the promise and then immediately show that we took the idea further than the spec did — one binary instead of a BEAM service, GitHub Projects instead of Linear, multi-project scheduling, a serialized merge train, a real operator surface. Symphony proposed managing work instead of agents; Detent is what it looks like shipped. That is a better story than either hiding the debt or avoiding the words.

The material for a much better story is already in this repo and in the Detent README, and the site simply did not lead with it. Three assets are being underused.

The name is the argument, and the existing design brief already says so: a detent is the catch that holds a moving part at a defined position until deliberately released — the click-stop on a dial, the notch on a ratchet. Note what that metaphor actually means mechanically, because the site has been reading it wrong. A detent does not block motion. A detent makes position *definite*. It is what makes a dial feel like it has settings instead of being a smear of continuous rotation. That is a far stronger and more accurate claim: Detent gives agentic work definite states. Work is always at a known stop, never in an ambiguous middle. Contrast that with the industry default, where an agent is somewhere in a session and you find out where by reading a transcript.

The inversion is the real promise, and the README states it better than the site does: "You are not steering an agent; you are running your own engineering process at scale." And: "The system does not try to be smarter than you; it tries to be as disciplined as you would be, every time, in parallel." That is the sentence the whole site should be built around. It concedes nothing to the reader's skepticism, it flatters their existing standards rather than replacing them, and it names the actual benefit — parallelism without loss of bar.

Detent builds itself, and this is the proof asset most sites would kill for. `digitaldrywood/detent-orchestration` is the production config that dispatches the agents that build Detent. The site has a `ShipsItself` section; it should be far more prominent and far more concrete, because it is the only evidence that matters to a skeptical staff engineer.

## Part 2 — what to change

Rewrite the hero around "manage work, not agents." That is the promise. The gate belongs in the mechanism section, where it is a feature supporting the promise rather than the promise itself. The supporting layer under the headline should answer the immediate question a skeptical reader asks — *how* do you manage work instead of agents — and the answer is the definite-states argument above plus the lane row that makes it visible. Write the subhead and body several ways and choose on strength, but do not bury the four words.

Own the Symphony lineage explicitly and early rather than burying it, especially now that the site leads with Symphony's phrasing. "Detent grew out of OpenAI's Symphony" is a credibility asset — it says the idea has independent validation and that we took it further. The README's comparison section already does this well and generously. Do not position Detent as anti-Symphony; position it as Symphony's idea taken from spec to shipped system, with the divergences that matter (one Go binary versus a BEAM service, GitHub Projects versus Linear, multi-project scheduling, the serialized merge train, a real operator surface).

Keep the lane row. Todo → In Progress → Human Review → Rework → Merging → Done is the most recognizable visual asset the product has, and it is the *visible form* of the definite-states claim. Tie the two together explicitly.

Do not soften the target audience. The primary reader is a staff or principal engineer who already has a high bar, is suspicious of agents that merge code nobody read, and will read a `detent.yaml` before a testimonial. Write for that person. Config beats adjectives. Concrete beats aspirational.

Preserve the honesty already in the codebase. `internal/content/content.go` opens by promising nothing is invented, and there is a comment marking the hand-maintained version constant as something that will rot. That discipline is part of the product's character — do not regress it while rewriting.

## Part 3 — the documentation build

The site has no product documentation. `DocsBase` in `internal/content/content.go` is `https://github.com/digitaldrywood/detent/blob/main/docs/`, so every documentation link on detent.build sends the reader to a GitHub blob view. That is the gap to close. The `docs/` directory in this repo holds `conventions.md` and `deploy.md`, which are about building the website, not about using Detent — do not confuse the two.

Do not start by writing pages. Start by deciding the sourcing model, because it determines everything downstream and it is the decision most likely to be made badly by default.

The core tension: Detent's documentation lives in `digitaldrywood/detent/docs/` and is the authoritative reference, versioned with the code that implements it. If you copy it into this repo you have forked your documentation and it will drift, which is exactly the class of failure Detent itself exists to prevent — and the irony will be noticed. If you render it remotely at request time you have coupled the site's availability to the GitHub API. If you vendor it at build time you get correctness and speed but need a defined refresh path.

Investigate and choose deliberately. Evaluate at minimum: build-time vendoring pinned to a release tag or commit, with the pin visible on the page and a documented refresh step; build-time fetch from the GitHub API with the result committed; and runtime fetch with caching. Weigh them against the constraints this site already lives under — it is a stateless single binary with no database, deployed to Dokploy via Nixpacks, and every deployment constraint here has a test guarding it, per this repo's README. Whatever you choose must preserve statelessness and must not make a page render depend on a live third-party call. State your choice and your reasoning before implementing, and if the sourcing decision has a consequence the operator should weigh, surface it rather than deciding silently.

Then determine what to document, from evidence rather than assumption. Read the Detent repo's `docs/` directory in full and inventory what exists. Read `README.md`'s Documentation section, which already groups the reference into "Get started", "Operate Detent", and "Reference and contribute" — that grouping is a real information architecture and you should not discard it without a reason. Cross-check against the site's existing routes, which are `/how-it-works`, `/why-detent`, `/dashboard`, `/install`, and `/open-source`, and decide where documentation sits relative to those, since some of them already carry explanatory load and you must not duplicate or contradict them.

Find the gaps the repository documentation does not cover, because those are where new writing is genuinely needed rather than merely re-hosted. Look for the questions a new operator hits that no current document answers end to end. Use the repository's own issue history and recent commits as evidence of what confuses people. One concrete example worth documenting from real operational experience: Detent reads a project's merge-gate policy from `detent.yaml` in the configured working checkout, so the branch that checkout happens to be on silently determines gate behavior — a stale checkout can demand a retired CI check name and loop work indefinitely. That is the kind of hard-won operational knowledge that belongs in documentation and is currently only in the code and in `detent doctor`.

Write documentation to the same evidential standard as the marketing copy. Every documented behavior must be traceable to the Detent repository at the pinned version. Where behavior differs by version, say so. Where something is a known sharp edge, document the sharp edge rather than pretending it is smooth — the target reader trusts documentation that admits limits and distrusts documentation that does not.

Match the engineering conventions already in this repo: Go, Echo, templ, HTMX, Tailwind v4, templUI, no database, stateless binary. Read `docs/conventions.md` here before writing code. Add tests in the style already established, including for whatever sourcing mechanism you choose, and run `make check` before you call any of this done.

## Part 4 — filing the follow-up work

When the narrative rewrite and the docs sourcing decision are settled, break the remaining build into issues rather than doing it all in one pass.

`digitaldrywood/detent.build` is an onboarded Detent project. It is registered in the host `global.yaml` as project id `detent.build` with weight 1 and priority 3, its contracts are `detent.yaml` and `WORKFLOW.md` in the repository root, and the orchestrator has loaded it. Read both contracts before filing anything — they are the authority on this project's lifecycle, and the notes below only summarize what matters for issue creation.

Board state is driven by **labels**, not a ProjectV2 board: `github_status_source: label` with prefix `detent:`. All eight state labels exist on the repository. Label each issue `detent:todo` at creation so it is eligible for dispatch.

Authorization is **by issue author**, not by a separate authorization label. `detent.yaml` sets `authorization.author_in` to `corylanou`, `loganlanou`, and `jarvislanou`. An issue filed by any other account will not dispatch regardless of its labels, so confirm which account `gh` is authenticated as before you file. There is no `detent` authorization label on this repository; do not try to apply one.

Include an explicit effort block in every issue. Choose from the rubric: `medium` for small, mechanical, tightly specified work with complete acceptance criteria; `high` for a standard feature or fix with some ambiguity or a cross-cutting surface; `xhigh` for a new subsystem, tricky state, or recovery semantics. Leave `model` unset so the issue inherits the fleet standard. The block is a fenced `detent-agent` code block containing `schema: 1` on one line and `effort: <level>` on the next.

Write concrete acceptance criteria in every issue — the runtime will dispatch an agent against what you wrote, and vague criteria produce vague pull requests. Note that this project currently sets `required_status_checks: []`, so CI check names are not part of its gate; do not write acceptance criteria that assume a named required check exists.

File flat issues with `Depends on #N` lines where ordering matters. Do not create an epic or a tracking issue.

## Constraints

Do not invent product capability, benchmarks, customer counts, testimonials, or social proof. Everything on this site must be substantiated by the Detent repository or the public orchestration config.

Do not include timeline estimates or phased schedules in anything you produce.

Do not remove the existing factual-sourcing discipline in `internal/content/content.go` while rewriting copy.

When the narrative rewrite and the documentation sourcing decision are ready, present them for review before building out the full page set. The narrative choice and the sourcing model are both decisions the operator will want to weigh; the implementation that follows them is not.
