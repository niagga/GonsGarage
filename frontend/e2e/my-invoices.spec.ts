import { expect, test } from '@playwright/test';
import { demoClient, seedSession } from './fixtures/auth';

test('a client can open the own-invoices route', async ({ page }) => {
  await seedSession(page, demoClient.email, demoClient.password);
  await page.goto('/my-invoices');

  await expect(page).toHaveURL(/\/my-invoices(?:\/.*)?(?:\?.*)?$/);
  await expect(page.getByText('A carregar faturas')).toBeVisible();
});
