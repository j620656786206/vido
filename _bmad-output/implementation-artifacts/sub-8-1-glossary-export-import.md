# Story 8.1: 詞彙表匯出／匯入（檔案）—— 驗證「共享」有沒有人要（全端，小）

Status: ready-for-dev

**Depends on:** sub-7-1（scope 綁 TMDb ID；沒有共同 ID 匯入對不上）。B 路線第一步；party-mode 裁定「先做最笨的：匯出一個檔，貼給朋友匯入」。

## Story

As a NAS owner with a friend who watches the same show,
I want to export a show's glossary as a file and import theirs,
so that we stop translating the same characters two different ways — before anyone builds a server for it.

## Acceptance Criteria

1. **格式 `[@contract-v1]`。** JSON：`{format:"vido-glossary", version:1, exported_at, scope:"tmdb:tv:66732", title, language, terms:[{term_src, term_zh, source, confirmed}]}`。只匯出 `tmdb:*` scope；`local:*` 回 409 `GLOSSARY_NOT_SHAREABLE`（新碼，`SUBTITLE_` 前綴下或新 `GLOSSARY_` 前綴——Rule 7 裁定：**用 `GLOSSARY_`**，因為 8-2 還會加碼；同步 code-review instructions.xml 前綴清單，prefix 數 17→18）。
2. **端點。** `GET /media/:id/glossary/export`（下載）、`POST /media/:id/glossary/import`（multipart 或 JSON body；scope 不符 → 400 `GLOSSARY_SCOPE_MISMATCH`；同詞已存在且 `confirmed=1`／`manual` → 跳過不覆寫；其餘 insert-if-absent，`source=community`，`confirmed=0`）。回 `{imported, skipped, conflicts:[{term_src, mine, theirs}]}`。
3. **UI。** `GlossaryPanelV2` 工具列加「匯出」「匯入」；匯入後顯示結果摘要與衝突列表（保留我的／改用他的，逐筆或全部）。
4. **測試。** 匯出 shape；匯入三種情況（新增／跳過／衝突）；scope 不符；`local:*` 拒絕；FE spec + fixture。

## Tasks / Subtasks

- [ ] **Task 1 — 格式 + 端點 + 錯誤碼（AC: #1, #2）**
- [ ] **Task 2 — UI（AC: #3）**
- [ ] **Task 3 — 測試（AC: #4）**

## Dev Notes

- 這支故意**不做**任何網路傳輸。用了才知道值不值得做 8-2。
- 記錄使用：匯出／匯入次數進 log（無遙測）。

### Time-dependent visual coverage

- N/A。

### References

- eval-1 backlog P2-1；party-mode John「先匯出一個檔」

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
