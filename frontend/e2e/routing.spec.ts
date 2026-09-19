import { expect, test } from '@playwright/test';

test('unauthenticated users are redirected away from protected routes', async ({ page }) => {
  await page.goto('/dashboard');
  await page.waitForURL(/\/auth\/login(?:\?.*)?$/, { timeout: 30_000 });
  await expect(page.getByRole('heading', { name: 'Iniciar sessão' })).toBeVisible();
});

test('unauthenticated users cannot open the my invoices area', async ({ page }) => {
  await page.goto('/my-invoices');
  await page.waitForURL(/\/auth\/login(?:\?.*)?$/, { timeout: 30_000 });
  await expect(page.getByRole('heading', { name: 'Iniciar sessão' })).toBeVisible();
});
