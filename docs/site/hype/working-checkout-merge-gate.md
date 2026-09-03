# The working-checkout merge-gate trap

> Verified against Detent [v0.57.0 at `154392918736`](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/README.md).

Detent can take a pull request through successful CI and still refuse to promote it because the merge gate is asking for a check that no longer exists. The policy may not come from the branch under test.

## The mechanism

There are two configuration layers to keep straight:

- The project's entry in Detent's global configuration selects `projects[].workflow`, `projects[].workdir`, and optionally `projects[].workflow_ref`.
- That workflow's sibling `detent.yaml` contains `workspace.source_root` and the merge-gate policy at `gate.required_status_checks`.

When `projects[].workflow_ref` is empty, Detent loads the workflow from its configured path in the working tree. The project-definition loader then reads `detent.yaml` beside that workflow. A different branch in that checkout can therefore change the effective gate for every issue, regardless of the branch in an issue's isolated worktree. [Source: working-tree workflow selection](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/project/workflow_source.go#L32-L45) and [sibling `detent.yaml` loading](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/config/project_definition.go#L74-L110).

The configured working checkout for this site is explicit:

<code src="detent.yaml" snippet="workspace-checkout"></code>

Its gate is also explicit:

<code src="detent.yaml" snippet="merge-gate"></code>

This site sets `required_status_checks: []`, so it does **not** reproduce the failure. In a project with named required checks, Detent treats an absent name as `missing`, and missing, skipped, failed, cancelled, neutral, or running checks block promotion. [Source: required-check evaluation](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/connector/github/pull_request_checks.go#L268-L310) and [gate contract](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/gate/gate.go#L451-L456).

## What it looks like

The operational symptom is a pull request whose actual CI is green while Detent keeps withholding promotion because the loaded policy names a retired check. The issue, pull request, and board describe the work under test; none of them inherently tells you which branch supplies the daemon's project definition.

On versions before the current safeguards, that mismatch could send work around the gate repeatedly with no useful board-level explanation. At the pinned `v0.57.0`, Detent records consecutive missing-check evaluations and blocks after three rather than cycling forever. [Source: persistent missing-check threshold](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/orchestrator/merge_required_checks.go#L16-L18) and [streak evaluation](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/orchestrator/merge_required_checks.go#L27-L61).

## Confirm it

Run the read-only doctor from the Detent host:

```sh
detent doctor --port 0
```

Find `Project <id> workflow source policy` in the output. With no `workflow_ref`, `v0.57.0` reports that the project reads merge policy from a mutable working tree, names the checked-out branch, and compares the effective `detent.yaml` with the repository's default branch. A branch mismatch or different file is a failure; even a matching default branch remains a warning because it is mutable. [Source: doctor mutable-source diagnostic](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/cli/doctor_workflow_source.go#L146-L234).

Also inspect the two facts doctor is testing:

```sh
git -C <projects[].workdir> branch --show-current
git -C <projects[].workdir> diff origin/<default-branch> -- detent.yaml
```

Then compare `gate.required_status_checks` with the check names on the current pull-request head. Do not infer a renamed or retired check from a green aggregate result; the configured names are exact gate inputs.

## Fix it

For immediate recovery, put the configured working checkout on the intended default branch, update it, and make `gate.required_status_checks` match the checks the repository actually emits. Rerun `detent doctor --port 0` before returning the item to the merge path.

The durable prevention is to set `projects[].workflow_ref` to a remote-tracking ref such as `origin/main` in Detent's global configuration after `WORKFLOW.md` and `detent.yaml` exist at that ref. Detent then reads both shared files from that commit and ignores the working-tree branch; local overlays remain working-tree inputs. [Source: ref-backed project definitions](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/project/workflow_source.go#L107-L171).

Doctor verifies that the configured remote-tracking ref is fresh. It checks the remote without fetching, so fetch the ref when doctor reports staleness, then rerun doctor until the policy check says the ref is fresh and ignores the working-tree branch. [Source: workflow-ref freshness diagnostic](https://github.com/digitaldrywood/detent/blob/1543929187369eca2703abd2a655cf86e9e5d83e/internal/cli/doctor_workflow_source.go#L111-L143).
