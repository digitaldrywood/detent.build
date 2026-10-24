import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { auditRequest, auditResult, deployedCommit, project, controllerRevision } from './release-audit.mjs';

const sha = 'a'.repeat(40);
const env = {
  GITHUB_REPOSITORY: 'digitaldrywood/detent.build', GITHUB_WORKSPACE: '/source',
  GITHUB_RUN_ID: '42', GITHUB_RUN_ATTEMPT: '2', GITHUB_SERVER_URL: 'https://github.com',
  CI_RELEASE_BRANCH: 'release/v0.1', DETENT_RELEASE_AUDIT_MODEL: 'project-selected-model',
};
const request = auditRequest(env, sha);
function execution(status, exit = 0, stderr = '') {
  return { status: exit, stderr, stdout: `native reporting receipt\n${JSON.stringify({ identity: request.identity, status, reason: status === 'incomplete' ? 'allowance_exhausted' : '', infrastructure: [] })}\n` };
}

test('deployment attestation waits for the exact expected SHA, never accepts a healthy old deployment', async () => {
  let calls = 0;
  const commit = await deployedCommit({ expected: sha, pause: async () => {}, send: async (url, options) => {
    assert.equal(url, 'https://detent.build/deployment');
    assert.equal(options.redirect, 'error');
    assert.equal(options.cache, 'no-store');
    return { ok: true, json: async () => ({ commit: ++calls === 1 ? 'b'.repeat(40) : sha }) };
  } });
  assert.equal(commit, sha);
  assert.equal(calls, 2);
  await assert.rejects(deployedCommit({ expected: sha, pause: async () => {}, send: async () => ({ ok: false }) }), /did not attest/);
  await assert.rejects(deployedCommit({ expected: sha, send: async () => ({ ok: true, json: async () => ({ commit: 'short' }) }) }), /full commit/);
  await assert.rejects(deployedCommit({ expected: 'develop' }), /full SHA/);
});

test('request pins this native project, deployed commit, configured model, run and explicit allowances', () => {
  assert.equal(request.identity.project_id, project);
  assert.equal(request.identity.commit, sha);
  assert.match(request.identity.run_url, /42\/attempts\/2$/);
  assert.equal(request.model, env.DETENT_RELEASE_AUDIT_MODEL);
  assert.equal(request.allowance.invocations, 40);
  for (const invalid of [{ DETENT_RELEASE_AUDIT_MODEL: '' }, { CI_RELEASE_BRANCH: 'develop' }, { GITHUB_REPOSITORY: 'other/repo' }]) {
    assert.throws(() => auditRequest({ ...env, ...invalid }, sha), /requires/);
  }
});

test('accepts findings and needs_validation statuses; explicit incomplete results remain nonblocking', () => {
  for (const status of ['clean', 'findings', 'needs_validation']) {
    assert.equal(auditResult(request, execution(status)).status, status);
  }
  for (const diagnostic of ['release audit is incomplete; retained state is not a clean audit', 'release audit allowance exhausted', 'release audit allowance exhausted\ncontext canceled']) {
    assert.equal(auditResult(request, execution('incomplete', 1, diagnostic)).status, 'incomplete');
  }
});

test('publication, infrastructure, crashes, missing output and cross-commit results fail visibly', () => {
  const wrongCommit = execution('clean');
  wrongCommit.stdout = wrongCommit.stdout.replace(sha, 'b'.repeat(40));
  const infrastructure = execution('incomplete', 1, 'release audit allowance exhausted');
  infrastructure.stdout = infrastructure.stdout.replace('"infrastructure":[]', '"infrastructure":["failed to publish"]');
  for (const result of [wrongCommit, infrastructure, execution('failed', 1),
    execution('failed', 1, 'release audit allowance exhausted\nfailed final publication'),
    execution('clean', 1, 'reporting failed'), { ...execution('incomplete'), signal: 'SIGTERM' },
    { ...execution('incomplete'), error: new Error('timeout') }, { status: 1, stdout: '', stderr: 'startup failed' }]) {
    assert.throws(() => auditResult(request, result), /infrastructure/);
  }
});

test('workflow stays release-only, audits after deployment, pins controller and retains evidence', () => {
  const workflow = readFileSync(new URL('../../.github/workflows/ci.yml', import.meta.url), 'utf8');
  const triggers = workflow.split('permissions:')[0];
  assert.match(triggers, /schedule:/);
  assert.match(triggers, /workflow_dispatch:/);
  assert.match(workflow, /node scripts\/ci\/test-release-audit-integration\.mjs/);
  assert.doesNotMatch(triggers, /pull_request:|push:/);
  const audit = workflow.split('  security-audit:')[1].split('  report:')[0];
  assert.match(audit, /needs: \[snapshot, promote, smoke\]/);
  assert.ok(audit.includes(`ref: ${controllerRevision}`));
  assert.match(audit, /DETENT_API_KEY: \$\{\{ secrets.DETENT_API_KEY \}\}/);
  assert.match(audit, /retention-days: 30/);
  assert.match(workflow, /needs: \[snapshot, check, browser, promote, smoke, security-audit\]/);
  // The deployment contract stays plain HTTP on 3000, apex HTTPS, Go 1.27.
  const config = readFileSync(new URL('../../internal/config/config.go', import.meta.url), 'utf8');
  const nixpacks = readFileSync(new URL('../../nixpacks.toml', import.meta.url), 'utf8');
  const deploy = readFileSync(new URL('../../docs/deploy.md', import.meta.url), 'utf8');
  assert.match(config, /getEnvOrDefault\("PORT", "3000"\)/);
  assert.match(deploy, /SITE_URL=https:\/\/detent\.build\n/);
  assert.match(deploy, /branch `main`/);
  assert.match(nixpacks, /"go_1_27", "nodejs_22"/);
  assert.match(nixpacks, /-buildvcs=true/);
  assert.match(nixpacks, /cmd = "\.\/out"/);
  assert.doesNotMatch(audit, /application\.deploy|SITE_URL:|PORT:/);
});

test('provider wrapper authenticates each scratch profile before starting app-server', t => {
  const root = mkdtempSync(join(tmpdir(), 'audit-provider-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const bin = join(root, 'audit-provider/node_modules/.bin');
  mkdirSync(bin, { recursive: true });
  writeFileSync(join(bin, 'codex'), `#!/bin/bash\nset -eu\nif [[ "$*" == *login* ]]; then\n  test "$(cat)" = fixture-key\n  echo login >> "$RUNNER_TEMP/order"\nelse\n  test "$*" = '-c cli_auth_credentials_store="file" app-server --listen stdio://'\n  echo server >> "$RUNNER_TEMP/order"\nfi\n`, { mode: 0o755 });
  const script = resolve(import.meta.dirname, 'audit-codex.sh');
  const result = spawnSync('bash', [script, 'app-server', '--listen', 'stdio://'], {
    encoding: 'utf8', env: { ...process.env, RUNNER_TEMP: root, CODEX_HOME: join(root, 'session/codex'), OPENAI_API_KEY: 'fixture-key' },
  });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(readFileSync(join(root, 'order'), 'utf8'), 'login\nserver\n');
  assert.doesNotMatch(result.stdout + result.stderr, /fixture-key/);
});

test('CLI wrapper passes deployed identity and retains results with a synthetic controller', t => {
  const root = mkdtempSync(join(tmpdir(), 'release-audit-cli-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const evidence = join(root, 'release-audit-evidence');
  mkdirSync(evidence);
  writeFileSync(join(evidence, 'deployment.json'), JSON.stringify({ commit: sha }));
  writeFileSync(join(root, 'cifailure'), `#!/usr/bin/env node
import { readFileSync, writeFileSync } from 'node:fs';
import assert from 'node:assert/strict';
const request = JSON.parse(readFileSync(0, 'utf8'));
assert.equal(process.env.DETENT_RELEASE_AUDIT, 'true');
assert.equal(request.identity.project_id, process.env.DETENT_PROJECT_ID);
assert.equal(request.identity.commit, process.env.CI_DEVELOP_SHA);
writeFileSync(process.env.RUNNER_TEMP + '/captured.json', JSON.stringify(request));
const status = process.env.FIXTURE_STATUS;
console.log(JSON.stringify({ identity: request.identity, status, reason: status === 'incomplete' ? 'coverage_incomplete' : '', infrastructure: status === 'failed' ? ['publication failed'] : [] }));
process.exitCode = ['incomplete', 'failed'].includes(status) ? 1 : 0;
`, { mode: 0o755 });
  // Node uses the extensionless executable's package type; make it explicit.
  writeFileSync(join(root, 'package.json'), '{"type":"module"}');
  for (const status of ['findings', 'needs_validation', 'incomplete', 'failed']) {
    const result = spawnSync(process.execPath, [resolve(import.meta.dirname, 'release-audit.mjs'), 'run'], {
      encoding: 'utf8', env: { ...process.env, ...env, RUNNER_TEMP: root, GITHUB_STEP_SUMMARY: join(root, 'summary'), FIXTURE_STATUS: status },
    });
    assert.equal(result.status, status === 'failed' ? 1 : 0, result.stderr);
    assert.equal(JSON.parse(readFileSync(join(root, 'captured.json'))).identity.commit, sha);
    assert.match(readFileSync(join(evidence, 'controller.log'), 'utf8'), new RegExp(status));
    if (status !== 'failed') assert.equal(JSON.parse(readFileSync(join(evidence, 'result.json'))).status, status);
    else assert.match(result.stderr, /Audit infrastructure/);
  }
});
