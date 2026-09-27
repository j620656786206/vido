# Story bugfix：日巡模式下，字幕與請求流程的稿上「壓在劇照上的字」讀得到，而且以後自動擋下

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who designs and builds Vido in both 夜行 and 日巡,
I want every heading and label that sits on artwork to stay readable when the theme flips,
so that the design stays a trustworthy reference in daylight mode — and the checker catches the next frozen gradient before a human has to.

## Context

收尾 `disc-2026-09-text-on-artwork-flips-with-theme`（**P0**，2026-09-10 立，in-progress）：Flow H（dsr-7）與 Flow B（dsr-2）已收，條目寫「剩 Flow F／L／Docs」。一併處理它的機器守門版本 `disc-2026-09-pen-hardcoded-hex-unguarded`（P2）——兩者是同一件事。

### 🔴 查到的事（main `9c5b84b5`；2026-09-27 以 Pencil MCP 唯讀掃描）

1. **剩下的全是設計稿，程式碼已經乾淨。** 程式碼裡已沒有寫死的深色漸層：`grep` `bg-gradient-to-*` 只剩 `from-[var(--bg-primary)]` 與 `from-[var(--overlay-scrim)]` 兩種；壓在 `--overlay-scrim` 上的字一律 `--text-on-scrim`（`PosterCard`、`PosterCardV2`、`SearchResultCard`、`RecentMediaPanel`…）。Flow F 的對話框與 Flow L 的詳情頁背景 hero 由 `DetailHeroV2`／`TMDbDetailV2` 畫，dsr-2 已改成跟主題翻轉的 `--bg-primary` 漸層。
2. **設計稿上還有 15 個寫死 `#0c1512` 的 hero 漸層**（stops：`#0c151200` → `#0c1512F2`；`#0c1512` 正是 `--bg-primary` 的**夜行**字面值），每個上面壓 7–8 個會翻轉的字（`$text-primary`／`$text-secondary`／`$text-muted`／`$success-text`…）：
   - Flow F（11）：`ilZ2v`（F1-D-v2）、`tm41t`（F2-D-v2）、`h1sp9`（F3-D-v2）、`rDFVJ`（F4-D-v2）、`M3OcM3`（F5-D-v2）、`v4LyDh`（F6-D-v2）、`stW5J`（F7-D-v2）、`v2Bud`（F10-D-v2）、`MK4u5`（F1-M-v2）、`jZMbE`（F3-M-v2）、`FDYFo`（F6-M-v2）。
   - Flow L（4）：`mpt8b`＋`fqiLM`（L2-D-v2）、`xLs1Y`（L3-D-v2）、`mxo5r`（L8-D-v2）。
3. **Docs**：`Themes · 主題證據` 的 `Light · H1-D-v3 首頁` 裡，漸層資訊層 `k5jrJj` 仍寫死 `#0c1512`，上面壓 `$text-primary`／`$text-secondary`／`$text-muted`／`$success-text`——**正是 P0 當初的證據畫面，現在自己還是壞的**（dsr-7 修了 H1-D-v3 本體，沒修這張日巡副本）。
4. **其他寫死 `#0c1512` 的節點不在範圍**：`Q4F8F`（Design System 的示範 scrim，上面是 `$text-on-scrim`，本來就該兩主題都深）、`z6dCJ`（B9-D）、`bv5nA`／`l1bBs7`（E1-M 捲動淡出）、`YrGLh`（L2 海報 hover scrim）——**上面沒有會翻轉的字**，保持原樣。
5. **Flow B 的修法已驗證可行**（dsr-2，`sprint-status.yaml` P0 條目「✅ Flow B 已收」）：漸層改成 `#00000000 → $bg-primary`，字維持會翻轉的 token——日巡時漸層變淺、字變深，夜行時照舊。
6. **守門缺口**：`scripts/check-design-tokens.py` 只比對五份 token 定義是否一致，**不看 `.pen` 節點**；它讀的 `.pen` 資訊只有 `export-pen-screenshots.py` 經 MCP 產生的快照 `_bmad-output/pen-tokens.json`（`counts.clippingWarnings`、`masters`，`export-pen-screenshots.py:555-575`）。所以「值對、但凍結在單一主題」的漸層每個 flow 都得靠人眼撿。

### ⚖️ 建單裁定（2026-09-27，SM；Alexyu 可在 review 推翻）

1. **設計稿照 Flow B 的做法改**：16 個漸層（F 11＋L 4＋Docs 1）改成 `#00000000 → $bg-primary`，字的 token 不動。不新增 `text-on-scrim-secondary` 之類的 token（P0 條目的方案 a）——Flow B／H 已證明不需要。
2. **守門一起做（長解）**：快照多一個計數 `artworkTextFlips`——「會翻轉的文字 token，最近一層有填色的祖先是**寫死 hex 的漸層**或**圖片**」的文字節點數；`check-design-tokens.py` 在它 > 0 時失敗並列出節點。這條規則正好涵蓋 🔴 #4 的例外（那些漸層上沒有會翻轉的字），不需要白名單。
3. **程式碼不動**（🔴 #1）。

## Acceptance Criteria

1. **設計稿修正。**
   - 🔴 #2 的 15 個節點與 🔴 #3 的 `k5jrJj`：`fill` 改成與 dsr-2 相同的漸層（stop 0：`#00000000`；stop 1：`$bg-primary`，旋轉與位置沿用原節點）；上面的文字 token 不動。
   - 🔴 #4 的節點不動。
   - 驗證：對 F1-D-v2、F1-M-v2、L2-D-v2 與 `Light · H1-D-v3 首頁` 各截一次**日巡**畫面（`theme:{mode:"light"}` 暫時套在副本上或用 Themes 群組），片名與狀態列讀得到；截圖附在 Completion Notes（描述即可）。暫時的副本用完刪掉。
   - `problems`（裁切警告）不增加；選單 Save 存檔、grep 磁碟檔確認新 fill 落盤（`feedback_verify_pen_saved_before_commit`）。
2. **守門。**
   - `export-pen-screenshots.py` 的快照 JS 計算 `artworkTextFlips`（規則見裁定 #2；「會翻轉的文字 token」＝`$text-primary`／`$text-secondary`／`$text-muted`／`$text-disabled`／`$accent-text`／`$success-text`／`$warning-text`／`$error-text`／`$info-text`），寫進 `counts.artworkTextFlips`，並把違規節點（id、所在畫面名、文字 token）寫進快照的新欄位（例如 `artworkTextFlipNodes`，最多列 50 個）。
   - `check-design-tokens.py`：`artworkTextFlips > 0` → 失敗，印出違規節點與一句修法（「漸層改用 `$bg-*` 或把字改成 `$text-on-scrim`」）；舊快照沒有這欄 → 當 0（向下相容，但 Completion Notes 要記錄重跑後的值）。
   - 修法完成後重跑 `export-pen-screenshots.py`：`artworkTextFlips = 0`；**修之前**（或用 `git stash` 的 `.pen`）跑一次要得到 16 個節點上的文字數（證明規則抓得到）——數字寫進 Completion Notes。
   - `check-design-tokens.py` 的既有測試（若有，dev 先 grep）擴充一條「快照有違規節點 → 失敗」。
3. **截圖**：重跑匯出，只 stage 設計真的變了的 PNG（F 與 L 的相關畫面、`Themes` 那張若有匯出）＋ `pen-tokens.json`；其餘 re-render 雜訊 `git checkout`（CLAUDE.md 規則）。
4. **收尾**：`disc-2026-09-text-on-artwork-flips-with-theme` → done、`disc-2026-09-pen-hardcoded-hex-unguarded` → done（↪ 本張）；`DESIGN.md` 的設計 SOP 若有「深色漸層」相關段落，補一句「壓字的漸層只能用 `$bg-*`，否則 CI 擋」。
5. **CI**：`pnpm run lint:all`（含 format）、`python3 scripts/check-design-tokens.py` 綠；不影響任何前端或後端測試。

## Tasks / Subtasks

- [x] **Task 1 — 守門先紅：快照計數＋checker 規則（AC: #2）**——在未修的稿上跑，得到違規數
- [x] **Task 2 — 改 16 個漸層、日巡截圖驗證、存檔（AC: #1）**
- [x] **Task 3 — 重跑匯出（違規＝0）、只 stage 真變更的截圖、關兩張 disc、DESIGN.md（AC: #3, #4, #5）**

## Dev Notes

### 這張的重點

- **先做守門再改稿**：守門第一次跑就要把 16 個漸層上的字全抓出來——這就是它有用的證據。
- **不動程式碼**：程式碼早在 dsr-2／dsr-7 修好了。

### 上游契約（Rule 20）

- 不涉及。

### 不要做的事

- 不要新增 token（P0 條目的方案 a）。
- 不要動 🔴 #4 那些沒有翻轉文字的深色 scrim。
- 不要跑 `test:visual:update`（本張沒有前端變更）。

### 已知陷阱

- **Pencil schema**：漸層 fill 物件格式照既有節點複製（`Get(id).fill` 看 dsr-2 改過的 `XLwlb`），只換 `colors`；Color 只吃 hex，透明用 `#00000000`（`project_pen_schema_gotchas` #3）。
- **全域變數不跨 `execute`**（gotchas #8）：helper 與迴圈寫在同一個 snippet。
- **存檔**：先對目標節點重下一次同值 `Update` 標髒，再用選單 Save；grep 磁碟檔確認。
- **快照 JS 在 `export-pen-screenshots.py` 內是字串**：注意 Python 字串跳脫；`Get` 的 `ctx.parentCtx` 往上找「最近一層有 fill 的祖先」。
- **匯出非決定性**：只 stage 設計真的變了的 PNG。

### Source tree

```
ux-design.pen（16 個漸層節點）                                   ← Task 2
scripts/export-pen-screenshots.py（快照計數）、scripts/check-design-tokens.py（規則）  ← Task 1
_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/*、flow-l-requests-v2/*  ← Task 3
DESIGN.md（SOP 一句）、_bmad-output/implementation-artifacts/sprint-status.yaml       ← Task 3
```

### Cross-Stack Split Check

不涉及前後端程式碼 → 不拆。

### Time-dependent visual coverage

- N/A。

### References

- [Source: `sprint-status.yaml` → `disc-2026-09-text-on-artwork-flips-with-theme`（含 Flow B／H 已收的紀錄）、`disc-2026-09-pen-hardcoded-hex-unguarded`]
- [Source: `scripts/export-pen-screenshots.py:555-605`（快照）；`scripts/check-design-tokens.py:340-350`（讀快照 counts）]
- [Source: `apps/web/src/components/media/DetailHeroV2.tsx:55`（程式碼的正確做法）；`components/library/PosterCardV2.tsx:103-114, 164-165`（`--text-on-scrim` 的用法）]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（守門先紅）**：`export-pen-screenshots.py` 的快照 JS 新增 `artworkTextFlips`——會翻轉的文字 token（9 個）且「最近一層有填色的祖先」是圖片或含**非全透明寫死 hex 色標**的漸層（`#rrggbb00` 全透明不算，所以 Flow B 修過的 `#00000000 → $bg-primary` 不會被誤判）；違規節點（id、畫面名、token、壓在哪個節點上）寫進 `artworkTextFlipNodes`（最多 50）。`check-design-tokens.py` 新增 `check_pen_artwork_text`：> 0 就失敗並列前 20 個。**在未修的稿上跑整套匯出＋檢查：`111 個會隨主題翻轉的字壓在寫死顏色的漸層或圖片上`**，正好落在建單列的 16 個漸層（k5jrJj 3、F 各 7、L 7–8）。檢查器沒有既有測試檔（`scripts/` 下只有 shell 腳本），紅綠證據即這兩次實跑。
- **Task 2（改稿）**：16 個漸層逐一把色標換掉——全透明的換 `#00000000`、其餘換 `$bg-primary`，位置與旋轉不動（`k5jrJj` 方向相反，照原方向換）。字的 token 全部沒動。暫時複製 F1-D-v2、F1-M-v2、L2-D-v2 套 `theme:{mode:"light"}` 截圖，加上 `Light · H1-D-v3 首頁`：片名（你的名字、沙丘：第二部）與資訊列在日巡下都讀得到；副本已刪。裁切警告 66 不變。選單 Save 後比對磁碟：`#0c1512F2` 15→0、`#0c1512` 6→5（k5jrJj）、其他 5 個不在範圍的深色 scrim 原封不動。
- **Task 3**：重跑匯出（第一次 179／199 有一個 chunk 失敗，重跑 199／199）→ `artworkTextFlips: 0`、`check-design-tokens.py` 綠。只 stage 設計真的變了的 15 張：`flow-f-subtitle-v2` 11 張、`flow-l-requests-v2` 3 張（l2／l3／l8）、`design-system/light-h1-d-v3.png`；其餘 re-render 雜訊（component-anatomy、a1p-d、b12p-*、i*、j9-d）還原。`DESIGN.md` §4 補「壓字的漸層只能漸到 `$bg-*`，CI 會擋」。兩張 disc 改 done。
- 🔗 **AC Drift: N/A**（只動設計稿與檢查腳本，沒有任何已上線 AC 的行為改變）。
- 📎 **Contract Stamps: NONE**。
- 🎭 **A11y Pre-Flight: N/A（沒有 apps/web/ 檔案）**——本張本身就是對比度修正，證據是日巡截圖。
- 🎨 **UX Verification: PASS**（日巡四張截圖讀得到；夜行截圖只有漸層最深處 95%→100% 的像素差）。

### 🔍 /ship Adversarial Review（2026-09-27）

0 HIGH／0 MEDIUM／1 LOW（已修）：
- **L1** 規則只看漸層與圖片；會翻轉的字壓在**寫死的純色**底（例如 `#0c1512` 實心）上同樣會在日巡變深字壓深底 → 規則補上「非全透明的寫死純色」。以 MCP 對現稿實跑：0 個命中，快照數字不變（仍為 0），所以沒有重跑整套匯出。

### Discovery Triage

- N/A — no out-of-scope work discovered.

### File List

- `ux-design.pen`、`_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-f-subtitle-v2/{f1-d-v2,f1-m-v2,f10-d-v2,f2-d-v2,f3-d-v2,f3-m-v2,f4-d-v2,f5-d-v2,f6-d-v2,f6-m-v2,f7-d-v2}.png`
- `_bmad-output/screenshots/flow-l-requests-v2/{l2-d-v2,l3-d-v2,l8-d-v2}.png`、`_bmad-output/screenshots/design-system/light-h1-d-v3.png`
- `scripts/export-pen-screenshots.py`、`scripts/check-design-tokens.py`
- `DESIGN.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-27 | 建單（SM）：收尾 P0 `disc-2026-09-text-on-artwork-flips-with-theme`（剩 Flow F 11＋L 4＋Docs 1 個寫死 `#0c1512` 的漸層，程式碼已乾淨）＋併入守門 `disc-2026-09-pen-hardcoded-hex-unguarded` |
| 2026-09-27 | 實作完成：守門（111→0）、16 個漸層改 `$bg-primary`、日巡截圖確認、只 stage 15 張截圖；狀態 review |
| 2026-09-27 | /ship CR：守門規則補上寫死純色底 |
| 2026-09-27 | PR #553 合併（`ab4c9677`），狀態改 done |
