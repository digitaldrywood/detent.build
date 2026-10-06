import { test } from 'node:test';
import assert from 'node:assert/strict';
import { reportFailures } from './report.mjs';

const context = { repo: { owner: 'example', repo: 'site' }, runId: 42 };
const env = {
  INTAKE_URL: 'https://native-intake.example/ci', INTAKE_TOKEN: 'test-secret',
  GITHUB_RUN_ATTEMPT: '2', GITHUB_SERVER_URL: 'https://github.com',
  GITHUB_REPOSITORY: 'example/site', TESTED_SHA: 'a'.repeat(40),
};
function fixture() {
  const jobs = ['check', 'browser', 'smoke'].map((name, id) => ({
    id, name, conclusion: name === 'smoke' ? 'skipped' : 'failure',
    html_url: `https://github.com/example/site/actions/runs/42/job/${id}`,
  }));
  const deliveries = [];
  const github = {
    paginate: async (_, args) => {
      assert.equal(args.attempt_number, 2);
      return jobs;
    },
    rest: { actions: {
      listJobsForWorkflowRunAttempt: {},
      downloadJobLogsForWorkflowRun: async ({ job_id }) => ({ data: `job ${job_id}: expected 200, got 500` }),
    } },
  };
  const send = async (url, request) => {
    assert.equal(url, env.INTAKE_URL);
    assert.equal(request.headers.Authorization, 'Bearer test-secret');
    assert.equal(request.redirect, 'error');
    deliveries.push(JSON.parse(request.body));
    return { ok: true, json: async () => ({ work_item_id: 'wi_test', action: 'created' }) };
  };
  return { github, send, deliveries, jobs };
}

test('reports each failing job with Todo/High, SHA, run URL, and actual output', async () => {
  const f = fixture();
  await reportFailures({ ...f, context, env });
  assert.equal(f.deliveries.length, 2);
  for (const body of f.deliveries) {
    assert.equal(body.project_id, 'prj_e7aab744eb84410fafd437f478992bd1');
    assert.equal(body.state, 'Todo');
    assert.equal(body.priority, 2);
    assert.match(body.details, /expected 200, got 500/);
    assert.match(body.details, /runs\/42\/attempts\/2/);
    assert.ok(body.details.includes(env.TESTED_SHA));
  }
});

test('repeat deliveries keep the fingerprint and delivery id, accept comment receipts', async () => {
  const f = fixture();
  const send = async (...args) => {
    const response = await f.send(...args);
    return { ...response, json: async () => ({ work_item_id: 'wi_existing', action: 'commented' }) };
  };
  await reportFailures({ ...f, send, context, env });
  await reportFailures({ ...f, send, context, env });
  assert.equal(f.deliveries[0].fingerprint, f.deliveries[2].fingerprint);
  assert.equal(f.deliveries[0].delivery_id, f.deliveries[2].delivery_id);
});

test('a delivery failure still attempts the other job and fails the reporter', async () => {
  const f = fixture();
  let calls = 0;
  const send = async (...args) => ++calls === 1 ? { ok: false, status: 503 } : f.send(...args);
  await assert.rejects(reportFailures({ ...f, send, context, env }), /check: Native intake returned HTTP 503/);
  assert.equal(calls, 2);
});

test('unavailable logs still report the failed job and keep missing evidence visible', async () => {
  const f = fixture();
  f.jobs[0].conclusion = 'startup_failure';
  f.jobs[0].steps = [{ name: 'Set up job', conclusion: 'failure' }];
  f.github.rest.actions.downloadJobLogsForWorkflowRun = async ({ job_id }) => {
    if (job_id === 0) throw new Error('HTTP 404');
    return { data: 'browser failed' };
  };
  await assert.rejects(reportFailures({ ...f, context, env }), /check: Job logs unavailable: HTTP 404/);
  assert.equal(f.deliveries.length, 2);
  assert.match(f.deliveries[0].details, /Job logs unavailable: HTTP 404/);
  assert.match(f.deliveries[0].details, /Set up job: failure/);
  assert.match(f.deliveries[1].details, /browser failed/);
});

test('missing credentials and unacknowledged HTML responses fail visibly', async () => {
  const f = fixture();
  await assert.rejects(reportFailures({ ...f, context, env: { ...env, INTAKE_TOKEN: '' } }), /not configured/);
  await assert.rejects(reportFailures({ ...f, context, env,
    send: async () => ({ ok: true, json: async () => ({}) }),
  }), /did not acknowledge/);
});
