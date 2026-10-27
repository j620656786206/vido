# disc-2026-10-gemini-parse-model-retired-again：檔名解析改用 Gemini 現役模型，沒劇名的檔名又能加進媒體庫

Status: done

<!-- SM Bob create-story 2026-10-27；行號為 main `5cbd5e8a` 之後。Rule 24 lane ③，Alexyu 10/27 要求開票並指定做這張。 -->

**Source:** 正式機 2026-10-27 11:20 掃描 log：檔名 AI 解析用的 `gemini-2.5-flash-lite` 回 404「This model models/gemini-2.5-flash-lite is no longer available to new users. Please update your code to use models/gemini-3.5-flash-lite」→ `AI parsing failed` → `SCANNER_UNMATCHED: cannot determine episode number, file not imported`。實例：`/media/tv/The Lord of the Rings - The Rings of Power/Season 1/S01E01.mkv`～`S01E04.mkv`（檔名沒有劇名，regex 失敗交給 AI）、`Renegade Immortal/Special/[GM-Team]…[Movie][2025]….mp4`。

## 查證（2026-10-27，ai.google.dev）

- `/gemini-api/docs/deprecations`：`gemini-2.5-flash`、`gemini-2.5-flash-lite` 只給「過去用過的使用者」繼續用，沒有停用日；新專案建議 `gemini-3.5-flash-lite` 或 `gemini-3.8-flash`。`gemini-3.5-flash-lite`（2026-07-21 上市）沒有停用日、沒有限制。`gemini-3.1-flash-lite` 2027-05-07 停用。
- `/gemini-api/docs/pricing`：`gemini-3.5-flash-lite` 有免費方案；付費 $0.30／$2.50 每 100 萬 token（`budget.go` 的價格表一致）。`gemini-2.5-flash-lite` 是 $0.10／$0.40。
- 這跟 2026-08 的 `bugfix-gemini-default-model-retired`（2.0-flash 停用）是同一類事。

## 決定（Dev 依查證採用，理由寫在這裡）

- **預設換成 `gemini-3.5-flash-lite`**：Google 對新專案的建議、沒有停用日、有免費方案（FB 招募主打「用自己的 Gemini 免費額度」），字幕翻譯的 Gemini 預設（`DefaultGeminiTranslationModel`）本來就是它。付費價貴 3～6 倍，但檔名解析一次只有幾百 token（約 US$0.0003／檔），而且只有 regex 解不出來的檔案才會叫 AI。
- **2.5 兩個從可選清單拿掉、價格保留**：新金鑰選到會在使用者同意付錢之後才 404；價格表保留，舊的執行紀錄與用 `GEMINI_MODEL` 釘住 2.5 的部署照樣算得出錢（跟 2.0-flash 同一套 `retired` 做法）。
- **畫面上的提示不做**：要設計稿，另立 `disc-2026-10-ai-model-unavailable-surface`（P3）。這張只把 log 講清楚。

## Acceptance Criteria

1. **預設模型**：沒設 `GEMINI_MODEL` 時，檔名解析用 `gemini-3.5-flash-lite`。設了就照設的（行為不變）。
2. **模型不存在講清楚**：檔名解析打 Gemini 回 404 → 錯誤是 `ErrAIModelNotFound`，訊息寫出模型名稱與「設定 `GEMINI_MODEL` 換成現役模型」（跟 Claude 的 404 一樣），不再只有 `status 404`。不重試（404 不是暫時性的）。
3. **可選清單**：字幕翻譯的模型選單不再出現 `gemini-2.5-flash`、`gemini-2.5-flash-lite`；`IsSelectableModel` 對它們回 false。兩者的價格列保留。
4. **不動**：字幕翻譯的 Gemini 預設、Claude 的任何設定、`GEMINI_MODEL` 覆寫、價格數字。
5. **測試**：預設值釘住新模型；Parse 收到 404 → `ErrAIModelNotFound`、訊息含模型名稱與 `GEMINI_MODEL`、只打一次；2.5 不在 `Catalog()`、`IsSelectableModel` false、價格查得到。
6. **NAS 實測**：正式資料庫備份＋臨時容器（用正式機存的 Gemini 金鑰，不設 `GEMINI_MODEL`）掃描一次，log 不再有 Gemini 404，檔名解析真的回結果。⚠️ 實測後修正：原本寫「《魔戒》第 1 季那幾集都進得了資料庫」——實際只進了一半，另一半是 Gemini 對沒有劇名的檔名回空的 `title`（見 Completion Notes），那是解析器的另一個問題，另立 `disc-2026-10-bare-episode-filename-uses-folder-title`。

## 不要做的事

- 不改 regex 解析器（讓 `S01E01.mkv` 從資料夾拿劇名是另一件事）。
- 不加開機時的模型檢查（會吃使用者的額度）。
- 不改畫面。

## Tasks

- [x] Task 1 — `DefaultGeminiModel` → `gemini-3.5-flash-lite`，註解更新（AC 1）
- [x] Task 2 — `GeminiProvider.Parse` 404 → `ErrAIModelNotFound`＋提示（AC 2）
- [x] Task 3 — `catalog.go` 2.5 兩個標 `retired`，註解說明「新專案不能用」（AC 3）
- [x] Task 4 — 測試、vet、staticcheck；NAS 實測（AC 5、6）

## Dev Notes

- 預設：`apps/api/internal/ai/gemini.go:26`；被 `parse_provider_holder.go:140` 拿來當沒設 `GEMINI_MODEL` 時的模型。
- Parse 的錯誤分支：`gemini.go:295-303`（只有 429 特別處理）；`Ping` 的 404 分支（`:191`）是寫法參考；Claude 的 404 訊息在 `claude.go:291`。
- 可選清單：`catalog.go:62-80` 的 `modelMetadata`，`retired` 讓模型不進 `Catalog()`、`IsSelectableModel` 回 false，價格列在 `budget.go` 不動。
- 既有測試：`gemini_model_test.go:18`（釘住預設值）要跟著改。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `DefaultGeminiModel` → `gemini-3.5-flash-lite`（註解寫明 2.0 停用、2.5 不給新用戶、3.5 的查證日期與價格）。
- `GeminiProvider.Parse` 的 404 → `ErrAIProviderError`＋`ErrAIModelNotFound`，訊息含模型名稱與「set GEMINI_MODEL to a current model」，不重試（`Ping` 原本就有，`Parse` 漏了）。
- `catalog.go`：`gemini-2.5-flash`、`gemini-2.5-flash-lite` 標 `retired`（不進 `Catalog()`、`IsSelectableModel` false），價格列不動。
- 既有測試跟著新預設改：`gemini_model_test`（釘住 3.5、明寫不是 2.5）、`gemini_test` 計費（$0.30＋$5.00）、`ai_parse_key_resolution_integration_test`（$0.0008）、`cmd/grade` 的「Gemini 被拒」改用仍可選的 3.5（2.5 現在會先因「不可選」被拒，測不到原本的理由）。
- 新測試：`TestGeminiProvider_Parse_ModelNotFoundSaysWhichAndHow`、`TestCatalog_DoesNotOfferModelsClosedToNewUsers`。反向驗證：關掉 404 分支 → 新測試紅。
- **NAS 實測（2026-10-27 14:07，正式資料庫備份＋臨時容器；只留 TMDb 與 Gemini 金鑰，媒體唯讀）**：log `parse provider (re)built … provider=gemini model=gemini-3.5-flash-lite key_source=secret`，**不再有 404**。`Renegade Immortal/Special/[GM-Team]…[Movie][2025]….mp4` 解析成功（`title=仙逆剧场版 神临之战 media_type=movie`），但它放在影集資料夾、沒有集數，掃描照舊不收（另一回事）。《魔戒：力量之戒》第 1 季 `S01E01～S01E08.mkv`：E01、E02、E04、E06 新進資料庫；E03、E05、E07、E08 是 Gemini 回 `AI_INVALID_RESPONSE: missing required field: title`——檔名裡本來就沒有劇名，AI 有時給 `title=S01E01`、有時給空的。根本做法是 regex 遇到純 `SxxExx` 檔名直接用資料夾名當劇名、不交給 AI（不花錢、不看運氣），另立 `disc-2026-10-bare-episode-filename-uses-folder-title`（P2）。測完容器與資料夾都刪了。
- 範圍外：`gemini-3.7-flash` 在 Google 那邊已經自動轉到 `gemini-3.8-flash`（價格一樣，2027-01-01 起兩者都漲價），可選清單的名字沒改，不影響使用。
- `go test ./internal/... ./cmd/...` 全過；`go vet`、`staticcheck` 乾淨。

### File List

- `apps/api/internal/ai/gemini.go`、`gemini_test.go`、`gemini_model_test.go`
- `apps/api/internal/ai/catalog.go`、`catalog_test.go`
- `apps/api/internal/services/ai_parse_key_resolution_integration_test.go`
- `apps/api/cmd/grade/main_test.go`
- `.env.example`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
