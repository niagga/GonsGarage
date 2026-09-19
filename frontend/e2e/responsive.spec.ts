import { expect, test, type Page } from '@playwright/test';

type ResponsiveRoute = {
  name: string;
  path: string;
  expectedUrl: RegExp;
  expectStableUi: (page: Page) => Promise<void>;
};

const overflowTolerancePx = 4;

const routes: ResponsiveRoute[] = [
  {
    name: 'home',
    path: '/',
    expectedUrl: /\/(?:\?.*)?$/,
    expectStableUi: async (page) => {
      await expect(page.getByRole('heading', { name: 'A sua oficina de confiança' })).toBeVisible();
    },
  },
  {
    name: 'login',
    path: '/auth/login',
    expectedUrl: /\/auth\/login(?:\?.*)?$/,
    expectStableUi: async (page) => {
      await expect(page.getByRole('heading', { name: 'Iniciar sessão' })).toBeVisible();
    },
  },
  {
    name: 'register',
    path: '/auth/register',
    expectedUrl: /\/auth\/register(?:\?.*)?$/,
    expectStableUi: async (page) => {
      await expect(page.getByRole('heading', { name: 'Criar conta' })).toBeVisible();
    },
  },
  {
    name: 'dashboard auth gate',
    path: '/dashboard',
    expectedUrl: /\/dashboard(?:\?.*)?$/,
    expectStableUi: async (page) => {
      await expect(page.getByText('A sessão a carregar')).toBeVisible({ timeout: 30_000 });
    },
  },
  {
    name: 'my invoices auth gate',
    path: '/my-invoices',
    expectedUrl: /\/my-invoices(?:\?.*)?$/,
    expectStableUi: async (page) => {
      await expect(page.getByText('A carregar faturas')).toBeVisible({ timeout: 30_000 });
    },
  },
];

async function expectNoHorizontalDocumentOverflow(page: Page, routeName: string): Promise<void> {
  const metrics = await page.evaluate(() => {
    const root = document.documentElement;
    const body = document.body;
    const bodyScrollWidth = body?.scrollWidth ?? 0;
    const bodyClientWidth = body?.clientWidth ?? 0;

    return {
      viewportWidth: window.innerWidth,
      documentScrollWidth: root.scrollWidth,
      documentClientWidth: root.clientWidth,
      bodyScrollWidth,
      bodyClientWidth,
      maxScrollWidth: Math.max(root.scrollWidth, bodyScrollWidth),
      maxClientWidth: Math.max(root.clientWidth, bodyClientWidth),
    };
  });

  expect(
    metrics.maxScrollWidth,
    `${routeName} should not create horizontal overflow: ${JSON.stringify(metrics)}`,
  ).toBeLessThanOrEqual(metrics.maxClientWidth + overflowTolerancePx);
}

test.describe('responsive route smoke @responsive', () => {
  for (const route of routes) {
    test(`${route.name} has stable responsive layout`, async ({ page }) => {
      await page.goto(route.path, { waitUntil: 'domcontentloaded' });

      await expect(page).toHaveURL(route.expectedUrl);
      await route.expectStableUi(page);
      await expect(page).toHaveURL(route.expectedUrl);
      await expectNoHorizontalDocumentOverflow(page, route.name);
    });
  }
});
