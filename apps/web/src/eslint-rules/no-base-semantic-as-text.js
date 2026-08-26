/**
 * Custom ESLint rule — `local/no-base-semantic-as-text`
 *
 * Closes the door behind the base-semantic-as-text sweep (2026-08-26). Every
 * semantic colour in styles.css ships as a PAIR: a base token that is a FILL
 * colour, and a `*-text` twin that is the AA-safe TEXT colour. The twins exist
 * because four separate token-debt PRs measured the base ones failing as text
 * (--error 4.05:1, --accent 3.52:1, --warning 4.31:1, --success 4.04:1 on their
 * own tints). styles.css says so on each token.
 *
 * The sibling rule `local/no-hardcoded-palette` catches `text-red-400`. It
 * cannot catch `text-[var(--error)]` — that IS the vocabulary, just the wrong
 * half of it — so 266 sites across 98 files re-created the defect INSIDE the
 * token system, where neither that rule nor styles-contrast.spec.ts looks. This
 * rule is the missing half.
 *
 * Trigger: a `text-[var(--x)]` utility (including variant prefixes like
 * `hover:` / `group-data-[…]:`) naming a BASE semantic token, in any string
 * Literal or TemplateElement.
 *
 * NOT in scope, deliberately:
 *  - every non-`text-` utility: `bg-[var(--error)]`, `fill-`, `border-`, `ring-`
 *    are what the base tokens are FOR.
 *  - `--*-tint` and `--*-pressed`, which have no text twin and no text role.
 *
 * The ONE legitimate reason to write a base token as a text colour is an
 * icon-only glyph, which owes WCAG 1.4.11's 3:1 rather than 4.5:1. That is a
 * per-site judgement (and it does NOT rescue cinnabar: --error measures
 * 2.42–3.00:1 as an icon, under even the non-text floor), so the escape hatch
 * is an explicit eslint-disable line with a reason — not a silent allowlist.
 *
 * No auto-fix: choosing the twin is mechanical, but deciding whether the site
 * is text at all is not, and a blanket fix would repaint icons that are fine.
 *
 * Scoping lives in eslint.config.mjs, mirroring the sibling local rules.
 */

/** Base semantic tokens that own an AA-safe `*-text` twin in styles.css. */
const BASE_TO_TWIN = {
  '--error': '--error-text',
  '--warning': '--warning-text',
  '--success': '--success-text',
  '--info': '--info-text',
  '--accent-primary': '--accent-text',
  // --accent-hover is byte-identical to --accent-text (#e0be72), so writing it
  // as a text colour beside --accent-text produces a hover that does nothing.
  '--accent-hover': '--accent-text',
};

// `text-[var(--token)]` with any variant prefix. The prefix is matched as
// "anything up to a colon" ON PURPOSE: an enumerated character class missed
// `group-data-[status=active]/parent:` (the `/` and `=`), which is exactly the
// site the sweep's reviewer had to find by hand because lint stayed silent.
const TEXT_TOKEN = /(?:^|[\s"'`])(?:[^\s"'`]*:)?text-\[var\((--[a-z-]+)\)\]/g;

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description: 'Use the AA-safe *-text twin, not the base semantic token, as a text colour.',
    },
    schema: [],
    messages: {
      baseAsText:
        "`{{base}}` is a FILL colour; as text it is sub-AA (that is why `{{twin}}` exists — see the token's comment in styles.css). Use `text-[var({{twin}})]`. If this is an icon-only glyph owing WCAG 1.4.11's 3:1 rather than 4.5:1, add an eslint-disable line saying so — but note cinnabar `--error` measures 2.42–3.00:1 and fails even that floor.",
    },
  },

  create(context) {
    function check(node, text) {
      if (typeof text !== 'string' || !text.includes('text-[var(')) return;
      TEXT_TOKEN.lastIndex = 0;
      let m;
      while ((m = TEXT_TOKEN.exec(text)) !== null) {
        const twin = BASE_TO_TWIN[m[1]];
        if (twin) {
          context.report({
            node,
            messageId: 'baseAsText',
            data: { base: m[1], twin },
          });
        }
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

// Plugin-shaped CJS export, matching the sibling rule files: eslint.config.mjs
// spreads each file's `.rules` into the shared `local` plugin namespace.
module.exports = { rules: { 'no-base-semantic-as-text': rule } };
