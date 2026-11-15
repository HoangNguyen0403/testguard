import { test, expect } from '@playwright/test';

test.skip('declared skipped', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test.fixme('known broken', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test.describe.skip('skipped suite', () => {
  test('inner', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Home');
  });
});
