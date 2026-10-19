# Disc（傘狀）：沒有字幕來源、也沒設語音辨識時，生成畫面寫灰色「已略過」，不再寫紅色「失敗 · 生成失敗」

Status: backlog（傘狀單，已拆 `-a` 後端／`-b` 前端）

**Source:** 2026-10-19 party mode（founder-pm／Winston／Sally／John／Bob／Mary＋負責 #848 片旁字幕的另一個 session）。單次按「生成字幕」時，片內沒有可用字幕、又沒設語音辨識，這次生成被記成 `skipped`。結果三個地方講三種話：
- 生成畫面的即時活動：紅色「失敗 · 生成失敗」（`failure_reason` 落到 `other`）；
- 活動頁：完全不列（#831 只列做完／沒做成）；
- 管理字幕視窗結果區：「這部影片沒有可用的字幕來源，也沒有設定語音辨識」（`apps/api/internal/subtitle/solo_runner.go:507-508`）。

⚖️ **Alexyu 2026-10-19：由這邊直接做**（不等 #848）。

## 為什麼現在做（範圍）

- 今天詳細頁大多會先擋（`generateCostView.ts:124-125` 沒設語音辨識時按鈕停用；後端 `solo_runner.go:211-212` 也會先回 409）。實際會碰到的是：估價說可以、執行時才發現沒來源（例如只有 SDH 軌、語言標 `und` 的軌）；批次生成；跑到一半語音辨識金鑰被刪。
- #848 的 `-b` 上線後（片旁英文字幕可以當來源，但時間軸對不上就不用）會更常走到「沒來源又沒聽寫」。`-b1`／`-b2`／`-b3` **沿用本單定的代碼 `no_text_source` 與文字**，不另外定義。

## 拆單（⚖️ SM Bob，Story Splitting Rule）

跨前後端，兩邊都超過 3 項 → 拆。

| 單 | 內容 | 項數 |
|---|---|---|
| `disc-2026-10-no-text-source-skip-a`（後端） | 新代碼 `no_text_source`＋寫紀錄時加前綴＋單次／批次即時事件都帶＋合約升版 | 5 |
| `disc-2026-10-no-text-source-skip-b`（前端） | 即時活動、生成佇列、產生字幕對話框畫成灰色「已略過」＋第二行原因 | 5 |

- **合併順序：`-a` 先，`-b` 後。** `-a` 單獨上線無害：舊前端遇到不認得的代碼顯示「生成失敗」（跟今天一樣）；批次 `skipped` 項目的代碼舊前端本來就丟掉（`useGenerationJobsFeed.ts:524` 只留 `error` 的代碼）。
- **`-b` 另外擋在 `disc-2026-10-generation-queue-failure-reason` 合併之後**（那張正在另一個 dev agent 手上，會新增 `runFailureCopy.ts`、讓佇列讀代碼、把即時活動改成兩行）。
- **不碰** `apps/api/internal/services/transcription_service.go`（`disc-2026-10-generate-keeps-user-en-srt-a` 正在改）。查過：本單不需要改它——「沒設語音辨識」的判斷由它丟出 `ErrTranscriptionDisabled`，本單只在 `process_item.go` 接住這個錯的地方寫紀錄。
- **不碰** `PredictFromTracks`、`SelectAndRoute`、片旁字幕判定、管理字幕對話框按鈕（#848 `-b1`／`-b2`／`-b3`）。

## 需要決定的事（彙整；細節在兩張子單）

| # | 誰 | 問題 | 預設（不擋開工） |
|---|---|---|---|
| 1 | Sally 追認 | 設計稿沒有「已略過」這種列／徽章。灰色用哪個樣式？ | 沿用既有中性色：佇列徽章同「已取消」（`bg-[var(--bg-tertiary)]`＋`--text-muted`、無圖示）；即時活動同「⑧ 批次已取消」的色調（圖示 `--text-muted`、階段字 `--text-secondary`），圖示用既有的 `circle-dashed`。要補稿另開 design 單。 |
| 2 | Sally 追認 | dsr-6d-b（dsr-6d-a CR L8）曾裁定**佇列不准用「已略過」開頭**（怕使用者以為是系統自己決定不做）。party mode 改成要用。 | 照 party mode：徽章寫「已略過」，第二行寫「沒有可用的字幕來源，也沒有設定語音辨識」，原因講清楚就不會誤會。舊註解與三支釘住「不准已略過」的測試改寫。 |
| 3 | Sally／Alexyu | 批次結尾那句「完成 N 部、失敗 M 部」仍把略過的算在「失敗」裡（後端 `fail_count`，sub-4-2 CR H1 的誠實規則；E2E 也釘住）。列寫「已略過」、總結寫「失敗」，會不會對不起來？ | **不改**（不在本單）。若要改成「完成 N 部、略過 K 部、失敗 M 部」是合約變更，另開單。 |
| 4 | Alexyu（查到的事，不在範圍） | 批次生成遇到「片內有字幕軌、但語言標成 `und`」的片，**就算有設語音辨識也不會聽寫**，直接略過（`pipeline.go:189-196`：只有詳細頁單次生成會把這種片送聽寫）。這種列在 `-b` 之後會顯示灰色「已略過 · 沒有可用的字幕來源」（不帶「也沒設語音辨識」，因為不是）。 | 建議另開 backlog（暫名 `backlog-batch-und-track-never-transcribed`），不在本單。 |
| 5 | Sally | 產生字幕對話框（F8）的列只有一個標籤位置，原因 19 個字比目前最長的還長，手機片名可能被擠掉。 | `-b` AC 量測把關：手機片名只剩「…」或 1～2 字就停下來回報，不合併。 |

## 不做（兩張共同）

- **活動頁不列 `skipped`**：#831 的設計只列做完／沒做成（`apps/api/internal/repository/subtitle_run_repository.go:484` `status IN (completed, failed)`）。要列需要 Sally 裁定，另開單。因此 `disc-2026-10-activity-subtitle-run-time-a` AC #1（活動頁的 `failure_reason`，`[@contract-v2]`）**不升版**——活動頁送出的值不會多。
- 管理字幕視窗結果區的文字（已經是對的那句）、詳細頁按鈕停用規則、#848 的路由／估價／片旁字幕。
- 批次計數與「失敗 M 部」（見上表 #3）。
- 舊紀錄回填：本單之前寫的 skipped 紀錄沒有前綴，分類仍是 `other`；這些紀錄只存在資料庫、畫面不列，不做遷移。
