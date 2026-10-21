# Backlog -b：「本月 AI 花費」和首頁「本月 $x」把檔名解析花的錢算進去（設計稿＋前端）

Status: done

**Source:** 從 `backlog-ai-parse-reads-env-keys-only` 拆出來，見 -a 的拆單說明。⚖️ Alexyu 2026-10-21 選 A：AI 檔名解析打開，花的錢要記進花費紀錄。-a 負責把錢逐筆記下來，並在 `/subtitles/spend` 回傳 `parse_calls`、`parse_usd`；本單負責讓使用者看得到。

⛔ **blocked-by：**
1. -a 合併（API 要先有這兩個欄位）。
2. 設計稿改好、截圖 PR 合併（見下方 Pen 提示詞）。

## ⚖️ Sally 裁定（2026-10-21）

- **放哪裡：** 「本月 AI 花費」卡片的頭條那一行（`row-用量`），在「語音辨識」右邊加第三格「檔名解析 N 次 · $x」。樣式完全照抄「語音辨識」那格。
  - 桌機：同一行往右排。
  - 手機：`row-用量` 本來就是直排，所以排在第三行。
- **什麼時候出現：** 這個月 `parse_calls > 0` 才出現。沒有就整格不畫，不用「—」。
  - 「—」的規則是給翻譯和語音辨識兩條字幕路線用的：它們一定會出現，所以「沒有紀錄」要明講。
  - 檔名解析只有設了金鑰、而且碰到看不懂的檔名才會發生。大多數月份是 0，每個月都掛一個「檔名解析 —」只會讓人疑惑。
- **卡片要不要出現：** 這個月如果只有檔名解析花了錢、沒有任何字幕 run，卡片也要出現（只有這一格）。不能讓花了錢卻看不到。
- **不放進「依模型」：** 那一欄統計的是字幕 run，把解析的次數混進去，「9 次」就不再代表 9 部片了。
- **首頁「本月 $x」：** 把 `parse_usd` 加進總額（`HomeReadoutBand.tsx:106-109` 的 `monthSpendUsd`）。首頁的總額要等於使用者這個月實際花的錢。
- **名稱用「檔名解析」**：跟設定頁的 hint「也用於 AI 檔名解析」、活動記錄的「解析失敗」一致。產生搜尋關鍵字也是為了認出是哪部片，所以一起算在這一格，不另外開一格。
- **報讀：** 這一格的無障礙名稱照「語音辨識」那格的寫法，例如「檔名解析 37 次，花費 $0.18」。開工時照 `SpendSection.tsx` 現有的 aria 寫法。

## Story

身為打開了 AI 檔名解析的使用者，
我想要在「本月 AI 花費」看到檔名解析這個月問了 AI 幾次、花了多少，首頁的「本月 $x」也包含這筆錢，
這樣我知道掃描片庫實際花了我多少錢，不會有看不到的花費。

## Acceptance Criteria

0. **開工前：** 用 Pencil MCP 唯讀 `Get`，確認 K5-D-v2 `xgYKA`、K5-M-v2 `ptNai` 的 `row-用量`（`hbiao`、`VIUSX`）裡有第三格 `grp-檔名解析`，文字逐字是「檔名解析」和「37 次 · $0.18」；截圖 `k5-d.png`、`k5-m.png` 已合併。-a 已合併。還沒好就停下來等。
1. **型別：** `subtitleSpendService.ts` 的 `SubtitleSpendSummary` 加 `parseCalls: number`、`parseUsd: number`，對應 snake_case 的 `parse_calls`、`parse_usd`。後端沒回這兩個欄位時（舊版 API）當成 0。
2. **K5 第三格：** 照上面的裁定。`parseCalls > 0` 才畫；文字「檔名解析」＋「N 次 · $x」，金額格式用卡片現有的 `usd()`；不進「依模型」。
3. **卡片出現條件：** 卡片現在的條件（`SpendSection.tsx` 裡「這個月有沒有值得畫的東西」那個函式）加上 `parseCalls > 0`。
4. **首頁總額：** `monthSpendUsd` 把 `parseUsd` 加進去。原本「只有測到東西才顯示金額」的規則照舊：只有 `parseCalls > 0` 才算「有測到」，0 次不能讓首頁從不顯示變成顯示 $0。
5. **測試（先寫、先看它紅）：**
   - `SpendSection.spec.tsx`：三種情況：只有解析、解析＋字幕、解析 0 次時不畫。
   - `HomeReadoutBand.spec.tsx`：總額有加上解析；0 次時行為不變。
   - `subtitleSpendService.spec.ts`：欄位對應，舊版 API 沒有欄位時當 0。
6. **gallery 夾具與視覺基準：** `-gallery.fixtures.tsx` 的 K5 情境，至少一個要帶 `parse_calls: 37`、`parse_usd: 0.18`，對齊設計稿。另加一個「只有檔名解析」的情境。
   - 有變的 darwin 基準：本機刪掉後跑 `update-missing` 重產。
   - 對應的 `-linux` 基準 `git rm`，交給 CI bootstrap。
   - 其他重產雜訊照慣例還原。
7. **UX 驗收：** 1440 截圖對照 `k5-d.png`、390 截圖對照 `k5-m.png`，附在 PR 上。
8. **全綠：** `pnpm nx run web:test`、`web:typecheck`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`、`scripts/check-design-refs.py`。

## Tasks / Subtasks

- [x] T0：確認設計稿、截圖、-a 都好了（AC #0）
- [x] T1：先寫測試，看它紅（AC #5）
- [x] T2：型別、K5 第三格、卡片出現條件、首頁總額（AC #1–#4）
- [x] T3：夾具與基準（AC #6）、UX 截圖（AC #7）
- [x] T4：全綠（AC #8），開 PR，合併

## Dev Notes

- K5 頭條的兩格在 `SpendSection.tsx:95-105`（`amount={summary.translatedUsd}`／`asrUsd`）。新的一格照同一個子元件寫。
- `monthSpendUsd` 在 `HomeReadoutBand.tsx:106-109`：`sumUsd([translatedUsd, asrUsd, unroutedUsd])`。
- 設定頁 hint「免費方案每分鐘 5 次」過時的問題**不在**本單（屬於 C 流程設計稿）。

### 設計稿 / Design ref

- K5-D-v2 `xgYKA`：卡片 `m9wRx`、`row-用量` `hbiao`、範本格 `grp-語音辨識` `ie9uz`、新格 `grp-檔名解析` `n64lI`。
- K5-M-v2 `ptNai`：卡片 `D847yg`、`row-用量` `VIUSX`（直排）、範本格 `grp-語音辨識` `jcUSq`、新格 `grp-檔名解析` `Z4ZUhQ`。

### Pen 提示詞（Alexyu 貼給 Pen 的 Inline AI Agent）

```
在 ux-design.pen 裡，只在兩個畫面的「本月 AI 花費」卡片加一格，不改其他東西。

1. K5-D-v2（id xgYKA）：在 row-用量（id hbiao）裡，複製 grp-語音辨識（id ie9uz）一份，放在它右邊（變成第三個子節點），改名 grp-檔名解析。裡面的兩段文字改成：標籤「檔名解析」、數值「37 次 · $0.18」。
2. K5-M-v2（id ptNai）：在 row-用量（id VIUSX，直排）裡，複製 grp-語音辨識（id jcUSq）一份，放在它下面（第三個子節點），改名 grp-檔名解析，文字一樣改成「檔名解析」「37 次 · $0.18」。
3. 新格子的字級、顏色、間距全部照抄被複製的那格，不要改。如果工具一定要寫樣式值，只能用檔案裡現有的 $ 變數，數值 0 一律寫 $Space/none，不要寫裸數字或色碼。
4. 不要動 row-節省、blk-依模型、標題「本月 AI 花費」和「10 月」。
5. 改完自己檢查：兩個畫面的 row-用量 都有三格，順序是 翻譯、語音辨識、檔名解析；文字沒有超出卡片；桌機三格在同一行。

不要切換分支、不要 git commit、不要開 PR；改完存檔（⌘S 或選單 File → Save），回報新節點的 id。
```

存檔後：Sally 用 Pencil MCP 唯讀 `Get` 逐字複審，再出 `flow-k-activity-v2/k5-d.png`、`k5-m.png` 兩張截圖，用 plumbing commit，**只 stage 這兩張**、`pen-node-ids.json` 和 `ux-design.pen`。截圖 PR 合併、-a 也合併後，本單改成 `ready-for-dev`。

### References

- [Source: apps/web/src/components/activity/SpendSection.tsx:1-40、:80-105]
- [Source: apps/web/src/components/homepage/HomeReadoutBand.tsx:106-109、:276-277]
- [Source: apps/web/src/services/subtitleSpendService.ts:50-60]
- [Source: _bmad-output/screenshots/flow-k-activity-v2/k5-d.png]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-21 | 建單（Bob）。Sally 裁定：K5 頭條加第三格「檔名解析」，有呼叫才顯示；不進「依模型」；首頁總額算進去。等設計稿和 -a。 |
| 2026-10-22 | dev（Amelia）→ done：K5 第三格、卡片出現條件、首頁總額、服務層預設 0；夾具 `parse-only`；PR #892。 |
| 2026-10-22 | 解除 blocked → ready-for-dev：-a 已合併（#890）；Alexyu 跑完 Pen，K5-D-v2 加 `grp-檔名解析` `n64lI`（`lab-檔名解析` `P0zNh7`、`val-檔名解析` `lVds7`），K5-M-v2 加 `Z4ZUhQ`（`b8R4Jv`、`U4t0w`）；Sally MCP 唯讀複審字與樣式 token 跟語音辨識那格一致。截圖 `k5-d.png` 隨設計稿 PR 提交；`k5-m.png` 不變（手機稿卡片本來就在摺線下，畫面外）。 |

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia, BMAD dev）

### Debug Log References

- AC #0：Pencil MCP 唯讀 `Get`：`hbiao`（K5-D-v2，橫排）與 `VIUSX`（K5-M-v2，`layout: vertical`）的子節點依序是 grp-翻譯、grp-語音辨識、grp-檔名解析（`n64lI`／`Z4ZUhQ`），文字逐字「檔名解析」「37 次 · $0.18」。-a 已合併（#890），Go 端欄位名 `parse_calls`／`parse_usd`（`subtitle_spend_service.go:59-60`，非 omitempty）。
- TDD：先寫測試，三支 spec 跑出 **7 failed / 56 passed**（紅）；實作後 **63 passed**（綠）。
- 全綠：`web:test` 298 檔／4878 測試全過；`web:typecheck` 過；`lint:all`「✖ 170 problems (0 errors, 170 warnings)」（警告都是既有的）；`format:check` 過；`check-design-refs.py` 400 個檔頭節點都在。
- 視覺：本機 `AI_PROVIDER=claude` 跑 `test:visual:update-missing` 重產兩張 darwin、新增一張，之後 `test:visual` 1 passed，沒有其他重產雜訊。

### Completion Notes List

- **服務層：** `SubtitleSpendSummary` 加 `parseCalls`／`parseUsd`；`getMonthSummary` 在舊版 API 沒回這兩個欄位時補 0（E2E `activity-spend.spec.ts` 的 stub 正好就是舊形狀，順便覆蓋到）。
- **K5 第三格：** 直接重用 `LaneStat`（跟「語音辨識」同一個子元件，字級／顏色／無障礙語意 `dt`／`dd` 都一樣），`parseCalls > 0` 才畫；`data-testid="activity-spend-parse"`；不進「依模型」。
- **卡片出現條件：** `spendHasContent` 加 `parseCalls > 0`，只有解析的月份也會出卡片。
- **首頁總額：** `monthSpendUsd` 把 `parseCalls` 算進「有沒有紀錄」、`parseUsd` 加進 `sumUsd`；0 次照舊回 null（不會從不顯示變成 $0）。
- **「只有這一格」的解讀：** Sally 說只有解析花錢時「卡片也要出現（只有這一格）」，同一段又說翻譯／語音辨識「一定會出現」、沒紀錄要寫「—」。實作採後者：卡片照常顯示翻譯「—」、語音辨識「—」，再加檔名解析那格（夾具 `parse-only`）。如果要把兩條字幕線藏起來，是另一個設計決定。
- **UX 對照：** 1440／390 實機截圖（gallery 夾具＋ `/activity` 實頁 stub API）。第三格的順序、文字、樣式跟語音辨識那格一致；手機直排在第三行，對照 `VIUSX` 節點結構。既有差異（非本單）：設計稿標籤和數值同一行、實作是標籤在上；設計稿寫「快取省下」、實作是「快取估算省下」——兩者都是 sub-7-6c 就有的。

### File List

- apps/web/src/services/subtitleSpendService.ts
- apps/web/src/services/subtitleSpendService.spec.ts
- apps/web/src/components/activity/SpendSection.tsx
- apps/web/src/components/activity/SpendSection.spec.tsx
- apps/web/src/components/activity/ActivityHub.spec.tsx
- apps/web/src/components/homepage/HomeReadoutBand.tsx
- apps/web/src/components/homepage/HomeReadoutBand.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/month-end/default-visual-darwin.png（重產）
- tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/month-end-mobile/default-visual-darwin.png（重產）
- tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/parse-only/default-visual-darwin.png（新增）
- tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/month-end/default-visual-linux.png（git rm，交給 CI bootstrap）
- tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/month-end-mobile/default-visual-linux.png（git rm，交給 CI bootstrap）
- _bmad-output/implementation-artifacts/backlog-ai-parse-reads-env-keys-only-b.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
