# Refactor: pay the token debt — blue family

Status: ready-for-review

> Slice 2 of `disc-2026-08-settings-token-bypass-retrofit`. 48 blue literals in
> 22 files → zero. Same playbook as the red slice (PR #291), same result: the
> swap exposed a broken token, and this time also completed a family.

## The token the swap exposed — exactly as predicted

PR #291 closed with: adding `--accent-text` to the contrast gate "is the blue
slice's first commit, and it will fail until the value is raised." It did.

**`--accent-text` raised #60a5fa → #93c5fd.** Measured worst case for the old
value: **3.52:1** on an accent-tinted chip over `--bg-tertiary`, 4.40:1 on plain
`--bg-tertiary` — sub-AA in **55 usages**, including the token PR #287 told
callers to swap TO. `#93c5fd` clears every surface (worst 4.97:1) and is the
value **24 call sites were already hardcoding as `text-blue-300`**. Identical
shape to the red slice: the intent was in the code, it never reached the token.

## The mapping

| literal | n | → |
|---|---:|---|
| `text-blue-300` / `text-blue-200` | 28 | `text-[var(--accent-text)]` |
| `bg-blue-{400,500,800,900}/xx` | 8 | `bg-[var(--accent-tint)]` |
| `bg-blue-400` (solid) | 1 | `bg-[var(--accent-primary)]` |
| `border-blue-{400,800}`, `ring-blue-400` | 8 | `--accent-hover` (blue-400 IS #60a5fa) |

## Three judgement calls

1. **`QBittorrentForm`'s disabled submit** — the critique's single worst reading
   (2.4:1, `bg-blue-800` + accent text). Not translated: replaced with the
   sibling 測試連線 button's standard disabled treatment
   (`bg-tertiary` + `text-muted`). Consistency inside the component, again.

2. **`SubtitleSearchDialog`'s checkbox** — `text-blue-600` on a native checkbox
   drives the CHECKED FILL, not text. Became `--accent-primary` (the brand
   fill), not `--accent-text`.

3. **`TechBadge` converted whole — all four variants, three of them "outside"
   this slice.** The red slice deferred border-removal so components would not
   ship half-converted; the same principle points the other way here. A 4-line
   self-contained badge map is better converted at once than left 1/4 done for
   three more PRs. The `*-500` text shades measured 2.42–4.31:1 on their own
   tints; the token variants measure 4.71–6.27:1. That surfaced a missing family
   member: **`--info-text: #22d3ee`** (worst 5.14:1) now completes the
   error/accent/info *-text symmetry and is in the contrast gate.

## Verification

- Contrast gate now covers `accent-text` + `info-text`; falsified against the
  old `#60a5fa`.
- 3 spec files updated from literal to token assertions.
- Full suite: green except the 16 pre-existing `src/eslint-rules` failures.
- **Visual churn, honestly reported: 5 darwin baselines regenerated** —
  `media-tech-badge`, `media-tech-badge-group` (the badge hues genuinely
  changed; HDR10 amber is the visible one), `settings-qbittorrent-form` ×3 (the
  disabled button treatment). Matching `-linux` deleted for CI bootstrap, per
  the standing rule. This slice's shifts EXCEEDED the 0.2 threshold — the red
  slice's did not — confirming the prediction in #291.

## Remaining

amber 32 · green 29 · yellow 25 · emerald 25 · purple 6 · orange 6, then the
exemption-free lint rule + tinted-surface border removal.
