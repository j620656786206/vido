# disc-2026-10-batch-mixed-selection-wrong-type：同時勾電影和影集再整批操作，兩種都照各自的類型處理，失敗原因講人話

Status: done

**Source:** 2026-10-22 `testsprite-2026-10-quota-run` 查 TC115 時本機重現（Rule 24 lane ③ 立案）。Alexyu 排序：三個 bug 裡第一個修。

## 為什麼要做

媒體庫「全部」頁可以同時勾電影和影集。按「重新解析」「刪除」「匯出」時，前端把整批 id 用**同一個類型**送出：最後點的那一個是電影，就全部當電影送。結果另一種全部失敗，失敗清單直接顯示後端的原始錯誤：

```
Unknown.Show.S01: movie with id seed-sr-101 not found: sql: no rows in result set
```

使用者看到的是：勾了 4 部，2 部「失敗」，原因是一串看不懂的英文。刪除也一樣——影集刪不掉，還以為是 Vido 壞了。

## Acceptance Criteria

1. **依類型分開送。** 勾選裡同時有電影和影集時，重新解析、刪除、匯出都對每一種類型各送一次（電影的 id 用 `movie`、影集的 id 用 `series`）；只有一種時跟現在一樣送一次。類型取自勾選當下每張卡片自己的類型，不是「最後點的那一個」。
2. **結果合併成一個。** 進度視窗顯示的成功數、失敗數、失敗清單是兩次加起來的；重新解析只要任一次有成功，就照舊開始追比對進度。
3. **匯出合成一個檔。** 混選時匯出一個檔，內容是兩種的項目接在一起；檔名 `vido-export-library-<日期>.json`。只有一種時檔名照舊（`vido-export-movie-…`／`vido-export-series-…`）。
4. **失敗原因講人話，不露出內部錯誤。** 整批操作的失敗清單：找不到那一筆時寫「找不到這個項目（可能已被刪除）」；其他失敗寫「處理失敗，詳細原因記在系統日誌」。原始錯誤只寫進伺服器日誌（現在已經有寫）。
5. **測試。** 前端：混選電影＋影集按重新解析／刪除／匯出，各送出兩個請求、類型正確、結果合併；只選一種時只送一個請求。後端：找不到 → 人話訊息；其他錯誤 → 通用訊息；不再出現 `sql:`、`not found` 等原始字串。

## 不要做的事

- 不改 API 形狀（`{ids, type}` 照舊），不讓後端接受混合清單。
- 不改進度視窗的版面。
- 不動「重新解析對話框卡在等待開始」（那是下一張 `disc-2026-10-reparse-dialog-misses-instant-pass`）。

## Tasks

- [x] 前端：`executeBatchAction` 依 `selectionTypesRef` 分組送出、合併結果（AC 1–3）
- [x] 後端：`BatchDelete`／`BatchReparse` 的 `BatchError.Message` 改成人話（AC 4）
- [x] 測試（AC 5）
- [x] :8090 本機重現原本的混選失敗，修完再操作一次確認

## Dev Notes

- `apps/web/src/components/library/LibraryBrowseV2.tsx`：`selectedType` 是最後點的那張卡的類型（`handleSelect`）；`selectionTypesRef` 已經記了每個 id 的類型，但送出時沒用。
- `apps/api/internal/services/library_service.go`：`BatchDelete`、`BatchReparse` 把 `err.Error()` 原樣放進 `BatchError.Message`。repository 的「找不到」有兩種寫法：`… not found: sql: no rows in result set`（`FindByID`，包了 `sql.ErrNoRows`）和 `movie with id … not found`（`Delete`）。
- 既有測試 `TestLibraryService_BatchErrors_CarryTitle` 斷言了原始訊息，要跟著改。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

- 前端：送出前用 `selectionTypesRef`（每張卡自己的類型）把 id 分組，每種類型送一次，成功數、失敗數、失敗清單相加；匯出把兩次的陣列接起來，混選時檔名 `vido-export-library-<日期>.json`。查不到類型時退回 `selectedType`（原本的行為）。
- 後端：`batchErrorMessage` 把失敗原因換成「找不到這個項目（可能已被刪除）」或「處理失敗，詳細原因記在系統日誌」；原始錯誤照舊寫在同一行 log。
- 測試：前端新增 4 條（混選重新解析／刪除／匯出各送兩次且結果合併、單一類型只送一次）；把元件改回舊版時前 3 條會紅。後端改 1 條、加 1 條（兩種「找不到」都變人話、不含 `sql`／`not found`）。
- 本機 :8090（重新 build）：勾《[FanSub] 未知電影》＋《Unknown.Show.S01》按重新解析，送出兩個請求（movie／series），對話框沒有失敗清單。修之前同樣操作會出現「movie with id seed-sr-101 not found: sql: no rows in result set」。
- 驗證：`go test`（services／handlers 的 Batch）、staticcheck、`lint:all`（0 errors）、tsc、library 相關 vitest 23 檔 346 條全綠。
- 自我審查：第一次呼叫丟例外時，對話框照舊只寫「操作失敗」，不會說另一種類型有沒有做到——跟修之前一樣，沒有擴大範圍。

### File List

- `apps/web/src/components/library/LibraryBrowseV2.tsx`、`LibraryBrowseV2.spec.tsx`
- `apps/api/internal/services/library_service.go`、`library_service_test.go`
- 本檔、`sprint-status.yaml`
