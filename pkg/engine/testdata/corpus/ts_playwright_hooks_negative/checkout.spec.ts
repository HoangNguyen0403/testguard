import { test, expect } from '@playwright/test';

test.use({ viewport: { width: 800, height: 600 } });
test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({ page }) => {
  await page.goto('/');
});

test.afterAll(async () => {
  await cleanup();
});

test.describe('checkout', () => {
  test('adds item', async ({ page }) => {
    await page.getByRole('button', { name: 'Add' }).click();
    await expect(page.getByTestId('count')).toHaveText('1');
  });

  test('steps with explicit timeout', async ({ page }) => {
    test.setTimeout(60_000);
    test.slow();
    await test.step('open cart', async () => {
      await page.getByRole('link', { name: 'Cart' }).click();
    });
    await expect(page).toHaveURL(/cart/);
  });

  test('conditional runtime skip', async ({ page, browserName }) => {
    test.skip(browserName === 'webkit', 'not supported');
    await expect(page.getByText('Pay')).toBeVisible();
  });
});
