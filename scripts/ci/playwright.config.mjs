import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './browser',
  workers: 1,
  reporter: 'list',
  outputDir: `${process.env.RUNNER_TEMP || process.env.TMPDIR || process.env.TMP || process.env.TEMP}/browser-results`,
  use: {
    baseURL: process.env.BROWSER_URL || 'http://127.0.0.1:3000',
    browserName: 'chromium',
    viewport: { width: 1280, height: 900 },
    deviceScaleFactor: 1,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
});
