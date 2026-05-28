# Time-Bomb Fixtures Audit — Story 19-9 (Rule 23 Phase-1 Backfill)

## Header

| Field | Value |
| ----- | ----- |
| Scan date | 2026-05-28 |
| Scan agent | Amelia (DEV) — party-mode 2026-05-26 blessed (Murat + Sally + Winston consensus) |
| Scope | `apps/web/src/components/**/*.{ts,tsx}` excluding `*.spec.*`, `index.ts` barrels — same scope as story 19-3 Rule 21 ESLint rule |
| Files scanned | 131 (per 19-3 ESLint rule's in-scope set) |
| Time-bomb candidates (raw grep hits) | 18 |
| AST trigger (a) hits — ambient wall-clock reads | **5** |
| Deterministic formatter hits — NOT in trigger (a) | 13 |
| Migrated (dual-state baselines) | 1 (`library/RecentlyAdded`) |
| Pre-existing-safe-via-fixed-date | 1 (`retry/CountdownTimer` — 19-4b precedent) |
| Time-bomb-exempt | 3 (`retry/RetryNotifications`, `metadata-editor/MetadataEditorDialog`, `parse/useParseProgress`) |

**Top-line conclusion:** the 19-8 PR #8 visual-regression failure was the surface of ONE genuine time bomb (`library/RecentlyAdded`). Four other files trip Rule 23's AST trigger but each has a documented reason its render is not affected by the wall clock — three are exempt, one is pre-existing-safe via the fixed-date trick learned in 19-4b. After this story closes, the Rule 23 ESLint rule prevents new candidates from landing without an explicit marker. Three lessons compounded across 19-4b → 19-8 (per Dev Notes in story 19-9): the inoculation is now in CI, not in human review discipline.

## Methodology

**Grep used (AC #3 verbatim):**

```bash
grep -rln -E 'Date\.now\(\)|new Date\(' apps/web/src/components/ \
  --include='*.tsx' --include='*.ts' --exclude='*.spec.*'
```

Raw grep returns 18 files. Per Rule 23 spec, the ESLint rule's AST visitor only flags TWO shapes:

1. **MemberExpression `Date.now` / `Date.UTC` / `Date.parse`** — ambient clock read.
2. **NewExpression `new Date()` with `arguments.length === 0`** — ambient clock read.

Calls like `new Date(dateStr).toLocaleString(...)` pass an argument and are **deterministic formatters** (the output is a pure function of the input). They are NOT time bombs — they receive a date value from props/state/server and format it. Re-rendering at any wall-clock time produces the same output for the same input.

Per-file triage applied this AST-aware filter, then for each ambient-read site asked Sally + Murat's party-mode framing:

- **Visual-state impact?** Does the wall-clock-read result reach the rendered DOM in a way the Playwright `toHaveScreenshot()` baseline would pixel-diff against?
- **Existing fixture coverage?** Is the call site exercised by a gallery fixture?
- **Rule 23 disposition** — one of the AC #6 outcomes: `migrated` / `clock-injected` / `time-bomb-exempt` / `pre-existing-safe-via-fixed-date` / `out-of-scope-non-rendering`.

## Time-bomb candidates table

| # | File | Wall-clock call site | AST trigger? | Visual-state impact | Existing fixture state | Rule 23 disposition |
|---|------|---------------------|--------------|---------------------|------------------------|---------------------|
| 1 | `apps/web/src/components/library/RecentlyAdded.tsx` | L11 `Date.now() - new Date(dateStr).getTime() < SEVEN_DAYS_MS` (the `isWithin7Days` predicate that decides whether the green 新增 badge renders) | ✅ Yes (`Date.now()`) | **YES** (badge presence flips on the 7-day boundary) | single `library-recently-added/default` baseline — the 19-8 PR-blocker | **migrated** — split into `recent` + `stale` per story 19-9 AC #5, `clockTime` pinned per fixture |
| 2 | `apps/web/src/components/retry/CountdownTimer.tsx` | L55-56 `new Date(targetTime).getTime()` + `Date.now()` (countdown computation) | ✅ Yes (`Date.now()`) | **YES** (countdown string changes every second) | fixture pins `nextAttemptAt: '2020-...'` (19-4b precedent — far-past targetTime means countdown shows a stable "已過期" string) | **pre-existing-safe-via-fixed-date** — Rule 23 header added: `// Time-bomb-exempt: fixture pins nextAttemptAt to far-past 2020 date; countdown shows stable expired string (19-4b precedent)` |
| 3 | `apps/web/src/components/retry/RetryNotifications.tsx` | L187 `` `notification-${Date.now()}-${Math.random()...}` `` (in-memory unique ID for notification list keys) | ✅ Yes (`Date.now()`) | NO (ID is a React `key` attribute, never rendered as visible text or pixel-asserted) | no time-pinned fixture needed | **time-bomb-exempt** — Murat call: `Date.now` is part of unique-ID generation, the resulting string is used as a React key + storage key, never visually rendered |
| 4 | `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx` | L95 + L110 `new Date().getFullYear()` (fallback default for the year field when `initialData.year` is undefined) | ✅ Yes (`new Date()` zero-args) | LOW (only relevant when fixture provides no `initialData.year`; gallery fixture provides explicit year so the fallback never fires in visual baselines) | gallery fixture sets `initialData.year` explicitly → ambient call never executes in baseline render path | **time-bomb-exempt** — Sally call: gallery fixture always passes `initialData.year`, so the `new Date().getFullYear()` fallback path is unreachable in baseline render; the visible year is fixture-controlled |
| 5 | `apps/web/src/components/parse/useParseProgress.ts` | L147 + L177 `new Date().toISOString()` (timestamp injected into emitted log/event objects) | ✅ Yes (`new Date()` zero-args) | NO (this is a custom hook — exports state + emit functions; never renders JSX. The ISO timestamp goes into emitted event payloads consumed by callers' logging, not into visual baselines) | hook is invoked by components but the timestamp doesn't flow to rendered DOM | **time-bomb-exempt** — Murat call: file is a hook (Rule 23 spec exemption (d) — non-rendering hooks/services/stores are out-of-scope); ambient time is for log-emit only |

### Deterministic formatters — NOT time bombs (13 files)

The following 13 files contain `new Date(...)` calls but ALL pass an argument from props/state/server data. They are pure formatters — the rendered output is a function of the input date, NOT the ambient wall clock. AST trigger (a) does NOT match them; the ESLint rule will NOT flag them. No Rule 23 header needed.

| File | Call site | Why not a time bomb |
|------|-----------|---------------------|
| `apps/web/src/components/settings/BackupScheduleConfig.tsx:112` | `new Date(schedule.nextBackupAt).toLocaleString(...)` | arg-based formatter |
| `apps/web/src/components/settings/LogEntry.tsx:23` | `new Date(log.createdAt).toLocaleString(...)` | arg-based formatter |
| `apps/web/src/components/settings/BackupTable.tsx:16` | `new Date(dateStr).toLocaleString(...)` | arg-based formatter |
| `apps/web/src/components/settings/ScannerSettings.tsx:28` | `new Date(lastAt)` | arg-based formatter |
| `apps/web/src/components/settings/ServiceStatusCard.tsx:128,134` | `new Date(service.lastSuccessAt / lastCheckAt)` | arg-based formatter |
| `apps/web/src/components/learning/LearnedPatternsSettings.tsx:169` | `new Date(pattern.createdAt).toLocaleDateString(...)` | arg-based formatter |
| `apps/web/src/components/library/LibraryTable.tsx:71` | `new Date(dateStr).toLocaleDateString(...)` | arg-based formatter |
| `apps/web/src/components/downloads/formatters.ts:59` | `new Date(dateStr)` | arg-based utility |
| `apps/web/src/components/media/MediaDetailPanel.tsx:253` | `new Date(createdAt).toLocaleDateString(...)` | arg-based formatter |
| `apps/web/src/components/media/TVShowInfo.tsx:31` | `new Date(dateStr)` | arg-based formatter |
| `apps/web/src/components/media/MetadataSourceBadge.tsx:34` | `new Date(fetchDate).toLocaleDateString(...)` | arg-based formatter |
| `apps/web/src/components/media/FallbackFailed.tsx:91` | `new Date(createdAt).toLocaleString(...)` | arg-based formatter |
| `apps/web/src/components/media/PosterCard.tsx:77` | `new Date(releaseDate).getFullYear()` | arg-based formatter |

## Migration log

### `library/RecentlyAdded.tsx` → `recent` + `stale` dual-state baselines

**Before** (single state, time-bomb live):

```ts
// Implements: <screen-section — pending epic-19-8 mapping>
// ...
return Date.now() - new Date(dateStr).getTime() < SEVEN_DAYS_MS;
```

Gallery fixture (single row):
```tsx
{
  id: 'library-recently-added',
  // ... single fixture ...
}
```
Baseline: `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/default-visual-{darwin,linux}.png` — green 新增 badge visible at capture time, then silently invisible once wall-clock advances past 7 days post-`createdAt`. Root cause of 19-8 PR #8 CI failure.

**After** (Rule 23 conformant):

```ts
// Clock-mocked: gallery fixture library-recently-added uses page.clock.setFixedTime
// Implements: <screen-section — pending epic-19-8 mapping>   // (upgraded to // Design ref: when 19-8 lands)
// ...
return Date.now() - new Date(dateStr).getTime() < SEVEN_DAYS_MS;
```

Gallery fixture (two rows):
```tsx
{
  id: 'library-recently-added/recent',
  clockTime: '2026-05-15T00:00:00Z',  // 3 days after createdAt=2026-05-12 → isWithin7Days=true
  // ...
}
{
  id: 'library-recently-added/stale',
  clockTime: '2026-05-30T00:00:00Z',  // 18 days after createdAt=2026-05-12 → isWithin7Days=false
  // ...
}
```

Baselines (committed in 19-9):
- `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/recent/visual-darwin.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/stale/visual-darwin.png`
- `-linux` counterparts via CI bootstrap (`requires-manual-review` PR per 19-4b Task 5 + 19-5 workflow).
- Old `default-visual-{darwin,linux}.png` removed (it represented neither state cleanly).

## Exemption ledger

| File | Disposition | Rationale | Reviewer (initials) |
|------|-------------|-----------|---------------------|
| `retry/CountdownTimer.tsx` | pre-existing-safe-via-fixed-date | Fixture pins `nextAttemptAt: '2020-...'` (19-4b precedent). Countdown computation `targetTime - Date.now()` is always a large negative number → countdown displays a stable "已過期" string. The ambient `Date.now()` does advance with wall clock, but the rendered output saturates — visible string doesn't change as long as `targetTime` stays in the far past. | Murat (test-architecture call: stable saturation = effective time-bomb immunity) |
| `retry/RetryNotifications.tsx` | time-bomb-exempt | `Date.now()` participates in unique-ID generation `notification-${Date.now()}-${rand}`. ID is used as React `key` and as in-memory map key; never rendered as visible text. Pixel baselines are unaffected by the ID value. | Murat (test-architecture call: ID generation has no visual contract) |
| `metadata-editor/MetadataEditorDialog.tsx` | time-bomb-exempt | `new Date().getFullYear()` is a FALLBACK default for the `year` field used ONLY when `initialData.year` is undefined. The gallery fixture provides an explicit `initialData.year` (e.g. 2024), so the baseline render path never executes the fallback. The visible year is fixture-controlled, not ambient. | Sally (visual-state call: the rendered year is fixture-pinned in all baselined states) |
| `parse/useParseProgress.ts` | time-bomb-exempt (out-of-scope-by-Rule-23-spec) | File is a custom hook (`useParseProgress`); exports state + emit functions; returns no JSX. Rule 23 spec exemption (d): "hooks/services/stores under `components/` that never render to the DOM" are out-of-scope. The ambient `new Date().toISOString()` flows into log/event payloads consumed by callers' logging, not into rendered DOM. | Murat (test-architecture call: Rule 23 is component-UI-only per the spec) |

## Cross-link to 19-3 audit + visual-baseline-19-4

This audit complements:

- `_bmad-output/audit/drift-19-3-2026-05.md` — the 19-3 ESLint rule's accepted-marker grammar list. Rule 23 markers (`// Clock-mocked` / `// Clock-injected` / `// Time-bomb-exempt`) coexist with Rule 21 markers (`// Implements:` / `// Design ref:` / `<utility>` / `<screen-section>`). Two-line header convention — Rule 23 marker FIRST, Rule 21 marker SECOND (no rule conflict; the 19-3 ESLint rule's "leading comment" definition accepts extra leading comments above the Rule 21 marker).
- `_bmad-output/audit/drift-sweep-2026-05.md` (story 19-8) — the comprehensive design-vs-code drift sweep. 19-8 examined the SPATIAL dimension (does the component match the `.pen` design?); 19-9 examines the TEMPORAL dimension (does the component's render survive a moving wall clock?). Together with the per-epic Rule 22 cadence, these are the three orthogonal coverage dimensions for visual-baseline correctness.

## Audit-trail markers

| Field | Value |
| ----- | ----- |
| Audit doc commit | (filled at story close — this audit doc is committed as part of story 19-9) |
| Methodology grep regenerated | `grep -rln -E 'Date\.now\(\)\|new Date\(' apps/web/src/components/ --include='*.tsx' --include='*.ts' --exclude='*.spec.*'` (run 2026-05-28) |
| Sample policy | FULL audit — every component-dir `.ts` / `.tsx` AST-walked; this is the Rule 23 baseline audit (analogous to the 19-3 Rule 21 backfill), not a Rule 22 ≥5-sample retro |
| Future re-scan trigger | Rule 23's ESLint rule prevents new AST trigger (a) hits from landing without a marker; this audit is the FIRST-PASS baseline. Future stories adding new time-bomb candidates either (a) include a Rule 23 marker in the same PR (ESLint forces it) or (b) get caught in the next Rule 22 retro |
