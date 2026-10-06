import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const script = resolve(import.meta.dirname, 'promote.sh');

function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), 'promotion-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const remote = join(root, 'remote.git');
  const cwd = join(root, 'checkout');
  const run = (args, path = cwd) => execFileSync('git', args, { cwd: path, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
  run(['init', '--bare', remote], root);
  run(['clone', remote, cwd], root);
  run(['config', 'user.name', 'CI test']);
  run(['config', 'user.email', 'ci@example.invalid']);
  run(['config', 'commit.gpgsign', 'false']);
  const commit = (name) => {
    writeFileSync(join(cwd, name), name);
    run(['add', name]);
    run(['commit', '-m', name]);
    return run(['rev-parse', 'HEAD']);
  };
  commit('base');
  run(['branch', '-M', 'main']);
  run(['push', 'origin', 'main']);
  run(['switch', '-c', 'develop']);
  run(['push', 'origin', 'develop']);
  const output = join(root, 'output');
  return {
    run, commit, remote,
    promote: sha => spawnSync('bash', [script], {
      cwd, encoding: 'utf8',
      env: { ...process.env, TESTED_SHA: sha, GITHUB_OUTPUT: output },
    }),
    output: () => readFileSync(output, 'utf8'),
  };
}

test('green tested SHA is promoted, but later develop commits are excluded; rerun skips', t => {
  const f = fixture(t);
  const tested = f.commit('tested');
  f.commit('untested');
  f.run(['push', 'origin', 'develop']);
  const result = f.promote(tested);
  assert.equal(result.status, 0, result.stderr);
  assert.match(f.output(), /promoted=true/);
  const main = f.run(['rev-parse', 'main'], f.remote);
  assert.equal(f.run(['rev-parse', `${main}^{tree}`]), f.run(['rev-parse', `${tested}^{tree}`]));
  assert.equal(f.run(['rev-list', '--parents', '-1', main]).split(' ').length, 3);
  assert.equal(f.promote(tested).status, 0);
  assert.match(f.output(), /promoted=false/);
});

test('main-only changes refuse promotion and leave remote main unchanged', t => {
  const f = fixture(t);
  const tested = f.commit('tested');
  f.run(['push', 'origin', 'develop']);
  f.run(['switch', 'main']);
  const original = f.commit('main-only');
  f.run(['push', 'origin', 'main']);
  const result = f.promote(tested);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /different from the tested/);
  assert.equal(f.run(['rev-parse', 'main'], f.remote), original);
});

test('a previous promotion merge permits the next tested develop tree', t => {
  const f = fixture(t);
  const first = f.commit('first');
  f.run(['push', 'origin', 'develop']);
  assert.equal(f.promote(first).status, 0);
  f.run(['switch', 'develop']);
  const next = f.commit('next');
  f.run(['push', 'origin', 'develop']);
  assert.equal(f.promote(next).status, 0);
  const main = f.run(['rev-parse', 'main'], f.remote);
  assert.equal(f.run(['rev-parse', `${main}^{tree}`]), f.run(['rev-parse', `${next}^{tree}`]));
});

test('a SHA outside develop refuses promotion', t => {
  const f = fixture(t);
  const unpushed = f.commit('not-in-develop');
  const original = f.run(['rev-parse', 'main'], f.remote);
  assert.notEqual(f.promote(unpushed).status, 0);
  assert.equal(f.run(['rev-parse', 'main'], f.remote), original);
});
