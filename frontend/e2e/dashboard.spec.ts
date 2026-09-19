import { expect, test } from '@playwright/test';
import { loginAsDemoAdmin } from './fixtures/auth';

test('admin sees the staff navigation and dashboard content', async ({ page }) => {
  await loginAsDemoAdmin(page);

  await expect(page.getByRole('navigation', { name: 'Principal' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Utilizadores' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Peças (stock)' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Contabilidade' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Terminar sessão' })).toBeVisible();
});
