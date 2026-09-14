# Story DSR.4: Flow D 下載中心——程式碼追回 D1–D7 桌機設計稿

Status: done

## Story

As a self-hoster watching qBittorrent from Vido,
I want 每張下載卡片只給我一顆「現在該按的」按鈕、其他動作收進 ⋯、只有真的會刪檔的那一下才問我,
so that 我不用每次移除都多按一次確認，也不會把「已暫停」看成出了問題。

## Context

`epic-dsr` 的第 7 張。範圍：`components/downloads/**`、`hooks/useDownloads.ts`、圖庫夾具、`tests/e2e/downloads-v2.spec.ts`，對應七張桌機稿 D1–D7（`cK1KF` `tx6U1` `lCFq2` `T95wy` `dVPuY` `UNVRU` `w3ipb`）。手機 D1-M 的卡片是同一個母版 `Component/DownloadCard-v2`（`Mz428`），跟著一起對齊。

**範圍收斂：** 手機三張抽屜稿（D8-M 篩選、D9-M 排序、D10-M 卡片動作）程式碼完全沒做，工作量等於另一個功能，拆成 **dsr-4b**。

立案時記的三個缺口，查證後：

1. **「只有 11/21 檔有 Design ref 標頭」**——現存檔案全部都有。缺標頭的是 6 個 v1 元件（`DownloadList`、`DownloadItem`、`DownloadDetails`、`DownloadFilterTabs`、`DownloadParseStatusBadge`、`ParseFailedActions`）：app 裡沒有任何地方用，只剩圖庫夾具在畫；其中兩個的標頭指向 G1 稿 `rWvuG`，那張稿 2026-09-10 已刪。→ 連同 spec、夾具、視覺基準一起刪。
2. **「2 檔舊字級任意值」**——卡片與表格的 `text-[11px]`，隨重寫消失。
3. **「批次選取／卡片動作／fail-soft 要逐一驗」**——卡片動作是整張單子最大的落差（下述）。

**最大的落差在動作，不在長相：**

> 設計稿 v2.1（已批准）：卡片右下一顆**依狀態變化的主按鈕**（下載中→暫停、已暫停→繼續、錯誤→重試、已完成→無）＋一顆 **⋯ 選單**（同一個狀態動作、移除（保留檔案）、分隔線、移除（連同檔案刪除））。**只有「連同檔案刪除」會跳確認**，確認框上寫著檔名與大小。
>
> 程式碼：暫停＋垃圾桶兩顆按鈕，兩種移除都先跳同一個對話框——「保留檔案」這種隨時可以重新加回的動作也要多按一次；真正不可逆的刪檔，確認框上反而沒有檔名。錯誤狀態沒有重試。

其他查到的：

- **已暫停用赭（warning）。** 赭的意思是「你要求了，但它沒發生」；暫停正是使用者要求的。稿 D1 本來就畫中性 → 改中性。
- **下載速度用青碧（success-text）。** 青碧＝已經有答案了，一個正在跳的數字不是完成 → 改中性（關 `disc-2026-08-download-speed-wears-completed-green`）。
- **同一個狀態兩個名字**：篩選 chip 叫「做種中」，狀態膠囊叫「做種」→ 統一「做種」。
- **前往設定**連到 `/settings`（設定首頁）→ 改 `/settings/qbittorrent`，那才是唯一要檢查的地方。
- **表格的速度／ETA／大小欄，稿上有排序箭頭**，但 API 只能依名稱／狀態／進度排序 → 刪掉稿上三個假箭頭。
- **稿 D2／D7 的批次列少了「批次繼續」**（程式碼一直都有）→ 補稿。
- **不適用的數字印成 0**：已暫停的卡片仍印出最後一次的速度，ETA 印「∞」→ 不適用一律「—」。

## Acceptance Criteria

### 設計稿 vs 程式碼對照表

| 畫面 | 元素 | `.pen`（改前） | 程式碼（改前） | 判定 |
| --- | --- | --- | --- | --- |
| D1／D3 | 卡片動作 | 狀態鈕＋⋯ 選單 | 暫停＋垃圾桶 | ❌ 碼錯 |
| D3 | 確認時機 | 只有連同檔案刪除 | 兩種移除都確認 | ❌ 碼錯 |
| D3 | 確認框內容 | 標題「移除並刪除檔案？」、說明、檔名·大小、取消／刪除檔案 | 「移除（保留檔案）／移除（連同檔案刪除）」二選一、無檔名 | ❌ 碼錯 |
| D3 | 確認框圖示底 | `$error-tint`＋`$error` 圖示 | —（無圖示） | ❌ **兩邊都錯**：DESIGN.md §赭 2026-09-11「事前警語不給語意色」→ `$bg-tertiary`＋`$text-primary`（CR 抓到） |
| D1 | 錯誤卡片 | 重試（rotate-cw） | 無 | ❌ 碼錯 |
| D1 | 已暫停膠囊 | 中性 | 赭 | ❌ 碼錯 |
| D1 | 膠囊 | 圓點＋12px | 無圓點＋11px | ❌ 碼錯 |
| D1 | 速度文字 | 中性 | 青碧（↓）／泥金（↑） | ❌ 碼錯 |
| D1 | 不適用的數字 | 「—」 | 隱藏或 0／∞ | ❌ 碼錯 |
| D1 | 大小 | 已下載 / 總量 | 只有總量 | ❌ 碼錯 |
| D1 | 來源 | qBittorrent chip＋停用的 NZBGet 預留位 | 只有 qBittorrent | ❌ 碼錯（ux3-4-1 決策 #4） |
| D1 | 做種／已完成卡片的字色 | 膠囊與百分比沿用母版的泥金 | — | ❌ 稿錯（instance 覆寫漏了）→ `$info-text`／`$success-text` |
| D1 | 暫停中的進度條 | `$text-disabled` | 泥金 | ❌ **兩邊都錯**：disabled 色只給停用控制項 → `$text-muted`（CR 抓到） |
| D1 | 標題副標 | 「管理所有下載任務」 | 「即時監控 qBittorrent 下載狀態」 | ❌ 碼錯 |
| D1 | 篩選 chip | 44px、選中 `$accent-subtle` | 32px、選中 accent-tint＋框線 | ❌ 碼錯 |
| D1 | 工具列 | 左「選取」、右排序下拉＋圖示切換 | 右：排序欄位＋方向鈕＋文字切換＋選取 | ❌ 碼錯 |
| D1 | 頁尾 | 「1–12 / 128」＋頁碼 | 只有多頁時才出現 | ❌ 碼錯 |
| D1 | specNote | 稿上一段說明文字 | — | ❌ 稿錯（說明應是獨立畫面）→ 刪 |
| D2 | 選取模式 | 副標「批次選取模式」＋右上取消；批次列取代工具列 | 批次列疊在工具列下、取消在批次列尾 | ❌ 碼錯 |
| D2／D7 | 批次繼續 | 無 | 有 | ❌ 稿錯 → 補 |
| D2／D7 | 批次列 | `$accent-tint` 底＋泥金框、44px 按鈕含圖示 | 灰卡片、小按鈕無圖示 | ❌ 碼錯 |
| D3 | 兩段提示文字 | 有 | — | ❌ 稿錯 → 刪 |
| D5 | 空狀態 | 72px 圖示方塊、20px 標題、主按鈕 | 卡片框、小圖示、outline 按鈕 | ❌ 碼錯 |
| D6 | fail-soft | 硃砂框、WifiOff、44px 按鈕 | 無框、PlugZap | ❌ 碼錯 |
| D6 | 前往設定 | — | `/settings` | ❌ 碼錯 → `/settings/qbittorrent` |
| D6／D2／D3／D7 | 圖示色 | `$error` | — | ❌ 稿錯（圖示讀文字階）→ `$error-text`（5 個） |
| D7 | 欄位 | ☑ 名稱 狀態 進度 速度 ETA 大小 動作 | ☑ 名稱 狀態 大小 進度 速度 ETA 操作 | ❌ 碼錯 |
| D7 | 速度／ETA／大小的排序箭頭 | 有 | 不可排序 | ❌ 稿錯（API 做不到）→ 刪 |
| D7 | 工具列左側 | 「共 128 筆任務」 | 無 | ❌ 碼錯 |
| D8–D10 | 抽屜把手 | `$text-disabled` | （未實作） | ❌ 稿錯 → `$text-muted`（與 dsr-5 E2-M 一致） |

1. **狀態鈕**：下載中／做種／停滯／佇列／檢查中→暫停；已暫停→繼續；錯誤→重試（呼叫 resume）；已完成→無。
2. **⋯ 選單**：狀態動作、移除（保留檔案）立即執行不確認、分隔線、移除（連同檔案刪除）開確認框；確認框顯示檔名與大小（大小未知時不印「0 B」）、取消／Esc 不刪、關閉後焦點回到 ⋯。
3. **顏色**：已暫停中性；速度中性；進度條只有下載中是泥金、做種靛青、完成青碧、錯誤硃砂、其餘中性。
4. **數字**：不適用的值是「—」；大小是「已下載 / 總量」，完成後只剩總量；總量未知（磁力連結還在抓 metadata）是「—」。
5. **選取**：清單的選取模式用批次列取代工具列、右上取消；表格的勾選框常駐；切換篩選、頁碼、每頁筆數、排序、檢視時清空選取；自己離開畫面的列（下載完被篩掉、被別處移除）不算在已選數量與批次動作裡。
6. **工具列**：排序一個下拉同時決定欄位與方向，表格欄頭能產生的每種組合都有對應選項；檢視切換是兩顆圖示鈕（`aria-label` 清單檢視／表格檢視）；qBittorrent 連不上時工具列與批次列隱藏。
7. **設計稿**同步修正（上表的「稿錯」各列），匯出 188/188、token 檢查一致。
8. 單元測試、E2E、typecheck、lint、視覺基準（新夾具 7 張 `-darwin`）全綠。

## Tasks / Subtasks

- [x] **Task 1 — 卡片動作（AC #1, #2）**
  - [x] `DownloadRowActions.tsx` 重寫：狀態鈕、Radix DropdownMenu（`modal={false}`，避免選單關閉那一格把 body 的 pointer-events 留給確認框）、受控確認框、`onCloseAutoFocus` 把焦點還給 ⋯
  - [x] `DownloadRowActions.spec.tsx`（新）：七種狀態的狀態鈕、完成無狀態鈕、選單重複狀態動作、保留檔案不確認、刪檔要確認、取消／Esc 不刪、焦點回 ⋯、大小未知不印 0 B

- [x] **Task 2 — 卡片、表格、狀態畫面（AC #3, #4）**
  - [x] `downloadStatus.ts`：已暫停中性；新增 `getDownloadTone`（進度條與百分比共用）
  - [x] `formatters.ts`：新增 `formatDownloadMeta`（卡片與表格共用）；刪除沒人用的 `formatDate`
  - [x] `DownloadCardV2.tsx` 照 `Mz428` 重寫；匯出 `DownloadStatusPill` 給表格共用
  - [x] `DownloadsTableV2.tsx`：欄位順序、`table-fixed`＋`min-w-[960px]`（長片名截斷、窄桌機橫向捲動而不是把狀態膠囊擠成直排）
  - [x] `DownloadsStatesV2.tsx`：D4 骨架拿掉重複的 chip 列（頁面的真 chip 一直都在）、D5、D6、前往設定落點

- [x] **Task 3 — 頁面（AC #5, #6）**
  - [x] `DownloadsBrowseV2.tsx`：標題列、chip、工具列、合併排序下拉、圖示切換、批次列、頁尾摘要、選取清空規則、`selectedHashes` 只算畫面上的列
  - [x] `DownloadsBrowseV2.spec.tsx`（新，11 個）

- [x] **Task 4 — 刪除 v1（缺口 ①）**
  - [x] 6 個元件＋6 份 spec＋6 個圖庫夾具＋6 個視覺基準資料夾
  - [x] `useDownloadDetails`（只剩 `DownloadDetails` 在用）與 `downloadKeys.detail`；spec 與 e2e 註解同步
  - [x] `tests/e2e/parse-trigger.spec.ts` 兩行指向已刪 spec 的註解改指 `disc-2026-09-downloads-v2-no-parse-status`

- [x] **Task 5 — 設計稿（AC #7）**
  - [x] 上表「稿錯」各列；存檔（AppleScript）→ 匯出 188/188，只有 flow-d 9 張（d1-d d1-m d2-d d3-d d6-d d7-d d8-m d9-m d10-m）與 `pen-tokens.json` 有差 → `check-design-tokens.py` 一致（73 母版）

- [x] **Task 6 — 驗證（AC #8）**
  - [x] `pnpm nx test web` 257 個測試檔全綠、typecheck、lint 0 errors
  - [x] E2E `downloads-v2.spec.ts` chromium 7/7（改了：移除流程走 ⋯ → 選單 → 刪除檔案；表格切換鈕叫「表格檢視」；「舊標題不見了」改用 heading 查——新標題「下載」＋副標「管理所有下載任務」的文字接起來剛好讀成「下載管理…」）
  - [x] 視覺：新夾具 7 張 `-darwin`（卡片 ×4、表格、空狀態、fail-soft），`-linux` 交給 CI bootstrap
  - [x] 互動實測（`/test/gallery?fixture=downloads-download-card-v2/downloading`）：⋯ hover 底色有變、選單開啟、確認框預設焦點在「取消」、Esc 後焦點回 ⋯

- [x] **Task 7 — 對抗式 CR（乾淨 context agent）**：2 HIGH／3 MED／數個 LOW，全部處理或立案（見 Completion Notes）

## Dev Notes

### 不要做的事

- **不做手機抽屜**（D8–D10）→ dsr-4b。
- **不改 `components/ui/Pagination.tsx`**。稿上的頁碼是 44px 有框方塊，現況 40px 無框；這個元件也用在探索與搜尋結果，改它會動三個流程的基準 → `disc-2026-09-pagination-44px-drift`。
- **不為了「全部都要有稿」補畫批次移除的確認框。** 批次移除仍是二選一對話框（保留／連同刪除），因為一次作用在很多列上；單列的規則（只有刪檔才確認）不直接套用。

### 固定詞彙檢查

| 顏色 | 用在哪 | 為什麼對 |
| --- | --- | --- |
| 泥金 | 下載中的膠囊、進度條、百分比；選中的 chip／卡片框 | 正在跑 |
| 青碧 | 已完成的膠囊、進度條 | 有答案了 |
| 靛青 | 做種、檢查中 | 純告知（完成了，還在分享） |
| 硃砂 | 錯誤膠囊、進度條、重試圖示、fail-soft 卡；刪檔選單項與刪除檔案按鈕 | 壞了／不可逆 |
| 中性 | 已暫停、停滯、佇列；所有速度數字；確認框的警告圖示 | 使用者要求的狀態不是警訊；事前警語不給語意色 |
| 赭 | （本頁不用） | 下載頁沒有「你要求了但沒發生」的狀態可以說 |

### References

- `ux-design.pen` D1-D-v2 `cK1KF`、D2 `tx6U1`、D3 `lCFq2`、D4 `T95wy`、D5 `dVPuY`、D6 `UNVRU`、D7 `w3ipb`、母版 `Mz428`
- DESIGN.md §赭（2026-09-11 事前警語裁定）、§Shadow Vocabulary（選單 lg、對話框 xl）
- `ux3-4-3-downloads-frontend.md`、`ux3-4-4`（表格檢視）、ux3-4-1 決策 #4（NZBGet 預留位）、#5（沿用 N1 lifecycle 色票）

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context)

### Debug Log References

- Radix DropdownMenu 在 jsdom 用 `userEvent` 直接可以開，不需要補 PointerEvent／ResizeObserver stub。
- 表格第一版把名稱欄設 `w-full max-w-0`，自動版面把狀態膠囊擠成「下／載／中」直排（視覺基準抓到）→ 改 `table-fixed`＋最小寬度。
- 本機視覺全套跑時 `retry-retry-notifications`、`glossary-panel-v2/seeded`、`parse-floating-parse-progress-card` 三張不符：差異在側欄計數與儲存空間列（本機 dev 狀態），與本單無關，未更新。

### Completion Notes List

對抗式 CR 結果與處理：

- **HIGH 已選取的列離開畫面仍會被批次刪除**（下載中篩選下完成的種子、別處移除、排序換頁）→ `selectedHashes` 只取畫面上的列；排序也清空；卡片只有在選取模式才顯示選中框。附測試。
- **HIGH 確認框關閉後焦點掉到 body**（受控 Dialog 沒有 trigger 可以還焦點）→ `onCloseAutoFocus` 還給 ⋯。附測試＋實測。
- **MED 確認框的警告圖示用硃砂底** → 違反 DESIGN.md §赭 2026-09-11 → 碼與稿都改中性。
- **MED e2e 註解引用不存在的單號** → 已立 `disc-2026-09-downloads-v2-no-parse-status`。
- **LOW 暫停進度條用 `--text-disabled`** → `--text-muted`（碼＋稿，含三張抽屜把手）。
- **LOW 大小 0 印成「0 B / 0 B」**（磁力連結抓 metadata 中）→「—」。附測試。
- **LOW 雜項**：每頁筆數下拉 36→44px；批次對話框移除 `aria-describedby={undefined}`（讓說明文字接回對話框）；骨架重複 chip 列；點已選中的檢視鈕不再清空選取；沒有列時「選取」停用；qBittorrent 錯誤時不再留著「批次選取模式」標題；長片名截斷。
- **測試缺口**：補 checking 狀態、Esc、focus、stalled／queued／checking／error 的數字、progress > 1、tone 的 checking／stalled、完成卡片的大小字樣。

### Discovery Triage

| 發現 | 處置 |
| --- | --- |
| 未設定 qBittorrent 時 fail-soft 也說「無法連線」 | 立案 `disc-2026-09-downloads-not-configured-says-unreachable`（文案，需稿或裁定） |
| `ui/Pagination` 40px 無框 vs 稿 44px 有框 | 立案 `disc-2026-09-pagination-44px-drift`（共用元件） |
| v2 下載頁沒有解析狀態 | 立案 `disc-2026-09-downloads-v2-no-parse-status`（產品裁定） |
| 手機 D8–D10 抽屜未實作 | 立案 `dsr-4b-flow-d-mobile-sheets` |
| qBittorrent 的 `missingFiles` 也映射成錯誤，按重試（resume）修不好 | 記錄；錯誤卡片的「重試」對多數錯誤成立，缺檔要重新校驗，屬 qBittorrent 行為，暫不另立 |
| 排序不會回到第 1 頁；表格欄頭第一次點是遞減（名稱先 Z–A） | 既有行為，記錄 |
| `formatProgress` 99.96% 顯示 100.0% | 既有行為，記錄 |
| `testsprite_tests/testsprite_frontend_test_plan.json` 仍斷言 v1 的 testid | 本來就打不到（v1 已不渲染），留給下次 TestSprite 重生計畫 |
| `downloadService.getDownloadDetails` 與 `DownloadDetails` 型別只剩 service 自己 | 保留（API client 與後端端點對應），hook 已刪 |
| `StatusIcon.tsx` 速度同樣用青碧 | 只剩儀表板 `DownloadPanel` 使用，不在 Flow D |

### File List

- `apps/web/src/components/downloads/DownloadRowActions.tsx`（重寫）＋`DownloadRowActions.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadCardV2.tsx`（重寫）＋spec
- `apps/web/src/components/downloads/DownloadsTableV2.tsx`＋spec
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx`＋`DownloadsBrowseV2.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadsStatesV2.tsx`＋spec
- `apps/web/src/components/downloads/downloadStatus.ts`＋spec
- `apps/web/src/components/downloads/formatters.ts`＋spec
- 刪：`DownloadList`、`DownloadItem`、`DownloadDetails`、`DownloadFilterTabs`、`DownloadParseStatusBadge`、`ParseFailedActions`（各含 spec）
- `apps/web/src/hooks/useDownloads.ts`＋spec（刪 `useDownloadDetails`）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（刪 6 夾具、增 7 夾具）
- `tests/visual/components.visual.spec.ts-snapshots/components/downloads-*`（刪 6 資料夾、增 7 張 `-darwin`）
- `tests/e2e/downloads-v2.spec.ts`、`tests/e2e/downloads.spec.ts`（註解）、`tests/e2e/parse-trigger.spec.ts`（註解）
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-d-downloads-v2/`（9 張）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-14 | 建立並實作（dev＋乾淨 context agent 對抗式 CR：2 HIGH／3 MED 全處理，LOW 處理或立案）。範圍收斂成桌機，手機抽屜拆 dsr-4b。狀態 → review。 |
| 2026-09-14 | PR #429 合併進 main（8cde34dc）。CI 17 項全綠；視覺基準 bootstrap PR #430（7 張 `-linux`，抽 4 張看過）。CI 第一輪 E2E 第 1 片紅燈：`tests/e2e/downloads.spec.ts` 還在找舊的空狀態文案，改測試後通過。狀態 → done。 |
