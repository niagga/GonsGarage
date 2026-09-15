import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const clientCss = readFileSync(
  join(process.cwd(), 'src/app/client/client.module.css'),
  'utf8',
);

describe('gentleman-cute dark comfort (client shell)', () => {
  it('does not use hardcoded white panel backgrounds', () => {
    expect(clientCss).not.toMatch(/background(-color)?:\s*white\b/);
    expect(clientCss).toMatch(/var\(--surface-panel\)/);
    expect(clientCss).toMatch(/var\(--surface-header\)/);
  });
});
