# Feat: settings becomes a page with tabs, not a page with a second rail

Status: in-progress

> Executes `ruling-2026-08-25-rail2-is-a-page-tool`. Alexyu chose the shape
> (tabs above content) and the grouping (three groups) at the ruling's close.

## Why the rail was wrong

The ruling says rail 2 is a **page tool**, not a second navigation level. The
settings rail was the one place that broke it: it **navigates** (changes route)
while wearing a page tool's costume, and the critique measured how far the two
had drifted apart —

| | 媒體庫 | 探索 | 設定 |
|---|---|---|---|
| width | 264px | 264px | **224px** |
| gap to rail 1 | +24px | +24px | **0px** |
| vertical origin | y=80 | y=128 | **y=56** |
| position | `sticky` | `sticky` | **`static`, scrolls away** |
| breakpoint | `lg` | `lg` | **`md`** |
| collapsible | yes | yes | **no** |
| item radius | ∞ (pill) | ∞ | **0px** |
| contract | filter | filter | **navigate** |

Worse, the visual signals pointed backwards: the rail that navigates looked like
the heaviest fixed structure (full-bleed rows, hard left bar), while the rails
that merely filter got a title, a live count badge and a collapse chevron. Infer
the contract from the appearance and you guess wrong.

## Decision

**One horizontal tab strip above the content, with the three groups separated by
a divider.** Not two levels — grouping exists to cut cognitive load, and a second
level would add a click instead. The divider is the structure made visible.

**Semantically navigation, visually tabs.** These change route; they are not ARIA
tabs (no tabpanels in the same document). So: `<nav aria-label="設定分類">` with
`<Link>` children and `aria-current="page"`. Using `role="tablist"` here would
promise a widget the DOM does not implement.

**One pattern for both device classes.** Desktop and mobile stop diverging: the
mobile chip strip stops being a fallback and becomes the real thing at a smaller
size. That collapses two problems into one — the P1 where 5 of 10 categories sit
off-screen at 390px with no affordance is now fixed once, for both.

**No new `<h1>`.** Each settings page already has exactly one; the strip is nav
above it. Adding a page-level 設定 heading would make two per page.

### Grouping (Alexyu, 2026-08-25)

| group | categories |
|---|---|
| 連線 | 連線設定 · 金鑰設定 · 服務狀態 |
| 媒體庫 | 媒體庫掃描 · 自訂首頁 |
| 維護 | 快取管理 · 系統日誌 · 備份與還原 |
| (trailing, unavailable) | 匯出/匯入 · 效能監控 |

Each live group is 2–3 items, inside the ≤4 rule the flat 10 broke. **This
reorders the list** — 服務狀態 and 備份與還原 move — which is the point: the
current order is insertion order, not meaning.

### Width, measured before committing to it

Removing the rail returns the content column from 976px → 1200px at a 1440
viewport. Eight enabled tabs at ~80–94px plus two disabled plus two dividers is
roughly 860px, so the strip fits without scrolling at 1440 and at 1280. Below
that it scrolls — which is the same code path as mobile, and therefore the same
fix.

## Acceptance Criteria

1. **The rail is gone.** `SettingsLayout` renders no `<aside>`/vertical nav; the
   content column is full width. `settings-sidebar` no longer exists.
2. **One strip, one pattern.** A single `<nav aria-label="設定分類">` renders at
   every breakpoint — no separate desktop and mobile navs. Sizing may differ;
   structure and testids may not.
3. **Grouping is visible and true.** The three groups render in the order above,
   separated by a divider element, with the unavailable pair trailing. The
   divider is presentational (`aria-hidden`) — the grouping is conveyed to AT by
   the accessible names, not by a decorative line.
4. **Overflow is signposted.** When the strip is wider than its container it
   scrolls, the clipped edge is visible (an edge fade), and the active tab is
   scrolled into view on mount. No category is reachable only by an undiscoverable
   swipe.
5. **Touch targets ≥44px** on the strip at every size — the system's own minimum,
   which the old 30px mobile chips missed.
6. **Semantics.** `aria-current="page"` on the active tab; the disabled pair keeps
   the reachable-and-self-explaining treatment from PR #287 (`role="link"` +
   `aria-disabled` + `tabIndex` + reason in the accessible name).
7. **Contrast holds.** The active tab keeps the PR #287 recipe (`--accent-subtle`
   wash + `--text-primary`), re-measured live, not assumed.
8. **Tests.** Group order, divider placement, scroll affordance, 44px targets,
   `aria-current`, and the absence of the old rail — each failing before the change.

## Out of scope

- Removing the per-page `<h1>`s so the tab label is the only heading. Real
  redundancy, but it touches eight files that PR #286 just changed; separable.
- The filter rails on 媒體庫/探索. They are already page tools and stay as they
  are; their own findings (24px gap, unpinned footer, no surface tone) are filed
  separately.
