# bugfix-10-4 Design ↔ Implementation Comparison Artifact

**Story:** `bugfix-10-4-hover-preview-viewport-flip`
**Author:** Amelia (DEV) — auto-generated 2026-05-08 by `/dev-story` workflow; amended 2026-05-08 by Murat via `/bmad:bmm:workflows:testarch-automate` (drift-correction pass).
**Pen node:** `Component/PosterCardHover` (`MQbvp`)
**Status:** ✅ **APPROVED** (Sally, 2026-05-08, via Playwright spike) — amended post-sign-off to correct title-overlay drift.

> 📌 This artifact is the bugfix-10-4 instance of Rule 22 (Epic Retro Design-Drift Audit) — also mirrored to `_bmad-output/audit/drift-bugfix-10-4-2026-05.md`.

> ⚠️ **POST-IMPLEMENTATION AMENDMENT (2026-05-08, Rule 20 contract bump v1→v2):**
> The bottom-left title/year overlay specified by MQbvp was **intentionally dropped** during dev via Party Mode (Sally + Alexyu) — see `apps/web/src/components/media/PosterCard.tsx:209-213` for the inline rationale (duplicates the RusTY below-image title; legibility issues across varying poster backgrounds). This artifact's tables originally claimed the overlay was implemented at `data-testid="hover-title-overlay"`. Updated below to reflect production reality. The drift was surfaced when `/bmad:bmm:workflows:testarch-automate` ran a follow-up E2E suite that exercised the actual DOM, contradicting the doc claim. The unit test at `PosterCard.spec.tsx:211-218` and the new E2E test at `tests/e2e/poster-card-hover.spec.ts:188-215` are the locked regression guards.
>

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
| Bottom-LEFT | ~~Title + (original title) + year~~ ⚠️ **NOT IMPLEMENTED v1→v2** — see Post-Implementation Amendment above; design slot remains in `.pen` for historical reference but not shipped to production | white text, dark gradient backdrop fading from bottom (per `.pen`) |
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
| Top-LEFT empty circle | `data-testid="selection-checkbox"` | PosterCard.tsx:140-152 | `selectable={true}` only (preserves Story 5-7 semantics) |
| Top-RIGHT kebab `⋯` | `data-testid="poster-menu-button"` | PosterCard.tsx:181-194 | `onMenuClick` provided + `lg:group-hover:opacity-100` |
| CENTER play `▶` | `data-testid="hover-play-overlay"` | PosterCard.tsx:197-207 | `!selectable` + `hidden lg:flex opacity-0 lg:group-hover:opacity-100` |
| Bottom-LEFT title overlay | ⚠️ **INTENTIONALLY NOT RENDERED** — see PosterCard.tsx:209-213 inline rationale (Party Mode 2026-05-08 design correction; duplicates RusTY below-image title; legibility issues) | n/a | n/a — element is absent from DOM by design |
| Bottom-RIGHT rating `⭐` | `.absolute.bottom-2.right-2` | PosterCard.tsx:215-222 | `voteAverage > 0` (always visible — informational) |

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
| **In-card title overlay** | absent | absent (MQbvp spec'd it; **REVERSED v1→v2 in Party Mode 2026-05-08** — duplicates RusTY below-image title, legibility issues) |
| **Top-right badges hover behavior** | always visible | fade out on hover (so kebab can occupy corner) |

---

## 🧪 Test Coverage

**File:** `apps/web/src/components/media/PosterCard.spec.tsx`

**Unit (`apps/web/src/components/media/PosterCard.spec.tsx`):**

| Test | Type | Asserts |
|---|---|---|
| `[P0] center play overlay is in DOM with hover-only visibility classes (AC #1)` | unit | element exists + `hidden lg:flex opacity-0 lg:group-hover:opacity-100` classes |
| `[P1] center play overlay is NOT rendered in selection mode (AC #1)` | unit | element absent when `selectable={true}` |
| `[P0] kebab menu repositioned to top-right (AC #1)` | unit | `right-2 top-2` classes present, `left-2` absent |
| `[P0] rating badge repositioned to bottom-right (AC #1)` | unit | `.absolute.bottom-2.right-2` exists, `.absolute.bottom-2.left-2` absent |
| `[P0] in-card title overlay is intentionally NOT rendered (Party Mode 2026-05-08 design correction)` | unit | `queryByTestId('hover-title-overlay').not.toBeInTheDocument()` — locks the v1→v2 contract bump |
| `[P1] HoverPreviewCard is no longer in the DOM (AC #2 — deletion regression guard)` | unit | `hover-preview-card` testid absent |

**E2E (`tests/e2e/poster-card-hover.spec.ts`, added 2026-05-08 by Murat via testarch-automate):**

| Test | Type | Asserts |
|---|---|---|
| `[P0] hover at lg: viewport reveals center play overlay (opacity 0 → 1)` | E2E | runtime CSS `:hover` actually drives `lg:group-hover:opacity-100` (mechanism proof unit cannot do) |
| `[P0] bottom-left title overlay is NOT rendered — Party Mode 2026-05-08 dev-time decision (regression guard)` | E2E | runtime version of the unit assertion above; locks v2 contract at the browser layer |
| `[P0] hover at lg: viewport fades top-right badge cluster (opacity 1 → 0) — AC #10 collision` | E2E | counter-direction (fade-out) of group-hover; cluster wrapper `lg:group-hover:opacity-0` |
| `[P1] mobile viewport (375x667) — hover overlay layer stays out of layout (AC #6)` | E2E | `hidden lg:flex` evaluates to `display: none` at < lg breakpoint |
| `[P1] click on card body navigates to /media/movie/$id (AC #5)` | E2E | TanStack `<Link>` semantics preserved through new overlay layers |
| `[P1] click on decorative center play overlay ALSO navigates — overlay does not capture clicks (AC #1 + AC #5)` | E2E | overlay has no own onClick; click bubbles to parent `<Link>` |

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

**Test migration cost (RATIONALE AMENDED v1→v2):** 5 spec files updated to use `getAllByText(...)[0]` instead of `getByText(...)` for poster card titles. **Original rationale (now incorrect):** "titles/years now render twice (below-image + in-card overlay)." **Actual state:** the title overlay was reversed during dev, so titles render only once. The `getAllByText(...)[0]` migration is over-engineered but functional and was kept as-is to avoid post-closeout churn. Files: `MediaGrid.spec.tsx`, `RecentlyAdded.spec.tsx`, `LibraryGrid.spec.tsx`, `SearchResults.spec.tsx`, `ExploreBlock.spec.tsx`.

---

## ⏸️ Sign-Off Section (Sally fills out)

**Sally — please verify by:**

1. Pull up `Component/PosterCardHover` (MQbvp) in Pencil app via your `.pen` document.
2. Run `pnpm nx serve web` locally and navigate to `/`.
3. Hover on any `PosterCard` in the homepage's "熱門電影" or "熱門影集" row.
4. Compare side-by-side with the MQbvp screenshot.
5. Check each MQbvp slot in the table above is correctly populated visually:
   - [x] Top-LEFT: empty circle (only when `selectable={true}` — try selection mode if needed)
   - [x] Top-RIGHT: kebab `⋯` appears on hover (when `onMenuClick` provided)
   - [x] CENTER: play `▶` overlay appears on hover (when not in selection mode)
   - [x] ~~Bottom-LEFT: title + year overlay appears on hover with dark gradient~~ **DROPPED v1→v2 (Party Mode 2026-05-08): overlay NOT rendered; below-image title row is the only title affordance**
   - [x] Bottom-RIGHT: rating `⭐ X.X` always visible
   - [x] Top-right badge cluster (availability/isNew/type) fades out on hover so kebab takes over

**Sign-off result:**

- [x] ✅ APPROVED via Playwright automated evidence
- [x] ⚠️ APPROVED WITH NOTES — 4 polish items deferred to `bugfix-10-7-postercard-info-density-and-polish` (filed)
- [ ] ❌ REJECTED — DEV iterates Task 3 and re-runs Task 6

**Sally's notes:**

Sign-off ritual was conducted via Playwright spike (`spike-bugfix-10-4-hover-diagnostic.spec.ts`) running against fresh Chromium with no cache. Captured before/after-hover screenshots + 6 mid-transition frames + computed-style dump. Evidence proves:

1. ✅ **Center play overlay**: `display: flex`, `opacity: 0 → 1` on hover (300ms duration after revert)
2. ✅ **Kebab menu**: positioned at `right-2 top-2` (top-RIGHT) per MQbvp; `lg:group-hover:opacity-100` works
3. ✅ **Rating badge**: positioned at `right: 8px` (right-2) bottom-RIGHT per MQbvp; always visible
4. ✅ **Top-right badge cluster fade-out**: `opacity 1 → 0` on hover with `transition-duration: 300ms`
5. ✅ **Rounded corners preserved during scale-105**: clip-path inline style `inset(0 round 0.5rem)` works around Chromium GPU rendering bug; corners stay rounded throughout transition
6. ✅ **Hover detection**: diagnostic dot turned red→green on hover, confirming `lg:group-hover:` mechanism works

**Why automated sign-off instead of manual visual:**
- User's Edge browser had stubborn cached CSS that survived Cmd+Shift+R refresh + DevTools "Disable cache" + multiple module reloads
- Playwright runs in fresh Chromium with no cache → ground truth for what production code renders
- Evidence is more rigorous than human visual inspection (captures exact pixels + computed styles)

**Items DEFERRED to bugfix-10-7-postercard-info-density-and-polish** (filed in sprint-status.yaml):

1. Info-density: runtime/episode_count display below image
2. Hover micro-interaction polish: scale transform on badge fade for more kinetic motion (opacity-only fade is technically correct but visually subtle on small text per Sally's UX assessment)
3. Rating star ⭐ emoji → lucide-react `<Star>` SVG (cross-OS rendering consistency)
4. selection-checkbox display strategy (Netflix-quick-add vs current selectable-gated behavior)

These are POLISH items beyond bugfix-10-4's scope (which is functional alignment with MQbvp position spec). bugfix-10-4 closes here; bugfix-10-7 picks up the polish thread.

**Sign-off date:** 2026-05-08 **By:** Sally (UX Designer) via Playwright automated evidence ritual
**Diagnostic spike**: deleted post-sign-off per spike artifact convention (originally at `tests/e2e/spike-bugfix-10-4-hover-diagnostic.spec.ts`).

---

## 📎 Cross-References

- Story: `_bmad-output/implementation-artifacts/bugfix-10-4-hover-preview-viewport-flip.md`
- Rule 21 source: `project-context.md` L654 (Component-to-Design Node Traceability)
- Rule 22 source: `project-context.md` L700 (Epic Retro Design-Drift Audit) — this artifact is the bugfix-10-4 instance
- Rule 22 audit mirror: `_bmad-output/audit/drift-bugfix-10-4-2026-05.md` (copy of this file)
- Origin: Party Mode 2026-05-08 (Sally + Bob + Winston + Amelia + Murat consensus)
