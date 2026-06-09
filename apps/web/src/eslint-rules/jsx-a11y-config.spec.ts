/**
 * Wiring spec for the `eslint-plugin-jsx-a11y` flat-config block (Epic 11 Retro
 * AI-1). Mirrors the wiring-test shape of
 * `time-dependent-fixture-stability.spec.ts` (story 19-9): import the RESOLVED
 * flat config and assert the jsx-a11y block's scope, ignores, plugin
 * registration, and `warn` severity semantically — so a scope refactor cannot
 * silently widen/narrow coverage or flip the severity without failing here.
 *
 * Runs under `pnpm nx test web` (vitest picks up `src/**\/*.spec.ts`).
 *
 * NOTE on severity (AC #3): the block is intentionally wired at `warn`, NOT
 * `error`, to preserve the `lint:all` 0-errors gate while the existing
 * component a11y batch surfaces as warnings for retro-11-AI1b to clear. When
 * AI1b ratchets warn→error, the `registers the recommended rules at 'warn'`
 * assertion below is the load-bearing line it must flip.
 */
import { resolve } from 'node:path';
import { describe, it, expect } from 'vitest';

describe('eslint.config.mjs wiring for eslint-plugin-jsx-a11y', () => {
  let jsxA11yConfig: {
    files?: unknown;
    ignores?: unknown;
    rules?: Record<string, unknown>;
    plugins?: Record<string, unknown>;
  };

  it('has exactly one config object enabling jsx-a11y rules', async () => {
    const configPath = resolve(__dirname, '../../../../eslint.config.mjs');
    const flatConfig = (await import(/* @vite-ignore */ configPath)).default as Array<{
      rules?: Record<string, unknown>;
    }>;
    const matches = flatConfig.filter(
      (c) => c.rules && Object.keys(c.rules).some((k) => k.startsWith('jsx-a11y/'))
    );
    expect(matches).toHaveLength(1);
    jsxA11yConfig = matches[0] as typeof jsxA11yConfig;
  });

  it('scopes the block to apps/web/src/components/**/*.{ts,tsx}', () => {
    expect(jsxA11yConfig.files).toEqual(['apps/web/src/components/**/*.{ts,tsx}']);
  });

  it('ignores spec/test files and index.ts barrels under components/', () => {
    expect(jsxA11yConfig.ignores).toEqual([
      'apps/web/src/components/**/*.spec.{ts,tsx}',
      'apps/web/src/components/**/*.test.{ts,tsx}',
      'apps/web/src/components/**/index.ts',
    ]);
  });

  it('registers the jsx-a11y plugin object', () => {
    expect(jsxA11yConfig.plugins?.['jsx-a11y']).toBeDefined();
  });

  it('registers the recommended rules at warn severity', () => {
    // Spot-check a representative recommended rule; AC #3 mandates `warn` so the
    // 0-errors gate stays green. AI1b flips this to 'error' as its closing move.
    expect(jsxA11yConfig.rules?.['jsx-a11y/alt-text']).toBe('warn');
    // Every registered jsx-a11y rule is at warn — none leaked through at error.
    const severities = Object.entries(jsxA11yConfig.rules ?? {})
      .filter(([k]) => k.startsWith('jsx-a11y/'))
      .map(([, v]) => v);
    expect(severities.length).toBeGreaterThan(0);
    expect(severities.every((s) => s === 'warn')).toBe(true);
  });
});
