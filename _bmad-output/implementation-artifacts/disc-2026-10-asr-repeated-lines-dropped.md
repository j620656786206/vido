# Disc：聽聲音時，劇情裡真的連喊三四次的台詞不再被當成幻聽刪掉

Status: review

**Source:** See S01E02 10/5 聽聲音實測：3:48「拜託哭吧」、4:21–4:37 連續「轉過來看我」、7:46–7:58 連喊三次「謝拉馬威」都沒字幕；第 1 段 log `dropped_by_reason=map[repeat_run:28]`、drop_ratio 0.26。10/6 走片內字幕後都在；只能聽聲音的片仍會被刪。

## Story

身為 Vido 的使用者，
我希望角色真的連喊三次的台詞，聽聲音生成時也會出現三次，
這樣劇情最緊張的那幾句不會剛好消失。

## 查到的事（2026-10-06，逐一開檔確認）

- `ai/whisper_segments.go` `hallucinationRepeatRun = 3`：連續 3 段以上文字相同（去大小寫與尾標點）就只留第一段，其餘記 `repeat_run`（`filterHallucinations` R2b，`:191-206`）。
- 第 1 段的 28 句 `repeat_run` 有兩種來源混在一起：配樂時連續的 `♪♪`（已由 `disc-2026-10-asr-music-only-cues` 的 `music_only` 規則接手）、以及真的連喊的台詞。
- 真的卡住的解碼器會把同一句吐很多次（遠超過四次）；三到四次更像有人在喊。
- 10/5 的原始 verbose_json 已被 `clearChunkCache` 清掉，無法回放，所以這個門檻是**憑一集的現象盲調**，要靠 10 分鐘片段實測確認。

## 設計

`hallucinationRepeatRun` 3 → 5；其他規則不動。既有測試的夾具從 4 段改成 6 段（期望「第一段留下、其餘刪」不變），新增「連喊三次／四次都保留」測試。

## Acceptance Criteria

1. 連續 5 段以上相同文字：第一段留下、其餘 `repeat_run`。
2. 連續 3 段或 4 段相同文字：全部保留（See 的兩個案例）。
3. 2 段相同仍保留（既有）。
4. `go test ./...`、vet、staticcheck、lint:all 全綠。
5. **實測（隨「修一半」三張用 10 分鐘片段驗證，約 $0.3）：** 3:48／4:21–4:37／7:46–7:58 的句子出現在聽聲音產出的字幕裡；第 1 段 `repeat_run` 數明顯下降。若實測顯示真的解碼迴圈只有 3～4 次，再回調。

## Tasks / Subtasks

- [x] T1 門檻 3 → 5＋註解（AC #1–#3）
- [x] T2 測試夾具與新測試（AC #1–#3）
- [x] T3 檢查（AC #4）

## Dev Notes

- 不要用「兩句之間有沒有空白」判斷：whisper 段落時間本來就首尾相接（見 `disc-2026-10-asr-coarse-timestamps`），用它判斷不可靠。
- 不要把 R2a（單段壓縮比）也放寬：那是抓單一段內部的卡住，跟這題無關。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **測試夾具調整（刻意）：** `TestFilterHallucinations_R2RepeatRunKeepsTheFirst` 的夾具從 4 段「Thank you」改成 6 段，期望（第一段留、其餘 `repeat_run`）不變。
- 🔗 AC Drift: FOUND — 9R-5 AC #3「連續相同 ≥3 視為迴圈」→ 本單改為 ≥5；理由與證據見上。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

- **CR 1M/2L（2026-10-06，fresh agent）**：M1 新測試的「Face me!」夾具第一句多了「Hey! 」，實際只是 2 連，沒測到「剛好 3 連保留」→ 改成三句一樣；另加「剛好 5 連才刪 4 留 1」測試。
- 🔎 **L1／L2 既有設計、非本單引入，記下不修**：R2b 算連續時會把已被 R0／R1 標掉的同文字段落（`silence`、`music_only`）也算進去，4 句真喊＋1 句被標 silence 的同句就湊成 5 連；`filterPromptEcho` 是在 `filterHallucinations` 之前**移除**段落，兩段喊叫中間若夾一句名單回音，移掉後也會接成一條。兩者都要 whisper 剛好在喊叫中間吐出東西，門檻拉到 5 後更罕見。→ 立 `disc-2026-10-asr-repeat-run-bridging`（backlog，P3），修法是把 echo 改成在 `filterHallucinations` 裡當 R0 標記、R2b 只數沒被標的段落。

### File List

- `apps/api/internal/ai/whisper_segments.go`（改）、`apps/api/internal/ai/whisper_segments_test.go`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-repeated-lines-dropped.md`（新）、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | CR 1M/2L：夾具改成真的 3 連、加剛好 5 連測試；L1／L2 立 disc-2026-10-asr-repeat-run-bridging。 |
| 2026-10-06 | create-story＋dev 同日：門檻 3 → 5；夾具調整＋新測試；全綠，狀態 review。 |
