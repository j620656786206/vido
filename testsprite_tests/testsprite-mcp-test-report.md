# TestSprite AI Testing Report (MCP) — round2 (CLOSED)

---

## 1️⃣ Document Metadata
- **Project:** vido
- **Round:** testsprite-v2-round2 — scanner + subtitle + qBittorrent-settings journeys
- **Dates:** run 2026-07-23; closeout 2026-07-23
- **Env:** local production build at `:8090` via `scripts/serve-test-env.sh` (seeded sqlite: 2 libraries / 17 movies / 3 series; no TMDb key, no subtitle-provider keys, no qBittorrent — all by design)
- **Selection:** 17 cases (env-safe subset — no live qBittorrent / TMDb / subtitle-provider key required), `serverMode: production`
- **Credits:** 28 spent (120 → 92)

---

## 2️⃣ Requirement Validation Summary

### qBittorrent Settings Form
- **TC035** Load settings page — ✅ Passed
- **TC038** Connection test failure shows inline error — ✅ Passed
- **TC039** Save blocked until successful test — ✅ Passed
- **TC040** Empty-host validation — ⛔ raw BLOCKED → **re-authored → ✅ Passed (rerun)**. The test mistook the Host **placeholder** (`http://192.168.1.100:8080`) for a value. v2 validation = the 測試連線 button is **disabled** while fields are empty (already covered by TC039). Re-authored to assert the disabled state.

### Scanner Settings & Manual Scan
- **TC063** Settings page loads — ✅ Passed
- **TC065** Minimize to pill / expand — ✅ Passed
- **TC066** Cancel active scan w/ confirm — ✅ Passed
- **TC068** Schedule persists — ✅ Passed
- **TC069** Global progress card off-settings — ✅ Passed
- **TC064 / TC067 / TC070** Scan progress card appear / cancel-dismiss / completion — ❌ raw FAILED → **surfaced a REAL product defect** (`bugfix-scan-progress-sse-unwired`): the scan-progress SSE was never opened from the UI trigger, so the card was unreachable in v2. **Fixed this round.** These transient SSE cases are the wrong tool for TestSprite (the seeded scan finishes <100 ms), so they are **migrated to `tests/e2e/scan-progress.spec.ts`** (mocked SSE, deterministic, runs in CI).

### Subtitle Search Dialog
- **TC071** Open from context menu — ✅ Passed
- **TC076** Close with Escape — ✅ Passed
- **TC077** Open from detail panel — ✅ Passed
- **TC078** Context-menu option present — ✅ Passed
- **TC073** Empty-state on no results — ⛔ raw BLOCKED → **re-authored → ✅ Passed (rerun)**. The empty state renders correctly (`尚無結果 — 線上來源成功率低，建議改用生成字幕`); the test wrongly assumed a manual query input (v2 auto-searches). Re-authored to click 搜尋 + assert the empty-state text.

---

## 3️⃣ Coverage & Matching Metrics

- **Raw first-pass:** 12 / 17 (70.59%).
- **After closeout:** **14 / 14 in-scope GREEN** (12 first-pass + TC040/TC073 re-authored→rerun) **+ 1 real defect found & FIXED + 3 migrated to Playwright.**

| Requirement | Total | Raw ✅ | Post-closeout | Real defects |
|---|---|---|---|---|
| qBittorrent Settings Form | 4 | 3 | 4 ✅ | 0 |
| Scanner Settings & Manual Scan | 8 | 5 | 5 ✅ + 3 → Playwright | **1 (fixed)** |
| Subtitle Search Dialog | 5 | 4 | 5 ✅ | 0 |
| **Total** | **17** | **12** | **14 green + 3 migrated** | **1 fixed** |

---

## 4️⃣ Key Gaps / Risks

1. **Real defect FIXED — scan-progress card was dead in v2** (`bugfix-scan-progress-sse-unwired`). The shell `<ScanProgress>` mounts `useScanProgress()` but the SSE opens only via `startTracking()`, which nothing called; the 掃描媒體庫 trigger is a separate hook instance. Fix: a lazy module-level signal (`requestScanTracking`/`subscribeScanTracking`) — the trigger signals the shell instance to connect (kept lazy so the 7 networkidle-based e2e specs are unaffected). **Note:** my first-pass triage wrongly called this a "too-fast-scan artifact" — the adversarial verification pass caught the real cause. Gates: web vitest 2457/2457, prettier clean, e2e spec added.
2. **Scheduled(background)-scan card visibility** not covered by the trigger signal (button-trigger only) — acceptable follow-up.
3. **Provider-key-dependent surfaces** remain deliberately out of scope (subtitle scored-results/preview/download, qBittorrent success path, TMDb metadata search) — need a keyed/mocked env.
4. **Positive:** scanner (settings/schedule/global-card/cancel), qBittorrent settings form (load/failure/save-gating), and the subtitle dialog (all entry points + close + empty state) all behave correctly in v2.
