import { expect, test } from '@playwright/test';
import { loginAsDemoAdmin } from './fixtures/auth';

test('admin can sign in and open the dashboard route', async ({ page }) => {
  await loginAsDemoAdmin(page);

  await expect(page).toHaveURL(/\/dashboard(?:\?.*)?$/);
  await expect(page.getByText('A sessão a carregar')).toBeVisible();
});
