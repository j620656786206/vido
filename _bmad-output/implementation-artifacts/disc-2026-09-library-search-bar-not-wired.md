# Disc：媒體庫自己的搜尋框從沒接上畫面，刪掉

Status: done

**Source:** `disc-2026-09-library-search-bar-not-wired`（2026-09-22 由 dsr-1b-b 的 /ship 對抗式 CR 立案，P2）。2026-10-12 Alexyu 選「刪掉」。

## 查到的事（2026-10-12，Bob 逐一開檔確認）

### 早就有的決定

- **ADR D4-5（`_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md:353-361`）：** 搜尋只留一個，就是頂欄的全域搜尋（omnisearch，甲案），同時搜本機片庫和 TMDb。「只在某一區搜尋」的乙案**明確否決**，理由是切換範圍本身就是 P9「三個搜尋框讓人搞不清楚」的來源。
- 頂欄搜尋已經會搜本機片庫（`bugfix-ts-r1-search-owned-degradation`，PR #171 加的本機那一路）。
- `disc-2026-09-unmounted-v1-components` 把 `library/LibrarySearchBar.tsx` 列為 v1 遺留、只活在視覺夾具裡。
- 所以這張不是新的產品決定，是照 D4-5 收尾。Alexyu 2026-10-12 也記得「之前有決議」。

### 現在的樣子

- `components/library/LibrarySearchBar.tsx`（112 行）＋ spec（212 行）：產品裡沒有任何地方用到，只有視覺夾具 `library-library-search-bar` 在用，佔 6 張基準圖（default／hover／focus × darwin／linux）。
- `hooks/useLibrary.ts` 的 `useLibrarySearch` 和 `libraryKeys.search`／`searches`：除了自己的 spec 沒有人呼叫。
- `services/libraryService.ts` 的 `searchLibrary`：只有 `useLibrarySearch` 用。`types/library.ts` 的 `LibrarySearchResponse`／`LibrarySearchResult` 只有它用。
- 媒體庫頁 `LibraryBrowseV2` 一律走 `useLibraryInfinite` → `GET /library`，沒有搜尋模式。

## Story

身為維護 Vido 的人，
我希望媒體庫那個從沒放上畫面的搜尋框和它專用的程式一起刪掉，
這樣 CI 不用再比對沒人看得到的畫面，下次也不會有人以為「媒體庫有自己的搜尋」而去修它。

## Acceptance Criteria

1. 刪掉 `LibrarySearchBar.tsx` 和它的 spec。
2. 刪掉視覺夾具 `library-library-search-bar`、它的 6 張基準圖，以及 `_bmad-output/audit/visual-baseline-19-4.md` 那一列。
3. 刪掉 `useLibrarySearch`、`libraryKeys.search`／`searches`、`libraryService.searchLibrary`、`LibrarySearchResponse`／`LibrarySearchResult`，以及它們的測試。
4. **後端 `GET /library/search` 不動**：它是公開 API 的一部分，`tests/manual/full-app-audit.spec.ts` 也會打。前端沒人用不代表 API 要拿掉。
5. 畫面上沒有任何變化：媒體庫頁、頂欄搜尋行為不變。
6. `pnpm nx test web`、`pnpm run lint:all`、typecheck 全綠；grep `tests/` 確認 E2E 沒有用到被刪的 testid 或元件。
7. `sprint-status.yaml`：本條 done；`disc-2026-09-unmounted-v1-components` 的清單劃掉 `LibrarySearchBar`。

## Tasks / Subtasks

- [x] T1 刪元件、spec、夾具、基準圖、稽核表那一列（AC #1、#2）
- [x] T2 刪 hook／service／型別與測試（AC #3）
- [x] T3 檢查全綠、grep E2E（AC #6）
- [x] T4 sprint-status 兩條更新（AC #7）

## Dev Notes

- E2E 的 `tests/e2e/empty-library.spec.ts:159-166` 和 `tests/support/helpers/batch-subtitle-stubs.ts:193` 會攔 `/library/search*`。那是「萬一有人打就回空的」保險，不依賴前端元件。這張不動，免得改到 E2E 的攔截順序。
- 只刪不加，沒有設計稿要對。

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- 刪 `components/library/LibrarySearchBar.tsx`、`LibrarySearchBar.spec.tsx`。
- `routes/test/-gallery.fixtures.tsx`：刪 import 和 `library-library-search-bar` 夾具。
- 刪 `tests/visual/components.visual.spec.ts-snapshots/components/library-library-search-bar/` 6 張圖；`visual-baseline-19-4.md` 刪那一列。
- `hooks/useLibrary.ts`：刪 `useLibrarySearch`、`libraryKeys.searches`／`search`。spec 刪 `useLibrarySearch` 那組測試和 mock 裡的 `searchLibrary`。
- `services/libraryService.ts`：刪 `searchLibrary` 和型別 import；spec 刪 `searchLibrary` 那組和「跟 listLibrary 送一樣的篩選」那條。
- `types/library.ts`：刪 `LibrarySearchResult`、`LibrarySearchResponse`。
- `hooks/useMediaDetails.spec.tsx`：mock 裡的 `searchLibrary` 拿掉。

### 驗證

- `pnpm nx test web`：299 檔、4661 測試全綠（少的 1 檔、32 條是刪掉的 LibrarySearchBar spec 和搜尋那幾組）。
- `tsc --noEmit -p apps/web/tsconfig.app.json` 綠。
- `pnpm run lint:all`：0 errors（172 warnings 都是原本就有的）；`format:check` 綠。
- grep `apps/web/src` 和 `tests/`：`searchLibrary`、`useLibrarySearch`、`LibrarySearchBar`、`LibrarySearchRes*`、`libraryKeys.search` 都已經沒有任何引用。E2E 沒有用到 `LibrarySearchBar` 的 testid。

### 範圍外（沒修）

- 後端 `GET /library/search` 留著（AC #4）。
- `EmptySearchResults.tsx` 也只剩夾具在用，歸 `disc-2026-09-flow-c-library-screens-c1-c3`。
- `disc-2026-09-unmounted-v1-components` 其餘 v1 遺留元件，等那張一次判。

### File List

- `apps/web/src/components/library/LibrarySearchBar.tsx`（刪）、`LibrarySearchBar.spec.tsx`（刪）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `apps/web/src/hooks/useLibrary.ts`、`useLibrary.spec.tsx`
- `apps/web/src/hooks/useMediaDetails.spec.tsx`
- `apps/web/src/services/libraryService.ts`、`libraryService.spec.ts`
- `apps/web/src/types/library.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/library-library-search-bar/`（刪 6 張）
- `_bmad-output/audit/visual-baseline-19-4.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
