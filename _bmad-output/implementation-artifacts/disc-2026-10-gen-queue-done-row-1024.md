# Disc：筆電寬度（1024）看生成畫面，左邊佇列「完成」那列的說明被「…」切掉

Status: ready-for-dev

**Source:** 2026-10-20 #871（`disc-2026-10-gen-queue-reason-wrap-desktop`）Dev Agent Record「附帶發現」：1024 寬、右邊即時活動欄有顯示時，左邊佇列「完成」列的說明行「已完成，字幕已寫入檔案」要 154px、格子只有 142px，被切成「已完成，字幕已寫入…」。#871 之前就是這樣；1280、1440 沒事。⚖️ Alexyu 2026-10-20：開單，請 Sally 決定怎麼修。

## ⚖️ Sally 裁定（2026-10-20）

**選 (a) 改短文案：「已完成，字幕已寫入檔案」→「字幕已寫入檔案」。**

白話：那一列右邊的徽章本來就寫「完成」，說明行再寫一次「已完成」是重複；拿掉之後剩 7 個字（約 98px），1024 也一行放得下，還留 44px 的餘裕。

為什麼不選其他三個：

- **(b) 所有說明行都可以折行**——不選。#871 才裁過「除了失敗／已略過，其他說明行 640 以上一律一行」，因為處理中那行（「翻譯中…」）會一直變，可以折行的話列會忽高忽低、整個佇列跟著跳。為了一句固定文字推翻這條，不划算。
- **(c) 1024 時把右邊即時活動欄變窄**——不選。即時活動欄自己也有「片名＋原因」要放（F11-SPEC-LOG ④④b），1024 時它已經從 400 縮到 396；再縮是拿右邊換左邊，而且是為了一句話動整個版面。
- **(d) 不重要的行就讓它被切**——不選。被切掉的正好是「檔案」兩個字，也就是「字幕寫到哪了」這個唯一的新資訊；「完成」徽章已經講了前半句。切掉後半句等於這行沒用。
- **為什麼不是更短的「字幕已寫入」**：「檔案」告訴使用者字幕已經放在影片旁邊（播放器讀得到），不是只存在 Vido 裡。7 個字已經放得下，不需要再砍。

文案守則照舊：**不寫「繁中」**（`queueRowSubStatus` 檔頭 ⚠️：一次執行可能留英文段落或照陸片政策保留簡體，P0 測試擋著）。新句子也不寫。

**設計稿要改（所以本單先 blocked）：** 設計稿四列「完成」的說明行跟程式不一致，而且其中三列寫的是程式**禁止說**的「已生成繁中字幕」：

| 節點 | 在哪 | 現在的字 |
|---|---|---|
| `oxc7d` | F11-D-v2 `l8FsB` row-完成 | 已生成繁中字幕，已寫入檔案 |
| `bbDtK` | F12-D-v2 `iH98f` row-完成 | 已生成繁中字幕，已寫入檔案 |
| `Qp7Cm` | F11-M-v2 `PXB0z` row-完成 | 已完成，字幕已寫入檔案 |
| `dzmKk` | Component Library `sJzat` GenQueueRow-v2 sample | 已生成繁中字幕 |
| `aw4Qr` › `lUZol` | 元件母版 Component/GenQueueRow-v2 的預設字 | 已生成繁中字幕 |

全部改成「字幕已寫入檔案」。只改字，版面、樣式不動。提示詞見下方「Pencil 提示詞」。

## Story

身為在筆電（1024 寬）上看生成畫面的使用者，
我想要佇列裡做完的那部片，說明行整句看得到「字幕已寫入檔案」，
這樣我不用把視窗拉寬，就知道字幕已經放在影片旁邊、播放器讀得到。

## Acceptance Criteria

0. **開工前：** 確認設計稿已改（Pencil MCP 唯讀 `Get` 逐字比對上表五個節點都是「字幕已寫入檔案」），且截圖 PR 已合併（`f11-d-v2.png`、`f12-d-v2.png`、`f11-m-v2.png`、`component-library.png`）。沒改就停下來等。
1. **文案：** `queueRowSubStatus` 的 `done` 分支回「字幕已寫入檔案」（7 個字，沒有標點）。其他分支一個字都不動。
   - 對話框（F8，`GenerationBatchDialogV2.tsx`）的完成列沒有第二行，不受影響——開工時 `git grep queueRowSubStatus` 確認只有工作區在用。
2. **class 不動：** 說明行維持 #871 的規則——失敗／已略過 `whitespace-normal break-words`；其他狀態（含完成）`truncate max-sm:whitespace-normal`。本單只改字，不改 class。
3. **量測合格線（真瀏覽器，改完量）：** 頁面 `/activity?view=generation`，照 #871 的做法（狀態查詢＋假 SSE stub，量完刪掉、不進版控）。佇列放：一部完成、一部處理中（翻譯中，有百分比）、一部排隊中、一部失敗（「影片所在的資料夾寫不進字幕」）、一部已略過（19 字原因）。另外跑一次預算上限終態（有「已暫停 — 下次繼續」的列）。
   - **1024**（右邊即時活動欄有顯示）：完成列說明行 `scrollWidth <= clientWidth`、畫面沒有「…」；**排隊中、已暫停、處理中**說明行也逐列量一次，全部沒有「…」；失敗／已略過原因整句（可折行）；片名最多兩行；徽章一行；頁面無橫捲。
   - **1280**：同上全部列一行整句（失敗／已略過一行）；無橫捲。
   - **390**、**320**：完成列說明行一行整句（7 字在 320 也放得下）；其他列跟改之前一樣；無橫捲。
   - ⚠️ 如果 1024 時「已暫停 — 下次繼續」（徽章「已暫停」比「完成」寬，格子估計約 130px、句子估計約 120px）或其他非失敗列還是被切：**不要順手改**，在 Dev Agent Record 記下數字，回報 Sally 另外裁定。
4. **測試：**
   - `generationQueueRow.spec.ts` 的 `[P0] a done row never claims 繁中` 改成期待「字幕已寫入檔案」，並加一條：不含「已完成」（徽章已經講了）。仍不含「繁中」。
   - `GenerationWorkspaceV2.spec.tsx` 兩處 `'已完成，字幕已寫入檔案'`（約 435、1328 行）改成新字；斷言內容（`truncate`／`max-sm:whitespace-normal`）不變。
   - 先改測試看它紅，再改程式看它綠。
5. **視覺基準：** `generation-workspace-v2/running`、`complete-with-failures`、`failures-long-titles` 這三張有完成列，基準**會變**（文字變短）。全套 visual 跑一次；這三張 darwin 本機刪掉後 `update-missing` 重產、`-linux` 版 `git rm`，交給 CI 的 `chore(visual): bootstrap` PR。其他生成相關基準不應變；有 byte 差但沒有完成列的照慣例當重產雜訊還原。
6. **UX 驗收：** 1024 截圖（右欄有顯示）對 F11-D-v2 `oxc7d` 的新文案；390 截圖對 F11-M-v2 `Qp7Cm`。截圖附 PR。
7. **全綠：** `pnpm nx run web:test`、`web:typecheck`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`、`scripts/check-design-refs.py`。沒有刪改 testid；E2E 先 `git grep 已完成，字幕已寫入 tests/e2e` 確認沒有斷言這句（2026-10-20 查過沒有）。

## Tasks / Subtasks

- [ ] T0：確認設計稿與截圖已更新（AC #0）
- [ ] T1：先改兩支 spec 的期待值，跑一次看紅（AC #4）
- [ ] T2：`queueRowSubStatus` `done` 分支改字（AC #1）；跑 spec 看綠
- [ ] T3：真瀏覽器量 1024／1280／390／320（AC #3），數字寫進 Dev Agent Record
- [ ] T4：視覺基準（AC #5）＋ UX 截圖（AC #6）
- [ ] T5：全綠（AC #7）

## Dev Notes

- 只改一個字串（`generationQueueRow.ts` `queueRowSubStatus` 的 `case 'done'`）和三處測試期待值。檔頭 ⚠️「`done` does NOT say 繁中」那段註解保留，可以補一句「也不重複『已完成』——徽章已經講了（disc-2026-10-gen-queue-done-row-1024）」。
- 為什麼 1024 才會切：工作區 `lg:` 以上左右並排、右欄 `lg:w-[400px]`；1024 是並排的最窄寬度，右欄縮到 396，左邊完成列的文字格剩 142px。640～1023 右欄在下面，佇列夠寬；1100 以上沒切（#871 量過）。
- 字寬估算：`text-sm` 14px，中文一字約 14px。「字幕已寫入檔案」7 字約 98px。
- 螢幕報讀：一列讀起來是「片名 字幕已寫入檔案 完成」，意思完整，不用另外補 sr-only。

### 不要做的事

- 不改說明行的 class、不讓其他狀態折行（#871 的規則）。
- 不改右欄寬度、不改徽章。
- 不改「排隊中」「已暫停」的文案——量到被切就回報，不在本單修。
- 不碰對話框（F8）與即時活動（右欄）。

### 設計稿 / Design ref

- F11-D-v2 `l8FsB`，完成列 `oxc7d`（佇列 `KZmNG`）
- F11-M-v2 `PXB0z`，完成列 `Qp7Cm`
- F12-D-v2 `iH98f`，完成列 `bbDtK`
- 元件 Component/GenQueueRow-v2 `aw4Qr`（說明行 `lUZol`）
- F11-SPEC-LOG `tT3JX`（即時活動規格；本單不改，只是同一個畫面）
- 沒有獨立的「工作區佇列規格」稿；列的兩行規則寫在 F8-SPEC-ROW `NfHKO`（完成維持一行）。

### Pencil 提示詞（Alexyu 貼給 Pen 的 Inline AI Agent）

```
在 ux-design.pen 裡只改文字內容，不改版面、不改樣式、不新增或刪除節點。

1. 元件母版 Component/GenQueueRow-v2（id aw4Qr）裡的說明行 qr-substatus（id lUZol）：預設文字「已生成繁中字幕」改成「字幕已寫入檔案」。
2. 下面四個 aw4Qr 實例，把各自覆寫的 lUZol 文字改成「字幕已寫入檔案」：
   - oxc7d（F11-D-v2 · 生成工作區 — 執行中，id l8FsB，queue-list KZmNG 裡的 row-完成）：原本「已生成繁中字幕，已寫入檔案」
   - bbDtK（F12-D-v2 · 生成工作區 — 預算上限終態，id iH98f，row-完成）：原本「已生成繁中字幕，已寫入檔案」
   - Qp7Cm（F11-M-v2 · 生成工作區（手機），id PXB0z，row-完成）：原本「已完成，字幕已寫入檔案」
   - dzmKk（Component Library，id sJzat，GenQueueRow-v2 sample）：原本「已生成繁中字幕」
3. 其他 aw4Qr 實例（row-進行中、row-排隊、row-失敗、row-已略過、已暫停的列、F13-D-v2 的 p2-live-row）的 lUZol 一個字都不要動；徽章 OLg1J（「完成」）不要動。
4. 不要改字級、顏色、間距、寬度。如果工具一定要寫樣式值，只能用檔案裡現有的 $ 變數（例如 $Type/Body/Size、$text-secondary、$Space/sm），數值 0 一律寫 $Space/none，不要寫裸數字或色碼。
5. 改完自己檢查：上面五個節點的文字逐字是「字幕已寫入檔案」（7 個字，沒有標點、沒有「繁中」、沒有「已完成」）；F11-D-v2 與 F12-D-v2 的 row-完成 高度仍是 88；文字沒有超出列。

不要切換分支、不要 git commit、不要開 PR；改完存檔（⌘S 或選單 File → Save），回報改了哪些節點 id。
```

存檔後：Sally 用 Pencil MCP 唯讀 `Get` 逐字複審五個節點；再出 `f11-d-v2`、`f12-d-v2`、`f11-m-v2`、`component-library` 四張截圖（另一個 session 占用工作目錄時，照記憶 `feedback_pen_inline_agent_workflow` 在 scratch 複本出圖、plumbing commit），**只 stage 這四張**，其餘重產雜訊不進版控。截圖 PR 合併後本單改 `ready-for-dev`。

### References

- [Source: apps/web/src/components/subtitle/generationQueueRow.ts:182-208（queueRowSubStatus 與檔頭 ⚠️）]
- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:331-350（說明行 class）、:520（右欄 lg:w-[400px]）]
- [Source: apps/web/src/components/subtitle/generationQueueRow.spec.ts:302-306]
- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx:435、:1328]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-gen-queue-reason-wrap-desktop.md Dev Agent Record「附帶發現」（1024：右欄 396、文字格 142、這句 154）]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-20 | 建單（Bob）。Sally 裁定 (a)：完成列說明行改成「字幕已寫入檔案」（徽章已寫「完成」，不重複）；設計稿四列＋元件母版同步改字，先 blocked 等 Pencil。 |
| 2026-10-20 | 解除 blocked → ready-for-dev：Alexyu 跑完 Pencil，母版 aw4Qr›lUZol＋oxc7d／bbDtK／Qp7Cm／dzmKk 的說明行改成「字幕已寫入檔案」，截圖 f11-d-v2／f12-d-v2／f11-m-v2／component-library 隨設計稿 PR 一起提交。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
