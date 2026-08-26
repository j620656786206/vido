/**
 * Custom ESLint rule — `local/no-hardcoded-duration`
 *
 * Closes the door behind the motion-vocabulary pass (2026-08-26). Before it,
 * this app had 459 motion sites and five different literal durations chosen one
 * component at a time — 150/200/300/500/700 — with no shared meaning, so
 * "routine state change" was 200ms in one file and 300ms in the next, and
 * nothing recorded which was right.
 *
 * styles.css now owns the vocabulary: --motion-touch (feedback),
 * --motion-state (a value changed), --motion-move (something travelled),
 * --motion-arrive / --motion-leave (the asymmetric pair). Those tokens are not
 * only a naming convention — they are the mechanism `prefers-reduced-motion`
 * turns down. A literal `duration-300` is invisible to that override and only
 * the blunt `*` net in styles.css catches it, which is why the tokens exist.
 *
 * Trigger: a Tailwind `duration-<number>` or `duration-[<time>]` utility (with
 * any variant prefix) in any string Literal or TemplateElement.
 *
 * NOT in scope, deliberately:
 *  - `transition-colors` with no duration at all. Tailwind's 150ms default is
 *    close enough to --motion-touch that sweeping 268 call sites to say so
 *    would be churn, not correctness.
 *  - `delay-*`, and animation durations declared in CSS, which styles.css owns
 *    directly.
 *
 * Escape hatch: an explicit eslint-disable line with a reason. A duration that
 * genuinely belongs to one component — an auto-dismiss timer, say, which is a
 * DEADLINE and not motion — is a real case, and it should say so out loud
 * rather than hide in an allowlist.
 *
 * No auto-fix: mapping 300ms to --motion-state vs --motion-move is a judgement
 * about what the transition MEANS, and a blanket rewrite would silently pick
 * wrong.
 *
 * Scoping lives in eslint.config.mjs, mirroring the sibling local rules.
 */

// `duration-300` / `duration-[250ms]`, with any variant prefix. The prefix is
// matched as "anything up to a colon" for the same reason the sibling rule
// does it: an enumerated character class misses `group-data-[status=active]/x:`.
const DURATION = /(?:^|[\s"'`])(?:[^\s"'`]*:)?duration-(\[[^\]]*\]|\d+)/g;

/** A `duration-[…]` arbitrary value already reaching for a token is the goal. */
const isToken = (v) => v.startsWith('[') && v.includes('var(--');

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description: 'Use a --motion-* duration token, not a literal Tailwind duration.',
    },
    schema: [],
    messages: {
      literalDuration:
        '`duration-{{value}}` is a literal; the motion vocabulary in styles.css owns durations. Use `duration-[var(--motion-touch)]` for hover/press/focus feedback, `--motion-state` when a value changed, `--motion-move` when something travelled, or the `--motion-arrive`/`--motion-leave` pair for a cross-fade. Literals are also invisible to the prefers-reduced-motion token override. If this is a deadline rather than motion (an auto-dismiss timer), add an eslint-disable line saying so.',
    },
  },

  create(context) {
    function check(node, text) {
      if (typeof text !== 'string' || !text.includes('duration-')) return;
      DURATION.lastIndex = 0;
      let m;
      while ((m = DURATION.exec(text)) !== null) {
        if (isToken(m[1])) continue;
        context.report({ node, messageId: 'literalDuration', data: { value: m[1] } });
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

module.exports = { rules: { 'no-hardcoded-duration': rule } };
