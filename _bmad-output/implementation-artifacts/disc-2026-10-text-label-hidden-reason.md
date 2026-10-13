# Disc：有字的標籤，原因只藏在瀏覽器 title 裡

Status: done

**Source:** `disc-2026-10-text-label-hidden-reason`（2026-10-13 從 `disc-2026-10-icon-only-controls-tooltip` 拆出，待 Sally 裁定）。2026-10-13 Alexyu：「做 disc-2026-10-text-label-hidden-reason」。

## 查到的事（2026-10-13，main `86c08533`）

這些標籤有字，所以不在 J11-D 規則 6（只有圖示）裡。但重要的原因只放在瀏覽器 `title`：手機看不到，鍵盤切不到。

| 位置 | 看得到的字 | 只在 title 的內容 | 觸發點 |
|---|---|---|---|
| 選片清單 `CandidateListPanel.tsx:378`（片名） | 解析後的片名 | 真實檔名（sub-6-10b AC #3：使用者要拿它去磁碟找檔案） | span |
| 同上 `:380`「未匹配」 | 未匹配 | TMDb 沒有比對到，片名由檔名解析 | span |
| 同上 `:402`「資料夾無法寫入」 | 資料夾無法寫入 | 是哪個資料夾 | span |
| 同上 `:411`「無法估價」 | 無法估價 | 為什麼不能選 | span |
| 模型選擇 `ModelPicker.tsx:140` | 品質 A | 等級的來源（`aria-label` 已帶） | span，在 `<label>` 裡 |
| 下載卡片 `DownloadCardV2.tsx:94` | NZBGet（灰） | 尚未支援（另有 sr-only） | span，每張卡都有 |
| 篩選預設 `PresetChips.tsx:112` | 預設名稱 | 右鍵或長按可刪除 | button |
| 下載表格 `DownloadsTableV2.tsx:158` | qB | qBittorrent | span，每列都有 |

## 設計裁定（Sally，2026-10-13）

1. **原因會影響你能做什麼的 → 提示框。** 「未匹配」「資料夾無法寫入」「無法估價」是狀態，原因決定這一列能不能選、要去哪裡修。照 DESIGN.md §Tooltips 的狀態類做法：真的按鈕當觸發點、`delay={0}`、`openOnPress`，名字＝標籤＋原因。
   - 「未匹配」的提示框兩行：第一行「TMDb 沒有比對到，片名由檔名解析」，第二行「檔名：…」（J11-D 規則 3：第一行是結論，第二行是來源）。片名本身不再掛 title——檔名改由這個提示框提供，手機也看得到。
2. **只能靠提示發現的操作 → 提示框。** 篩選預設的「右鍵或長按可刪除」用控制類做法（移過、切到時出現；本來就是按鈕）。
3. **縮寫 → `<abbr title>`。** 下載表格的「qB」每列都一樣，每列變成按鈕會多一個 Tab 停點（50 列就多按 50 下）。用 HTML 縮寫標記，輔助科技可以展開；卡片檢視已經寫全名。
4. **維持原樣：**
   - 模型品質等級的來源：它在 `<label>` 裡，不能再放一顆按鈕；`aria-label` 已經帶來源。
   - 下載卡片灰掉的「NZBGet」：每張卡都有，灰色已經說明不能用，螢幕閱讀器另有「尚未支援」。每張卡多一個 Tab 停點不划算。
5. 不畫新設計稿：標籤的樣子沒變，只是多了提示框。規則寫進 DESIGN.md §Tooltips。

## Story

身為用手機或鍵盤的人，
我希望選片清單上「為什麼這一部不能選」「這個檔案真正叫什麼」也看得到，
這樣不用拿滑鼠停在上面等。

## Acceptance Criteria

1. 選片清單三個標籤照裁定 1 改；片名拿掉 title。
2. 篩選預設照裁定 2 改。
3. 「qB」改成 `<abbr title="qBittorrent">`，外觀不變（不加底線）。
4. `DESIGN.md` §Tooltips 補上這三種情況與「維持原樣」的理由。
5. 測試：標籤的名字含原因、沒有 title、點一下會出現；預設鍵盤切到會出現提示；qB 是 `<abbr>` 且不能 Tab 到。
6. 檢查：vitest、`tsc`、`pnpm run lint:all`、`format:check`、visual（外觀不應該變）。

## Tasks / Subtasks

- [x] T1 選片清單 `ReasonBadge`（AC #1）
- [x] T2 篩選預設（AC #2）
- [x] T3 qB（AC #3）
- [x] T4 DESIGN.md（AC #4）
- [x] T5 測試與檢查（AC #5、#6）

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- `CandidateListPanel.tsx` 新增 `ReasonBadge`：按鈕觸發點＋`Tooltip`（`delay={0}`、`openOnPress`）。名字是「標籤：原因」；原因本來就以標籤開頭時（「資料夾無法寫入：ro-folder」）不重複。className 跟原本的 span 一樣。
- 片名的 `title` 拿掉，真實檔名改在「未匹配」提示框第二行。原本三條斷言 `title` 的測試改成斷言名字與提示框。
- `PresetChips`：`key` 移到外層 `Tooltip`，`title` 拿掉。
- `DownloadsTableV2`：`<abbr title="qBittorrent">`，加 `no-underline` 蓋掉瀏覽器預設的點狀底線。
- 檢查：vitest 全部 4,484 個測試、`tsc`、`pnpm run lint:all`（0 errors）、`format:check`、本機 visual 全綠（基準圖沒變）。`tests/e2e` 沒有用到這些 title。

### File List

- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`、`.spec.tsx`
- `apps/web/src/components/search/PresetChips.tsx`、`.spec.tsx`
- `apps/web/src/components/downloads/DownloadsTableV2.tsx`、`.spec.tsx`
- `DESIGN.md`
- `_bmad-output/implementation-artifacts/disc-2026-10-text-label-hidden-reason.md`（新）、`sprint-status.yaml`
