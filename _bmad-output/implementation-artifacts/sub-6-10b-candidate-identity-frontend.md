# Story 6.10b: 候選列身分 —— 封面、片長並列、未匹配標記（前端）

Status: review — 程式與測試交付（Task 1／2／4 的 spec 部分）；**Task 3（`.pen` + 截圖）與視覺基準線重烤需要你的機器**，見 Completion Notes

**Depends on:** sub-6-10a（`poster_path`、`runtime_source`、`tmdb_matched`、`display_title`）。

## Story

As a BYOK NAS owner scrolling the consent list,
I want each row to show its poster, its title and its route AND runtime side by side,
so that「片長未知」never erases the one line that told me why this row costs money.

## Acceptance Criteria

1. **封面。** `CandidateRow` 的佔位 span（`CandidateListPanel.tsx:86-89`）改為 `<img>`：`getImageUrl(posterPath,'w92')`（`lib/image.ts:19`）、`loading="lazy"`、`alt=""`（裝飾，標題已在旁）；無封面 → 維持既有灰底但**加一個片名首字**（不是空方塊）。
   ~~12px~~ → **Title 階 18px / 600 / `--text-secondary`**（Sally 裁定 3，2026-09-05）。

2. **副標不再互斥。** `routeSubtitle`（`:58-61`）改為兩段並列：「內嵌英文字幕 → 翻譯 · 2 小時 46 分」／「無文字字幕軌 → 語音辨識 + 翻譯 · 片長未知（估 45 分）」。`runtime_source=fallback` 時**金額**前保留 `≈`；`ffprobe`／`tmdb` 時不加。
   ~~副標寫 `≈ 45 分（片長未知）`~~ → **`片長未知（估 45 分）`**（Sally 裁定 1，2026-09-05：一列之內 `≈` 只能有一個意思）。

3. **未匹配標記。** `tmdb_matched=false` → 標題用 `display_title`，加中性灰標「未匹配」（tooltip：「TMDb 沒有比對到，片名由檔名解析」）；標題 hover 顯示原始檔名。
   ~~右側徽章區~~ → **身分欄內、緊接標題右側**（Sally 裁定 2，2026-09-05）。

4. **設計同步。** F15-D／F15-M（`pwMzT`／`fdu4y`）的列規格更新：封面實圖、兩段副標、未匹配標；依 `CLAUDE.md` 流程重出 `flow-f-subtitle-v2/f15-*` 截圖同 commit。
   核定稿與 Pencil 提示詞：**`sub-6-10b-f15-row-pen-prompt.md`**（Sally 2026-09-05）。

5. **測試。** specs：三種 `runtime_source` 的副標與 `≈`；封面有／無；未匹配標與 tooltip；gallery fixtures 更新（`-darwin` 本機、`-linux` 等 CI）。

## Tasks / Subtasks

- [x] **Task 1 — 型別（AC: #1-#3）** — 四個 optional 欄位已於 sub-6-10a 的 PR #386 隨契約 ack 一併加入 `subtitleService.ts`，本 story 直接消費。
- [x] **Task 2 — `CandidateRow` 改造（AC: #1, #2, #3）** — 封面 `<img>` + 首字 fallback、副標兩段並列、未匹配灰標 + tooltip + 標題 hover 顯示原始檔名。
- [ ] **Task 3 — 設計更新 + 截圖（AC: #4）** — ⛔ **需要 Pencil.app**，不在這個沙盒能做。提示詞已備妥（見下）。
- [x] **Task 4 — 測試（AC: #5 的 spec 部分）** — 9 個新測試 + 1 個既有測試改寫到新契約。⛔ **gallery fixture 的視覺基準線重烤需要你的機器與 CI**（見下）。

## Dev Notes

- **Inherited from sub-6-1:** F15 rows now have an「資料夾無法寫入」state (disabled checkbox + error badge with the blocker as tooltip). The `.pen` F15-D/M update in AC #4 should include this state; the shipped idiom is the route badge in error tint.

- 舊伺服器（無新欄位）→ 行為與今日相同（fallback 分支），不得壞。
- Rule 21 header 已是 F15-D/M/F18 併列，不變。
- 12px 底線（DESIGN.md）：「未匹配」標為 12px（Label 階）。首字佔位改為 18px（Title 階）——
  這是對 DESIGN.md「預設小字」的**刻意例外**，理由見 Sally 裁定 3：它是海報的圖形替身，
  不是文字標籤。用既有 Title 階而非自創 16px，避免多一個沒有授權來源的字級。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched。

### References

- critique P1「列身分崩塌」；sub-6-10a AC #2/#3/#4（ack）

## Dev Agent Record

### Agent Model Used

Claude Code on the web（2026-09-05）

### Completion Notes List

**交付**：同意清單的每一列現在有封面（或片名首字，不再是空灰塊）、路線與片長**並列**、
未匹配的列標成「未匹配」且標題改用後端解析出的乾淨片名。

**⚠️ 三點裁量**

1. **片長用 `formatRuntime`（`2 小時 46 分`），不是 AC 例子寫的 `1h 52m`。**
   AC 的例子是英文速記，但 `lib/formatMedia.ts` 的 `formatRuntime` 已經是 PosterCard 與
   詳情頁在用的 zh-TW 形式。同一個產品裡有兩套片長寫法，就是兩個畫面對同一部片說不同話的開始。

2. **`≈` 的判定改讀 `runtime_source`，舊伺服器才退回 `runtimeKnown`。**
   `runtimeKnown` 分不出「檔案實測」與「TMDb 的編輯數字」；`runtime_source` 可以。沒有該欄位的
   伺服器行為與本 story 之前完全一致（有測試釘住）。

3. **未匹配標用 `=== false` 判定。** 舊伺服器不送這個欄位，而「伺服器沒告訴我們」不等於
   「TMDb 找不到」—— 後者才該標。

**⚖️ Sally 的三條設計裁定（2026-09-05，推翻已交付的實作，程式已跟改）**

你問「要不要讓 Sally 進 `.pen`」，答案不是把程式碼抄進稿子，是讓她真的做判斷。她看完
之後推翻了三處，程式與測試都已依裁定改過（核定稿：`sub-6-10b-f15-row-pen-prompt.md`）：

1. **副標的 `≈` 拿掉** —— 原本一列裡有兩個 `≈` 指兩件不同的事：金額的 `≈` 是「這個數字
   建立在假設的片長上」，副標的 `≈ 45 分` 是「不知道多長」。而且後者語意本身就錯：
   **約略是量測的誤差，假設是沒有量測** —— 45 分鐘是假設，不是約略量到的。
   改成 `片長未知（估 45 分）`：陳述事實，把假設放進括號。`≈` 每列最多一次，只有一個意思。

2. **「未匹配」移到標題右邊** —— 右側徽章區講的是**這列會發生什麼**（能不能執行／哪條
   路線／多少錢）；「未匹配」講的是**這列是什麼**（身分沒把握）。不同級的東西不並排，
   而且懷疑要貼在被懷疑的東西旁邊 —— 隔半個列寬的灰標，讀者不會把它連回標題。
   現在標題 `truncate`、徽章 `shrink-0`，檔名再長也擠不掉它。

3. **首字放大到 Title 階（18px/600/`--text-secondary`）** —— 原本 12px `--text-muted`
   放在 `--bg-tertiary` 上是灰底灰字，做不到它唯一的任務（讓第 47 列和第 48 列長得
   不一樣）。這是對 DESIGN.md 小字規則的刻意例外，理由已寫進 spec 註記與測試。

**⛔ 兩件需要你的機器**

- **AC #4 `.pen` + 截圖**：這個沙盒沒有 Pencil.app。提示詞見下方「.pen inline-agent 提示詞」。
- **AC #5 的視覺基準線**：`generation-consent/list`／`grouped`／`over-budget` 三個 fixture
  渲染候選列，本 story 改了列的長相 ⇒ **六張基準線（3 fixture × darwin/linux）會出現真實視覺差異**。
  其餘 consent fixture（`analyzing`／`empty`／`confirm`／`confirm-over-budget`／`f16-*`／`f19-*`）
  渲染的是別的元件，不受影響 —— 已逐一核對過 fixture 的 `component`。

  **我刻意不先刪掉舊基準線**：那會讓 CI 從「真實差異」降級成「缺基準線」，等於把該由人眼看的
  變更藏起來。CI 會產出 before/after 的 diff artifact，那正是你確認新版列長相的材料 ——
  也正好回答「要不要讓 Sally 進 `.pen`」。

  **✅ 2026-09-05 更新：artifact 已產出，舊基準線已刪（`e49594b`）。**
  PR #388 的 run 33977572563 上傳了
  [visual-regression-diffs-pr-33977572563](https://github.com/j620656786206/vido/actions/runs/33977572563/artifacts/9973138846)
  （25 檔 / 64 MB / 保存 14 天），三張的 `actual.png` 與 `diff.png` 都在裡面。留著舊基準線的
  理由既已用完，就刪掉那三張 `-linux.png` —— 而且 bootstrap 的判定條件是
  `missing > 0 且 pixel-diff == 0`，舊檔還在就永遠進不了那條分支。

  重烤步驟（確認圖沒問題之後）：
  1. `-darwin`：你的 Mac 上 `pnpm run test:visual:update`，只 stage 那三張。
  2. `-linux`：⛔ **卡在 dispatch** —— `POST /actions/workflows/visual-regression.yml/dispatches`
     回 `403 Resource not accessible by integration`（與 #386 同）。需要你在 Actions 手動
     dispatch `Visual Regression`、分支選 `claude/next-story-recommendation-i04f7j`，
     CI 會自動開 bootstrap PR，合併後檢查轉綠。

### .pen inline-agent 提示詞（AC #4，交給你在 Pencil 跑）

⛔ **已被取代。** 這一版提示詞（首字 12px `--text-muted`、副標 `≈ 45 分（片長未知）`、
未匹配標放右側徽章區）在 2026-09-05 被 Sally 的三條裁定推翻。

**現行版本：`sub-6-10b-f15-row-pen-prompt.md`** —— 內含裁定理由、核定後的列版式表、
六項 Pencil Inline AI Agent 提示詞（多涵蓋 sub-5-3 的群組 header 與 F8 重試按鈕）、
以及交付流程。跑 Pencil 時請用那一份，不要用上面這段。

### Discovery Triage

- ⚖️ **既有測試 `[P0] unknown-runtime rows…` 的斷言改寫**：它原本斷言 `片長未知，以 45 分鐘估算`
  這串**單獨成行**的文案 —— 那正是 AC #2 要消滅的東西（它會蓋掉路線那句）。改為斷言兩段都在，
  不是放寬而是換到新契約。
- 查證後**不成立**：`analyzing`／`empty`／`confirm` 等 fixture 不渲染 `CandidateListPanel`，
  不受本 story 影響（逐一核對 fixture 的 `component` 欄位）。

- 🔧 **CI gate 缺陷 → `backlog-visual-gate-cannot-report-real-diff`（lane ③，雙向連結）**：
  PR #388 的 `Visual Regression / PR` 顯示 **cancelled 而不是 ❌**。根因不是本 story ——
  視覺套件單次要 ~9 分鐘（通過的 run 944 = 10m06s、945 = 10m04s），而
  `playwright.config.ts:48` 在 CI 下是 `retries: 2` ⇒ 任何一次失敗都需要 ~29 分鐘，
  但 `visual-regression.yml:156` 是 `timeout-minutes: 25`。run 946 實測：第 1 次 ✘ 9.0m、
  重試 1 ✘ 9.9m、重試 2 撞上限被砍。**這個 gate 在最需要說話的時候只會說「cancelled」。**
  本 story 是第一個真的踩到它的變更（此前的紅都是「缺基準線」那種快速失敗）。
  **刻意不在本 PR 修**：與本 story 主題無關的 CI 設定，混進來會讓這個 PR 同時做兩件事。
  建議 (a) `visual` project 加 `retries: 0`（同檔 `bisect` project L200-207 已有同理由先例）
  或 (b) PR job `timeout-minutes` 25 → 40，另開 PR 由 Alexyu 裁定。

### File List

- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx`
- `_bmad-output/implementation-artifacts/sub-6-10b-f15-row-pen-prompt.md`（Sally 核定稿 + Pencil 提示詞）
- ⏳ `ux-design.pen` + `_bmad-output/screenshots/flow-f-subtitle-v2/f15-*`、`f8-*`（待你在 Pencil 執行）
- ⏳ 六張視覺基準線（待確認 diff 後重烤）
