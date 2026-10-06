# Disc：聽聲音生成的字幕時間改用「逐字時間」，不再比演員開口早好幾秒

Status: done

**Source:** See S01E02 10/5 聽聲音實測：659 句裡 93% 長度是整秒、420 句一句緊接一句，637 句對白有 **138 句出現在沒人說話時**（對官方英文軌），8:15 就出字幕、8:21 才開口。10/6 走片內字幕後是 0 句；只能聽聲音的片仍會這樣。

## Story

身為 Vido 的使用者，
我希望聽聲音生成的字幕在演員開口時才出現、說完就消失，
這樣不會提早劇透、也分得清是誰在說話。

## 查到的事（2026-10-06，逐一開檔確認）

- `ai/whisper_segments.go:87-106` `segmentsToSRT`：一段＝一句，start／end 直接用 whisper 的段落時間；註解寫明刻意 1:1、不重切。
- whisper 的段落時間粗（整秒、首尾相接）；`timestamp_granularities[]=word`（verbose_json 才有）會多回一個扁平的 `words` 陣列，每個字有自己的 start／end；要同時列 `segment`，否則 OpenAI 不回 `segments`。
- `whisper.go:263-300` `transcribeVerbose`：verbose_json 被 4xx 就永久退回 `srt`（`markVerboseUnsupported`）。自架引擎（Speaches 等）可能不認識新欄位而 4xx——**不能**讓它因此永久失去幻覺過濾。
- `buildTranscribeBody`（`:388-`）在 #697 後已有 `prompt` 參數。

## 設計

- 請求：verbose_json 時加 `timestamp_granularities[]=word` 與 `=segment`；被 4xx **且錯誤訊息提到 granularit／word** 就**只重試一次**不帶欄位，並記住這個引擎不支援（`wordTimestampsUnsupported`），之後直接不送；錯誤訊息講的是 response_format 本身（「unsupported response_format」）就照舊走 srt 退回，不多打一次（既有 `RejectedVerboseFallsBackToSRTAndLatches`／`MetersAudioExactlyOnce` 的請求序列不變）。
- 解析：`verboseTranscription.Words`。
- `tightenSegmentsWithWords`：每段的 start／end **只往內縮**到第一個／最後一個落在段內的字；沒有字落在段內、或縮完短於 0.6 秒就保留原時間；句數與順序不變（1:1 仍成立）。在幻覺過濾之前、渲染之前做；log Debug 記縮了幾段。
- 不動 `segmentsToSRT`、不動過濾規則、不動 srt 退路。

## Acceptance Criteria

1. 解析帶 `words` 的 verbose_json；沒有 `words` 時一切照舊。
2. `tightenSegmentsWithWords`：往內縮、不往外推、沒字不動、太短不動、句數順序不變、相鄰句不重疊。
3. 請求帶 `timestamp_granularities[]` 兩個值；引擎 4xx 時重試一次不帶欄位且 verbose_json 仍成功、之後的檔案不再送欄位。
4. 端到端：有 `words` 的回應產出的 SRT 用字的時間。
5. `go test ./...`、vet、staticcheck、lint:all 全綠。
6. **實測（隨「修一半」三張用 10 分鐘片段驗證，約 $0.3）：** 「出現在沒人說話時」的句數 vs 官方英文軌，從 10 分鐘內的基準明顯下降（10/5 整集 138／637）。

## Tasks / Subtasks

- [x] T1 `Words` 解析＋`tightenSegmentsWithWords`＋測試（AC #1、#2）
- [x] T2 請求欄位＋一次退回＋latch＋測試（AC #3、#4）
- [x] T3 檢查（AC #5）

## Dev Notes

- 只往內縮：往外推會蓋到下一句，也違反「不能比引擎給的範圍更寬」。
- 0.6 秒下限：一個字的「No!」仍要停留得夠久能讀。
- founder-pm：這是補模型弱點，做到可用就好。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **AC #6 實測（2026-10-06／07，10 分鐘片段 A/B，eval 第三節）：** 正常時時間最準（沒人說話處 7%、p90 0.59 秒、8:17 對到 0.5 秒內），但 4 次崩 3 次（整段只剩「♪♪」）。Alexyu 裁定選 A：預設關、留開關（`disc-2026-10-asr-word-timestamps-default-off`），切短音檔後再重測。
- **Adversarial CR（2026-10-06，fresh agent）1M/2L＋nits，全部修掉：**
  - M1 上一句最後一個字常常拖過整秒邊界（「name.」10.4–15.9 對上 15 秒結尾），它會被當成下一句的第一個字，下一句的起點就被釘在粗略的整秒——正是本單要修的症狀。改成起點取「第一個在這段內開口的字」，沒有才退回用拖尾字；加兩個 fixture。
  - L1 ASR 分段快取 key 沒有版本：10/5 跑過的 See 每段都在快取裡，重跑會 $0 拿回**粗略時間**的舊字幕，看起來像修沒效。快取前綴 `asrchunk:v1:` → `v2:`（含 manifest），舊的 30 天後自然過期。
  - L2 判斷「引擎不認得字級時間」原本比對裸字串 `word`，`invalid password` 也會中 → 改比對 granularit／word_timestamp／word timestamp／word-level 等片語；加表測。
  - nits：刪掉死寫入 `wordTimestamps = false`；加「5xx 不設 latch」測試、空白字跳過測試。
- 🔗 AC Drift: FOUND — 9R-5 AC「一段＝一句、時間不重切」：句數與順序仍 1:1，只有 start／end 往內縮；9R-5 的 verbose→srt 退路不變，多一層「先退掉字時間再退掉 verbose」。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/ai/whisper_segments.go`（改）、`apps/api/internal/ai/whisper.go`（改）、`apps/api/internal/ai/whisper_words_test.go`（新）、`apps/api/internal/services/asr_chunk_store.go`（改：快取前綴 v2）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-coarse-timestamps.md`（新）、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 實測：準但會崩（4 次崩 3 次）→ 預設關、留開關；後續 `disc-2026-10-asr-chunk-at-silence`／`asr-vad-pretrim`。 |
| 2026-10-06 | CR 1M/2L 修掉：起點取段內第一個字、快取前綴 v2、片語比對；全綠。 |
| 2026-10-06 | create-story＋dev 同日：word timestamps 請求＋退回、`tightenSegmentsWithWords`；全綠，狀態 review。 |
