import { execFileSync, spawnSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { isAbsolute, join } from 'node:path';
import { pathToFileURL } from 'node:url';

export const project = 'prj_e7aab744eb84410fafd437f478992bd1';
export const controllerRevision = '30d8a2c26b9539b59013b8428f7dd033529b50af';
const shaPattern = /^[a-f0-9]{40}$/;

export async function deployedCommit({ expected, send = fetch, pause = ms => new Promise(resolve => setTimeout(resolve, ms)) }) {
  if (!shaPattern.test(expected)) throw new Error('Expected deployment must be a full SHA');
  for (let attempt = 0; attempt < 15; attempt++) {
    const response = await send('https://detent.build/deployment', {
      redirect: 'error', cache: 'no-store', signal: AbortSignal.timeout(15000),
    });
    if (response.ok) {
      const body = await response.json();
      if (!shaPattern.test(body.commit)) throw new Error('Deployment did not attest a full commit');
      if (body.commit === expected) return body.commit;
    }
    if (attempt < 14) await pause(30000);
  }
  throw new Error('Production did not attest the expected deployed commit');
}

export function auditRequest(env, commit) {
  if (!shaPattern.test(commit) || env.GITHUB_REPOSITORY !== 'digitaldrywood/detent.build' ||
      !/^release\/v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/.test(env.CI_RELEASE_BRANCH || '') ||
      !env.DETENT_RELEASE_AUDIT_MODEL?.trim() || !env.GITHUB_RUN_ID || !env.GITHUB_RUN_ATTEMPT ||
      !isAbsolute(env.GITHUB_WORKSPACE || '')) {
    throw new Error('Release audit requires pinned source, configured series/model, and scheduled run identity');
  }
  return {
    identity: {
      project_id: project, repository: env.GITHUB_REPOSITORY, commit,
      release_series: env.CI_RELEASE_BRANCH, run_id: env.GITHUB_RUN_ID,
      attempt: env.GITHUB_RUN_ATTEMPT, job_id: 'security-audit',
      run_url: `${env.GITHUB_SERVER_URL}/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}/attempts/${env.GITHUB_RUN_ATTEMPT}`,
    },
    repository_directory: env.GITHUB_WORKSPACE,
    model: env.DETENT_RELEASE_AUDIT_MODEL,
    allowance: { invocations: 40, usage: { input_tokens: 200000, output_tokens: 50000, cached_input_tokens: 150000 } },
  };
}

// The pinned controller prints native reporting receipts before its final JSON.
// Incomplete coverage is nonblocking only with a final retained result. Failed
// execution/publication has status failed; a killed process has no valid exit.
export function auditResult(request, execution) {
  const last = (execution.stdout || '').trim().split('\n').at(-1);
  let result;
  try { result = JSON.parse(last); } catch { throw new Error('Audit infrastructure: missing final controller result'); }
  if (!result?.identity || Object.keys(result.identity).length !== Object.keys(request.identity).length ||
      !Object.entries(request.identity).every(([key, value]) => result.identity[key] === value) ||
      !Array.isArray(result.infrastructure) || result.infrastructure.length ||
      !['clean', 'findings', 'needs_validation', 'incomplete'].includes(result.status)) {
    throw new Error('Audit infrastructure: invalid identity, status, or infrastructure result');
  }
  const incomplete = result.status === 'incomplete' &&
    ['allowance_exhausted', 'coverage_incomplete'].includes(result.reason);
  if (execution.error || execution.signal || (execution.status !== 0 &&
      !(execution.status === 1 && incomplete))) {
    throw new Error('Audit infrastructure: controller execution or publication failed');
  }
  return result;
}

async function main(env) {
  const directory = join(env.RUNNER_TEMP, 'release-audit-evidence');
  mkdirSync(directory, { recursive: true });
  if (process.argv[2] === 'pin') {
    const expected = env.EXPECTED_DEPLOYED_SHA || execFileSync('git', ['rev-parse', 'origin/main'], { encoding: 'utf8' }).trim();
    const sha = await deployedCommit({ expected });
    execFileSync('git', ['merge-base', '--is-ancestor', sha, 'origin/main']);
    writeFileSync(join(directory, 'deployment.json'), JSON.stringify({ commit: sha, url: 'https://detent.build/deployment', controller: controllerRevision }));
    writeFileSync(env.GITHUB_OUTPUT, `sha=${sha}\n`, { flag: 'a' });
    console.log(`Attested deployed commit: ${sha}\nController: ${controllerRevision}`);
    return;
  }
  const commit = JSON.parse(readFileSync(join(directory, 'deployment.json'))).commit;
  const request = auditRequest(env, commit);
  writeFileSync(join(directory, 'request.json'), JSON.stringify(request));
  const execution = spawnSync(join(env.RUNNER_TEMP, 'cifailure'), [], {
    input: JSON.stringify(request), encoding: 'utf8', maxBuffer: 64 * 1024 * 1024,
    timeout: 55 * 60 * 1000,
    cwd: env.RUNNER_TEMP,
    env: { ...env, DETENT_RELEASE_AUDIT: 'true', DETENT_PROJECT_ID: project, CI_DEVELOP_SHA: commit },
  });
  writeFileSync(join(directory, 'controller.log'), execution.stdout || '');
  writeFileSync(join(directory, 'controller-error.log'), execution.stderr || '');
  const result = auditResult(request, execution);
  writeFileSync(join(directory, 'result.json'), JSON.stringify(result));
  const summary = `Release security audit: ${result.status}\nDeployed SHA: ${commit}\nController: ${controllerRevision}\nIncomplete and needs_validation records do not assert clean coverage.\n`;
  writeFileSync(env.GITHUB_STEP_SUMMARY, summary, { flag: 'a' });
  console.log(summary);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main(process.env).catch(error => { console.error(`Audit infrastructure: ${error.message}`); process.exitCode = 1; });
}
