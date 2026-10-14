# Disc：「產生字幕」的選片清單把已經有繁中的片當成缺中文，報價要你付錢

Status: review

**Source:** 2026-10-14 正式機 0.2.0 smoke test。泰德拉索 S01E02 季清單徽章說「zh_hant」（內嵌繁體 subrip 軌），選片清單卻報價 US$0.24；跑的那集神祕關卡 S01E10 旁邊本來就有 `.zh-TW.srt`，白花 US$0.055。Alexyu 選 **A：現在修**（「是 bug 不是新功能，影響測試者的第一個畫面」）。

## 查到的事（2026-10-14，main `410b11d5`）

- 候選枚舉用的是 SQL 條件 `missingZhHantSubtitleEpisodeWhere`（`apps/api/internal/repository/episode_repository.go:155-158`）：`subtitle_language IS NULL OR != 'zh-Hant'`＋有檔案。電影版 `missingZhHantSubtitleWhere`（`movie_repository.go:999-1003`）一樣。`subtitle_language='zh-Hant'` 只有 Vido 自己放完字幕才會寫（9R-16 AC 12）。
- 五個地方吃這個 predicate：選片清單 `generation_candidates.go:1361/1384`、批次預覽計數 `generation_batch.go:464/470`、批次「missing」範圍 `:582`、`subtitle/worker_pool.go:401/418`、`subtitle/auto_generation.go:572/592`。
- 季清單徽章用的是 `models.ChineseSubtitleVerdict(status, language, tracksJSON)`（`chinese_subtitle.go:309-349`）：看 `subtitle_tracks` JSON 裡每條軌的語言標籤／標題／抽樣判定，繁體勝簡體勝不明；`.zh-TW` 旁檔是 `external:true` 的一條軌（`episode_chinese_subtitle.go:153-160` `sidecarTag`）。
- 正式機數字（DB 直查）：2,398 個候選裡 558 個的 `subtitle_tracks` 含 chi／繁／zh-TW／Traditional，171 個含 `.zh-TW` 旁檔；泰德拉索 S01E02 `subtitle_status=not_searched`、`subtitle_language` 空、tracks 第 7 條 `chi` "Traditional Chinese (繁體中文)"；API `GET /series/{id}/seasons/1/episodes` 對同一集回 `"chinese_subtitle":"zh_hant"`。
- 候選的路線分類 `classify`（`generation_candidates.go:1454`）用 `predictor.FromTracks` 只分「有文字軌→extract／沒有→asr」，不分中文。
- ⚠️ 旁檔若尚未被背景掃描併進 `subtitle_tracks`（掃描 `skipped=2159` 表示檔案簽章沒變就不重讀），本修法還是會漏；徽章是即時讀目錄才看得到。先用儲存的軌，之後若正式機還有漏再加即時讀旁檔。

## Acceptance Criteria

1. `FindMissingZhHantSubtitle`（episode／movie）不再回傳 `ChineseSubtitleVerdict` 為 `zh_hant` 的列——同一條規則給徽章、篩選、產生字幕三處用。簡體或不明的仍回傳（還需要轉換或判斷）。
2. `CountMissingZhHantSubtitle`（episode／movie）跟枚舉同步：刪掉一樣的列。
3. 五個消費者（選片清單、批次預覽計數、批次 missing 範圍、worker pool、自動生成）自動跟著正確，不另寫邏輯。
4. 測試：episode／movie 各一條——內嵌「繁體中文」軌、`.zh-TW` 旁檔 → 排除；簡體軌、純英文軌 → 保留；count 等於枚舉長度。
5. `go test ./internal/models/ ./internal/repository/ ./internal/services/ ./internal/subtitle/`、vet、staticcheck 綠。
6. 正式機驗證：重新 analyze，候選數從 2,398 降、泰德拉索 S01E02 不在清單；季清單徽章不變。

## Tasks / Subtasks

- [x] T1 `models.HasTraditionalChineseSubtitle`（AC 1）
- [x] T2 兩個 repository 的 Find／Count（AC 1–3）
- [x] T3 測試（AC 4）
- [x] T4 全套檢查（AC 5）
- [ ] T5 正式機驗證（AC 6，合併後）

## Dev Notes

- 選在 repository 層改，而不是在選片清單服務裡過濾：五個消費者只有一個真相，批次預覽計數才不會跟清單對不上。
- 簡體軌的片仍在清單裡但其實跑起來是免費的（router 走 convert）——報價偏高，屬另一件事，立案見 Discovery Triage。
- 不動 `subtitle_language` 的寫回規則。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- `models.HasTraditionalChineseSubtitle` 包住既有的 `ChineseSubtitleVerdict`；episode／movie 的 `FindMissingZhHantSubtitle` 在 SQL 之後用它過濾，`CountMissingZhHantSubtitle` 改成只撈三個欄位再用同一條規則數（共用 `countMissingZhHantRows`）。
- 測試各一條：內嵌「繁體中文」軌、`.zh-TW` 旁檔排除；簡體軌、純英文保留；count＝枚舉長度。
- `go test` models／repository／services／subtitle／handlers、vet、staticcheck 綠。

### Discovery Triage

- ③ 簡體軌（`zh_hans`）的片仍被報「extract」價，實際 router 會免費轉換 → `disc-2026-10-candidates-simplified-quoted-as-paid`（sprint-status 立案）。
- ③ 旁檔尚未進 `subtitle_tracks` 的片（背景掃描簽章沒變不重讀）本修法會漏 → 併入同一張單觀察正式機數字再決定要不要即時讀目錄。

### File List

- `apps/api/internal/models/chinese_subtitle.go`、`apps/api/internal/repository/{episode_repository.go,movie_repository.go,episode_generation_test.go,movie_repository_test.go}`
