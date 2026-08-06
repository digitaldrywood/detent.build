# detent.build

Website for [Detent](https://github.com/digitaldrywood/detent) — status-driven
agentic work orchestration in a single Go binary.

Go · Echo · templ · HTMX · Tailwind CSS v4 · templUI. No database; the site is
a stateless binary.

## Develop

```sh
cp .envrc.example .envrc && direnv allow
make setup     # air, templ, templui, golangci-lint, npm deps
make dev       # http://localhost:3000
```

## Validate

```sh
make check     # templ generate, go vet, go test -race, golangci-lint
```

## Deployment

Deployed to Dokploy at https://detent.build, built with Nixpacks from
`nixpacks.toml` and served on container port 3000 behind Traefik. See
[docs/deploy.md](docs/deploy.md) — the port, the absence of a www host, and the
HSTS-preloaded `.build` TLD each impose a constraint that has a test guarding
it.

## Where the content lives

Every factual claim on the site is in `internal/content/content.go`, and each
traces back to the Detent repository. There are no invented metrics, no
testimonials, and no logo wall — see [docs/conventions.md](docs/conventions.md)
for why that is a rule rather than an oversight, along with the design-system
rules the site inherits from the product.
