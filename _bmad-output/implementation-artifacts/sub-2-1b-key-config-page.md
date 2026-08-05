# Story sub-2.1b: Key-configuration page + dead-end fix (NFR-S3)

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5) · **Risk: 🟡 MEDIUM** · **FRONTEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 2.1 · PRD **FR25** · architecture **D9 / NFR-S3** · spec §5
**Split:** the `b` half of 2-1's mandatory cross-stack split (backend 5 / frontend 4, both > 3 — Epic 8 retro rule; 13-1a/13-1b precedent).
**Depends on:** **sub-2-1a merged** (the `GET/PUT/POST-test /api/v1/settings/keys` triad + `source`/`writable` semantics) **and sub-1-7a merged** (the F5 「前往設定」 copy revision states M1 behaviour — 2-1b makes the M1.5 behaviour real).
**Blocks:** nothing — this closes Epic 2.
**Cross-stack split check:** backend tasks = **0**, frontend tasks = 4 → single story (post-split).

---

## Story

As a NAS owner,
I want to configure and edit TMDB / Claude / (optional) ASR keys in a settings page,
so that I don't need env-vars and the 「前往設定」 link actually goes somewhere.

---

## 🔎 Findings (verified 2026-07-27)

1. **It is a dead *end*, not a dead *loop*.** `ManageSubtitleDialogV2.tsx:373` does `navigate({ to: '/settings' })` with `data-testid="go-to-settings"` — `/settings` **exists and renders**. The break is that it has no key surface, so the user lands on a settings index with nothing to set. Precision matters: the fix is **re-pointing to `/settings/keys`**, not repairing navigation.
2. **`/settings/*` already has 11 sibling routes** (`backup · cache · connection · export · homepage · index · logs · performance · qbittorrent · scanner · status`). `keys.tsx` is a **12th sibling**, and **`qbittorrent.tsx` is the closest precedent** — a settings page backed by secrets with a test action. Match its structure; do not invent a new settings shell.
3. **NFR-S3 is a frontend judgement.** Whether the connection is secure is knowable reliably only in the browser (`window.isSecureContext` — true for HTTPS **and** localhost). A backend check would have to trust `X-Forwarded-Proto`, which a misconfigured reverse proxy sets wrong. Hence the warning + confirmation live here (AC #3).

---

## Acceptance Criteria

### AC #1 — `/settings/keys` page, mirroring `qbittorrent.tsx`

**Given** the 2-1a triad, **then** `apps/web/src/routes/settings/keys.tsx` renders one row per key — **Claude（翻譯）** · **TMDB** · **雲端 ASR（選配）** — each showing state from `GET /settings/keys`:

| `source` | Rendered state |
|---|---|
| `secret` | masked value (`sk-ant-…7f3a`) + 「已設定」 + 編輯 / 清除 |
| `env` | 「目前由環境變數提供」 + a note that saving here will override it — **honest about precedence** (2-1a AC #1: secret wins) |
| `none` | empty input + 「尚未設定」 |

- Password-type inputs, never rendering a fetched value into a text field (there is none to fetch — 2-1a returns masks only).
- 測試 action per key → `POST /settings/keys/test`; result inline (成功 / 金鑰無效或已撤銷 / 逾時), never a toast-only signal.
- 儲存 → `PUT`; partial payloads (only changed rows). 清除 sends `""` (2-1a's delete path) with a confirm — it falls back to env or disables the capability.
- **Reachable from the settings index** — add the entry alongside the existing 11 (do not orphan the route).
- Server state via **TanStack Query** (Rule 5); no Zustand.

### AC #2 — `writable: false` degrades honestly

**Given** 2-1a AC #4 (`ENCRYPTION_KEY` absent → `writable: false`, `reason: "encryption_key_missing"`), **then** the page renders **read-only**: current state still visible, inputs and 儲存 **disabled** (not hidden), with 「未設定加密金鑰，無法安全儲存 API 金鑰 —— 請設定 `ENCRYPTION_KEY` 後重啟」.

Rule 24 capability-honor, and the repo's own precedent in this very dialog family (`ManageSubtitleDialogV2`'s header comment: *"never draw a dead control as live"* — series CTA disabled with 影集字幕生成即將推出). A hidden control leaves the user hunting; a disabled one with a reason tells them what to do.

### AC #3 — NFR-S3: HTTPS required, or warn + explicit confirmation

**Given** D9 (*"must require an HTTPS connection, or explicitly warn the user and require confirmation when served over plain HTTP"*), **then** on `!window.isSecureContext`:

1. A persistent warning above the form: 「目前連線未加密（HTTP），API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。」
2. **儲存 is blocked until an explicit checkbox** 「我了解風險，仍要在未加密連線下儲存」 is ticked. The tick is **per page visit** — not persisted, not remembered.
3. `window.isSecureContext` (not a `location.protocol === 'https:'` string test) so **localhost stays warning-free** — the dev/first-run path is secure by definition and must not train users to dismiss the warning.
4. The warning is **advisory, never a hard block** — Vido ships HTTP by default (`docs/deployment.md` uses `http://localhost:8080`; the NAS target is `http://192.168.50.52:8088`). Blocking outright would make the feature unusable for its actual audience; D9's own wording is "warn + require confirmation".

### AC #4 — The dead end is fixed at its source

**Given** Finding 1, **then** `ManageSubtitleDialogV2.tsx:373` becomes `navigate({ to: '/settings/keys' })`. The `data-testid="go-to-settings"` is **kept** (TestSprite/e2e selectors depend on it — renaming is gratuitous churn). Any sibling 前往設定 affordance introduced by sub-1-7a's F5 copy revision is re-pointed in the same change.

**Cross-story note:** sub-1-7a's F5 revision states the **M1** behaviour (no page exists yet). This story makes the **M1.5** behaviour real. If 1-7a chose "hide the button in M1", this story **restores it** pointing at `/settings/keys`; if 1-7a chose "point at documentation", this story re-points it. Read 1-7a's Completion Notes first — do not guess which.

### AC #5 — Tests (Rule 9/16)

1. `keys.spec.tsx` — the three `source` states render their distinct affordances; masked value never appears in an editable input; partial `PUT` payload contains only changed rows; 清除 sends `""` after confirm.
2. `writable: false` → inputs and 儲存 **disabled** (`toBeDisabled()`, not absent) + reason text present.
3. NFR-S3 — mocked `isSecureContext: false` ⇒ warning + 儲存 disabled until the checkbox; `true` ⇒ neither; **localhost-over-http is treated as secure** (the `isSecureContext` semantics test).
4. `ManageSubtitleDialogV2.spec.tsx` (extend) — `go-to-settings` navigates to `/settings/keys`; the existing dialog tests stay green.
5. Test action — 401-shaped response renders 金鑰無效; timeout renders 逾時; neither throws.
6. `pnpm nx test web` green (**foreground** — background vitest orphans workers) + `pnpm lint:all` green.

### AC #6 — Scope fence

- ❌ **No backend** — zero `apps/api/**` files. The triad is 2-1a's.
- ❌ No provider/model selection UI (FR24, Tier-2), no local-worker-URL field (spec §5's Tier-2 half), no ASR provider choice.
- ❌ No changes to the settings shell/nav pattern beyond adding the entry.
- ❌ No `.pen` edits or screenshot regeneration — 2-1's own design surface is the F5 gate already revised by sub-1-7a. **If a genuinely new screen is needed** (e.g. the page has no design at all), that is a **lane ② stop-and-file**, not silent invention.
- ❌ No visual baselines — `settings/*` routes have no gallery fixtures today (same Rule 22 backfill boundary as sub-1-7b AC #5).

---

## Tasks / Subtasks

- [ ] **Task 1 — Page (AC #1):** `routes/settings/keys.tsx` mirroring `qbittorrent.tsx`; TanStack Query hooks for GET/PUT/test; three-source row states; settings-index entry.
- [ ] **Task 2 — Degradation + NFR-S3 (AC #2, #3):** `writable:false` read-only mode; `isSecureContext` warning + per-visit confirmation gate.
- [ ] **Task 3 — Dead end (AC #4):** re-point `ManageSubtitleDialogV2.tsx:373` (read 1-7a's Completion Notes first); re-point any sibling affordance.
- [ ] **Task 4 — Tests + gates (AC #5):** the five test groups; foreground vitest; `pnpm lint:all`.

---

## Dev Notes

- **Mirror `apps/web/src/routes/settings/qbittorrent.tsx`** — it is the same shape (secrets-backed settings + test action) and already solves layout, form state, and error rendering in this codebase's idiom.
- **Rule 5** — server state through TanStack Query; the key list is server state, the un-saved form is local state.
- **Rule 20** — this story **consumes** sub-2-1a's `[@contract-v1]` AC #1/#3. Record `confirmed against [@contract-v1] sub-2-1a AC #1` and `… AC #3` in Dev Notes at implementation. It stamps nothing.
- **Rule 24 capability honor** — AC #2's disabled-with-reason is the established pattern in this very dialog (`ManageSubtitleDialogV2` header: *"never draw a dead control as live"*).
- **Rule 26 N/A** — no new search params.
- **Feedback rules in play:** foreground tests only; `pnpm run format:check` before commit; verify against the design surface before marking done.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** `routes/settings/keys.tsx` and `ManageSubtitleDialogV2.tsx` read no `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`; all rendered state comes from server responses. Rule 23 does not apply and no `clockTime` fixtures are owed. Reference: `project-context.md` Rule 23.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 2.1 · PRD FR25 · architecture #D9 (NFR-S3, :280-284) · spec §5]
- [`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`:373-378 — the dead end + the `data-testid`; header comment :13-15 for the capability-honor precedent]
- [`apps/web/src/routes/settings/qbittorrent.tsx` — the page precedent; `routes/settings/` — the 11 existing siblings]
- [`sub-2-1a-key-resolution-api.md`#AC #1/#3/#4 · `sub-1-7a-subtitle-status-badge-design.md`#AC #7 (F5 copy)]
- [`project-context.md`#Rule 5/20/24 · feedback: no background tests, format before commit, design verification]

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO** beyond the pre-recorded item: state `N/A — no further out-of-scope work discovered`.
  - **① clarified in place:** the epic's 「前往設定」**死迴圈** is precisely a dead **end** (`/settings` renders; it has no key surface) — absorbed as AC #4's re-point ruling rather than a navigation fix.
- Reference: `project-context.md` Rule 24.

### File List
