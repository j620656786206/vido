#!/usr/bin/env node
/**
 * One-shot backfill script for story 19-3 — prepends Rule 21 `// Implements:`
 * headers to every file under apps/web/src/components/ that the new ESLint rule
 * `local/implements-pen-node-id` flags.
 *
 * Mapping decisions (file → marker) are baked into MAPPING below. Files NOT in
 * MAPPING and not already headed get either:
 *   - `<utility — no .pen counterpart>`              if listed in CATEGORY_B (genuine pure utility/layout/helper), or
 *   - `<screen-section — pending epic-19-8 mapping>` otherwise (Category C — renders a section of a designed screen
 *                                                    frame; canonical screen-frame mapping is tracked by epic-19-8;
 *                                                    see _bmad-output/audit/drift-19-3-2026-05.md).
 * Files already carrying a `// Implements:` line are left untouched.
 *
 * Idempotent: re-running skips files that already start with `// Implements:`.
 *
 * Usage: node scripts/backfill-rule21-headers.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const COMPONENTS_DIR = 'apps/web/src/components';

// file path (relative to repo root) → exact marker line (without leading `// `)
const MAPPING = {
  // --- mapped to .pen Reusable Components (unprefixed Component/* nodes) ---
  'apps/web/src/components/media/PosterCard.tsx': null, // already headed — leave alone
  'apps/web/src/components/ui/Button.tsx':
    'Implements: Component/ButtonPrimary (otvKh) + Component/ButtonSecondary (YDPhc)',
  'apps/web/src/components/search/SearchBar.tsx': 'Implements: Component/SearchInput (6MxLT)',
  'apps/web/src/components/search/MediaTypeTabs.tsx':
    'Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4)',
  'apps/web/src/components/shell/TabNavigation.tsx':
    'Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4)',
  'apps/web/src/components/library/FilterChips.tsx': 'Implements: Component/FilterChip (jD7gF)',
  'apps/web/src/components/library/SortSelector.tsx': 'Implements: Component/SortDropdown (955EZ)',
  'apps/web/src/components/metadata-editor/GenreSelector.tsx':
    'Implements: Component/GenreTag (L1NP6)',
  'apps/web/src/components/media/TechBadge.tsx':
    'Implements: Component/TechBadge-Video (L9m19) + Component/TechBadge-Audio (9iTW3) + Component/TechBadge-Subtitle (f84BM) + Component/TechBadge-HDR (cUjyv)',
  'apps/web/src/components/library/EmptyNoQBT.tsx': null, // already headed
  'apps/web/src/components/library/EmptyNoFolder.tsx': null, // already headed
  'apps/web/src/components/library/EmptyReadyForScan.tsx': null, // already headed
};

const UTILITY_EXEMPTION = 'Implements: <utility — no .pen counterpart>';
const SCREEN_SECTION_PLACEHOLDER = 'Implements: <screen-section — pending epic-19-8 mapping>';

// Genuine pure-utility / layout / helper / type-module files (paths relative to repo root).
// Everything else not in MAPPING is Category C → screen-section placeholder.
const CATEGORY_B = new Set([
  'apps/web/src/components/ui/Badge.tsx',
  'apps/web/src/components/ui/Card.tsx',
  'apps/web/src/components/ui/Dialog.tsx',
  'apps/web/src/components/ui/HighlightText.tsx',
  'apps/web/src/components/ui/Pagination.tsx',
  'apps/web/src/components/ui/SidePanel.tsx',
  'apps/web/src/components/ui/Skeleton.tsx',
  'apps/web/src/components/media/ColorPlaceholder.tsx',
  'apps/web/src/components/media/PosterCardSkeleton.tsx',
  'apps/web/src/components/media/TechBadgeGroup.tsx',
  'apps/web/src/components/media/FileInfo.tsx',
  'apps/web/src/components/homepage/ExploreBlockSkeleton.tsx',
  'apps/web/src/components/degradation/PlaceholderContent.tsx',
  'apps/web/src/components/degradation/types.ts',
  'apps/web/src/components/settings/SettingsPlaceholder.tsx',
  'apps/web/src/components/settings/SettingsLayout.tsx',
  'apps/web/src/components/shell/AppShell.tsx',
  'apps/web/src/components/dashboard/DashboardLayout.tsx',
  'apps/web/src/components/dashboard/CollapsibleSection.tsx',
  'apps/web/src/components/setup/SetupWizard.tsx',
  'apps/web/src/components/setup/StepProgress.tsx',
  'apps/web/src/components/downloads/StatusIcon.tsx',
  'apps/web/src/components/downloads/formatters.ts',
  'apps/web/src/components/parse/types.ts',
  'apps/web/src/components/parse/useParseProgress.ts',
]);

// Discover the files the rule flags by running eslint with only this rule.
let flagged;
try {
  const out = execSync(`npx eslint ${COMPONENTS_DIR} --format json`, {
    cwd: ROOT,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  });
  flagged = JSON.parse(out)
    .filter((r) => r.messages.some((m) => m.ruleId === 'local/implements-pen-node-id'))
    .map((r) => r.filePath.replace(ROOT + '/', ''));
} catch (e) {
  // eslint exits non-zero when there are lint errors — stdout still has the JSON
  flagged = JSON.parse(e.stdout)
    .filter((r) => r.messages.some((m) => m.ruleId === 'local/implements-pen-node-id'))
    .map((r) => r.filePath.replace(ROOT + '/', ''));
}

let mappedCount = 0;
let exemptCount = 0;
for (const rel of flagged) {
  const abs = resolve(ROOT, rel);
  const src = readFileSync(abs, 'utf8');
  if (src.startsWith('// Implements:')) continue; // idempotent guard
  if (rel in MAPPING && MAPPING[rel] === null) continue; // explicitly skipped (already headed)
  if (rel in MAPPING) {
    writeFileSync(abs, `// ${MAPPING[rel]}\n// Source: ux-design.pen (Pencil app)\n` + src);
    mappedCount++;
  } else {
    const marker = CATEGORY_B.has(rel) ? UTILITY_EXEMPTION : SCREEN_SECTION_PLACEHOLDER;
    writeFileSync(abs, `// ${marker}\n` + src);
    exemptCount++;
  }
}

console.log(
  `Backfilled ${mappedCount} mapped + ${exemptCount} exemption headers (${flagged.length} flagged total).`
);
