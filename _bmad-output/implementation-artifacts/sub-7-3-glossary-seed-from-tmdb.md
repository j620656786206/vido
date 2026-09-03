# Story 7.3: 掃描當下用 TMDb 角色／演員名播種 `show_glossary`（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want a show's glossary to already know its characters before the first episode is translated,
so that name consistency does not depend on the model getting lucky in episode one.

## Context

eval-1 發現 13：`show_glossary` 261 筆 `source` 全是 `subtitle`，schema 允許的 `metadata` 播種**從未發生**。這是修「同一人名一下英文一下中文」最直接的槓桿，也是「A 路線第一天就有料」的加速器①。

**前置：** sub-7-1（scope 綁 TMDb ID）、sub-7-2（cast 進 context）。

## Acceptance Criteria

1. **播種時機。** enrichment 寫入 credits 後（movie／series 各一處）呼叫 `GlossarySeeder.SeedFromCredits(ctx, scope, credits)`；insert-if-absent（既有詞不覆寫，`InsertNew` 語意）。
2. **播什麼。** 每位 cast：`term_src=Character`（英文角色名）→ `term_zh`＝TMDb zh-TW 的角色名（`GET /credits?language=zh-TW`；若 TMDb 回簡體或英文則 **OpenCC s2twp 轉繁**；仍是英文／空 → **不播**，不捏造）；演員本名同樣一筆（`source=metadata`，`confirmed=0`）。上限 `MetadataCastLimit`（10）。
3. **不放雜訊。** 角色名為「Self」「Narrator」「(voice)」等泛稱、或長度 > 40、或含檔名形狀 → 跳過（表格驅動，測試列舉）。
4. **重掃冪等。** 同 scope 重跑播種不產生重複（unique index + NOCASE）；使用者已改過的詞（`confirmed=1` 或 `source=manual`）永不被覆寫。
5. **可觀測。** 每次播種 `slog.Info` 帶 `scope`、`seeded`、`skipped`；`GlossaryPanelV2` 因 sub-7-1 已能顯示「中繼資料」徽章，無 FE 改動。
6. **測試。** (a) zh-TW 角色名直用；(b) 簡體 → 繁；(c) 英文角色名不播；(d) 泛稱過濾表；(e) 冪等與不覆寫；(f) enrichment 呼叫點各一條整合測試。

## Tasks / Subtasks

- [ ] **Task 1 — TMDb zh-TW credits 取得（client 若已支援 language 參數則沿用）（AC: #2）**
- [ ] **Task 2 — `GlossarySeeder` + 過濾表（AC: #2, #3）**
- [ ] **Task 3 — enrichment 接線 + 冪等（AC: #1, #4）**
- [ ] **Task 4 — 測試（AC: #6）**

## Dev Notes

- Rule 27：走既有 `internal/tmdb` client（限流／快取／降級都在），不新開 client。
- 與 sub-7-5 共用 `GlossarySeeder` 介面（來源不同：TMDb vs 官方字幕）。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「發現 13」「加速器①」；`glossary_store.go` `InsertNew`；`enrichment_service.go`（credits 寫入處）

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
