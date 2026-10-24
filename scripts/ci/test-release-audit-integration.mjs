// Exercise the reviewed controller's real stage, validators, REST transport and
// reporter. Only provider responses, sandbox observations and native storage are
// fixtures. No provider credential or external native write is used.
import { execFileSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { auditRequest, auditResult, controllerRevision } from './release-audit.mjs';

const root = mkdtempSync(join(tmpdir(), 'detent-build-audit-integration-'));
const site = resolve(import.meta.dirname, '../..');
const controller = join(root, 'controller');
const run = (command, args, cwd) => execFileSync(command, args, { cwd, stdio: 'inherit' });
try {
  mkdirSync(controller);
  // Fetch the exact workflow pin, never the moving upstream branch.
  run('git', ['init', '--quiet'], controller);
  run('git', ['fetch', '--quiet', '--depth=1', 'https://github.com/digitaldrywood/detent.git', controllerRevision], controller);
  run('git', ['checkout', '--quiet', '--detach', 'FETCH_HEAD'], controller);
  const revision = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: controller, encoding: 'utf8' }).trim();
  if (revision !== controllerRevision) throw new Error('Controller revision differs from the release workflow pin');
  console.log(`Controller: ${revision}`);
  console.log('Controller production sources: unchanged; adding one integration test in job scratch.');
  const fixture = new URL('./fixtures/release-audit-integration.go.txt', import.meta.url);
  console.log(`Integration fixture SHA-256: ${createHash('sha256').update(readFileSync(fixture)).digest('hex')}`);
  copyFileSync(fixture, join(controller, 'tools/cifailure/detent_build_integration_test.go'));
  run('git', ['diff', '--exit-code', controllerRevision, '--'], controller);

  const commit = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: site, encoding: 'utf8' }).trim();
  const request = auditRequest({
    GITHUB_REPOSITORY: 'digitaldrywood/detent.build', GITHUB_WORKSPACE: site,
    GITHUB_RUN_ID: '42', GITHUB_RUN_ATTEMPT: '1', GITHUB_SERVER_URL: 'https://github.com',
    CI_RELEASE_BRANCH: 'release/v0.1', DETENT_RELEASE_AUDIT_MODEL: 'controlled-provider-fixture',
  }, commit);
  const requestPath = join(root, 'request.json');
  const resultsPath = join(root, 'stage-results.json');
  writeFileSync(requestPath, JSON.stringify(request));
  console.log(`Site request: project=${request.identity.project_id} commit=${commit}`);
  console.log('Command: GOTOOLCHAIN=go1.26.10 go test -count=1 -timeout=120s -run ^TestDetentBuildReleaseAudit -v ./tools/cifailure');
  execFileSync('go', ['test', '-count=1', '-timeout=120s', '-run', '^TestDetentBuildReleaseAudit', '-v', './tools/cifailure'], {
    cwd: controller, stdio: 'inherit',
    env: { ...process.env, GOTOOLCHAIN: 'go1.26.10', DETENT_BUILD_AUDIT_REQUEST: requestPath, DETENT_BUILD_AUDIT_RESULTS: resultsPath },
  });
  for (const outcome of JSON.parse(readFileSync(resultsPath))) {
    const execution = { status: outcome.exit_code, stdout: JSON.stringify(outcome.result), stderr: outcome.error };
    const request = { identity: outcome.result.identity };
    if (outcome.result.status === 'failed') assert.throws(() => auditResult(request, execution), /infrastructure/);
    else assert.equal(auditResult(request, execution).status, outcome.result.status);
    console.log(`Site result handler: ${outcome.name} => ${outcome.result.status === 'failed' ? 'infrastructure failure' : `nonblocking ${outcome.result.status}`}`);
  }
} finally {
  rmSync(root, { recursive: true, force: true });
}
