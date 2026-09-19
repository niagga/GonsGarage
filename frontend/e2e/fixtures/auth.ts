import { expect, type Page } from '@playwright/test';

export const demoAdmin = {
  email: 'admin@gonsgarage.com',
  password: 'admin123',
};

export function uniqueEmail(prefix = 'playwright'): string {
  const stamp = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  return `${prefix}.${stamp}@example.com`;
}

export async function loginAs(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/auth/login');
  await expect(page.getByRole('heading', { name: 'Iniciar sessão' })).toBeVisible();
  await page.getByLabel('E-mail').fill(email);
  await page.getByLabel('Palavra-passe').fill(password);
  await page.getByRole('button', { name: 'Iniciar sessão' }).click();
  await page.waitForURL(/\/dashboard(?:\?.*)?$/, { timeout: 30_000 });
}

export async function loginAsDemoAdmin(page: Page): Promise<void> {
  await loginAs(page, demoAdmin.email, demoAdmin.password);
}

export type ClientRegistration = {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
};

export async function registerClient(page: Page, registration: ClientRegistration): Promise<void> {
  await page.goto('/auth/register');
  await expect(page.getByRole('heading', { name: 'Criar conta' })).toBeVisible();
  await page.getByLabel('Nome').fill(registration.firstName);
  await page.getByLabel('Apelido').fill(registration.lastName);
  await page.getByLabel('E-mail').fill(registration.email);
  await page.getByLabel('Palavra-passe').fill(registration.password);
  await page.getByLabel('Confirmar palavra-passe').fill(registration.password);
  await page.getByRole('button', { name: 'Criar conta' }).click();
  await page.waitForURL(/\/auth\/login(?:\?.*)?$/, { timeout: 30_000 });
}
