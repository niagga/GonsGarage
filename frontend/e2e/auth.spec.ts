import { expect, test } from '@playwright/test';
import { loginAsDemoAdmin } from './fixtures/auth';

test('admin can sign in and reach the dashboard shell', async ({ page }) => {
  await loginAsDemoAdmin(page);

  await expect(page).toHaveURL(/\/dashboard(?:\?.*)?$/);
  await expect(page.getByRole('navigation', { name: 'Principal' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Contabilidade' })).toBeVisible();
  await expect(page.getByText('Olá,')).toBeVisible();
});
