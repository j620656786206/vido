/**
 * Wiring spec for the `eslint-plugin-jsx-a11y` flat-config block (Epic 11 Retro
 * AI-1; ratcheted to native error severity by retro-11-AI1b). Mirrors the
 * wiring-test shape of `time-dependent-fixture-stability.spec.ts` (story
 * 19-9): import the RESOLVED flat config and assert the jsx-a11y block's
 * scope, ignores, plugin registration, and native `error` severity — so a
 * scope refactor cannot silently widen/narrow coverage, flip the severity, or
 * shrink the enabled rule set without failing here.
 *
 * Runs under `pnpm nx test web` (vitest picks up `src/**\/*.spec.ts`).
 *
 * NOTE on severity (retro-11-AI1b ratchet): the block now applies the
 * recommended ruleset at its NATIVE severities — enabled rules at `error`
 * (options preserved), rules recommended deliberately ships as `off` stay
 * off. The AI1-era `warn` remap (which kept the 0-errors gate green while
 * the batch surfaced) is gone: AI1b cleared the batch, so any new violation
 * must now FAIL `lint:all`, not accumulate as an ignorable warning.
 */
import { resolve } from 'node:path';
import { describe, it, expect } from 'vitest';
import jsxA11y from 'eslint-plugin-jsx-a11y';

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

  it('registers the recommended rules at their native error severity (AI1b ratchet)', () => {
    // Spot-check a representative recommended rule; the AI1b warn→error
    // ratchet means a new violation now FAILS lint:all instead of accumulating
    // as a warning.
    expect(jsxA11yConfig.rules?.['jsx-a11y/alt-text']).toBe('error');
    // Native recommended severities only: 'error' (string or [severity, opts]
    // tuple) for enabled rules, 'off' for the rules recommended deliberately
    // disables. No rule may be left at the AI1-era 'warn'.
    const severityOf = (v: unknown) => (Array.isArray(v) ? v[0] : v);
    const entries = Object.entries(jsxA11yConfig.rules ?? {}).filter(([k]) =>
      k.startsWith('jsx-a11y/')
    );
    expect(entries.length).toBeGreaterThan(0);
    expect(entries.some(([, v]) => severityOf(v) === 'error')).toBe(true);
    expect(entries.every(([, v]) => severityOf(v) === 'error' || severityOf(v) === 'off')).toBe(
      true
    );
  });

  it('pins the rule set to the FULL native recommended ruleset (no silent shrinkage)', () => {
    // The block must stay exactly `jsxA11y.flatConfigs.recommended.rules` —
    // a future hand-pruned subset would pass the severity assertions above
    // but fail here, so the enabled set cannot be quietly gutted.
    expect(jsxA11yConfig.rules).toEqual(jsxA11y.flatConfigs.recommended.rules);
  });
});
