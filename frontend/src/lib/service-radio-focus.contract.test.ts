import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const css = readFileSync(
  join(process.cwd(), 'src/app/appointments/new/new-appointment.module.css'),
  'utf8',
);

describe('service selection radios stay keyboard reachable', () => {
  it('never hides the radio with display: none', () => {
    // `display: none` removes the input from the tab order entirely: the service
    // cards look interactive but a keyboard user can neither reach nor change them.
    const baseRule = /\.serviceOption input\[type=["']radio["']\]\s*\{([^}]*)\}/.exec(css);

    expect(baseRule).not.toBeNull();
    expect(baseRule?.[1]).not.toMatch(/display:\s*none/);
  });

  it('gives the focused radio a visible indicator on its card', () => {
    expect(css).toMatch(
      /\.serviceOption input\[type=["']radio["']\]:focus-visible\s*\+\s*\.serviceCard\s*\{[^}]*outline:/,
    );
  });
});
