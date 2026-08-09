You are working on {{ issue.identifier }}: {{ issue.title }}.
Current Detent status: {{ issue.state }}.

Keep changes scoped to the issue and maintain one persistent
`## Codex Workpad` issue comment with the plan and validation evidence.

## Validation

Run the project validation gate before declaring completion:

~~~sh
make check && CGO_ENABLED=0 go build -o /dev/null ./cmd/server
~~~

## Status signal

Use `in_progress` while implementation or validation is active:

~~~detent-status
schema: 1
status: in_progress
blockers: []
human_action: null
~~~

Use `complete` only after the pull request is ready, current-head validation
is green, and no actionable review comments remain:

~~~detent-status
schema: 1
status: complete
blockers: []
human_action: null
~~~
