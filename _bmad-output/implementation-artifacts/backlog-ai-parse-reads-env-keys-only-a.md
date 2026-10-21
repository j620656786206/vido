# Backlog -a：AI 檔名解析改讀網頁上填的金鑰，花的錢記下來（後端）

Status: done

**Source:** `backlog-ai-parse-reads-env-keys-only`（2026-10-19 由 disc-2026-10-setup-wizard-byok 建單時查到）。⚖️ 2026-10-21 party mode（founder-pm、founder-growth、BMAD）把它列為「首次安裝修正」第 3 張，Alexyu 選 **A：打開，而且把這筆錢記進花費紀錄**。

**拆單（story splitting rule：跨前後端、單邊超過 3 個 task）：**

- **-a（本單，後端）**：每次解析都依當下的金鑰決定用哪一家；花的錢逐筆記下來；花費 API 回傳本月檔名解析的次數和金額。
- **-b（`backlog-ai-parse-reads-env-keys-only-b`，設計稿＋前端）**：「本月 AI 花費」卡片和首頁的「本月 $x」顯示這筆錢。要先等 Sally 改設計稿。

-a 可以先合併。部署時機由 Alexyu 決定；-a 先上線的話，錢已經會記下來，只是 -b 上線前畫面還看不到。

## 現在會怎樣（2026-10-21 查證）

- `services.NewAIService`（`ai_service.go:81-110`）在啟動時讀一次 `config` 的環境變數金鑰（`GetGeminiAPIKey`／`GetClaudeAPIKey`，只讀 env）。沒有金鑰時回傳 nil，`main.go:328-337` 就印「AI parsing disabled」。在設定精靈或設定頁存的金鑰（加密存在 secrets store）**它都看不到**。設定頁的 hint 卻寫「也用於 AI 檔名解析」（`ApiKeysForm.tsx:66`、`:76`）。
- **正式機自己就是這樣**：金鑰都存在網頁上，啟動 log 是「AI service not configured - no API keys set」。AI 檔名解析在正式機從來沒開過。
- 開機時決定一次的還有：health 的 `aiPingable`（`main.go:345-349`）、parser 的 `parserAI`（:391-395）、`KeywordService` 的接線（:420-424）、`ai_cache` 清理（:576-577）。
- **潛在開機失敗**：`AI_PROVIDER` 預設是 `gemini`（`config.go:231`）。只設 Claude 環境變數時，`HasAIProvider` 為 true，但 `ai.NewProvider` 會因為沒有 Gemini 金鑰而回錯（`factory.go:45-48`），結果 `main.go` 直接 `os.Exit(1)`。這是看程式推論的，沒有實際跑過；本單要用測試確認並修好。README:127 叫只有 Claude 的人自己填 `AI_PROVIDER=claude`，就是在繞過這個問題。
- **花費不記帳是之前刻意定的**：sub-5-1 AC #4 裁定解析路徑「unmetered-by-DESIGN」（`ai_service.go:1-10` 檔頭、`docs/deployment.md:139-148`）。理由是解析是背景工作，沒有同意的時間點，也沒有「一次 run」可以設上限；如果掃描途中被上限擋住，片庫會只解析一半，而且沒有畫面說明原因。另外開了 `backlog-parse-path-ai-metering` 等以後要不要加計數器。

## ⚖️ 裁定

**Alexyu 2026-10-21 選 A**，推翻 sub-5-1 AC #4 的「不記帳」那一半，**但保留「不設上限」那一半**。

**Winston 的做法：**

1. **每次呼叫都決定用哪一家（金鑰熱載）。** 照 `ClaudeProviderHolder`（`claude_provider_holder.go:77-158`）的模式，新寫一個回傳 `ai.Provider`（有 `Parse`）的 holder。現有的 holder 回傳的是 `ai.TextCompleter`，沒有 `Parse`。
   - 每次呼叫都透過 `KeyResolver` 讀金鑰。順序照舊：secrets store → env。
   - 用「金鑰＋model＋provider」做指紋，指紋不同就重建 client。
   - 沿用 `main.go:322` 那一個 `aiGovernor`。Governor 註解寫明不能每次請求都新建。
2. **用哪一家：** `AI_PROVIDER` 指定的那一家有金鑰就用它；沒有的話用另一家；兩家都沒有 → 未設定。這樣只有 Claude 金鑰的人不用再自己填 `AI_PROVIDER`，也修好上面的開機失敗。
3. **只記帳、不設上限：** `AIService` 打 provider 的每個呼叫點（`ParseFilename`、`ParseFansubFilename`、`GenerateKeywords`），在呼叫前把 ctx 換成自己的 `ai.WithBudget(ctx, ai.NewBudget(0))`。`NewBudget(0)` 代表沒有上限，`Exceeded` 永遠是 false，但金額照樣累計（`budget.go:214-216`）。呼叫後讀 `SpentUSD()`。
   - 快取命中時不會打 provider，也不會寫紀錄。
   - **一定要用自己的 Budget，不能沿用 ctx 上原本的**：如果解析剛好是在某次字幕 run 裡被呼叫，這筆錢不能算進那次 run 的上限和金額，否則同一筆錢會記兩次，還可能讓字幕 run 被解析的錢擋住。
4. **逐筆記下來：** 新增 migration（目前最新是 046）建 `ai_side_spend` 表。欄位：
   - `id`
   - `kind`：`parse`／`fansub`／`keywords`，用 CHECK 限制
   - `provider`
   - `model_id`
   - `input_tokens`
   - `output_tokens`
   - `spent_usd`（REAL NOT NULL）
   - `created_at`：UTC，照 migration 042 以後的時間格式慣例
   
   有打 provider 而且 `SpentUSD() > 0` 才寫一列。
   - 呼叫失敗但有回 usage 的，也照記（錢已經花了）。
   - 寫入失敗只記 warn，**絕不能讓解析失敗**。
   - 不塞進 `subtitle_runs`：那張表的 `media_id`、`media_type` 是 NOT NULL，`media_type` 的 CHECK 只能是 `movie/series/episode`（migration 030）。
5. **花費 API：** `GET /subtitles/spend`（`subtitle_spend_service.go:129`）的 `SubtitleSpendSummary` 加兩個欄位：`parse_calls`（int）、`parse_usd`（float64）。統計範圍是同一個月份區間（`from`／`to`），三種 kind 合在一起算。`?batch_id=` 的收據不受影響。

**Sally（畫面的部分，-b 做）：**

- 「本月 AI 花費」卡片的頭條那一行，在「語音辨識」右邊加第三格「檔名解析 N 次 · $x」。這個月有呼叫才顯示。
- 不放進「依模型」，那一欄是字幕 run 的統計。
- 首頁的「本月 $x」把這筆錢算進去。
- 名稱用「檔名解析」，跟設定頁 hint、活動記錄的「解析」一致。關鍵字產生也是為了認出是哪部片，所以一起算在這一格。

## Story

身為把金鑰填在設定精靈裡的使用者，
我想要 Vido 在檔名看不懂時，真的用我的金鑰問 AI 認出是哪部片，而且花了多少錢有記下來，
這樣格式很亂的檔名也能進片庫，我也知道這件事花了我多少錢。

## Acceptance Criteria

1. **網頁上存的金鑰有效、不用重開：**
   - 開機時沒有任何金鑰：檔名解析照舊退回 regex 或 `needs_ai`，行為和訊息都不變。
   - 在設定頁存入 Gemini 或 Claude 金鑰後，**下一次**解析就用 AI。不用重開。
   - 刪掉金鑰後，下一次解析就退回沒有 AI 的行為。
   - `IsConfigured()`、`GetProviderName()`、`Ping()` 都改成每次呼叫時判斷。
2. **選哪一家：**
   - `AI_PROVIDER` 指定的那家有金鑰 → 用它。
   - 指定的那家沒有、另一家有 → 用另一家。
   - 兩家都沒有 → 未設定。
   - 只設 `CLAUDE_API_KEY` 環境變數、`AI_PROVIDER` 沒設時，服務**可以正常開機**而且用 Claude 解析。要有測試證明以前會失敗、現在不會。
   - 各家用的 model 照舊：`GetClaudeModel()`／`GetGeminiModel()` 覆寫，沒有就用 provider 的預設。
3. **啟動接線不再凍結：** `main.go` 一定建立 `AIService`，不再回傳 nil。
   - parser、`KeywordService`、`ai_cache` 清理一律接上，呼叫時才判斷有沒有設定。
   - health 的 AI 項目：沒有金鑰時，顯示跟現在一樣（開工先記下現在的輸出，改完逐字比對）；有金鑰時顯示已設定。
   - `KeywordService` 在未設定時，要跟現在「沒接上」一樣安靜跳過，不能噴錯或多打 log。
4. **記帳：** 依上面裁定 3、4 寫進 `ai_side_spend`。
   - 快取命中：不打 provider，也不寫紀錄。
   - 寫入失敗：解析結果不受影響。
   - ctx 原本就帶著字幕 run 的 Budget 時：那個 Budget 的金額**不會**增加（要有測試）。
5. **API：** `/subtitles/spend` 回傳 `parse_calls`、`parse_usd`，月份區間跟其他欄位相同。
   - 沒有紀錄時是 `0`、`0`。（「—」怎麼顯示由 -b 決定；後端回 0 次就代表沒有。）
   - 更新 swagger 註解或文件，照這個 handler 現有的做法。
   - 前端型別在 -b 改。本單先確認多了欄位不會讓前端壞掉：跑 `subtitleSpendService.spec.ts`。
6. **文件和註解改成新裁定：**
   - `ai_service.go:1-10` 檔頭改寫成：記帳不設上限，Alexyu 2026-10-21，並說明為什麼仍然不設上限。
   - `docs/deployment.md:139-148` 的「Deliberately NOT covered」改成：解析會記帳、不設上限，花費出現在「本月 AI 花費」。
   - 照 bilingual 規則，如果有 zh-TW 版的 deployment 文件也要改。
   - README:127 的 `AI_PROVIDER` 那列改成：不用再填，指定的那家沒有金鑰時會自動用另一家；填了就是「兩家都有時優先用哪家」。
   - `setup_service.go:199-201` 過時的「Gemini is env-only」註解改掉。
   - `ApiKeysStep.spec.tsx:186-188` 斷言旁邊的理由註解改掉。斷言本身照舊：精靈的 skip 警告不提檔名解析，因為那句話的重點是翻譯。
   - sprint-status 的 `backlog-parse-path-ai-metering` 標成被本單取代。
7. **測試（先寫、先看它紅）：** 用真的 secrets store、`KeyResolver`、假的 provider factory 跑整合測試。照 `key_resolution_integration_test.go`、`asr_hot_reload_test.go`、`setup_gemini_integration_test.go` 的寫法。至少要有：
   - 沒有金鑰 → `needs_ai`。
   - 存入 Gemini → 用 Gemini 解析，並寫一列 `ai_side_spend`。
   - 再存入 Claude，而且 `AI_PROVIDER=claude` → 換 Claude。
   - 刪掉金鑰 → 退回。
   - 快取命中不寫紀錄。
   - 帶著外層 Budget 時，外層金額不變。
   - 只有 Claude 環境變數也能開機。
   - spend API 的月份彙總，包含上個月的紀錄不算進這個月。
8. **全綠：** `go test ./...`（apps/api）、`~/go/bin/staticcheck ./...`、`pnpm nx run web:test`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`。

## Tasks / Subtasks

- [x] T1：先寫 AC #7 的測試，看它紅
- [x] T2：parse provider holder，每次呼叫解析金鑰，沿用 Governor（AC #1、#2）
- [x] T3：`AIService` 一律建立，`main.go` 接線改成呼叫時判斷（AC #3）
- [x] T4：migration 047 `ai_side_spend`、repository、AIService 每個呼叫點包 Budget 並寫紀錄（AC #4）
- [x] T5：spend API 加 `parse_calls`／`parse_usd`（AC #5）
- [x] T6：文件和註解（AC #6），全綠（AC #8），開 PR，合併

## Dev Notes

- **原生 provider 已經會計帳：** `ClaudeProvider.Parse`（`claude.go:382`）和 `GeminiProvider.Parse`（`gemini.go:215`）都走 `governed()`，ctx 上有 Budget 時才會 `RecordLLM`（`claude.go:343-345`、`gemini.go:327-339`）。所以本單只要在呼叫前掛上 Budget，不用改 provider。
- **為什麼仍然不設上限：** sub-5-1 的擔心還是成立，掃描到一半被擋住，片庫會只解析一半。快取（`ai_cache`，30 天 TTL）讓同一個檔名只問一次。花多少錢用「看得到」來處理，不用「擋下來」。
- **掃描沒有傳 ctx：** `scanner_service.go:278-283` → `parser_service.go:67` 用的是 `context.Background()`。因為 Budget 是 `AIService` 自己在呼叫點掛上的，**本單不需要**把 ctx 從掃描一路傳下來。如果順手改成 `ParseFilenameWithContext` 會碰到很多呼叫端，不要做。
- **Gemini 解析用的是原生 `GeminiProvider`**，不是翻譯用的 `OpenAICompatProvider`。不要換掉。
- **花費估算：** 用 Claude 預設的 `claude-sonnet-5`，一次解析大約 US$0.005；Gemini 預設的 `gemini-2.5-flash-lite` 不到 US$0.001。這只是數量級，不用寫進程式。
- 正式機部署後，AI 解析第一次會在正式機打開，花的是 Alexyu 自己的金鑰。之前被標成「比對失敗」的片**不會**自動重新解析。重新掃描時，只有還沒進片庫的檔案才會走解析，這是現有行為，不在本單範圍。

### 不要做的事

- 不加上限、不擋掃描。
- 不改 `subtitle_runs`。
- 不改前端畫面（-b 做）。
- 不改 Gemini hint「免費方案每分鐘 5 次」那句過時的文字。它屬於設定頁 C 流程的設計稿，留在原本的 backlog 註解，不在本單範圍。

### References

- [Source: apps/api/internal/services/ai_service.go:1-10、:65-110、:127/178/276（快取）、:375-400]
- [Source: apps/api/cmd/api/main.go:234-241（resolver）、:320-337、:345-349、:388-396、:418-425、:576-577]
- [Source: apps/api/internal/services/claude_provider_holder.go:77-158；translation_dispatcher.go:87-132]
- [Source: apps/api/internal/services/key_resolver.go:93、:172-212；key_gemini.go]
- [Source: apps/api/internal/ai/factory.go:35-60；budget.go:214-216；governor.go:16-37、:68-79]
- [Source: apps/api/internal/config/api_keys.go:40-70；config.go:231]
- [Source: apps/api/internal/services/subtitle_spend_service.go:31-62、:129、:162-221]
- [Source: apps/api/internal/database/migrations/030_create_subtitle_runs_table.go:50-51]
- [Source: docs/deployment.md:139-148；README.md:127]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-21 | 建單（Bob）。Alexyu 選 A：打開，並記進花費紀錄。拆成 -a 後端、-b 設計稿＋前端。Winston：每次呼叫決定用哪一家、`ai_side_spend` 逐筆記、只記帳不設上限。 |
| 2026-10-21 | 實作完成（Amelia），PR #890。 |

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，BMAD dev）

### Debug Log References

- 開工前實測（main 19ecbf5e 編出的 binary）：
  - 沒有金鑰時，`/api/v1/health/services` 的 ai 列是 `status: degraded`、`error_count: 1`、`message: "AI provider not configured"`。
  - 只設 `CLAUDE_API_KEY`、沒設 `AI_PROVIDER` 時，**exit 1**，log 是 `Failed to initialize AI service error="AI_NOT_CONFIGURED: ... GEMINI_API_KEY not configured"`。story 的推論因此實際確認了。
- 改完後實測：
  - 沒有金鑰時，ai 列逐字相同；`/subtitles/spend` 回 `parse_calls: 0, parse_usd: 0`。
  - 只有 Claude 環境變數時，正常開機，log 是 `AI parsing available provider=claude`，health 的 ai 列為 healthy。
- TDD 紅→綠：
  - `TestAIParseKeys_*` 共 17 個：先加只夠編譯的空殼，跑起來 15 紅、2 綠（2 綠的是 JSON 欄位名，以及沒有紀錄時回 0／0）。實作後 17 綠。
  - `TestOrchestrator_Search_UnconfiguredKeywordGenerator…`：1 紅 → 綠。
  - migration 047 的 4 個、repository 的 3 個：實作前編譯失敗 → 實作後綠。
  - 自我審查後再加 `ParseProviderHolder` 單元測試 4 個（含 7 個子案例）。

### Completion Notes List

- 新增 `ParseProviderHolder`（AC #1、#2）：
  - 每次呼叫都透過 `KeyResolver` 選 provider。指紋是 `provider|key|model`。
  - 沿用 main.go 的 `aiGovernor`；兩把金鑰都刪掉時清掉快取的 client。
  - `AI_PROVIDER` 填了不認得的值時，記一行 warn，然後當成 gemini。以前有金鑰時會開不了機。
- `AIService` 一律建立（AC #3）：
  - `IsConfigured`／`GetProviderName`／`Ping` 改成每次呼叫時判斷。沒有金鑰時，`Ping` 的訊息跟以前 nil Pingable 的逐字相同。
  - `NewAIServiceWithProvider` 和零值 `AIService` 的舊行為不變，既有測試不用改。
- 記帳（AC #4）：三個呼叫點都改走 `meteredParse`。
  - 用自己的 `ai.NewBudget(0)` 蓋掉 ctx 上原本的 Budget。
  - `Spent > 0` 就寫 `ai_side_spend`。寫入用 `context.WithoutCancel`，逾時的呼叫也照記。
  - 寫入失敗只記 warn。
- migration 047：`created_at` 沒有 default，一律由 app 綁時間，用 sqlite-utc 的格式，避免 `CURRENT_TIMESTAMP` 的短格式跟月份區間比較時出錯。
- spend API（AC #5）：`SetSideSpend` 接 repository。`parse_usd` 跟其他金額一樣四捨五入到美分。`by_model`／`by_batch` 不變。
- `KeywordService` 在未設定時跳過（AC #3）：`metadata.Orchestrator` 會檢查 generator 有沒有 `IsConfigured()`，回 false 就整段跳過，不呼叫、不寫 log、不設 KeywordError。
- 文件（AC #6）：
  - `ai_service.go` 檔頭、`docs/deployment.md`（沒有 zh-TW 版）、README `AI_PROVIDER` 列。
  - 順手改了 unraid 安裝指南（中英）、`.env.example`、`unraid-template/vido.xml` 裡 `AI_PROVIDER` 的說明。
  - `setup_service.go`、`gemini.go`、`ApiKeysStep.spec.tsx` 的註解（斷言不變）。
- 全綠（AC #8）：
  - `go test ./...` 43 個 package 全過；`staticcheck`、`go vet` 乾淨。
  - `web:test` 4869 個全過；`lint:all` 是 0 errors（170 個 warning 原本就有）；`format:check` 通過。
- 不在範圍：之前「比對失敗」的片不會自動重新解析（Dev Notes 已寫明）。

### File List

- apps/api/internal/services/parse_provider_holder.go（新）
- apps/api/internal/services/parse_provider_holder_test.go（新）
- apps/api/internal/services/ai_parse_key_resolution_integration_test.go（新）
- apps/api/internal/services/ai_service.go
- apps/api/internal/services/subtitle_spend_service.go
- apps/api/internal/services/setup_service.go
- apps/api/internal/models/ai_side_spend.go（新）
- apps/api/internal/repository/ai_side_spend_repository.go（新）
- apps/api/internal/repository/ai_side_spend_repository_test.go（新）
- apps/api/internal/database/migrations/047_create_ai_side_spend_table.go（新）
- apps/api/internal/database/migrations/047_create_ai_side_spend_table_test.go（新）
- apps/api/internal/metadata/orchestrator.go
- apps/api/internal/metadata/orchestrator_test.go
- apps/api/internal/handlers/subtitle_spend_handler.go
- apps/api/internal/ai/gemini.go
- apps/api/cmd/api/main.go
- apps/web/src/components/setup/ApiKeysStep.spec.tsx
- docs/deployment.md
- docs/unraid-installation-guide.md
- docs/unraid-installation-guide.zh-TW.md
- README.md
- .env.example
- unraid-template/vido.xml
- _bmad-output/implementation-artifacts/sprint-status.yaml
