import React, { type HTMLAttributes } from 'react';

export type AppLoadingSize = 'sm' | 'md' | 'lg';

export type AppLoadingProps = {
  size: AppLoadingSize;
  /** Default `true`. Set `false` when an ancestor already sets `aria-busy`. */
  'aria-busy'?: boolean;
  /** Announced to screen readers (visually hidden). */
  label?: string;
  className?: string;
} & Omit<HTMLAttributes<HTMLSpanElement>, 'children' | 'size'>;

/**
 * Unified loading spinner (`mvp-ui-visual-parity` Phase 1).
 * Sizes map to `--app-loading-size-*` in `tokens.css` + `.app-loading--*` in `utilities.css`.
 */
export function AppLoading({
  size,
  'aria-busy': ariaBusy = true,
  label,
  className,
  ...rest
}: AppLoadingProps) {
  const classes = ['app-loading', `app-loading--${size}`, className].filter(Boolean).join(' ');
  return (
    <span role="status" aria-busy={ariaBusy} className={classes} {...rest}>
      {label ? <span className="sr-only">{label}</span> : null}
    </span>
  );
}
