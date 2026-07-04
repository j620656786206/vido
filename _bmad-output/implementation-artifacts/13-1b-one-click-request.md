# Story 13.1b: One-Click Request — Frontend (想要 button + lit 想要清單 entry)

Status: done

**Epic:** Epic 13 — Request System · **FR:** P3-001 (G-1) · **Artery #1 (FE half)**
**Depends on: 13-1a (backend API must be ready)** — dev sequence a → b. GATE-A satisfied (13-0 design done, PR #118).
**Split:** cross-stack 13-1 split per Epic 8 Agreement 5; this is the **frontend** half.

## Story

As a user browsing 探索 or a TMDB detail page,
I want a one-click 想要 button that honestly shows 可請求 / 已請求·處理中 / 已入庫, and a live 想要清單 entry in Discover,
so that I can ask Vido to acquire a title without leaving the page and see what I've asked for.

## Acceptance Criteria

1. **想要 button — three honest states (design L2-D-v2).** **Given** a TMDB title on a Discover result card or the TMDB detail page, **then** a 想要 button renders exactly one of:
   - `可請求` — not owned, no active request → click fires `POST /api/v1/requests` `{tmdb_id, media_type}`;
   - `已請求·處理中` — an active request exists → non-actionable (no duplicate request can be fired from the UI);
   - `已入庫` — owned (`useOwnedMedia`/detail `owned`) → no action affordance.
     Never a broken or duplicate-request affordance (13-0 decision #5). Movie/TV differentiation: TV 想要 = **whole-series** request (`seasons`/`episodes` omitted — the season/episode tree is 13-2b, do NOT build any tree UI here).
2. **Surfaces.** The button lands on: (a) Discover result cards — inside the `PosterCard` hover overlay, following the kebab pattern (`PosterCard.tsx:230-244`) incl. `e.preventDefault(); e.stopPropagation()` (the card is a bare `<Link>`); (b) the TMDB detail page — `TMDbDetailV2`/`DetailHeroV2` action area (today only the `已入庫` badge exists there; `LocalDetailV2` is N/A — always owned).
3. **Requested-state data + stub flip.** A `useRequestedMedia` hook (mirror `useOwnedMedia.ts`) exposes `isRequested(tmdbId, mediaType)` backed by `GET /api/v1/requests` via TanStack Query (`requestKeys` factory per house convention). **Then** the stubbed `isRequested` prop (`PosterCard.tsx:29` "Stubbed to false until Phase 3", fed from `ExploreBlock.tsx:187`) is flipped to real data — the `已請求` `AvailabilityBadge` lights up on homepage ExploreBlocks AND Discover cards from the same hook. Badge copy stays `已請求` (v0-frozen component); the button uses the L2 vocabulary above.
4. **Create mutation with optimistic update.** Clicking 可請求 uses a `useRequestActions` mutation (clone the `useDownloadActions.ts` optimistic pattern: cancelQueries → cache patch → rollback onError → invalidate onSettled) so the button flips to `已請求·處理中` immediately; on error the button rolls back and the backend's zh-TW `error.message` surfaces via the L8 feedback (AC #6). A `REQUEST_DUPLICATE` 409 response settles into the `已請求·處理中` state (not an error banner — the state is true).
5. **Light up the reserved 想要清單 entry (PH3-R2 → live).** The inert `discover-requests-inert` button (`DiscoverBrowseV2.tsx:193-203`, `想要清單 · 即將推出`) becomes a live `想要清單` toolbar entry that opens the **Discover-hosted requests view** (nav-ADR:630 — lands in `discover.tsx`, NOT a new destination, NO `navModel.ts` entry): deep-linkable via a new `view` search param on `/discover` (`validateSearch`: string-enum guard `'requests'` — Rule-26-safe, never all-digits), rendering per design L1:
   - rows via a new `RequestRow` component implementing `Component/RequestRow-v2` (title + status token + Mono progress slot + action), status rendered through the DL-v2 §2.5 shared token map (`pending`→`info-tint`/想要, `searching`→`warning-tint`/搜尋中, `downloading`→`accent-tint`/下載中·{pct}%, `completed`→`success-tint`/已入庫, `failed`→`error-text`/失敗) — **no bespoke palette, all five wired** (capability-honor: only `pending` occurs until 13-3/13-4 land; invent no extra states);
   - N4 states: loading skeleton (L5), `尚無請求` empty with quiet `前往探索` (L6, distinct from failure), fetch-failure fail-soft `無法載入請求狀態` + `重試` (L7 — shell/toolbar still render).
   - **Boundary vs 13-3b (scoping ruling, recorded here AND to be honored at 13-3 create-story):** this story ships the static list view (plain fetch). 13-3b upgrades it with the live status pipeline — `request_progress` SSE (lazy, per §8), real progress %, live transitions. Do NOT implement any SSE/EventSource here.
6. **Submitted feedback (L8).** On successful create, show the inline transient toast pattern (`role="status" aria-live="polite"`, ~3s — clone `$type.$id.tsx:474-488` / `notifications/` components; there is NO global toast library) confirming the request was recorded.
7. **Rule 18 + contract ack.** `requestService.ts` uses the house `fetchApi` shape: envelope check, `snakeToCamel` on responses, `camelToSnake` on the POST body. Types (`MediaRequest`, `RequestStatus`) live with the service or `libs/shared-types` per neighbors. Dev Notes ack: confirmed against `[@contract-v1]` (Story 13-1a AC #2, AC #3).
8. **Design conformance + a11y.** All new UI is token-only (no hex), CJK in Noto Sans TC, numerics (progress %) in JetBrains Mono with Rule TY-3 number/CJK-unit split, 44px touch targets (mobile per L4-M-v2), `focus-ring`, button/entry keyboard-operable. Verify rendered UI against `_bmad-output/screenshots/flow-l-requests-v2/` (L1/L2/L4/L5/L6/L7/L8) before completion (mandatory UX verification step).
9. **Tests + gates.** Co-located vitest specs (memory-router + `vi.mock`, template `DiscoverBrowseV2.spec.tsx`): button 3-state logic incl. stopPropagation, `useRequestedMedia`, `useRequestActions` optimistic+rollback, `requestService` case transforms, requests-view N4 branches; **replace** the `discover-requests-inert` disabled assertion (`DiscoverBrowseV2.spec.tsx:103`); update `ExploreBlock`/`PosterCard` specs for the stub flip. Gallery fixtures for `RequestButton` (3 states) + `RequestRow` (5 statuses) → visual baselines; **never generate `-linux` baselines locally** — merge the CI bootstrap PR (CLAUDE.md). `pnpm nx test web`, `pnpm nx lint web` (Rule 21 headers!), `pnpm lint:all` green.

## Tasks / Subtasks

- [x] Task 1 (AC #7): `services/requestService.ts` — `fetchApi` clone (envelope + `snakeToCamel`), `createRequest({tmdbId, mediaType})` (POST, `camelToSnake` body), `listRequests()`; `MediaRequest`/`RequestStatus` types; co-located spec.
- [x] Task 2 (AC #3): `hooks/useRequestedMedia.ts` — mirror `useOwnedMedia.ts`; `requestKeys` factory (`{ all: ['requests'], list: … }`); `isRequested(tmdbId, mediaType)` = active statuses only (pending/searching/downloading); spec.
- [x] Task 3 (AC #4): `hooks/useRequestActions.ts` — clone `useDownloadActions.ts` optimistic template; 409 `REQUEST_DUPLICATE` settles to requested-state; spec.
- [x] Task 4 (AC #1, #2): `components/requests/RequestButton.tsx` — 3-state per L2; Rule 21 header (look up the L2 / RequestRow node IDs via Pencil MCP `get_editor_state` — use `// Implements: Component/…` if a reusable exists for the button, else `// Design ref: ux-design.pen Screen L2-D-v2 (…)`); wire into `PosterCard` hover overlay (preventDefault/stopPropagation) + `DetailHeroV2`/`TMDbDetailV2`; spec incl. Link-navigation NOT triggered on click.
- [x] Task 5 (AC #3): Flip the `isRequested` stub — `ExploreBlock.tsx:187` (+ `ExploreBlocksList`) and Discover cards consume `useRequestedMedia`; update affected specs.
- [x] Task 6 (AC #5): Requests view — `components/requests/RequestRow.tsx` (`// Implements: Component/RequestRow-v2 (…)`) + `RequestsView` container; `discover.tsx` `validateSearch` gains `view` (string-enum, Rule 26 note in code); `DiscoverBrowseV2` toolbar entry goes live (keep/replace testid deliberately, e.g. → `discover-requests-entry`); N4 states; specs for all branches.
- [x] Task 7 (AC #6): L8 inline `role="status"` submitted toast; spec.
- [x] Task 8 (AC #8, #9): Gallery fixtures (`-gallery.fixtures.tsx`) for RequestButton ×3 + RequestRow ×5; run visual baselines (darwin), let CI bootstrap `-linux`; screenshot-compare against `flow-l-requests-v2/`; `pnpm nx test web` + `pnpm lint:all`.

## Dev Notes

### Developer context — read these first

- **Everything is greenfield except reserved seams (scouted 2026-07-04):** inert entry `DiscoverBrowseV2.tsx:193-203` (testid `discover-requests-inert`); stubbed `isRequested` (`PosterCard.tsx:29`, fed by `ExploreBlock.tsx:187` from an ownership-shaped `() => false` stub); `AvailabilityBadge` `'requested'`/`已請求` variant; token `--info-tint` pre-reserved for the 想要 pill (`styles.css:45`). NO existing request service/hook/type/route — do not search for more, build on these seams.
- **House patterns to clone (do NOT invent):** service = `downloadService.ts:99-141` (fetchApi + mutateApi + envelope); query keys = `useDownloads.ts:19-25` factory; optimistic mutation = `useDownloadActions.ts:35-96` (snapshot/patch/rollback/invalidate); read hook = `useOwnedMedia.ts`; component test = `DiscoverBrowseV2.spec.tsx` (vi.hoisted + memory router); toast = `$type.$id.tsx:474-488` + `components/notifications/`.
- **Detail page routing:** `routes/media/$type.$id.tsx` `classifyId` → tmdb-numeric renders `TMDbDetailV2` (v2). The button integrates there via `DetailHeroV2` props/children — today it has zero action buttons (only the `owned` badge, `TMDbDetailV2.tsx:72-76`).
- **Button primitive:** `ui/Button.tsx` (cva variants, no loading spinner built in — ad-hoc `<Loader2 className="animate-spin">` is the precedent). i18n: NONE — hardcode zh-TW inline, matching neighbors.
- **Rule 21 (lint-enforced):** every new file under `components/` needs a valid header or `pnpm nx lint web` fails. Examples: `// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)`; `// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)`. Node IDs via Pencil MCP `get_editor_state` (Reusable Components list — `RequestRow-v2` is registered there by 13-0). Never read `.pen` directly.
- **Contract ack (Rule 20):** confirmed against `[@contract-v1]` (Story 13-1a AC #2 create shape, AC #3 list shape). Response resource fields (camelCase after transform): `id, tmdbId, mediaType('movie'|'tv'), title, status, fulfilmentSource, externalId, seasons, episodes, errorMessage, requestedAt, updatedAt`.
- **Vocabulary note:** `media_type` on the wire is `'movie'|'tv'` (matches the `$type` route param) — NOT `'series'`.
- **What NOT to build here (scope walls):** no SSE (13-3b), no progress polling, no season/episode tree (13-2b), no nav destination (ADR: Discover-hosted), no backend changes (13-1a), no Activity summary-row (deferred ③ from 13-0).

### Previous story intelligence (13-0, done 2026-07-04, PR #118)

- L1–L8 all drawn in `flow-l-requests-v2`; screenshots in `_bmad-output/screenshots/flow-l-requests-v2/` (l*-d/l*-m PNGs) are the verification target.
- Ratified during 13-0 review: `searching`→`warning-tint`「搜尋中」 (DL-v2 §2.5 + .pen ref-frame §8); **Rule TY-3**: numbers in Mono, CJK units in Noto — split spans (e.g. `107` Mono + `分` Noto).
- flow-i/flow-b frames were deliberately NOT mutated — L2's context excerpts ARE the affordance spec for card + detail. Don't look for a hover frame in flow-i; there isn't one.
- 13-0 was design-only; this story is the first FE code in Epic 13 — the first consumer of the v2 request design.

### Latest-tech note

No new dependency (TanStack Query/Router, cva, lucide-react all repo-pinned). Web research skipped — nothing to version-check.

### Project Structure Notes

- New: `components/requests/{RequestButton,RequestRow,RequestsView}.tsx(+specs)`, `services/requestService.ts(+spec)`, `hooks/{useRequestedMedia,useRequestActions}.ts(+specs)`; edits: `DiscoverBrowseV2.tsx(+spec)`, `discover.tsx` (validateSearch), `PosterCard.tsx`, `ExploreBlock(.spec).tsx`, `ExploreBlocksList.tsx`, `TMDbDetailV2.tsx`/`DetailHeroV2.tsx`, `routes/test/-gallery.fixtures.tsx`.
- Conventional commit scope: `feat(13-1b): …`; branch off `main`; gh account `j620656786206`; run Prettier before commit (subagent edits skip it).

### Time-dependent visual coverage

- **Default: N/A** — `RequestButton`/`RequestRow`/`RequestsView` render title/status/progress/action per design; NO relative-time display is specified, so no wall-clock read should exist. **Conditional:** IF the L1 frame (verify against `flow-l-requests-v2` screenshots) shows a relative timestamp (e.g. 「3 小時前」) and you implement it, Rule 23 fires: pair the component with a `Clock-mocked` header + ≥2 fixture states (`recent`/`stale` with `clockTime` per `-gallery.fixtures.tsx:2278-2345` precedent, `withFixedClock` helper). Prefer NOT implementing relative time this story if the design doesn't demand it.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#13-1]
- [Source: _bmad-output/implementation-artifacts/13-0-requests-design.md (L1–L8 frames, decisions 1–6, Change Log 2026-07-04)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2.5 status→token + §3.1 Rule TY-3]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md#L630 (Requests lands in discover.tsx)]
- [Source: _bmad-output/implementation-artifacts/ux3-3-1-discover-design.md#Epic-13-Requests-reservation (PH3-R2)]
- [Source: _bmad-output/implementation-artifacts/13-1a-one-click-request.md#AC-2/#AC-3 ([@contract-v1])]
- [Source: project-context.md#Rule-5/6/16/18/20/21/23/26]

## Change Log

| Date       | Change                                                                                                                                                              |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-04 | Adversarial code review (CR): 1H/2M/1L found, all fixed in-session (toast portal escape from clip-path containing block; wiring-presence gate on the card scrim; group-focus-within keyboard reveal; dedupe intent comment). Status → done. |
| 2026-07-04 | Implemented (dev-story): service/hooks/RequestButton/RequestRow/RequestsView + PosterCard scrim + detail hero action + entry lit (?view=requests) + one-point useOwnedMedia flip; 8 gallery fixtures + 10 darwin baselines; UX verify PASS. Discoveries ③×2 filed (13-7, preexisting ui-dialog darwin). Status → review. |
| 2026-07-04 | Story created (SM create-story, yolo). Cross-stack split 13-1 → 13-1a (BE) / 13-1b (FE, this). Depends on 13-1a; acks its [@contract-v1] AC #2/#3. Scoping ruling: 13-1b ships the static 想要清單 view (lit entry + L1 list + N4); 13-3b owns SSE/live status — recorded for 13-3 create-story. Status → ready-for-dev. |

## Senior Developer Review (AI)

**Date:** 2026-07-04 · **Outcome:** Approve (all findings fixed in-session) · **Reviewer:** adversarial CR workflow (claude-fable-5)

- **Git vs File List:** 0 discrepancies. **🔒 Rule 7:** N/A (no Go files in scope). **🔒 Rule 20:** N/A (pure consumer, no bumps). **🔒 Rule 25:** N/A (mega-line untouched).
- **Findings (1 High / 2 Medium / 1 Low) — ALL FIXED:**
  - [x] H1 toast clipped in card contexts — `position: fixed` inside PosterCard's clip-path + transform-gpu container (both create a fixed-position containing block) → `createPortal(document.body)`.
  - [x] M1 unwired MediaGrid surfaces (legacy Search/Library) showed a half-functional 想要 scrim whose state never reflects the created request → scrim now gates on `isRequested !== undefined` (wiring present); 2 new PosterCard gate tests.
  - [x] M2 keyboard a11y — invisible-but-focusable scrim button → `lg:group-focus-within` reveal mirrors `group-hover`.
  - [x] L1 duplicate queryKey config (RequestsView vs useRequestedMedia) — intent comment records the deliberate split (view needs error/refetch; TanStack dedupes the fetch).
- **Post-fix gates:** `pnpm nx test web` 216 files / 2322 tests ALL green (incl. the tracked flake this run); `pnpm lint:all` exit 0.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5)

### Debug Log References

- RequestsView L7 spec: the view's own `retry: 1` overrides the test QueryClient's `retry: false`, and the default exponential retryDelay outlasts waitFor — fixed with reject×2 + `retryDelay: 0` client default.
- Visual baselines: the components.visual mega-test aborts at the PRE-EXISTING ui-dialog darwin drift (stash-verified on clean main; filed `preexisting-fail-ui-dialog-darwin-visual`) before reaching the new fixtures at the array tail. Workaround: a temporary local spec (never committed) generated + stability-verified the 8 request fixtures' baselines, PNGs moved into `components.visual.spec.ts-snapshots/`, temp spec deleted.
- Four specs render the real PosterCard whose new hover scrim mounts RequestButton (needs a QueryClient): MediaGrid / PosterCard / RelatedContent / SearchResults specs stub `../requests/RequestButton` to stay presentation-focused.

### Completion Notes List

- 🔗 AC Drift: NONE (checked 'Stubbed to false until Phase 3' + 'inert until that epic lands' across apps/web — 2 hits, both explicitly Epic-13-reserved seams whose prior ACs (Story 10-4 AC #5, ux3-3-2 PH3-R2) anticipate this flip = designed evolution, REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (consumer side — this story acks 13-1a [@contract-v1] AC #2/#3; upstream shipped #126 still at v1, versions reconcile; this story stamps nothing new)
- 🎭 A11y Pre-Flight: PASS (7 components checked, 0 jsx-a11y warnings on story files, 0 introduced by this story — pills/toast/status regions carry role=status aria-live=polite per the 10-4 L1 precedent; entry button aria-pressed; RequestButton focus-visible ring; pre-existing repo-wide jsx-a11y batch = retro-11-AI1b scope, untouched)
- 🎨 UX Verification: PASS — comparison table below; rendered baselines match L1/L2 (VH3Tq/K7fiy/LkjRd node specs read via Pencil MCP + PNG spot-checks)

  | Area | Design Spec | Implementation | Match? | Fix Needed |
  |------|------------|----------------|--------|------------|
  | 可請求 button | otvKh ref, 44px, ＋想要, accent/text-on-accent | h-11 accent-primary + Plus icon | ✅ | — |
  | 已請求 pill | $info-tint pill, 6px dot, 已請求 · 處理中 $info 13/600 | rounded-full info-tint + dot + 13px/600 | ✅ | — |
  | 已入庫 pill | $success-tint pill, check 14, $success | success-tint + Check icon | ✅ | — |
  | RequestRow | LkjRd: 40×60 film thumb / title 14·600 / 電影·Mono date / status pill [4,10] r100 / Mono % | 1:1 incl. TY-3 Mono date+% | ✅ | — |
  | 5-status tokens | DL-v2 §2.5 (pending info / searching warning / downloading accent / completed success / failed error-text) | STATUS_TOKENS map exact | ✅ | — |
  | L8 toast | bg-tertiary, check $success, 已加入想要清單 + 查看清單 accent-text | fixed toast, same copy + view deep-link | ✅ | — |
  | Card hover scrim | bottom gradient scrim + full-width button | gradient scrim, rating recedes (badge-cluster collision strategy) | ✅ (deviation noted: rating fade is additive, mirrors existing pattern) | — |
  | N4 (L5/L6/L7) | skeleton / 尚無請求+前往探索 / 無法載入請求狀態+重試 | exact copy incl. motion-reduce:animate-none | ✅ | — |

- One-point stub flip: `useOwnedMedia.isRequested` now delegates to `useRequestedMedia` — ALL existing consumers (ExploreBlock/ExploreBlocksList/homepage) light up the 已請求 badge with ZERO prop changes; the button uses the exact (tmdbId, mediaType) check.
- Capability-honor: RequestRow wires all 5 DL-v2 §2.5 statuses but the design's 取消/重試 action-area is NOT built (no backend endpoint — lane ③ below); NO SSE anywhere (13-3b SCOPE WALL).
- Full gates: `pnpm nx test web` 2319/2320 (1 fail = tracked flake `preexisting-fail-instant-search-debounce-flake`, passes standalone 8/8), `pnpm nx test api` exit 0, `pnpm lint:all` exit 0, orphan-process cleanup verified. 43 new tests across 6 new spec files; 6 existing specs updated.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - If **YES**: classify each into exactly one lane per Rule 24 — ① absorbed (cite added AC/sub-task) / ② spawn-blocking story (cite sprint-status ID, mark this blocked) / ③ backlog with bidirectional carry-forward link (cite entry ID). Prose-only mentions are banned.

### File List
