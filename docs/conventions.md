# Conventions

Rules that are not stylistic preferences — breaking one of these produces a
site that misrepresents the product or drifts away from it. Deployment
constraints live separately in [deploy.md](deploy.md).

## Content is sourced, not written

`internal/content/content.go` holds every factual claim on the site, and each
one traces to the Detent repository: `README.md`, `docs/comparison.md`,
`docs/concepts.md`, or the live `digitaldrywood/detent-orchestration` config.

Do not add a capability claim, a metric, a testimonial, or a logo the
repository cannot substantiate. The design brief is explicit that this is a
young open-source project and that pretending otherwise destroys the exact
credibility the site is trying to build.

Specifically:

- The comparison matrix is copied from `docs/comparison.md` **including the
  rows Detent loses**. Keep it that way, and keep the "last verified" date
  honest.
- The hero board uses real merged pull requests from `digitaldrywood/detent`.
  The arrangement across lanes is composed — one item at every stop at once,
  which a live board never shows — and `content.BoardCaption` says so on the
  page. If you change the cards, keep the caption true.
- No signup form, no "book a demo", no pricing table. The product is free and
  self-hosted; any of those would misrepresent it.

## The design system is the product's, imported not approximated

The `@theme` block in `static/css/input.css` is copied from
`detent/static/css/input.css` so the marketing site and the dashboard are
demonstrably the same system rather than two things that look similar.

- **Five semantic colors** — ok, warn, err, info — **plus one accent.** Never
  invent a sixth semantic.
- **The teal accent is for interactivity only**: links, focus, selected state.
  **Never for status.** This rule is enforced in the product's own codebase and
  the site honors it.
- **Indigo (`--color-brand`) is the brand mark's color** and appears nowhere
  else. It is never a status and never a link.
- Card radius 6px, chip radius 4px, hairline borders, no shadows, strict 4px
  spacing grid.
- **Mono means machine.** State names, config, CLI output, and lane labels are
  Geist Mono because they are literal strings, not decoration.
- **One thing animates**: the 6px pulsing dot per active agent, and the held
  card's ring. `prefers-reduced-motion` turns both off. Do not add more.

Fonts are self-hosted from the product repo and HTMX is vendored into
`static/js`, so the site makes no third-party requests and the CSP stays tight.

## templUI

Components are added with the CLI (`templui add <component>`), which fetches
the `Script()` template that manual copies miss. Currently installed:
`copybutton` and `icon`.

templUI ships shadcn-shaped token names. They are mapped onto Detent's palette
once, in the `@theme inline` block in `input.css`, so its components inherit
this design system instead of importing a second one. Any component needing
JavaScript must have its `Script()` called in `templates/layouts/base.templ`,
and `assets/js` must stay served.

The install platform tabs are HTMX rather than templUI's `tabs` component on
purpose: every tab is a real URL, so it works with JavaScript off, and the
swap replaces the whole tab strip so the selected state stays truthful.

## templ gotchas hit while building this

- A prose line that **begins** with `for`, `if`, or `switch` is parsed as a
  control keyword, not text. Rephrase, or move the word off the line start.
- Closing a `<div>` with `}` instead of `</div>` produces a misleading
  "expected nodes, but none were found" error pointing at the enclosing
  component rather than the mistake.
- Go expressions do not interpolate inside `<script>` tags. Use data
  attributes or `templ.JSFuncCall`.

## Section numbering

Interior pages number their sections from 01. Partials shared across pages
(`Inversion`, `MergeTrainSection`, `ShipsItself`) take the number as a
parameter, because the same block sits at a different position on each page.
If you reorder sections, renumber the whole page.

## Note

`CLAUDE.md` mirrors this document locally for agent sessions. It is covered by
a global gitignore, so this file is the committed source of truth — update
this one.
