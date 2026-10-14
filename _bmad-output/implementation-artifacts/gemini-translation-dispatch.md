# Story gemini-translation-dispatch: 字幕翻譯可以選 Gemini（用自己的免費額度），預設仍是 Claude

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-14。升自 backlog-gemini-translation-dispatch（sub-6-8a CR H2，2026-09-04）。
     證據：eval-gemini-free-tier-2026-10.md——gemini-3.5-flash-lite 免費方案在 golden-v2 考卷 A 級（2.0%／89.5%），跟 Sonnet 持平。
     ⚖️ Alexyu 2026-10-14：「Gemini 跟 ElevenLabs 都應該是可選的免費模型可以用」；founder-pm 提醒：對外說「用自己的免費額度」，預設路徑維持填一把金鑰就能用。
     FB 上已承諾（Ming Jwo 那串）：「Vido 會把 Gemini 加進可選的翻譯模型，預設先不動」。行號為 main 9133224e。 -->

## Story

身為不想付 Claude 錢的 Vido 使用者，
我要在金鑰頁填自己的 Gemini 金鑰，然後在產生字幕時選 Gemini 3.5 Flash-Lite 當翻譯模型，
這樣我可以用 Google 的免費額度翻字幕，而且 Vido 會自己照免費方案的速率慢慢送，不會一半被擋回來留著英文。

## 查到的事（2026-10-14）

- 翻譯只會送 Claude：`main.go` `services.NewTranslationService(claudeHolder, sseHub)`；`TranslationService.provider` 是一個 `ai.TextCompleter`（`translation_service.go:219-225`）；模型由 ctx 帶（`ai.WithModelID`），`ClaudeProviderHolder.GetFor` 明確拒絕非 Claude 的 model（`claude_provider_holder.go:102-116`）。
- 模型目錄 `ai/catalog.go`：`gemini-3.5-flash-lite` 等已在 `modelMetadata` 與 `defaultLLMPricing`（`budget.go:61-63`），`ai.ProviderOf(model)` 可分辨。`ModelCatalogService.Available`（`model_catalog.go:57`）現在只在有 Claude 金鑰時列 Claude 的模型，Gemini 一律不列（sub-6-8a CR H2 的處置，註解標了位置）。
- Gemini 的 provider 已存在：`ai/gemini.go` `NewGeminiProvider(apiKey, opts…)`（base URL、model、timeout、Governor），目前只給檔名解析用。要確認它有沒有實作 `CompleteTextWithUsage`（翻譯記帳走這個）——沒有的話翻譯花費會記不到。
- Gemini 金鑰不在 `KeyResolver`（`backlog-gemini-key-in-resolver`）：只能來自 `GEMINI_API_KEY` 環境變數，設定頁沒欄位。asr-scribe-b 剛示範過加一種金鑰要動哪些檔（resolver、key settings、handler、config、前端 `KEY_ROWS`）。
- 免費方案速率（2026-10-14 實測）：每分鐘 5 次；3.8-flash 每天 20 次、3.5-flash-lite 每天額度夠考 200 句。翻譯程式的重試等 1～2 秒，不控速一定撞。`cmd/grade` 的 `--pace` 是參考做法（`cmd/grade/paced.go`）。Governor（`ai/governor.go`，`AIRatePerSec`）是全域的，不分 provider。
- 評分表 `ai/model_ratings.json` 只有 eval-1 的兩筆（judge_model 寫 human，#775 待修）；`grade --merge` 拒收 `local:` 開頭的報告。Gemini 的考卷報告是 `local:gemini-3.5-flash-lite`。

## Acceptance Criteria

1. **金鑰種類加 `gemini`**（env `GEMINI_API_KEY` 已存在、secret `gemini.api_key`），金鑰頁多一列「Gemini（選填）」，說明寫「填了就能在翻譯模型裡選 Gemini；免費方案每分鐘 5 次，Vido 會自動放慢」。不進首次設定精靈。順便結掉 `backlog-gemini-key-in-resolver`。
2. **翻譯分派**：`TranslationService` 依 `ai.ProviderOf(model)` 挑 holder——Claude 走現在的 `ClaudeProviderHolder`，Gemini 走新的 `GeminiProviderHolder`（同樣 per-call 解析金鑰、fingerprint 快取、共用 Governor）。沒填 Gemini 金鑰時選到 Gemini 模型要在**同意之前**就擋（目錄不列），而不是付錢後 404。
3. **目錄**：`ModelCatalogService.Available` 改成「有哪家的金鑰就列哪家的模型」；預設模型不變（Sonnet）。Gemini 的列顯示品質等級（見 AC 6）。
4. **控速**：Gemini 的呼叫之間最少隔 `VIDO_GEMINI_MIN_INTERVAL`（預設 13 秒，對應免費方案 5 RPM；付費方案的人可以設 0）。只限 Gemini；Claude 不受影響。同意畫面的「預估時間」要把這個算進去（一集約 18 分鐘）。
5. **記帳**：Gemini 的翻譯花費照 `defaultLLMPricing` 記（免費方案實際 $0，但 Vido 不知道使用者是哪個方案，照表記、文案說「免費方案實際不計費」）。
6. **評分表**：`grade --merge` 接受 `local:<id>` 報告當 `<id>` 在 `ai.Catalog()` 裡且 provider 不是 Claude（去掉前綴、`judge_model` 照報告寫 AI 評）；把 `eval/golden-v2-2026-10/gemini-3.5-flash-lite.json` 合進去，選模型畫面顯示「A・AI 評分」。
7. **文案**：金鑰頁、模型選單、README 一律寫「用你自己的 Gemini 免費額度」，不寫「免費」。
8. **測試**：分派（Claude 模型→Claude holder、Gemini 模型→Gemini holder、沒金鑰→不列）、控速（兩次呼叫間隔 ≥ 設定值、ctx 取消會停）、目錄依金鑰列、merge 規則。既有 Claude 路徑測試不改。
9. **正式機驗證**：NAS 填 Gemini 金鑰，用 `vido-eval-clip` 丟棄容器選 Gemini 翻 See S01E02 一整集（有內嵌英文字幕，走抽取＋翻譯），看沒有被 429 擋回來的英文句、花費記錄、時間。

## 不做

- 不改預設模型。
- 不做「自動退回 Claude」：Gemini 被擋就照 Vido 既有的部分失敗處理（留英文、標記），不偷偷換家扣錢。
- 不做付費方案偵測。

## Tasks / Subtasks

- [ ] T1 金鑰種類 gemini（後端＋前端列）
- [ ] T2 `GeminiProviderHolder`＋`TranslationService` 依 provider 分派
- [ ] T3 目錄依金鑰列、Gemini 控速（min interval）＋預估時間
- [ ] T4 `grade --merge` 規則＋合併 Gemini 報告
- [ ] T5 文案、README
- [ ] T6 測試、正式機驗證

### Time-dependent visual coverage

金鑰頁多一列、模型選單多一列 → visual 基準會變（故意的）。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List
