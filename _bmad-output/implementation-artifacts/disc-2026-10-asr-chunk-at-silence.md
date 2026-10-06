# Disc：聽聲音的音檔改成兩分鐘一段、在靜音處切——一段被配樂帶崩不會拖垮整集

Status: done

**Source:** 2026-10-06／07 第三次實測（`eval-see-s01e02-asr-vs-official.md` 第三節）：10 分鐘片段整個一次送，前 3 分半的配樂把 Whisper 帶進「只聽到音樂」模式，整段 0 句對白（字級時間開著 4 次崩 3 次；舊版也有 9 秒漂移）。Alexyu 2026-10-07 選 A 的第 2 步。

## Story

身為 Vido 的使用者，
我希望一集裡配樂多的那幾分鐘就算讓語音辨識糊塗了，也只影響那兩分鐘，
這樣不會因為片頭配樂整集沒字幕或整集時間漂掉。

## 查到的事

- 原本只有 WAV > 24 MiB（約 13 分鐘）才切，切成 600 秒固定格；10 分鐘以下整個一次送。
- Whisper 內部是 30 秒一窗、前一窗的文字會餵給下一窗；配樂一旦被「聽」成假句子或 ♪♪ 就一路傳下去。切短＝強制重置。
- 計費按秒，請求數變多不加錢（57 分鐘：6 次 → 29 次）。
- 快取 key 用「第 i 格的名目起點」（`Start = i × grid`），manifest 記 `chunk_seconds` 與 `done` 起點列表；估價與續跑都靠它。

## 設計

- `WhisperChunkDuration` 600 → **120**（名目格，只決定快取身分與段數）。
- `NeedsChunking`：大小超過預算 **或** 長度超過一格就切。
- `SplitAudioChunks` 回 `[]AudioChunk{Path, StartMS, DurationMS}`：先跑一次 `ffmpeg … -af silencedetect=noise=-35dB:d=0.4 -f null -` 掃整檔（幾秒），每條格線往前後 20 秒內找最近的靜音、切在它正中間（`planCuts`，純函式）；找不到就切在格線上。尾巴不到 30 秒併入前一段。ffmpeg 失敗＝「不知道靜音在哪」→ 退回純格線，不會讓執行失敗。
- `MergeSRTChunks(srts, startsMS)`：每段用自己真正的起點（毫秒）位移，不再是 `i × 600`。
- 快取值多存 `start_ms`；命中但切點不同 → 當作沒命中重聽（避免切點規則改了之後拿舊的相對時間亂接）。
- 段數 = ceil(長度／120)，尾巴 <30 秒時 −1；快取身分仍是「第 i 格」。

## Acceptance Criteria

1. 10 分鐘片段 → 5 段；每段起點在格線 ±20 秒內的靜音正中間（log `audio split into chunks at silences chunks=5 grid_seconds=120`）。
2. 合併後的字幕時間連續、以毫秒位移（第 2 段 0.5 秒的句子落在「切點＋0.5 秒」）。
3. silencedetect 失敗或沒有靜音 → 切在格線上，執行照常。
4. 續跑：同檔同切點的段落從快取拿；切點不同的段落重聽並留 log。
5. `go test ./...`、vet、staticcheck、lint:all 全綠。
6. **實測（NAS 10 分鐘片段，約 $0.3）：** 字級時間關／開各跑 2 次。看：對白句數不再 0（崩只限一段）、「出現在沒人說話時」、8:17 那句時間。開著不崩 → 另立單把 `VIDO_ASR_WORD_TIMESTAMPS` 預設翻成 true。

## Tasks / Subtasks

- [x] T1 常數、NeedsChunking、AudioChunk、silencedetect 解析、planCuts（AC #1、#3）
- [x] T2 MergeSRTChunks 毫秒位移、service 接線、快取 start_ms（AC #2、#4）
- [x] T3 測試：parse／planCuts 邊界／切點跟著靜音／毫秒合併／長度觸發切分／續跑夾具改 120 格／預算測試改 79 段（AC #5）
- [x] T4 NAS 實測（AC #6）

## Dev Notes

- 不要把切點做成「真的在靜音處才切」而沒有上限：沒靜音就切格線，段長上限 = 120＋2×20 秒，位元組遠低於預算。
- 這張做完才輪到 `disc-2026-10-asr-vad-pretrim`（只送有人說話的段落）；它需要的「每段自己的起點」管線就是這張鋪的。
- 既有測試改動（刻意）：`TestSplitAudioChunks_LargeFile_ChunksUnderLimit` 852 秒 → 7 段（12 秒尾巴併入）；`HighByteRate` 夾具改 384 KB/s 才會縮格；續跑測試 16 段 × 120 秒；預算測試 157 分鐘現在是 79 次呼叫，ASR 假延遲 300 ms → 10 ms、每分鐘預算 10 ms → 30 ms。
- 測試裡假的 ffmpeg shim 會把「最後一個參數」當輸出檔——silencedetect 的最後一個參數是 `-`，shim 要略過，否則會在套件目錄留下一個叫 `-` 的檔。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **AC #6 實測（2026-10-07，eval 第四節）：** 切點 121／240／359／477 秒都貼到靜音；4 次都沒崩（字級時間關 82／82 句，開 75／68 句）；字級時間開著 p90 0.65～0.74 秒、沒人說話處 6～13%。→ 預設翻 true（`disc-2026-10-asr-word-timestamps-default-on`）。
- **Adversarial CR（2026-10-07，fresh agent）1 MH／1 M／2 L＋nits，全部修掉：**
  - MH 片尾規則（R3）原本每一段都當「片尾」跑，120 秒格＋切在停頓處＝每段最後幾句都是「停頓前的輕聲對白」，會被吃掉 → 加 ctx 旗標 `ai.WithMidFileChunk`，只有最後一段跑 R3；`filterHallucinationsWith(segs, applyTail)`＋測試。
  - M 快取只比起點：第 0 段永遠從 0 開始，第一刀位置變了也會拿舊字幕 → 快取值多存 `duration_ms`，起訖都一樣才重用。
  - L `-ss` 放在 `-i` 後面是 output seeking，79 段會從頭解碼 79 次（157 分鐘約 12 GB 讀取）→ 改成 `-ss … -i`（PCM 一樣精確）。
  - L 格子被位元率縮到 <40 秒時切點可能不遞增 → 窗口 = min(20 秒, 格/3)，保護分支改 `max(nominal, prev+0.4s)`。
  - nits：silencedetect 檔頭的 `-0.0001` 夾成 0、科學記號解析、三處過時註解。
  - 已知不修：升版前在 600 秒格中斷的片，舊 manifest 會被新的覆寫、舊鍵靠 30 天 TTL 自然過期、`HasResumeProgress` 可能誤報一次（一次性遷移成本）。
- 🔗 AC Drift: FOUND — 9R-3「依大小切 600 秒」→ 依大小或長度切 120 秒並貼靜音；sub-6 續跑的 manifest 語意不變（名目格）。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

- 見 sprint-status：`disc-2026-10-asr-vad-pretrim`（設計筆記已補）、`disc-2026-10-series-credits-empty`（根因已查）。

### File List

- `apps/api/internal/ai/whisper.go`、`whisper_segments.go`、`asr_prompt.go`、`whisper_test.go`、`whisper_segments_test.go`、`whisper_chunk_error_test.go`（改）
- `apps/api/internal/services/transcription_service.go`、`asr_chunk_store.go`、`transcription_chunk_resume_test.go`、`transcription_run_budget_test.go`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-chunk-at-silence.md`（新）、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 實測通過（4/4 沒崩）→ done；字級時間預設翻 true 另立單。 |
| 2026-10-07 | CR 1MH/1M/2L 修掉：R3 只在最後一段、快取比起訖、input seeking、窗口守門。 |
| 2026-10-07 | create-story＋dev 同日：120 秒格＋靜音切點＋毫秒合併＋快取 start_ms；全綠，狀態 review，待 NAS 實測。 |
