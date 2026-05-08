# bugfix-10-4 Design ↔ Implementation Comparison Artifact

**Story:** `bugfix-10-4-hover-preview-viewport-flip`
**Author:** Amelia (DEV) — auto-generated 2026-05-08 by `/dev-story` workflow
**Pen node:** `Component/PosterCardHover` (`MQbvp`)
**Status:** ⏸️ **AWAITING SALLY (UX) SIGN-OFF** — DEV cannot self-sign per AC #6.4.

> 📌 This artifact is the bugfix-10-4 instance of Rule 22 (Epic Retro Design-Drift Audit) — also mirrored to `_bmad-output/audit/drift-bugfix-10-4-2026-05.md`.

---

## 🎨 Design Reference (Source of Truth)

**`.pen` Component/PosterCardHover (node `MQbvp`):**

Captured via Pencil MCP `get_screenshot(nodeId="MQbvp")` on 2026-05-08 during DEV step. The image lives in DEV agent's context (Sally has Pencil app open and can re-capture for sign-off).

**MQbvp visual elements (top-to-bottom, left-to-right):**

| Slot | Element | Style |
|---|---|---|
| Top-LEFT | Empty circle outline (selection slot) | white border, transparent fill, ~24×24 |
| Top-RIGHT | Kebab `⋯` button | white ellipsis on dark rounded button bg |
| CENTER | Large play `▶` button | white triangle on semi-transparent dark circle, ~64×64 |
| Bottom-LEFT | Title + (original title) + year | white text, dark gradient backdrop fading from bottom |
| Bottom-RIGHT | Star rating `⭐ 8.4` | yellow star + number on dark pill bg |

**`.pen` Component/PosterCard (node `RusTY`, default state for reference):**

| Slot | Element |
|---|---|
| Image | full-bleed poster |
| Below image | title (zh-TW), no year visible in this snapshot |

---

## 💻 Implementation State (after this story)

**File:** `apps/web/src/components/media/PosterCard.tsx`
**Header:** `// Implements: Component/PosterCardHover (MQbvp)` (Rule 21 ✅)

**Element-by-element mapping:**

| MQbvp Slot | Implementation | File:Line | Visibility Gate |
|---|---|---|---|
| Top-LEFT empty circle | `data-testid="selection-checkbox"` | PosterCard.tsx:130-141 | `selectable={true}` only (preserves Story 5-7 semantics) |
| Top-RIGHT kebab `⋯` | `data-testid="poster-menu-button"` | PosterCard.tsx:172-183 | `onMenuClick` provided + `lg:group-hover:opacity-100` |
| CENTER play `▶` | `data-testid="hover-play-overlay"` | PosterCard.tsx:186-195 | `!selectable` + `hidden lg:flex opacity-0 lg:group-hover:opacity-100` |
| Bottom-LEFT title overlay | `data-testid="hover-title-overlay"` | PosterCard.tsx:198-203 | `hidden lg:block opacity-0 lg:group-hover:opacity-100` |
| Bottom-RIGHT rating `⭐` | `.absolute.bottom-2.right-2` | PosterCard.tsx:206-211 | `voteAverage > 0` (always visible — informational) |

**Default-state preserved (RusTY parity + production additions):**

| Element | Status |
|---|---|
| Below-image title + year (`<div className="mt-2">`) | ✅ kept (RusTY parity for non-hover state, mobile + desktop) |
| Top-right badge cluster (availability/isNew/type) | ✅ kept; fades out on hover (`lg:group-hover:opacity-0`) so kebab takes over corner — AC #10 collision strategy |
| Image scale + shadow on hover | ✅ unchanged from pre-bugfix-10-4 |

---

## 🔄 Diff vs Pre-bugfix-10-4

| Change | Before (Story 2-3) | After (this story) |
|---|---|---|
| **HoverPreviewCard.tsx** | 52-line component rendering overview/genre/originalTitle floating below card | DELETED |
| **HoverPreviewCard.spec.tsx** | 91-line test suite | DELETED |
| **isHovered React state** | `useState(false)` + `onMouseEnter`/`onMouseLeave` handlers | REMOVED — hover is now pure CSS via `lg:group-hover:` |
| **Kebab menu position** | `absolute left-2 top-2` (top-LEFT) | `absolute right-2 top-2` (top-RIGHT, MQbvp) |
| **Rating position** | `absolute bottom-2 left-2` (bottom-LEFT) | `absolute bottom-2 right-2` (bottom-RIGHT, MQbvp) |
| **Center play overlay** | absent | NEW — `data-testid="hover-play-overlay"` (MQbvp) |
| **In-card title overlay** | absent | NEW — `data-testid="hover-title-overlay"` (MQbvp) |
| **Top-right badges hover behavior** | always visible | fade out on hover (so kebab can occupy corner) |

---

## 🧪 Test Coverage

**File:** `apps/web/src/components/media/PosterCard.spec.tsx`

| Test | Type | Asserts |
|---|---|---|
| `[P0] center play overlay is in DOM with hover-only visibility classes (AC #1)` | unit | element exists + `hidden lg:flex opacity-0 lg:group-hover:opacity-100` classes |
| `[P1] center play overlay is NOT rendered in selection mode (AC #1)` | unit | element absent when `selectable={true}` |
| `[P0] kebab menu repositioned to top-right (AC #1)` | unit | `right-2 top-2` classes present, `left-2` absent |
| `[P0] rating badge repositioned to bottom-right (AC #1)` | unit | `.absolute.bottom-2.right-2` exists, `.absolute.bottom-2.left-2` absent |
| `[P0] in-card title overlay exists with hover-only visibility classes (AC #1)` | unit | element + `hidden lg:block opacity-0 lg:group-hover:opacity-100` classes |
| `[P1] HoverPreviewCard is no longer in the DOM (AC #2 — deletion regression guard)` | unit | `hover-preview-card` testid absent |

**Per Rule 16:** Tests use `toBeInTheDocument` + `toHaveClass` (specific matchers). No `toBeVisible` for CSS-hover-dependent elements (RTL cannot fire CSS `:hover`).

---

## 🚥 Regression Gate Results

| Check | Result |
|---|---|
| `pnpm nx test web` | ✅ 1761/1761 PASS |
| `pnpm nx test api` | ✅ PASS (Nx-flagged flaky retry on `TestScannerService_SSEBroadcast_ScanCancelled` — known pre-existing per project history) |
| `pnpm lint:all` (go vet + staticcheck + eslint + prettier) | ✅ 0 errors / 122 warnings (= bugfix-10-2 baseline; AC #8 met) |
| ESLint baseline maintained | ✅ no new warnings introduced |
| Prettier clean | ✅ all matched files |

**Test migration cost:** 5 spec files updated to use `getAllByText(...)[0]` instead of `getByText(...)` for poster card titles/years that now render twice (below-image + in-card overlay). Files: `MediaGrid.spec.tsx`, `RecentlyAdded.spec.tsx`, `LibraryGrid.spec.tsx`, `SearchResults.spec.tsx`, `ExploreBlock.spec.tsx`.

---

## ⏸️ Sign-Off Section (Sally fills out)

**Sally — please verify by:**

1. Pull up `Component/PosterCardHover` (MQbvp) in Pencil app via your `.pen` document.
2. Run `pnpm nx serve web` locally and navigate to `/`.
3. Hover on any `PosterCard` in the homepage's "熱門電影" or "熱門影集" row.
4. Compare side-by-side with the MQbvp screenshot.
5. Check each MQbvp slot in the table above is correctly populated visually:
   - [ ] Top-LEFT: empty circle (only when `selectable={true}` — try selection mode if needed)
   - [ ] Top-RIGHT: kebab `⋯` appears on hover (when `onMenuClick` provided)
   - [ ] CENTER: play `▶` overlay appears on hover (when not in selection mode)
   - [ ] Bottom-LEFT: title + year overlay appears on hover with dark gradient
   - [ ] Bottom-RIGHT: rating `⭐ X.X` always visible
   - [ ] Top-right badge cluster (availability/isNew/type) fades out on hover so kebab takes over

**Sign-off result:**

- [ ] ✅ APPROVED — proceed to Task 7 closeout (story → review)
- [ ] ⚠️ APPROVED WITH MINOR NOTES — list below, file as bugfix-10-X follow-up
- [ ] ❌ REJECTED — DEV iterates Task 3 and re-runs Task 6

**Sally's notes (if any):**

_(populated by Sally at sign-off)_

**Sign-off date:** _____ **By:** _____

---

## 📎 Cross-References

- Story: `_bmad-output/implementation-artifacts/bugfix-10-4-hover-preview-viewport-flip.md`
- Rule 21 source: `project-context.md` L654 (Component-to-Design Node Traceability)
- Rule 22 source: `project-context.md` L700 (Epic Retro Design-Drift Audit) — this artifact is the bugfix-10-4 instance
- Rule 22 audit mirror: `_bmad-output/audit/drift-bugfix-10-4-2026-05.md` (copy of this file)
- Origin: Party Mode 2026-05-08 (Sally + Bob + Winston + Amelia + Murat consensus)
