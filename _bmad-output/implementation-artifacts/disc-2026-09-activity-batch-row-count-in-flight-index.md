# Disc：活動頁的批次數字跟批次視窗對不起來

Status: done

**Source:** `disc-2026-09-activity-batch-row-count-in-flight-index`（2026-09-17 由 dsr-6d create-story 稽核發現，P3）。2026-10-13 Alexyu 從 backlog 選 1。

## 查到的事（2026-10-13，Bob，main `7494dfe3`）

- 活動頁「批次生成」那一列右邊寫「3 / 10」，進度條用 `percentDone`。兩個都來自 `GenerationBatchProcessor.ActivityProgress()`（`apps/api/internal/services/generation_batch.go:408`）。
  - `current` = `CurrentIndex`，意思是**正在做第幾部**（從 1 算）。
  - `percentDone` = `CurrentIndex × 100 ÷ 總數`。第 1 部剛開始就是 10%。
- 同一批在別的地方都是算「已完成」，也就是成功＋失敗：
  - 批次視窗：「已完成 2 / 10」（`GenerationBatchDialogV2.tsx` 的 `processed`）。
  - 生成工作區：上方總進度那一條。
- 前端（`ActivityHub.tsx:146-150`）只把 `current / total` 原樣印出來，沒有文字說明它是什麼意思。所以**只改後端**，數字就跟另外兩處一致，畫面文字不用動，也不用設計稿。
- 另一個舊的 `subtitle.BatchProcessor`（字幕搜尋批次，`subtitle_batch`）也用 `CurrentIndex`。它自己的視窗也是這樣算，兩邊本來就一致，這張不動。

## Story

身為看活動頁追進度的人，
我希望活動頁的批次數字跟批次視窗一樣是「已完成幾部」，
這樣兩個地方不會一個寫 2、一個寫 3，進度條也不會在第一部剛開始就跑到 10%。

## Acceptance Criteria

1. `GenerationBatchProcessor.ActivityProgress()` 回的 `current` = `SuccessCount + FailCount`（已完成，跟批次視窗同一個算法）。
2. `percentDone` = 已完成 × 100 ÷ 總數。第 1 部剛開始時是 0。
3. `active`、`total`、`currentItem`（正在做的那一部的片名）不變。活動列下面那行「正在做哪一部、第幾段」的說明不受影響。
4. 測試：
   - 第 1 部正在跑：`current` 0、`percentDone` 0、`currentItem` 是第 1 部。
   - 第 1 部成功、第 2 部失敗、第 3 部正在跑（共 4 部）：`current` 2、`percentDone` 50。
   - 原本寫死「剛開始 current == 1」的兩個測試改成新的意思。
5. 檢查：`go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 `ActivityProgress` 改算法（AC #1–#3）
- [x] T2 測試（AC #4）
- [x] T3 檢查（AC #5）

## Dev Notes

- 不動 `subtitle.BatchProcessor`（理由見上）。
- 不動前端。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Debug Log References

- 突變測試：把 `current` 改回 `CurrentIndex`，兩條新測試都變紅。還原後全綠。

### Completion Notes List

- `ActivityProgress` 的 `current` 改成 `SuccessCount + FailCount`。`percentDone` 本來就從 `current` 算，所以一起變對。
- 原本寫死「剛開始 current == 1」的兩個測試改成 0。
- 新測試：
  - 4 部中 A 成功、B 失敗、C 正在跑 → 2 / 4、50%、片名 C。
  - 第 1 部剛開始 → 0、0%。
- 檢查全綠：`go build ./...`、`go vet`、`go test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`（0 errors）、`format:check`。
- 順手關了兩張早就修好的單（Alexyu 同意）：`disc-2026-09-manage-subtitle-running-job-not-attached`（#673）、`backlog-metadata-cast-both-legs`（#630）。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／1 Low，接受。

- **L1（接受）**：舊的字幕搜尋批次（`subtitle_batch`）在活動頁還是「正在做第幾部」。它自己的視窗也這樣算，兩邊一致，而且那個視窗目前沒有掛在任何頁面上（`disc-2026-10-more-unmounted-components`）。不動。

### File List

- `apps/api/internal/services/generation_batch.go`
- `apps/api/internal/services/generation_batch_activity_test.go`（新）
- `apps/api/internal/services/generation_batch_test.go`
- `apps/api/internal/services/generation_batch_items_test.go`
- `_bmad-output/implementation-artifacts/disc-2026-09-activity-batch-row-count-in-flight-index.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
