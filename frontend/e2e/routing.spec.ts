import { expect, test } from '@playwright/test';

test('unauthenticated users see the auth gate on protected routes', async ({ page }) => {
  await page.goto('/dashboard');
  await expect(page.getByText('A sessão a carregar')).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole('navigation', { name: 'Principal' })).toHaveCount(0);
});

test('unauthenticated users see the auth gate on my invoices', async ({ page }) => {
  await page.goto('/my-invoices');
  await expect(page.getByText('A carregar faturas')).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole('navigation', { name: 'Principal' })).toHaveCount(0);
});
