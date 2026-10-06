import { test, expect } from '@playwright/test';

test('homepage and all published documentation render', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const home = await page.goto('/');
  expect(home.status()).toBe(200);
  await expect(page.locator('h1')).toBeVisible();
  // This also catches a missing stylesheet build.
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(11, 13, 16)');
  const index = await page.goto('/docs');
  expect(index.status()).toBe(200);
  const paths = await page.locator('main a[href^="/docs/"]').evaluateAll(links =>
    [...new Set(links.map(link => link.getAttribute('href')))]);
  expect(paths.length).toBeGreaterThan(0);
  for (const path of paths) {
    const response = await page.goto(path);
    expect(response.status(), path).toBe(200);
    await expect(page.locator('article.docs-prose'), path).toBeVisible();
    await expect(page.locator('article.docs-prose'), path).not.toBeEmpty();
  }
  expect(errors).toEqual([]);
});

test('install links swap the selected platform through HTMX', async ({ page }) => {
  await page.goto('/install/macos');
  const platforms = page.getByRole('navigation', { name: 'Install platform' });
  const linux = platforms.locator('a[href="/install/linux"]');
  // A document navigation loses this marker; an HTMX swap keeps it.
  await page.evaluate(() => { window.ciHTMXMarker = true; });
  await linux.click();
  await expect(page).toHaveURL(/\/install\/linux$/);
  await expect(platforms.locator('a[href="/install/linux"]')).toHaveAttribute('aria-current', 'page');
  await expect(page.locator('#install-linux')).toBeVisible();
  expect(await page.evaluate(() => window.ciHTMXMarker)).toBe(true);
});
