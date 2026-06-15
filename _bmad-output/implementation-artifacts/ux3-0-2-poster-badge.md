# Story ux3-0-2 — Poster badge renders the lifecycle/subtitle field (frontend)

**Epic:** ux3-foundation (UX Redesign Phase 3) · **Status:** review (impl done, tests green)
**Owner:** Dev (Amelia) · **Type:** frontend · **FRs:** PH3-F1 (consumer) · **Depends on:** ux3-0-1 (merged #78)

## Story

As a user browsing the library,
I want the v2 poster badge to reflect the real item lifecycle/subtitle state,
So that I can see each title's status at a glance (N1), without badge noise.

## Acceptance Criteria

**Given** ux3-0-1 now exposes `parseStatus` / `subtitleStatus` / `subtitleLanguage` on the
library list,
**When** `PosterCardV2` derives its badge,
**Then** the subtitle dimension prefers the **authoritative** engine result
(`subtitleStatus=found` + `subtitleLanguage` → 繁中/簡中/有字幕; `not_found` → 缺字幕),
falling back to embedded-track inference (`subtitleTracks`) when the engine has no terminal
result, mapped via the DL-v2 §2.5 status→token classes.

**Given** the elicitation refinement (badge = exception signal),
**When** an item is in the happy steady state (`已入庫` + `繁中`),
**Then** **no badge renders** — the badge surfaces only attention states
(整理中 / 失敗 / 缺字幕 / 簡中 / 有字幕), avoiding always-on info-noise (Epic 10/19 density).

**Given** a lifecycle exception and a subtitle exception could both apply,
**When** the badge is picked,
**Then** the lifecycle exception (整理中 / 失敗) wins; otherwise the subtitle exception;
otherwise null (steady/unknown → no badge, never errors — F3).

**Given** the legacy (flag-OFF) library,
**When** it renders,
**Then** it is unaffected (changes are confined to the v2 `PosterCardV2` + shared
`libraryStatus` util + additive optional type fields).

## Tasks

1. [x] **Types** — add `subtitleStatus?` + `subtitleLanguage?` to `LibraryMovie` +
   `LibrarySeries` (`types/library.ts`); they flow through `LibraryBrowseV2`'s
   `DisplayFields.media` (whole-object pass-through, no caller change).
2. [x] **Derivation** — `libraryStatus.ts`: `deriveSubtitleStatus` prefers
   `subtitleStatus`/`subtitleLanguage` over `subtitleTracks`; add `steadyState` to
   `StatusDescriptor`; add **`pickPosterBadge`** (exception-signal: lifecycle exception >
   subtitle exception > null, steady states suppressed).
3. [x] **Component** — `PosterCardV2` consumes `pickPosterBadge`; `media` Pick widened to
   include the two new fields.
4. [x] **Tests** — `libraryStatus.spec.ts` (authoritative source + steadyState +
   `pickPosterBadge` cases) + `PosterCardV2.spec.tsx` (steady-state suppressed; 缺字幕
   exception shown). Full web suite **2166 green**; `nx build web` green; lint 0 errors.

## Dev notes

- Activity hub (Epic 2) owns the transient process states (簡轉繁/AI校正中) via live SSE —
  out of this badge's scope (ux3-0-1 decision).
- `nx typecheck web` is pre-existing-broken on `main` (route-type/image-size errors,
  unrelated files — verified by stashing this story's diff); the CI gate is `nx build web`
  (green). Not a regression from this story.
