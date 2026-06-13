/**
 * Wiring spec for the Base UI wrap-once import ban (UX Redesign Phase 2 — UX2-1,
 * ADR D1-d F2). Asserts that `eslint.config.mjs` bans `@base-ui/react` outside
 * `apps/web/src/components/ui/`, so a refactor cannot silently drop the ban or
 * the ui/ carve-out. Reads the config as TEXT (node:fs) rather than dynamically
 * importing it — the import path goes through vite under the jsdom test env and
 * is fragile for a repo-root config file; a text assertion is reliable and
 * sufficient for a wiring guard.
 *
 * Runs under `pnpm nx test web` (vitest picks up `src/**\/*.spec.ts`).
 */
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, it, expect } from 'vitest';

const CONFIG = readFileSync(resolve(__dirname, '../../../../eslint.config.mjs'), 'utf8');

describe('eslint.config.mjs wiring for the Base UI import ban (F2)', () => {
  it('configures no-restricted-imports for @base-ui/react', () => {
    expect(CONFIG).toContain("'no-restricted-imports'");
    expect(CONFIG).toContain("'@base-ui/react'");
    expect(CONFIG).toContain("'@base-ui/react/*'");
  });

  it('associates the ban with the @base-ui/react pattern group at error severity', () => {
    // no-restricted-imports → 'error' → patterns group containing @base-ui/react,
    // all within one config block (≤600 chars between the rule and the package).
    expect(CONFIG).toMatch(/'no-restricted-imports':\s*\[\s*'error',[\s\S]{0,600}@base-ui\/react/);
  });

  it('exempts the components/ui wrap-point dir from the ban', () => {
    expect(CONFIG).toContain("'apps/web/src/components/ui/**'");
    // The ignore sits in the same block as the ban (ui dir appears before the rule).
    expect(CONFIG).toMatch(
      /apps\/web\/src\/components\/ui\/\*\*[\s\S]{0,400}'no-restricted-imports'/
    );
  });
});
