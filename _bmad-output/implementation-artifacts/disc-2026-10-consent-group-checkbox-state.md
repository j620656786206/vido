# Disc：選片清單「選整部／選整季」勾選框的狀態要對

Status: done

**Source:** 2026-10-13 Alexyu 選 A 時加註：「另外也要確認每個卡片的勾選狀態要是對的，不管影集或者季數」。

## 查到的事（2026-10-13，main `979a25da`）

- 群組標題列（影集、季、電影分組）的勾選框由 `GroupHeaderRow`（`CandidateListPanel.tsx`）算：
  - `ids` = 這組裡「看得到、而且能選」的集數（`visibleSelectableIds`）。
  - `all` = `ids` 不是空的，而且 `computeTotals(...).visibleSelectedCount === ids.length`。
  - `some` = 有選但不是全選 → 半勾（indeterminate）。
- **問題**：`visibleSelectedCount` 數的是「看得到而且已選、有價錢」的列，**不管能不能選**；`ids` 只算能選的。兩個數字算的不是同一群，所以：
  - 如果一個不能選的列（資料夾無法寫入）因為某種原因還在已選名單裡，它會被算進分子，標題可能在「其實還有一集沒選」時顯示全勾。
  - 一組裡全部都不能選時（`ids` 是空的），勾選框照樣可以按，按了什麼都不會發生。
- 現有測試涵蓋：影集半勾、全選後取消、不能選的那集不擋「全勾」。**沒有**涵蓋：季標題各自的狀態、影集與季同時的狀態組合、已選名單裡混進不能選的列、全組都不能選。

## Story

身為要決定哪幾集付錢的人，
我希望「選整部」「選整季」的勾選框永遠說實話：全勾就是全選了，空的就是一集都沒選，
這樣我不會以為選好了其實沒選，或以為沒選其實選了。

## Acceptance Criteria

1. 勾選狀態只看 `ids` 本身：`已選的能選集數 = ids 裡有幾個在已選名單`。`all = ids.length > 0 且 已選 === ids.length`；`some = 已選 > 0 且不是 all`。
2. 一組裡沒有能選的集數時，勾選框停用（`disabled`），不是全勾也不是半勾。
3. 測試（兩季＋特別篇的影集）：
   - 什麼都沒選：影集、各季都是空的。
   - 只選第 1 季的一集：第 1 季半勾、特別篇空、影集半勾。
   - 第 1 季全選：第 1 季全勾、特別篇空、影集半勾。
   - 全部都選：每一層都全勾。
   - 按第 1 季的勾選框：只送出第 1 季能選的集數；按影集：送出全部能選的集數。
   - 已選名單裡混進一個不能選的集數：不會讓標題變成全勾。
   - 全組都不能選：勾選框停用。
   - 電影分組標題同樣成立（抽一個情況）。
4. 檢查：vitest、`tsc`、`pnpm run lint:all`、`format:check`、visual。

## Tasks / Subtasks

- [x] T1 `GroupHeaderRow` 改算法、全組不能選時停用（AC #1、#2）
- [x] T2 測試（AC #3）
- [x] T3 檢查（AC #4）

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- 先寫 8 條測試再改程式。結果：
  - 影集／季／特別篇的空、半勾、全勾組合，按季只送那一季、按影集送全部，電影分組：**原本就對**，測試全綠。
  - 「已選名單裡混進不能選的集數」：原本也沒出錯——`computeTotals` 對不能選的列回不出價錢，所以不會算進分子。但那是剛好碰到，不是設計好的。
  - 「全組都不能選」：**原本是錯的**，勾選框可以按、按了沒反應。這條測試先紅後綠。
- `GroupHeaderRow` 改成直接數「`ids` 裡有幾個在已選名單」，分子分母永遠是同一群，不再依賴 `computeTotals` 剛好排除不能選的列。
- `ids` 是空的時候勾選框 `disabled`。
- 檢查：vitest 全部 4,492 個測試、`tsc`、`pnpm run lint:all`（0 errors）、`format:check`、本機 visual 全綠（基準圖沒變）。

### File List

- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`、`.spec.tsx`
- `_bmad-output/implementation-artifacts/disc-2026-10-consent-group-checkbox-state.md`（新）、`disc-2026-10-consent-tree-indent.md`（新）、`disc-2026-10-consent-tree-indent-pen-prompt.md`（新）、`sprint-status.yaml`
