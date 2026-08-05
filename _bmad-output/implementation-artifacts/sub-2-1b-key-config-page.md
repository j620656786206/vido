# Story sub-2.1b: Key-configuration page + dead-end fix (NFR-S3)

Status: done

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

- [x] **Task 1 — Page (AC #1):** `routes/settings/keys.tsx` mirroring `qbittorrent.tsx`; TanStack Query hooks for GET/PUT/test; three-source row states; settings-index entry.
- [x] **Task 2 — Degradation + NFR-S3 (AC #2, #3):** `writable:false` read-only mode; `isSecureContext` warning + per-visit confirmation gate.
- [x] **Task 3 — Dead end (AC #4):** re-point `ManageSubtitleDialogV2.tsx:373` (read 1-7a's Completion Notes first); re-point any sibling affordance.
- [x] **Task 4 — Tests + gates (AC #5):** the five test groups; foreground vitest; `pnpm lint:all`.

---

## Dev Notes

- **Mirror `apps/web/src/routes/settings/qbittorrent.tsx`** — it is the same shape (secrets-backed settings + test action) and already solves layout, form state, and error rendering in this codebase's idiom.
- **Rule 5** — server state through TanStack Query; the key list is server state, the un-saved form is local state.
- **Rule 20** — this story **consumes** sub-2-1a's `[@contract-v1]` AC #1/#3. Record `confirmed against [@contract-v1] sub-2-1a AC #1` and `… AC #3` in Dev Notes at implementation. It stamps nothing.
  - **Recorded at implementation (2026-08-05):** confirmed against [@contract-v1] sub-2-1a AC #1 — `KeyResolver` resolves `secret` **before** `env`, and surfaces `source` so this page can say 「目前由環境變數提供」 honestly and warn that saving overrides it (`apps/api/internal/services/key_resolver.go:56-70`). confirmed against [@contract-v1] sub-2-1a AC #3 — the `GET/PUT/POST-test /api/v1/settings/keys` triad: GET returns `{keys:[{name,configured,source,masked?}],writable,reason?}` with `masked` present for `secret`-sourced keys only, PUT takes `{claude?,tmdb?,openai?}` where an omitted field is untouched and `""` deletes, POST-test takes `{claude}` and answers `{valid:true}` or a Rule-3 error envelope (`key_settings_handler.go`, `key_settings_service.go:17-39`). Neither version was bumped; this story stamps nothing.
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

Amelia (Developer Agent) — Claude Opus 5 (1M context), effort xhigh. Implemented 2026-08-05.

### Debug Log References

- `pnpm nx build web` — run once solely to regenerate `apps/web/src/routeTree.gen.ts` (the TanStack route tree is a tracked generated file and `@tanstack/router-generator` is not installed standalone; the Vite plugin is the only generator). Build succeeded, tree gained the 8 `SettingsKeys*` entries.
- Design verification screenshots (git-ignored `tmp/`, not committed): `keys-mixed-d.png`, `keys-mixed-m.png`, `keys-readonly-d.png`, `keys-readonly-m.png`, `keys-insecure-d.png`, plus `conn-d.png` of the shipped `/settings/connection` page as the comparison baseline. Captured against `nx serve web` with `**/api/v1/settings/keys` stubbed by Playwright `route.fulfill` (no backend needed) and `window.isSecureContext` overridden via `addInitScript` for the NFR-S3 state. Dev server stopped afterwards.

### Completion Notes List

**🔗 AC Drift: NONE** (checked: `grep -rn "go-to-settings\|/settings\b\|isSecureContext\|SETTINGS_CATEGORIES" _bmad-output/implementation-artifacts/*.md` plus a targeted read of `sub-2-1a-key-resolution-api.md` and `sub-1-7a-subtitle-status-badge-design.md` — all hits are REUSE, not DRIFT. The `data-testid="go-to-settings"` contract is preserved verbatim; only the `navigate()` target changed, which no prior AC pins. `SETTINGS_CATEGORIES` grew 9→10, which is additive — no prior AC fixes its length, only `SettingsLayout.spec.tsx` asserted it and that assertion was updated in the same change.)

**📎 Contract Stamps: FOUND** (2 stamped upstream ACs across 1 file — this story stamps none; `grep -nE '\[@contract-v[0-9]+\]' sub-2-1b-key-config-page.md` returns only the Rule-20 Dev Notes ack lines. Upstream `sub-2-1a-key-resolution-api.md` carries `[@contract-v1]` on AC #1 (KeyResolver) and AC #3 (the settings triad); both are acked verbatim in Dev Notes → Rule 20, and both are still at v1 — no bump, no stale-mark obligation.)

**🎭 A11y Pre-Flight: PASS** (3 components checked — `ApiKeysForm.tsx` (new), `SettingsLayout.tsx`, `ManageSubtitleDialogV2.tsx`; **0** jsx-a11y warnings on touched files and **0** repo-wide, `eslint` on the six touched paths exits clean. Manual pass over the four recurring classes: **responsive images** N/A (no `<img>`); **modal focus management** N/A and deliberately so — 清除 uses an inline two-step confirm instead of a modal precisely to avoid adding another hand-rolled dialog with no focus trap (the `RestoreConfirmDialog.tsx` pattern already has that gap); **aria-live on async-revealed content** — every panel revealed after the query resolves carries `role="status" aria-live="polite"` (`api-keys-load-error`, `keys-not-writable`, `insecure-context-warning`, per-row `key-test-result-*`, save success/error); **keyboard + ARIA semantics** — all controls are native `<button>`/`<input>`, every visible input has a real `<label htmlFor>`, hints are wired via `aria-describedby`, and the acknowledgement checkbox is inside its own `<label>`. Lazy-load contract N/A (no IntersectionObserver/pagination). One a11y defect found and fixed while building: a stored (`source: secret`) row draws no input, so its `<label htmlFor>` would have been an orphan — the label degrades to a `<span>` in that state.)

**Pre-existing fix: `pnpm lint:all` could not be green on any machine with local scratch files.** `tmp/` is git-ignored (`.gitignore:74`) but was NOT in `eslint.config.mjs`'s global `ignores`, so two pre-existing local scratch scripts (`tmp/shot.mjs`, `tmp/verify-posters.mjs`, top-level `await` in `.mjs`) produced 2 hard eslint **errors** and failed the story's own AC #5.6 gate. Nothing under `tmp/` can ever reach CI, so a stray scratch file must never turn a local gate red while the pipeline stays green — `tmp/**` added to the global ignores. Also ran Prettier over two unformatted untracked skill docs (`.agents/skills/testsprite-{onboard,verify}/SKILL.md`) that were failing `format:check` before this story. Both are Epic 9c Retro AI-2 option 1 (quick in-place fix).

**What shipped**

- **AC #1** — `/settings/keys` renders three rows (Claude（翻譯） · TMDB · 雲端 ASR（選配）) with a distinct affordance per `source`: `secret` → 「已設定」 + the `first6…last4` mask + 編輯/清除; `env` → 「目前由環境變數提供」 + an explicit 「在此儲存的金鑰會覆蓋環境變數提供的設定。」; `none` → 「尚未設定」 + an empty field. Every input is `type="password"`, never seeded from the server (2-1a returns masks only — there is no value to fetch), and 編輯 opens an **empty** field rather than pre-filling the mask. 儲存 sends a partial PUT carrying only the rows that were typed into. 清除 sends an explicit `""` after an inline confirm. Server state via TanStack Query (Rule 5); the un-saved form and the HTTPS acknowledgement are local state.
- **AC #2** — `writable:false` renders read-only: inputs, 儲存, 編輯 and 清除 are **disabled, not hidden**, with 「未設定加密金鑰，無法安全儲存 API 金鑰 —— 請設定 ENCRYPTION_KEY 後重啟。」 and the current state still visible. 測試 stays **enabled** on purpose — probing writes nothing, so an operator without `ENCRYPTION_KEY` can still find out whether the env key they deployed actually works.
- **AC #3** — the gate reads `window.isSecureContext`, so localhost-over-HTTP (the dev/first-run path) is warning-free by definition and users are never trained to dismiss the one warning that matters. Warning + acknowledgement checkbox above the form; 儲存 blocked until ticked; the tick lives in component state only (a remount starts unticked). Advisory, never a hard block.
- **AC #4** — `ManageSubtitleDialogV2.tsx:373` now navigates to `/settings/keys`, `data-testid="go-to-settings"` unchanged.
- **AC #5** — 37 new tests (28 component + 9 service) plus 1 added to the dialog spec; full `pnpm nx test web` **227 files / 2520 tests green**, full `pnpm nx test api` green, `pnpm lint:all` green, `pnpm run format:check` green. Foreground only; `pnpm run test:cleanup` reports no residual workers.
- **AC #6 scope fence honoured** — 0 files under `apps/api/**`, no provider/model selection UI, no local-worker-URL field, no `.pen` edits, no screenshot regeneration, no visual baselines.

**Findings worth carrying forward**

1. **The story's stated precedent is stale, and the real one is better.** `apps/web/src/routes/settings/qbittorrent.tsx` is a **7-line redirect stub** to `/settings/connection` — it has not been a page for some time. The live secrets-backed-settings precedent is `components/settings/QBittorrentForm.tsx` rendered by `routes/settings/connection.tsx`, and the settings **card** (`rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 p-6`) lives on the ROUTE, not the form. First pass missed the card and the page rendered visually inverted against the design (transparent rows, lighter inputs). Caught by screenshotting the shipped page side by side; the page now reuses the identical card so the two sit at the same tone and radius by construction.
2. **「Reachable from the settings index」 has no index to add to.** `/settings/` is itself a redirect to `/settings/connection`; `SettingsLayout`'s sidebar + mobile tab strip IS the index. The entry was added there (`金鑰設定` / `金鑰`, `KeyRound`, second position next to 連線設定 — both configure how Vido reaches an external service), which also keeps the route from being orphaned.
3. **`POST /settings/keys/test` probes Claude and nothing else** — its request body is `{claude}` and its tester is `ClaudeProviderHolder`. AC #1's 「測試 action per key」 is therefore only half-backed by 2-1a. Rather than hide the control on TMDB/雲端 ASR, both rows draw 測試 **disabled with 「目前僅支援 Claude 金鑰測試」** — the repo's own capability-honor idiom (`SettingsLayout`'s Coming Soon entries, the dialog's 影集字幕生成即將推出).
4. **測試 can leak a typed key too.** AC #3 gates only 儲存, but 測試 POSTs the typed candidate in cleartext just as 儲存 does. The gate was therefore extended to disable 測試 **when — and only when — a candidate has been typed**; testing the already-resolved key transmits no secret and stays available.
5. **2-1a CR L1 honoured.** The encryption-key 409 and the no-key 409 both carry `AI_NOT_CONFIGURED`, so read-only mode is driven entirely by GET's `writable`/`reason` and never by branching on the error code. Backend messages are surfaced verbatim rather than re-written, because 2-1a deliberately distinguishes a bad key from a deprecated model id from a quota ceiling — three different next actions that flattening would destroy.
6. **The TMDb restart caveat is stated in the UI**, as `backlog-tmdb-runtime-key-resolution` explicitly asked of this story: 「用於中繼資料與海報。儲存後需重啟伺服器才會生效。」
7. **Sibling 前往設定 affordances checked and cleared, deliberately untouched.** `DownloadPanel.tsx:147` and `DownloadsStatesV2.tsx:123` both point at `/settings` — correctly, since both are qBittorrent-connection failures and `/settings` redirects to `/settings/connection`. Re-pointing them at `/settings/keys` would be wrong.

### 🎨 UX Verification: PASS (with one drift routed to Sally)

Compared against `_bmad-output/screenshots/flow-c-search-settings/c4-d.png` + `c4-m.png` (Screen 10 Settings Desktop/Mobile, `6UCtX`) and `flow-f-subtitle-v2/f5-d-v2.png`, plus a live capture of the shipped `/settings/connection` page as the implementation baseline.

| Area | Design Spec | Implementation | Match? | Fix Needed |
|---|---|---|---|---|
| Settings shell | c4-d/c4-m: desktop sidebar with icon+label rows and an accent active row; mobile horizontal pill tabs | `SettingsLayout` reused unmodified; only a 10th category added | ✅ | — |
| Nav entry | 12th `/settings/*` sibling, same row treatment | 金鑰設定 / 金鑰 with `KeyRound`, active-state accent identical | ✅ | — |
| Page header | Title then a one-line description directly beneath | `金鑰設定` h1 (`mb-2`) + description (`mb-6`) | ✅ | — |
| Form container | Single rounded card, mid-tone fill, subtle border, generous padding | Same classes as `routes/settings/connection.tsx` | ✅ | Fixed mid-story (see Finding 1) |
| Field treatment | Label above a full-width input darker than the card | Label + `type="password"` input, `INPUT_CLASS` copied from `QBittorrentForm` | ✅ | — |
| Primary action | Filled accent button, bottom-right inside the card | 儲存, bottom-right inside the card | ✅ | — |
| Secondary action | Neutral-fill button beside the primary | 測試 / 編輯 / 清除, `--bg-tertiary` fill; placed **per row** rather than in the card footer | ⚠️ Deliberate | None — AC #1 mandates a per-key 測試; a footer action cannot address one of three keys |
| Mobile | c4-m: tabs scroll, card stacks full-width, no horizontal overflow | Verified at 390×900 — stacks, scrolls, no overflow | ✅ | — |
| F5 尚未設定 panel structure | f5-d-v2: warning-tint panel, icon + title + body + right-aligned affordance | Unchanged; only the button's `navigate()` target moved | ✅ | — |
| **F5 panel copy** | f5-d-v2 (post-1-7a, M1): 「尚未設定翻譯服務金鑰」 / 「請設定 CLAUDE_API_KEY 環境變數後重啟伺服器。設定頁面將於 M1.5 提供。」 + 查看部署說明 | Code still carries the **pre-1-7a** copy: 「字幕生成尚未設定」 / 「需要 FFmpeg 與 AI API Key…」 | ❌ | **NOT this story** — routed to Sally as `backlog-f5-not-configured-panel-copy-ruling` (lane ③) |

The last row is a genuine drift and deliberately not fixed here. sub-1-7a was design-only and no story has landed its code half; more importantly the ratified string was written against the **wrong trigger** — this panel fires on 503 `TRANSCRIPTION_DISABLED` (FFmpeg + ASR), not the 409 `AI_NOT_CONFIGURED` (Claude translation key) it is documented as being byte-aligned with, so pasting it in would introduce a new inaccuracy rather than remove one. Deciding whether F5 stays one panel or splits is a design ruling, and AC #6 fences this story out of it.

### Senior Developer Review (AI) — 2026-08-05

**Outcome: Approve (all findings fixed same session).** Adversarial CR (Amelia-as-reviewer, Fable 5): **0 High / 4 Medium / 2 Low — all 6 fixed.** Git-vs-File-List discrepancies: 0. Rule 7: N/A (no Go files). Rule 20: N/A (acks only, both verified against upstream v1). Rule 25: N/A. All 6 ACs verified IMPLEMENTED; all 4 [x] tasks verified with evidence; TestSprite TC002 hits only the API URLs, so the `go-to-settings` re-point carries no selector/URL-assertion risk.

- ✅ **[M1] Stale test verdict vouched for a key it never tested** — 編輯 → type → 測試 → 「金鑰驗證成功」 → 取消 left the green verdict standing next to the STORED key; typing a new candidate also kept the old verdict. 清除 already reset — three paths, two behaviors. Fix: edit-cancel and onChange both void that row's `testResults`. +2 tests.
- ✅ **[M2] Read-only 編輯/清除 path had zero coverage** — the READ_ONLY fixture carried only env/none sources, so the `secret`-row `disabled={!writable}` branch was never exercised. +1 test (secret-sourced row under `writable:false` → 編輯/清除 `toBeDisabled()`, state + mask still visible).
- ✅ **[M3] The NFR-S3 gate extension to 測試 had zero coverage** — the story's own deviation (typed candidate + unacked insecure ⇒ 測試 disabled) could regress silently. +1 test covering the three-position contract (no candidate → enabled; typed → disabled; acked → enabled).
- ✅ **[M4] `useKeySettings` cache-seeding untested** — the "PUT response seeds the cache, no second round trip" claim had no test at any layer (component spec mocks the hooks; service spec stops at fetch), and hooks/ has a spec convention. NEW `useKeySettings.spec.tsx` (4 tests): key hierarchy, GET-through-service, **setQueryData seeding with `getKeys` called exactly once**, refused-save leaves cache untouched.
- ✅ **[L1] `handleTest` accepted any `KeyName` but hardcoded `testClaudeKey`** — a future `testable: true` on another row would POST that key as a Claude candidate. Fix: explicit `if (name !== 'claude') return` guard with the reason in a comment.
- ✅ **[L2] Load-error rendered 尚未設定 on all rows** — unknown ≠ not set; a server outage must not badge a configured key as unset. Fix: `isError && !state` ⇒ 「無法確認」 badge. +2 assertions in the load-error test.

Gates re-run post-fix: `pnpm nx test web` **228 files / 2528 tests green** (+1 file, +8 tests), `pnpm lint:all` 0 errors, `format:check` green, no residual workers.

### Discovery Triage

- **Did this story discover any work outside its current scope?** Yes — one new item, plus the pre-recorded one.
  - **① clarified in place:** the epic's 「前往設定」**死迴圈** is precisely a dead **end** (`/settings` renders; it has no key surface) — absorbed as AC #4's re-point ruling rather than a navigation fix.
  - **① absorbed in place (no new AC needed):** two pre-existing local-gate blockers fixed inline under the Epic 9c Retro AI-2 quick-fix rule — `tmp/**` added to `eslint.config.mjs` global ignores, and two untracked skill docs run through Prettier. Both are recorded above under "Pre-existing fix"; neither adds behaviour.
  - **③ backlog with carry-forward link:** `backlog-f5-not-configured-panel-copy-ruling` — the F5 尚未設定 panel's code copy vs sub-1-7a's ratified `.pen` copy, blocked on a Sally ruling because the ratified string targets a different capability than the one that triggers the panel. Bidirectional: the sprint-status entry names this story, and the UX Verification table above names the entry.
- Reference: `project-context.md` Rule 24.

### File List

**Added**

- `apps/web/src/services/keySettingsService.ts`
- `apps/web/src/services/keySettingsService.spec.ts`
- `apps/web/src/hooks/useKeySettings.ts`
- `apps/web/src/hooks/useKeySettings.spec.tsx` (CR M4)
- `apps/web/src/components/settings/ApiKeysForm.tsx`
- `apps/web/src/components/settings/ApiKeysForm.spec.tsx`
- `apps/web/src/routes/settings/keys.tsx`

**Modified**

- `apps/web/src/routeTree.gen.ts` (generated — `/settings/keys` registered)
- `apps/web/src/components/settings/SettingsLayout.tsx` (10th category)
- `apps/web/src/components/settings/SettingsLayout.spec.tsx` (nav/tab/label assertions + count 9→10)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` (AC #4 re-point)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx` (`/settings/keys` route + navigation assertion)
- `eslint.config.mjs` (pre-existing fix — ignore git-ignored `tmp/**`)
- `.agents/skills/testsprite-onboard/SKILL.md`, `.agents/skills/testsprite-verify/SKILL.md` (pre-existing fix — Prettier only)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (status transitions + `backlog-f5-not-configured-panel-copy-ruling`)
- `_bmad-output/implementation-artifacts/sub-2-1b-key-config-page.md` (this file)

## Change Log

| Date | Change |
|---|---|
| 2026-08-05 | Task 1 — `/settings/keys` page: `keySettingsService` + `useKeySettings` hooks (Rule 5) + `ApiKeysForm` with three per-`source` row states, per-key 測試, partial-PUT 儲存, confirm-then-`""` 清除; registered as the 10th `SettingsLayout` category so the route is reachable and not orphaned. |
| 2026-08-05 | Task 2 — `writable:false` read-only mode (inputs/儲存/編輯/清除 disabled **with a reason**, 測試 still live because probing writes nothing) + NFR-S3 `window.isSecureContext` warning with a per-visit acknowledgement gating 儲存, extended to 測試 whenever a typed key would cross the wire. |
| 2026-08-05 | Task 3 — AC #4: `ManageSubtitleDialogV2.tsx:373` re-pointed from `/settings` (a dead **end**) to `/settings/keys`, `data-testid="go-to-settings"` preserved. Sibling `前往設定` affordances in `DownloadPanel`/`DownloadsStatesV2` verified as correctly qBittorrent-bound and left alone. |
| 2026-08-05 | Task 4 — 37 new tests (28 `ApiKeysForm.spec.tsx` + 9 `keySettingsService.spec.ts`) + 1 dialog-navigation test; full regression `pnpm nx test web` (227 files / 2520 tests) and `pnpm nx test api` green; `pnpm lint:all` + `format:check` green; a11y pre-flight PASS. |
| 2026-08-05 | UX verification against `c4-d`/`c4-m`/`f5-d-v2` + the shipped `/settings/connection` page: adopted the route-level settings card so tone/radius match the sibling page. F5 panel copy drift routed to Sally as `backlog-f5-not-configured-panel-copy-ruling` (Rule 24 lane ③). |
| 2026-08-05 | Pre-existing fixes (Epic 9c Retro AI-2 option 1): `tmp/**` added to `eslint.config.mjs` ignores so a git-ignored scratch file can no longer fail `pnpm lint:all` locally; two untracked skill docs Prettier-formatted. |
| 2026-08-05 | Adversarial CR: 4M/2L found, all fixed — stale test-verdict voiding (M1), read-only 編輯/清除 coverage (M2), NFR-S3 測試-gate coverage (M3), new `useKeySettings.spec.tsx` cache-seeding tests (M4), claude-only probe guard (L1), 無法確認 badge on load error (L2). Web suite 228 files / 2528 tests green. Story → done. |
