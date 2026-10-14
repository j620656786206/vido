# Story asr-scribe-a-provider: 新寫一個 ElevenLabs Scribe v2 的聽寫接頭（後端，先不接上開關）

Status: done

<!-- SM Bob create-story 2026-10-14。來源：spike-asr-exam-2026-10（Scribe v2 考卷 3/3 過，whisper-1 0/3）；
     Alexyu 2026-10-14 裁定 A：預設換成 Scribe，Whisper 保留可選。拆兩張：-a 接頭（本單）、-b 金鑰／預設切換／估價／文案／文件。
     行號為 main 591b7403。 -->

## Story

身為片子沒有字幕軌、只能靠聽寫的 Vido 使用者，
我要 Vido 能用 ElevenLabs Scribe v2 把對白聽成文字（打熊那段不會再變成「喬！你還好嗎？」），
而且聽出來的字幕時間要準、句子要切得能讀，花多少錢要照 ElevenLabs 的價錢記。

這張只做「接頭」本身和它的測試；誰在什麼時候用它（金鑰、預設）是 `asr-scribe-b`。

## 查到的事（2026-10-14）

- 現有的聽寫介面：`apps/api/internal/ai/asr.go:18-24` `ASRProvider`（`Transcribe`／`TranscribeWithLanguage`），加上可選的 `DetailedTranscriber.TranscribeDetailed`（`whisper_segments.go:621-626`，回 `TranscriptionDetail{SRT, Unfiltered, SegmentsIn/Kept, Filtered}` `:597-612`）。`transcription_service.go:1756` 用型別斷言找 detailed 版。
- 幻覺過濾 `whisper_segments.go:423-515`：純文字規則（prompt_echo、music_only、symbol_only、credit_line、repeat_run）Scribe 可以直接用；靠 `NoSpeechProb`／`AvgLogprob`／`CompressionRatio` 的三條（`:442`、`:444`、`:493`）Scribe **沒有這些欄位**——給 0 就不會觸發，安全但沒保護。Scribe 自己會把非說話段標成 `audio_event`（`[bear roaring]`、`[dramatic music]`），這是它的替代品。
- 切段：`transcription_service.go:1521-1599` 的 chunk 迴圈是 provider 無關的（120 秒、在靜音處切，`whisper.go:785`），resume 扣錢和預算暫停都靠它（`asr_chunk_store.go`）。**本單沿用**：Scribe 雖然單檔可到 10 小時，但沿用切段才有「跑到一半停、下次接著」。chunk 快取 key 含 `EndpointFingerprint`（`asr_provider_holder.go:60-66`，Whisper 是 `"|whisper-1"`），Scribe 要給自己的（`"elevenlabs|scribe_v2"`），兩家的快取不會混。
- 計費：`budget.go:97` `whisperPerMinuteUSD = 0.006`；`RecordASRWithRate`（`:248`）可以帶自己的費率，Whisper 在 `whisper.go:273-277` 呼叫。Scribe $0.22／小時 → **$0.0036667／分**（2026-10-14 官方 API 價目）。
- ElevenLabs API（2026-10-14 實測 ＋ 官方文件）：
  - `POST https://api.elevenlabs.io/v1/speech-to-text`，header `xi-api-key`，multipart：`model_id=scribe_v2`、`file`、`language_code`（ISO-639-1 可）、`timestamps_granularity=word`、`tag_audio_events=true`。
  - 回 `{language_code, language_probability, text, words[], transcription_id, audio_duration_secs}`；`words[]` 每個 `{text, type: word|spacing|audio_event, start, end, logprob, speaker_id}`。**沒有 segments**。
  - `additional_formats=[{format:"srt",…}]` 要同時開 `diarize=true`；實測它切出來的 SRT 很差（音效標記撐 20 秒、「Hey, face / me!」斷在句中），**不用它，自己切**。
  - 422 是 `{"detail":[{loc,msg,type}]}`；400 是 `{"detail":{"status":"invalid_parameters",…}}`。額度用完的回應沒實測到（免費額度還夠），文件沒寫；先依 `status` 含 `quota_exceeded` 或 HTTP 402 判定，並在 Dev Agent Record 標「未經真回應驗證」。
  - 錄下的真回應：`eval/asr-exam-2026-10/runs/scribe-v2-1.json`（10 分鐘、839 個 word）→ 當 Rule 28 夾具。
- 逐字切句的參考做法：`eval/asr-exam-2026-10/score.py` 的 `scribe_cues`（句尾標點或 ≥0.8 秒空隙就斷），考卷上跟官方切法對得很好（時間中位 0.11 秒）。

## Acceptance Criteria

1. **新接頭 `ai.ScribeClient`**（`apps/api/internal/ai/scribe.go`）實作 `ASRProvider` 與 `DetailedTranscriber`；建構 `NewScribeClient(apiKey, opts...)`，選項：base URL（預設 `https://api.elevenlabs.io`）、model（預設 `scribe_v2`）、HTTP client、逾時。`EndpointFingerprint()` 回 `"elevenlabs|<model>"`。
2. **請求**：`model_id`、`file`、`timestamps_granularity=word`、`tag_audio_events=true`、有語言就 `language_code`。不送 `diarize`、`additional_formats`、`keyterms`（名字提示這張先不送：有 20% 加價，另立案）。
3. **逐字 → 句子**：只收 `type=word`；`spacing` 用來還原空格；`audio_event` 丟掉但**當作斷句點**。句尾 `.?!` 或跟上一個字空 ≥ 0.8 秒就斷；一句最長 7 秒、最多 84 字元（超過就在最近的空隙斷）。每句的 start／end 取字的時間；一句最短顯示 0.6 秒（沿用 `minTightenedCueSeconds`），延長時不蓋到下一句。
4. **過濾**：切出來的句子走既有的 `filterHallucinationsWith`（純文字規則生效；三個 Whisper 欄位給 0）。`TranscriptionDetail.SegmentsIn/Kept/Filtered` 照實填。
5. **計費**：每次呼叫用 `RecordASRWithRate(ctx, seconds, scribePerMinuteUSD)`，`scribePerMinuteUSD = 0.22/60`，放 `budget.go` 並有測試；預算不夠時跟 Whisper 一樣在呼叫前被擋（`ErrBudgetExceeded`）。
6. **錯誤**：401／403／422 → `ErrScribeAPIError`（跟 `ErrWhisperAPIError` 同一慣例）帶伺服器訊息、不重試；429／5xx／逾時 → `retryTransient`；額度用完（402 或 `status` 含 `quota_exceeded`）→ `CreditExhaustedError{Provider: "elevenlabs"}`（`credit_exhausted.go:26-64` 加這個 provider）。ctx 取消回 `ctx.Err()`。
7. **Rule 28 真形狀測試**：用 `eval/asr-exam-2026-10/runs/scribe-v2-1.json` 當 `testdata/scribe_v2_ok.json`，斷言：打熊段切出 ≥ 8 句且沒有任何句子只含音效；「Who's there? Answer me. Who's there?」附近切成 3 句；全部句子 ≤ 7 秒；第一句在 228.8 秒附近（前 4 分鐘純配樂一句都沒有）。另錄一個 422 當錯誤夾具。
8. **不做**：不接 holder、不加金鑰、不改估價、不改文案（全在 -b）。既有 Whisper 測試一字不改且全綠。

## Tasks / Subtasks

- [x] T1 `budget.go`：`scribePerMinuteUSD` ＋ 測試
- [x] T2 `credit_exhausted.go`：provider `elevenlabs`（含 `creditStop.ts:13` 的對照表要在 -b 加，這裡先後端）
- [x] T3 `scribe.go`：client、請求、回應解析、逐字切句、過濾、計費、錯誤
- [x] T4 `scribe_test.go`：httptest 假伺服器（請求欄位、auth、重試、額度、取消）＋ Rule 28 夾具測試
- [x] T5 `go vet`／staticcheck／既有測試全綠

### Time-dependent visual coverage

N/A（後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- `scribe.go`：`ScribeClient`（`ASRProvider`＋`DetailedTranscriber`），`xi-api-key` 認證，送 `model_id`／`timestamps_granularity=word`／`tag_audio_events=true`／`language_code`；不送 diarize／additional_formats／keyterms。`EndpointFingerprint()`＝`elevenlabs|scribe_v2`。
- 逐字切句 `scribeCues`：句尾標點、≥0.8 秒空隙、音效標記、7 秒、84 字寬（漢字算 2）就斷；空格只在 Scribe 給 `spacing` 的地方補（中文不會變成「我 們 走 吧」）；單一 token 拖超過 7 秒切到 7 秒；短句最少 0.6 秒但不蓋到下一句；`Mr.`／`Dr.`／`U.S.`／單字母縮寫不當句尾。
- 計費：`scribePerMinuteUSD = 0.22/60`（`budget.go`），用 Scribe 回的 `audio_duration_secs`，沒有才讀 WAV 表頭；兩者都沒有就不記（跟 Whisper 一樣）。
- 錯誤：4xx 不重試、5xx／429／逾時重試；額度用完只在 4xx 判（`quota_exceeded`／`insufficient_credits`／`payment_required`／402）→ `CreditExhaustedError{Provider:"elevenlabs"}`，**未經真回應驗證**。
- Rule 28 夾具 `testdata/scribe_v2_ok.json`（2026-10-14 真回應，10 分鐘 839 字）：首句 228.8 秒、232–260 秒沒有句子、打熊段 10 句（斷言 ≥8）、「Who's there? / Answer me. / Who's there?」三句、全部 ≤7 秒、短句只有在下一句緊接時才短。
- 檢查：`go test ./...`（api 全部）、`go vet`、staticcheck、prettier 全綠；既有 Whisper 測試未改。

### Senior Developer Review (AI) — 2026-10-14（Fable 5.1 跨模型）

結論：1 High／3 Medium／5 Low，全部處理。

- **H1（已修）**：原本不看 `spacing` token、一律用空格接字，中文會變成「我 們 走 吧 。」，84 字上限也沒算漢字寬。改成只在有 spacing 的地方補空格、漢字算 2 格；加兩條中文測試。
- **M1（已修）**：單一 token 拖超過 7 秒不會被切。改成切到 7 秒；加測試。
- **M2（已修）**：額度用完的字串比對連 5xx 也做，proxy 的 502 頁面寫到 payment 就會停整批。改成只在 4xx 判；加測試。
- **M3（已修）**：夾具測試斷言打熊 ≥6 句，AC 寫 ≥8。改成 ≥8（實際 10 句）。
- **L1（已修）**：沒有長度資料時不再記一筆 0 元的呼叫。**L2（已修）**：`{}` 空回應當無效，不當靜音快取。**L3（已修）**：`Mr.`／`Dr.`／`U.S.`／單字母縮寫不斷句。**L4（接受）**：錯誤 body 整個印進 log，跟 Whisper 一樣。**L5（已處理）**：AC6 的 sentinel 名稱改成實作用的 `ErrScribeAPIError`。

### File List

- `apps/api/internal/ai/scribe.go`、`scribe_test.go`、`testdata/scribe_v2_ok.json`（新）
- `apps/api/internal/ai/budget.go`、`credit_exhausted.go`
- `_bmad-output/implementation-artifacts/asr-scribe-a-provider.md`、`asr-scribe-b-default-and-keys.md`（新）、`sprint-status.yaml`
