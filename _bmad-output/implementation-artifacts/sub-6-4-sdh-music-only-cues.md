# Story 6.4: `FilterSDH` 丟掉純音符 cue —— 不再把 ♪ 送給 LLM 付錢再被閘門退回（後端）

Status: done

## Story

As a BYOK NAS owner,
I want cues that contain nothing but music marks to be dropped before translation,
so that an SDH track stops costing me ~40 wasted LLM cues per episode that the quality gate then rejects as "echoed".

## Context — 這個 story 為什麼存在

eval-1 發現 8：Lioness（SDH 軌）兩個模型各約 40 句「保留英文」，實看幾乎全是純 `♪` cue。`isMusicLine`（`sdh_filter.go`）要求**頭尾都是同一音符且長度 ≥ 2**，所以單一 `♪`、`♪♪`、`♪ ♪`（頭尾同但中間空白）中只有部分被抓到；單一 `♪` 因 `len(runes) < 2` 直接漏網。

## Acceptance Criteria

1. **純音符行是 annotation。** `isWholeLineAnnotation` 新增規則：去除空白後，若整行只由 `musicMarks`（`♪`、`#`）與空白組成（長度 ≥ 1），視為 annotation 丟棄。`#` 單獨一行也算（SDH 慣例）；**含任何非音符字元**的行維持既有規則（保守）。

2. **既有行為不變。** `♪ dramatic music ♪` 仍照舊丟；`♪ lyrics` 開頭單邊仍保留（AC #4 保守讀法不動）；speaker label 規則不動；P7 index/timestamp 不重編。

3. **測試。** `sdh_filter_test.go` 加：`♪`、`♪♪`、`♪ ♪`、`#`、` ♪ `（帶空白）→ 丟；`♪ lyrics`、`lyrics ♪`、`a ♪ b` → 留；混合多行 cue 中只有音符行被去、其餘行保留。

## Tasks / Subtasks

- [x] **Task 1 — 規則（AC: #1, #2）**：`sdh_filter.go` `isMusicOnly(s)` + 接入 `isWholeLineAnnotation`；檔頭註解補 eval-1 引用
- [x] **Task 2 — 測試（AC: #3）**

## Dev Notes

- 三行的修改，但要保守：不要順手把「開頭有音符」也丟掉——那是 AC #4 明文的 under-strip 取捨。
- 純後端、無契約影響（CR H1 修正：router `betterCandidate` 以過濾後 cue 數排序，本 story 改變 SDH 軌的排名——這是**預期**行為，已加 router 測試鎖住）。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「發現 8」；後續 Backlog P0-4；`apps/api/internal/subtitle/sdh_filter.go` `isMusicLine`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Completion Notes List

- `isMusicOnly(s)`：整行只由 `musicMarks`（`♪`、`#`）與空白／tab 組成且至少一個記號 → annotation；接入 `isWholeLineAnnotation`。`isMusicLine`（頭尾同記號、≥2 runes）與 AC #4 的 under-strip 取捨（`♪ lyrics` 保留）完全不動。
- 測試：`TestFilterSDH_Rules` 表加 10 列（`♪`／`♪♪`／`♪ ♪`／`#`／帶空白／混合記號 → 丟；`♪ lyrics`／`lyrics ♪`／`a ♪ b` → 留；多行 cue 只去音符行）。RED 先失敗、GREEN 後 `go test ./internal/subtitle` 全過。
- 🔗 AC Drift: NONE (checked: 'isMusicLine|music mark|FilterSDH' across _bmad-output/implementation-artifacts/*.md — 3 hits; sub-1-4 AC #4 的「`♪…♪` 包住的行丟、單邊 `♪ lyrics` 留」兩條都仍成立，本 story 只新增「純記號行」這一類，REUSE not DRIFT)
- 📎 Contract Stamps: NONE (no [@contract-v*] stamps in this story or upstream refs — sub-1-4 為 pre-Rule-20 implicit v0)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🔌 Route Sync: N/A (no backend route touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- 全回歸：`pnpm nx test api` ✅、`pnpm nx test web` 255 files / 3125 tests ✅、`test:cleanup` 無殘留；`pnpm lint:all`：go vet／staticcheck 過，eslint 0 errors（126 warnings 為 retro-11-AI1b 既有批次），prettier 唯一紅字是未追蹤的 `testsprite_tests/testsprite-mcp-test-report.html`（不在 repo）。

### Discovery Triage

- ① expand-scope-in-place — CR M4：`♫`／`♬`（U+266B／266C）不在 `musicMarks`，一併納入（Task 1 擴大、測試列新增）。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `sdh_filter.go` 加 `isMusicOnly`／`isMusicMark`，`isWholeLineAnnotation` 納入純音符行（eval-1 發現 8）。 |
| 2026-09-04 | Task 2 — `sdh_filter_test.go` 規則表 +10 列（正反例與多行）。 |
| 2026-09-04 | CR fixes — `isMusicOnly` 以 `unicode.IsSpace` + ZWSP/ZWNJ/BOM 判空白（H2）；`musicMarks` 加 ♫ ♬（M4）；`isMusicMark` 改 `strings.ContainsRune`（L7）；測試列標明 RED／regression guard（M3）、加 `JOHN: ♪`／`[JOHN]: ♪`／nbsp／ZWSP／全形空白／`#1 fan`（H2、M5）、`AllSDH` 加純 ♪ cue；`router_test.go` 加 `MusicOnlyCuesDoNotInflateCandidateRank`（H1）；doc 註解修正（L6、L8）。 |

### File List

- `apps/api/internal/subtitle/sdh_filter.go`（modified）
- `apps/api/internal/subtitle/sdh_filter_test.go`（modified）
- `_bmad-output/implementation-artifacts/sub-6-4-sdh-music-only-cues.md`（this file）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（status）

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → all items resolved in-session (branch `fix/sub-6-4-5-7-code-review`) → **Approve**

Mandatory checks: Rule 7 PASS（0 codes）· Rule 20 N/A · Rule 25 N/A。

### Action Items

- [x] [H1] Router 排名受影響未測 — `router_test.go` 新增 `TestSelectAndRoute_MusicOnlyCuesDoNotInflateCandidateRank`；Dev Notes 改述。
- [x] [H2] `isMusicOnly` 只認 ASCII 空白 — 改 `unicode.IsSpace` + U+200B/U+200C/U+FEFF；測試 nbsp／ZWSP／U+3000。
- [x] [M3] 10 列中 6 列本來就綠 — 分成「RED 前四列」與「regression guards」兩段標明。
- [x] [M4] `♫`／`♬` 漏網 — 加入 `musicMarks`，測試各一列；Discovery Triage 記 lane ①。
- [x] [M5] 標籤剝除後的二次判定無測試 — 加 `JOHN: ♪`、`[JOHN]: ♪`。
- [x] [L6] 註解不準 — 重寫 `isMusicOnly` doc，AC 引用改「sub-1-4 AC #4」。
- [x] [L7] `isMusicMark` 重複 — 移除，改 `strings.ContainsRune(string(musicMarks), r)`。
- [x] [L8] 測試表頭混用 — 改「Per-rule table (sub-1-4 AC #4; sub-6-4 rows marked)」。
