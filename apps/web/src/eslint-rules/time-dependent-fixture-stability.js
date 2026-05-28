/**
 * Custom ESLint rule — `local/time-dependent-fixture-stability`
 *
 * Enforces project-context.md Rule 23 (Time-Dependent Component Fixture
 * Stability). Story 19-9 [@contract-v1].
 *
 * Every file under `apps/web/src/components/` whose source body contains an
 * ambient wall-clock read MUST carry a leading-comment header declaring how
 * the time-bomb is neutralised. Trigger AST shapes (criterion (a)):
 *
 *   - `MemberExpression` matching `Date.now` / `Date.UTC` / `Date.parse`
 *   - `NewExpression` with `Date` callee and zero arguments (`new Date()`)
 *
 * `new Date(arg)` with arguments is NOT in scope — it is a deterministic
 * formatter (output is a pure function of input). The rule's AST visitor
 * relies on the parser's `arguments.length` check, robust against string-
 * literal "Date.now" mentions in comments or strings.
 *
 * Accepted Rule 23 marker forms (leading comment, BEFORE the first
 * non-comment statement — same definition as the 19-3 rule's "leading
 * comment"):
 *
 *   // Clock-mocked: gallery fixture {fixture-id} uses page.clock.setFixedTime
 *   // Clock-injected: component accepts `clock` prop; no fixture-side mock needed
 *   // Time-bomb-exempt: <one-line rationale>
 *
 * Rule 23 markers coexist with Rule 21 (19-3) markers — two-line header
 * convention, Rule 23 marker FIRST, Rule 21 marker SECOND. The 19-3 rule
 * sees the Rule 23 line as an extra leading comment and ignores it; this
 * rule sees the Rule 21 line as an extra leading comment and ignores it.
 *
 * Scoping (which files this rule runs on) is done in `eslint.config.mjs`
 * via the flat-config `files`/`ignores` of the config object that enables
 * this rule — NOT inside the rule. Same scoping as the 19-3 rule:
 * `apps/web/src/components/**\/*.{ts,tsx}` excluding `*.spec.*` +
 * `index.ts` barrels.
 *
 * No auto-fix is provided: the correct marker depends on the dev's
 * intent (migrate vs inject vs exempt) — a one-line `Time-bomb-exempt`
 * rationale is not safely synthesisable.
 *
 * @see project-context.md Rule 23
 * @see _bmad-output/audit/time-bomb-fixtures-2026-05.md
 */

'use strict';

// `// Clock-mocked: gallery fixture {kebab-id} uses page.clock.setFixedTime`
// Accept any kebab/path-shaped fixture id (allow `/` for state-suffixed forms
// like `library-recently-added/recent`).
const CLOCK_MOCKED_RE =
  /^Clock-mocked:\s+gallery fixture [a-z0-9][a-z0-9-/]*\s+uses page\.clock\.setFixedTime$/i;

// `// Clock-injected: component accepts `clock` prop; no fixture-side mock needed`
// Accept some prose flexibility (trailing rationale after the canonical phrase).
const CLOCK_INJECTED_RE = /^Clock-injected:\s+component accepts `?clock`?\s+prop(.+)?$/i;

// `// Time-bomb-exempt: <one-line rationale>` — rationale must be non-empty.
const TIME_BOMB_EXEMPT_RE = /^Time-bomb-exempt:\s+\S.+$/i;

/**
 * Split a comment into candidate marker lines, normalising both `//` line
 * comments and JSDoc-style block comments (strip a leading `*`).
 * Mirrors the 19-3 rule's `commentLines` to keep header conventions
 * symmetric.
 * @param {{type: string, value: string}} comment
 * @returns {string[]}
 */
function commentLines(comment) {
  return comment.value.split('\n').map((line) => line.replace(/^\s*\*?\s?/, '').trim());
}

function isAcceptedMarkerLine(line) {
  return (
    CLOCK_MOCKED_RE.test(line) || CLOCK_INJECTED_RE.test(line) || TIME_BOMB_EXEMPT_RE.test(line)
  );
}

/** @type {import('eslint').Rule.RuleModule} */
const timeDependentFixtureStability = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Enforce project-context.md Rule 23: files under apps/web/src/components/ that read the wall clock (Date.now / Date.UTC / Date.parse / new Date()) must carry a leading Clock-mocked / Clock-injected / Time-bomb-exempt header',
      recommended: false,
    },
    schema: [],
    messages: {
      'time-bomb-detected':
        'Rule 23 (project-context.md): this component reads the wall clock (ambient `Date.now()` / `Date.UTC()` / `Date.parse()` / `new Date()`) but carries no Rule 23 marker. Add a leading comment BEFORE the first statement, one of: `// Clock-mocked: gallery fixture {fixture-id} uses page.clock.setFixedTime` (canonical — pair with a `clockTime` fixture field per story 19-9 AC #4), `// Clock-injected: component accepts `clock` prop; no fixture-side mock needed`, or `// Time-bomb-exempt: <one-line rationale>` (last resort; rationale must identify the reviewer — Sally for visual-state calls, Murat for test-architecture calls). See `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.',
    },
  },

  create(context) {
    const sourceCode = context.sourceCode || context.getSourceCode();
    let hasTimeBomb = false;

    return {
      // AST trigger (a) — bare `Date.now` / `Date.UTC` / `Date.parse` references.
      MemberExpression(node) {
        if (
          node.object.type === 'Identifier' &&
          node.object.name === 'Date' &&
          node.property.type === 'Identifier' &&
          (node.property.name === 'now' ||
            node.property.name === 'UTC' ||
            node.property.name === 'parse')
        ) {
          hasTimeBomb = true;
        }
      },
      // AST trigger (b) — `new Date()` with zero arguments only. `new Date(arg)`
      // with any argument is a deterministic formatter, NOT a time bomb.
      NewExpression(node) {
        if (
          node.callee.type === 'Identifier' &&
          node.callee.name === 'Date' &&
          node.arguments.length === 0
        ) {
          hasTimeBomb = true;
        }
      },
      'Program:exit'(node) {
        if (!hasTimeBomb) {
          return;
        }

        const firstStatement = node.body[0];
        const allComments = sourceCode.getAllComments();
        const leadingComments = firstStatement
          ? allComments.filter((c) => c.range[1] <= firstStatement.range[0])
          : allComments;

        const hasMarker = leadingComments.some((comment) =>
          commentLines(comment).some(isAcceptedMarkerLine)
        );

        if (!hasMarker) {
          context.report({ node, loc: { line: 1, column: 0 }, messageId: 'time-bomb-detected' });
        }
      },
    };
  },
};

module.exports = {
  rules: {
    'time-dependent-fixture-stability': timeDependentFixtureStability,
  },
};
