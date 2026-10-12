# Disc：元件檔頭指向的設計稿節點，CI 要確認它還在

Status: done

**Source:** `disc-2026-09-rule21-node-existence-unguarded`（2026-09-16 由 dsr-1 立案，P2）。一併收 `disc-2026-09-dangling-design-refs-outside-library`（P3，同一類問題，當初就建議「等掃描做出來一次清」）。2026-10-12 Alexyu 選定（「先 A 再 C」）。

## 查到的事（2026-10-12，Bob 開檔＋用 Pencil MCP 實查）

- ESLint `local/implements-pen-node-id`（`apps/web/src/eslint-rules/implements-pen-node-id.js`）只驗格式：一行、結尾 `(節點ID)`。不驗節點在不在 `ux-design.pen`。
- `.pen` 加密，CI 讀不到。已有先例：`export-pen-screenshots.py` 把設計 token dump 到 `_bmad-output/pen-tokens.json`（帶 `.pen` sha256），CI 的 `check-design-tokens.py` 比對。
- 實查（Pencil MCP，2026-10-12）：設計稿 18,886 個節點，其中 frame／group／元件實例 9,992 個。元件檔頭引用 219 個不同節點，全部是 frame；**5 個不存在**，分布在 7 個檔案：
  - `sAaCR`：`dashboard/DownloadPanel.tsx`、`dashboard/RecentMediaPanel.tsx`
  - `RgSxQ`：`degradation/DegradationBadge.tsx`
  - `KNI8F`：`degradation/ServiceHealthBanner.tsx`、`notifications/NewMediaNotifications.tsx`
  - `NXijD`：`subtitle/BatchSubtitleDialog.tsx`
  - `cOrOR`：`subtitle/SubtitleSearchDialog.tsx`
- **這 7 個元件全部沒有掛在畫面上**（正式程式沒有任何地方 import 或渲染它們），跟今天刪掉的 13 個是同一類。

## Story

身為維護 Vido 的人，
我希望元件檔頭指向一個已經不存在的設計稿畫面時，CI 會擋下來，
這樣「看起來對齊過、其實指向空氣」的檔頭不會再悄悄累積。

## Acceptance Criteria

1. `export-pen-screenshots.py` 另外寫 `_bmad-output/pen-node-ids.json`：`{penSha256, nodeIds}`，nodeIds 是所有 frame／group／元件實例的 id（排序、不重複）。新增 `--snapshot-only`：只寫兩份快照、不重新匯出截圖。`pen-tokens.json` 的內容不變。
2. 新腳本 `scripts/check-design-refs.py`：掃 `apps/web/src` 非 spec 檔的開頭註解，`Implements: Component/…(id)` 和 `Design ref: ux-design.pen Screen …(id)` 裡的每個 id 都要在快照裡；快照的 sha 跟 `.pen` 不同時也失敗。訊息是 zh-TW，寫清楚怎麼修。
3. CI（`test.yml`，跟 token 檢查同一個 job）加這一步。
4. 那 7 個檔頭改成 lint 已接受的誠實寫法：`// Design ref: ux-design.pen — no current screen frame; <原因>`。元件本身這張不刪——它們是另一批沒掛載的元件，立 `disc-2026-10-more-unmounted-components` 照今天 (a) 的裁定處理。
5. `CLAUDE.md` 的截圖流程補一行：匯出腳本會寫哪兩份快照、`--snapshot-only` 的用法。

## Tasks / Subtasks

- [x] T1 匯出腳本：節點快照＋`--snapshot-only`（AC #1）
- [x] T2 `check-design-refs.py`（AC #2）
- [x] T3 CI 步驟（AC #3）
- [x] T4 7 個檔頭（AC #4）
- [x] T5 CLAUDE.md、sprint-status（AC #5）

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- `scripts/export-pen-screenshots.py`：token 快照那段 JS 走訪時順便收集 frame／group／ref 的 id，Python 端從 payload 拿出來寫成 `pen-node-ids.json`（9,992 個，約 90 KB），不放進 `pen-tokens.json`。加 `--snapshot-only`。
- 開著 Pencil 跑 `--snapshot-only`：`pen-tokens.json` 沒有任何變動（確認 token 那份輸出不受影響）。
- `scripts/check-design-refs.py`：只看檔案開頭的註解（跟 ESLint 規則一致）；目前 380 個檔頭節點全部存在。
- 7 個檔頭改成「no current screen frame」寫法，原因寫明原畫面已刪、元件沒掛載。

### 驗證

- 修檔頭前跑：抓到的就是 MCP 實查的那 5 個 id、7 個檔案。
- 突變：把 `ButtonCost.tsx` 的 `qAERt` 改成 `qAERx` → 失敗並指出檔案；把快照 sha 改掉 → 報「快照過期」。都還原。
- 改過的 7 個檔案 eslint 0 error（新寫法符合 `implements-pen-node-id`）。

### 範圍外

- 7 個沒掛載的元件本身 → `disc-2026-10-more-unmounted-components`。
