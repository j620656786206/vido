# Disc：一次刪掉畫面上沒用到的舊版元件

Status: done

**Source:** `disc-2026-09-unmounted-v1-components`（2026-09-16 由 dsr-1 的對抗式 CR 立案，P2，選項 (a) 全刪／(b) 全留標註／(c) 個別判）。2026-10-12 Alexyu 選 (a)：「先 A 再 B」。

## 查到的事（2026-10-12，Bob 逐一開檔確認，main `09784cc4`）

用腳本掃 `apps/web/src`，下面這些元件**正式畫面沒有任何地方 import**，只活在 `routes/test/-gallery.fixtures.tsx`（截圖測試夾具）和自己的 spec：

| 元件 | 誰取代了它 |
|---|---|
| `library/LibraryGrid.tsx`、`LibraryTable.tsx`、`RecentlyAdded.tsx` | `LibraryBrowseV2`、`LibraryListRowV2`、`RecentlyAddedRowV2` |
| `homepage/TrailerModal.tsx` | 詳情頁的 `TrailerSection`／`TrailerEmbed` |
| `media/MediaDetailPanel.tsx`，以及只有它在用的 `DetailPanelMenu.tsx`、`FileInfo.tsx`、`MetadataSourceBadge.tsx` | `LocalDetailV2`／`DetailHeroV2` |
| `media/TVShowInfo.tsx` | 同上 |
| `media/TechBadgeGroup.tsx`，以及只有它在用的 `TechBadge.tsx` | `DetailTechInfoV2` |
| `dashboard/QuickSearchBar.tsx` | 頂欄全域搜尋（ADR D4-5） |
| `media/ColorPlaceholder.tsx` 的**元件本體** | 它匯出的 `filenameToGradient` 有 `PosterCardV2`、`ManualMatchDialogV2`、`DetailHeroV2` 在用，要留 |

另外：

- `routes/media/-$type.$id.spec.tsx` 的「Media Detail Route」那組測試，是在測一個用 `MediaDetailPanel`＋`TVShowInfo` 拼出來的假路由，正式路由早就不是這樣，一起刪。
- `styles-type-scale.spec.ts` 有兩列指名 `ColorPlaceholder.tsx`、`MediaDetailPanel.tsx` 的字級，跟著刪。
- E2E：上面這些元件的 testid，E2E 只用到 `media-detail-panel`（`tests/e2e/manual-search.spec.ts:148`），而且是「三選一、出現才檢查」的選擇器，刪了不影響。其他 testid（`library-grid`、`detail-poster`…）V2 元件也有，照常存在。
- 視覺基準：`media-media-detail-panel` 就是 `backlog-visual-main-4-workers-flaky-media-detail-focus` 那個會偶爾失敗的截圖，刪掉它，那張單的來源就消失了。

### 不刪的（之前已有決定）

- `library/ParseFailureCard.tsx` 和 `manual-search/` 四個元件：9/17 決定保留，`/test/manual-search` 和它的 E2E 還在用。
- `library/EmptySearchResults.tsx`：歸 `disc-2026-09-flow-c-library-screens-c1-c3`。

## Story

身為維護 Vido 的人，
我希望畫面上沒用到的舊版元件連同它們的截圖一起刪掉，
這樣 CI 不用比對沒人看得到的畫面，也少一個會偶爾失敗的截圖。

## Acceptance Criteria

1. 刪掉上表的元件（`ColorPlaceholder` 只刪元件本體）和它們的 spec、gallery 夾具、`-darwin`／`-linux` 基準圖、`_bmad-output/audit/visual-baseline-19-4.md` 對應的列。
2. `filenameToGradient` 搬到 `apps/web/src/utils/filenameToGradient.ts`，三個使用處改 import，測試跟著搬。
3. 刪掉 `-$type.$id.spec.tsx` 的「Media Detail Route」那組、`styles-type-scale.spec.ts` 那兩列。
4. 畫面完全不變。
5. `pnpm nx test web`、`tsc`、`pnpm run lint:all` 全綠；grep `apps/web/src` 和 `tests/`，被刪的元件已經沒有 import。
6. `sprint-status.yaml`：本條 done；`backlog-visual-main-4-workers-flaky-media-detail-focus` 註記夾具已刪。

## Tasks / Subtasks

- [x] T1 刪元件、spec、夾具、基準圖、稽核表的列（AC #1）
- [x] T2 搬 `filenameToGradient`（AC #2）
- [x] T3 清兩個 spec 裡的舊測試（AC #3）
- [x] T4 檢查全綠（AC #5）
- [x] T5 sprint-status（AC #6）

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- 刪 13 個元件和它們的 13 個 spec（`ColorPlaceholder` 整檔刪掉，函式先搬走）。
- `utils/filenameToGradient.ts`＋spec：函式和它的 4 條測試原封搬過來；`PosterCardV2`、`ManualMatchDialogV2`、`DetailHeroV2` 改 import。
- `-gallery.fixtures.tsx`：刪 14 個夾具（`library-recently-added` 有 recent／stale 兩個）、13 個 import、兩個只剩它們的分區標題、用不到的型別 import；HeroBanner 夾具註解裡提到 recently-added 夾具的那句改掉。
- 刪 13 個基準圖資料夾（darwin＋linux）；`visual-baseline-19-4.md` 刪 13 列。
- `-$type.$id.spec.tsx`：刪「Media Detail Route」那組（測的是用舊元件拼的假路由）和只給它用的 helper／import。
- `styles-type-scale.spec.ts`：刪兩列豁免。

### 驗證

- `pnpm nx test web`：287 檔、4425 測試全綠（少 12 檔：刪 13 個 spec、加 1 個 utils spec）。
- `tsc --noEmit -p apps/web/tsconfig.app.json` 綠；改過的檔案 eslint 0 error。
- grep：被刪的元件在 `apps/web/src` 沒有任何 import。E2E 只有 `manual-search.spec.ts:148` 的「三選一、出現才檢查」選擇器提到 `media-detail-panel`，不影響。
- `tests/visual/clock-mock.ts`、`bootstrap-detection.spec.ts`、`visual-regression.yml`、`time-dependent-fixture-stability.js` 裡提到 `library-recently-added` 的是說明範例或測試用的 log 字串，不依賴夾具存在，沒改。

### 範圍外

- `EmptySearchResults.tsx`（歸 flow-c-c1-c3）、`ParseFailureCard.tsx`＋manual-search（9/17 決定保留）。
