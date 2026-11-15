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
  test('L15 adds item', async ({ page }) => {
    await page.getByRole('button', { name: 'Add' }).click();
    await expect(page.getByTestId('count')).toHaveText('1');
  });

  test('L20 steps', async ({ page }) => {
    test.setTimeout(60_000);
    test.slow();
    await test.step('open cart', async () => {
      await page.getByRole('link', { name: 'Cart' }).click();
    });
    await test.step('verify', async () => {
      await expect(page).toHaveURL(/cart/);
    });
  });

  test('L31 conditional skip in body', async ({ page, browserName }) => {
    test.skip(browserName === 'webkit', 'not supported');
    await expect(page.getByText('Pay')).toBeVisible();
  });

  test('L36 soft and poll', async ({ page }) => {
    await expect.poll(() => page.title()).toBe('Shop');
    expect.soft(await page.title()).toBe('Shop');
  });

  test('L41 waitForURL is an implicit check', async ({ page }) => {
    await page.getByRole('link', { name: 'Cart' }).click();
    await page.waitForURL('**/cart');
  });
});
