import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createRequire } from 'node:module';
import { describe, it, expect } from 'vitest';
import { Linter } from 'eslint';

/**
 * local/no-hardcoded-palette — the door-closer behind the token-debt migration
 * (PRs #291/#292/#294/#295).
 *
 * The sibling wiring specs import eslint.config.mjs, which vitest cannot load
 * in every environment (they fail locally, pass in CI). This spec avoids that:
 * the RULE is tested through eslint's Linter with an inline flat config, and
 * the WIRING is asserted against the config file's text.
 */
const require = createRequire(import.meta.url);
// eslint-disable-next-line @typescript-eslint/no-var-requires
const plugin = require('./no-hardcoded-palette.js') as {
  rules: Record<string, import('eslint').Rule.RuleModule>;
};

function lint(code: string): string[] {
  const linter = new Linter({ configType: 'flat' });
  const messages = linter.verify(
    code,
    [
      {
        files: ['**/*.tsx'],
        plugins: { local: plugin },
        rules: { 'local/no-hardcoded-palette': 'error' },
        languageOptions: { ecmaVersion: 2022, sourceType: 'module' },
      },
    ],
    'component.tsx'
  );
  return messages.map((m) => m.ruleId ?? '');
}

describe('local/no-hardcoded-palette', () => {
  it.each([
    ['plain literal', `const c = "text-blue-300";`],
    ['inside a class string', `const c = "rounded p-4 bg-red-900/30 text-sm";`],
    ['template literal', 'const c = `border ${x} border-amber-500/30`;'],
    ['checkbox accent-color', `const c = "accent-blue-500";`],
    ['gradient stop', `const c = "from-emerald-400";`],
  ])('flags %s', (_name, code) => {
    expect(lint(code)).toContain('local/no-hardcoded-palette');
  });

  it.each([
    ['the token vocabulary itself', `const c = "bg-[var(--error-tint)] text-[var(--error-text)]";`],
    ['spacing-scale utilities', `const c = "p-4 gap-2 w-60 rounded-lg border";`],
    ['non-palette colour words', `const c = "text-white bg-black/50";`],
    // Comments are not AST string nodes — the migration left explanatory
    // comments naming old literals, and they must not trip the rule.
    [
      'literals in comments',
      `// was bg-blue-800 text-blue-300\nconst c = "bg-[var(--accent-tint)]";`,
    ],
  ])('does not flag %s', (_name, code) => {
    expect(lint(code)).toEqual([]);
  });

  describe('wiring in eslint.config.mjs (asserted as text — see header)', () => {
    const config = readFileSync(join(__dirname, '../../../../eslint.config.mjs'), 'utf8');

    it('imports the rule file', () => {
      expect(config).toContain('./apps/web/src/eslint-rules/no-hardcoded-palette.js');
    });

    it('spreads it into the shared local plugin object', () => {
      expect(config).toContain('...noHardcodedPalette.rules,');
    });

    it('enables it at error severity over components AND routes', () => {
      expect(config).toMatch(/'local\/no-hardcoded-palette': 'error'/);
      const block = config.slice(config.indexOf("'local/no-hardcoded-palette'") - 600);
      expect(block).toContain('apps/web/src/routes/**/*.{ts,tsx}');
    });

    it('is exemption-free: no per-file disables anywhere in src', () => {
      // The migration's whole point — nothing left to allowlist.
      expect(config).not.toContain('no-hardcoded-palette": "off');
      expect(config).not.toContain("no-hardcoded-palette': 'off");
    });
  });
});
