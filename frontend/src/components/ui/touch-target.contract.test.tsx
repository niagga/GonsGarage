import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';

import { buttonVariants } from './button';
import { Input } from './input';

const MIN_HEIGHT_PX: Record<string, number> = {
  'min-h-11': 44,
  'min-h-12': 48,
};

const TOUCH_GOAL_PX = 44;

describe('shared controls keep a 44px touch target', () => {
  it.each(['default', 'sm', 'lg', 'icon'] as const)(
    'Button size %s meets the goal without a fixed height',
    (size) => {
      const classes = buttonVariants({ size }).split(/\s+/);

      // A fixed `h-*` caps the control: a long or translated label gets clipped
      // instead of growing. The touch floor must come from `min-h-*`.
      expect(classes.filter((c) => /^h-\d/.test(c))).toHaveLength(0);

      const heights = classes
        .filter((c) => c in MIN_HEIGHT_PX)
        .map((c) => MIN_HEIGHT_PX[c]);

      expect(heights.length).toBeGreaterThan(0);
      expect(Math.min(...heights)).toBeGreaterThanOrEqual(TOUCH_GOAL_PX);
    },
  );

  it('gives the icon Button a square target', () => {
    expect(buttonVariants({ size: 'icon' }).split(/\s+/)).toContain('min-w-11');
  });

  it('Input has a 44px floor and no fixed height', () => {
    const { container } = render(<Input aria-label="campo" />);
    const classes = container.querySelector('input')!.className.split(/\s+/);

    expect(classes).toContain('min-h-11');
    expect(classes.some((c) => /^h-\d/.test(c))).toBe(false);
  });

  it('does not shrink the Select trigger back to the old 36px box', () => {
    const source = readFileSync(join(process.cwd(), 'src/components/ui/select.tsx'), 'utf8');
    const triggerClasses = /"flex ([^"]*)w-full/.exec(source)?.[1] ?? '';

    expect(triggerClasses).toContain('min-h-11');
    expect(triggerClasses).not.toMatch(/(^|\s)h-9(\s|$)/);
  });
});
