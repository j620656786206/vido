# Fix: settings graduation — the last two P1s + the emoji door-closer

Status: ready-for-review

> Closes the two P1s left standing after critique R4 (31/40), plus the lint
> rule Alexyu asked to ride along:「修掉最後兩個 P1，順便把 lint 規則加上」.

## P1#1 — the vocabulary's last two stragglers (plus one the critique missed)

- **ApiKeysForm**「✅ 金鑰已儲存」→ neutral `--text-secondary` + lucide
  `<Check>` — QBittorrentForm's exact save-state pattern, so the two forms on
  the same page finally speak one language.
- **MetadataExport** result strings carried ✅/⚠️ prefixes and one green wash.
  Now outcome-typed `{tone: 'ok' | 'error', text}`: ok → neutral + `<Check>`,
  error → `--error-text` + `<XCircle>` + `role="alert"`.
- **BackupScheduleConfig** (bonus, found by the rule's own grep, not the
  critique): two more「✅ …已啟用/已儲存」→ de-emoji'd.

## P1#2 — the export tab stops lying

`/settings/export` claimed「尚未實作」while a working exporter lived one tab
away inside 備份與還原 — the strip's one outright false statement (filed R1).
Now:

- The route renders the real `MetadataExport` under the route-header contract
  (h1 匯出/匯入 + description), with 匯入's pending half stated honestly ON
  the page (dashed card:「匯入功能尚未實作 — 目前請以「備份與還原」還原完整資料」).
- The tab graduates `unavailable → maintenance`: groups go [3,2,3,2] →
  [3,2,4,1], still inside the ≤4-per-decision-point rule; 尚未開放 now marks
  performance alone — every badge tells the truth.
- `BackupManagement` drops the `<MetadataExport />` mount — the backup page
  falls back to 3 modules / 1 gold primary, shrinking R4's「backup 頁三金」
  P2 as a side effect.

## The door-closer — `local/no-emoji-in-ui`

All three P1#1 sites wore the same glyph, and emoji escape BOTH enforcement
layers at once: not tokens (invisible to `no-hardcoded-palette`) and not CSS
colours (invisible to `styles-contrast.spec.ts`) — a ✅ is a hardcoded green
no gate can measure.

- Trigger: `\p{Emoji_Presentation}` (colour-by-default: ✅❌⭐🔴⏳, all
  U+1F300+ pictographs) or U+FE0F (forces colour onto ⚠️✏️) in string
  Literals, TemplateElements, or JSXText. Text-presentation glyphs
  (✓ ✗ ★ ☆ ☰, bare ⚠) are typography, deliberately legal. Comments exempt
  by construction (not AST string nodes).
- Scope: `components/settings/**` + `routes/settings/**` — exemption-free
  there after this PR. The ~20 files elsewhere still carrying 🎬✅📁 are
  filed as `disc-2026-08-emoji-sweep-repo-wide`; widen the glob when the
  sweep lands.
- Falsified live: re-adding a ✅ to BackupScheduleConfig fails lint with the
  rule's message; removing it restores 0 errors.

## Verification

- `SettingsLayout.spec` updated (+「export graduated: enabled, navigable, no
  尚未開放 badge」— red before the fix, green after) — 32/32.
- `no-emoji-in-ui.spec.ts` — 15/15 (rule via `Linter` flat-config +
  filename; wiring as config text, same strategy as the palette rule's spec).
- Settings suite 337/337; full web 2830 passed with only the 16 known
  local-env eslint-config-import failures (pass in CI).
- Visual churn exactly where expected: `settings-backup-management` ×3
  (default/hover/focus — the removed export card). darwin regenerated,
  `-linux` deleted for bootstrap. Everything else untouched.
