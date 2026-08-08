// Package content holds the site's factual copy in one place.
//
// Every claim here is traceable to the Detent repository — README.md,
// docs/comparison.md, docs/concepts.md, docs/merge-train.md, or the live
// digitaldrywood/detent-orchestration config. Nothing in this package is
// invented, and nothing describes behavior the repository cannot substantiate.
package content

import (
	"context"

	"detent.build/internal/ctxkeys"
)

const (
	RepoURL            = "https://github.com/digitaldrywood/detent"
	OrchestrationURL   = "https://github.com/digitaldrywood/detent-orchestration"
	ReleasesURL        = "https://github.com/digitaldrywood/detent/releases"
	LicenseURL         = "https://github.com/digitaldrywood/detent/blob/main/LICENSE"
	IssuesURL          = "https://github.com/digitaldrywood/detent/issues"
	ContributingURL    = "https://github.com/digitaldrywood/detent/blob/main/CONTRIBUTING.md"
	CIURL              = "https://github.com/digitaldrywood/detent/actions/workflows/ci.yml"
	SymphonyURL        = "https://github.com/openai/symphony"
	CodexCLIURL        = "https://github.com/openai/codex"
	NonCodeWorkflowURL = RepoURL + "/blob/main/docs/examples/non-code-artifact/README.md"

	DocsBase = RepoURL + "/blob/main/docs/"

	// Version is hand-maintained and will rot. It should come from the
	// releases API or a build flag; until then TestVersionMatchesBare keeps the
	// two spellings in sync, and nothing else.
	Version     = "v0.55.0"
	VersionBare = "0.55.0"
	License     = "MIT"
)

// Doc returns a link to a document in the Detent repository.
func Doc(path string) string { return DocsBase + path }

// VersionFromCtx returns the release tag for this request. A background
// watcher refreshes it from the GitHub releases API; when that is disabled or
// has not succeeded yet, this falls back to the compiled-in Version so a page
// never renders an empty release.
func VersionFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxkeys.Version).(string); ok && v != "" {
		return v
	}
	return Version
}

// Lane is one configured state on the board.
type Lane struct {
	Name  string
	Slug  string
	Blurb string
}

// Lanes is the delivery path in Detent's own production workflow.
var Lanes = []Lane{
	{"Todo", "todo", "Claimed by the scheduler when capacity and dependencies allow."},
	{"In Progress", "in-progress", "A Codex agent is working in an isolated worktree on its own branch."},
	{"Human Review", "human-review", "Held. The workflow asked for a human gate; nothing advances until you release it."},
	{"Rework", "rework", "Unresolved review feedback sent the work back for another pass."},
	{"Merging", "merging", "In the serialized merge train: rebase, CI-watch, merge — one at a time."},
	{"Done", "done", "Merged green, or closed as cancelled."},
}

// Step is one numbered stage in the how-it-works mechanism, drawn from the
// README's "How it works" section.
type Step struct {
	Num     int
	Title   string
	Body    string
	Detail  string
	DocPath string
	DocName string
}

var Steps = []Step{
	{
		Num:   1,
		Title: "You write the contracts",
		Body:  "Each project has a checked-in detent.yaml machine contract — tracker bindings, states, lifecycle policy, scheduling, retries, leases, gates — plus a checked-in, portable WORKFLOW.md agent instruction contract.",
		Detail: "The prompt declares the project's required CI stage categories and the commands and check names that satisfy each one. " +
			"Optional gitignored detent.local.yaml and WORKFLOW.local.md apply machine-specific overrides without touching the shared contracts.",
		DocPath: "config.md",
		DocName: "Configuration reference",
	},
	{
		Num:     2,
		Title:   "You mark an issue Todo",
		Body:    "Detent claims it, creates an isolated Git worktree from your source checkout, and dispatches a Codex agent with the contract — moving the issue to In Progress.",
		Detail:  "The source of truth can be a GitHub ProjectV2 board, or Detent can run boardless from an issue Status field or repository status labels while supplying its own Kanban view.",
		DocPath: "concepts.md",
		DocName: "Connectors and board states",
	},
	{
		Num:     3,
		Title:   "The agent works",
		Body:    "In its own branch. It runs your validation gate and opens or updates a pull request.",
		Detail:  "Review-gate workflows move the issue to Human Review. Autopilot workflows leave it active with status: complete in the Workpad.",
		DocPath: "execution-seams.md",
		DocName: "Execution seams",
	},
	{
		Num:   4,
		Title: "Gates decide",
		Body:  "The workflow decides whether promotion to Merging waits in Human Review, waits in the active lane, requires a current-head automated PR review, or only needs linked PR + green CI + quiet time.",
		Detail: "Unresolved feedback sends the issue to Rework for another pass. Code defaults to make check plus CI plus automated review; " +
			"a human approval-label gate is available when the workflow explicitly asks for one.",
		DocPath: "concepts.md",
		DocName: "Review gates",
	},
	{
		Num:     5,
		Title:   "The merge train is serialized",
		Body:    "One rebase, CI-watch, and merge at a time, so concurrent candidates never invalidate each other's CI. Then the issue is Done.",
		Detail:  "Train width is configuration, not a fixed rule — Detent's own board runs it at one candidate at a time.",
		DocPath: "merge-train.md",
		DocName: "Merge train",
	},
	{
		Num:     6,
		Title:   "One host, many repos",
		Body:    "A global.yaml runs multiple projects with weights, priority, pause, and fair scheduling.",
		Detail:  "The web dashboard and terminal UI show live counts, running agents, token / budget / rate-limit state, and board flow.",
		DocPath: "multi-project.md",
		DocName: "Multi-project operation",
	},
}

// ProofPoint is a substantiated claim for the credibility section.
type ProofPoint struct {
	Title string
	Body  string
}

var ProofPoints = []ProofPoint{
	{"One CGO-free Go binary", "macOS, Linux, and Windows. go install, Homebrew, Winget, Scoop, .deb, .rpm, or copy a single file. No service to stand up."},
	{"Self-hosted, air-gappable", "Runs fully local. There is no Detent vendor control plane and no Detent telemetry — the only outbound traffic is to your tracker and your model provider."},
	{"Deterministic gated merge train", "Serialized rebase, CI-watch, and merge — one candidate at a time, so what lands is always green."},
	{"Multi-instance fleet governance", "Many projects from one host with weights, priority, pause, and fair scheduling."},
	{"Budget and cost caps", "Token, budget, and rate-limit state live on the dashboard, with caps you configure."},
	{"Pluggable validation gates", "Code defaults to make check plus CI plus automated review. A human approval-label gate is available when the workflow asks for one."},
	{"detent doctor preflight", "Cross-platform config discovery, a GoReleaser pipeline, and checksum-verified self-update."},
	{"MIT licensed, free", "You bring the model cost. Nothing else is metered."},
}

// InstallTarget is one platform tab on the install surface.
type InstallTarget struct {
	Key      string
	Label    string
	Shell    string
	Primary  Command
	Others   []Command
	Footnote string
}

// Command is a single copyable install command with a plain-English caption.
type Command struct {
	Caption string
	Cmd     string
}

var InstallTargets = []InstallTarget{
	{
		Key:     "macos",
		Label:   "macOS",
		Shell:   "sh",
		Primary: Command{"Homebrew — recommended when you already manage CLI tools with brew", "brew install digitaldrywood/tap/detent"},
		Others: []Command{
			{"Or build from source with Go", "go install github.com/digitaldrywood/detent/cmd/detent@latest"},
		},
		Footnote: "Upgrade with brew upgrade digitaldrywood/tap/detent.",
	},
	{
		Key:     "linux",
		Label:   "Linux",
		Shell:   "sh",
		Primary: Command{"Shell installer — downloads the release archive and verifies its SHA-256 checksum", "curl -fsSL https://raw.githubusercontent.com/digitaldrywood/detent/main/install.sh | sh"},
		Others: []Command{
			{"Or a native .deb, so apt owns the binary, removal, and upgrades",
				"DETENT_VERSION=" + VersionBare + " DETENT_ARCH=amd64 \\\n" +
					"  curl -LO \"https://github.com/digitaldrywood/detent/releases/download/v${DETENT_VERSION}/detent_${DETENT_VERSION}_linux_${DETENT_ARCH}.deb\" \\\n" +
					"  && sudo apt install \"./detent_${DETENT_VERSION}_linux_${DETENT_ARCH}.deb\""},
			{"Or a native .rpm",
				"DETENT_VERSION=" + VersionBare + " DETENT_ARCH=amd64 \\\n" +
					"  curl -LO \"https://github.com/digitaldrywood/detent/releases/download/v${DETENT_VERSION}/detent_${DETENT_VERSION}_linux_${DETENT_ARCH}.rpm\" \\\n" +
					"  && sudo rpm -Uvh \"./detent_${DETENT_VERSION}_linux_${DETENT_ARCH}.rpm\""},
			{"Homebrew works on Linux too", "brew install digitaldrywood/tap/detent"},
		},
		Footnote: "The installer writes to /usr/local/bin when writable, otherwise $HOME/.local/bin. Set DETENT_INSTALL_DIR to override.",
	},
	{
		Key:     "windows",
		Label:   "Windows",
		Shell:   "powershell",
		Primary: Command{"Winget — use the package manager that already manages your developer tools", "winget install --id DigitalDrywood.Detent --source winget"},
		Others: []Command{
			{"Or Scoop, for a user-local install", "scoop bucket add digitaldrywood https://github.com/digitaldrywood/scoop-bucket\nscoop install detent"},
			{"Or the PowerShell installer, for bootstrap and CI images", "irm https://raw.githubusercontent.com/digitaldrywood/detent/main/install.ps1 | iex"},
		},
		Footnote: "The PowerShell installer verifies the SHA-256 checksum, installs to %LOCALAPPDATA%\\detent\\bin, and adds it to the user PATH.",
	},
	{
		Key:     "source",
		Label:   "Source",
		Shell:   "sh",
		Primary: Command{"Any platform, straight from the module path", "go install github.com/digitaldrywood/detent/cmd/detent@latest"},
		Others: []Command{
			{"Or run the repository-local installer from a checkout", "./install.sh"},
		},
		Footnote: "Source builds need Go 1.26 or newer. detent update prints the recommended command rather than overwriting a source-built binary.",
	},
}

// Requirement is one prerequisite for a first run.
type Requirement struct {
	Name     string
	Detail   string
	Verify   string
	Optional bool
}

var Requirements = []Requirement{
	{
		Name:   "OpenAI Codex CLI, signed in",
		Detail: "Detent drives every agent through codex app-server on the host that dispatches them.",
		Verify: "codex --version",
	},
	{
		Name:     "GitHub CLI (gh)",
		Detail:   "Used for authentication and GitHub lookups. Optional, but assumed throughout the docs.",
		Verify:   "gh auth status",
		Optional: true,
	},
	{
		Name:   "A GitHub token scoped to your tracker mode",
		Detail: "ProjectV2 mode usually needs repo, read:org, read:project, and write project. Boardless issue-field mode needs repository issue access plus organization issue-field read access.",
		Verify: "detent doctor --port 0",
	},
	{
		Name:     "Go 1.26 or newer",
		Detail:   "Only for go install or building from source. Release binaries need nothing installed.",
		Verify:   "go version",
		Optional: true,
	},
	{
		Name:     "Claude Code CLI, signed in",
		Detail:   "Only when routing selected roles to the claude_code backend. Detent stores no Claude credentials; it uses the ambient CLI login or ANTHROPIC_API_KEY.",
		Verify:   "claude --version",
		Optional: true,
	},
}

// Competitor is a column in the comparison matrix.
type Competitor struct {
	Name string
	URL  string
	What string
}

// Competitors mirrors docs/comparison.md, last verified July 6, 2026.
var Competitors = []Competitor{
	{"Detent", RepoURL, "Single-binary Go orchestrator for GitHub-native issue-to-PR work, using ProjectV2 or boardless status sources."},
	{"OpenAI Symphony", SymphonyURL, "Detent's origin point: an Apache-2.0 spec plus an Elixir reference implementation for Codex on Linear."},
	{"Copilot agent", "https://docs.github.com/en/copilot/concepts/agents/cloud-agent/about-cloud-agent", "GitHub's paid issue- or prompt-to-branch-and-PR cloud agent."},
	{"Cursor", "https://cursor.com/cloud", "IDE-first agent product with cloud and background agents, automations, and optional self-hosted workers."},
	{"Hermes", "https://github.com/NousResearch/hermes-agent", "Nous's MIT personal assistant with memory, skills, model providers, and a messaging gateway."},
	{"OpenClaw", "https://github.com/openclaw/openclaw", "MIT personal assistant centered on a local gateway and cross-channel automation."},
	{"Hyperagent", "https://www.hyperagent.com/", "Airtable's closed-source, cloud-hosted agent platform for persistent agents with identity, tools, and knowledge."},
}

// Verdict is one cell in the comparison matrix.
type Verdict struct {
	// Mark is "yes", "partial", "no", or "na".
	Mark string
	Note string
}

// Row is one capability across every tool. Cells are ordered to match Competitors.
type Row struct {
	Capability string
	Cells      []Verdict
}

// Matrix mirrors the feature matrix in docs/comparison.md exactly.
var Matrix = []Row{
	{"Self-hosted, no vendor control plane", []Verdict{
		{"yes", ""}, {"yes", ""}, {"no", ""}, {"no", ""}, {"yes", ""}, {"yes", ""}, {"no", "hosted"},
	}},
	{"Runs fully local / air-gappable", []Verdict{
		{"yes", ""}, {"yes", ""}, {"no", ""}, {"no", ""}, {"yes", ""}, {"yes", ""}, {"no", "cloud"},
	}},
	{"Board/tracker-native (issue→PR)", []Verdict{
		{"yes", "GH Projects, issue fields, or labels"}, {"yes", "Linear"}, {"yes", "GH Issues"},
		{"no", ""}, {"no", ""}, {"no", ""}, {"no", "own workspace"},
	}},
	{"Deterministic gated merge train", []Verdict{
		{"yes", ""}, {"partial", "per spec"}, {"no", ""}, {"no", ""}, {"no", ""}, {"no", ""}, {"no", ""},
	}},
	{"Budget / cost caps", []Verdict{
		{"yes", ""}, {"no", ""}, {"partial", ""}, {"partial", ""}, {"no", ""}, {"no", ""}, {"yes", "hosted controls"},
	}},
	{"Multi-project", []Verdict{
		{"yes", ""}, {"no", ""}, {"yes", ""}, {"yes", ""}, {"no", ""}, {"no", ""}, {"partial", "workspace-scoped"},
	}},
	{"Multi-instance fleet governance", []Verdict{
		{"yes", ""}, {"no", ""}, {"no", ""}, {"no", ""}, {"no", ""}, {"no", ""}, {"partial", "hosted agent controls"},
	}},
	{"Model-agnostic, BYO incl. local", []Verdict{
		{"partial", "codex now, seam shipped"}, {"no", "Codex"}, {"yes", "vendor-managed"}, {"yes", "vendor-managed"},
		{"yes", ""}, {"yes", ""}, {"no", "cloud-managed"},
	}},
	{"Local skills / workflows (your e2e etc.)", []Verdict{
		{"yes", ""}, {"partial", ""}, {"no", ""}, {"partial", ""}, {"yes", ""}, {"yes", ""}, {"yes", "hosted skills/knowledge"},
	}},
	{"Multi-channel triggers", []Verdict{
		{"partial", "tracker-driven"}, {"partial", "Linear"}, {"partial", "GitHub"}, {"partial", "IDE/cloud tasks"},
		{"yes", "messaging gateway"}, {"yes", "local gateway"}, {"yes", "Slack, schedules, webhooks, email, Telegram, Live Mode"},
	}},
	{"Open source", []Verdict{
		{"yes", "MIT"}, {"yes", "Apache-2.0"}, {"no", ""}, {"no", ""}, {"yes", "MIT"}, {"yes", "MIT"}, {"no", "closed-source"},
	}},
	{"Free (bring your own model cost)", []Verdict{
		{"yes", ""}, {"yes", ""}, {"no", "paid"}, {"no", "paid"}, {"yes", ""}, {"yes", ""}, {"no", "usage-billed"},
	}},
	{"Single static binary", []Verdict{
		{"yes", ""}, {"no", "Elixir/BEAM"}, {"na", "SaaS"}, {"na", "SaaS"}, {"no", "gateway"}, {"no", "gateway"}, {"na", "SaaS"},
	}},
	{"~5-min setup", []Verdict{
		{"yes", ""}, {"no", ""}, {"yes", "zero-install"}, {"yes", ""}, {"partial", ""}, {"partial", ""}, {"partial", "hosted onboarding"},
	}},
}

// MatrixAsOf is the verification date carried in docs/comparison.md.
const MatrixAsOf = "July 6, 2026"

// FleetProject is a row in the multi-project scheduling table. These are the
// projects in Detent's own global.yaml.
type FleetProject struct {
	ID       string
	Weight   string
	Priority string
}

// Fleet is the projects block of Detent's own global.yaml. Only the fields
// that file actually sets — there is no per-project state column in it, and
// inventing one would make the "from Detent's own config" claim false.
var Fleet = []FleetProject{
	{"detent", "1", "0"},
	{"gopher-ai", "1", "3"},
	{"gopher-corp", "1", "3"},
}

// FleetNote explains what the table above is.
const FleetNote = "The projects block of Detent's own global.yaml. That file also sets " +
	"max_concurrent_agents: 5, strict scheduling, and a 1h fair-share half-life."

// Card is one work item sitting in a lane on the hero board.
type Card struct {
	Ref   string
	Repo  string
	Title string
	// Status is the one-line runtime state shown under the title.
	Status string
	// Tone selects the status color: "ok", "warn", "info", or "" for none.
	Tone string
	// Held marks the card that is stopped at a gate — the detent itself.
	Held bool
}

// BoardColumn is a lane on the hero board with the cards currently in it.
type BoardColumn struct {
	Lane  Lane
	Count string
	Cards []Card
	// Overflow is an optional footer line, e.g. "+51 more".
	Overflow string
}

// Board is the hero visual: Detent's production delivery path with real work.
//
// Every issue number and title below is real, merged work from
// digitaldrywood/detent that Detent's own agents produced. The arrangement
// across the displayed delivery path is composed — a live board only ever
// shows one instant. BoardCaption says so on the page.
var Board = []BoardColumn{
	{
		Lane:  Lanes[0],
		Count: "2",
		Cards: []Card{
			{Ref: "#1626", Repo: "detent", Title: "feat(admission): surface pending proposals"},
			{Ref: "#1619", Repo: "detent", Title: "feat(health): surface stranded active work"},
		},
	},
	{
		Lane:  Lanes[1],
		Count: "2",
		Cards: []Card{
			{Ref: "#1614", Repo: "detent", Title: "chore(safety): guard dispatch capacity paths",
				Status: "agent working", Tone: "ok"},
			{Ref: "#1611", Repo: "detent", Title: "fix(scheduler): clean orphan cycle state",
				Status: "make check running", Tone: "ok"},
		},
	},
	{
		Lane:  Lanes[2],
		Count: "1",
		Cards: []Card{
			{Ref: "#1585", Repo: "detent", Title: "fix(orchestrator): add token progress brake",
				Status: "held at the gate", Tone: "warn", Held: true},
		},
	},
	{
		Lane:  Lanes[3],
		Count: "1",
		Cards: []Card{
			{Ref: "#1593", Repo: "detent", Title: "fix(board): collapse staleness warnings",
				Status: "2 threads unresolved", Tone: "info"},
		},
	},
	{
		Lane:  Lanes[4],
		Count: "1",
		Cards: []Card{
			{Ref: "#1628", Repo: "detent", Title: "docs(cli): refresh version output example",
				Status: "rebased, CI watch", Tone: "ok"},
		},
	},
	{
		Lane:  Lanes[5],
		Count: MergedPRs,
		Cards: []Card{
			{Ref: "#1624", Repo: "detent", Title: "test(project): relax watcher deadlock guard",
				Status: "merged, green", Tone: "ok"},
		},
	},
}

// MergedPRs is the number of pull requests landed on digitaldrywood/detent.
// Derived, and checkable:
//
//	git log --format='%s' | rg -c '\(#[0-9]+\)$'
//
// Squash merges put the PR reference in the commit subject, so this counts
// landed pull requests rather than raw commits. Hand-updated; see the note on
// Version about the same staleness problem.
const MergedPRs = "722"

// BoardLabel replaces a "live" indicator. The board is a composition, and
// saying so in the header rather than in a caption underneath is the whole
// difference between an illustration and a false claim.
const BoardLabel = "composed snapshot"

// BoardCaption is the honesty note printed under the hero board.
const BoardCaption = "The six lanes shown come from Detent's own production configuration, not a fixed " +
	"product state model; each workflow defines its own states. This is not a live board: the issue numbers and " +
	"titles come from real merged work in digitaldrywood/detent, composed across the path for this " +
	"illustration. The counts describe the illustration; " + MergedPRs +
	" is the all-time landed pull requests on that repository."

// MergeTrainStage is one step in the serialized merge train diagram.
type MergeTrainStage struct {
	Label string
	Tone  string
}

var MergeTrain = []MergeTrainStage{
	{"queued candidates wait", ""},
	{"rebase onto main", "info"},
	{"CI watch", "warn"},
	{"merge", "ok"},
	{"main stays green", ""},
}
