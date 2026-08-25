import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createRequire } from 'node:module';
import { describe, it, expect } from 'vitest';
import { Linter } from 'eslint';

/**
 * local/no-emoji-in-ui — the vocabulary door-closer behind
 * fix-settings-graduation: emoji pictographs are hardcoded colour that
 * bypasses both the token rule and the contrast gate.
 *
 * Same testing strategy as no-hardcoded-palette.spec.ts: the RULE through
 * Linter with an inline flat config (vitest cannot import eslint.config.mjs
 * in every environment), the WIRING as config-file text.
 */
const require = createRequire(import.meta.url);
// eslint-disable-next-line @typescript-eslint/no-var-requires
const plugin = require('./no-emoji-in-ui.js') as {
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
        rules: { 'local/no-emoji-in-ui': 'error' },
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

describe('local/no-emoji-in-ui', () => {
  it.each([
    // The three real defects this rule closes the door on:
    ['the ApiKeysForm straggler', `setStatus('✅ 金鑰已儲存');`],
    ['the MetadataExport error prefix', `setMessage(\`⚠️ 匯出失敗：\${err}\`);`],
    ['the BackupScheduleConfig toggle', `setMessage('✅ 自動備份已啟用');`],
    // The wider glyph class:
    ['colour-by-default pictographs', `const icon = '🎬';`],
    ['emoji star ratings', `const r = '⭐ 8.1';`],
    ['VS16-forced colour on a text glyph', `const e = '✏️ 手動輸入';`],
    ['JSX text children', `const x = <p>✅ 解析完成！</p>;`],
  ])('flags %s', (_name, code) => {
    expect(lint(code)).toContain('local/no-emoji-in-ui');
  });

  it.each([
    // Text-presentation glyphs are typography, not colour smuggling —
    // LearnPatternPrompt's ✓, DoubanSection's ★☆, DownloadFilterTabs' ☰
    // must all stay legal.
    ['dingbat check/cross', `const s = '✓ 已套用你之前的設定'; const f = '✗';`],
    ['star-rating glyphs', `const s = '★★★☆☆';`],
    ['hamburger + bare warning sign', `const s = '☰'; const w = '⚠ 注意';`],
    ['ordinary zh-TW copy with punctuation', `const s = '匯出失敗：未知錯誤 — 請重試…';`],
    // Comments are not AST string nodes — the codebase documents old emoji
    // in comments (e.g.「in place of 🎬/📺 emoji」) and must not trip it.
    ['emoji in comments', `// replaced the ✅ with a lucide <Check>\nconst s = '已儲存';`],
  ])('does not flag %s', (_name, code) => {
    expect(lint(code)).toEqual([]);
  });

  describe('wiring in eslint.config.mjs (asserted as text — see header)', () => {
    const config = readFileSync(join(__dirname, '../../../../eslint.config.mjs'), 'utf8');

    it('imports the rule file', () => {
      expect(config).toContain('./apps/web/src/eslint-rules/no-emoji-in-ui.js');
    });

    it('spreads it into the shared local plugin object', () => {
      expect(config).toContain('...noEmojiInUi.rules,');
    });

    it('enables it at error severity over the settings surfaces', () => {
      expect(config).toMatch(/'local\/no-emoji-in-ui': 'error'/);
      const block = config.slice(config.indexOf("'local/no-emoji-in-ui'") - 700);
      expect(block).toContain('apps/web/src/components/settings/**/*.{ts,tsx}');
      expect(block).toContain('apps/web/src/routes/settings/**/*.{ts,tsx}');
    });
  });
});
