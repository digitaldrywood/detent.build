package docsregistry

type Origin string

const (
	OriginUpstream Origin = "upstream"
	OriginSite     Origin = "site"
)

type Page struct {
	Group      string
	Title      string
	SourcePath string
	PublicPath string
	Origin     Origin
}

type Alias struct {
	PublicPath    string
	CanonicalPath string
}

type Tombstone struct {
	PublicPath string
}

type ChangeKind string

const (
	ChangeAdded          ChangeKind = "added"
	ChangeDeleted        ChangeKind = "deleted"
	ChangeProbableRename ChangeKind = "probable-rename"
)

type Resolution string

const (
	ResolutionPublished   Resolution = "published"
	ResolutionUnpublished Resolution = "unpublished"
	ResolutionStable      Resolution = "stable"
	ResolutionAlias       Resolution = "alias"
	ResolutionTombstone   Resolution = "tombstone"
)

type InventoryDecision struct {
	FromCommit    string
	ToCommit      string
	Kind          ChangeKind
	PreviousPath  string
	CurrentPath   string
	Resolution    Resolution
	PublicPath    string
	CanonicalPath string
}

type Registry struct {
	Pages      []Page
	Aliases    []Alias
	Tombstones []Tombstone
	Inventory  []InventoryDecision
}

var Current = Registry{
	Pages: []Page{
		{"detent.build guides", "Project contracts", "project-contracts.md", "/docs/site/project-contracts", OriginSite},
		{"Get started", "Quick Start", "getting-started.md", "/docs/getting-started", OriginUpstream},
		{"Get started", "Project Onboarding", "ONBOARDING.md", "/docs/project-onboarding", OriginUpstream},
		{"Get started", "Bootstrap a new machine", "bootstrap.md", "/docs/bootstrap", OriginUpstream},
		{"Get started", "Configuration", "config.md", "/docs/configuration", OriginUpstream},
		{"Operate Detent", "Concepts", "concepts.md", "/docs/concepts", OriginUpstream},
		{"Operate Detent", "Dependency workflows", "dependency-workflows.md", "/docs/dependency-workflows", OriginUpstream},
		{"Operate Detent", "Merge train", "merge-train.md", "/docs/merge-train", OriginUpstream},
		{"Operate Detent", "Multi-project operation", "multi-project.md", "/docs/multi-project", OriginUpstream},
		{"Operate Detent", "Machine-local workflow overlays", "workflow-overlays.md", "/docs/workflow-overlays", OriginUpstream},
		{"Operate Detent", "Webhook freshness", "webhook-freshness.md", "/docs/webhook-freshness", OriginUpstream},
		{"Operate Detent", "Scheduled operations", "scheduled-routines.md", "/docs/scheduled-operations", OriginUpstream},
		{"Operate Detent", "Admission criteria", "admission.md", "/docs/admission", OriginUpstream},
		{"Operate Detent", "Efficiency retrospection", "retrospection.md", "/docs/retrospection", OriginUpstream},
		{"Operate Detent", "Dashboard and APIs", "dashboard-api.md", "/docs/dashboard-api", OriginUpstream},
		{"Reference and contribute", "CLI reference", "cli.md", "/docs/cli", OriginUpstream},
		{"Reference and contribute", "Execution seams", "execution-seams.md", "/docs/execution-seams", OriginUpstream},
		{"Reference and contribute", "Local models", "local-models-ollama.md", "/docs/local-models", OriginUpstream},
	},
}
