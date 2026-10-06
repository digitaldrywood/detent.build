// A provisioned native Cloud intake owns issue creation, deduplication, and comments.
// The URL is configuration: do not assume the GitHub-only OSS intake is Cloud's API.
export async function reportFailures({ github, context, env, send = fetch }) {
  if (!env.INTAKE_URL?.startsWith('https://') || !env.INTAKE_TOKEN) {
    throw new Error('Native Detent CI intake URL/token are not configured');
  }
  const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRunAttempt, {
    ...context.repo, run_id: context.runId,
    attempt_number: Number(env.GITHUB_RUN_ATTEMPT), per_page: 100,
  });
  const failed = jobs.filter(job => ['failure', 'timed_out', 'startup_failure', 'action_required'].includes(job.conclusion));
  const errors = [];
  for (const job of failed) {
    try {
      let output;
      try {
        const logs = await github.rest.actions.downloadJobLogsForWorkflowRun({
          ...context.repo, job_id: job.id,
        });
        output = typeof logs.data === 'string' ? logs.data : Buffer.from(logs.data).toString('utf8');
      } catch (error) {
        // Startup failures and expired logs can have no downloadable output.
        // Still file the failed job, but keep the reporter red for missing evidence.
        errors.push(`${job.name}: Job logs unavailable: ${error.message}`);
        const steps = (job.steps || []).filter(step => step.conclusion && !['success', 'skipped'].includes(step.conclusion));
        output = `Job logs unavailable: ${error.message}\nAvailable step results:\n${steps.map(step => `${step.name}: ${step.conclusion}`).join('\n') || '(none)'}`;
      }
      const excerpt = output.slice(-16000).replaceAll('```', '`\u200b``');
      const runURL = `${env.GITHUB_SERVER_URL}/${env.GITHUB_REPOSITORY}/actions/runs/${context.runId}/attempts/${env.GITHUB_RUN_ATTEMPT}`;
      const payload = {
        project_id: 'prj_e7aab744eb84410fafd437f478992bd1',
        fingerprint: `detent.build:hourly-ci:${job.name}`,
        // Receiver must create Todo/High, or comment on the open matching issue
        // without resetting its status. Delivery retries must be idempotent.
        state: 'Todo', priority: 2,
        delivery_id: `${context.runId}:${env.GITHUB_RUN_ATTEMPT}:${job.id}`,
        summary: `Hourly CI failed: ${job.name}`,
        details: `Run: ${runURL}\nJob: ${job.html_url}\nTested develop SHA: ${env.TESTED_SHA}\nConclusion: ${job.conclusion}\n\nFailing output (last 16000 characters; full log at the job link):\n\n\`\`\`text\n${excerpt}\n\`\`\`\n\n\`\`\`detent-agent\nschema: 1\neffort: high\n\`\`\``,
      };
      const response = await send(env.INTAKE_URL, {
        method: 'POST', redirect: 'error', signal: AbortSignal.timeout(30000),
        headers: { Authorization: `Bearer ${env.INTAKE_TOKEN}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!response.ok) throw new Error(`Native intake returned HTTP ${response.status}`);
      // A 200 HTML login page is not evidence that a native issue was filed.
      const receipt = await response.json();
      if (!/^wi_[a-z0-9]+$/.test(receipt.work_item_id) || !['created', 'commented', 'duplicate'].includes(receipt.action)) {
        throw new Error('Native intake did not acknowledge a work item and delivery action');
      }
      console.log(`${job.name}: ${receipt.action} ${receipt.work_item_id}`);
    } catch (error) {
      errors.push(`${job.name}: ${error.message}`);
    }
  }
  // Attempt every failed job even if one delivery fails. Never hide intake errors.
  if (errors.length) throw new Error(errors.join('\n'));
}
