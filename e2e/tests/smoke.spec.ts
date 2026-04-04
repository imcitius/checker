import { test, expect } from '@playwright/test';

test('smoke: page loads successfully', async ({ page }) => {
  const response = await page.goto('/');
  expect(response).not.toBeNull();
  expect(response!.status()).toBeLessThan(400);
});
