/**
 * Custom ESLint rule — `local/implements-pen-node-id`
 *
 * Enforces project-context.md Rule 21 (Component-to-Design Node Traceability),
 * Phase 2 (machine enforcement). Story 19-3.
 *
 * Every file under `apps/web/src/components/` that renders a designed UI element
 * MUST carry a leading comment that links it back to its `.pen` design node:
 *
 *   // Implements: Component/{Name} ({penNodeId})
 *   // Implements: Component/{Name} ({penNodeId}) + Component/{Name2} ({penNodeId2})   // multi
 *   // Implements: <utility — no .pen counterpart>                                      // exemption
 *   // Implements: <route-only>                                                         // exemption
 *   // Implements: <screen-section — pending epic-19-8 mapping>                          // Phase-2 placeholder
 *
 * "Leading comment" = a comment that appears before the first non-comment,
 * non-whitespace token of the file (i.e. before the first import/export/statement).
 * A correctly-shaped marker placed after the first statement does NOT satisfy
 * the rule (AC #2 / AC #7f of story 19-3).
 *
 * Scoping (which files this rule runs on) is done in `eslint.config.mjs` via the
 * flat-config `files`/`ignores` of the config object that enables this rule — NOT
 * inside the rule. Tests/specs, `index.ts` barrels, hooks/services/stores/utils,
 * and route files are out of scope by that config, not by logic here.
 *
 * No auto-fix is provided: a `.pen` node ID cannot be invented — look it up via
 * the Pencil MCP `get_editor_state` tool ("Reusable Components" listing).
 */

'use strict';

// `// Implements: Component/Foo (abc123)` optionally `+ Component/Bar (def456)` ...
// {Name} allows letters/digits/`-` (e.g. `TechBadge-Video`); {nodeId} is letters/digits only
// (real .pen node IDs — `RusTY`, `MQbvp`, `U3SGxG` — never contain `-`). See AC #1.
const COMPONENT_MARKER_RE =
  /^Implements:\s*Component\/[A-Za-z0-9-]+\s*\([A-Za-z0-9]+\)(\s*\+\s*Component\/[A-Za-z0-9-]+\s*\([A-Za-z0-9]+\))*$/;
// `// Implements: <utility — no .pen counterpart>` (accept em-dash or hyphen)
const UTILITY_EXEMPTION_RE = /^Implements:\s*<utility\s*[—–-]\s*no \.pen counterpart>$/;
// `// Implements: <route-only>`
const ROUTE_EXEMPTION_RE = /^Implements:\s*<route-only>$/;
// `// Implements: <screen-section — pending epic-19-8 mapping>` — Phase-2 placeholder for a
// component that renders a section of a designed *screen frame* (not a Reusable Component);
// canonical screen-frame mapping is tracked by epic-19-8. Accept em-dash or hyphen.
const SCREEN_SECTION_PLACEHOLDER_RE =
  /^Implements:\s*<screen-section\s*[—–-]\s*pending epic-[0-9]+-[0-9]+ mapping>$/;

/**
 * Split a comment into candidate marker lines, normalising both `//` line
 * comments and `/* *\/` / JSDoc-style block comments (strip a leading `*`).
 * @param {{type: string, value: string}} comment
 * @returns {string[]}
 */
function commentLines(comment) {
  return comment.value.split('\n').map((line) => line.replace(/^\s*\*?\s?/, '').trim());
}

function isAcceptedMarkerLine(line) {
  return (
    COMPONENT_MARKER_RE.test(line) ||
    UTILITY_EXEMPTION_RE.test(line) ||
    ROUTE_EXEMPTION_RE.test(line) ||
    SCREEN_SECTION_PLACEHOLDER_RE.test(line)
  );
}

/** @type {import('eslint').Rule.RuleModule} */
const implementsPenNodeId = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Enforce project-context.md Rule 21: files under apps/web/src/components/ must carry a leading `// Implements: Component/{Name} ({penNodeId})` header (or a documented exemption)',
      recommended: false,
    },
    schema: [],
    messages: {
      missing:
        'Rule 21 (project-context.md): missing a leading `// Implements: Component/{Name} ({penNodeId})` header linking this component to its .pen design node. Acceptable forms: `// Implements: Component/Foo (nodeId)`, `// Implements: Component/Foo (id) + Component/Bar (id)`, the exemptions `// Implements: <utility — no .pen counterpart>` / `// Implements: <route-only>`, or the Phase-2 placeholder `// Implements: <screen-section — pending epic-19-8 mapping>` (for a component that renders a section of a designed screen frame). Look up the node ID via the Pencil MCP `get_editor_state` tool → "Reusable Components".',
    },
  },

  create(context) {
    const sourceCode = context.sourceCode || context.getSourceCode();
    return {
      Program(node) {
        const firstStatement = node.body[0];
        const allComments = sourceCode.getAllComments();
        const leadingComments = firstStatement
          ? allComments.filter((c) => c.range[1] <= firstStatement.range[0])
          : allComments;

        const hasMarker = leadingComments.some((comment) =>
          commentLines(comment).some(isAcceptedMarkerLine)
        );

        if (!hasMarker) {
          context.report({ node, loc: { line: 1, column: 0 }, messageId: 'missing' });
        }
      },
    };
  },
};

module.exports = {
  rules: {
    'implements-pen-node-id': implementsPenNodeId,
  },
};
