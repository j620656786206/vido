# Story 6.4: `FilterSDH` 丟掉純音符 cue —— 不再把 ♪ 送給 LLM 付錢再被閘門退回（後端）

Status: ready-for-dev

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

- [ ] **Task 1 — 規則（AC: #1, #2）**：`sdh_filter.go` `isMusicOnly(s)` + 接入 `isWholeLineAnnotation`；檔頭註解補 eval-1 引用
- [ ] **Task 2 — 測試（AC: #3）**

## Dev Notes

- 三行的修改，但要保守：不要順手把「開頭有音符」也丟掉——那是 AC #4 明文的 under-strip 取捨。
- 純後端、無契約影響。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「發現 8」；後續 Backlog P0-4；`apps/api/internal/subtitle/sdh_filter.go` `isMusicLine`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
