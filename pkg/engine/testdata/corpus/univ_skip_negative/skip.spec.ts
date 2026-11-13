import { test, expect } from '@playwright/test';

test('conditional skip by browser', async ({ page, browserName }) => {
  test.skip(browserName === 'webkit', 'clipboard API unsupported on webkit');
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test('normal passing test', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});
