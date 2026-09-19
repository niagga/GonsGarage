import { expect, type Page } from '@playwright/test';

const apiBaseUrl = process.env.PLAYWRIGHT_API_BASE_URL ?? 'http://localhost:8080';

export const demoAdmin = {
  email: 'admin.demo@gonsgarage.local',
  password: 'AdminDemo123',
};

export const demoClient = {
  email: 'cliente.demo@gonsgarage.local',
  password: 'ClienteDemo123',
};

export async function seedSession(page: Page, email: string, password: string): Promise<void> {
  const loginResponse = await page.request.post(`${apiBaseUrl}/api/v1/auth/login`, {
    data: { email, password },
  });
  expect(loginResponse.ok()).toBeTruthy();
  const loginBody = (await loginResponse.json()) as { token?: string };
  const token = loginBody.token;
  expect(token, 'expected login token').toEqual(expect.any(String));
  if (typeof token !== 'string' || token.length === 0) {
    throw new Error('expected login token');
  }

  const meResponse = await page.request.get(`${apiBaseUrl}/api/v1/auth/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'application/json',
    },
  });
  expect(meResponse.ok()).toBeTruthy();
  const meBody = (await meResponse.json()) as { user?: Record<string, unknown> };
  expect(meBody.user, 'expected auth/me user').toBeTruthy();
  if (!meBody.user) {
    throw new Error('expected auth/me user');
  }

  await page.addInitScript(({ authToken, authUser }) => {
    localStorage.setItem('auth_token', authToken);
    localStorage.setItem('auth_user', JSON.stringify(authUser));
    localStorage.setItem('gons-garage-auth', JSON.stringify({ user: authUser, token: authToken }));
  }, { authToken: token, authUser: meBody.user });
}

export async function loginAsDemoAdmin(page: Page): Promise<void> {
  await seedSession(page, demoAdmin.email, demoAdmin.password);
  await page.goto('/dashboard');
  await expect(page.getByText('A sessão a carregar')).toBeVisible({ timeout: 30_000 });
}
