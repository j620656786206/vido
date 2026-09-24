# Story DSR.3f：備份與還原、匯出對齊設計稿——手機上看得到備份的操作鈕、還原前的確認框從底部滑上來、建立失敗說人話

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who backs up and restores Vido's data,
I want 手機上每一份備份是一張卡、按得到還原與刪除；還原前的確認框在手機上從底部滑上來、告訴我會用哪一份（多大、哪一天）而且會先幫我留快照；建立失敗時講中文、有重試,
so that 我在手機上也能安心做「還原」這種會蓋掉資料的事。

## Context

`dsr-3` 拆出來的第六張：**「維護」分組的備份與還原、匯出／匯入**。**依賴 `dsr-3a` 先合併**。與其他子單互不相依。拆單理由見 `dsr-3a` Context。

| 稿 | 節點 | 元件 |
| --- | --- | --- |
| C5-D／C5-M 備份與還原 | `uhAKd`／`gEQX4` | `routes/settings/backup.tsx`、`BackupManagement.tsx`、`BackupTable.tsx`、`BackupScheduleConfig.tsx` |
| C19-D／C19-M 還原確認 | `G8BYO`／`gPZU6` | `RestoreConfirmDialog.tsx` |
| C20-D 建立備份失敗 | `v2C4xr` | `BackupManagement.tsx:127-158` |
| C13-D／C13-M 匯出／匯入 | `nwn6a`／`Ytjrj` | `routes/settings/export.tsx`、`MetadataExport.tsx` |

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

**備份列表（C5）**
1. **手機是真的壞掉**：碼在 390 寬沿用桌機的固定寬表格（約 610px＋操作欄），外層 `overflow-hidden` → **操作欄與狀態被切掉，手機上按不到還原與刪除**。稿 C5-M 把每筆畫成卡片（`nzgTk`）→ 稿→碼（`<640` 卡片清單，≥640 表格）。
2. 摘要列：稿「7 份備份 · 已使用 346.5 MB」＋`hard-drive` 圖示（`mbZrH`／C20 `ZiEhh`）；碼「已使用 X（N 個備份）」無圖示（`BackupManagement.tsx:131`）→ 稿→碼。
3. 狀態 pill「完成」：稿 `oktnb` 青碧＋勾、Label 12／600；碼刻意中性 `bg-tertiary`（`BackupTable.tsx:10-14` 註解寫明依狀態詞彙「完成＝中性」）、`text-[11px]`、前面小圓點（`:91-95`）。⚖️ **顏色碼→稿（中性）**——備份完成是「平常狀態」，列表裡每一列都是綠色等於沒有訊號；**字級與圖示稿→碼**（`text-xs font-semibold`＋勾）。⚠️ `text-[11px]` 是全 settings 三個舊字級之一。
4. 執行中那列的操作鈕：稿 4 顆（`kOkTf`／`AKosK`）；碼只有完成的備份才有還原／驗證／下載，執行中只剩刪除（`:100`）→ **碼→稿**（對還沒做完的備份按還原是錯的）。
5. 操作鈕外觀（稿→碼）：稿 32×32 `$bg-tertiary` `$radius-md`、圖示 14、垃圾桶 `$error-text`；碼無底 16px、垃圾桶平時 `text-secondary`（`:102-138`）。手機卡片上的鈕 44。
6. 表格文字（稿→碼）：稿檔名／大小／時間都 Mono 12，時間「2026-09-11 03:00」；碼非等寬、`toLocaleString('zh-TW')`（`:30,77-88`）。新 formatter 本地時區 `YYYY-MM-DD HH:mm`；`<time dateTime>`。⚠️ `dsr-3e` 也要做一個本地時區 formatter（秒級）——**後做的那張重用先做的**（同一個函式帶精度參數），Completion Notes 註明。
7. 表頭（稿→碼）：稿 `Na3S4` `$bg-tertiary`、`$text-muted`、欄寬 440／120／200／120／176、列高 56；碼 `bg-secondary/80`、`text-secondary`、任意寬 `w-[320px]/w-[80px]/w-[140px]/w-[70px]`、`py-2.5`＋底線（`:58-63,74`）。欄寬改成**一個常數**的 `grid-template-columns`（比例照稿：`minmax(0,440fr) 120px 200px 120px 176px` 之類，dev 取捨後寫註解），表頭與列共用。

**自動備份排程（C5 `Jt3DP`）**
8. 碼→稿：標題下說明「系統會在指定時間自動執行備份」碼只在開啟時顯示、有下次時間時換成「下次備份：…」、還多一個 Clock 圖示 → 稿照碼畫；選「每週」時碼多一個「備份日」選單 → 稿補；碼有「儲存排程」鈕、稿沒有 → ⚖️ **碼→稿**（改成自動儲存是行為變更，不在對齊範圍）。
9. 頻率：稿「每日／每週」分段按鈕（`qxfV5`），碼 `<select>`、標籤「頻率」vs 稿「備份頻率」→ 稿→碼（分段按鈕：`role="radiogroup"`、兩顆 `role="radio"`、方向鍵切換；標籤用稿的「備份頻率」）。
10. **「執行時間 03:00」產品做不到分鐘**：後端 `BackupSchedule` 只有整數小時 0–23，碼是整點下拉 → ⚖️ **碼→稿**（稿改畫整點選單）。
11. **「保留份數 7」產品做不到**：後端沒有這個欄位，保留規則寫死；碼顯示唯讀「保留策略：最近 7 個每日備份＋最近 4 個每週備份」→ ⚖️ **碼→稿**（稿改成這段唯讀文字，dev 從碼逐字抄）。
12. 開關圓點：碼 `text-on-scrim`，稿 `text-on-accent` → 稿→碼。
13. 手機稿沒畫排程卡（`gEQX4` page-body 只有 head 與 list）→ **碼→稿**：稿補畫。

**還原確認（C19）**
14. **沒用共用對話框**：`RestoreConfirmDialog.tsx:23-26` 是自己寫的 `fixed` div——**沒有 `role="dialog"`、沒有焦點鎖定、Esc 關不掉**。手機稿 `DvvOs` 是底部抽屜（390×440、`radius-xl`、40×4 拉把 `E992A`），按鈕上下疊、滿版 48 高、「確認還原」在上（`n5XqGx`）。→ ⚖️ **改用 `ui/Dialog`＋`MOBILE_SHEET_CONTENT`＋`SheetGrabber`**，照 `components/downloads/DeleteWithFilesDialog.tsx` 的寫法（Flow D 已驗證過的手機破壞性確認）。
15. 內容（稿→碼）：圖示 `rotate-ccw`（碼 `AlertTriangle`，`:29`）；標題 H3 20（手機 18；碼 `text-lg`）；備份檔資訊框 `NOEJm` `$bg-tertiary` `$radius-md` 兩行等寬——檔名 `U0exxg`（`$text-primary`）＋「51.8 MB · 2026-09-10 03:00」`HjE3s`（`$text-muted`）（碼 `:39` `bg-primary`、只有檔名、非等寬；`Backup` 資料本來就有 `sizeBytes` 與 `createdAt`）；快照說明 `JDzHR` 靛青提示框 `$info-tint`＋`info`＋`$info-text`（碼 `:42` 一行 `text-muted`）；取消 `IvUhq` `$bg-tertiary` 底＋`$text-primary` 44 高（碼 `:51` 外框樣式）；「確認還原」`lzMdn` `$error` 碼已一致。
16. 說明句的強調：碼把「這將會取代目前所有的資料」包 `<strong>`（`:37`），稿整句 `$text-secondary` 無強調。⚖️ **碼→稿（保留強調）**：這是整個框最重要的一句，破壞性確認裡強調後果是對的；稿補強調。
17. 手機快照說明較短（`PAmv4`「還原前會自動建立目前資料的快照。」）→ ⚖️ **碼→稿**（一套文案；dev 先確認碼那句與稿桌機句是否逐字相同，不同就以碼為準、稿兩張都改）。
18. 稿的陰影 `p95rQR` 是寫死的 `#00000066`／blur 32 → 稿改用陰影 token（若 `.pen` 有 shadow 變數；沒有就保留並在 Completion Notes 記一筆）。

**建立失敗（C20）**
19. 碼（`BackupManagement.tsx:150-158`）只有一行純文字、位置在建立按鈕列**下面**；內容是後端英文 `BACKUP_CREATE_FAILED`／"Failed to create backup"（`backupService.ts:98` 原樣顯示）。稿 `Bdimi`：`$error-tint` `$radius-md`、18 `circle-alert`、標題 `qiiIx`「建立備份失敗」Body 600＋說明 `y7rQ6J` Label＋右側「重試」`syomc`（`$bg-tertiary` 36 高），**位置在摘要列上面** → 稿→碼。
20. **稿的說明「磁碟空間不足：需要 52 MB，可用 18 MB。清出空間後再試一次。」產品做不到**：`backup_service.go` 不預檢磁碟、也不分類錯誤。⚖️ **碼→稿＋另立單**：說明句用前端通用句「備份沒有建立成功。請稍後再試；若持續失敗，到「系統日誌」查看原因。」（「系統日誌」是 `<Link to="/settings/logs">`；SM 擬，待 Sally），稿同步改字；立 `disc-2026-09-backup-disk-space-precheck`。後端英文**不顯示**。
21. 稿 C20 在錯誤條下面沒畫備份表與排程 → 只是省略；碼**照常顯示**（`spec-note-dsr-3f` 提一句）。
22. 整頁載入失敗（`BackupManagement.tsx:105-120`）→ `SettingsErrorState`：「無法載入備份資訊」／「與後端的連線中斷了。已存在的備份檔不受影響。」／`refetch`（SM 擬，待 Sally）。

**匯出（C13）**
23. 標題、說明、三種格式說明、匯入尚未實作的提示逐字相符（桌機）。
24. 寬度：稿匯出卡與匯入提示都 768（`sF26K`、`JfvjU`），碼整欄 1152（`export.tsx:20`）→ `max-w-3xl`。
25. 匯出卡（稿→碼）：內距碼 `p-4`→`p-6`（`MetadataExport.tsx:45`）；清單到按鈕 12→16（`:53`）；圓角 → `rounded-[var(--radius-lg)]`；標頭 500→600（`:50`）。
26. 格式選項（稿→碼）：選中底 `accent-primary/10`（`:66`）→ `bg-[var(--accent-subtle)]`（token）；原生 radio（`:71-78`）→ 自繪 18 圓＋勾（**保留原生 input**，見 Dev Notes）；名稱 600。
27. 匯出鈕（稿→碼）：44 高、`px-5`、600、手機滿寬（`i8Tcke`）；碼約 36、`px-4`、500、手機不滿寬（`:91`）。
28. 匯入提示框：稿實線＋`$radius-md`，碼 `border-dashed`（`export.tsx:23`）→ ⚖️ **碼→稿**（虛線本身在說「還沒做」）。手機稿縮寫（`qI9hw`「適合程式處理」、`daYFd`）→ **碼→稿**（一套文案）。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| C5-D／C5-M | `uhAKd`／`gEQX4` | `mbZrH` 摘要、`oktnb` pill、`kOkTf`／`AKosK` 執行中列、`Na3S4` 表頭、`Jt3DP` 排程卡、`qxfV5` 頻率、`Q5Qx2` 時間、`iirbs` 保留份數、`nzgTk` 手機卡片、`G7vg0C` |
| C19-D／C19-M | `G8BYO`／`gPZU6` | `kxaMw` 圖示、`EOjW3`／`FqWF2` 標題、`ndpsv` 說明、`NOEJm`／`U0exxg`／`HjE3s` 檔案框、`JDzHR`／`PAmv4` 快照、`IvUhq` 取消、`lzMdn` 確認、`DvvOs` 手機抽屜、`E992A` 拉把、`n5XqGx` 手機按鈕列、`p95rQR` 陰影 |
| C20-D | `v2C4xr` | `Bdimi` 錯誤條、`qiiIx`／`y7rQ6J`／`syomc`、`ZiEhh` 摘要 |
| C13-D／C13-M | `nwn6a`／`Ytjrj` | `sF26K`／`JfvjU` 寬度、`xwygm` 選中、`W36o90` radio、`i8Tcke` 匯出鈕、`qI9hw`／`daYFd` 手機文案 |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP）**：🔴 #3（中性 pill）、#4、#8、#10、#11、#13、#16、#17、#18、#20（說明句）、#28。規格註記 `spec-note-dsr-3f`：
   > 「備份：手機每份備份一張卡，操作鈕 44；完成＝中性（平常狀態不給顏色），執行中只能刪除。排程只能選整點、保留規則固定（唯讀）、要按『儲存排程』。還原確認在手機是底部抽屜，顯示要用的備份（大小、日期）與會先建快照。建立失敗只說『沒成功＋去系統日誌看』（後端不分類錯誤），錯誤條在摘要上方，下面的表與排程照常顯示。匯入未實作的框用虛線。」
   收尾同 `dsr-3a` AC #1（只 stage `c5-*`／`c13-*`／`c19-*`／`c20-d`＋`pen-tokens.json`）。

2. **備份列表**：🔴 #1–#7。手機卡片清單與桌機表格**共用同一份列資料與同一組操作鈕元件**（不要寫兩份按鈕邏輯）；`<640` 用 `useIsPhone` 或純 CSS 切換皆可（dev 選，寫註解；純 CSS 會讓兩份 DOM 同時存在 → 既有 `getByRole` 斷言可能撞重複，選 `useIsPhone` 較安全，照 `dsr-4b-1` 的用法）。

3. **排程卡**：🔴 #9、#12。⛔ 排程欄位、儲存行為、每週備份日**不動**。

4. **還原確認**：🔴 #14–#16。⛔ 還原的 API 呼叫與成功後的行為不動。

5. **建立失敗與載入失敗**：🔴 #19、#20、#22。錯誤條 `role="alert"`；「重試」再呼叫一次建立（同一個 mutation）。

6. **匯出**：🔴 #24–#27。⛔ 匯出的格式值、下載行為不動。

7. **既有的行為不准回歸。**
   - `BackupManagement.spec.tsx`、`BackupTable.spec.tsx`、`BackupScheduleConfig.spec.tsx`、`RestoreConfirmDialog.spec.tsx`、`MetadataExport.spec.tsx` 的行為斷言不改；時間格式、頻率控制項型別（select→radiogroup）、錯誤文字屬本張刻意變更 → Completion Notes 逐條列。
   - ⛔ 不改後端、`backupService.ts` 的請求。

8. **測試。** 紅／守（Rule 16）。
   - `BackupTable.spec.tsx`：（紅）手機模式（mock `useIsPhone` true）渲染卡片清單、每張卡有還原／刪除且 44；執行中那筆只有刪除；pill `text-xs` 不帶 `text-[11px]`、中性 token、帶勾；時間 `YYYY-MM-DD HH:mm`（固定 TZ）；表頭與列用同一個欄寬常數（import 比對）；垃圾桶 `--error-text`。
   - `BackupManagement.spec.tsx`：（紅）摘要「N 份備份 · 已使用 X」＋圖示；建立失敗 → 錯誤條在摘要**之前**（DOM 順序）、標題「建立備份失敗」、說明逐字、有連到 `/settings/logs` 的連結、**不含** mock 的英文錯誤、按重試再呼叫 mutation；載入失敗 → `SettingsErrorState`。
   - `BackupScheduleConfig.spec.tsx`：（紅）頻率是 radiogroup、方向鍵切換、標籤「備份頻率」；開關圓點 token。
   - `RestoreConfirmDialog.spec.tsx`：（紅）`role="dialog"`、Esc 關閉、開啟時焦點在框內、關閉後焦點回到觸發鈕；檔案框兩行（檔名＋「大小 · 日期」）等寬；快照說明 `--info-tint`；取消鈕 `bg-tertiary`；手機模式渲染拉把且按鈕上下疊、確認在上。（守）確認呼叫還原、取消不呼叫。
   - `MetadataExport.spec.tsx`：（紅）選中 `--accent-subtle`；原生 radio 仍在且方向鍵可換選；匯出鈕 `min-h-11`、手機滿寬。
   - **視覺夾具**：既有 `settings-backup-management`、`settings-backup-table`、`settings-backup-schedule-config`、`settings-restore-confirm-dialog`、`settings-metadata-export` 基準**會變**、`penNode` 改真節點；新增 `settings-backup-table/mobile`（`width: 390`、`penNode: 'gEQX4'`）、`settings-backup-management/create-failed`（`penNode: 'v2C4xr'`）、`settings-restore-confirm-dialog/mobile`（**Portal → 用 `viewport: { width: 390, height: 844 }`**，照 `dsr-4b-2` 抽屜夾具的寫法；`penNode: 'gPZU6'`）、`settings-metadata-export/mobile`（`width: 390`、`penNode: 'Ytjrj'`）。
   - **e2e**（追加到 `tests/e2e/settings-shell.spec.ts`）：390 開 `/settings/backup`（stub 兩份完成＋一份執行中）→ 每張卡的還原與刪除鈕**在視窗內且可點**（`toBeInViewport`）、執行中那張沒有還原；按還原 → 底部抽屜出現、Esc 關閉、焦點回到還原鈕。1440：同樣流程是置中對話框。
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。

9. **CI 全綠**：同 `dsr-3a` AC #9。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：C5 pill／執行中列／排程卡（整點、唯讀保留、備份日、儲存鈕、手機補畫）、C19 強調與文案與陰影、C20 說明句、C13 虛線與文案、規格註記（AC: #1）**
- [x] **Task 2 — 備份列表：手機卡片（真 bug）、欄寬常數、表頭、等寬與時間、pill、操作鈕、摘要（AC: #2, #8）**
- [x] **Task 3 — 排程卡：頻率分段、開關圓點（AC: #3, #8）**
- [x] **Task 4 — 還原確認改 `ui/Dialog`＋手機抽屜＋內容對稿（AC: #4, #8）**
- [x] **Task 5 — 建立失敗錯誤條、整頁載入失敗（AC: #5, #8）**
- [x] **Task 6 — 匯出卡對稿（AC: #6, #8）**
- [x] **Task 7 — 夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [x] dev-story Step 9：`c5-d`／`c5-m`／`c13-d`／`c13-m`／`c19-d`／`c19-m`／`c20-d`

## Dev Notes

### 這張的重點

- **手機備份列表是真 bug**：今天手機上按不到還原與刪除。這一條先做、先驗。
- **還原確認換成共用 Dialog**：今天沒有焦點鎖定、Esc 關不掉——對一個會覆蓋全部資料的動作來說，這是最該補的。
- **後端給不出來的三樣（分鐘、保留份數、磁碟空間）全部改稿**，一個都不在這張加。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ 完成 pill 中性（碼→稿）；字級與勾（稿→碼）。
2. ⚖️ 排程：整點、唯讀保留、保留「儲存排程」鈕（全部碼→稿）。
3. ⚖️ 還原確認：`ui/Dialog`＋手機抽屜（照 `DeleteWithFilesDialog`）；保留 `<strong>`。
4. ⚖️ 建立失敗說明用通用句＋連到系統日誌；磁碟預檢另立 `disc-2026-09-backup-disk-space-precheck`。
5. ⚖️ 匯入提示維持虛線。
6. ⚖️ 本地時區 formatter 與 `dsr-3e` 共用（後做的重用）。

### 不要做的事

- 不要加後端欄位、磁碟檢查、錯誤分類。
- 不要把「儲存排程」改成自動儲存。
- 不要讓手機與桌機各自一份操作鈕邏輯。
- 不要把後端英文錯誤放上畫面。

### 已知陷阱

- **Portal 夾具要用 `viewport`**（`width` 框寬對 Portal 無效）——`dsr-4b-2`／`dsr-6f-*` 的先例。
- **`useIsPhone`** 在 `test-setup.ts` 的全域 `matchMedia` stub 下回 false（`dsr-4b-1` 的設計）→ 手機分支的 spec 要 mock hook。
- **自繪 radio**：`dsr-3d` C9 也做（20px）。後做的那張重用並統一成 20，Completion Notes 註明。保留原生 input、`peer-focus-visible:` 畫焦點。
- **Pencil／匯出／gh**：同 `dsr-3a`。

### Source tree

```
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c5,c13,c19}-{d,m}.png、c20-d.png   ← Task 1
apps/web/src/components/settings/BackupTable.tsx、BackupManagement.tsx（+spec）                          ← Task 2/5
apps/web/src/components/settings/BackupScheduleConfig.tsx（+spec）                                      ← Task 3
apps/web/src/components/settings/RestoreConfirmDialog.tsx（+spec）                                      ← Task 4
apps/web/src/routes/settings/export.tsx；components/settings/MetadataExport.tsx（+spec）                ← Task 6
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（追加）                ← Task 7
```

### Cross-Stack Split Check

後端 task **0**、前端／設計／測試 task 7 → 不觸發。規模：本系列最大（五個元件、一個 dialog 改寫、一個手機清單）。⚖️ 仍維持一張：切開的話「備份列表」與「還原確認」要各自 mock 同一份備份資料、同一個觸發鈕，兩張會各動 `BackupTable` 一次（`feedback_split_oversized_stories`：不要讓同一個元件被兩張單子各動一次）。若 dev 開工後發現超過 `dsr-4b-2` 的一倍半，可把 **C13 匯出**（Task 6，完全獨立的元件）拆成 `dsr-3f-2`，在 sprint-status 補條目。

### Time-dependent visual coverage

- `BackupTable`／`RestoreConfirmDialog` 顯示備份時間（不讀「現在」）→ 固定時間戳＋固定 TZ。`BackupScheduleConfig` 的「下次備份：…」若由前端用 `Date.now()` 算 → 凍結時間；若由後端給 → 固定值。dev 確認後寫進 Completion Notes。

### References

- [Source: `BackupManagement.tsx:105-158`；`BackupTable.tsx:10-14, 30, 58-138`；`BackupScheduleConfig.tsx`；`RestoreConfirmDialog.tsx:23-51`；`MetadataExport.tsx:45-91`；`routes/settings/export.tsx:20-27`；`services/backupService.ts:98`；`components/downloads/DeleteWithFilesDialog.tsx`（手機破壞性確認範本）；`components/ui/Dialog.tsx`、`mobileSheet.tsx`]
- [Source: `apps/api/internal/services/backup_service.go`（無磁碟預檢）；`BackupSchedule` 模型（整點）]
- [Source: `ux-design.pen` 節點見上表 —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `dsr-3a-settings-shell-page-header-and-states.md`；`dsr-4b-1`（`useIsPhone`）、`dsr-4b-2`（抽屜夾具 viewport）；`project_pen_design_token_system`（狀態詞彙）]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (1M context)（Amelia / dev-story）

### Debug Log References

- Pencil：裁切警告 66 → 66。存檔（選單 Save）後跑匯出時 Pencil 把檔案關掉了（MCP 回「A file needs to be open」、匯出 0/196）；磁碟檔已含全部改動（grep 到 `spec-note-dsr-3f` 與新說明句），`open -a Pen ux-design.pen` 重開、確認改動都在，再匯出 196/196。只 stage `c5-d`／`c5-m`／`c13-m`／`c19-d`／`c19-m`／`c20-d`＋`pen-tokens.json`（`c13-d` 沒動）。
- Pencil 的 lucide 沒有 `clock`（要用 `clock-3`）；schema 沒有虛線 stroke、`.pen` 沒有陰影變數——兩者都寫進 `spec-note-dsr-3f`。
- 本機 8080 有一個前次留下的 `api` 程序（`pkill -f "go run"` 只殺掉父程序），直接沿用。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **規模判斷**：五個元件原始碼合計約 800 行、spec 約 1180 行，沒有超過 `dsr-4b-2` 的一倍半 → 維持一張，沒拆 `dsr-3f-2`。
- **Task 1（稿）**：C5-D／C5-M 五顆「完成」pill 改中性（`bg-tertiary`／`text-secondary`，保留勾）；執行中那列刪掉還原／驗證／下載（桌機 3 顆、手機 3 顆）；手機操作鈕 36→44、建立鈕 36→44；手機摘要改成「7 份備份 · 已使用 346.5 MB」＋`hard-drive`；排程卡：標題前加 `clock-3`、說明改「下次備份：2026-09-14 03:00」、「頻率」→「備份頻率」、選中改「每週」、「執行時間」→「備份時間」＋下拉箭頭、「保留份數 7」→「備份日 週日」下拉、加唯讀「保留策略：…」與「儲存排程」鈕；手機補畫整張排程卡（欄位直排、儲存鈕滿寬 44），C5-M 高 844→1200。C19：說明句拆三段、「這將會取代目前所有的資料」加粗（兩張）；手機快照說明改成碼的整句（碼與桌機稿本來就逐字相同）；陰影保留 `#00000066`（沒有 token）。C20：說明改「備份沒有建立成功。請稍後再試；若持續失敗，到「系統日誌」查看原因。」。C13-M：三種格式說明與匯入提示改回完整句。`spec-note-dsr-3f`（`y9Jxg5`）。
- **Task 2（真 bug）**：`BackupTable` 手機（`useIsPhone`）改成卡片清單：時間＋「大小 · 檔名」等寬、pill、下面一排 44 高等寬操作鈕；桌機保留表格。**兩邊共用 `BackupActions`**（一份按鈕邏輯，`variant` 只換尺寸）。欄寬是一個常數 `BACKUP_GRID`（表頭與列共用）：`xl` 起照稿 440fr／120／200／120／176、gap 16；640–1280 之間設定欄只有約 600–1000px，固定欄縮成 80／128／80／auto、gap 12，檔名欄吃剩下的並截斷。表頭 `bg-tertiary`／`text-muted`；列 56 高、全部 Mono 12；時間 `<time dateTime>`＋新 `utils/formatLocalDateTime`（本地時區 `YYYY-MM-DD HH:mm`，帶 `precision: 'second'` 給 `dsr-3e` 重用）；pill `text-xs font-semibold`＋每種狀態一個圖示（完成＝中性）；操作鈕 32×32 `bg-tertiary`、垃圾桶 `--error-text`、每顆有 `aria-label`。摘要「N 份備份 · 已使用 X」＋`HardDrive`。
- **Task 3**：頻率改分段 `role="radiogroup"`（兩顆 `role="radio"`、roving tabindex、方向鍵切換並移焦點）；標籤「備份頻率」用 `aria-labelledby`（`getByLabelText('備份頻率')` 仍指到同一個 testid）；開關圓點 `text-on-accent`；「下次備份」改用同一個 formatter；卡片內距桌機 24／手機 16、下拉 40 高 `bg-tertiary`、欄位手機直排；「儲存排程」保留（桌機 36、手機滿寬 44）。
- **Task 4**：`RestoreConfirmDialog` 換 `ui/Dialog`：`role="dialog"`、焦點鎖定、Esc／✕／遮罩都能關；**開啟時焦點在「取消」**（破壞性確認不讓預設焦點落在確認鈕）；**還原進行中 Esc 關不掉**；沒有 Radix Trigger，所以自己記住開啟前的焦點（還原鈕），關閉時還回去。手機（`MOBILE_SHEET_CONTENT`＋`SheetGrabber`）是底部抽屜、按鈕上下疊滿寬 48、「確認還原」在上——用 DOM 順序交換而不是 CSS order，Tab 順序與畫面一致。內容：`rotate-ccw`、標題 H3（手機 18／桌機 20，`DialogTitle asChild` 保留 `<h3>`）、強調句保留 `<strong>`、檔案框兩行等寬（檔名＋「大小 · 日期」）、快照說明靛青 `info-tint`、取消 `bg-tertiary` 實心。
- **Task 5**：建立失敗 → `CreateBackupFailed`（抽出來給夾具用）：`role="alert"`、在摘要**上方**、「建立備份失敗」＋通用說明＋連到 `/settings/logs`＋「重試」（同一個 mutation；重試期間橫幅不消失、按鈕 `aria-disabled` 保住焦點，成功才消失）；後端英文不上畫面。整頁載入失敗 → `SettingsErrorState`「無法載入備份資訊／與後端的連線中斷了。已存在的備份檔不受影響。」＋`refetch`；轉圈只在「沒資料、沒錯誤、還沒抓完過」時出現（`dsr-3c` CR 的教訓：沒資料的失敗查詢一重抓就退回 pending）。
- **Task 6**：匯出卡 `radius-lg`、內距 24（手機 16）、標頭 600、清單到按鈕 16；選中 `--accent-subtle`；自繪 18px 圓＋勾（**原生 radio 保留**，`sr-only peer`，`peer-checked`／`peer-focus-visible` 畫狀態；`dsr-3d` C9 要做 20px 的，建議從這裡抽共用）；匯出鈕 44 高、`px-5`、600、手機滿寬；頁面 `max-w-3xl`；匯入提示保留虛線、改 `radius-md`。
- **Time-dependent visual coverage**：`BackupTable`／`RestoreConfirmDialog` 不讀「現在」；夾具用**不帶時區的 ISO**（當地時間解析），darwin 與 linux CI 不論時區都印出同一個「2026-09-11 03:00」。「下次備份」由後端給（`nextBackupAt`），夾具給固定值，不需要凍結時間。
- **既有 spec 改動（刻意，逐條）**：`BackupManagement.spec` — 載入失敗文字「無法載入備份資料」→「無法載入備份資訊」並斷言說明句；`shows error when backup creation fails` 改斷言**不出現** `Disk full`；`[P2] shows error message text from API error` 反轉成不顯示；摘要「N 個備份」→「N 份備份」。其餘行為斷言（`toBeDisabled` 的刪除／驗證／還原鈕、建立中停用、還原流程）未改。
- 🔗 **AC Drift: NONE**（checked: 6-5／6-7／6-8／6-9 的 AC——還原前確認與快照、每日／每週／停用與整點、JSON／YAML／NFO——全部是 REUSE：確認框仍警告取代資料與先建快照，排程選項與匯出格式值未動）。
- 📎 **Contract Stamps: NONE**（本張與上游皆無 `[@contract-v*]`）。
- 🎭 **A11y Pre-Flight: PASS**（5 個元件；觸碰檔 jsx-a11y 警告 0；還原框 modal 焦點鎖定、開啟聚焦取消、關閉還焦點、Esc；頻率 radiogroup 方向鍵；匯出原生 radio 保留；所有圖示鈕有名字；建立失敗 `role="alert"`）。
- **測試**：新 `formatLocalDateTime.spec` 6 條；`BackupTable.spec` +5、`BackupScheduleConfig.spec` +4、`RestoreConfirmDialog.spec` +8、`BackupManagement.spec` +5（含「Esc 關閉後焦點回到還原鈕」）、`MetadataExport.spec` +4。`nx test web` **288 files／4244 tests 全綠**；`web:typecheck`、`lint:all` 綠（0 error；新測試照檔案既有寫法用 `as any` mock hook，多了約 19 條 `no-explicit-any` 警告）。e2e `settings-shell.spec.ts` 追加 3 條（390 每張卡的還原／刪除在畫面內、≥44、執行中沒有還原；390 與 1440 各一條：還原開真的對話框、焦點在取消、手機貼底滿寬／桌機置中、Esc 關閉後焦點回還原鈕），`--repeat-each=3` **9／9**。
- **Mutation：unit 16／16 紅、e2e 2／2 紅**（拿掉手機卡片、手機鈕 32、舊表頭、舊日期格式、舊 pill 字級、垃圾桶中性、方向鍵失效、開關舊 token、還原中可 Esc、預設焦點、手機按鈕順序、快照中性色、錯誤條移到摘要下、重抓時轉圈、選中舊底色、原生 radio 被 `hidden`；e2e：拿掉手機卡片、拿掉還焦點）。第一輪「預設焦點」與「原生 radio」兩條沒紅 → 補了手機焦點與 `sr-only` 斷言後才紅。
- **視覺基準**：`settings-backup-table`、`settings-backup-management`、`settings-backup-schedule-config`、`settings-restore-confirm-dialog`、`settings-metadata-export`（3 張）的 darwin 基準改變，`penNode` 改成真節點；management／schedule 的 hover／focus 基準刪除（改 `statesOnly: ['default']`）；新增 `settings-backup-table/mobile`、`settings-backup-management/create-failed`、`settings-restore-confirm-dialog/mobile`、`settings-metadata-export/mobile`。過期 `-linux` 全部 `git rm`，等 CI bootstrap。手機夾具一律 `viewport: 390×844`（不是 `width: 390`）：`useIsPhone` 看視窗、Portal 不吃框寬，而且 `gallery-fixture-viewport.spec` 只允許這個尺寸。
- ⚠️ **與 story／稿的偏離**：① 還原框開啟時「取消」帶焦點環（程式聚焦＝`focus-visible`），稿沒畫；這是鍵盤使用者需要的，保留。② 手機還原說明句用 14（稿 12）：沿用 `dsr-3b` 的「手機不縮內文」，句子在 390 會折成兩行。③ 桌機排程欄位碼是「頻率＋時間（＋備份日）」各佔等寬，稿三欄等寬一致；但選「每日」時只有兩欄。④ 手機與桌機的列之間都沒有分隔線（照稿）。
- ~~沒處理、另外記一筆：驗證／刪除／還原失敗仍顯示後端英文~~ → CR ② 已在本張修掉（連同排程與匯出）。

- 🔍 **/ship 對抗式 CR（2026-09-24，獨立 context）1 HIGH／2 MEDIUM／4 LOW／4 NIT，吸收 1H／2M／4L／3N**：
  ① 🔴 **640–~810 的平板仍然切掉還原與刪除**：側欄從 640 起佔 240px，768 時表格只拿到約 480px，而小於 `xl` 的固定欄就要 522px，`overflow-hidden` 又把右邊切掉——同一個 bug 換了斷點。→ 卡片／表格改由**元件自己量到的寬度**決定（新 `hooks/useElementWidth`，ResizeObserver；`< TABLE_MIN_WIDTH`＝600 就用卡片），`useIsPhone` 只管第一次 render 與 jsdom；補 unit（量到 480／900）與 e2e（768 是卡片、按鈕不超出列表）。
  ② 🟠 驗證／刪除／還原／排程／匯出失敗仍印後端英文 → 全部改成固定中文（還原失敗附「系統日誌」指引），各補一條「不出現原文」測試。
  ③ 🟠 重試又失敗時錯誤條內容不變、報讀器不會再念 → 失敗次數計數，第二次起標題「建立備份失敗（已試 N 次）」；補「重試中錯誤條不消失、按鈕 `aria-disabled` 仍可聚焦」測試。
  ④ 還原中「取消」被 `disabled` 會把焦點丟到 body、✕ 看起來可按 → 兩顆鈕改 `aria-disabled`＋防呆，還原中隱藏 ✕。⑤ 手機 ✕ 比標題高約 18px → `max-sm:top-[38px]`。⑥ 開著對話框時轉手機（表格↔卡片）原本的還原鈕會消失 → 找同一份備份的還原鈕還焦點。⑦ 視覺夾具補一筆失敗的備份、排程卡保留 focus 基準。
  NIT：操作鈕名稱帶檔名（「還原 vido-backup-…db」）、頻率加 Home／End、執行中圖示會轉。未改：`ui/Dialog` 的 ✕ 報讀名稱是英文「Close」（共用元件、全站既有）。
  CR 後：mutation 再 7／7 紅；e2e 768 那條拿掉寬度判斷會紅；`nx test web` **288 files／4251 tests** 綠；`lint:all`、typecheck 綠；e2e 整檔 `--repeat-each=3` 51／51。

### File List

- `ux-design.pen`
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-c-search-settings/{c5-d,c5-m,c13-m,c19-d,c19-m,c20-d}.png`
- `apps/web/src/utils/formatLocalDateTime.ts`（新）
- `apps/web/src/utils/formatLocalDateTime.spec.ts`（新）
- `apps/web/src/hooks/useElementWidth.ts`（新，CR）
- `apps/web/src/components/settings/BackupTable.tsx`（+spec）
- `apps/web/src/components/settings/BackupManagement.tsx`（+spec）
- `apps/web/src/components/settings/BackupScheduleConfig.tsx`（+spec）
- `apps/web/src/components/settings/RestoreConfirmDialog.tsx`（+spec）
- `apps/web/src/components/settings/MetadataExport.tsx`（+spec）
- `apps/web/src/routes/settings/export.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/settings-shell.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-{backup-table,backup-management,backup-schedule-config,restore-confirm-dialog,metadata-export}/…`（darwin 改／新增、linux 刪除、management／schedule 的 hover／focus 刪除）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### UX Verification（dev-story Step 9）

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| 摘要＋建立（C5） | hard-drive＋「7 份備份 · 已使用…」、建立鈕 40 | 同（手機 44） | ✅ | — |
| 表格（C5-D） | 表頭 bg-tertiary／muted、56 列、Mono 12、中性完成 pill、32 操作鈕、紅垃圾桶、執行中只刪除 | 同 | ✅ | — |
| 手機卡片（C5-M） | 時間＋大小·檔名、pill、44 操作鈕一排 | 同 | ✅ | — |
| 排程卡（C5） | 分段頻率、整點、備份日、唯讀保留、儲存鈕 | 同 | ✅ | 欄寬見偏離 ③ |
| 還原確認（C19-D／M） | 置中／底部抽屜、強調句、兩行檔案框、靛青快照、實心取消 | 同 | ✅ | 焦點環、手機字級見偏離 ①② |
| 建立失敗（C20） | 摘要上方錯誤條、18 圖示、標題＋說明＋重試 | 同（「系統日誌」是連結） | ✅ | — |
| 匯出（C13-D／M） | 768 寬、選中 accent-subtle、18 自繪圓、44 匯出鈕、手機滿寬 | 同；匯入框虛線（稿畫不出） | ✅ | — |

🎨 UX Verification: PASS — `c5-d`／`c5-m`／`c13-d`／`c13-m`／`c19-d`／`c19-m`／`c20-d` 對照視覺基準逐項比對。

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-24 | Task 1：設計稿——中性完成 pill、執行中只刪除、排程卡照產品（整點／備份日／唯讀保留／儲存鈕）、手機排程卡、還原強調句、建立失敗通用句、C13-M 完整文案、`spec-note-dsr-3f` |
| 2026-09-24 | Task 2：手機備份卡片（真 bug）、共用操作鈕、欄寬常數、等寬時間、pill、摘要；新 `formatLocalDateTime` |
| 2026-09-24 | Task 3：頻率分段 radiogroup、開關圓點 token |
| 2026-09-24 | Task 4：還原確認換 `ui/Dialog`＋手機底部抽屜、焦點進出、還原中不能 Esc |
| 2026-09-24 | Task 5：建立失敗錯誤條（摘要上方、通用句、系統日誌連結、重試）、整頁載入失敗換 `SettingsErrorState` |
| 2026-09-24 | Task 6：匯出卡對稿（accent-subtle、自繪 radio、44 匯出鈕、768 寬） |
| 2026-09-24 | /ship CR：平板寬度改卡片（量元件寬度）、所有失敗訊息改中文、重試失敗再宣讀、還原中焦點與 ✕、手機 ✕ 對齊、焦點回退 |
| 2026-09-24 | Task 7：夾具 5 改 4 新、e2e 3 條、mutation 18／18、全套 web 4244 綠 |
