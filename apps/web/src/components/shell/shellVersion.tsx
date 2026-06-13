// Implements: <utility — no .pen counterpart>
/**
 * Shell-version context (UX Redesign Phase 2 — UX2-2).
 *
 * The `new_shell_enabled` flag is read in exactly one place (`__root.tsx`, F4),
 * which selects `AppShellV2` vs the legacy `AppShell`. Migrated routes still need
 * to know which chassis they render inside so they can show v2 vs legacy content
 * WITHOUT re-reading the flag (preserving the single chokepoint). `AppShellV2`
 * provides `'v2'`; everywhere else defaults to `'legacy'`. A route calls
 * `useShellVersion()` and branches its content — the flag itself stays read-once.
 */
import { createContext, useContext } from 'react';

export type ShellVersion = 'legacy' | 'v2';

const ShellVersionContext = createContext<ShellVersion>('legacy');

export const ShellVersionProvider = ShellVersionContext.Provider;

export function useShellVersion(): ShellVersion {
  return useContext(ShellVersionContext);
}
