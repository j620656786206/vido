# Disc：聽聲音時先把角色名交給語音辨識，人名不再一集七種寫法（聽的那一半）

Status: review

**Source:** See S01E02 10/5 實測：Jerlamarel 6～7 種寫法（傑拉·莫瑞爾、杜拉·莫瑞爾、丘拉·莫瑞爾…）、Paris／Maghra 各兩種、Baba Voss → 「沃斯爸爸」。10/6 party mode 裁定分兩半修：**聽**（本單：把名字交給語音辨識）與**翻**（名詞表；走片內字幕時已一致，污染問題另立 `disc-2026-10-asr-harvest-pollutes-glossary`）。

## Story

身為 Vido 的使用者，
我希望片子只能聽聲音時，語音辨識事先知道這部劇的角色和演員叫什麼，
這樣同一個人名不會每句聽成不同拼法、翻出來也不會一集七種寫法。

## 查到的事（2026-10-06，逐一開檔確認）

- `ai/whisper.go:388-411` 的 `buildTranscribeBody` 只送 `model`、`response_format`、`language`——**沒有 `prompt`**。whisper 的 `prompt` 欄位會影響拼字（只讀最後約 224 個 token）。
- `ASRProvider`（`ai/asr.go:18-24`）是別的引擎也要實作的介面，不想為此加參數；每次執行的 model 與 Budget 都走 ctx（`ai/model_context.go`、`ai.WithBudget`）。
- 角色名在 TMDb credits：`models.Credits.Cast[].Character`／`.Name`（`models/movie.go:171-192`），`movie.Credits`／`series.Credits`（`movie.go:301`、`series.go:76`）。翻譯那半已用 `CastLabels` 餵 prompt（`transcription_service.go:1800`）。
- 名詞表來源有五種（`models/glossary.go:15-27`）；`source=subtitle` 是翻譯完自動寫回的，See 那 20 條裡有 7 種錯的 Jerlamarel，**不能**餵回去。`glossaryRepo.ListByScope`（`repository/glossary_repository.go:35-36`）回含 `Source` 的列。
- ASR 呼叫點：`runPipeline` → `transcribeAudio(phaseCtx, …)`（`transcription_service.go:945-958`）；`glossaryKeyFor(mediaID, episodeRowFor(...))` 解析劇集的父劇 id（`:1703-1722`）。

## 設計

- `ai.WithASRPrompt(ctx, prompt)`／`ai.ASRPromptFromContext(ctx)`（ctx 值，與 model id 同款）；`ai.BuildASRPrompt(names)`：去空白、不分大小寫去重、保序、上限 40 個名字／700 個字（whisper 視窗；重要的放前面）。
- `WhisperClient.buildTranscribeBody` 多一個 `prompt` 參數，`postTranscription` 從 ctx 取；空就不送欄位（跟以前位元相同）。
- `TranscriptionService.asrPromptFor`：角色名（billing 序）→ 演員名 → 名詞表 `source != subtitle` 的 `term_src`；任何查詢失敗就少一部分，不會讓 run 失敗。`runPipeline` 在 `transcribeAudio` 前用 `ai.WithASRPrompt` 掛上，log `asr prompt built`（names／runes）。
- 不碰 `ASRProvider` 介面、不碰續跑／chunk cache key。

## Acceptance Criteria

1. `BuildASRPrompt`：去重、保序、空字串跳過、超過上限截斷且保留前面。ctx 來回。
2. whisper 多部請求：ctx 沒 prompt → 沒有 `prompt` 欄位；有 → 原字送出。
3. `asrPromptNames`：角色 → 演員 → 可信名詞表；`source=subtitle` 永遠不進去；全空回空。
4. `go test ./...`、vet、staticcheck、lint:all 全綠。
5. **實測（部署後，隨「修一半」三張一起用 10 分鐘片段驗證，約 $0.3）：** log 有 `asr prompt built`；Jerlamarel 的寫法種數 7 → 1～2。

## Tasks / Subtasks

- [x] T1 `ai` ctx 值＋`BuildASRPrompt`＋測試（AC #1）
- [x] T2 whisper `prompt` 欄位＋測試（AC #2）
- [x] T3 服務端 `asrPromptNames`／`asrPromptFor`＋接線＋測試（AC #3）
- [x] T4 檢查（AC #4）

## Dev Notes

- 不要把 `source=subtitle` 的名詞餵進去（會教它自己的錯字）。
- 不要把 prompt 塞進 chunk cache key：prompt 對同一部片是決定性的，續跑語意不變。
- 這是在補語音模型現在的弱點（founder-pm 判斷）：做到可用就好，不要再疊更多啟發式。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **Adversarial CR（2026-10-06，fresh agent）2M/3L＋nits，全部修掉：**
  - M1 影集整季跑（`series`）時原本的守門 `glossaryKey != mediaID` 會把它自己擋掉、拿不到演員表 → 改成只擋「找不到母劇的單集」（跟 `mediaMetadataFor` 一樣），加 `TestASRPromptFor_ReadsCreditsPerMediaType` 五個案例。
  - M2 whisper 收到 prompt 後，在無聲／配樂處可能把名單原文「聽」出來 → 新規則 `prompt_echo`（`filterPromptEcho`：段落文字正規化後是名單的子字串且 ≥12 字元就丟；單一人名、提到人名的對白保留），加 `TestFilterPromptEcho`。
  - L2 查電影／影集失敗原本無聲 → 加 Warn log（`asr prompt: … lookup failed`）。
  - L1 chunk 快取 key 沒含 prompt：接受（有 prompt 的新跑才會重聽；舊快取段落本來就沒有 prompt），在 story 記下。
  - nits：角色名去掉括號尾巴（`Baba Voss (voice)`）、跳過 Self／Himself／Herself／Narrator（`promptCharacterName`）。
- 🔗 AC Drift: FOUND — 9R-2 AC #3「multipart 帶 language」不變；多一個可選的 `prompt` 欄位（沒有 ctx prompt 時 body 位元相同，既有 `TranscribeWithLanguage_MultipartCarriesLanguage` 不改）。
- 📎 Contract Stamps: NONE（`ASRProvider` 介面未動）。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/ai/asr_prompt.go`（新）、`asr_prompt_test.go`（新）
- `apps/api/internal/ai/whisper.go`（改）、`whisper_segments.go`（改：`prompt_echo`）、`whisper_test.go`（改）
- `apps/api/internal/services/transcription_asr_prompt.go`（新）、`transcription_asr_prompt_test.go`（新）
- `apps/api/internal/services/transcription_service.go`（改：ASR 前掛 prompt）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-proper-names-inconsistent.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | CR 2M/3L 修掉：series 守門、`prompt_echo` 規則、lookup 失敗 log、角色名清理；全綠。 |
| 2026-10-06 | create-story＋dev 同日（聽的那一半）：ctx prompt、whisper `prompt` 欄位、角色／演員／可信名詞表組名單；全綠，狀態 review。翻的那一半（名詞表污染）在 `disc-2026-10-asr-harvest-pollutes-glossary`。 |
