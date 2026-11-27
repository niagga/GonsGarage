'use client';

import { useEffect, useId, useSyncExternalStore } from 'react';
import { applyTheme, getStoredPreference, type ThemePreference } from '@/lib/theme';
import styles from './ThemeSwitcher.module.css';

// The stored preference is external state, so it is read with
// `useSyncExternalStore` instead of a `useState` filled from an effect. That keeps
// the render honest under SSR (see `getServerSnapshot`), avoids the
// `react-hooks/set-state-in-effect` lint rule, and needs no `mounted` flag, so the
// control is present in the first paint and the header never shifts.
const listeners = new Set<() => void>();

function emitChange() {
  for (const listener of listeners) listener();
}

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange);

  const onExternalChange = () => onStoreChange();
  globalThis.addEventListener('storage', onExternalChange);

  // `matchMedia` is missing in jsdom and exotic webviews; guard it rather than
  // assume every environment is a full browser.
  const mediaQuery =
    typeof globalThis.matchMedia === 'function'
      ? globalThis.matchMedia('(prefers-color-scheme: dark)')
      : null;
  mediaQuery?.addEventListener('change', onExternalChange);

  return () => {
    listeners.delete(onStoreChange);
    globalThis.removeEventListener('storage', onExternalChange);
    mediaQuery?.removeEventListener('change', onExternalChange);
  };
}

function getSnapshot(): ThemePreference {
  return getStoredPreference();
}

function getServerSnapshot(): ThemePreference {
  return 'system';
}

function setPreference(pref: ThemePreference) {
  applyTheme(pref);
  emitChange();
}

export default function ThemeSwitcher() {
  const pref = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  // The switcher renders inside several shells; `useId` keeps the label/select
  // association unique instead of repeating a hardcoded id.
  const selectId = useId();

  // Syncing the resolved theme to <html> is an external-system update, not state.
  useEffect(() => {
    applyTheme(pref);
    if (pref !== 'system') return undefined;
    if (typeof globalThis.matchMedia !== 'function') return undefined;

    const mediaQuery = globalThis.matchMedia('(prefers-color-scheme: dark)');
    const onSystemChange = () => applyTheme('system');
    mediaQuery.addEventListener('change', onSystemChange);
    return () => mediaQuery.removeEventListener('change', onSystemChange);
  }, [pref]);

  return (
    <div className={styles.root}>
      <label className={styles.label} htmlFor={selectId}>
        Tema
      </label>
      <select
        id={selectId}
        className={styles.select}
        value={pref}
        onChange={(e) => setPreference(e.target.value as ThemePreference)}
        aria-label="Tema da aplicação"
      >
        <option value="system">Sistema</option>
        <option value="light">Claro</option>
        <option value="dark">Escuro</option>
      </select>
    </div>
  );
}
