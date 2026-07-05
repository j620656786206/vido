# Story 13-7b: Request cancel + retry — frontend (RequestRow action-area wiring)

Status: ready-for-dev

> **Depends on: 13-7a (backend API must be ready)** — `DELETE /api/v1/requests/{id}` + `POST /api/v1/requests/{id}/retry` [@contract-v1].
> Split note: 13-7 counted 5 BE + 5 FE tasks → mandatory a/b split (Epic 8 Retro Agreement 5). This is the **frontend** half.

## Story

As a Vido user looking at my 想要清單,
I want a 取消 action on pending requests and a 重試 action on failed ones — exactly as the design draws them,
so that I can act on my requests without leaving the list, and a failed request is one tap from a fresh attempt.

## Acceptance Criteria

1. **Action-area per the `.pen` matrix (RequestRow-v2 `LkjRd`, action-area `pbeYJ`).** `RequestRow.tsx` renders, per status: **pending** → `取消` button ONLY; **failed** → fail-caption + `重試` button; **searching / downloading / completed** → NO actions (the design draws the area `enabled:false` — render nothing interactive). Styling per the drawn nodes: 取消 (`yTntT`/`XZQtz`) is a PLAIN text button — `$text-secondary`, 13px, 44px tall, `$radius-md`, padding [0,14], NOT the `Button` component; 重試 (`qDo2F`) is a `Component/ButtonSecondary` ref (`YDPhc`) → use the existing secondary-button styling. **Caption relocation (ruled):** the failed row's `$error-text` caption already ships on main (13-1b renders `errorMessage` UNDER the title, `RequestRow.tsx:72-74`), but the `.pen` failed override (`iyYqV` = `fail-caption XJ1hS` + `btn-retry qDo2F`, gap 10) draws it INSIDE the trailing action cluster next to 重試 — this story MOVES the existing caption rendering into the action cluster to match the frame (presentational move only; the caption's DATA/live-update stays 13-3b's concern). **NO confirm dialog** — the design draws none (cancel of a pending request is low-stakes: hard-delete, immediately re-requestable); do not invent one.
2. **Service methods (Rule 18).** `requestService.ts` += `cancelRequest(id)` → `DELETE /requests/{id}` (204, no body) and `retryRequest(id)` → `POST /requests/{id}/retry` (200 → updated `MediaRequest`, `snakeToCamel`). ⚠️ **204 trap:** the existing `fetchApi` (`requestService.ts:53-65`) THROWS on a 204 — `response.json()` rejects on an empty body → `!data?.success` → spurious `RequestApiError` despite `response.ok`. `cancelRequest` MUST NOT reuse `fetchApi` unmodified: early-return on `response.status === 204` (or a dedicated no-body path). Error envelope surfaces `RequestApiError` with Rule-7 `.code` (`REQUEST_NOT_CANCELLABLE` / `REQUEST_NOT_RETRYABLE` / `DB_NOT_FOUND`).
3. **Optimistic mutations.** `useRequestActions.ts` += `cancel` + `retry` mutations cloned from the existing `create` shape: `onMutate` → `cancelQueries(requestKeys.all)` + snapshot `requestKeys.list()` + optimistic patch (cancel: remove row by id; retry: patch row `status:'searching'` — the server may return pending or searching; `onSettled` invalidate reconciles); `onError` → rollback snapshot; `onSettled` → `invalidateQueries(requestKeys.all)`. A 409 (`REQUEST_NOT_CANCELLABLE`/`REQUEST_NOT_RETRYABLE`) means the row moved under us: rollback + invalidate resyncs — surface the zh-TW error via toast, not a crash.
4. **Feedback toast.** Reuse the 13-1b `RequestToast` portal pattern (`createPortal(document.body)`, `role="status"`, ~3-4s auto-dismiss): success `已取消請求` / `已重新嘗試，開始搜尋來源`; error path shows the API's zh-TW message (`role="alert"`). Lift/share `RequestToast` out of `RequestButton.tsx` only if reuse demands it — otherwise a local sibling is fine.
5. **Wiring.** `RequestsView.tsx` passes the mutation handlers down (or `RequestRow` consumes `useRequestActions` directly — match the existing `RequestButton` convention of hook-in-component). Buttons disable while their mutation is in-flight (no double-fire). Testids: `request-cancel-btn`, `request-retry-btn`.
6. **Scope walls / coordination.** Already on main via 13-1b (do NOT rebuild, only relocate per AC 1): the `errorMessage` caption (`RequestRow.tsx:72-74`) and the `role="status" aria-live="polite"` region (:79-82). Owned by sibling `13-3b` (do NOT implement): SSE live updates, progress %, and the LIVE refresh of caption/aria-live content — 13-3b edits the same `RequestRow.tsx` on disjoint concerns; whichever story lands second rebases the shared file (note: 13-3b's prose still says it "adds" the caption — stale, it ships already; flagged on 13-3b). Also out: any view-level 重試 (L7 fail-soft shipped in 13-1b — different thing). The 取消 must NOT appear on searching/downloading rows (cancel-active is `disc-2026-07-cancel-active-requests`, out of scope).
7. **Tests + gates.** Specs: `requestService.spec.ts` (both methods, 204 no-body handling, error codes); `useRequestActions.spec.tsx` (optimistic remove/patch + rollback + invalidate, per existing spec harness); `RequestRow.spec.tsx` (per-status button matrix — pending shows only 取消, failed shows only 重試, other three show none; click handlers fire; in-flight disable); toast assertions per `RequestButton.spec` pattern. Gallery fixtures: extend RequestRow fixtures so pending + failed action states are visually baselined (`-linux` via CI bootstrap PR — never local). `pnpm lint:all` + `nx test web` affected + build green. Screenshot-verify against `_bmad-output/screenshots/flow-l-requests-v2/l1-d-v2.png` (Sally gate).
8. **Contract ack.** Dev Notes record: confirmed against [@contract-v1] (Story 13-7a AC #1/#2 — endpoint shapes); confirmed against [@contract-v1] (Story 13-1a AC #3 — list resource shape unchanged). Rule 21: `RequestRow.tsx` keeps its existing `// Implements: Component/RequestRow-v2 (LkjRd)` header (no new component files expected; hooks/services exempt).

## Tasks / Subtasks

- [ ] Task 1: Service methods (AC: 2)
  - [ ] `cancelRequest` / `retryRequest` + spec (204 empty-body path, error-code surfacing)
- [ ] Task 2: Mutations (AC: 3)
  - [ ] `cancel` + `retry` in `useRequestActions.ts` + spec (optimistic/rollback/invalidate/409 resync)
- [ ] Task 3: RequestRow action-area (AC: 1, 5)
  - [ ] Per-status matrix + drawn styling + testids + in-flight disable + spec
- [ ] Task 4: Toast feedback (AC: 4)
  - [ ] Success/error toasts per RequestToast portal pattern + spec
- [ ] Task 5: Verification (AC: 7)
  - [ ] Gallery fixtures (pending/failed action states); lint:all; affected tests; build; browser-verify vs `l1-d-v2.png` @390/768/1440

**Cross-stack split check:** backend tasks = 0 (13-7a owns them), frontend tasks = 5 → single story. ✓

## Dev Notes

### Backend contract (13-7a [@contract-v1] — re-verify at dev time if 13-7a changed in review)

- `DELETE /api/v1/requests/{id}`: 204 no body; 404 `DB_NOT_FOUND`; 409 `REQUEST_NOT_CANCELLABLE` (row not pending anymore).
- `POST /api/v1/requests/{id}/retry`: 200 `{success:true, data:<request resource>}` (13-1a shape — camelCase after transform: `id, tmdbId, mediaType, title, status, fulfilmentSource, externalId, seasons, episodes, errorMessage, requestedAt, updatedAt`); 404 `DB_NOT_FOUND`; 409 `REQUEST_NOT_RETRYABLE`. Post-retry status = `pending` (terminal-fail flavor) or `searching` (queue-fail flavor) — treat both as "active again".

### Existing FE anchors (all merged on main — verified 2026-07-05)

- `apps/web/src/components/requests/RequestRow.tsx` — the target. Docblock lines 9-11 records the deliberately-unbuilt action-area (delete that note when building it). Current structure: thumb → title/meta (+ `errorMessage` caption at :72-74 — relocates per AC 1) → status pill (`request-status-${status}`) + aria-live region (:79-82) → optional Mono `%` slot. Add the action cluster as a trailing `shrink-0` flex group (mirror `DownloadRowActions`'s trailing cluster). Props today: `{ request: MediaRequest & { progress?: number } }`.
- `apps/web/src/hooks/useRequestActions.ts` — mutation template (`onMutate` cancelQueries → snapshot → patch → rollback → invalidate; `REQUEST_DUPLICATE` special-case shows the shape for 409 handling).
- `apps/web/src/hooks/useRequestedMedia.ts` — `requestKeys` factory (`all: ['requests']`, `list()`); 30s staleTime; the shared cache the mutations patch. ⚠️ Keep the key shape stable — 13-3b's future `applyRequestSnapshot` targets `[...requestKeys.all,'list']`; invalidate-on-settle is fine today, and optimistic patches on the same key keep a later switch to snapshot-merge mechanical.
- `apps/web/src/services/requestService.ts` — `fetchApi` (envelope + `snakeToCamel`), `camelToSnake` on POST bodies, `RequestApiError` carries `.code`. `listRequests()` returns a BARE array (no pagination envelope).
- `apps/web/src/components/requests/RequestButton.tsx` — `RequestToast` portal precedent (lines ~156-203): `createPortal(document.body)` (escapes clip-path/transform containing blocks), `role="status"|"alert"`, 4s auto-dismiss cleared on unmount.
- `apps/web/src/components/requests/RequestsView.tsx` — list container (Discover-hosted, `?view=requests`); holds its own `useQuery` on the same key.
- Styling tokens: `--text-secondary`, `--radius-md`, `--error-text`; secondary button per existing `Button` variants / ButtonSecondary usage in downloads. Numerics stay `font-mono tabular-nums` (untouched here).
- Spec harnesses: `useRequestActions.spec.tsx` / `useDownloadActions.spec.ts` (mock service + QueryClientProvider wrapper, assert optimistic patch/rollback), `RequestRow.spec.tsx` (extend), `RequestButton` spec for toast assertions.

### Design sources

- `.pen` nodes: RequestRow-v2 `LkjRd` (action-area `pbeYJ`; 取消 `yTntT`/label `XZQtz`; failed override `iyYqV` with 重試 `qDo2F` ref → ButtonSecondary `YDPhc`). L1 rows: `daLMM` pending / `v0AZ5e` searching / `Z5Dr9s` downloading / `yzpjn` completed / `XfjmU` failed. Screenshot `flow-l-requests-v2/l1-d-v2.png`.
- The failed row's `$error-text` caption (`找不到可用來源`) is drawn inside `iyYqV` as `fail-caption XJ1hS` + `btn-retry qDo2F` (gap 10) — caption RENDERING exists on main (13-1b, under the title); this story relocates it into the action cluster per AC 1 so caption + 重試 compose per the frame. Caption CONTENT/live-refresh remains 13-3b's concern.
- No `.pen` modification in this story → no screenshot regen.

### Architecture compliance

- Rule 5 (TanStack Query only), Rule 18 (boundary transforms), Rule 21 (existing header kept), Rule 26 (no new search params — the view already uses `?view=requests` string enum).
- Rule 23: no ambient wall-clock reads (`requestedAt` renders server data). Expected N/A.
- §8 SSE: untouched (13-3b's domain).

### Project Structure Notes

- Edits only: `requestService.ts`, `useRequestActions.ts`, `RequestRow.tsx`, `RequestsView.tsx` (+ specs, + gallery fixtures). No new routes, no BE, no shared-types changes (MediaRequest already models every field).

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - Expected **N/A — no wall-clock-reading components touched** (buttons + mutations only; dates rendered are server-supplied fields already present). If a relative-time display sneaks in: Rule 23 marker + ≥2 clock-pinned fixture states (`withFixedClock` / `clockTime`) mandatory.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: _bmad-output/implementation-artifacts/13-7a-request-cancel-retry.md AC #1/#2 [@contract-v1]]
- [Source: _bmad-output/implementation-artifacts/13-1b-one-click-request.md — RequestRow/RequestsView/RequestButton patterns + capability-honor note]
- [Source: _bmad-output/implementation-artifacts/13-3b-request-status-tracking.md — sibling scope walls (progress %, error_message, aria-live)]
- [Source: ux-design.pen RequestRow-v2 `LkjRd` + L1 `K7fiy`; screenshots flow-l-requests-v2/]
- [Source: project-context.md Rules 5/18/21/23/24/26; §8]

## Dev Agent Record

### Agent Model Used

(fill at dev time)

### Debug Log References

### Completion Notes List

### Discovery Triage

- Authoring-time discoveries are recorded on 13-7a (shared split): `disc-2026-07-cancel-active-requests` (③) + the seed-vs-design narrowing. Nothing FE-specific filed.
- (Dev: add further in-flight discoveries per Rule 24 before marking done.)

### File List
