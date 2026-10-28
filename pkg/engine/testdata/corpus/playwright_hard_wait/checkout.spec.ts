import { test, expect } from '@playwright/test';

test('completes checkout flow', async ({ page }) => {
  await page.goto('/checkout');
  await page.waitForTimeout(2000);
  await expect(page.locator('#order-status')).toHaveText('Success');
});
