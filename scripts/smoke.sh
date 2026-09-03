#!/usr/bin/env bash
#
# Post-deploy smoke check. Asserts the invariants that only a real deployment
# can break — the ones the unit tests cannot see because they depend on the
# environment the container was actually given.
#
#   ./scripts/smoke.sh                      # defaults to https://detent.build
#   ./scripts/smoke.sh http://localhost:3000
#
# The first production deploy shipped with no environment set, so ENV fell back
# to development and every canonical pointed at localhost. Everything else was
# green: CI passed, all routes returned 200, the container was healthy. This
# script exists because none of those signals could see it.

set -uo pipefail

BASE="${1:-https://detent.build}"
BASE="${BASE%/}"
FAILED=0

pass() { printf '  \033[32mok\033[0m   %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m %s\n' "$1"; FAILED=1; }

get()      { curl -sS --max-time 20 "$BASE$1"; }
status()   { curl -sS -o /dev/null -w '%{http_code}' --max-time 20 "$BASE$1"; }

echo "smoke: $BASE"

echo "routes"
for path in / /how-it-works /why-detent /dashboard /install /install/macos \
            /install/linux /install/windows /install/source /open-source \
            /health /robots.txt /sitemap.xml; do
  code=$(status "$path")
  [ "$code" = "200" ] && pass "$path -> 200" || fail "$path -> $code, want 200"
done

code=$(status /this-page-does-not-exist)
[ "$code" = "404" ] && pass "unknown path -> 404" || fail "unknown path -> $code, want 404"

echo "canonical host"
home=$(get /)
hiw=$(get /how-it-works)

# detent.build has no www record, and .build is HSTS-preloaded, so neither a
# www host nor a plain-http absolute URL for it is reachable.
if grep -q 'www\.detent\.build' <<<"$home$hiw"; then
  fail "a page references www.detent.build, which has no DNS record"
else
  pass "no www host referenced"
fi

if grep -q 'http://detent\.build' <<<"$home$hiw"; then
  fail "a page emits a plain-http URL for an HSTS-preloaded host"
else
  pass "no plain-http production URL"
fi

# The failure that motivated this script: SITE_URL unset, so every canonical
# and og:url pointed at localhost while the site otherwise looked healthy.
if grep -qE 'localhost|127\.0\.0\.1' <<<"$home$hiw"; then
  fail "a page references localhost — SITE_URL is probably unset in the deployment"
else
  pass "no localhost references"
fi

if grep -q '<link rel="canonical" href="'"$BASE"'/how-it-works">' <<<"$hiw"; then
  pass "canonical matches the deployed host"
else
  fail "canonical does not match $BASE: $(grep -o '<link rel="canonical"[^>]*>' <<<"$hiw" | head -1)"
fi

if grep -q '<meta property="og:image" content="'"$BASE"'/static/images/og-default.png">' <<<"$hiw"; then
  pass "og:image is absolute and on the deployed host"
else
  fail "og:image is missing or not absolute"
fi

echo "sitemap"
sitemap=$(get /sitemap.xml)
if grep -q "<loc>$BASE/</loc>" <<<"$sitemap"; then
  pass "sitemap uses the deployed host"
else
  fail "sitemap does not use $BASE"
fi
if grep -q '//</loc>' <<<"$sitemap"; then
  fail "sitemap has a doubled slash"
else
  pass "no doubled slashes"
fi

echo "transport"
if [[ "$BASE" == https://* ]]; then
  redirect=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 20 "${BASE/https:/http:}/")
  if [ "$redirect" = "301" ] || [ "$redirect" = "308" ]; then
    pass "http redirects to https ($redirect)"
  else
    fail "http returned $redirect, want a 301/308 from the proxy"
  fi

  headers=$(curl -sSI --max-time 20 "$BASE/")
  for h in "content-security-policy" "strict-transport-security" \
           "x-content-type-options" "referrer-policy"; do
    grep -qi "^$h:" <<<"$headers" && pass "$h present" || fail "$h missing"
  done
fi

echo
if [ "$FAILED" -eq 0 ]; then
  echo "smoke: all checks passed"
else
  echo "smoke: FAILURES above"
fi
exit "$FAILED"
