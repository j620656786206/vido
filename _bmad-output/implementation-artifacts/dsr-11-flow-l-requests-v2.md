# Story DSR.11: Flow L 請求系統——對齊 L1–L8，但不准刪掉超前的稿

Status: review

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
| 1 | `searching` 狀態色 | 赭 | 赭 | ✅ **已裁定→泥金**，兩邊都改完 |
| 2 | 狀態點的 token 形式 | — | 3 個用飽和色、2 個用 `*-text` | ✅ **五個統一成飽和階** |
| 3 | L4-M 的整條 meta 行（類型 · 日期） | **刻意隱藏**（`metaRow` `enabled:false`） | 不分斷點都渲染 | ⚠️ **待裁定**，見下 |
| 4 | ~~L4-M 的日期全是同一天~~ | — | — | ❌ **SM 誤判，已撤銷**（那些節點在手機上根本不渲染） |
| 5 | ~~L4-M 失敗列沒有錯誤訊息行~~ | — | — | ❌ **SM 誤判，已撤銷**（同上，meta 行整條隱藏） |
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

2. ✅ **`searching` 改成泥金——⚖️ Alexyu 2026-09-11 裁定「改成金色」，已執行。**

   ⚠️ **dev-story 的 AC-drift 檢查發現這會推翻一個有記錄的設計決定。** `13-0-requests-design.md:111` 白紙黑字寫著：**「Decision: `searching`→`warning-tint`／「搜尋中」(transient-work family) — added to DL-v2 §2.5 + new §8 pipeline section」**，`13-1a:113` 又把它複述成契約。那個映射目前活在五個地方：兩張 story 檔、`.pen` 的 Design Language v2 §8 註解與色票、`RequestRow.tsx`、L1-D／L4-M、以及 Flow J 的 J3-D 規格稿。而且它有同族成員——`libraryStatus.ts:61` 的「整理中」用的是同一個 `TINT.warning`，那顆徽章出現在整個媒體庫。

   換句話說，這不是 Flow L 的小改，是「赭色要不要涵蓋『暫態處理中』」這個跨全站的詞彙裁定。dsr-11 把證據備齊、沒有自行翻案，**Alexyu 裁定「改成金色」之後才動手**。

   **實際改動範圍（七處）**：`RequestRow.tsx` 的 `STATUS_TOKENS.searching` 與整組 `DOT_BG`、`libraryStatus.ts:61` 的「整理中」、`.pen` DL-v2 §8 的註解 (`J3o8xi`) 與五組色票、L1-D 五列、L4-M 五列、以及跟著裁定走的 `HeroBanner.spec.tsx` 斷言。J3-D 只有文字說明、沒有色票，不需要動。

   **支持改成泥金的一方**：DESIGN.md §固定詞彙 —— 赭＝「你要求了，但它**沒發生**」，泥金＝「正在跑（真的有工作在進行）」。「搜尋中」就是正在發生。同一節還寫著「**赭色的價值來自它零誤報，不是來自它涵蓋得廣**」，而「暫態處理中家族」正是「涵蓋得廣」。這一節是 2026-09-10／11 的裁定，比 13-0 晚兩個月。

   **支持維持赭色的一方**：13-0 的理由是「暫態處理中」自成一族，與「整理中」同族——那是一套自洽的分類，只是沒有被寫進 DESIGN.md。而且改成泥金之後 `searching` 與 `downloading` 會同色（兩者都是「正在跑」），區別只剩標籤與百分比。

2b. ⏸️ **L4-M 的 meta 行——本 story 不動，等裁定。** L4-M 五列的 `metaRow`（類型 · 日期）都是 `enabled:false`，手機上**整條不渲染**；`RequestRow.tsx` 沒有任何斷點條件，手機上照樣渲染。兩邊差一整行資訊。稿的選擇在 390px 上說得通（標題＋狀態藥丸＋百分比已經很擠），但沒有任何註記說明那是刻意的。

3. ✅ **狀態點統一用飽和階，已執行。** `DOT_BG` 目前 `pending`/`searching`/`completed` 用 `--info`/`--warning`/`--success`（飽和），但 `downloading` 用 `--accent-text`、`failed` 用 `--error-text`（文字階）。依 DESIGN.md §兩種金規則「`--accent-primary` 給人按，`--accent-text` 給人讀」，一個實心圓點是填色不是文字。改成 `downloading: --accent-primary`、`failed: --error`。

4. ❌ **撤銷。** SM 建單時用 `Get` 收集文字，沒有沿著父節點檢查 `enabled`，於是把 L4-M 五列**已停用**的 meta 行讀成了「畫錯的假資料」。實際上那一整行在手機上不渲染，所以既沒有「類型欄全寫電影」也沒有「日期全同一天」的問題。真正的差異是 meta 行本身該不該在手機上出現——移到第 2b 條待裁定。
   📌 **教訓寫在這裡給後面的 dsr story**：從 `.pen` 讀畫面內容時，`enabled:false` 加在**父節點**上時子節點的 `enabled` 仍是 `undefined`。要判斷「使用者真的看得到什麼」，必須沿著 `ctx.parentCtx` 往上走一遍。

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

- [x] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [x] `READABLE_FLOWS` 加入 `"flow-l-requests-v2"`
  - [x] 跑匯出，只有 `flow-l-requests-v2/*.png` 這 8 張有 byte 差異
  - [x] 打開 l1-d-v2.png 與 l4-m-v2.png 確認讀得到字

- [x] **Task 2 — `searching` 改泥金（AC: #2, #3）** ⚖️ **Alexyu 裁定「改成金色」**
  - [x] 先寫紅的測試（`RequestRow.spec.tsx` ＋ `libraryStatus.spec.ts`，2 紅）再改
  - [x] `RequestRow.tsx`：`STATUS_TOKENS.searching` → `--accent-tint`／`--accent-text`
  - [x] `DOT_BG` 五個統一成飽和階：`searching`／`downloading` → `--accent-primary`、`failed` → `--error`
  - [x] `libraryStatus.ts:61` 的「整理中」→ `TINT.accent`（同族裁定，影響整個媒體庫的海報徽章）
  - [x] `HeroBanner.spec.tsx` 的 `toContain('warning')` 斷言跟著改成 `accent`
  - [x] `.pen` DL-v2 §8：註解重寫＋五組色票更新（順手修掉 downloading／failed 的圓點用文字階、completed 的標籤用飽和色）
  - [x] L1-D 五列、L4-M 五列的藥丸同步；全畫布掃過，沒有任何 `LkjRd` instance 還帶 warning

- [ ] **Task 3 — L4-M 的 meta 行（AC: #2b）** ⏸️ **BLOCKED，等 Alexyu 裁定**
  - [x] ~~對齊假資料~~ — SM 誤判，已撤銷（見 AC #4）
  - [ ] 裁定「手機不顯示」→ `RequestRow.tsx` 加斷點條件，並在 L4-M 加一則註記說明那是刻意的
  - [ ] 裁定「手機也顯示」→ L4-M 五列的 `metaRow` 改回 `enabled:true`

- [x] **Task 4 — 兩行說明補進程式碼（AC: #6, #7）**
  - [x] 先改 `RequestsView.spec.tsx` 的斷言讓它變紅（2 紅）
  - [x] 空狀態補「從探索或詳情頁按「想要」開始追蹤」
  - [x] fail-soft 補「請求服務暫時無法連線」

- [x] **Task 5 — 檔頭與字級（AC: #5, #8, #9）**
  - [x] 兩個檔頭補齊畫面清單
  - [x] `RequestButton.tsx` 2 處 `text-[13px]` → `text-xs`
  - [x] L1-D 的計數補「筆」
  - [x] `npx eslint apps/web/src/components/requests/` → 0 errors

- [x] **Task 6 — 加註超前的稿（AC: #10）**
  - [x] Flow L 群組加了 `L Flow Note`：取消・重試對應 13-7a／13-7b，季集樹對應 13-2b，皆 ready-for-dev，不是漂移、不得刪
  - [x] Flow L 溢出從 5 降到 **0**（順手修掉 L4-M 底部分頁列的垂直裁切，見 Discovery Triage ①）

- [x] **Task 7 — 收尾驗證（AC: #11, #12）**
  - [x] `pnpm run lint` → 0 errors（127 warnings，與改動前同數）
  - [x] `pnpm nx run web:typecheck --skip-nx-cache` → PASS
  - [x] `pnpm run format:check` → 通過
  - [x] `python3 scripts/check-design-tokens.py` → 一致
  - [x] `pnpm nx test web` 3346/3346、`pnpm nx test api` PASS
  - [x] 重跑匯出，只有 Flow L 的 8 張有 byte 差異

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

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-11

### Debug Log References

- `pnpm nx test web`（RED）：2 failed —— 兩行新文案的斷言如預期找不到元素
- `pnpm nx test web`（裁定前 GREEN）：3346 / 3346
- `pnpm nx test web`（裁定後 RED 2 → GREEN）：**3347 / 3347 passed**（+1 新測試：searching 穿泥金）
- `pnpm nx test api`：PASS
- `pnpm run lint`：**0 errors**、127 warnings（與改動前同數）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `python3 scripts/check-design-tokens.py`：一致（188 畫面／72 母版／clipping 從 72 降到 **67**）
- 匯出：188 張，**只有 Flow L 的 8 張**有 byte 差異

### Completion Notes List

- 🔗 **AC Drift: FOUND（兩條，都擋下了）**
  - **① `searching` 的顏色。** `13-0-requests-design.md:111` 記著「**Decision: `searching`→`warning-tint`／「搜尋中」(transient-work family) — added to DL-v2 §2.5 + new §8 pipeline section**」，`13-1a:113` 複述成契約。那個映射活在五處：兩張 story 檔、`.pen` DL-v2 §8 的註解 (`J3o8xi`) 與色票 (`R96BLI`/`Q6ac5`/`W4F63V`)、`RequestRow.tsx`、L1-D／L4-M、Flow J 的 J3-D。而且它有同族成員——`libraryStatus.ts:61` 的「整理中」用同一個 `TINT.warning`，那顆徽章出現在整個媒體庫。**這是跨全站的詞彙裁定，不是 Flow L 的小改**，因此 Task 2／3 停在這裡等 Alexyu。
  - **② L4-M 的 meta 行。** 建單時判定是「畫錯的假資料」，實際是 `metaRow` 整條 `enabled:false`、手機上不渲染。程式碼沒有斷點條件，手機上照樣渲染。兩邊差一整行資訊，但哪一邊對沒有記錄可循，一併等裁定。
- 📎 **Contract Stamps: NONE**（本 story 與引用的 `13-0`／`13-1a` 都沒有 `[@contract-v*]` 標記；`13-7a`／`13-7b` 有 STALE 標記但那屬它們自己的範圍）。
- 🎭 **A11y Pre-Flight: PASS**（3 個 component：`RequestsView.tsx`／`RequestButton.tsx`／`RequestRow.tsx`，jsx-a11y warning 0，本 story 引入 0）。四類回歸項：沒有新增圖片、沒有新增 aria-modal、沒有新增自訂 widget；新增的兩行是純文字，掛在既有的 `role="alert"`／空狀態容器內，既有的 `aria-live="polite"` 與 `role="progressbar"` 原封不動。
- ⚠️ **Pre-existing failure**：darwin 視覺那 3 個既有紅沿用 `preexisting-fail-visual-darwin-three-stale-baselines`，本 story 沒有新增視覺夾具。
- ❌ **SM 誤判兩處，已在 AC 中撤銷並留下教訓**：從 `.pen` 讀內容時，`enabled:false` 加在**父節點**上時子節點的 `enabled` 仍是 `undefined`。要判斷使用者真的看得到什麼，必須沿 `ctx.parentCtx` 往上走。建單時沒走，於是把五列隱藏的 meta 行讀成了畫錯的假資料。**後面的 dsr story 請沿用這個檢查。**
- 📝 **另外更正立案時的三處描述**（已同步回 sprint-status）：① 「RequestButton 有 raw shadow」是 grep 誤判——那是 token 形式，DESIGN.md §Buttons L632 明文規定 Primary 就帶 `--shadow-sm`；② 「6 檔只有 2 檔有標頭」不準，3 個 spec 豁免、3 個非 spec 全有，問題是列得不全；③ 「RequestRow 用 --warning-tint，要確認它落在『你要求了但沒發生』」——確認結果是**不落在**，那是 `searching`。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，一項，lane ③：
    - **③ DESIGN.md 的 §Buttons 與 §Cards 互相矛盾。** §Buttons 說 Primary／Secondary 帶 `--shadow-sm`；§Cards 在 2026-09-11 改寫後說「陰影保留給真的浮在頁面之上的東西（Dialog／Sheet／Popover／Toast）」，按鈕不在清單裡。兩條都是現行正典，實作者無從判斷。屬 DESIGN.md 層級的裁定，**不在 Flow L 範圍**。**待立案：若 dev 執行時尚無此條目，開 `disc-2026-09-button-shadow-vs-card-rule: backlog` 並在此列回填 ID，同時在 `dsr-9-flow-j-specs-and-design-system` 的條目註明由它承接。**
  - 實作中另外撿到一項，走 lane **① expand-scope-in-place**：
    - **① L4-M 底部分頁列垂直裁切。** 五個分頁 62px 高，塞在 80px 高、下內距 24px 的列裡（可用高度 56px），五個標籤全部 `partially clipped`。下內距改 16px 即解。由 AC #10 的「Flow L 溢出必須 0」追蹤。
- Reference: `project-context.md` Rule 24

### File List

**修改（程式碼）：**
- `apps/web/src/components/requests/RequestsView.tsx` — 補兩行說明（空狀態、fail-soft）＋ Rule 21 檔頭補齊四張稿
- `apps/web/src/components/requests/RequestButton.tsx` — Rule 21 檔頭補 L8 ＋ 2 處字級
- `apps/web/src/components/requests/RequestsView.spec.tsx` — 2 條新斷言

**修改（設計與腳本）：**
- `ux-design.pen` — L1-D 計數補「筆」；Flow L 群組新增 `L Flow Note`；L4-M 底部分頁列下內距 24→16（修掉 5 個垂直裁切）
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` 加入 `flow-l-requests-v2`
- `_bmad-output/pen-tokens.json` — 快照同步（clipping 72 → 67）
- `_bmad-output/screenshots/flow-l-requests-v2/*.png`（8 張）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-11 | Task 1 — `flow-l-requests-v2` 進 `READABLE_FLOWS`，8 張稿從 400px 改為 2x／最小寬 1400。 |
| 2026-09-11 | **AC-drift 擋下兩條**：`searching` 的顏色會推翻 13-0 的記錄決定且牽動全站的「整理中」；L4-M 的 meta 行是刻意隱藏不是假資料錯誤。Task 2／3 改為 BLOCKED。 |
| 2026-09-11 | Task 4 — 空狀態補「從探索或詳情頁按「想要」開始追蹤」、fail-soft 補「請求服務暫時無法連線」（先紅 2 後綠）。 |
| 2026-09-11 | Task 5 — 兩個檔頭補齊真正實作的畫面；2 處 `text-[13px]` → `text-xs`；L1-D 計數補「筆」。 |
| 2026-09-11 | Task 6 — Flow L 群組加 `L Flow Note` 標明三處超前實作的稿不得刪；順手修掉 L4-M 底部分頁列的垂直裁切，Flow L 溢出 5 → 0。 |
| 2026-09-11 | Task 7 — 閘門：web 3346/3346、api PASS、lint 0 errors、typecheck PASS、token 一致、prettier 通過。 |
| 2026-09-11 | ⚖️ **Alexyu 裁定「改成金色」** → Task 2／3 解除阻塞並執行：`searching` 與同族的「整理中」從赭改泥金，`DOT_BG` 五個統一成飽和階，`.pen` DL-v2 §8 的註解與色票重寫，L1-D／L4-M 十列藥丸同步，`HeroBanner.spec.tsx` 斷言跟上。先紅 2 後綠 3347。 |
| 2026-09-11 | 順手修掉裁定時撞見的三處同類問題：DL-v2 §8 的 downloading／failed 圓點用的是「給人讀」的文字階、completed 的標籤用飽和色當字（§Badges 明寫底用 tint、字用 `*-text`）。 |
