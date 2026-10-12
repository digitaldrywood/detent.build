#!/usr/bin/env bash
# Runs only in release CI, never in a source worker. Promote the tested tree.
set -euo pipefail

: "${TESTED_SHA:?TESTED_SHA is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
[[ "$TESTED_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid tested SHA' >&2; exit 1; }

git fetch --no-tags origin develop main
git merge-base --is-ancestor "$TESTED_SHA" origin/develop
if git merge-base --is-ancestor "$TESTED_SHA" origin/main; then
  echo 'Tested develop commit is already in main; no promotion.'
  echo 'promoted=false' >> "$GITHUB_OUTPUT"
  exit 0
fi

git config user.name 'detent-release[bot]'
git config user.email 'detent-release[bot]@users.noreply.github.com'
git switch --detach origin/main
git merge --no-ff --no-edit -m "ci: promote tested develop $TESTED_SHA" "$TESTED_SHA"

# A main-only change or conflict resolution must never deploy an untested tree.
if ! git diff --quiet "$TESTED_SHA" HEAD --; then
  echo 'Promotion would deploy a tree different from the tested develop SHA.' >&2
  exit 1
fi

# A concurrent main update is rejected by the ordinary fast-forward push.
# Do not force, bypass branch protections, or include newer develop commits.
git push origin HEAD:refs/heads/main
echo 'promoted=true' >> "$GITHUB_OUTPUT"
echo "head=$(git rev-parse HEAD)" >> "$GITHUB_OUTPUT"
