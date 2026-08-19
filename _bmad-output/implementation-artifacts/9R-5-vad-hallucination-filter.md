# Story 9R-5: VAD 幻覺過濾 —— Whisper 靜音/片尾幻覺的後置過濾器

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P1 M · **PROVEN**（POC 實證，非假設）
**Source:** sprint-status `9R-5-vad-hallucination-filter`；ADR `adr-subtitle-route-c-generation` Decision 5 第 5 項裁定在案（「VAD / tail-detection post-filter」）。2026-08-19 策略重審列為品質欠帳第一條（ASR 路徑佔抽樣庫 68.3%，此缺陷在唯一繁中來源的多數路徑上）。

---

## Story

As a NAS owner whose credits-roll has no dialogue,
I want hallucinated cues (like the POC's invented "like & subscribe" outro on silent credits) filtered out of ASR output before translation,
so that generated subtitles don't contain fabricated text that then gets translated (and paid for) as if it were dialogue.

---

## Context —— 現況與設計空間

1. **POC 實證**：Whisper 對無聲片尾憑空生成「like & subscribe」outro（`subtitle-v4-replan-and-feasibility-audit-2026-06.md:188`；ADR Decision 5.5 裁定後置過濾入 scope）。
2. **現行輸出拿不到信心值**：`WhisperClient.TranscribeWithLanguage` 以 `response_format=srt` 請求（`ai/whisper.go:217`），回傳純 SRT 字串 —— **沒有** `no_speech_prob` / `avg_logprob` / `compression_ratio` 可供過濾判斷。
3. **分塊路徑**：>24MB 音訊走 `SplitAudioChunks` → 逐塊轉錄 → `MergeSRTChunks`（`whisper.go:414-432`）只重編號與位移時間戳 —— 過濾須與此路徑相容。
4. **下游串接**：過濾發生在轉錄輸出、翻譯之前（`transcription_service.go` transcribeAudio → translateSRT）—— 幻覺 cue 越早刪，越不會付翻譯費。與 bugfix-j（partial-failure 誠實 verdict）無耦合。

## Acceptance Criteria

### AC #1 — 取得判斷依據：改用 `verbose_json`
- `TranscribeWithLanguage` 改請求 `response_format=verbose_json`，取回逐 segment 的 `text/start/end/no_speech_prob/avg_logprob/compression_ratio`，SRT 由本地組裝（時間戳格式與現行 `adjustSRTTimestamps` 邏輯一致）。
- 🛡️ **Fail-soft（Rule 13）**：provider 回應缺 segments 或欄位（非完整 OpenAI 相容的本地 server）→ 回退現行 SRT 路徑、跳過過濾、`slog.Warn` 一行 —— 生成永不因過濾器而失敗。

### AC #2 — 過濾規則（常數化，附 rationale 註解）
- **靜音幻覺**：`no_speech_prob` 高且 `avg_logprob` 低的 segment 剔除（閾值為具名常數，註解引用 POC 案例）。
- **重複幻覺**：連續 ≥N 個 text 相同（正規化後）的 segment 保留首個、其餘剔除（Whisper 迴圈幻覺的典型型態）。
- **尾段偵測（ADR 用語 tail-detection）**：最後一段高信心語音之後、只剩低信心 segment 的尾巴整段剔除。
- 每個被剔除的 segment 記 `slog.Debug`（start/end/text/依據），總剔除數記 `slog.Info`。

### AC #3 — 分塊相容
- 過濾以 **chunk 為單位在 merge 前**執行（各 chunk 邊界時間戳未位移時判斷最單純）；merge 後重編號照舊。單檔（不分塊）路徑等價。

### AC #4 — 測試
- fixture：含 POC 型幻覺尾段（無聲區高 `no_speech_prob` + outro 文案）的 verbose_json 樣本 → 尾段被剔、正常對白 byte-preserved。
- 回歸釘：全高信心輸入 → 零剔除、SRT 與現行輸出等價；provider 缺欄位 → 回退路徑走通。
- 既有 whisper/transcription 測試全綠。

### AC #5 — 範圍紅線
- 不引入本地 VAD 模型/依賴（silero 等）—— M 級 story 用 API 自帶信心值即可；本地 VAD 屬 Tier-2。
- 不動 ASR 併閘門管線的結構（`backlog-asr-leg-unify-gated-pipeline` 另案）。

## Tasks / Subtasks

- [ ] Task 1: AC #1 verbose_json 遷移＋本地 SRT 組裝＋fail-soft 回退
- [ ] Task 2: AC #2 三條過濾規則＋常數＋log
- [ ] Task 3: AC #3 分塊路徑整合
- [ ] Task 4: AC #4 fixture 與回歸測試
