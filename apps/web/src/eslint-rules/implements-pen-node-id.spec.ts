/**
 * Tests for the custom ESLint rule `local/implements-pen-node-id` (story 19-3).
 *
 * Runs under `pnpm nx test web` (vitest picks up `src/**\/*.spec.ts`). ESLint's
 * `RuleTester` uses the ambient `describe`/`it` globals, which vitest provides
 * (`globals: true` in vite.config.ts).
 */
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { RuleTester } from 'eslint';
import { describe, it, expect } from 'vitest';
import plugin from './implements-pen-node-id.js';

const rule = plugin.rules['implements-pen-node-id'];

const ruleTester = new RuleTester({
  languageOptions: { ecmaVersion: 2022, sourceType: 'module' },
});

ruleTester.run('implements-pen-node-id', rule, {
  valid: [
    // (a) valid single-component header
    {
      code: '// Implements: Component/PosterCard (RusTY)\nexport const x = 1;\n',
    },
    // (b) valid multi-component header (` + `-joined)
    {
      code: '// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)\nexport const x = 1;\n',
    },
    // (c) exemption: utility (em-dash form, as documented in Rule 21)
    {
      code: '// Implements: <utility — no .pen counterpart>\nexport const x = 1;\n',
    },
    // (c') exemption: utility (plain-hyphen tolerated)
    {
      code: '// Implements: <utility - no .pen counterpart>\nexport const x = 1;\n',
    },
    // (c'') exemption: route-only wrapper
    {
      code: '// Implements: <route-only>\nexport const x = 1;\n',
    },
    // (c''') Phase-2 placeholder: screen-section (em-dash, as documented)
    {
      code: '// Implements: <screen-section — pending epic-19-8 mapping>\nexport const x = 1;\n',
    },
    // (c'''') screen-section placeholder with plain hyphen tolerated
    {
      code: '// Implements: <screen-section - pending epic-19-8 mapping>\nexport const x = 1;\n',
    },
    // header may sit below other leading comments / above imports
    {
      code: '/* eslint-disable */\n// Implements: Component/Foo (abc123)\nimport { y } from "z";\nexport const x = y;\n',
    },
    // block-comment / JSDoc form is tolerated
    {
      code: '/**\n * Implements: Component/Foo (abc123)\n */\nexport const x = 1;\n',
    },
  ],
  invalid: [
    // (d) no header at all
    {
      code: 'export const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (e) malformed: no `Component/` prefix
    {
      code: '// Implements: PosterCard\nexport const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (e') malformed: `Component/Name` but no `(nodeId)`
    {
      code: '// Implements: Component/PosterCard\nexport const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (e'') malformed: empty node id
    {
      code: '// Implements: Component/PosterCard ()\nexport const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (e''') malformed exemption text
    {
      code: '// Implements: <some other reason>\nexport const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (e'''') screen-section placeholder missing the `pending epic-N-M mapping` clause
    {
      code: '// Implements: <screen-section>\nexport const x = 1;\n',
      errors: [{ messageId: 'missing' }],
    },
    // (f) a correctly-shaped marker that appears AFTER the first statement is NOT a leading comment
    {
      code: 'import { y } from "z";\n// Implements: Component/Foo (abc123)\nexport const x = y;\n',
      errors: [{ messageId: 'missing' }],
    },
  ],
});

// (g) Out-of-scope files (specs, hooks/services/stores/utils, route files,
// index.ts barrels) are excluded by the flat-config `files`/`ignores` of the
// config object in eslint.config.mjs, not by the rule body. Verify that wiring
// is present so a refactor can't silently widen/narrow scope.
describe('eslint.config.mjs wiring for local/implements-pen-node-id', () => {
  const configPath = resolve(__dirname, '../../../../eslint.config.mjs');
  const configText = readFileSync(configPath, 'utf8');

  it('registers the rule at error severity', () => {
    expect(configText).toContain("'local/implements-pen-node-id': 'error'");
  });

  it('scopes the rule to apps/web/src/components/**/*.{ts,tsx}', () => {
    expect(configText).toContain('apps/web/src/components/**/*.{ts,tsx}');
  });

  it('ignores spec/test files and index.ts barrels under components/', () => {
    expect(configText).toMatch(/apps\/web\/src\/components\/\*\*\/\*\.spec\.\{ts,tsx\}/);
    expect(configText).toMatch(/apps\/web\/src\/components\/\*\*\/\*\.test\.\{ts,tsx\}/);
    expect(configText).toContain('apps/web/src/components/**/index.ts');
  });
});
