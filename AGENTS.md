# AGENTS.md

Orientation for coding agents working in this repository. `WORKFLOW.md` is the
Detent agent contract and takes precedence for state transitions, the
validation gate, and the CI quality-gate mapping. `docs/conventions.md` is the
committed source of truth for conventions, and `docs/deploy.md` for production
constraints. Read both before changing anything.

## Reasoning Effort

Use Codex Astra (`gpt-6-astra`) at low effort by default for features, fixes,
tests, reviews, and routine implementation, including cross-component work.

Every issue must include an explicit override, with model unset:

```detent-agent
schema: 1
effort: low
```

- `low` — the default for all work without a specific documented reason to escalate.
- `medium` — an exception for a concrete reasoning difficulty or evidence that low was insufficient; explain the reason in the issue.
- `high` — rare, significant research or architecture work with a written justification.
- `xhigh` and `max` — operator-designated only; never assign automatically.

Concurrency, recovery, routing, multiple files, or a new endpoint alone do not
justify higher effort. Preserve intentional operator exceptions. Configured
complexity levels default to low; verify any approved exception against the
runtime effort ceiling rather than raising broad defaults.

## Validation

The configured gate is:

```sh
make check && CGO_ENABLED=0 go build -o /dev/null ./cmd/server
```

`make check` runs `templ generate`, the Tailwind build, `go vet ./...`,
`go test -race ./...`, and `golangci-lint run`. It does not run the
CGO-disabled server build that CI's `check` job runs, and it does not run
`templ fmt`. Run `make lint` when you touch templ files.

## Rules That Are Not Negotiable

These are restated in full in `WORKFLOW.md`. In short:

- **Content is sourced, not written.** Every claim lives in
  `internal/content/content.go` and must trace to the `digitaldrywood/detent`
  repository. Do not invent a capability, metric, testimonial, or logo. The
  comparison matrix keeps the rows where Detent loses.
- **The design system is imported, not approximated.** Teal is for
  interactivity only and never for status. Indigo is the brand mark alone.
  6px card radius, 4px chip radius, hairline borders, no shadows, 4px grid.
- **Mono means machine.** Geist Mono is for literal strings — state names,
  config, CLI output, lane labels — not decoration.
- **One thing animates.** The pulsing agent dot and the held card's ring.
  `prefers-reduced-motion` turns both off.
- **No signup, no pricing, no demo booking, no fake social proof.**
- **Deployment.** `PORT` stays 3000. No www, ever. `SITE_URL` is
  `https://detent.build` with no trailing slash. No TLS or http-to-https
  redirect in Go; Traefik already does both. `go.mod` stays at `go 1.25`.
  Each of these has a test in `internal/handler/handler_test.go`.
