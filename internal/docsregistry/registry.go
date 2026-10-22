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
		{"Get started", "Cloud onboarding", "cloud-onboarding.md", "/docs/cloud-onboarding", OriginUpstream},
		{"Get started", "Self-hosted Hub operations", "hub-self-hosting.md", "/docs/hub-self-hosting", OriginUpstream},
		{"Get started", "Sprite runners", "sprite-runners.md", "/docs/sprite-runners", OriginUpstream},
		{"Get started", "Hosted capacity allowances", "hosted-allowances.md", "/docs/hosted-allowances", OriginUpstream},
		{"Get started", "API & MCP setup", "api-mcp-setup.md", "/docs/api-mcp-setup", OriginUpstream},
		{"Get started", "Project delivery", "project-delivery.md", "/docs/project-delivery", OriginUpstream},
		{"Get started", "Local board migration", "hub-local-migration.md", "/docs/hub-local-migration", OriginUpstream},
		{"Get started", "Configuration", "config.md", "/docs/configuration", OriginUpstream},
		{"Operate Detent", "Concepts", "concepts.md", "/docs/concepts", OriginUpstream},
		{"Operate Detent", "Dependency workflows", "dependency-workflows.md", "/docs/dependency-workflows", OriginUpstream},
		{"Operate Detent", "Admission criteria", "admission.md", "/docs/admission", OriginUpstream},
		{"Operate Detent", "Runner capacity", "runner-capacity.md", "/docs/runner-capacity", OriginUpstream},
		{"Operate Detent", "Runner usage", "runner-usage.md", "/docs/runner-usage", OriginUpstream},
		{"Operate Detent", "Runner telemetry and APIs", "dashboard-api.md", "/docs/dashboard-api", OriginUpstream},
		{"Operate Detent", "Operations report", "operations.md", "/docs/operations", OriginUpstream},
		{"Operate Detent", "Native review", "native-review.md", "/docs/native-review", OriginUpstream},
		{"Operate Detent", "MCP capabilities", "mcp-capabilities.md", "/docs/mcp-capabilities", OriginUpstream},
		{"Operate Detent", "Hub API", "hub-api.md", "/docs/hub-api", OriginUpstream},
		{"Operate Detent", "Diagnosis", "diagnosis.md", "/docs/diagnosis", OriginUpstream},
		{"Operate Detent", "Webhook freshness", "webhook-freshness.md", "/docs/webhook-freshness", OriginUpstream},
		{"detent.build guides", "Project contracts", "project-contracts.md", "/docs/site/project-contracts", OriginSite},
		{"detent.build guides", "The working-checkout merge-gate trap", "working-checkout-merge-gate.md", "/docs/site/working-checkout-merge-gate", OriginSite},
		{"Reference and contribute", "CLI reference", "cli.md", "/docs/cli", OriginUpstream},
		{"Reference and contribute", "Local models", "local-models-ollama.md", "/docs/local-models", OriginUpstream},
		{"Reference and contribute", "Pi agent", "pi-agent.md", "/docs/pi-agent", OriginUpstream},
	},
	Aliases: []Alias{
		{"/docs/getting-started", "/docs/cloud-onboarding"},
	},
	Tombstones: []Tombstone{
		{"/docs/project-onboarding"},
		{"/docs/bootstrap"},
		{"/docs/merge-train"},
		{"/docs/multi-project"},
		{"/docs/workflow-overlays"},
		{"/docs/scheduled-operations"},
		{"/docs/retrospection"},
		{"/docs/execution-seams"},
	},
	Inventory: []InventoryDecision{
		{FromCommit: release057, ToCommit: release164, Kind: ChangeProbableRename, PreviousPath: "getting-started.md", CurrentPath: "cloud-onboarding.md", Resolution: ResolutionAlias, PublicPath: "/docs/getting-started", CanonicalPath: "/docs/cloud-onboarding"},
		published("api-mcp-setup.md", "/docs/api-mcp-setup"),
		unpublishedAddition("artifact-access-contract.md"),
		unpublishedAddition("artifacts-deployment.md"),
		unpublishedAddition("branching.md"),
		unpublishedAddition("cloud-attachments.md"),
		unpublishedAddition("cloud-domain-migration.md"),
		unpublishedAddition("cloud-hub-rfc.md"),
		unpublishedAddition("cloud-pilot-evidence.md"),
		unpublishedAddition("cloud-workflow-migration.md"),
		unpublishedAddition("conversation/configuration-map.md"),
		unpublishedAddition("conversation/decisions.md"),
		unpublishedAddition("conversation/design-inventory.md"),
		unpublishedAddition("conversation/operations.md"),
		unpublishedAddition("deployment-mcp-fixtures.md"),
		unpublishedAddition("design-system/README.md"),
		unpublishedAddition("design-system/accessibility.md"),
		unpublishedAddition("design-system/brand.md"),
		unpublishedAddition("design-system/color-review.md"),
		unpublishedAddition("design-system/components.md"),
		unpublishedAddition("design-system/contributing.md"),
		unpublishedAddition("design-system/element-catalog.md"),
		unpublishedAddition("design-system/foundations.md"),
		unpublishedAddition("design-system/images/button.jpg"),
		unpublishedAddition("design-system/images/composer.jpg"),
		unpublishedAddition("design-system/images/conversation-timeline.jpg"),
		unpublishedAddition("design-system/images/foundations-type.jpg"),
		unpublishedAddition("design-system/images/gallery-overview-dark.jpg"),
		unpublishedAddition("design-system/images/gallery-overview-light.jpg"),
		unpublishedAddition("design-system/images/menu-open.jpg"),
		unpublishedAddition("design-system/images/sidebar-390.jpg"),
		unpublishedAddition("design-system/images/work-board.jpg"),
		unpublishedAddition("design-system/patterns.md"),
		unpublishedAddition("design-system/workpad.md"),
		published("diagnosis.md", "/docs/diagnosis"),
		unpublishedAddition("examples/artifacts/Caddyfile"),
		unpublishedAddition("examples/artifacts/detent-artifacts.service"),
		unpublishedAddition("examples/artifacts/service.json"),
		unpublishedAddition("examples/cloud/connection.conf"),
		unpublishedAddition("examples/cloud/nginx-check.conf"),
		unpublishedAddition("examples/cloud/production.conf"),
		unpublishedAddition("examples/cloud/staging.conf"),
		unpublishedAddition("examples/detent-cloud-migration-workflow.json"),
		unpublishedAddition("examples/hub/Caddyfile"),
		unpublishedAddition("examples/hub/README.md"),
		unpublishedAddition("examples/hub/com.digitaldrywood.detent.hub.plist"),
		unpublishedAddition("examples/hub/detent-hub.service"),
		unpublishedAddition("github-profiles.md"),
		published("hosted-allowances.md", "/docs/hosted-allowances"),
		unpublishedAddition("hosted-billing.md"),
		unpublishedAddition("hosted-identity.md"),
		published("hub-api.md", "/docs/hub-api"),
		published("hub-local-migration.md", "/docs/hub-local-migration"),
		published("hub-self-hosting.md", "/docs/hub-self-hosting"),
		unpublishedAddition("instruction-activity-audit.md"),
		unpublishedAddition("instruction-cost.md"),
		unpublishedAddition("invariants.md"),
		unpublishedAddition("local-board-migration.md"),
		published("mcp-capabilities.md", "/docs/mcp-capabilities"),
		unpublishedAddition("monthly-budgets.md"),
		unpublishedAddition("native-implementation-review-rollout.md"),
		unpublishedAddition("native-issue-archive.md"),
		published("native-review.md", "/docs/native-review"),
		published("operations.md", "/docs/operations"),
		published("pi-agent.md", "/docs/pi-agent"),
		published("project-delivery.md", "/docs/project-delivery"),
		published("runner-capacity.md", "/docs/runner-capacity"),
		published("runner-usage.md", "/docs/runner-usage"),
		unpublishedAddition("scheduled-cloud-diagnostics.md"),
		published("sprite-runners.md", "/docs/sprite-runners"),
		unpublishedAddition("sprite-usage.md"),
		unpublishedAddition("sprites-cloud-runners-spike.md"),
		unpublishedAddition("superpowers/specs/2026-09-11-merge-lane-operator-invariants-design.md"),
		unpublishedAddition("templates/blocked-handoff.md"),
		unpublishedAddition("templates/detent-orchestration-workflow.patch"),
		unpublishedAddition("templates/onboarding-legacy-paragraphs.md"),
		unpublishedAddition("test-suite-audit.md"),
		unpublishedAddition("validation/test-suite-audit/codex-review-keep.tsv"),
		unpublishedAddition("validation/test-suite-audit/codex-verify-25.tsv"),
		unpublishedAddition("validation/test-suite-audit/deleted.tsv"),
		unpublishedAddition("validation/test-suite-audit/near-duplicates.tsv"),
		unpublishedAddition("workspace-retention.md"),
		tombstoned("ONBOARDING.md", "/docs/project-onboarding"),
		tombstoned("bootstrap.md", "/docs/bootstrap"),
		unpublishedDeletion("examples/non-code-artifact/README.md"),
		unpublishedDeletion("examples/non-code-artifact/WORKFLOW.md"),
		unpublishedDeletion("examples/non-code-artifact/output/package-approved-newsletter/manifest.json"),
		unpublishedDeletion("examples/non-code-artifact/seed/01-research-brief.json"),
		unpublishedDeletion("examples/non-code-artifact/seed/02-review-package.json"),
		tombstoned("execution-seams.md", "/docs/execution-seams"),
		tombstoned("merge-train.md", "/docs/merge-train"),
		tombstoned("multi-project.md", "/docs/multi-project"),
		unpublishedDeletion("parity-audit.md"),
		tombstoned("retrospection.md", "/docs/retrospection"),
		tombstoned("scheduled-routines.md", "/docs/scheduled-operations"),
		unpublishedDeletion("templates/WORKFLOW.github_local.md"),
		unpublishedDeletion("templates/WORKFLOW.non_code_artifact.md"),
		unpublishedDeletion("templates/detent.github_local.yaml"),
		unpublishedDeletion("templates/detent.non_code_artifact.yaml"),
		tombstoned("workflow-overlays.md", "/docs/workflow-overlays"),
	},
}

const (
	release057 = "1543929187369eca2703abd2a655cf86e9e5d83e"
	release164 = "13d15fec6a142058dd74d1275256bfa261442958"
)

func published(currentPath, publicPath string) InventoryDecision {
	return InventoryDecision{FromCommit: release057, ToCommit: release164, Kind: ChangeAdded, CurrentPath: currentPath, Resolution: ResolutionPublished, PublicPath: publicPath}
}

func unpublishedAddition(currentPath string) InventoryDecision {
	return InventoryDecision{FromCommit: release057, ToCommit: release164, Kind: ChangeAdded, CurrentPath: currentPath, Resolution: ResolutionUnpublished}
}

func tombstoned(previousPath, publicPath string) InventoryDecision {
	return InventoryDecision{FromCommit: release057, ToCommit: release164, Kind: ChangeDeleted, PreviousPath: previousPath, Resolution: ResolutionTombstone, PublicPath: publicPath}
}

func unpublishedDeletion(previousPath string) InventoryDecision {
	return InventoryDecision{FromCommit: release057, ToCommit: release164, Kind: ChangeDeleted, PreviousPath: previousPath, Resolution: ResolutionUnpublished}
}
