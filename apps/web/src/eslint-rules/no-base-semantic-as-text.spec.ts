import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createRequire } from 'node:module';
import { describe, it, expect } from 'vitest';
import { Linter } from 'eslint';

/**
 * local/no-base-semantic-as-text — the OTHER half of the token door.
 *
 * `no-hardcoded-palette` catches `text-red-400`; this catches
 * `text-[var(--error)]`, which is the same sub-AA defect expressed INSIDE the
 * token vocabulary — where 266 sites across 98 files had quietly re-created it
 * and neither that rule nor styles-contrast.spec.ts could see them.
 *
 * Same testing shape as the sibling spec (and for the same reason): the RULE
 * runs through eslint's Linter with an inline flat config, and the WIRING is
 * asserted against the config file's text, because the config-importing specs
 * are not loadable in every local environment.
 */
const require = createRequire(import.meta.url);
// eslint-disable-next-line @typescript-eslint/no-var-requires
const plugin = require('./no-base-semantic-as-text.js') as {
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
        rules: { 'local/no-base-semantic-as-text': 'error' },
        languageOptions: { ecmaVersion: 2022, sourceType: 'module' },
      },
    ],
    'component.tsx'
  );
  return messages.map((m) => m.ruleId ?? '');
}

function messages(code: string): string[] {
  const linter = new Linter({ configType: 'flat' });
  return linter
    .verify(
      code,
      [
        {
          files: ['**/*.tsx'],
          plugins: { local: plugin },
          rules: { 'local/no-base-semantic-as-text': 'error' },
          languageOptions: { ecmaVersion: 2022, sourceType: 'module' },
        },
      ],
      'component.tsx'
    )
    .map((m) => m.message);
}

describe('local/no-base-semantic-as-text', () => {
  it.each([
    ['--error as text', `const c = "text-[var(--error)]";`],
    ['--warning as text', `const c = "flex text-[var(--warning)] gap-2";`],
    ['--success as text', `const c = "text-[var(--success)]";`],
    ['--info as text', `const c = "text-[var(--info)]";`],
    ['--accent-primary as text', `const c = "text-[var(--accent-primary)]";`],
    // #e0be72 either way, so this one also produces a hover that does nothing.
    ['--accent-hover as text', `const c = "hover:text-[var(--accent-hover)]";`],
    ['behind a variant prefix', `const c = "hover:text-[var(--error)]";`],
    [
      'behind a group-data variant',
      `const c = "group-data-[status=active]/parent:text-[var(--accent-primary)]";`,
    ],
    ['in a template literal', 'const c = `flex ${x} text-[var(--success)]`;'],
  ])('flags %s', (_name, code) => {
    expect(lint(code)).toContain('local/no-base-semantic-as-text');
  });

  it.each([
    // The twins are the whole point — they must never be flagged.
    ['the *-text twins', `const c = "text-[var(--error-text)] text-[var(--accent-text)]";`],
    // Base tokens are FILL colours; every non-text utility is their job.
    ['base tokens as a background', `const c = "bg-[var(--error)] text-white";`],
    ['base tokens as a fill', `const c = "fill-[var(--warning)] stroke-[var(--success)]";`],
    ['base tokens as a border or ring', `const c = "border-[var(--error)] ring-[var(--info)]";`],
    // Tints and pressed states have no text twin and no text role.
    ['tint tokens', `const c = "bg-[var(--error-tint)] text-[var(--error-text)]";`],
    ['pressed states', `const c = "active:bg-[var(--accent-pressed)]";`],
    ['neutral text tokens', `const c = "text-[var(--text-primary)] text-[var(--text-muted)]";`],
    // Comments are not AST string nodes; the sweep left explanatory comments
    // naming the old tokens and they must not trip the rule.
    [
      'token names in comments',
      `// was text-[var(--error)]\nconst c = "text-[var(--error-text)]";`,
    ],
  ])('does not flag %s', (_name, code) => {
    expect(lint(code)).toEqual([]);
  });

  it('names the twin to use, so the fix does not need a lookup', () => {
    expect(messages(`const c = "text-[var(--warning)]";`)[0]).toContain('--warning-text');
  });

  it('flags every base token on one line rather than stopping at the first', () => {
    // A status map often paints several tones in one object literal; reporting
    // only the first would hide the rest behind a single fix.
    const code = `const c = "text-[var(--error)]"; const d = "text-[var(--success)]";`;
    expect(lint(code)).toHaveLength(2);
  });

  describe('wiring in eslint.config.mjs (asserted as text — see header)', () => {
    const config = readFileSync(join(__dirname, '../../../../eslint.config.mjs'), 'utf8');

    it('imports the rule file', () => {
      expect(config).toContain('./apps/web/src/eslint-rules/no-base-semantic-as-text.js');
    });

    it('spreads it into the shared local plugin object', () => {
      expect(config).toContain('...noBaseSemanticAsText.rules,');
    });

    it('enables it at error severity over components AND routes', () => {
      expect(config).toMatch(/'local\/no-base-semantic-as-text': 'error'/);
      const block = config.slice(config.indexOf("'local/no-base-semantic-as-text'") - 900);
      expect(block).toContain('apps/web/src/routes/**/*.{ts,tsx}');
    });

    it('is exemption-free: nothing turns it off', () => {
      // The sweep moved all 266 sites, so — like its sibling — there is
      // nothing left to allowlist.
      expect(config).not.toContain('no-base-semantic-as-text": "off');
      expect(config).not.toContain("no-base-semantic-as-text': 'off");
    });
  });
});
