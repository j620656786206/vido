/**
 * Tests for the custom ESLint rule `local/time-dependent-fixture-stability`
 * (story 19-9).
 *
 * Runs under `pnpm nx test web` (vitest picks up `src/**\/*.spec.ts`).
 * ESLint's `RuleTester` uses the ambient `describe`/`it` globals, which
 * vitest provides (`globals: true` in vite.config.ts). Mirrors the
 * structural shape of `implements-pen-node-id.spec.ts` (19-3).
 */
import { resolve } from 'node:path';
import { RuleTester } from 'eslint';
import { describe, it, expect } from 'vitest';
import plugin from './time-dependent-fixture-stability.js';

const rule = plugin.rules['time-dependent-fixture-stability'];

const ruleTester = new RuleTester({
  languageOptions: { ecmaVersion: 2022, sourceType: 'module' },
});

ruleTester.run('time-dependent-fixture-stability', rule, {
  valid: [
    // (a) Clock-mocked header + Date.now() in body — accepted
    {
      code:
        '// Clock-mocked: gallery fixture library-recently-added uses page.clock.setFixedTime\n' +
        'export const f = () => Date.now();\n',
    },
    // (a') Clock-mocked header + state-suffixed fixture id (kebab/slash)
    {
      code:
        '// Clock-mocked: gallery fixture library-recently-added/recent uses page.clock.setFixedTime\n' +
        'export const f = () => Date.now();\n',
    },
    // (b) Clock-injected header + new Date() zero-args — accepted
    {
      code:
        '// Clock-injected: component accepts `clock` prop; no fixture-side mock needed\n' +
        'export const f = () => new Date();\n',
    },
    // (c) Time-bomb-exempt header + Date.now() — accepted (exempt with rationale)
    {
      code:
        '// Time-bomb-exempt: Date.now used only for unique-ID generation; never visually rendered (Murat)\n' +
        'export const f = () => `id-${Date.now()}`;\n',
    },
    // (d) No Date use → no header required, no error
    {
      code: 'export const x = 1;\n',
    },
    // (e) new Date(arg) is a deterministic formatter — NOT in trigger (a); no header needed.
    // (RuleTester uses ESLint's default Espree parser without TS — keep test code plain JS.)
    {
      code: 'export const f = (s) => new Date(s).toLocaleString();\n',
    },
    // (f) Block-comment / JSDoc form of marker — accepted
    {
      code:
        '/**\n * Time-bomb-exempt: parse-progress is a hook; ambient time only flows into log payloads (Murat)\n */\n' +
        'export const f = () => new Date().toISOString();\n',
    },
    // (g) Two-line header — Rule 23 marker above Rule 21 marker — both accepted; both rules ignore the other's line
    {
      code:
        '// Clock-mocked: gallery fixture library-recently-added uses page.clock.setFixedTime\n' +
        '// Implements: Component/Foo (abc123)\n' +
        'export const f = () => Date.now();\n',
    },
    // (h) `Date.UTC` triggered but marker present — accepted
    {
      code:
        '// Time-bomb-exempt: Date.UTC normalisation in a pure formatter; output is wall-clock-independent\n' +
        'export const f = () => Date.UTC(2026, 0, 1);\n',
    },
    // (i) `Date.parse` triggered but marker present — accepted
    {
      code:
        '// Clock-injected: component accepts `clock` prop; uses Date.parse on caller-provided strings\n' +
        'export const f = (s) => Date.parse(s);\n',
    },
    // (j) String containing the literal text "Date.now()" — NOT an AST trigger; no header needed
    {
      code: 'export const log = "Date.now() is the wall clock";\n',
    },
  ],
  invalid: [
    // (k) Date.now() with no header
    {
      code: 'export const f = () => Date.now();\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
    // (l) new Date() zero-args with no header
    {
      code: 'export const f = () => new Date();\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
    // (m) Date.UTC() with no header
    {
      code: 'export const f = () => Date.UTC(2026, 0, 1);\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
    // (n) Clock-mocked marker placed AFTER first statement — not a leading comment
    {
      code:
        'export const x = 1;\n' +
        '// Clock-mocked: gallery fixture foo uses page.clock.setFixedTime\n' +
        'export const f = () => Date.now();\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
    // (o) Malformed Clock-mocked — missing fixture id phrase ("uses page.clock.setFixedTime" absent)
    {
      code:
        '// Clock-mocked: gallery fixture library-recently-added is mocked somehow\n' +
        'export const f = () => Date.now();\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
    // (p) Malformed Time-bomb-exempt — rationale empty (just trailing colon)
    {
      code: '// Time-bomb-exempt:\nexport const f = () => Date.now();\n',
      errors: [{ messageId: 'time-bomb-detected' }],
    },
  ],
});

// Out-of-scope files (specs, hooks/services/stores, route files, index.ts barrels)
// are excluded by the flat-config `files`/`ignores` of the config object that
// enables this rule in `eslint.config.mjs`, NOT by the rule body. Load the
// resolved flat config and assert the scoping object semantically — a refactor
// can't silently widen/narrow scope. Mirrors the 19-3 spec's wiring test block.
describe('eslint.config.mjs wiring for local/time-dependent-fixture-stability', () => {
  let ruleConfig: {
    files?: unknown;
    ignores?: unknown;
    rules?: Record<string, unknown>;
    plugins?: Record<string, unknown>;
  };

  it('has exactly one config object enabling the rule', async () => {
    const configPath = resolve(__dirname, '../../../../eslint.config.mjs');
    const flatConfig = (await import(/* @vite-ignore */ configPath)).default as Array<{
      rules?: Record<string, unknown>;
    }>;
    const matches = flatConfig.filter(
      (c) => c.rules && 'local/time-dependent-fixture-stability' in c.rules
    );
    expect(matches).toHaveLength(1);
    ruleConfig = matches[0] as typeof ruleConfig;
  });

  it('registers the rule at error severity', () => {
    expect(ruleConfig.rules?.['local/time-dependent-fixture-stability']).toBe('error');
  });

  it('registers the local plugin object that exposes the rule', () => {
    const local = ruleConfig.plugins?.local as { rules?: Record<string, unknown> } | undefined;
    expect(local?.rules?.['time-dependent-fixture-stability']).toBe(rule);
  });

  it('scopes the rule to apps/web/src/components/**/*.{ts,tsx}', () => {
    expect(ruleConfig.files).toEqual(['apps/web/src/components/**/*.{ts,tsx}']);
  });

  it('ignores spec/test files and index.ts barrels under components/', () => {
    expect(ruleConfig.ignores).toEqual([
      'apps/web/src/components/**/*.spec.{ts,tsx}',
      'apps/web/src/components/**/*.test.{ts,tsx}',
      'apps/web/src/components/**/index.ts',
    ]);
  });
});
