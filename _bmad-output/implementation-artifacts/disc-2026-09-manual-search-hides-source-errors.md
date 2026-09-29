# Story: disc-2026-09-manual-search-hides-source-errors — TMDb 斷線時，手動選片不再怪你打錯字

Status: review

<!-- SM Bob create-story 2026-09-29。Alexyu 從推薦清單選定（「第二張」）；本張不需要產品裁定。
     行號為 main `994baac2`，改動前；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為在「手動選片」對話框搜尋片名的人，
當 TMDb 斷線、沒設金鑰、或被暫時擋下時，
我要看到「搜尋暫時無法使用」，而不是「找不到符合的作品，換個關鍵字試試」——後者讓我以為是自己打錯字。

## 背景（查到的事）

- `apps/api/internal/services/metadata_service.go:676-687`：`ManualSearch` 逐一搜尋來源，出錯只 `slog.Debug` 然後 `continue`；`:719` 最後一律 `return response, nil` → 200＋空結果。
- `apps/api/internal/metadata/orchestrator.go:561-613`：`SearchSource`
  - 來源沒註冊（`:575-580`）或 `!IsAvailable()`（例如沒設 TMDb 金鑰；`:582-587`）→ `return nil, nil`——跟「搜尋成功但沒結果」分不出來。
  - 斷路器開著 → `ErrCircuitOpen`（`:589-601`）；搜尋出錯 → 回傳錯誤（`:604-611`）。
  - `git grep -n "SearchSource(" -- apps/api ':!*_test.go'`：唯一呼叫者是 `metadata_service.go:680`；`git grep -n SearchSource -- '*_test.go'` 無結果（沒有直接測試）。
- Handler `apps/api/internal/handlers/metadata_handler.go:148-196`：service 回錯誤時，`ErrManualSearchInvalidSource` → 400，其餘一律 400 `MANUAL_SEARCH_INVALID_REQUEST`。
- 前端「手動選片」`apps/web/src/components/media/ManualMatchDialogV2.tsx:96` 只搜 `source: 'tmdb'`；`:167-170` 已經有 `search.isError` →「搜尋暫時無法使用，請稍後再試。」分支，`ManualMatchDialogV2.spec.tsx:235-239` 已測；但後端從不回錯誤，所以走不到。
- `useManualSearch`（`hooks/useManualSearch.ts:27-35`）沿用全域 `retry: 1`（`queryClient.ts:5-12`）——錯誤會重試一次後顯示。
- 舊的 `/test/manual-search` 頁（`components/manual-search/ManualSearchDialog.tsx:302-304`）遇錯顯示「搜尋失敗：{message}」，不會壞。
- E2E `tests/e2e/manual-search.api.spec.ts` 期待 200；CI 有設 `TMDB_API_KEY`（`.github/workflows/test.yml:428`、`gh secret list` 有此項），TMDb 可用時行為不變。

## Acceptance Criteria

1. `SearchSource` 在來源沒註冊或 `!IsAvailable()` 時回新的 `metadata.ErrSourceUnavailable`（取代 `nil, nil`）。其他行為不變。
2. `ManualSearch` 記錄每個來源是否**搜尋成功**（`err == nil`）。
   - 至少一個來源成功 → 照舊回 200（只含成功來源的結果；成功但沒結果就是真的空結果）。失敗的來源改用 `slog.Warn` 記錄。
   - **沒有任何來源成功**（全部出錯／不可用／斷路器開）→ 回 `ErrManualSearchSourcesUnavailable`（用 `%w` 包住最後一個來源錯誤，方便記錄）。
3. Handler：`ErrManualSearchSourcesUnavailable` → **503** `MANUAL_SEARCH_SOURCES_UNAVAILABLE`（附 suggestion）。`ErrManualSearchInvalidSource`、其他錯誤的對應不變。
4. 前端零改動：`ManualMatchDialogV2` 的既有錯誤分支會顯示「搜尋暫時無法使用，請稍後再試。」
5. 測試：
   - service（新測試，用假的 provider 註冊進 orchestrator）：只搜 TMDb 且出錯 → 錯誤；TMDb 不可用 → 錯誤；TMDb 斷路器開 → 錯誤；TMDb 成功但空 → 200 空；`all`：TMDb 出錯＋豆瓣成功有結果 → 回豆瓣結果；`all`：TMDb 出錯、豆瓣與 Wikipedia 不可用 → 錯誤。
   - handler：service 回 `ErrManualSearchSourcesUnavailable` → 503＋錯誤碼。
   - 每條先紅後綠。

## Tasks / Subtasks

- [x] Task 1 — `metadata.ErrSourceUnavailable`＋`SearchSource` 改回它（AC #1）
- [x] Task 2 — `ManualSearch` 成功計數與錯誤（AC #2）
- [x] Task 3 — handler 503 對應（AC #3）
- [x] Task 4 — service／handler 測試先紅後綠；`pnpm nx test api`、`pnpm nx test web`、`lint:all` 全綠（AC #4, #5）

## Dev Notes

- **不要**讓「部分來源失敗」變成錯誤：`all` 模式下 TMDb 掛了、豆瓣有結果，使用者仍該看到結果。
- **不要**把錯誤訊息做成新的前端文案（沒設金鑰時「暫時」一詞不準確，另案，見 Discovery Triage）。
- `ErrCircuitOpen` 已是錯誤，自然算「失敗」。
- 頂欄搜尋（`search_service.go`）是**另一條**路徑，刻意保留「TMDb 壞了本地照常」，由 `disc-2026-09-instant-search-tmdb-outage-silent` 追；本張不動。

### Project Structure Notes

- 純後端：`internal/metadata/`（orchestrator＋錯誤）、`internal/services/metadata_service.go`、`internal/handlers/metadata_handler.go`＋測試。後端 4 個 task、前端 0 → 單張。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 立案：`dsr-2b-b` 的 /ship 對抗式 CR
- 同類：dsr-8「符合 0 部」、`disc-2026-09-instant-search-tmdb-outage-silent`

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- RED：先只加 `ErrManualSearchSourcesUnavailable` 變數，新 service 測試 5 條「沒有來源能搜」全紅（回 200 空結果）、2 條「有來源搜過」本來就綠（守舊行為）；handler 測試紅（回 400 `MANUAL_SEARCH_INVALID_REQUEST`）。
- GREEN：`internal/metadata`、`internal/services`、`internal/handlers` 全綠。

### Completion Notes List

- `metadata.ErrSourceUnavailable`（`circuit_breaker.go`，與 `ErrCircuitOpen` 同處）；`SearchSource` 沒註冊／不可用時回它。
- `ManualSearch`：計算搜過的來源數；0 → `fmt.Errorf("%w: %w", ErrManualSearchSourcesUnavailable, lastErr)`；失敗來源改 `slog.Warn`。
- Handler：`errors.Is(err, ErrManualSearchSourcesUnavailable)` → 503 `MANUAL_SEARCH_SOURCES_UNAVAILABLE`。
- 前端零改動：`ManualMatchDialogV2.spec.tsx:235-239` 已測錯誤時顯示「搜尋暫時無法使用，請稍後再試。」
- 全量回歸：`pnpm nx test api` 綠；`pnpm nx test web` 291 檔／4,351 條綠；`lint:all` 0 errors。
- /ship 對抗式 CR：0 HIGH／0 MED／5 LOW；確認 `SearchSource`／`ManualSearch` 各只有一個呼叫者、各 provider「沒找到」都回空結果（不會變 503）、前端與 E2E 不會壞。已修：LOW-1（豆瓣／Wikipedia 預設關閉，每次 `all` 搜尋會多 2 行 Warn → 「不可用」維持 Debug，只有真的出錯才 Warn；且回報原因時優先用真錯誤）、LOW-2（只打空白的查詢會被 provider 拒絕變 503 並計入斷路器 → `Validate` 先 `TrimSpace`，回 400）、LOW-3（503 建議文字不再只提 TMDb）、LOW-5（測試改驗原因有被包住；`ErrSourceUnavailable` 搬到 `orchestrator.go`；Swagger 補 `@Failure 503`）。立案：LOW-4、Wikipedia 殘留（見 Discovery Triage）。新增測試共 9 條。
- 🔗 AC Drift: FOUND — `3-7-manual-metadata-search-and-selection`（手動搜尋）`SearchSource`「Returns nil if source not found or unavailable」（`orchestrator.go` 原註解）→ 回 `ErrSourceUnavailable`；唯一呼叫者 `ManualSearch` 同步改。`manual-search.api.spec.ts` 的 200 期待在 TMDb 可用（CI 有金鑰）時不變。
- 📎 Contract Stamps: NONE（本張新增錯誤碼 `MANUAL_SEARCH_SOURCES_UNAVAILABLE`，回應形狀不變；無上游 stamp）。
- 🎭 A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）。
- 🎨 UX Verification: SKIPPED — no UI changes（使用既有錯誤文案）。

### Discovery Triage

- ③ backlog-with-carry-forward-link — 沒設 TMDb 金鑰時，手動選片會說「搜尋暫時無法使用，請稍後再試」，但這不是「暫時」的問題，要去設定填金鑰 → `disc-2026-09-manual-search-no-key-copy`（P4，建單時立案）。
- ③ — CR LOW-4：503 會被 `retry: 1` 重試，TMDb 卡住時錯誤訊息最久約 3 分鐘才出現 → `disc-2026-09-manual-search-error-slow-retry`（P3）。
- ③ — CR：Wikipedia `fetchFullMetadata` 失敗仍回空結果 → `disc-2026-09-wikipedia-detail-failure-reads-empty`（P4）。

### File List

- apps/api/internal/metadata/orchestrator.go
- apps/api/internal/services/metadata_service.go
- apps/api/internal/services/metadata_manual_search_sources_test.go（新）
- apps/api/internal/handlers/metadata_handler.go
- apps/api/internal/handlers/metadata_handler_test.go
- _bmad-output/implementation-artifacts/disc-2026-09-manual-search-hides-source-errors.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/3-7-manual-metadata-search-and-selection.md（AC drift reference — see Completion Notes；未修改）

## Change Log

| Date       | Change                            |
| ---------- | --------------------------------- |
| 2026-09-29 | create-story（SM Bob）：建單；立案 `disc-2026-09-manual-search-no-key-copy`。 |
| 2026-09-29 | dev-story（Amelia）：`ErrSourceUnavailable`＋`ManualSearch` 成功計數＋503；8 條新測試（6 條先紅後綠）；全量回歸綠 → review。 |
| 2026-09-29 | /ship 對抗式 CR：修 4 個低嚴重度問題（log 雜訊、空白查詢、建議文字、測試／文件）；立案 2 條。 |
