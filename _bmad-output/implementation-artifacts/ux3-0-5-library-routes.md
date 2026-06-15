# Story ux3-0-5 — `/library/{movies,tv}` clean-route split (frontend)

**Epic:** ux3-foundation (UX Redesign Phase 3) · **Status:** review (impl done, tests green)
**Owner:** Dev (Amelia) · **Type:** frontend · **FRs:** PH3-F3 (D2) · **Approach:** full layout restructure (Alexyu, 2026-06-15)

## Story

As a user,
I want clean type URLs (`/library/movies`, `/library/tv`),
So that library links are clean and bookmarkable (D2), with deep links preserved.

## Acceptance Criteria

**Given** the `/library` route,
**When** restructured,
**Then** `library.tsx` is the LAYOUT: the v2 Browse UI is mounted ONCE here (so movies↔tv
preserves filter + scroll state — ADR F5), with `library/index` · `library/movies` ·
`library/tv` child routes as path markers. The single shared grid (`LibraryBrowseV2`) is
NOT forked across views (P7 guard).

**Given** old deep links,
**When** `/library?type=movie` (or `type=tv`) loads,
**Then** the layout's `beforeLoad` throws `redirect()` to `/library/movies` (`/library/tv`)
— a route-level redirect (never a component redirect, F1), never 404 (P2); `?type=all`/
absent stays at `/library` (merged view). No redirect loop (the clean routes carry no
`type`).

**Given** the active clean child,
**When** the layout renders,
**Then** it derives the type via `useMatchRoute` and passes it to `LibraryBrowseV2`
(`type` prop), which uses it over the legacy `?type=`. `onTypeChange` (filter sheet)
navigates to the clean route, preserving filters (drops `type` from search).

**Given** the strangler flag,
**When** OFF,
**Then** `/library` renders the legacy `LibraryPage` unchanged (the layout shell-gates;
the markers render null via `<Outlet/>` — zero pixel change).

## Tasks

1. [x] New child routes `routes/library/{index,movies,tv}.tsx` — path markers (`component:
   () => null`); search params + `staticData.shell` inherited from the layout.
2. [x] `routes/library.tsx` — `beforeLoad` `?type=`→clean-route redirect; `LibraryRoute`
   derives type from the matched child (`useMatchRoute`) and mounts `LibraryBrowseV2 type={…}`
   once + `<Outlet/>`; legacy branch unchanged.
3. [x] `LibraryBrowseV2.tsx` — accepts a `type?` prop (over `?type=`); `onTypeChange`
   navigates to the clean route (filters preserved).
4. [x] `routeTree.gen.ts` regenerated (TanStackRouterVite, committed).
5. [x] Tests — new LibraryBrowseV2 type-prop test (layout passes type → infinite query
   uses it). Full web suite 2173/2173 green; `nx build web` green; `nx lint web` 0 errors.

## Dev notes

- Pilot §4.4 flagged the clean-route split as "fights the strangler flag, marginal value";
  Alexyu chose the **full layout restructure** (over the lighter redirect-only / defer
  options). F5 is honored by mounting the Browse once in the layout (children are markers),
  not per-child (which would remount + lose scroll).
- **Browser-verify (P10) is the human gate** for this daily-driver route — the redirect +
  movies↔tv live navigation + flag-OFF legacy fidelity should be walked at 390/768/1440
  after merge (CI covers unit + build; live nav is the user's gate, per the pilot model).
- `?type=` stays in `validateSearch` (the redirect reads it; forward-compatible).
