# Disc：首頁「繁中字幕 X/Y」改用篩選與標籤同一套判斷

Status: done

**Source:** filed 2026-10-06 by `disc-2026-10-subtitle-filter-disagrees-with-badges`（Rule 24 lane ③）。⚖️ 2026-10-08 **Alexyu 裁定 A**：已覆蓋＝判斷結論為「繁中」的片子，不管繁中從哪來（Vido 生成、官方字幕檔、片內繁中軌）。

## Story

身為片庫的使用者，
我要首頁「繁中字幕 X/Y」把已經有繁中的片子都算成已覆蓋，
這樣首頁不會對已經有官方繁中或片內繁中的片子一直叫我「產生字幕」，數字也跟篩選、海報標籤講同一件事。

## 問題

首頁的已覆蓋數是第三套「有沒有中文」的算法：電影只數 `subtitle_language = 'zh-Hant'`（Vido 自己記錄過的繁中）；影集要每一集都有這個記錄。片內有繁中軌、或旁邊放了官方繁中檔的片子都不算，所以首頁一直顯示有沒覆蓋的、一直給「產生字幕」。

## Acceptance Criteria

1. 電影已覆蓋＝有檔案、沒被移除、且 `vido_chinese_subtitle(...)` 結論是 `zh_hant`（跟海報標籤、篩選同一個函式）。
2. 影集已覆蓋＝沒被移除、至少一集有檔案、且影集彙總結論（`seriesChineseSubtitleSQL`，各集裡最需要處理的那個）是 `zh_hant`。有一集是簡中／看不出繁簡／缺中文／不知道，整部就不算。
3. 只有簡中、只有「中文（看不出繁簡）」、缺中文、不知道 → 都不算已覆蓋（標題是「繁中字幕」）。
4. 分母（總數）不變；已覆蓋 ≤ 總數仍成立；首頁畫面、文案、按鈕邏輯不動。
5. 測試用真 migration 鏈：Vido 繁中、官方繁中字幕檔、片內繁中軌三種來源都算；簡中、未標繁簡的 chi、只有英文、沒讀過都不算；沒有檔案／被移除不算；影集一集不是繁中整部不算。
6. 檢查：`go build／vet`、`go test ./...`、`staticcheck` 全綠。

## Tasks / Subtasks

- [x] T1（AC #1、#3）：`MovieRepository.CountZhHantSubtitle` 改用 `vido_chinese_subtitle`。
- [x] T2（AC #2）：`SeriesRepository.CountZhHantCovered` 改用 `seriesChineseSubtitleSQL`。
- [x] T3（AC #5）：改寫 `home_summary_queries_test.go` 兩個測試。
- [x] T4（AC #6）：全套檢查。

## Dev Notes

- 方法名稱不改（介面與 mock 不用動），只改查詢與註解。
- 「產生字幕」按鈕後面的生成清單用的是另一個條件（`missingZhHantSubtitleWhere`，給生成流程挑候選），不在這張範圍；舊測試裡「已覆蓋＋缺少 ≤ 總數」那條檢查比的是兩套不同定義，改掉後不再成立，移除並說明。

## Dev Agent Record

**Agent:** Opus 5.5（create-story／dev）；CR：Sonnet（換模型慣例）。

### Completion Notes

- `MovieRepository.CountZhHantSubtitle`：有檔、未移除、`vido_chinese_subtitle(status, language, tracks) = 'zh_hant'`（AC #1、#3）。
- `SeriesRepository.CountZhHantCovered`：未移除、≥1 集有檔、`seriesChineseSubtitleSQL("s") = 'zh_hant'`（AC #2）。
- 方法名與介面不變（只改查詢與註解）；只有 `HomeSummaryService.coverageCell` 呼叫這兩個方法。分母不變，Covered ≤ Total 仍成立（AC #4）。
- 測試（真 migration 鏈）：電影 Vido 繁中／官方字幕檔／片內繁中軌三種算；英文、未標繁簡 chi、簡中、沒讀過、跑到一半（status 不是 found）不算；沒檔／移除不算。影集全繁中（含字幕檔）、片內繁中算；一集英文、一集簡中、零集有檔（即使影集列說繁中）、已移除不算（AC #5）。
- 舊測試「已覆蓋＋缺少 ≤ 總數」移除：「缺少」是生成流程挑候選用的另一套條件，兩者本來就不同定義。
- 檢查：`go build／vet` 綠；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅（main 一樣，雲端容器 root）；`staticcheck 2026.1` 0 項（AC #6）。

### CR（Sonnet，0H／0M／3L，全修）

- 測試裡的狀態更新改成只動指定集數；補「一集繁中一集簡中」影集、「status 不是 found 的 zh-Hant」電影。

### 交付後待辦

- 正式機更新後看首頁「繁中字幕 X/Y」：X 應該變大（片內／官方繁中的片子算進去了），「產生字幕」按鈕只在真的還有沒繁中的片子時出現。

### File List

- `apps/api/internal/repository/movie_repository.go`
- `apps/api/internal/repository/series_repository.go`
- `apps/api/internal/repository/interfaces.go`
- `apps/api/internal/repository/home_summary_queries_test.go`
- `_bmad-output/implementation-artifacts/disc-2026-10-home-coverage-counts-only-vido-zh-hant.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
