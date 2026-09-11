import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AppLoading } from './AppLoading';

describe('AppLoading', () => {
  it('applies size class sm', () => {
    const { container } = render(<AppLoading size="sm" aria-busy={false} />);
    const el = container.querySelector('.app-loading--sm');
    expect(el).toBeTruthy();
  });

  it('applies size class md (triangulation)', () => {
    const { container } = render(<AppLoading size="md" aria-busy={false} />);
    expect(container.querySelector('.app-loading--md')).toBeTruthy();
  });

  it('applies size class lg', () => {
    const { container } = render(<AppLoading size="lg" aria-busy={false} />);
    expect(container.querySelector('.app-loading--lg')).toBeTruthy();
  });

  it('sets aria-busy true by default', () => {
    render(<AppLoading size="md" />);
    const status = screen.getByRole('status');
    expect(status).toHaveAttribute('aria-busy', 'true');
  });

  it('does not mark busy when aria-busy false', () => {
    render(<AppLoading size="md" aria-busy={false} />);
    const status = screen.getByRole('status');
    expect(status.getAttribute('aria-busy')).not.toBe('true');
  });

  it('renders sr-only label when label provided', () => {
    const { container } = render(
      <AppLoading size="md" aria-busy={false} label="A carregar dados" />,
    );
    const hidden = container.querySelector('.sr-only');
    expect(hidden?.textContent).toBe('A carregar dados');
  });
});
