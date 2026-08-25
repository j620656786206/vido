# Refactor: token debt endgame — the door and the borders

Status: ready-for-review

> Closes `disc-2026-08-settings-token-bypass-retrofit`. Slices 1–4 (PRs #291,
> #292, #294, #295) moved all 231 palette literals to tokens; this lands the
> two deferred pieces: the lint rule that keeps the count at zero, and the
> 色調優先 border removal that every slice postponed so components would never
> ship half-converted.

## 1. `local/no-hardcoded-palette`

New ESLint rule + wiring, mirroring the Rule 21/23 conventions (plugin-shaped
CJS export, spread into the shared `local` namespace, scoped in
eslint.config.mjs). Scope adds `routes/**` — the debt lived there too.
**Exemption-free by construction**: the migration left nothing to allowlist.

It earned its keep on its first run: it caught `accent-blue-500` in
`MetadataExport.tsx:74` — a checkbox `accent-color` utility that all four
slice scans missed because none of their regexes included the `accent-`
prefix. Fixed here (`accent-[var(--accent-primary)]`).

Spec note: the sibling wiring specs import `eslint.config.mjs`, which vitest
cannot load in this environment (the 16 pre-existing local failures). This
rule's spec avoids the trap — rule behaviour through eslint's `Linter` with an
inline flat config (needs the filename arg, or `files` globs never match);
wiring asserted against the config file's text. 13/13 green locally.

## 2. 色調優先 border removal — 25 sites, all families at once

Two treatments, by role:

- **Static message surfaces** (banners, notices, score pills, the admonition):
  the same-family border is REMOVED — the tint is the state.
  `ConnectionTestResult` needed care: its `border` width class lived in the
  shared string, not the variants. `LibraryEditModal:287` loses the tree's only
  `border-l-4` (the critique's `side-tab` P3).
- **Interactive chips** (LogFilters levels, DownloadsBrowseV2 tabs): the border
  box stays for shape stability between states, but the active colour becomes
  `border-transparent` — the wash carries selection, per the SidebarNavItem
  precedent.

Neutral card chrome (`border-subtle` + `bg-secondary`, 17 files) is NOT touched:
that is a design-pass question (/impeccable quieter territory), not token debt.

## Known remainder, filed not smuggled

`text-[var(--error)]` (and siblings) still appear as READ text on tinted
surfaces in a few banners (e.g. BackupManagement's alerts) — the base colour as
text measures sub-AA on its own tint, which is what the `*-text` tokens exist
for. That is a usage-level sweep (which var to read), not a literal-level one,
and belongs to `disc-2026-08-rails-active-nav-below-aa`'s "swap where read"
family. The contrast gate cannot see usage, only token values.

## Verification

- Lint: whole tree clean under the new rule.
- Rule spec 13/13, including comment-immunity (the migration left explanatory
  comments naming old literals; comments are not AST string nodes).
- Full suite green except the 16 pre-existing `src/eslint-rules` env failures.
- Visual churn: 4 components' darwin baselines regenerated (LearnPatternPrompt,
  ConnectionTestResult, LogFilters, ApiKeysStep — all border removals);
  `-linux` deleted for CI bootstrap.
