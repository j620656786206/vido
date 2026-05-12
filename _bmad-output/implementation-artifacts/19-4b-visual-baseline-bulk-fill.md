# Story 19.4b: Visual-Baseline Bulk Fill (remaining ~99 components)

Status: backlog

<!-- Created by the Story 19-4 Party Mode ruling (2026-05-12, Sally + Bob + Murat + Winston + Amelia; Alexyu ratified): 19-4 delivered the harness + ~25 reference components; this story fills the rest. -->

## Story

As a frontend maintainer,
I want a `toHaveScreenshot()` visual baseline (default / hover / focus where applicable) for every
remaining in-scope `apps/web/src/components/` component,
so that the epic-19 visual-regression net covers the *whole* component surface (not just the ~25
reference set 19-4 shipped) before 19-8's component-vs-`.pen` sweep and any Rule 22 retro audit
relies on it.

## Acceptance Criteria

1. Every `apps/web/src/components/**/*.tsx` component that renders visible UI and is **not** already
   in `apps/web/src/routes/test/-gallery.fixtures.tsx` gets a fixture entry there
   (`{ id, label, component, props?, penNode, statesOnly?, width? }`) — `penNode` from the file's
   `// Implements:` header (`_bmad-output/audit/drift-19-3-2026-05.md` mapping; `screen-section` /
   `utility` for the placeholder/exemption forms). Reuse each component's existing `*.spec.tsx`
   mock-data shapes; data-driven components get their React-Query keys seeded (extend the gallery
   route with a `<GalleryQuerySeed>` helper, or per-fixture `queryClient.setQueryData`).
2. Deliberate skips (type/util modules `parse/types.ts`, `degradation/types.ts`,
   `downloads/formatters.ts`; the hook `parse/useParseProgress.ts`; bare layout shells with no
   sensible isolated fixture) are recorded with reasons in `_bmad-output/audit/visual-baseline-19-4.md`.
3. `pnpm run test:visual:update` regenerates the full baseline set; **UX (Sally) reviews the rendered
   gallery (`/test/gallery`)** before commit; every baseline PNG is committed; `pnpm run test:visual`
   is green on a clean re-run; burn-in (≥5 re-runs) shows 0 flake.
4. `_bmad-output/audit/visual-baseline-19-4.md` "Delivered" table is updated to the full set; the
   "Pending" section is emptied (or lists only the documented skips).
5. **Platform/CI:** decide and implement the Linux-baseline strategy 19-5 needs — either commit a
   `-linux` set generated in the CI Docker image (add `scripts/visual-baseline.sh`), or document that
   19-5's CI job regenerates it via `pnpm run test:visual:update` in a one-off commit. Update
   `tests/visual/README.md` accordingly.
6. `pnpm lint:all` 0 errors / 122 warnings; `pnpm nx test web` + `pnpm nx test api` pass;
   `pnpm test:e2e` count unchanged; `pnpm run test:cleanup` no orphans; `ux-design.pen` untouched.

## Tasks / Subtasks

- [ ] Task 1: Inventory the remaining components; bucket data-driven vs. presentational (AC: #1, #2)
- [ ] Task 2: Add fixtures — presentational components first (AC: #1)
- [ ] Task 3: Add fixtures — data-driven components (seed React-Query; `<GalleryQuerySeed>` helper) (AC: #1)
- [ ] Task 4: Generate baselines, UX review, commit; burn-in (AC: #3)
- [ ] Task 5: Linux-baseline strategy for CI (`scripts/visual-baseline.sh` or documented CI-regen) (AC: #5)
- [ ] Task 6: Update `visual-baseline-19-4.md` to the full set; full regression + close (AC: #4, #6)

## Dev Notes

- **Depends on 19-4 (the harness):** the `visual` Playwright project, `test:visual*` scripts, the
  `/test/gallery` route + `-gallery.fixtures.tsx` shape, the `data-gallery-id`/`data-pen-node`
  convention, and the ErrorBoundary-per-fixture pattern all already exist — this story only adds
  fixture entries + their baselines (and the CI Linux-baseline decision).
- Out of scope: the component-vs-`.pen` *diff* sweep + `bugfix-N` filing (19-8), CI workflow (19-5),
  upgrading `<screen-section …>` placeholders to canonical Rule 21 headers (19-8).
- All frontend / 0 backend → single story.
- See `tests/visual/README.md` for the harness, the "Adding a component" steps, and the
  baseline-update discipline.

### References

- [Source: _bmad-output/audit/visual-baseline-19-4.md] — harness + delivered set + this story's worklist
- [Source: _bmad-output/audit/drift-19-3-2026-05.md] — file→`.pen`-node mapping (`penNode` values)
- [Source: apps/web/src/routes/test/gallery.tsx + -gallery.fixtures.tsx] — the harness this extends
- [Source: tests/visual/README.md] — running, discipline, "Adding a component"
- [Source: _bmad-output/implementation-artifacts/19-4-playwright-visual-snapshot-baseline.md] — predecessor; Party Mode scope ruling

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-12 | Created by the Story 19-4 Party Mode ruling (2026-05-12) — split out the bulk fixture/baseline fill (~99 components) so 19-4 could land the harness + ~25 reference components atomically (19-5 depends on the harness). backlog. |
