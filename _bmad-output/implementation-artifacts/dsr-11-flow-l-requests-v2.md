# Story DSR.11: Flow L 請求系統——對齊 L1–L8，但不准刪掉超前的稿

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As someone tracking what they asked the NAS to go find,
I want 請求清單的狀態顏色說的是真話，而且手機版的稿跟桌機版是同一份假資料,
so that 「搜尋中」不會用一個意思是「你要求了但它沒發生」的顏色。

## Context

`epic-dsr` 的第三張（dsr-12、dsr-10 已 done）。`components/requests/` 6 個檔，扣掉 3 個 spec 是 3 個。

⛔ **這張最重要的一條在最前面：Flow L 有三處「稿有、碼沒有」，但那不是漂移，是設計超前實作。不准刪。**

| 稿上畫的 | 對應的 story | 狀態 |
| --- | --- | --- |
| L1-D／L4-M 每一列的「取消」、失敗列的「重試」 | `13-7a-request-cancel-retry` · `13-7b-request-cancel-retry` | **ready-for-dev** |
| L3-D 的「選擇要請求的內容」季集樹 | `13-2b-partial-request` | **ready-for-dev** |

後端目前只有 `GET /requests`、`POST /requests`、`GET /requests/tv/:tmdb_id/coverage`（`request_handler.go:42-44`）——沒有 DELETE、沒有 retry，`RequestRow` 也沒有渲染任何動作按鈕。**但那是因為那兩張單子還沒開跑，不是因為稿畫錯。** 一個只比對「稿 vs 碼」的人會把它們當成幻覺刪掉，那會刪掉 13-0 已經驗收過的設計。

⚠️ **驗收基準是 `.pen` 節點值。** Flow L 的 8 張稿現在是 400px 縮圖，讀不到字。Task 1 先修。

## Acceptance Criteria

### 設計稿節點（逐字抄，不要重查）

`L1-D-v2 (K7fiy)` · `L2-D-v2 (VH3Tq)` · `L3-D-v2 (He04g)` · `L4-M-v2 (n7isVa)` · `L5-D-v2 骨架 (ER39x)` · `L6-D-v2 空狀態 (x4CNb)` · `L7-D-v2 fail-soft (oopme)` · `L8-D-v2 (G0xib)` · `Component/RequestRow-v2 (LkjRd)` · `Component/Button/Primary (otvKh)`

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | `searching` 狀態色 | 赭（`--warning-tint`／`--warning-text`） | 同左 | **碼＋稿都要改** → 泥金 |
| 2 | 狀態點的 token 形式 | — | 3 個用飽和色、2 個用 `*-text` | **碼→稿** |
| 3 | L4-M 的類型欄 | 五列全寫「電影」 | 由 `request.mediaType` 決定，不會錯 | **稿→碼** |
| 4 | L4-M 的日期 | 五列全是 `2026-06-28` | 由 `request.createdAt` 決定 | **稿→碼** |
| 5 | L4-M 失敗列 | 沒有錯誤訊息行 | failed 時渲染 `errorMessage` | **稿→碼** |
| 6 | L1-D 的計數 | `5` | `5 筆` | **稿→碼** |
| 7 | L6 空狀態說明行 | `從探索或詳情頁按「想要」開始追蹤` | 沒有這一行 | **碼→稿** |
| 8 | L7 fail-soft 副行 | `請求服務暫時無法連線` | 沒有這一行 | **碼→稿** |
| 9 | `RequestsView.tsx` 檔頭 | — | 只寫 L1-D-v2，但它同時實作 L5／L6／L7 | **碼→稿** |
| 10 | `RequestButton.tsx` 檔頭 | — | 只寫 L2-D-v2，但它的 toast 也畫在 L8-D | **碼→稿** |
| 11 | `RequestRow.tsx` 檔頭 | — | `Component/RequestRow-v2 (LkjRd)` **已驗證正確** | ✅ 不動 |
| 12 | 字級 | Body 14 / Label 12 | `RequestButton.tsx` 2 處 `text-[13px]` | **碼→稿** |
| 13 | L1-D 五種狀態的標籤 | 想要／搜尋中／下載中／失敗＋找不到可用來源／已入庫 | 逐字相同 | ✅ |
| 14 | L2-D 三態 | ＋想要／已請求 · 處理中／已入庫 | 逐字相同 | ✅ |
| 15 | L6 標題 / L7 主行 / L7 按鈕 | 尚無請求／無法載入請求狀態／重試 | 逐字相同 | ✅ |
| 16 | 取消・重試・季集樹 | 有 | 沒有 | ⛔ **不是漂移**，見 Context |

---

1. **Flow L 的稿要讀得到。** `READABLE_FLOWS` 加入 `"flow-l-requests-v2"`，重跑匯出，8 張改成 2x／最小寬 1400。**只 stage 這 8 張**。

2. **`searching` 從赭改成泥金——兩邊都改。** 這是本 story 唯一需要同時動程式碼與設計稿的一條。
   - 依據：DESIGN.md §固定詞彙 —— 赭＝**「你要求了，但它沒發生」**，泥金＝**「正在跑（真的有工作在進行）」**。「搜尋中」就是正在發生。DESIGN.md 對赭的註解寫著「**赭色的價值來自它零誤報**」，用在一個確實在跑的狀態上就是誤報。
   - 改 `RequestRow.tsx` 的 `STATUS_TOKENS.searching`：`--warning-tint`／`--warning-text` → `--accent-tint`／`--accent-text`；`DOT_BG.searching`：`--warning` → `--accent-primary`。
   - 改 L1-D (K7fiy)、L4-M (n7isVa) 的「搜尋中」藥丸同步換色。
   - ⚠️ 改完之後 `searching` 與 `downloading` 同為泥金。**那是對的**，兩者都是「正在跑」；區別由標籤與 `downloading` 才有的百分比承擔，不靠顏色。**不要**為了讓它們不同而發明第六個顏色——DESIGN.md 明寫色相上已無處可放。

3. **狀態點統一用飽和色。** `DOT_BG` 目前 `pending`/`searching`/`completed` 用 `--info`/`--warning`/`--success`（飽和），但 `downloading` 用 `--accent-text`、`failed` 用 `--error-text`（文字階）。依 DESIGN.md §兩種金規則「`--accent-primary` 給人按，`--accent-text` 給人讀」，一個實心圓點是填色不是文字。改成 `downloading: --accent-primary`、`failed: --error`。

4. **L4-M 的假資料改成跟 L1-D 同一份。** 三件事：① 「熊家餐館 S3」與「幕府將軍 S1」的類型欄從「電影」改成「影集」（它們是影集，S3／S1 就寫在標題裡，而 L1-D 寫的就是「影集」）；② 五列的日期從全部 `2026-06-28` 改成 L1-D 的五個不同日期；③ 失敗列補上錯誤訊息行「找不到可用來源」，`--error-text`，對上 `RequestRow.tsx:80-82`。

5. **L1-D 的計數補「筆」。** 程式碼是 `<span class="font-mono">{n}</span> 筆`。稿只有數字。

6. **L6 空狀態補說明行。** 稿有「從探索或詳情頁按「想要」開始追蹤」，`RequestsView.tsx` 的空狀態只有「尚無請求」＋「前往探索」。**稿的版本告訴人怎麼開始，程式碼的沒有**——補進程式碼，用 `--text-muted`、Label 12，放在「尚無請求」與按鈕之間。

7. **L7 fail-soft 補副行。** 稿有「請求服務暫時無法連線」，程式碼只有「無法載入請求狀態」＋「重試」。補進程式碼，同樣 `--text-muted`、Label 12。這一行說的是**為什麼**，主行說的是**發生了什麼**，兩者不重複。

8. **兩個檔頭補齊它們真正實作的畫面。**
   - `RequestsView.tsx` → `// Design ref: ux-design.pen Screen L1-D-v2 (K7fiy) · L5-D-v2 (ER39x) · L6-D-v2 (x4CNb) · L7-D-v2 (oopme)`
   - `RequestButton.tsx` → `// Design ref: ux-design.pen Screen L2-D-v2 (VH3Tq) · L8-D-v2 (G0xib)`
   - `RequestRow.tsx` **不動**（`LkjRd` 已驗證存在且 reusable）
   🚨 那一行必須以 `(節點ID)` 結尾，說明另起一行。

9. **2 處 `text-[13px]` 收斂。** `RequestButton.tsx:92`（已入庫藥丸）與 `:115`（已請求藥丸）。DESIGN.md §Badges and Pills 明寫「**12px 標籤字**」，所以兩處都是 `text-xs`。順帶檢查那兩顆藥丸的 `px-4 py-2.5` 是否與 §Badges 的「4px／10px 內距」相差太遠——**只記錄不修**，內距屬 dsr-9 的元件規格範圍。

10. **⛔ 不准刪超前的稿。** L1-D／L4-M 的「取消」「重試」與 L3-D 的整張季集樹**一律保留**。改為在 Flow L 群組加一則 Note 說明它們對應 `13-7a`／`13-7b`／`13-2b`（都是 `ready-for-dev`），讓下一個做比對的人一眼看到，不會再誤判。

11. **既有測試全數保留通過。** `RequestRow.spec.tsx`／`RequestButton.spec.tsx`／`RequestsView.spec.tsx` 的斷言只有第 2、3、6、7 條涉及的可以改。

12. **CI 全綠。** lint 0 errors、typecheck、format:check、check-design-tokens、`nx test web`、`nx test api`。

## Tasks / Subtasks

- [ ] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [ ] `READABLE_FLOWS` 加入 `"flow-l-requests-v2"`
  - [ ] 跑匯出，只 stage `flow-l-requests-v2/*.png` 這 8 張，其餘 `git checkout`
  - [ ] 打開 l1-d-v2.png 與 l4-m-v2.png 確認讀得到字

- [ ] **Task 2 — `searching` 改泥金（AC: #2, #3）**
  - [ ] 先改 `RequestRow.spec.tsx` 的斷言讓它變紅，再改 `STATUS_TOKENS` 與 `DOT_BG`
  - [ ] L1-D (K7fiy)、L4-M (n7isVa) 的「搜尋中」藥丸同步換色
  - [ ] 確認 `searching` 與 `downloading` 同色是刻意的，並在 `RequestRow.tsx` 留一行註解說明為什麼

- [ ] **Task 3 — L4-M 的假資料對齊 L1-D（AC: #4）**
  - [ ] 兩列類型欄「電影」→「影集」
  - [ ] 五列日期改成 L1-D 的 `2026-07-02` / `2026-07-01` / `2026-06-29` / `2026-06-25` / `2026-06-20`
  - [ ] 失敗列補「找不到可用來源」，`--error-text`
  - [ ] L1-D 計數補「筆」

- [ ] **Task 4 — 兩行說明補進程式碼（AC: #6, #7）**
  - [ ] 先改 `RequestsView.spec.tsx` 的斷言讓它變紅
  - [ ] 空狀態補「從探索或詳情頁按「想要」開始追蹤」
  - [ ] fail-soft 補「請求服務暫時無法連線」

- [ ] **Task 5 — 檔頭與字級（AC: #8, #9）**
  - [ ] 兩個檔頭補齊畫面清單
  - [ ] `RequestButton.tsx` 2 處 `text-[13px]` → `text-xs`
  - [ ] `npx eslint apps/web/src/components/requests/` → 0 errors

- [ ] **Task 6 — 加註超前的稿（AC: #10）**
  - [ ] Flow L 群組加一則 Note（照 M5 Note／M7 Note 的體例）：取消・重試對應 `13-7a`／`13-7b`，季集樹對應 `13-2b`，三者皆 `ready-for-dev`，**不是漂移、不得刪除**
  - [ ] Flow L 溢出檢查必須 0

- [ ] **Task 7 — 收尾驗證（AC: #11, #12）**
  - [ ] `pnpm run lint` → 0 errors
  - [ ] `pnpm nx run web:typecheck --skip-nx-cache`
  - [ ] `pnpm run format:check`
  - [ ] `python3 scripts/check-design-tokens.py`
  - [ ] `pnpm nx test web` 與 `pnpm nx test api`（⛔ 不用 `run_in_background`）
  - [ ] 重跑匯出，確認只有 Flow L 的 8 張有 byte 差異

## Dev Notes

### 這張的形狀

`dsr-12` 是碼追稿、`dsr-10` 是一半稿在說謊。**這張的主要工作是顏色語意**，而且有一條要兩邊同時改——那是前兩張都沒出現過的形狀。

另一個新形狀是**「稿有、碼沒有」不等於漂移**。Flow L 的稿在 `13-0-requests-design` 就一次畫完了整個 Epic 13，而 Epic 13 有三張單子還沒跑。這是刻意的：設計先行、實作分批。做 dsr 比對的人必須先查 sprint-status 再下結論。

### 固定詞彙的完整推導（不要憑印象改）

五個狀態逐一對照 DESIGN.md §固定詞彙：

| status | 標籤 | 現在的色 | 詞彙說什麼 | 判定 |
| --- | --- | --- | --- | --- |
| `pending` | 想要 | 靛青 | 純告知，不帶評價 | ✅ 對。剛排進佇列、系統會處理，用赭會是誤報 |
| `searching` | 搜尋中 | **赭** | 你要求了但它**沒發生** | ❌ **錯**。它正在發生 → 泥金 |
| `downloading` | 下載中 | 泥金 | 正在跑 | ✅ |
| `completed` | 已入庫 | 青碧 | 有答案了 | ✅ |
| `failed` | 失敗 | 硃砂 | 壞了 | ✅ |

改完之後 Flow L **沒有任何一處用赭色**。那是對的結果，不是遺漏——赭色的價值是零誤報。

### 不要做的事

- **不要刪 L3-D 整張稿**，也不要刪 L1／L4 的動作按鈕。見 Context 與 AC #10。
- **不要碰 `13-7a`／`13-7b`／`13-2b`**。那是三張獨立的 story，不在本 story 範圍。
- **不要改 `RequestButton.tsx` 的 `shadow-[var(--shadow-sm)]`。** 立案時寫「RequestButton 有 raw shadow-*」是我的 grep 誤判——那是 token 形式，而且 DESIGN.md §Buttons 第 632 行明文規定 Primary 就是「泥金填色、`--text-on-accent` 深墨字、`--shadow-sm`」。⚠️ 但 §Cards 在 2026-09-11 改寫後說「陰影保留給真的浮在頁面之上的東西（Dialog／Sheet／Popover／Toast）」，按鈕不在那個清單裡——**§Buttons 與 §Cards 現在互相矛盾**。那是 DESIGN.md 層級的裁定，屬 `dsr-9`，見下方 Discovery Triage。
- **不要碰底部那顆 `shadow-[var(--shadow-xl)]`**（`RequestButton.tsx:179`）。那是 fixed 定位的 toast，正是 §Cards 點名允許有陰影的四種之一。

### Source tree

```
apps/web/src/components/requests/RequestRow.tsx        ← Task 2（狀態色，主要）
apps/web/src/components/requests/RequestsView.tsx      ← Task 4, 5
apps/web/src/components/requests/RequestButton.tsx     ← Task 5
apps/web/src/components/requests/*.spec.tsx            ← Task 2, 4
scripts/export-pen-screenshots.py                      ← Task 1
ux-design.pen（K7fiy / n7isVa 與 Flow L 群組）          ← Task 2, 3, 6，只能用 Pencil MCP
apps/api/internal/handlers/request_handler.go          ← 只讀，確認端點清單
```

### Project Structure Notes

- `sprint-status.yaml` 的 `dsr-11` 條目有三處不準，本 story 一併更正：① 「RequestButton.tsx 有 raw shadow-*」是誤判（見上）；② 「6 檔只有 2 檔有 Design ref 標頭」——3 個 spec 本來就豁免，3 個非 spec **全都有**標頭，問題是其中 2 個**列得不全**；③ 「RequestRow.tsx 目前用 --warning-tint，要確認它落在『你要求了但沒發生』」——確認結果是**不落在**，那一格是 `searching`，它正在發生。
- `RequestsView` 掛在 `/discover?view=requests`，不是獨立 route（nav-ADR:630）。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads the wall clock?**
  - **NO** — `RequestRow` 顯示的是 `request.createdAt` 格式化後的**絕對日期**（`2026-07-02` 這種），不是相對時間，不讀 `Date.now()`。`RequestsView` 與 `RequestButton` 也沒有。
  - 本 story 不新增視覺夾具。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` L1/L2/L3/L4/L5/L6/L7/L8 與 `Component/RequestRow-v2 (LkjRd)`] — 內容用 Pencil MCP `{resolveInstances:true}` 展開後讀出
- [Source: DESIGN.md#固定詞彙規則（The Fixed Vocabulary Rule）] — AC #2 的依據，含「赭色的價值來自它零誤報」
- [Source: DESIGN.md#兩種金規則] — AC #3 的依據
- [Source: DESIGN.md#Badges and Pills] — AC #9 的 12px 依據
- [Source: DESIGN.md#Buttons L632] — Primary 的 `--shadow-sm` 是規定，不是漂移
- [Source: `apps/api/internal/handlers/request_handler.go:42-44`] — 端點只有三個，AC #16 的事實基礎
- [Source: `sprint-status.yaml` → `13-7a` / `13-7b` / `13-2b`（皆 `ready-for-dev`）] — Context 的事實基礎
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

（dev agent 填寫）

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，一項，lane ③：
    - **③ DESIGN.md 的 §Buttons 與 §Cards 互相矛盾。** §Buttons 說 Primary／Secondary 帶 `--shadow-sm`；§Cards 在 2026-09-11 改寫後說「陰影保留給真的浮在頁面之上的東西（Dialog／Sheet／Popover／Toast）」，按鈕不在清單裡。兩條都是現行正典，實作者無從判斷。屬 DESIGN.md 層級的裁定，**不在 Flow L 範圍**。**待立案：若 dev 執行時尚無此條目，開 `disc-2026-09-button-shadow-vs-card-rule: backlog` 並在此列回填 ID，同時在 `dsr-9-flow-j-specs-and-design-system` 的條目註明由它承接。**
- Reference: `project-context.md` Rule 24

### File List

（dev agent 填寫）
