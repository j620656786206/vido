# Story 9R-13a: TV .nfo 在地化 —— tvshow.nfo / 每集 nfo 的 backup-and-replace 路線

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c（Track 6 尾款）· **Priority:** P1（differentiator 的 TV 半邊）· **Depends:** 9R-13 ✅（movies 版已出貨）、9R-6/9R-7 ✅
**Source:** sprint-status `9R-13a-tv-nfo-localization`（Rule-24 ③ from 9R-13，S1 movies-first re-spec 遞延）。2026-08-19 策略重審：.nfo 在地化是 ADR 自封的「category-level differentiator」，但 TV 半邊缺席 —— 而繁中 NAS 使用者的庫**以 TV/動漫為大宗**，差異化最薄的地方正是最重要的地方。

---

## Story

As a NAS owner whose library is mostly TV shows,
I want tvshow.nfo and per-episode .nfo localized to zh-TW like movies already are,
so that the player shows 繁中劇情與集名 for the media type that dominates my library — with my originals always backed up before any replace.

---

## Context —— 為什麼 TV 是 replace-only（S1 實證）

Spike S1（#134）證明：電影有兩個被辨識的 nfo 槽位（`<basename>.nfo` / `movie.nfo`），寫到空槽即 additive；**TV 的名稱是單槽**（`tvshow.nfo`；每集 `<episode-basename>.nfo`）—— 語言字尾版本三家播放器都不認，**沒有 additive 選項**。因此 TV 路線是 sprint-status 條目裁定的「backup-and-replace only」：先備份 `.nfo.orig`（僅首次，movies 版 `writeAdditiveNFO` both-occupied 分支的既有先例，`nfo_localizer_service.go:202+`），再原槽覆寫。**覆寫即使有備份仍是侵入性寫入 → 需 opt-in**（條目原文「needs episode metadata + replace opt-in UX」）。

可鏡射的既有件：`NFOLocalizerService.LocalizeMovieNFO`（DB 為 metadata 主來源、glossary fail-soft、單批 `TranslateRequest` 翻譯 title/plot/genres/**cast roles**、fail-soft per field、`marshalNFO`）。`SeriesNFO` struct 與 `marshalNFO` 已在 `nfo_generator.go`；episode 的 NFO struct 是本案淨新增。

## Acceptance Criteria

### AC #1 — `LocalizeTVShowNFO(ctx, series)`
- 翻譯 series title / plot / genres ＋ cast roles（若 series 側查得到角色資料；查不到則略過該欄，**明示的範圍縮減**而非靜默漏掉 —— movies 版 `translateFields` 有翻 `Actors[i].Role` 的先例）。9R-7 單批 + per-show glossary，鏡射 movies 版逐步驟；originaltitle / 人名 / 年份 / uniqueids 保留。
- 寫入 `tvshow.nfo`：存在 → 首次備份 `tvshow.nfo.orig` 後覆寫（備份已存在則不覆蓋備份）；不存在 → 直接寫入（無原檔可保護）。

### AC #2 — `LocalizeEpisodeNFO(ctx, episode)`
- 翻譯集名（`Episode.Title`）+ 集簡介（`Episode.Overview`，NullString —— **episodes 表沒有 plot 欄位**，`Overview` 映射到 NFO `<plot>`，與 movies 版 `movieToNFO` 的 Overview→Plot 映射同款；NULL fail-soft 保留原值）；glossary key 用 **series 層級** id（sub-5-5 H1 的 glossaryMediaKey 教訓，`transcription_service.go:801-812` 先例，測試釘住）。
- 寫入 `<episode-basename>.nfo`，備份語意同 AC #1。

### AC #3 — Replace opt-in（紅線）
- 新路由 `POST /series/:id/localize-nfo`（含 `?include_episodes=true` 批次整劇），request body 需帶 `"confirm_replace": true` —— 缺席或 false → 409 + 錯誤碼說明 TV 為 replace-only（沿用既有 NFO 錯誤前綴，不發明新 code family）。單集路由 `POST /episodes/:id/localize-nfo` 同語意。
- 🔴 未經 confirm 的呼叫**零檔案寫入**（測試釘住）。

### AC #4 — 批次整劇的 fail-soft
- `include_episodes` 逐集處理，單集失敗記錄並續行（Rule 13 case 3）；回傳 result 含成功/失敗/跳過計數與各檔路徑（鏡射 `NFOLocalizeResult`）。

### AC #5 — 測試
- 槽位矩陣：tvshow.nfo 存在/不存在 × 備份已存在/未存在；episode 同款。
- confirm 缺席 → 409 且零寫入；glossary series-key 釘；欄位 fail-soft 釘。
- movies 版 `LocalizeMovieNFO` 行為 byte-unchanged（回歸釘）。

### AC #6 — 範圍紅線與已知留白
- **FE 表面不在本案**：連 movies 版路由都還零前端呼叫者（策略重審實證）—— nfo 的 UI 曝光是獨立的 FE story（建檔於 Dev Notes，援引 `9R-10a` 正在處理的 ManageSubtitleDialogV2 動線討論）。
- 不做自動觸發（nfo 在地化納入 9R-10b 常設同意政策的範圍問題，留該 story AC #1 裁定時一併考慮）。

## Tasks / Subtasks

- [ ] Task 1: AC #1 LocalizeTVShowNFO＋備份寫入
- [ ] Task 2: AC #2 LocalizeEpisodeNFO（series-key glossary）
- [ ] Task 3: AC #3 路由＋confirm 閘門
- [ ] Task 4: AC #4 批次 fail-soft＋AC #5 測試矩陣
