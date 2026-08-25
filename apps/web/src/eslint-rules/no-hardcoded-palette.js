/**
 * Custom ESLint rule — `local/no-hardcoded-palette`
 *
 * Closes the door behind `disc-2026-08-settings-token-bypass-retrofit`
 * (PRs #291/#292/#294/#295): 231 hardcoded Tailwind palette literals were
 * migrated to the semantic tokens in styles.css, and this rule keeps the
 * count at zero — exemption-free by construction, because nothing is left
 * to exempt.
 *
 * Why a rule and not review vigilance: the four migration slices each
 * uncovered the same defect class (a family's base colour used as TEXT is
 * sub-AA on its own tint — --error-text 4.05:1, --accent-text 3.52:1,
 * --warning-text 4.31:1, --success-text 4.04:1). Literals bypass both the
 * vocabulary AND the styles-contrast.spec.ts gate that now guards it, so a
 * single new `text-blue-300` reopens the whole class.
 *
 * Trigger: any string Literal or TemplateElement whose text contains a
 * Tailwind palette utility — `{prefix}-{hue}-{50..900}` with an optional
 * `/alpha` — for the colour prefixes (bg/text/border/ring/from/to/via/
 * fill/stroke/outline/decoration/divide/accent/caret/placeholder/shadow).
 * Comments never trip it (they are not AST string nodes). Spacing-scale
 * utilities (`p-4`, `gap-2`, `w-60`) never match — the hue segment is
 * required.
 *
 * NOT in scope: `bg-[var(--…)]` arbitrary values (that IS the vocabulary),
 * and non-colour Tailwind classes. No auto-fix: the right token depends on
 * ROLE (text vs fill vs tint vs border), which the four slices resolved
 * case by case — a mechanical hue→token map would re-create the sub-AA
 * bugs the migration just removed.
 *
 * Scoping lives in eslint.config.mjs (files/ignores of the enabling block),
 * mirroring local/implements-pen-node-id and
 * local/time-dependent-fixture-stability.
 */

const PALETTE =
  /\b(?:bg|text|border|ring|from|to|via|fill|stroke|outline|decoration|divide|accent|caret|placeholder|shadow)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-(?:50|[1-9]00)(?:\/\d{1,3})?\b/;

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Ban hardcoded Tailwind palette colour literals in favour of the semantic tokens in styles.css',
    },
    schema: [],
    messages: {
      hardcodedPalette:
        'Hardcoded Tailwind palette colour "{{match}}" — use the semantic tokens in styles.css ' +
        '(--error/--success/--warning/--info/--accent families; *-text for readable text, *-tint ' +
        'for surfaces, *-pressed for button hover). The token vocabulary is gated for WCAG AA by ' +
        'styles-contrast.spec.ts; literals bypass that gate.',
    },
  },
  create(context) {
    const check = (node, text) => {
      const m = PALETTE.exec(text);
      if (m) {
        context.report({ node, messageId: 'hardcodedPalette', data: { match: m[0] } });
      }
    };
    return {
      Literal(node) {
        if (typeof node.value === 'string') check(node, node.value);
      },
      TemplateElement(node) {
        check(node, node.value.raw);
      },
    };
  },
};

// Plugin-shaped CJS export, matching the sibling rule files: eslint.config.mjs
// spreads each file's `.rules` into the shared `local` plugin namespace.
module.exports = { rules: { 'no-hardcoded-palette': rule } };
