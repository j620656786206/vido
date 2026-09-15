# Story DL-IMPORT.2: 下載頁顯示「進媒體庫了沒」（前端＋設計稿）

Status: review

## Story

As a self-hoster looking at my downloads page,
I want each finished download to tell me whether it made it into the library — and take me there when it did,
so that I can stop cross-checking Sonarr, Radarr and the library by hand.

## Context

`disc-2026-09-downloads-v2-no-parse-status` 裁定拆出的第三張，也是最後一張：

1. 13-6（Sonarr／Radarr 設定頁，PR #434）
2. dl-import-1（後端 `import_status`，PR #436）
3. **本張**：畫面。

後端合約（dl-import-1 [@contract-v1]）：`GET /api/v1/downloads` 每個下載完的種子可選帶 `import_status`：

- `state`：`in_library`｜`awaiting_scan`｜`awaiting_import`｜`import_failed`｜`import_ignored`
- `source`：`radarr`｜`sonarr`；`media_type`：`movie`｜`tv`；`media_id`（Vido 有這部片／劇時）
- Sonarr 已匯入時另有 `episodes_imported`、`episodes_in_library`
- Sonarr／Radarr 不認得的種子：不帶欄位

## Acceptance Criteria

1. **卡片**（D1-D-v2／D2-D-v2／D1-M-v2）：來源標籤列（qBittorrent · NZBGet 之後）顯示入庫標籤，窄手機卡上長標籤截斷、不超出卡片；**表格**（D7-D-v2）：狀態膠囊下方顯示短版標籤（不佔名稱欄，每列檔名起點一致）。
2. **文字與顏色**（D12-D-v2 規格稿）：

   | 狀態 | 卡片 | 表格 | 顏色 |
   | --- | --- | --- | --- |
   | `in_library` | 已入庫 | 已入庫 | 青碧：有答案了 |
   | `awaiting_scan`（電影，或影集一集都還沒有） | Radarr 已匯入 · Vido 還沒有 | 已匯入 | 靛青：純告知 |
   | `awaiting_scan`（影集部分入庫） | Vido 有 6/9 集 | 6/9 集 | 靛青 |
   | `awaiting_import` | 等 Radarr 匯入 | 待匯入 | 中性：正常過渡 |
   | `import_failed` | Radarr 匯入失敗 | 匯入失敗 | 赭：你要求了但沒發生 |
   | `import_ignored` | 已略過匯入 | 已略過 | 中性：使用者自己的選擇 |

   不認識的狀態不顯示（後端之後加狀態時，舊前端不亂說）。
3. **連結**：只有文字和去處一致時才是連結——`in_library`，或 Vido 已經有其中幾集的季包——進到 `/media/movie|tv/:id`；「Vido 還沒有」永遠不是連結。點擊範圍補到 44px。可讀名稱以畫面上的字開頭（WCAG 2.5.3，語音控制說「點已入庫」點得到）。
4. **頁面開著也會更新**：SSE 進度快照沒有 `import_status`，合併時保留快取裡的值；但只要有種子剛下載完、或還在「等匯入／Vido 還沒有」，就每 60 秒重讀一次清單。進度掉回 100% 以下（例如重新檢查）時拿掉入庫標籤。
5. 設計稿：新增 D12-D-v2；D1／D2／D1-M／D7 補示範；匯出、token 檢查一致。
6. 單元測試、e2e、typecheck、lint、視覺基準全綠。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC #1, #2, #5）**：母版 `Component/DownloadCard-v2`（Mz428）的來源列加一個預設隱藏的 `importChip`，D1／D2 的做種卡（已入庫）與已完成影集卡（Vido 有 6/9 集）打開；D1-M 已完成卡（等 Radarr 匯入，手機卡的來源列是另一個 frame，直接插入；實作上窄卡會換行或截斷）；D7 四列在狀態膠囊下方加短版標籤（CR M5 後從名稱欄移過來）；新增 D12-D-v2（tvp15）規格稿，含 CR 後多出的「匯入失敗」「已略過」兩列；匯出腳本 SCREENS、CLAUDE.md 流程說明補 d12
- [x] **Task 2 — 型別與資料（AC #4）**：`downloadService.ts` 拿掉 `ParseJobStatus`／`DownloadParseStatus`／`parseStatus`，加 `ImportState`／`DownloadImportStatus`／`importStatus`；`useDownloadProgress` 合併保留 `importStatus`；兩份 hook spec 同步
- [x] **Task 3 — 元件（AC #1–#3）**：`ImportStatusChip.tsx`（`describeImportStatus` 文字／圖示／色調；卡片／表格兩種；可連結時用 TanStack `<Link>`）；接進 `DownloadCardV2`、`DownloadsTableV2`
- [x] **Task 4 — 測試**：`ImportStatusChip.spec.tsx` 15 個、卡片 spec 2 個、表格 spec 1 個、`useDownloadProgress` spec 4 個；e2e `downloads-v2.spec.ts` 新增 1 個（wire 形狀 snake_case → 標籤與連結）；圖庫夾具（做種卡、表格第三列）加上狀態
- [x] **Task 5 — 驗證**：`pnpm nx test web` 全綠、typecheck、lint；e2e `downloads-v2.spec.ts` chromium 8/8；視覺：`downloads-download-card-v2/seeding` 與 `downloads-downloads-table-v2` 重生 `-darwin`、刪掉舊 `-linux`；互動實測（圖庫）：標籤上下各 21px 內都點得到、滑過有底線；設計稿匯出 192/192、token 一致
- [x] **Task 6 — 對抗式 CR（乾淨 context agent）**：0H／5M／5L，見 Completion Notes

## Dev Notes

### 固定詞彙檢查

見 AC #2 表格。重點：「已匯入但 Vido 還沒有」不用赭——它不是故障，也可能只是 Radarr 的根資料夾不在 Vido 媒體庫路徑裡；文案只說 Vido 還沒有，這句永遠是真的。

### References

- `ux-design.pen` D12-D-v2（tvp15）、D1-D-v2（cK1KF）、D2-D-v2（tx6U1）、D1-M-v2（uMDjw）、D7-D-v2（w3ipb）、母版 Mz428
- dl-import-1 story（合約與真實資料驗證）

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context)

### Completion Notes List

對抗式 CR 結果與處理：

- **M1 可讀名稱不含畫面上的字**（語音控制說「點已入庫」點不到）→ `aria-label` 以畫面文字開頭。附測試。
- **M2 頁面開著時狀態不會更新**（等匯入一直停著，要切走視窗再回來）→ 快照合併回報「可能過期」，每 60 秒重讀清單；進度掉回 100% 以下時拿掉標籤。附測試。
- **M3「Vido 還沒有」卻是個連結**（Vido 有這部劇、但這包一集都沒有）→ 只有文字和去處一致時才連結。附測試。
- **M4 手機卡上長標籤超出卡片** → 卡片標籤可截斷。附測試。
- **M5 表格名稱欄在 1280px 下只剩三個字、各列檔名起點不齊** → 標籤移到狀態膠囊下方；設計稿 D7 同步，名稱改回完整長度。
- **L 已處理**：表格版點擊範圍 44px、連結不再同時有 `title`（避免唸兩次）、`mediaType` 不是 movie／tv 時不連結、hover 改底線（不降對比）、匯出腳本註解錯位。
- **範圍外（立案或記錄）**：
  - 「等匯入」可能其實是 Sonarr／Radarr 佇列裡卡住（要手動匯入）——要讀佇列 API 才分得出來 → `disc-2026-09-import-status-queue-blocked`。
  - 後端還在送沒人用的 `parse_status` → 併入 `disc-2026-09-parse-pipeline-never-wired`。
  - 「已入庫」以 TMDb id 判斷，升級後舊檔被刪時可能誤報、集數查詢失敗時算成 0 → dl-import-1 已記的限制。

### File List

- `apps/web/src/components/downloads/ImportStatusChip.tsx`（新）＋`.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadCardV2.tsx`＋spec、`DownloadsTableV2.tsx`＋spec
- `apps/web/src/services/downloadService.ts`
- `apps/web/src/hooks/useDownloadProgress.ts`＋spec、`useDownloadActions.spec.ts`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/downloads-v2.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/downloads-download-card-v2/seeding/`、`downloads-downloads-table-v2/`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-d-downloads-v2/`（d1-d、d1-m、d2-d、d7-d、d12-d）
- `scripts/export-pen-screenshots.py`、`CLAUDE.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-15 | 建立並實作；對抗式 CR 0H／5M 全修（M5 連設計稿一起改）。狀態 → review。 |
