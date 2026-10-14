# Disc：聽聲音生成的字幕每句一閃就過——最少亮 1 秒、依字數加長、碎句併入下一句

Status: review

**Source:** 2026-10-14 正式機驗證（See S01E02 前 10 分鐘片段，ElevenLabs Scribe、字級時間）。Alexyu 一開始的回報就是「字幕一閃就過」。數字：79 句裡 41 句顯示不到 1 秒、5 句不到半秒、1 句 0 秒，中位數 0.94 秒。本機用一套規則試修後（最少 1 秒、每字 0.25 秒、不蓋到下一句、0.3 秒以下碎句併入下一句）→ 不到 1 秒剩 4 句、中位數 1.74 秒；Alexyu 播放後：「OK」。

## 查到的事（2026-10-14，main `01ba0b5d`＋#793）

- 切句在 `apps/api/internal/ai/scribe.go:461` `scribeCues`：每句的起訖＝第一個字開始到最後一個字結束（字級時間）；`:533-540` 短於 `scribeMinCueSeconds`（`:70`＝0.6）才延長、延到下一句開始為止。
- 為什麼還是短：Scribe 的字級時間很準，一句「Hey.」只有 0.18 秒；延長上限 0.6 秒也不夠人讀；而且下一句常緊接著開始（gap < 0.6），延長被下一句擋住，結果連 0.6 都到不了（`:535-537`）。
- 0 秒那句：`:522` 一個字的 start==end（Scribe 偶爾給），延長又被下一句擋 → 0。
- 對照：官方英文字幕同一段 79 句的時長分布完全一樣？不是——`verify-scribe` 資料夾裡的 `.en.srt` 是 Scribe 自己的英文（同一套切句），所以數字相同；真正的官方軌（`eval/asr-exam-2026-10/answer.en.srt`）中位數約 1.2 秒以上、沒有 0 秒句。
- 閱讀速度的常識：英文字幕約 15–17 字／秒（0.06–0.07 秒／字母）、中文約 4–5 字／秒；Vido 切句時只有英文，翻成中文後字數約為英文字母數的 0.5–0.6 倍，所以用「英文寬度 × 0.12 秒」估中文閱讀時間（CJK 寬度算 2，`displayWidth`）。
- Whisper 路徑有自己的 `minTightenedCueSeconds`（`whisper_segments.go`），本單不動。

## Acceptance Criteria

1. 每句最少亮 `scribeMinCueSeconds`＝**1.0 秒**（0.6→1.0）。
2. 依字數加長：目標時長＝max(1.0, 顯示寬度 × 0.12 秒)，上限 `scribeCueMaxSeconds`（7 秒）；只延長、不縮短；延長不蓋到下一句（留 0.08 秒）。
3. 碎句併入：一句不到 0.3 秒、且下一句在 0.6 秒內開始 → 併進下一句（文字接在前面、起點用碎句的起點），不再出現 0 秒句。
4. 既有行為不變：句尾／停頓 0.8 秒／音效／7 秒／84 寬的切法；CJK 不加空格；一個拖很長的字仍截到 7 秒。
5. 測試：`TestScribeCues_ShortCueIsHeldButNeverIntoTheNext` 改成新數字；新增「依寬度加長」「碎句併入」「0 秒字不再產生 0 秒句」三條；`go test ./internal/ai/`、vet、staticcheck 綠。
6. 正式機驗證：同一段重聽（跟 scribe-keyterms 一起跑），不到 1 秒的句子 ≤5 句、沒有 0 秒句、中位數 ≥1.5 秒。

## Tasks / Subtasks

- [x] T1 `scribe.go` 常數與後處理（AC 1–4）
- [x] T2 測試（AC 5）
- [ ] T3 NAS 驗證（AC 6，合併後）

## Dev Notes

- 翻譯後的中文可能比估的長，但 Vido 在切句時沒有中文；0.12 秒／寬度是保守值，Alexyu 看過本機試修（每中文字 0.25 秒）說 OK，兩者對同一句算出來的時長接近。
- 不碰 `whisper_segments.go`。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- `scribe.go`：`scribeMinCueSeconds` 0.6→1.0；新增每寬度 0.12 秒的閱讀時間、0.08 秒 lead、0.3 秒碎句在 0.6 秒內併入下一句；只延長不縮短。
- 測試：三條既有測試改新數字（含真實回應那條改成「停在 lead 之前或自然結束在 lead 內」）；新增依寬度加長、碎句併入（含 0 秒字）兩條。`go test ./internal/ai/`、vet、staticcheck 綠。

### Discovery Triage

N/A — no out-of-scope work discovered.

### File List

- `apps/api/internal/ai/scribe.go`、`apps/api/internal/ai/scribe_test.go`
