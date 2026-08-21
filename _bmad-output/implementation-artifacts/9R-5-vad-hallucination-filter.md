# Story 9R-5: Whisper 幻覺後置過濾 —— 靜音／重複／尾段的信心值過濾器

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P1 M · **PROVEN**（POC 實證，非假設）
**Depends:** 9R-3 ✅（分塊）、9R-4 ✅（retry）、9R-9 ✅（`ASRProvider` 介面）、9R-11 ✅（budget 計量）
**Source:** sprint-status `9R-5-vad-hallucination-filter`；ADR `adr-subtitle-route-c-generation` **Decision 5 第 5 項**（「Whisper hallucination on silence/credits (POC produced a fake "like & subscribe" outro) → VAD / tail-detection post-filter」）；策略重審 `strategy-review-whatsub-2026-08-19.md:56` 列為 O 情境的品質欠帳。

> **⚠️ 2026-08-21 RE-AUTHORED（第二版）。** 初稿（2026-08-19）**在 Task 清單處截斷** —— 沒有 Dev Notes、References、Dev Agent Record、Discovery Triage。
> 本版逐行 grep 實查 `main` HEAD `cabc4f99`，補齊全部 template 區段，並**新增四項初稿未觸及的設計裁定**（見下方「初稿缺口」），其中兩項會直接決定實作對錯。

> **📛 命名誠實聲明。** ADR 的用語是「VAD / tail-detection post-filter」，story key 也叫 `vad-hallucination-filter`，
> 但**本案不做真正的 VAD**（不引入 silero 等語音活動偵測模型）—— 實作的是**基於 ASR 自身信心值的後置過濾**。
> 保留 key 是為了與 sprint-status／ADR 對得上，但 AC、程式碼與 log 一律用「hallucination filter」而非「VAD」，避免下一個人誤以為有 VAD 模型。

---

## Story

As a NAS owner whose credits roll has no dialogue,
I want Whisper's hallucinated cues filtered out of the ASR output **before** translation,
so that my subtitles don't contain fabricated lines — and I don't pay Claude to translate them.

**POC 硬證據**（`subtitle-v4-replan-and-feasibility-audit-2026-06.md:185-192`）：同一集，官方字幕 **1029 句**，Whisper 產出 **1082 句**（「含幻覺尾＋較細切」），片尾靜音段被**憑空掰出「點讚訂閱」**。差額約 5%，且幻覺內容會**原封不動進入翻譯批次並計費**。

---

## Context —— 現況逐行查證 @ `cabc4f99`

| 事實 | 位置 | 對本案的意義 |
|---|---|---|
| 目前請求 `response_format=srt` | `internal/ai/whisper.go:217` | 回來是**純 SRT 字串**，**拿不到** `no_speech_prob` / `avg_logprob` / `compression_ratio` ⇒ 沒有任何過濾依據 |
| `ASRProvider` 介面回傳 `(string, error)` | `internal/ai/asr.go:18-24` | 9R-9 的可抽換承諾；**本案不得改動此介面**，否則自架引擎的 swap-in 契約破裂 |
| `ai` 是 **leaf package**（零內部相依） | `project-context.md` Rule 19 + `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` | 過濾器必須**完全住在 `ai` 內**，不得 import `subtitle`／`services`；`subtitle.ParseSRT` / `SerializeSRT` **不可重用** |
| SRT 時間戳工具已在 `ai` 內 | `whisper.go:599 parseSRTTimestamp` / `:617 formatSRTTimestamp`（毫秒 ↔ `HH:MM:SS,mmm`） | 從 segments 組 SRT **有現成 formatter**，不要重寫 |
| 分塊路徑 | `NeedsChunking:303` → `SplitAudioChunks:322` → 逐塊 `TranscribeWithLanguage` → `MergeSRTChunks:414` | **每個 chunk 是一次獨立 API 呼叫** ⇒ 過濾天然就是 per-chunk、天然在 merge 前 |
| merge 只重編號＋位移 | `MergeSRTChunks:414-432` → `adjustSRTTimestamps:507` | 只要各 chunk 的 SRT **格式合法**（`isSequenceNumber` + `isTimestampLine` 認得），merge 完全不用改 |
| 回應上限 10MB | `WhisperMaxResponseSize`（`whisper.go:32`） | verbose_json 比 srt 大約 3-5×，但 10 分鐘 chunk 約 150-300 segments ≈ 100KB ⇒ **遠低於上限，無需調整** |
| budget 計量在 governed 呼叫**之後**一次 | `whisper.go`（`b.RecordASRWithRate(dur, ...)`） | ⚠️ 若加入「verbose 失敗→改用 srt 重送」的回退，**必須確保每個音檔只計量一次**（見 AC #2.3） |
| 自架判定單一來源 | `IsSelfHostedASRBaseURL:117` / `isSelfHosted:126` | 回退路徑的 log 應帶 `self_hosted`，方便分辨「OpenAI 異常」vs「自架引擎不支援」 |
| 空 SRT 會被無條件寫檔 | `transcription_service.go:518-524`（`os.WriteFile(srtPath, []byte(srtContent), 0644)`） | ⚠️ **沒有空內容守衛** ⇒ 過濾若清空全檔，會落一個 0 byte 的 `.en.srt`，接著 `translateSRT` 以「no subtitle blocks」失敗、被 `translateAndPersist` 吞掉 ⇒ 磁碟上留空檔、verdict 記 `untranslated`。這正是 bugfix-j 剛修掉的「狀態說謊」類。**本案必須自己擋住**（AC #5） |

### 初稿（2026-08-19）缺口 —— dev 必讀

初稿的四條 AC 方向正確，但**漏掉四個會決定實作對錯的裁定**，本版補齊：

| # | 初稿沒說的 | 本版裁定 |
|---|---|---|
| 1 | **verbose_json 失敗時「回退現行 SRT 路徑」具體怎麼回退？** verbose_json 的 `text` 欄位沒有時間戳，拿不到就等於**沒有字幕**，不是「少了過濾」而已。 | **重送一次 `response_format=srt`**，並用 **client 層 sticky 旗標**記住「這個 endpoint 不支援 verbose」，避免對 10 個 chunk 探測 10 次（AC #2） |
| 2 | **回退會不會重複計費／重複計量？** | 明確紅線：**每個音檔只 `RecordASRWithRate` 一次**（AC #2.3），並附回歸測試 |
| 3 | **過濾把整檔清空怎麼辦？** 而「整個 chunk 被清空」恰恰是本案的招牌情境（片尾 chunk 全是幻覺）。 | 守衛放在**檔案層、不是 chunk 層**：per-chunk 全清空**合法**；合併後全檔零 cue 但未過濾版有 cue ⇒ **改用未過濾版**＋`Warn`（AC #5） |
| 4 | **改用 verbose_json 會不會改變 cue 切法？** 這會影響**每一份** ASR 字幕，不只幻覺的那幾句。 | 組裝規則**逐字對齊 whisper 參考實作的 `write_srt`**（直接用 segment 的 start/end/text，**不重新斷行、不重新合併**）＋結構等價回歸測試（AC #4、AC #7） |

---

## Acceptance Criteria

### AC #1 — 取得判斷依據：`verbose_json` 解析（`ai` 套件內）

1.1 新檔 `apps/api/internal/ai/whisper_segments.go`，定義**未匯出**的回應型別：
```go
type whisperSegment struct {
    ID               int     `json:"id"`
    Start            float64 `json:"start"`             // 秒
    End              float64 `json:"end"`               // 秒
    Text             string  `json:"text"`
    NoSpeechProb     float64 `json:"no_speech_prob"`
    AvgLogprob       float64 `json:"avg_logprob"`
    CompressionRatio float64 `json:"compression_ratio"`
}
type verboseTranscription struct {
    Language string           `json:"language"`
    Duration float64          `json:"duration"`
    Text     string           `json:"text"`
    Segments []whisperSegment `json:"segments"`
}
```
1.2 `segmentsToSRT(segs []whisperSegment) string` —— **重用既有 `formatSRTTimestamp`**（`whisper.go:617`）。
- 序號自 1 起連號；時間戳 `HH:MM:SS,mmm --> HH:MM:SS,mmm`；`strings.TrimSpace(seg.Text)`；每 cue 後空行。
- 🔴 **不重新斷行、不合併相鄰 segment、不做任何文字改寫** —— 逐字對齊 whisper 參考實作 `write_srt` 的語義（AC #7 的等價性靠這條）。
- 產物必須被 `adjustSRTTimestamps` 的 `isSequenceNumber` / `isTimestampLine`（`whisper.go:566/578`）認得 —— 這是 merge 路徑不用改的前提。

### AC #2 — 請求切換 + sticky 回退（fail-soft，Rule 13）

2.1 `TranscribeWithLanguage`（`whisper.go:176`）的 `response_format` 由 `"srt"` 改為 `"verbose_json"`。
2.2 **Sticky 回退**：`WhisperClient` 新增未匯出欄位 `verboseUnsupported atomic.Bool`。
- 當 verbose 請求得到 **4xx**（引擎不認這個 format）**或** 200 但 JSON 無法解析／`len(Segments) == 0` ⇒ 設旗標、以 `response_format=srt` **重送一次**、回傳未過濾的 SRT、記一行 `slog.Warn`（帶 `self_hosted`、`reason`、`base_url`）。
- 旗標為 true 時**直接**送 `srt`，不再探測 ⇒ 10 個 chunk 只浪費 1 次探測。
- 🔴 **5xx／逾時／網路錯誤不得設旗標**：那是暫時性故障，已由 9R-4 的 `retryTransient` 處理；設旗標會讓一次抖動永久關閉過濾器。
2.3 🔴 **每個音檔只計量一次。** `b.RecordASRWithRate(...)` 必須在「最終成功取得 SRT」之後執行**恰好一次**，不論中途是否走過回退。附回歸測試（AC #6.5）。
2.4 `Transcribe`（`whisper.go:168`）委派不變；**`ai.ASRProvider` 介面一個字不動**（9R-9 的抽換契約）。

### AC #3 — 三條過濾規則（純函式、常數化、附出處）

3.1 `filterHallucinations(segs []whisperSegment) (kept []whisperSegment, dropped []droppedSegment)` —— **純函式，零 I/O**，可獨立單元測試。`droppedSegment` 攜帶 `whisperSegment` + `reason string`。

3.2 閾值一律具名常數，**採用 OpenAI whisper 參考實作自己的預設值**（不要自行發明數字），並在註解引用出處：

| 常數 | 值 | 出處 |
|---|---|---|
| `hallucinationNoSpeechThreshold` | `0.6` | whisper `--no_speech_threshold` 預設 |
| `hallucinationLogprobThreshold` | `-1.0` | whisper `--logprob_threshold` 預設 |
| `hallucinationCompressionThreshold` | `2.4` | whisper `--compression_ratio_threshold` 預設 |
| `hallucinationTailNoSpeechThreshold` | `0.4` | 尾段專用的**較寬鬆**門檻（見 3.5 的理由） |
| `hallucinationTailMinRun` | `3` | 尾段規則最少要連續幾段才啟動 |
| `hallucinationRepeatRun` | `3` | 連續同文字幾段才算迴圈幻覺 |

3.3 **R1 靜音幻覺**：`NoSpeechProb > 0.6 && AvgLogprob < -1.0` → 剔除，`reason = "silence"`。
（此 AND 條件即 whisper 自己判定「這段其實是靜音」的條件，誤殺風險最低。）

3.4 **R2 重複幻覺**：兩個子規則，命中任一即剔除。
- `CompressionRatio > 2.4` → `reason = "repetition"`（單段內部自我重複）。
- 連續 **≥3** 段 `strings.TrimSpace` 後文字相同 → **保留第一段**，其餘剔除，`reason = "repeat_run"`（Whisper 迴圈幻覺的典型型態）。

3.5 **R3 尾段偵測**：從**尾端**往前掃，剔除滿足全部條件的結尾連續段：
- 該連續段**每一段** `NoSpeechProb > 0.4`（比 R1 寬鬆），且
- 連續段長度 **≥ 3**，且
- 該連續段**一路延伸到最後一段**（中間不得有高信心段）。
- `reason = "tail"`。
- 🔴 **較寬鬆的門檻只准用在尾端**：POC 的失敗就發生在片尾靜音區，且片尾是最不可能有對白的位置。同樣的門檻用在檔案中段會誤殺真對白 —— 註解必須寫下這個理由。

3.6 🔴 **禁止關鍵字黑名單。** 不得比對「點讚訂閱」/「like and subscribe」之類的字串 —— 語言相關、脆弱、且會誤殺真的在講這句話的影片。只用信心值與結構訊號。

3.7 **可觀測性**：每個被剔除的段記 `slog.Debug`（`start` / `end` / `text` / `reason` / 三個信心值）；每次呼叫記一行 `slog.Info`（`segments_in` / `segments_kept` / `dropped_by_reason` map / `drop_ratio`）。
- `drop_ratio > 0.2` 時額外 `slog.Warn` —— POC 的預期差額約 5%（1082→1029），掉兩成代表門檻有問題，操作者需要看得見。

### AC #4 — 分塊路徑：零改動即相容（明文驗證，不是假設）

4.1 過濾發生在 `TranscribeWithLanguage` **內部**，而每個 chunk 是一次獨立呼叫 ⇒ **per-chunk 過濾、merge 前過濾**兩件事**自動成立**，`SplitAudioChunks` / `MergeSRTChunks` / `adjustSRTTimestamps` **一行都不用改**。
4.2 但必須以測試證明，不得只在文件裡宣稱：
- 多 chunk 情境（其中一個 chunk 全數被剔除、變成空字串）→ `MergeSRTChunks` 仍產出**連號正確、時間戳位移正確**的合法 SRT。
- 單檔（不分塊）路徑與分塊路徑的過濾行為等價。

### AC #5 — 🔴 檔案層空結果守衛（過濾器不得讓整部片消失）

5.1 **per-chunk 全數剔除是合法的** —— 片尾 credits 自成一個 chunk 且整段是幻覺，正是本案要處理的招牌情境。
5.2 但在**呼叫端**（`services.TranscriptionService.transcribeAudio`，`transcription_service.go`）必須守住檔案層：若**過濾後合併結果沒有任何 cue**、而**未過濾版本有 cue** ⇒ **改用未過濾版本**，並記 `slog.Warn("hallucination filter would have emptied the whole transcript — using unfiltered output")`。
5.3 理由：目前 `runPipeline`（`transcription_service.go:518-524`）會把 `srtContent` **無條件寫檔**，沒有空內容守衛。過濾器清空全檔的話，磁碟上會留一個 0 byte 的 `.en.srt`，接著 `translateSRT` 以「no subtitle blocks to translate」失敗並被 `translateAndPersist` 吞掉 —— 使用者得到一個空檔加一個 `untranslated` 徽章，正是 bugfix-j 剛修掉的「狀態說謊」類。
5.4 實作備註：`ai` 是 leaf package 不能回頭呼叫 services，所以「未過濾版本」必須由 `ai` 側提供。最小做法是讓過濾在 `ai` 內完成後，`TranscribeWithLanguage` **在自己這一層**就保證「有輸入 segments 就一定有輸出 cue」——
即：**若某次呼叫的 `kept` 為空但 `segs` 非空，仍回傳過濾後的空 SRT（chunk 層允許），但額外提供一個匯出函式或回傳旗標讓呼叫端能判斷全檔情況**。
⚠️ **此處的具體 seam 由 dev 決定**，但必須滿足：(a) 不改 `ASRProvider` 介面；(b) 不讓 `ai` import `services`；(c) 全檔空結果**不得**落到磁碟。
建議做法：`transcribeAudio` 自己保留每個 chunk 的**未過濾** SRT（由 `ai` 以第二個匯出入口提供，或由 client 在回退旗標外再帶一個 `Unfiltered` 取得器），合併後比對；若過濾版空、未過濾版非空 → 用未過濾版。

### AC #6 — 測試

6.1 **過濾純函式表格測試**（`whisper_segments_test.go`）：R1 / R2（兩個子規則）/ R3 各自的命中與**不命中**案例；正常對白全數保留。
6.2 **POC 型 fixture**：一段正常對白 + 片尾 3 段高 `no_speech_prob` 的假 outro → 斷言尾段被剔、**前面每一句 text 與時間戳 byte-preserved**。
6.3 **零剔除等價**：全高信心輸入 → `dropped` 為空，且 `segmentsToSRT` 產出的 cue 數／文字／毫秒時間戳與同一份 segments 逐一對應（AC #7 的結構等價）。
6.4 **回退路徑**（httptest，鏡射 `whisper_test.go:22` 既有樣式）：
- 伺服器對 `verbose_json` 回 400、對 `srt` 回正常 SRT → 拿到未過濾 SRT、旗標已設；**第二次呼叫直接送 `srt`**（斷言伺服器只收到一次 `verbose_json`）。
- 伺服器 200 但回傳非 JSON → 同樣回退。
- 伺服器 500 → **旗標不得被設**（斷言後續呼叫仍送 `verbose_json`）。
6.5 🔴 **計量一次**：走過回退的呼叫，`ai.Budget` 的 ASR 分鐘數**只被記錄一次**。
6.6 **分塊**：AC #4.2 的兩個情境。
6.7 **檔案層守衛**（services 側）：全部 chunk 都被清空 → 落到磁碟的是未過濾內容、且有 Warn。
6.8 既有 `internal/ai` 38 個測試 + `internal/services` transcription 系列**全綠**。
⚠️ 既有 `TestWhisperClient_Transcribe_Success`（`whisper_test.go:30`）斷言 `assert.Equal(t, "srt", r.FormValue("response_format"))` —— **這條會紅，屬預期**；必須改為 `verbose_json` 並在 story 的 Completion Notes 記錄「刻意變更的既有斷言」。

### AC #7 — 🔴 cue 切法不得漂移（每一份字幕都受影響，不只幻覺那幾句）

7.1 改用 verbose_json 後，SRT 由**我方**組裝。組裝規則必須是「一個 segment ＝ 一個 cue，start/end/text 原封不動」，這與 whisper 參考實作 `write_srt` 的語義一致 —— 因為 OpenAI 的 `response_format=srt` 本來就是從**同一組 segments** 產生的。
7.2 以 fixture 對照證明結構等價（cue 數、每句文字、每個毫秒時間戳），寫成回歸測試。
7.3 **選作、但強烈建議**：dev 若手上有真 API key 與一段音訊，對同一檔各取一次 `srt` 與 `verbose_json`，diff 兩者的 cue 切法並把結果記進 Completion Notes。這是唯一能證明「線上行為沒漂移」的證據；做不到就明說做不到，不要含糊帶過。

### AC #8 — 範圍紅線

- ❌ **不引入本地 VAD 模型／依賴**（silero、webrtcvad 等）—— 屬 Tier-2，且會把 M 級 story 變成引擎選型案（與 `9R-S2-nas-whisper-benchmark-spike` 耦合）。
- ❌ **不改 `ai.ASRProvider` 介面**（9R-9 抽換契約）。
- ❌ **不動 `MergeSRTChunks` / `adjustSRTTimestamps` / `SplitAudioChunks`**（AC #4.1 已證明不需要）。
- ❌ **不碰翻譯側**（`TranslationOutcome`、verdict 三態、glossary、9R-8 的 metadata）。
- ❌ **不做 `backlog-asr-leg-unify-gated-pipeline`**（ASR 腿併入閘門化 `TranslateTrack`）—— 本案是該重構之前的獨立品質修補。
- ❌ **不加關鍵字黑名單**（AC #3.6）。

---

## Tasks / Subtasks

- [x] **Task 1 —— verbose_json 型別與 SRT 組裝（AC #1 / #7）** `internal/ai/whisper_segments.go`（新檔）
  - [x] 1.1 `whisperSegment` / `verboseTranscription` 型別
  - [x] 1.2 `segmentsToSRT` —— 重用 `formatSRTTimestamp`，一 segment 一 cue，不重新斷行
  - [x] 1.3 結構等價測試（AC #6.3 / #7.2）
  - [x] 1.4 驗證產物能被 `isSequenceNumber` / `isTimestampLine` 認得（merge 相容前提）

- [x] **Task 2 —— 三條過濾規則（AC #3）** 同檔
  - [x] 2.1 六個具名常數 + 出處註解（whisper 參考實作預設值）
  - [x] 2.2 `filterHallucinations` 純函式：R1 silence / R2 repetition + repeat_run / R3 tail
  - [x] 2.3 R3 的「寬鬆門檻只准用在尾端」理由寫進註解
  - [x] 2.4 Debug／Info／>20% Warn 三層 log
  - [x] 2.5 表格測試（AC #6.1 / #6.2）

- [x] **Task 3 —— 請求切換與 sticky 回退（AC #2）** `internal/ai/whisper.go`
  - [x] 3.1 `response_format` → `verbose_json`；`verboseUnsupported atomic.Bool` 欄位
  - [x] 3.2 4xx／JSON 解析失敗／零 segments → 設旗標 + 以 `srt` 重送一次 + Warn（帶 `self_hosted`）
  - [x] 3.3 🔴 5xx／逾時**不得**設旗標（交給 9R-4 的 `retryTransient`）
  - [x] 3.4 🔴 `RecordASRWithRate` 每個音檔恰好一次
  - [x] 3.5 httptest 回退測試四案（AC #6.4 / #6.5）
  - [x] 3.6 更新既有 `TestWhisperClient_Transcribe_Success` 的 `response_format` 斷言（AC #6.8 的刻意變更）

- [x] **Task 4 —— 分塊相容驗證（AC #4）**
  - [x] 4.1 多 chunk 且其中一 chunk 全清空 → merge 產出合法連號 SRT
  - [x] 4.2 單檔 vs 分塊行為等價

- [x] **Task 5 —— 檔案層空結果守衛（AC #5）** `internal/services/transcription_service.go`
  - [x] 5.1 決定並實作 seam（不改 `ASRProvider`、不讓 `ai` import `services`）
  - [x] 5.2 全檔空結果 → 改用未過濾版 + `slog.Warn`
  - [x] 5.3 services 側測試（AC #6.7）

- [x] **Task 6 —— 閘門（Rule 12 / Rule 15）**
  - [x] 6.1 `pnpm nx test api`
  - [x] 6.2 `pnpm nx lint api`（釘版 staticcheck-2026.1）
  - [x] 6.3 `pnpm run lint:all`（含全 repo `format:check`）
  - [x] 6.4 `gofmt -l` 本案檔案為空

---

## Dev Notes

### 重用清單（❌ 禁止重新發明）

| 需求 | 已存在 | 位置 |
|---|---|---|
| 毫秒 → `HH:MM:SS,mmm` | `formatSRTTimestamp` | `ai/whisper.go:617` |
| `HH:MM:SS,mmm` → 毫秒 | `parseSRTTimestamp` | `ai/whisper.go:599` |
| SRT 行辨識 | `isSequenceNumber` / `isTimestampLine` | `ai/whisper.go:566` / `:578` |
| 分塊合併 | `MergeSRTChunks` / `adjustSRTTimestamps` | `ai/whisper.go:414` / `:507` |
| 暫時性錯誤重試 | `retryTransient` / `isTransientStatus` | `ai/retry.go`（9R-4） |
| 節流 + 預算 | `governed` / `BudgetFromContext` / `RecordASRWithRate` | `ai/governor.go` / `ai/budget.go`（9R-11） |
| 自架判定 | `IsSelfHostedASRBaseURL` / `isSelfHosted` | `ai/whisper.go:117` / `:126` |
| httptest 測試樣式 | `TestWhisperClient_Transcribe_Success` | `ai/whisper_test.go:19-56` |

### 架構護欄

- **Rule 19（package 邊界）—— 本案最硬的限制。** `ai` 是 **leaf package**，由 `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` 強制。
  過濾器與 SRT 組裝**必須完全住在 `ai` 內**，且**不得 import `internal/subtitle`**（會製造 cycle：`subtitle` 已 import `ai`）。
  ⇒ `subtitle.ParseSRT` / `subtitle.SerializeSRT` **不可重用**，這不是懶得重用，是編譯不過。
- **Rule 13（錯誤處理完整性）**：所有 fail-soft 吞掉的路徑都要有 `slog.Warn` 並帶足夠診斷欄位（`reason` / `self_hosted` / `base_url`）。
- **Rule 2（只用 slog）**：`ai` 套件用 `c.logger`（`WhisperClient.logger`）；純函式沒有 receiver，log 由呼叫端負責 —— **保持 `filterHallucinations` 純淨、零 log**，把彙總資訊透過回傳的 `dropped` 交給呼叫端記錄。
- **Rule 7（錯誤碼）**：本案**預期不新增錯誤碼**（全路徑 fail-soft）。若真的需要新 sentinel，`ai` 套件的前綴是 `AI_`，且必須同步 `project-context.md` Rule 7 清單。
- **Rule 20（AC 契約版本）**：`ASRProvider`（`ai/asr.go:18`）**未帶 `[@contract-vN]` 戳記** ⇒ 隱含 v0，且本案不改它 ⇒ **不欠 bump、不欠 ack**。dev 開工時仍需自行 grep 複查（`grep -rnE '\[@contract-v[0-9]+\]' internal/ai/`）。
- **Rule 16（測試斷言品質）**：AC #6.3 / #7.2 的等價性必須逐欄位斷言（cue 數、文字、毫秒），禁止只斷言「非空」。
- **Rule 14（資源生命週期）**：回退重送要複用**同一份 multipart body bytes**（現行程式已把 body 建好成 `bodyBytes` 再重複使用），不要重開檔案。

### 五個最容易踩的坑

1. **把「回退」理解成「跳過過濾」。** verbose_json 拿不到就是**完全沒有字幕**（`text` 欄位無時間戳）。回退＝**用 `srt` 重送一次**，不是「繼續用剛才那個回應」。
2. **5xx 設了 sticky 旗標。** 一次網路抖動就永久關掉整個 deployment 的過濾器，而且沒人會發現。只有 4xx／解析失敗才算「這個引擎不支援」。
3. **回退路徑重複計量 ASR 分鐘。** 預算會被虛耗一倍，9R-11 的天花板提早觸頂。
4. **在 chunk 層擋空結果。** 那會**直接廢掉本案的招牌情境**（片尾 chunk 整段是幻覺，本來就該全清）。守衛屬於檔案層。
5. **想用 `subtitle.ParseSRT`。** 編譯不會過（import cycle）。`ai` 是 leaf。

### Testing standards

- Go 測試與被測檔**同 package 同目錄**（Rule 9）。
- 純函式測試用**表格驅動**（`whisper_test.go` 與 `translation_service_test.go` 皆有既有樣式）。
- HTTP 行為測試用 `httptest.NewServer`，鏡射 `whisper_test.go:19-56`：在 handler 內 `r.ParseMultipartForm` 後直接斷言 `r.FormValue("response_format")`。
- 回退／sticky 測試需要**計數 handler 收到的 format**，用 closure 捕獲 slice 累積，最後斷言序列（例：`["verbose_json", "srt", "srt"]`）。

### Project Structure Notes

- 新檔 1 個：`apps/api/internal/ai/whisper_segments.go`（＋同名 `_test.go`）。落在既有 `ai` 套件內，無新目錄、無新 package、**零新第三方相依**（`encoding/json` + `sync/atomic` 皆為 stdlib）。
- 修改：`apps/api/internal/ai/whisper.go`、`apps/api/internal/ai/whisper_test.go`、`apps/api/internal/services/transcription_service.go`（＋其測試）。
- 與統一結構無衝突：`ai` 已是 Rule 19 的 leaf 套件，過濾器是它的內部關注點。

### Time-dependent visual coverage

- **N/A —— 本 story 不觸及任何 `apps/web/src/components/**` 檔案**（純 Go 後端）。無 wall-clock-reading 元件、無 fixture 基準需求。Rule 23 不適用。

### References

- [Source: `_bmad-output/planning-artifacts/architecture/adr-subtitle-route-c-generation.md#59-65`] — Decision 5 第 5 項：VAD / tail-detection post-filter 裁定在案
- [Source: `_bmad-output/planning-artifacts/subtitle-v4-replan-and-feasibility-audit-2026-06.md#185-192`] — POC 對照表：官方 1029 句 vs Whisper 1082 句、片尾「點讚訂閱」幻覺
- [Source: `_bmad-output/planning-artifacts/subtitle-v4-replan-and-feasibility-audit-2026-06.md#197,207`] — 幻覺尾巴為「Whisper 通病，與引擎無關」
- [Source: `_bmad-output/planning-artifacts/strategy-review-whatsub-2026-08-19.md#56`] — O 情境的品質欠帳定位
- [Source: `apps/api/internal/ai/whisper.go#176-300`] — `TranscribeWithLanguage` 現行 srt 請求、retry、budget 計量順序
- [Source: `apps/api/internal/ai/whisper.go#414-432,507-546,566-625`] — merge / 位移 / 行辨識 / 時間戳工具
- [Source: `apps/api/internal/ai/whisper.go#19-35`] — `WhisperMaxResponseSize` 等常數
- [Source: `apps/api/internal/ai/asr.go#18-27`] — `ASRProvider` 介面（9R-9 抽換契約，本案不動）
- [Source: `apps/api/internal/services/transcription_service.go#495-535`] — `runPipeline` 無條件寫 SRT（AC #5 的成因）
- [Source: `project-context.md#Rule 19`] — leaf package 清單與 `TestLeafPackagesHaveNoInternalDeps`
- [Source: `project-context.md#Rule 2 / 7 / 13 / 14 / 16 / 20`] — slog、錯誤碼、錯誤處理、資源、斷言品質、契約版本
- [Source: sprint-status `backlog-asr-leg-unify-gated-pipeline`] — 本案刻意不做的長線重構（同案曾列 9R-5）

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (BMAD `dev-story`, 2026-08-21)

### Debug Log References

- **真 RED（測試抓到真 bug）**：`TestFilterHallucinations_R2RepeatRunKeepsTheFirst` 第一次執行**紅** ——
  fixture 用 `"Thank you." / "thank you" / " Thank you. " / "Thank you."` 測迴圈幻覺，但初版 `normalizedSegmentText` 只做
  `ToLower + TrimSpace`，**尾端標點差異讓 run 斷成 1+1+2**，最長只有 2 段、達不到門檻 3 ⇒ 零剔除。
  修法：正規化再加 `TrimRight` 去尾端標點（`. , ! ? ; : … 。 、 ！ ？ ， ； ：`），並在註解寫下理由（卡住的 decoder 會重發同一句但尾標點漂移）。
  **這是本案唯一一次真 bug，由測試而非人眼抓到。**
- **Fault injection 反證守衛（AC #5）**：把 `guardAgainstEmptyTranscript` 的條件改成 `if true`（等同停用）→
  `TestTranscribeAudio_EmptyAfterFilteringFallsBackToUnfiltered` **轉紅**；還原後**轉綠**。證明該測試真的釘住守衛，不是恆真斷言。
- **兩個既有測試被本案的 wire 變更打紅，皆為預期並已更正**（詳見 Completion Notes 的「刻意變更的既有斷言」）。

### Completion Notes List

**實作總結（6/6 task、8/8 AC）**

1. **AC #1 / #7 —— verbose_json 型別與 SRT 組裝。** 新檔 `internal/ai/whisper_segments.go`：
   `whisperSegment` / `verboseTranscription` 未匯出型別 + `parseVerboseTranscription`（JSON 壞掉或 `segments` 為空 → `errNoSegments`）。
   `segmentsToSRT` **一 segment 一 cue**、start/end/text 原封不動、重用既有 `formatSRTTimestamp`。
   新增 `secondsToMillis` **四捨五入**（`1.9995s` → `2000ms`，直接截斷會變 `1999`）。
   AC #7.2 的等價性以 `TestVerboseJSONToFilteredSRT_DropsOutroKeepsDialogueVerbatim` 逐 cue 斷言。
   **merge 相容不是用嘴巴說的**：`TestSegmentsToSRT_OutputIsRecognisedByTheMergePath` 直接對產物跑 `isSequenceNumber` / `isTimestampLine`，再真的走一次 `MergeSRTChunks`。
2. **AC #3 —— 三條規則，純函式零 I/O。** `filterHallucinations(segs) (kept, dropped)`。
   六個常數全部採 **whisper 參考實作自己的預設值**（0.6 / -1.0 / 2.4），未自行發明數字；尾段的寬鬆門檻 0.4 與「只准用在尾端」的理由寫進註解。
   R1 silence（兩個訊號必須**同時**成立）／R2 repetition（`CompressionRatio > 2.4`）／R2b repeat_run（連續 ≥3 同句保留第一句）／R3 tail（尾端連續 ≥3 段且一路到底）。
   **未加任何關鍵字黑名單**（AC #3.6）。
3. **AC #2 —— 請求切換與 sticky 回退。** `TranscribeWithLanguage` 改請求 `verbose_json`，並拆成
   `TranscribeDetailed` / `transcribeVerbose` / `postTranscription` / `buildTranscribeBody` / `readAudioForUpload`。
   `verboseUnsupported atomic.Bool` 在 **4xx 或 200-但-body-不可用**時 latch（`Swap(true)` 保證 Warn 只發一次）；
   🔴 **5xx／逾時不 latch**，由 `TestWhisperClient_TransientFailureDoesNotLatchTheFallback` 釘住（伺服器先全 500 → 斷言旗標仍 false 且後續請求仍送 `verbose_json`）。
   回退＝**以 `srt` 重送一次**（verbose 的 `text` 無時間戳，拿不到不等於「少了過濾」而是「沒有字幕」）。
   `TestWhisperClient_RejectedVerboseFallsBackToSRTAndLatches` 斷言請求序列 **`["verbose_json","srt","srt"]`** —— 只探測一次。
4. **AC #2.3 —— 計量恰好一次。** `RecordASRWithRate` 移到取得最終答案之後、只呼叫一次。
   `TestWhisperClient_MetersAudioExactlyOnceAcrossTheFallback` 用真 WAV header 走「兩次 HTTP、一次計費」，斷言 `Snapshot().ASRCalls == 1` 且 `ASRSeconds ≈ 30`。
5. **AC #4 —— 分塊零改動即相容，並以測試證明。** 每個 chunk 是獨立 API 呼叫 ⇒ per-chunk 過濾、merge 前過濾自動成立；
   `SplitAudioChunks` / `MergeSRTChunks` / `adjustSRTTimestamps` **一行未動**。
   `TestMergeSRTChunks_HandlesAFullyFilteredChunk` 用「第二個 chunk 整段被清空」證明連號與位移仍正確（第三 chunk 拿到 `3\n00:20:05,000`）。
6. **AC #5 —— 檔案層守衛。** 新增 `ai.TranscriptionDetail` + `ai.DetailedTranscriber`（**可選**介面，`ai.CachingCompleter` 先例），
   `ASRProviderHolder` 同步轉發（否則 pipeline 持有的是 holder，type-assert 會失敗、守衛靜默永不啟動）。
   `transcribeAudio` 逐 chunk 同時收集 filtered/unfiltered，合併後由 `guardAgainstEmptyTranscript` 判斷：
   **per-chunk 全清空合法**（片尾 credits 正是招牌情境），**全檔清空則改用未過濾版 + Warn**。
   未實作 `DetailedTranscriber` 的引擎會「同一份字串回兩次」⇒ 守衛自動變 no-op，不是錯誤。

**刻意變更的既有斷言（2 條，皆為 wire 變更的直接後果）**

| 測試 | 原斷言 | 變更與理由 |
|---|---|---|
| `whisper_test.go:30` `TestWhisperClient_Transcribe_Success` | `response_format == "srt"` | 改為 `"verbose_json"`，並讓 fake server 回真 verbose_json。**這是 9-2a task 2.3 的 wire 契約變更**，註解已就地標示 |
| `retry_test.go:104` `TestWhisperClient_RetriesTransientThenSucceeds` | server 回 SRT 文字、斷言 `hits == 2` | server 改回合法 verbose_json。原本 SRT echo 會觸發回退探測、讓 `hits` 變 3，**把一個「重試」測試污染成「重試＋回退」測試**。改後該測試回到只量測重試本身 |

**新增測試（29 條）**
- `whisper_segments_test.go`（19）：`segmentsToSRT` ×4、`secondsToMillis`、`parseVerboseTranscription` ×3、`filterHallucinations` ×9（含 R1 五案表格、R2 門檻邊界、R2b 兩句不算迴圈、R3 不吃中段軟訊號、R3 短尾保留、整塊 credits 崩解）、`dropReasonCounts`、端到端 verbose→filtered
- `whisper_test.go`（8）：回退三案 + 5xx 不 latch + 計量一次 + `TranscribeDetailed` 兩案 + `DetailedTranscriber` 編譯證明 + 空 chunk merge
- `transcription_hallucination_test.go`（5）：守衛四案 + `countSRTCues`

**閘門結果（全部實跑）**

| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（Go 全 package，0 FAIL） |
| `pnpm nx test web` | ✅ exit 0，235 檔 / **2722 測試**全綠 |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 乾淨 |
| `pnpm run lint:all` | ✅ **0 errors** / 119 warnings（main 既有基準） |
| `prettier --check .` | ✅ 乾淨 |
| `gofmt -l`（本案 8 檔） | ✅ 乾淨 |
| `pnpm run test:cleanup` | ✅ No test processes found |

**強制稽核項**

- 🔗 **AC Drift: FOUND** —— **9-2a-whisper-audio-transcription task 2.3**（`Model: whisper-1, response_format: `srt``）→ 本案改為 `verbose_json`（保留 `srt` 作為 fail-soft 回退）。
  9-2a 已 `done` = frozen（Rule 20 前向唯一），不回改其檔；變更點就地標註於 `whisper_test.go` 的斷言註解，並在此記錄。
  grep 範圍：`response_format` 於 `_bmad-output/implementation-artifacts/*.md`，命中 4 檔（9-2a / 9R-9 / sub-5-1 / sub-5-2），其餘 3 檔皆為 REUSE（介面、計量、金鑰，均未變）。
- 📎 **Contract Stamps: NONE** —— 本 story 無 stamp；`ai.ASRProvider`（`asr.go:18`）**未帶戳記**＝Rule 20 前向唯一下的隱含 v0，且本案**未改動該介面**（新增的是分離的可選 `DetailedTranscriber`）⇒ 不欠 bump、不欠 ack。
  `internal/ai` 內既有的兩個 v1 戳記（`provider.go:55` `CachingCompleter`、`claude.go:380`）屬 LLM 路徑，本案未觸及。
- 🔒 **Rule 7 Wire Format: PASS** —— 0 個新錯誤碼。新增的 `errNoSegments` 是**內部 sentinel**、不進 API 回應（Rule 7 管的是 `ApiResponse.error.code` 的字串常數）⇒ `project-context.md` 與 `code-review/instructions.xml` 零編輯。
- 🔌 **Route Sync: N/A**（no backend route touched）。
- 🎭 **A11y Pre-Flight: N/A**（100% backend —— `git status` 零 `apps/web/` 檔案）。
- 🎨 **UX Verification: SKIPPED** —— no UI changes in this story。
- 🕰️ **Rule 23: N/A** —— 未觸及任何 `apps/web/src/components/**`。
- **Pre-existing failures: NONE**（`nx test api` / `nx test web` 於本分支皆 0 FAIL）。全庫 gofmt 既有漂移未動（依 9R-10a 先例，本案 8 檔乾淨）。

**AC #7.3 誠實聲明**

> 手上**沒有**真 OpenAI key + 真音訊可做「同一檔各取 `srt` 與 `verbose_json` 再 diff」的線上比對，**因此沒有做**。
> 現有證據只到「我方組裝規則與 whisper 參考實作 `write_srt` 語義一致（一 segment 一 cue、不重新斷行）」＋ fixture 層的逐 cue 等價測試。
> **殘餘風險**：若某引擎的 `srt` 渲染器與其 `segments` 陣列不一致，cue 切法會有肉眼可見的變化。上線第一份 ASR 字幕值得人工看一眼。

### Discovery Triage

**YES —— 發現 1 項超出範圍的工作。**

| Lane | 發現 | 追蹤 |
|---|---|---|
| ① expand-scope-in-place | **`runPipeline` 無條件寫 SRT、無空內容守衛**（`transcription_service.go` Phase 3）—— 本案的過濾器讓這個既有缺口第一次變得可觸發。 | 由 **AC #5** 就地吸收（`guardAgainstEmptyTranscript` + 4 條測試），非另案 |

其餘：未發現需要 lane ② 或 lane ③ 的項目。
`9R-S2-nas-whisper-benchmark-spike` 仍 in-progress，其待答問題現在多了一條「該引擎是否支援 `verbose_json`」—— 但本案的 sticky 回退已讓答案為「否」時也能正常出貨，**不構成阻擋**，故不新立條目（依 story authoring 階段的指引）。

### File List

| 檔案 | 變更 |
|---|---|
| `apps/api/internal/ai/whisper_segments.go` | **new** —— verbose_json 型別、`segmentsToSRT`、`filterHallucinations` 三規則、六個常數、`TranscriptionDetail` / `DetailedTranscriber` |
| `apps/api/internal/ai/whisper_segments_test.go` | **new** —— 19 條純函式測試 |
| `apps/api/internal/ai/whisper.go` | modified —— `verboseUnsupported atomic.Bool`；`TranscribeWithLanguage` 拆為 `TranscribeDetailed` / `transcribeVerbose` / `postTranscription` / `buildTranscribeBody` / `readAudioForUpload`；計量移到最終答案之後 |
| `apps/api/internal/ai/whisper_test.go` | modified —— 8 條新測試；`Transcribe_Success` 的 `response_format` 斷言刻意變更 |
| `apps/api/internal/ai/retry_test.go` | modified —— fake server 改回合法 verbose_json，讓重試測試回到只量測重試 |
| `apps/api/internal/services/asr_provider_holder.go` | modified —— 轉發 `TranscribeDetailed` + 編譯期證明 |
| `apps/api/internal/services/asr_provider_holder_test.go` | modified（CR M2）—— +3 條轉發測試 |
| `apps/api/internal/services/retry_service_test.go` | modified（CR L3，**Pre-existing fix**）—— `MockRetryRepository` 11 個方法加 `sync.RWMutex`，修掉乾淨 main 上就存在的 data race |
| `apps/api/internal/services/transcription_service.go` | modified —— `transcribeAudio` 收集 filtered/unfiltered、`transcribeOne`、`guardAgainstEmptyTranscript`、`countSRTCues` |
| `apps/api/internal/services/transcription_hallucination_test.go` | **new** —— 5 條守衛測試 |
| `_bmad-output/implementation-artifacts/9R-5-vad-hallucination-filter.md` | modified —— tasks 全勾、Dev Agent Record、File List、Change Log、Status → review |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | modified —— `9R-5` ready-for-dev → in-progress → review |

### Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **Task 1（AC #1 / #7）** —— `whisper_segments.go` 新檔：verbose_json 型別 + `parseVerboseTranscription` + `segmentsToSRT`（一 segment 一 cue，重用 `formatSRTTimestamp`）+ `secondsToMillis` 四捨五入。merge 相容以真 `MergeSRTChunks` 往返證明。 |
| 2026-08-21 | **Task 2（AC #3）** —— 三條規則純函式化，六個常數採 whisper 參考實作預設值。**RED 抓到真 bug**：正規化未去尾端標點導致 repeat_run 永不成立，已修並補理由註解。無關鍵字黑名單。 |
| 2026-08-21 | **Task 3（AC #2）** —— 請求改 `verbose_json`；`verboseUnsupported atomic.Bool` sticky 回退（4xx／body 不可用才 latch，**5xx／逾時不 latch**）；回退＝以 `srt` 重送一次；`RecordASRWithRate` 每音檔恰好一次。8 條 httptest 測試含請求序列斷言。 |
| 2026-08-21 | **Task 4（AC #4）** —— 分塊路徑**零程式碼改動**，以「整個 chunk 被清空仍連號正確」測試證明而非宣稱。 |
| 2026-08-21 | **Task 5（AC #5）** —— `ai.TranscriptionDetail` / `ai.DetailedTranscriber` 可選介面 + holder 轉發 + `guardAgainstEmptyTranscript`（per-chunk 全清空合法、全檔清空改用未過濾版）。fault injection 反證守衛有效。 |
| 2026-08-21 | **Task 6（Rule 12 / 15）** —— 閘門全綠：`nx test api` 0 FAIL、`nx test web` exit 0 / 2722 測試、`nx lint api` 乾淨、`lint:all` 0 errors、prettier、本案 8 檔 gofmt、test cleanup 無殘留。 |
| 2026-08-21 | **Rule 24 lane ①** —— `runPipeline` 無空內容守衛由 AC #5 就地吸收。🔗 AC Drift FOUND：9-2a task 2.3 `srt` → `verbose_json`（9-2a done=frozen，不回改）。Status → review。 |

---

## Senior Developer Review (AI)

**Reviewer:** Bob (SM) 代跑 `/code-review`，claude-opus-5[1m] · **Date:** 2026-08-21 · **Outcome:** APPROVED WITH FIXES

> ⚠️ **同 context 自審警告。** 由**實作者本人在同一 session** 執行，非跨模型獨立審查。工作流程自己的建議是換 LLM。
> **強烈建議 PR 上另跑一輪跨模型審查** —— 不過本輪已抓到一個 HIGH，見下。

**Git vs Story File List：0 落差**（10 個檔案逐一對上，含 review 期間新增的 2 個）。

### 強制閘門

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **PASS** —— 4 個 Go 生產檔以 `Err[A-Z]\w*[^=]*= *"[A-Z][A-Z0-9_]*"` 掃描，**0 個錯誤碼常數**。`errWrongJSONShape` 是內部 sentinel（`errors.New` 語義，不是 wire 字串）⇒ `project-context.md` 與 `code-review/instructions.xml` 零編輯 |
| 🔒 Rule 20 Contract Bump | **N/A** —— diff 中 `[@contract-vN→vM]` 命中數 **0** |
| 🔒 Rule 25 Mega-line | **N/A** —— `project-context.md` 未被修改 |

### findings：1 HIGH / 3 MEDIUM / 3 LOW —— **7/7 全修**

| # | 嚴重度 | 發現 | 處置 |
|---|---|---|---|
| **H1** | 🔴 **HIGH** | **一個安靜的音訊分塊會永久關掉整個過濾器。** OpenAI 對真正無聲的音訊回的是**合法的** verbose_json：`{"duration":600,"text":"","segments":[]}`。原實作把「`segments` 為空」一律當成「這台引擎不支援 verbose_json」⇒ **latch sticky 旗標** ⇒ 之後**每一個分塊、每一個媒體項目**都退回 `srt`、幻覺過濾**靜默停止運作**，而且每次還多付一次無用的 `srt` 請求。諷刺之處：**十分鐘的無聲片尾正是本 story 存在的理由**，卻是觸發這個 bug 的第一個輸入。**已寫探針測試實地重現**（旗標變 true、後續分塊拿到 `from srt`、請求序列 `[verbose_json srt srt]`）。 | ✅ 修 —— 區分兩種空 segments：**有 text 沒 segments** = 引擎給錯 JSON 形狀（`errWrongJSONShape`，latch + 回退，因為字有了但沒時間戳）；**text 也空** = **真的無聲**（回傳空的 `TranscriptionDetail{Filtered:true}`、**不 latch、不重送**、記一行 `no speech` INFO）。新增 `TestWhisperClient_SilentChunkDoesNotLatchTheFallback` 迴歸守衛，斷言旗標仍 false、後續分塊仍走 verbose_json、**且完全沒有 srt 請求**。原本那條斷言舊行為的 `EmptySegmentArrayFallsBack` 改寫為 `TextWithoutSegmentsFallsBack` |
| **M1** | MEDIUM | **R3 尾段實作比 AC #3.5 寫的更兇。** AC 要求「該連續段**每一段** `NoSpeechProb > 0.4`」，但實作額外允許 `reasons[i] != ""`（已被 R1/R2 剔除）也算入尾段連續段。後果：檔案中段一個因壓縮比被剔的段，會**架橋**讓兩個原本不該被碰的安靜結尾句一起消失（2 段本來低於門檻 3）。 | ✅ 修 —— 尾段迴圈改為只認 `NoSpeechProb > 門檻`，每一段都得自己站得住。POC 測試不受影響（三段 outro 的 0.55/0.72/0.61 本來就全數過關）。新增 `TestFilterHallucinations_R3TailRunIsNotBridgedByAnAlreadyDroppedSegment` 釘住 |
| **M2** | MEDIUM | **AC #5 最脆弱的一環零測試覆蓋。** `ASRProviderHolder.TranscribeDetailed` 是整條守衛鏈的關鍵接點 —— pipeline 持有的是 **holder** 不是 client，holder 若沒轉發，`transcribeAudio` 的 type assertion 就失敗、守衛**靜默永不啟動**（我自己的程式碼註解就是這樣寫的），而且**不會有任何錯誤或 log**。這麼危險的接點卻**一條測試都沒有**。 | ✅ 修 —— 新增 3 條：`SatisfiesDetailedTranscriber` 編譯期證明、`TranscribeDetailedForwardsToTheClient`（真 httptest，斷言 `Filtered` / `SegmentsIn=5` / `SegmentsKept=2` / **`Unfiltered` 含 outro**＝守衛依賴的復原文字確實穿過了 holder）、`PropagatesUnconfigured` |
| **M3** | MEDIUM | **`SegmentsKept` 與實際 cue 數可能對不上。** `segmentsToSRT` 會跳過空白文字的 segment，但 `SegmentsKept` 照數 ⇒ Info log 的 `segments_kept` 與 `drop_ratio` 會與磁碟上的 cue 數不符，操作者對帳時被誤導。 | ✅ 修 —— 空白文字 segment 在 **parse 時**就丟掉，`SegmentsIn` / `SegmentsKept` / 渲染 cue 數三者永遠一致；`segmentsToSRT` 的跳過邏輯降為防禦性。新增 `TestParseVerboseTranscription_DropsBlankTextSegments` |
| **L1** | LOW | **holder 內「provider 不是 `DetailedTranscriber`」的分支今天不可達** —— `Get()` 永遠回 `*ai.WhisperClient`，而它必定實作該介面。 | ✅ 修（記錄型）—— 保留為防禦性程式碼（未來替代引擎），註解已說明；不寫無法觸發的測試去假裝有覆蓋 |
| **L2** | LOW | **未 commit 狀態未記錄**（CR 步驟 3 的透明度項）。 | ✅ 修（記錄型）—— 截至 review 完成，**10 個檔案**位於 `feat/9R-5-hallucination-filter` 且**尚未 commit**，交由 `/ship` 處理 |
| **L3** | LOW | **重構過的 client 沒跑過 `-race`。** 本案新增了 `atomic.Bool` 與跨請求共享狀態，值得驗一次。 | ✅ 跑了 —— `internal/ai` **乾淨**；但 `internal/services` 報出一個 **DATA RACE**。查證為**既有問題**（`git stash` 後在乾淨 main 上同樣重現）：`MockRetryRepository`（`retry_service_test.go`）的 map 同時被測試 goroutine 與 `RetryScheduler.TriggerImmediate` 生出的 goroutine 存取，**11 個方法全部無鎖**。依 dev-story Step 7「快速可修就地修」：加 `sync.RWMutex` 並鎖住全部存取點，修後 `-race` 兩個套件皆綠。記為 **Pre-existing fix**，非本案引入 |

### 看過但**判定不是** finding（避免下一輪重查）

- **`markVerboseUnsupported` 的 `Swap(true)`** —— 併發下最多多送幾次探測，不影響正確性；分塊迴圈本來就是序列的。
- **`postTranscription` 的 `lastStatus` 閉包捕捉** —— `retryTransient` 同步執行，無 race；`-race` 已實測乾淨。
- **回退路徑的記憶體** —— 第一份 body 在 `postTranscription` 回傳後即可回收；峰值仍是 audio + 一份 body，與改動前相同。
- **Bounded I/O**（Rule 安全檢查）—— `io.ReadAll(io.LimitReader(file, WhisperMaxFileSize))` 且大小守衛**在讀取之前**；回應讀取沿用 `WhisperMaxResponseSize`。比改動前更嚴格。
- **`ai.ASRProvider` 未被改動** —— `git diff internal/ai/asr.go` 為空；新增的是分離的可選 `DetailedTranscriber`。
- **分塊三個函式未被改動** —— `SplitAudioChunks` / `MergeSRTChunks` / `adjustSRTTimestamps` 的 diff 為零行。
- **`secondsToMillis` 的四捨五入 vs Python `round()` 的銀行家捨入** —— 僅在剛好 .5ms 邊界不同，對字幕無感，不列為 finding。

### 修後閘門（全部重跑）

| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（0 FAIL） |
| `pnpm nx test web` | ✅ exit 0，235 檔 / 2722 測試 |
| `go test -race ./internal/ai/ ./internal/services/` | ✅ **兩個套件皆乾淨**（修掉既有 mock race 之後） |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 |
| `pnpm run lint:all` | ✅ 0 errors / 119 warnings（既有基準） |
| `prettier --check .` | ✅ |
| `gofmt -l`（本案檔案） | ✅ 乾淨 |
| `pnpm run test:cleanup` | ✅ No test processes found |

**最終 `git diff --numstat`（apps/api）**：`whisper.go` 220+/40− · `whisper_test.go` 280+/2− · `transcription_service.go` 68+/4− · `asr_provider_holder.go` 26+/0− · `asr_provider_holder_test.go` 51+/0− · `retry_service_test.go` 43+/1−（pre-existing race 修復）· `retry_test.go` 5+/1− · 新檔 3 個（`whisper_segments.go` / `whisper_segments_test.go` / `transcription_hallucination_test.go`）。

### Action Items

無 —— 1 HIGH + 3 MEDIUM + 3 LOW **全數在 review 內修畢**，測試同步補齊（新增 6 條，總計 **35 條**）。
唯一 carry-forward 是 story 已載明的 **AC #7.3 缺口**（無真 key/音訊，未做 `srt` vs `verbose_json` 的線上 cue 切法比對）—— 上線第一份 ASR 字幕仍值得人工看一眼。

### Change Log（review 追加）

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **CR 修復 7/7** —— **H1（實地重現後修復）**：安靜分塊的空 `segments` 不再被誤判為引擎不支援，改以「有無 text」區分「給錯 JSON 形狀」與「真的無聲」，後者不 latch 不重送；+1 迴歸守衛、改寫 1 條斷言舊行為的測試。**M1** 尾段連續段回歸 AC #3.5 的嚴格定義（每段自證），+1 測試。**M2** holder 轉發補 3 條測試（含 `Unfiltered` 穿透斷言）。**M3** 空白文字 segment 於 parse 時剔除，計數與 cue 數永遠一致，+1 測試。**L1/L2** 記錄型。**L3** 跑 `-race`，發現並就地修復**既有**的 `MockRetryRepository` 無鎖 map（11 方法加 `sync.RWMutex`），修後兩套件 `-race` 皆綠。閘門全部重跑綠。Status review → done。 |
