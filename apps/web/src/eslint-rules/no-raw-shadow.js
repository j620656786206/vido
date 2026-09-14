/**
 * Custom ESLint rule — `local/no-raw-shadow`
 *
 * Closes the door behind the elevation pass (dsr-9, 2026-09-14). Before it, the
 * app wrote shadows in two dialects and the wrong one was winning 4.6 to 1:
 * 46 raw Tailwind utilities (`shadow-lg`, `shadow-xl`, `shadow-2xl`) against
 * 10 `shadow-[var(--shadow-*)]` tokens.
 *
 * That is not a style preference. The two themes carry two different shadow
 * MATERIALS — 夜行 uses coal black (alpha .3–.6, single layer), 日巡 uses ink
 * (a low-alpha --text-primary, two layers from md up, total ink ≤15.4%),
 * because 30–60% black on warm paper reads as a bruise. That rework landed in
 * `styles.css` on the `--shadow-*` tokens only. A raw Tailwind utility is a
 * hardcoded black rgba that does not know the light theme exists, so all 42
 * legitimately-floating surfaces were painting bruises in 日巡 — the exact
 * thing DESIGN.md §Elevation says to avoid.
 *
 * Trigger: a Tailwind `shadow-<tier>` utility (sm/md/lg/xl/2xl/inner/none) with
 * any variant prefix, in any string Literal or TemplateElement.
 *
 * NOT in scope, deliberately:
 *  - `shadow-[var(--shadow-lg)]` and friends — that IS the goal.
 *  - `ring-*` and `drop-shadow-*`. Rings are borders wearing a different name
 *    and drop-shadow is an SVG/filter concern; neither is elevation.
 *
 * Escape hatch: an explicit eslint-disable line with a reason.
 *
 * No auto-fix. `shadow-2xl` maps to `--shadow-xl` rather than a 2xl token
 * (DESIGN.md caps the scale at xl on purpose), and choosing lg vs xl is a
 * judgement about whether the surface sits on the content or covers it — a
 * blanket rewrite would silently pick wrong.
 *
 * Scoping lives in eslint.config.mjs, mirroring the sibling local rules.
 */

// `shadow-lg` / `dark:shadow-xl` / `lg:group-hover:shadow-2xl`. The prefix is
// matched as "anything up to a colon" for the same reason the sibling rules do
// it: an enumerated character class misses `group-data-[status=active]/x:`.
const SHADOW = /(?:^|[\s"'`])(?:[^\s"'`]*:)?shadow-(sm|md|lg|xl|2xl|inner|none)(?![\w-])/g;

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description: 'Use a --shadow-* elevation token, not a raw Tailwind shadow utility.',
    },
    schema: [],
    messages: {
      rawShadow:
        '`shadow-{{value}}` is a raw Tailwind shadow: a hardcoded black rgba that does not change with the theme, so it paints a bruise in 日巡 (the ink/two-layer rework landed on the --shadow-* tokens only). Use `shadow-[var(--shadow-lg)]` for a surface that floats ON the content (dropdown, suggestion list, hover-revealed control) or `shadow-[var(--shadow-xl)]` for one that COVERS it (Dialog, Sheet, side panel, modal). ⚖️ 2026-09-14: buttons and poster tiles carry NO shadow — if this is one of those, delete the utility instead of converting it. See DESIGN.md §Shadow Vocabulary.',
    },
  },

  create(context) {
    function check(node, text) {
      if (typeof text !== 'string' || !text.includes('shadow-')) return;
      SHADOW.lastIndex = 0;
      let m;
      while ((m = SHADOW.exec(text)) !== null) {
        context.report({ node, messageId: 'rawShadow', data: { value: m[1] } });
      }
    }

    return {
      Literal(node) {
        check(node, node.value);
      },
      TemplateElement(node) {
        check(node, node.value?.cooked ?? node.value?.raw);
      },
    };
  },
};

module.exports = { rules: { 'no-raw-shadow': rule } };
