# Story: disc-2026-09-batch-error-shows-id-not-title — 批次操作的失敗清單顯示片名，不是 id

Status: done

<!-- ⚠️ 補建：SM Bob 2026-09-29。實作先於 story（Dev 直接從 sprint-status 一行做起，PR #582 已開），
     Alexyu 攔下「應先請SM CS story?」→ 補這份，AC 依既有實作與設計稿反推，並做過事實查證。
     以後不論多小的 disc 都先 create-story（memory: feedback_create_story_before_dev_even_small）。 -->

## Story

身為在媒體庫勾了幾部片按「刪除」或「重新解析」的人，
我要失敗清單告訴我是**哪一部**沒成功（片名），
而不是一串資料庫 id，讓我知道該去看哪一部。

## 背景（查到的事，附親自讀過的位置；行號為 main `46c93ab8`，改動前）

- `apps/api/internal/services/library_service.go:751-754`：`BatchError{ID, Message}`，沒有片名。
- 兩個組裝點：`:772`（`BatchDelete`，直接 `Delete`，**沒有**先讀列）與 `:812`（`BatchReparse`，`:793` 已 `FindByID` 讀到 `movie`／`series`，片名就在手上但沒帶出去）。
- 前端 `apps/web/src/components/library/BatchProgress.tsx:122-124`：每列渲染 `{err.id}: {err.message}`；型別 `apps/web/src/types/library.ts:293-296` `BatchError{id, message}`；頁面狀態 `LibraryBrowseV2.tsx:332` 自帶 `{ id; message }[]`。
- handler 直接把 `BatchResult` 回給前端（`library_handler.go` `BatchReparse` → `SuccessResponse(c, result)`），所以後端多一個欄位前端就拿得到。
- 設計稿 C24-D（`L7EVR`，#578）失敗清單畫的是片名：「寄生上流：找不到符合的作品」。
- 負面查證：`git grep -n "BatchError\b" -- apps/api/internal/handlers ':!*_test.go'` 無結果（handler 不另外組 BatchError）；前端 `git grep "err.id\|errors.map"` 只有 `BatchProgress.tsx` 一處消費者；E2E `tests/support/helpers/api-helpers.ts:536` 的 `batchReparse` 型別只看 `success_count`／`failed_count`，加欄位不影響。

## Acceptance Criteria

1. `BatchError` 多一個 `title`（JSON `title`，空值省略）。重新解析：已讀到列時帶片名；刪除：**只在失敗那一筆**才讀一次片名，成功的項目不多查資料庫。
2. 列讀不到（已不存在、DB 錯誤）時 `title` 為空，前端退回顯示 id。
3. 前端失敗清單每列顯示 `{title || id}: {message}`。
4. 測試：Go — 重新解析寫入失敗帶片名、列不存在不帶、刪除失敗查片名、刪除成功不查；前端 — 有片名顯示片名、沒有退回 id。
5. 視覺基準零變動：畫廊「比對完成」夾具改成帶 `title`，畫面文字與 #579 相同。

## Tasks / Subtasks

- [x] Task 1 — 後端 `BatchError.Title` ＋ `batchTitle()`，兩個組裝點接上（AC #1, #2, #4）
- [x] Task 2 — 前端型別、`BatchProgress` 顯示、頁面狀態型別、夾具（AC #3, #5）
- [x] Task 3 — 測試與 lint／typecheck（AC #4）

## Dev Notes

- 兩側都是加欄位，不改既有欄位；`omitempty` 讓舊前端不受影響。
- **不要**在成功路徑呼叫 `FindByID`（批次刪 200 部就多 200 次查詢）——AC #1 明寫、測試 `delete: success does not read the row at all` 守著。
- 失敗訊息本身仍是英文（`database is locked`、`movie not found`），另案（見 Discovery Triage）。

### Project Structure Notes

- 後端只動 `services/library_service.go`（型別＋兩個組裝點）；handler 零改動。
- 前端動 `types/library.ts`、`components/library/BatchProgress.tsx`、`LibraryBrowseV2.tsx`（型別一行）、`routes/test/-gallery.fixtures.tsx`。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 設計稿 C24-D：`_bmad-output/screenshots/flow-c-search-settings/c24-d.png`（node `L7EVR`）
- 前置：`disc-2026-09-batch-reparse-progress-in-dialog`（#579）

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 後端：`BatchError.Title`；`batchTitle(ctx, id, mediaType)` 只在失敗路徑呼叫；`BatchReparse` 直接用已讀到的 `movie.Title`／`series.Title`。
- 前端：`BatchProgress` 顯示 `err.title || err.id`；夾具帶 `title`，畫面文字不變。
- 自審：刪除成功路徑不多查（測試守住）；`omitempty` 保住舊格式；handler 不用改。
- 驗證：Go `services`／`handlers` 套件全綠（4 條新測試）；前端相關 52 條全綠（1 條新）；`lint:all`、typecheck 綠。
- 流程備註：實作先於 story，Alexyu 攔下後補建；PR #582。

### Discovery Triage

- ③ backlog-with-carry-forward-link — 失敗訊息本身是後端英文原字串（`err.Error()`，`library_service.go:772/:812`）→ `disc-2026-09-batch-error-message-english`（P4）。

### File List

- apps/api/internal/services/library_service.go
- apps/api/internal/services/library_service_test.go
- apps/web/src/types/library.ts
- apps/web/src/components/library/BatchProgress.tsx
- apps/web/src/components/library/BatchProgress.spec.tsx
- apps/web/src/components/library/LibraryBrowseV2.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- _bmad-output/implementation-artifacts/sprint-status.yaml
