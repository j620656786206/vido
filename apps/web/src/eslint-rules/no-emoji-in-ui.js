/**
 * Custom ESLint rule — `local/no-emoji-in-ui`
 *
 * Closes the door behind fix-settings-graduation (critique R4): the last two
 * 固定詞彙 stragglers (ApiKeysForm's「✅ 金鑰已儲存」, MetadataExport's
 * ✅/⚠️-prefixed result strings) plus a grep-found third
 * (BackupScheduleConfig's two ✅) were all the SAME defect wearing the same
 * glyph. Emoji pictographs escape both enforcement layers at once: they are
 * not tokens (so `local/no-hardcoded-palette` never sees them) and they are
 * not CSS colours (so styles-contrast.spec.ts never measures them) — a ✅
 * is a hardcoded green that no gate can catch. lucide icons + the token
 * vocabulary are the sanctioned channel for every such glyph.
 *
 * Trigger: any string Literal, TemplateElement, or JSXText containing
 *   - a code point with `Emoji_Presentation` (colour-by-default: ✅ ❌ ⭐
 *     🔴 ⏳ and all U+1F300+ pictographs), or
 *   - U+FE0F VARIATION SELECTOR-16 (forces colour presentation onto a text
 *     glyph: ⚠️ ✏️).
 * Text-presentation glyphs (✓ ✗ ★ ☆ ☰ bare ⚠) are typography, not colour
 * smuggling — deliberately NOT matched. Comments never trip it (not AST
 * string nodes).
 *
 * Scoping lives in eslint.config.mjs: settings/** only for now —
 * exemption-free there by construction. The repo-wide sweep (~20 files in
 * media/notifications/parse still carry 🎬 ✅ 📁 …) is filed as
 * disc-2026-08-emoji-sweep-repo-wide; widen the files glob when it lands.
 */

const EMOJI = /\p{Emoji_Presentation}|️/u;

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Ban emoji pictographs in UI strings — they smuggle hardcoded colour past both the token vocabulary and the contrast gate',
    },
    schema: [],
    messages: {
      emojiInUi:
        'Emoji "{{match}}" in a UI string — emoji are hardcoded colour that bypasses both ' +
        'styles.css tokens and the styles-contrast.spec.ts gate. Use a lucide icon coloured ' +
        'with the semantic tokens (and remember 固定詞彙: completion wears neutral, not green).',
    },
  },
  create(context) {
    const check = (node, text) => {
      const m = EMOJI.exec(text);
      if (m) {
        context.report({ node, messageId: 'emojiInUi', data: { match: m[0] } });
      }
    };
    return {
      Literal(node) {
        if (typeof node.value === 'string') check(node, node.value);
      },
      TemplateElement(node) {
        check(node, node.value.raw);
      },
      JSXText(node) {
        check(node, node.value);
      },
    };
  },
};

// Plugin-shaped CJS export, matching the sibling rule files: eslint.config.mjs
// spreads each file's `.rules` into the shared `local` plugin namespace.
module.exports = { rules: { 'no-emoji-in-ui': rule } };
