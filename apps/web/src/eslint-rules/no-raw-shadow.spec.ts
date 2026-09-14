import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createRequire } from 'node:module';
import { describe, it, expect } from 'vitest';
import { Linter } from 'eslint';

/**
 * local/no-raw-shadow — the door-closer behind the dsr-9 elevation pass.
 *
 * Same testing strategy as the sibling rules: the RULE through Linter with an
 * inline flat config (vitest cannot import eslint.config.mjs in every
 * environment), the WIRING as config-file text.
 */
const require = createRequire(import.meta.url);
// eslint-disable-next-line @typescript-eslint/no-var-requires
const plugin = require('./no-raw-shadow.js') as {
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
        rules: { 'local/no-raw-shadow': 'error' },
        languageOptions: {
          ecmaVersion: 2022,
          sourceType: 'module',
          parserOptions: { ecmaFeatures: { jsx: true } },
        },
      },
    ],
    'component.tsx'
  );
  return messages.map((m) => m.ruleId ?? '');
}

describe('local/no-raw-shadow', () => {
  it('flags every raw Tailwind tier', () => {
    for (const tier of ['sm', 'md', 'lg', 'xl', '2xl', 'inner', 'none']) {
      expect(lint(`const c = "rounded shadow-${tier} p-2";`), tier).toEqual([
        'local/no-raw-shadow',
      ]);
    }
  });

  it('flags a raw shadow behind any variant prefix', () => {
    // An enumerated character class would miss the slash/bracket forms, which
    // is exactly how `lg:group-hover:shadow-2xl` survived the first census.
    expect(lint('const c = "hover:shadow-lg";')).toEqual(['local/no-raw-shadow']);
    expect(lint('const c = "lg:group-hover/card:shadow-2xl";')).toEqual(['local/no-raw-shadow']);
    expect(lint('const c = "group-data-[status=active]/tab:shadow-xl";')).toEqual([
      'local/no-raw-shadow',
    ]);
  });

  it('flags raw shadows inside template literals', () => {
    expect(lint('const c = `rounded ${x} shadow-xl`;')).toEqual(['local/no-raw-shadow']);
  });

  it('allows the token form — that IS the goal', () => {
    expect(lint('const c = "shadow-[var(--shadow-lg)]";')).toEqual([]);
    expect(lint('const c = "shadow-[var(--shadow-xl)]";')).toEqual([]);
  });

  it('leaves rings and drop-shadow alone', () => {
    // A ring is a border wearing another name; drop-shadow is an SVG/filter
    // concern. Neither is elevation, so neither is this rule's business.
    expect(lint('const c = "ring-2 ring-[var(--focus-ring)]";')).toEqual([]);
    expect(lint('const c = "drop-shadow-md";')).toEqual([]);
  });

  it('does not fire on unrelated words containing the prefix', () => {
    expect(lint('const c = "shadow-[var(--shadow-lg)] no-shadow-guard";')).toEqual([]);
  });

  it('is wired into eslint.config.mjs as an error', () => {
    const config = readFileSync(join(__dirname, '../../../../eslint.config.mjs'), 'utf8');
    expect(config).toContain(
      "import noRawShadow from './apps/web/src/eslint-rules/no-raw-shadow.js'"
    );
    expect(config).toContain('...noRawShadow.rules,');
    expect(config).toContain("'local/no-raw-shadow': 'error'");
  });
});
