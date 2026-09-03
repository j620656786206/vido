# Story 6.9: TMDb attribution —— logo + 未經認可聲明（前端，合規）

Status: ready-for-dev

## Story

As the operator of a public self-hosted app,
I want Vido to display the TMDB logo and the required non-endorsement notice wherever TMDB data is used,
so that we comply with the TMDB API Terms of Use §3 before the app reaches anyone outside this house.

## Context — 這個 story 為什麼存在

party-mode 2026-09-03 查證：TMDB 條款第 3 條要求 logo 與「This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.」JustWatch 的 attribution 已做（`StreamingAvailability.tsx:179`，註解標「mandatory licensing requirement」），**TMDb 的完全沒有**。內嵌預設 TMDb key（P1-7）與任何對外測試都以此為前提。

## Acceptance Criteria

1. **設定頁「資料來源」段。** 在設定 shell（Screen C4-D `6UCtX`）的 API 金鑰區塊（`ApiKeysForm.tsx` TMDB 列下方）新增 attribution：TMDB 官方 logo（SVG，從 TMDB brand 頁下載的「TMDB — The Movie Database」primary short logo，放 `apps/web/public/images/tmdb-logo.svg`，檔頭註明來源 URL 與下載日期）＋兩行文案：英文原句（條款要求原文）＋ zh-TW 說明「本應用程式使用 TMDB 與 TMDB API，但未經 TMDB 認可、認證或核准。」logo 連到 `https://www.themoviedb.org/`（`rel="noopener noreferrer"`）。

2. **詳情頁頁尾。** 電影／影集詳情頁（TMDB 資料的主要消費面）在 `StreamingAvailability` 的 JustWatch attribution 同區塊下方加一行小字「資料來源：TMDB」＋小 logo（沿用同一 className 與 `data-testid="tmdb-attribution"`）。

3. **設計。** 兩處都 ride 既有 frame：設定頁循 `ApiKeysForm.tsx:1` 的 `// Design ref: … no current screen frame` 先例；詳情頁 attribution 列循 JustWatch 既有樣式。**不改 `.pen`**（純文字＋小 logo 行，Sally 在 party-mode 已認可「合規行不佔設計位」）；若 Sally 要求入 `.pen`，改走 `CLAUDE.md` 截圖流程。

4. **主題與 a11y。** logo 在 light/dark 都可讀（TMDB 提供的藍綠漸層版兩者皆可；若對比不足用單色版）；`alt="TMDB"`；連結有可見焦點。

5. **測試。** `ApiKeysForm.spec.tsx`／`StreamingAvailability.spec.tsx` 加：logo `img` 存在、alt、英文原句逐字、連結 href；visual gallery fixture 更新（`-darwin` 本機、`-linux` 等 CI）。

## Tasks / Subtasks

- [ ] **Task 1 — 資產與設定頁（AC: #1, #3, #4）**
- [ ] **Task 2 — 詳情頁列（AC: #2）**
- [ ] **Task 3 — 測試與 fixtures（AC: #5）**

## Dev Notes

- 英文原句**不得改寫**（條款原文）。
- `README.md` + `README.zh-TW.md`（Rule 17）若有「資料來源」段一併補 TMDB 一行。
- 純前端 ≤3 task。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched。

### References

- TMDB API Terms of Use §3 — https://www.themoviedb.org/api-terms-of-use
- eval-1「後續 Backlog」P0-9；`apps/web/src/components/media/StreamingAvailability.tsx:176-192`、`apps/web/src/components/settings/ApiKeysForm.tsx`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
