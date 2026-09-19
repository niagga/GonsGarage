import { expect, test } from '@playwright/test';
import { loginAs, registerClient, uniqueEmail } from './fixtures/auth';

test('a client can register, log in, and open the own-invoices area', async ({ page }) => {
  const email = uniqueEmail('client');
  const password = 'Playwright123!';

  await registerClient(page, {
    firstName: 'Playwright',
    lastName: 'Client',
    email,
    password,
  });

  await loginAs(page, email, password);
  await page.goto('/my-invoices');

  await expect(page).toHaveURL(/\/my-invoices(?:\/.*)?(?:\?.*)?$/);
  await expect(page.getByRole('navigation', { name: 'Principal' })).toBeVisible();
  await expect(page.getByText('As minhas faturas')).toBeVisible();
  await expect(page.getByText('Ainda não tem faturas registadas.')).toBeVisible();
});
