# Refactor: pay the token debt — red family

Status: in-progress

> Slice 1 of `disc-2026-08-settings-token-bypass-retrofit`. Alexyu, 2026-08-25:
> 「先還 token 債」, ahead of any re-skin.

## The debt is 2.5× what was filed

The critique measured the SETTINGS surface and found 94 hardcoded Tailwind
palette literals. Re-measured across all of `apps/web/src`, excluding specs and
gallery fixtures:

| family | count |
|---|---:|
| red | **59** |
| blue | 48 |
| amber | 32 |
| green | 29 |
| yellow | 25 |
| emerald | 25 |
| purple | 6 |
| orange | 6 |
| **total** | **231 across 61 files** |

Every semantic slot these literals want already exists as a token
(`--error` 163 uses, `--error-tint` 24, `--error-text` 34, …). So this is not a
design question. It is 231 places that should have been reading the vocabulary
and were not.

## Why it is sliced

231 edits in one PR is unreviewable and would invalidate a large batch of visual
baselines at once — and `-linux` baselines need a CI round-trip, so a bad batch
is expensive to unwind. Per the Story Splitting Rule this goes family by family,
largest first. The lint rule that closes the door lands LAST, with no allowlist,
once there is nothing left to exempt.

## The mapping

Red is 18 distinct literals. Grouped by what they actually mean:

| literal(s) | n | intent | → |
|---|---:|---|---|
| `bg-red-900/30` `bg-red-900/20` `bg-red-950/30` `bg-red-400/10` `bg-red-400/20` | 24 | error surface (banner, toast, pill) | `bg-[var(--error-tint)]` |
| `bg-red-400` | 6 | status dot / solid mark | `bg-[var(--error)]` |
| `bg-red-700` `bg-red-600` | 8 | destructive button fill + hover | `--error` / **new** `--error-pressed` |
| `text-red-300` `text-red-200` | 6 | error body text | `text-[var(--error-text)]` |
| `border-red-*` (+ `/xx`) | 15 | container edge | see below |

### The borders: I planned to remove them, and did not. Here is why.

`ConnectionTestResult.tsx:25`, `ServiceStatusDashboard.tsx:150`,
`BackupManagement.tsx:145,175`, `LibraryCard.tsx:149` put a red border **and** a
red tint on the same element, which DESIGN.md's 色調優先規則 forbids outright:
「別給已經有色階的填色面再加邊框」. The original plan was to delete those borders
and close the rule breach in the same edit.

Reading the call sites killed that plan. Every one of them has a SIBLING variant
in a colour family this slice does not touch — `ConnectionTestResult` pairs the
red branch with `border-green-700 bg-green-900/30`; `LogFilters` pairs the ERROR
chip with WARN/INFO/DEBUG chips that all carry borders. Removing the border from
the red half alone would leave the component internally inconsistent **for the
duration of the migration**, which is several PRs.

So this slice **tokenises** the borders and defers the removal to a final pass,
once every family is converted and all of a component's variants can change
together. Consistency within a component beats purity within a slice.

### One new token, and one raised

`--error-pressed: #b91c1c` (= the red-700 these buttons already hover to). It
mirrors `--accent-pressed`, which already exists for exactly this role, so the
vocabulary gains symmetry rather than an exception.

**`--error-text` raised #f87171 → #fca5a5.** Swapping `text-red-300` for the
token made contrast WORSE, which is how the real bug surfaced: the token was
already below AA in 34 existing usages —

| surface | old #f87171 | new #fca5a5 |
|---|---:|---:|
| `--bg-primary` | 5.66 | 8.26 |
| `--bg-secondary` | 4.75 | 6.93 |
| **`--bg-tertiary`** | **4.05 ✗** | **5.90** |
| tinted `--bg-secondary` | **4.34 ✗** | 6.32 |
| tinted `--bg-tertiary` | **3.73 ✗** | 5.44 |

`#fca5a5` is exactly the value five call sites were already hardcoding as
`text-red-300`, so the intent was there — it just never reached the token. Same
precedent as `--text-muted`, raised from `#808080` for the same reason, with the
reason still in the comment above it.

`styles-contrast.spec.ts` now gates every body-text token against every surface,
so this class of defect fails a test instead of waiting for a critique. Falsified
against the old value.

## Acceptance Criteria

1. **Zero `*-red-*` Tailwind literals** left in `apps/web/src`, excluding
   `*.spec.*` and `-gallery.fixtures.tsx`.
2. `--error-pressed` added to `styles.css` beside `--accent-pressed`, documented.
3. **Every red border reads a token.** Removal of borders on tinted surfaces is
   explicitly DEFERRED (see above) so components do not go half-converted.
4. **No contrast regression.** Every text/background pair the change touches is
   re-measured live and is ≥4.5:1, or is a non-text fill.
5. Existing behaviour tests still pass; assertions that named a red literal are
   rewritten to name the token, and each is falsified.
6. Visual baseline churn reported honestly. **The suite passes with zero churn,
   and that is not the same as "nothing changed."** `LogEntry`'s fixture renders
   `level: 'ERROR'`, so its red DOES change: `rgb(57,55,78)` → `rgb(61,50,73)`,
   a per-channel delta of 4/−5/−5. Playwright's per-pixel `threshold` is unset
   and therefore the 0.2 default, and a shift that small scores ≈0.02 — so every
   changed pixel is scored identical. **The visual suite will not catch this
   class of colour shift**; later slices with larger deltas may well churn.

## Out of scope

blue (48), amber (32), green (29), yellow (25), emerald (25), purple (6),
orange (6) — their own slices. The `no-hardcoded-palette` lint rule lands after
the last slice, exemption-free. Border removal on tinted surfaces lands with it.

⚠️ Found while measuring, belongs to the BLUE slice, not fixed here:
**`--accent-text` is 4.40:1 on `--bg-tertiary`** — the same sub-AA defect as
`--error-text`, in the token PR #287 recommended people swap TO. The new contrast
gate deliberately does not cover it yet; adding it to `BODY_TEXT_TOKENS` is the
first commit of the blue slice, and it will fail until the value is raised.
