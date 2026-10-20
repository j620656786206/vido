# Disc：生成畫面左邊佇列，桌機中等寬度時失敗／已略過的原因可能被「…」切掉

Status: done

**Source:** 2026-10-20 Sally 畫 F11「已略過」稿（PR #866）時發現。**推測、還沒量過**——開工第一步就是量。

⚖️ **SM 裁定（Bob，2026-10-20）：**
- P3 前端單張，規模小（一個 class＋兩段註解＋一張基準），不拆。
- **先量再改**：1024／1280 量起來沒被切，本單以 no-op 結案（只修兩段註解另說，見 AC #1）。
- ⛔ **依賴 PR #866**（設計稿 `LJCbp`／`XHbwN` 在那張進 `.pen`）。開工時 #866 還沒合併就等它。

## 查到的事（2026-10-20，`origin/main` 9d6873be）

- 工作區左邊佇列每一列的第二行（說明行，`queueRowSubStatus` 回的字，`GenerationWorkspaceV2.tsx:336-343`）class 是 `truncate text-sm max-sm:whitespace-normal`：**640 以下會折行，640 以上一律一行、超出就「…」**。
- 失敗／已略過列的說明行就是原因句（`queueRowSubStatus` 的 `failed` 分支回 `labelText`，`generationQueueRow.ts:193-208`），最長的一句是 19 字「沒有可用的字幕來源，也沒有設定語音辨識」。
- F8-SPEC-ROW（`NfHKO`）與 F11-SPEC-LOG（`tT3JX`）都寫：**原因一律整句顯示、會折行，不截斷**。
- Sally 推測：1024～1200 寬時，右邊即時活動欄固定 400 寬，左邊佇列的說明行可能放不下 19 字 → 被切成「沒有可用的字幕來源，也沒有…」。
- 已量過的：1440 一行整句（skip-b AC #12，佇列片名格 562px）；390／320 會折行、整句（`max-sm:whitespace-normal`）。**1024／1280 沒量過。**
- 兩段註解過期：
  - `generationEventCopy.ts` 檔頭寫「F11-SPEC-LOG has no such row yet（待 Sally 補稿）」——#866 已補 ④b `XHbwN`。
  - `GenerationWorkspaceV2.tsx` `FeedRowItem` 上方註解寫「Line 2 (④ 失敗 / ⑤ / ⑤b 已停止 only)」——已略過列（④b）也有第二行。

## Story

身為在筆電（1024～1280 寬）上看生成畫面的使用者，
我想要左邊佇列的失敗／已略過原因整句看得到，
這樣我不用把視窗拉寬或點進去，就知道那部片為什麼沒做。

## Acceptance Criteria

0. **開工前：** 確認 PR #866 已合併（`.pen` 有 `LJCbp`、`XHbwN`，`_bmad-output/pen-node-ids.json` 列得到）；沒合併就停下來等。
1. **先量（真瀏覽器）：** 真頁面 `/activity?view=generation`（照 skip-b AC #12 的做法：狀態查詢＋假 SSE stub），佇列放三部片：`skipped + no_text_source`（19 字原因）、一般失敗（例如「影片所在的資料夾寫不進字幕」）、一部完成。寬度 **1024、1280**，右邊即時活動欄要有顯示。量每列說明行的 `scrollWidth` 與 `clientWidth`（或看有沒有「…」）。
   - **兩個寬度都沒被切** → 不改 class；只做 AC #4（註解）＋在 Dev Agent Record 寫下量測數字，Status → done（no-op）。
   - **有任何一個寬度被切** → 做 AC #2～#5。
2. **只有失敗／已略過列的原因行會折行**（`status === 'failed'`，含 `queueRowIsSkipped` 為真的列）：拿掉這一行的 `truncate`，改成可折行（`whitespace-normal break-words` 之類），不截斷、不加行數上限。
   - **其他狀態的說明行**（完成、處理中「翻譯中…」、已暫停、已取消、排隊中）**維持現狀**：640 以上 `truncate`、640 以下折行。
   - 處理中那行的顏色（`label.className`）不動。
3. **量測合格線（改完再量一次）：** 1024、1280、1440 三個寬度，失敗與已略過列原因整句看得到（沒有「…」、`scrollWidth <= clientWidth`）；片名仍最多兩行；徽章仍一行；頁面無橫捲。390／320 跟改之前一樣（原本就折行）。
4. **兩段註解（不論 AC #1 結果都要做）：**
   - `generationEventCopy.ts` 檔頭：「F11-SPEC-LOG has no such row yet（待 Sally 補稿）」改成指向 F11-SPEC-LOG ④b（`XHbwN`）。
   - `GenerationWorkspaceV2.tsx` `FeedRowItem` 上方註解：「Line 2 (④ 失敗 / ⑤ / ⑤b 已停止 only)」補上 ④b 已略過。
5. **測試（只在 AC #2 有改時）：** `GenerationWorkspaceV2.spec.tsx` 加一支：失敗列與已略過列的說明行**沒有** `truncate` class；完成／排隊中列**仍有**。故意改壞一次（把失敗列改回 `truncate`）確認會紅，再還原。
6. **視覺基準：** `generation-workspace-v2` 的基準可能變（主要是 `complete-with-failures`）。全套 visual 跑一次，超過門檻的照慣例：darwin 本機刪掉後 `update-missing` 重產、linux `git rm`，交 CI bootstrap。其餘生成相關基準不應變。
7. **UX 驗收：** 1280 截圖對 F11-D-v2（`l8FsB`）的佇列列 `LJCbp`，原因折行方式對 F8-SPEC-ROW（`NfHKO`）與 F11-SPEC-LOG ④b（`XHbwN`）「整句、會折行、不截斷」。截圖附 PR。
8. **全綠：** `pnpm nx run web:test`、`web:typecheck`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`、`scripts/check-design-refs.py`。沒有刪改 testid（E2E 不受影響；改了就先 grep `tests/e2e`）。

## Tasks / Subtasks

- [x] T0：確認 #866 已合併（AC #0）
- [x] T1：1024／1280 真瀏覽器量測，記錄數字（AC #1）
- [x] T2：修兩段註解（AC #4）
- [x] T3（被切才做）：失敗／已略過列原因行改可折行（AC #2）＋ spec（AC #5）
- [x] T4（被切才做）：改完量 1024／1280／1440／390／320（AC #3）
- [x] T5（被切才做）：視覺基準（AC #6）＋ UX 截圖（AC #7）
- [x] T6：全綠（AC #8）

## Dev Notes

- 判斷「失敗或已略過」用 `item.status === 'failed'` 就夠（已略過列的 status 也是 `failed`，`queueRowIsSkipped` 只決定灰色）；不用另外呼叫 `queueRowIsSkipped`。
- `queueRowSubStatus` 的文字不動，只動 class。
- 產生字幕對話框（F8，`GenerationBatchDialogV2.tsx`）的兩行結構 #851 已做好、原因本來就會折行，**不在本單**。
- 即時活動（右欄）的原因行本來就整句（`FeedRowView.detail`），不在本單；只改它的註解。

### 不要做的事

- 不改原因文字、不改徽章、不改片名行數上限。
- 不動其他狀態說明行的截斷行為（處理中那行長了會撐高列、讓佇列跳動）。
- 不碰對話框、即時活動的版面。

### 設計稿 / Design ref

- F11-D-v2 `l8FsB`，佇列列 `LJCbp`
- F11-SPEC-LOG `tT3JX`，④b 已略過 `XHbwN`
- F8-SPEC-ROW `NfHKO`（原因一律整句、會折行）
- 以上 `LJCbp`／`XHbwN` 由 PR #866 加進 `.pen`。

### References

- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:336-343（說明行 class）、:396-405（FeedRowItem 註解）]
- [Source: apps/web/src/components/subtitle/generationQueueRow.ts:193-208（queueRowSubStatus）]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:24-31（檔頭註解）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-no-text-source-skip-b.md AC #12（1440／390／320 量測做法與數字）]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-20 | 建單（Bob）。Sally 推測 1024～1200 寬佇列原因被「…」切掉，先量再改；只讓失敗／已略過列原因折行；順手修兩段過期註解。⛔ 等 PR #866 合併。 |
| 2026-10-20 | Dev（Amelia）：1024 量到被切（266 > 242）→ 失敗／已略過列原因行改折行；兩段註解修好；基準不變。Status → done。 |

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- 量測用暫時的 Playwright spec（chromium，`/activity?view=generation`，狀態查詢 `last` ＋假 SSE stub，量完刪掉、不進版控）。截圖在 session scratchpad `shots/`（before-*／after-*）。

### Completion Notes List

- **AC #0：** #866 已合併（7806c98d）；`pen-node-ids.json` 列得到 `LJCbp`、`XHbwN`。
- **AC #1 量測（改之前）：** 佇列三部片：和平使者 S01E01（`skipped + no_text_source`，19 字原因）、魔戒首部曲：魔戒現身 加長版導演剪輯特別收藏版（`error + target_not_writable`「影片所在的資料夾寫不進字幕」）、駭客任務（完成）。即時活動欄都有顯示、三列。
  - **1024：被切。** 略過列說明行 `scrollWidth 266 > clientWidth 242`，畫面是「沒有可用的字幕來源，也沒有設定語…」。即時活動欄 304 寬。
  - 1280：沒切（略過列 402／402、失敗列 394／394）；即時活動欄 400。
  - 1440：沒切（562／562）。390、320：本來就折行、整句（略過列 390 兩行、320 三行）。
  - → 有一個寬度被切，做 AC #2～#5。
- **附帶發現（改之前，同一份舊程式）：** 1024 的即時活動欄 304 寬是長原因「不折行」把左欄最小寬度撐大、擠窄右欄的副作用。換成一般內容再量：只有完成／排隊中時即時活動欄 396、文字欄 142，**「已完成，字幕已寫入檔案」本來就被切**（154 > 142）；一完成一失敗時即時活動欄 368、文字欄 170，**失敗原因「影片所在的資料夾寫不進字幕」也被切**（182 > 170）。1100、1200 都沒切。所以原因被切在 1024 不只略過那句，一般失敗也會。
- **AC #2：** 說明行 class 改成：`status === 'failed'`（含已略過）→ `whitespace-normal break-words`（不截斷、不限行數）；其他狀態 → 照舊 `truncate max-sm:whitespace-normal`。處理中那行顏色（`label.className`）沒動，文字沒動。
- **AC #3 量測（改之後）：**
  - 1024：略過列原因 150／150 折兩行整句；失敗列原因 142／142 折兩行整句；片名最多兩行（魔戒兩行後「…」）；徽章一行（高 30）；頁面寬 1024 無橫捲。即時活動欄回到 396（跟一般內容的舊行為一樣）。完成列「已完成，字幕已寫入…」照舊被切——屬上面講的既有狀況，story 規定其他狀態維持 truncate，本單不動。
  - 1280：略過 402／402、失敗 394／394，一行整句；即時活動 400；無橫捲。
  - 1440：562／562、554／554，一行整句；無橫捲。
  - 390：略過兩行、失敗一行、完成一行，跟改之前一樣；320：略過三行、失敗兩行、完成兩行，跟改之前一樣；無橫捲。
- **AC #4：** `generationEventCopy.ts` 檔頭改指 F11-SPEC-LOG ④b 已略過（`XHbwN`）；`FeedRowItem` 註解補上 ④b 已略過。
- **AC #5：** `GenerationWorkspaceV2.spec.tsx` 加一支：失敗列與已略過列說明行沒有 `truncate`、有 `whitespace-normal`、沒有 `line-clamp`、仍是次要色；完成、排隊中、處理中仍有 `truncate max-sm:whitespace-normal`。先寫先紅（舊 class 有 `truncate`）；改完綠；故意把失敗分支加回 `truncate` → 紅，還原。
- **AC #6 視覺基準：** 全套 visual 本機綠。`generation-workspace-v2` 六張 darwin 刪掉後 `update-missing` 重產比對：五張 byte 相同（含 `complete-with-failures`——畫廊寬 1200、沒有側欄，原因本來就一行）；`credit_exhausted` 有 byte 差，但它沒有失敗列，是重產雜訊，已還原。**沒有基準要換**，不需要 CI bootstrap。
- **AC #7 UX：** 1280 截圖對 F11-D-v2（`f11-d-v2.png`）佇列列：失敗、已略過列原因整句在片名下一行，排法一致；1024 原因折行整句，符合 F8-SPEC-ROW／F11-SPEC-LOG ④b「整句、會折行、不截斷」。
- **AC #8：** `pnpm nx run web:test` 298 檔／4854 支全綠；`web:typecheck` 綠；`lint:all` 0 errors（✖ 168／170 problems 都是 warnings）；`format:check` 綠；`check-design-refs.py` 綠。沒刪改 testid；另跑 `generation-workspace.spec.ts`、`generation-workspace-mobile.spec.ts`（chromium＋mobile-chrome）14 支綠、6 支照設計 skip。

### File List

- apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx
- apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx
- apps/web/src/components/subtitle/generationEventCopy.ts（只改註解）
- _bmad-output/implementation-artifacts/disc-2026-10-gen-queue-reason-wrap-desktop.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
