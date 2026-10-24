#!/usr/bin/env bash
# The controller creates a fresh CODEX_HOME under job scratch for every session.
set -euo pipefail
: "${OPENAI_API_KEY:?audit provider key is required}"
: "${CODEX_HOME:?controller-owned provider profile is required}"
: "${RUNNER_TEMP:?job scratch is required}"
[[ "$CODEX_HOME" == "$RUNNER_TEMP/"* ]] || { echo 'Provider profile is outside job scratch' >&2; exit 1; }
audit_cli="$RUNNER_TEMP/audit-provider/node_modules/.bin/codex"
if ! printf '%s' "$OPENAI_API_KEY" | "$audit_cli" -c 'cli_auth_credentials_store="file"' login --with-api-key >/dev/null 2>&1; then
  echo 'Audit provider authentication failed' >&2
  exit 1
fi
exec "$audit_cli" -c 'cli_auth_credentials_store="file"' "$@"
