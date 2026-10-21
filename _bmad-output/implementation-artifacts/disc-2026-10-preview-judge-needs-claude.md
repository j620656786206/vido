# Disc：只填 Gemini 金鑰時，「試跑 20 句」一定失敗，畫面還說「尚未設定翻譯服務金鑰」

Status: done

**Source:** 2026-10-14 gemini-translation-dispatch CR（Fable M2）填的 Rule 24 lane ③。⚖️ 2026-10-21 party mode（founder-pm、founder-growth、BMAD）盤點「新使用者第一次裝就會撞到的問題」，Alexyu 選 A「首次安裝修正」，本單是第 2 張（原列表的第 3 個；第 2 個「.ts／.mov 掃不到」Alexyu 說之後再做）。

**為什麼現在要修：** 設定精靈從 #836 起明說「Claude 或 Gemini 擇一」，Gemini 又有免費額度，外部新使用者很可能只填 Gemini。這些人在生成字幕的同意清單裡選模型時，按「試跑 20 句」一定失敗，錯誤還說他沒設金鑰，但其實他設好了。這是新使用者第一次想確認「翻得好不好」的地方。

**原因（2026-10-21 查證）：**

- 試跑分兩步：先用使用者選的模型翻 20 句，再請「裁判」模型打分數。
- 裁判寫死是 `claude-sonnet-5`：`preview/service.go:127` `judgeModel: eval.DefaultJudgeModel`。
- `realRun` 在 :283 用 `s.completers.GetFor(ctx, s.judgeModel)` 取裁判。只有 Gemini 金鑰時，`ClaudeProviderHolder.GetFor` 回 `ErrAINotConfigured`（`claude_provider_holder.go:118-124`）。
- handler 把這個錯誤轉成 409 `AI_NOT_CONFIGURED`，畫面顯示「尚未設定翻譯服務金鑰」（`model_settings_handler.go:146-148`、`useModelPreview.ts:43-44`）。
- 所以就算第一步翻譯成功，整次試跑還是失敗。

## ⚖️ 裁定（Winston＋Sally，2026-10-21）

**裁判改成「有 Claude 用 Claude，沒有就用 Gemini」。** 有 Claude 金鑰的安裝，行為完全不變，裁判仍是 `claude-sonnet-5`，分數可以和模型等級表比。只有 Gemini 的安裝，裁判改用 `gemini-3.7-flash`，這是 catalog 裡最新的 balanced Gemini（`ai/catalog.go:73-80`）。

不選的另外兩個做法：

- **只有 Gemini 時不顯示試跑按鈕**：不選。只填 Gemini 的人就永遠不能先試再花錢，而「先報價、先試、再花錢」是 Vido 對外講的核心承諾。
- **只改文案成「試跑的評審需要 Claude 金鑰」**：不選。等於叫只想用免費 Gemini 的人另外去辦一組 Claude 金鑰，才能試跑 20 句。

**畫面不改：** 這張單不加新文字，所以不用先改設計稿。J10 `ctRsy` 的四種狀態都照舊。

本地評分已經存了 `LocalGrade.JudgeModel`（`service.go:69`），只是畫面沒顯示。換成 Gemini 當裁判時，分數跟表上 Claude 裁判的分數不能直接比。要不要在畫面標出裁判，另外立案給 Sally，見 AC #7。

## Story

身為只填了 Gemini 金鑰的新使用者，
我想要按「試跑 20 句」就真的試跑出結果，
這樣我不用另外辦 Claude 金鑰，也能在花錢生成整部之前，先看看翻得怎麼樣。

## Acceptance Criteria

1. **裁判每次試跑都重新決定（金鑰熱載）：** 每次試跑開始時，依當下的金鑰決定裁判。
   - 有 Claude 金鑰 → `claude-sonnet-5`（`eval.DefaultJudgeModel`，跟現在一樣）。
   - 沒有 Claude、有 Gemini → `gemini-3.7-flash`。在 `eval` 或 `preview` 定義一個具名常數，例如 `GeminiFallbackJudgeModel`，並寫註解說明為什麼選它。
   - 兩個都沒有 → 跟現在一樣回 `ErrAINotConfigured`，回 409。
   - 「有沒有金鑰」要用跟翻譯一樣的來源判斷：dispatcher／KeyResolver，也就是網頁上存的金鑰和環境變數都算。不能只看環境變數，也不能在啟動時決定一次。試跑途中如果改了金鑰，下一次試跑要用新的金鑰。
2. **報告記對裁判：** `eval.Run` 收到的 `JudgeModel`、存下來的 `LocalGrade.JudgeModel`，都要是實際用到的裁判模型 id。
3. **報價跟著裁判走：** `EstimateUSD`／`BudgetUSD` 要用實際會用到的裁判價格（`ai.PricingFor(<解析出的裁判>)`）。只有 Gemini 時，按鈕上的報價反映 Gemini 裁判的價格。預算下限 0.05、上限 0.20 的規則照舊。
   - 報價和試跑必須用同一個「決定裁判」的函式，不可以各寫一份。
   - 如果 `EstimateUSD` 沒有 ctx 可用（目前的簽章是 `EstimateUSD(model string)`），可以加 ctx 參數，或另外加一個帶 ctx 的版本；呼叫端跟著改。
4. **偏誤說明不寫死 Claude：** `eval.JudgeNote` 目前寫死「裁判為 Claude（claude-sonnet-5），評 Claude 模型時有自評偏誤…」。改成依裁判產生文字，例如 `JudgeNoteFor(judgeModel)`：
   - Claude 裁判時，文字**逐字不變**。`model_ratings.json`、`cmd/grade` 的既有報告和 CI 評分 feed 都靠這句，不能有任何差異。
   - Gemini 裁判時，寫「裁判為 Gemini（gemini-3.7-flash），評 Gemini 模型時有自評偏誤；分數跟模型等級表（Claude 裁判）不能直接比。」
   - `cmd/grade`（`--judge-model` 預設 Claude）行為不變。
5. **測試（先寫、先看它紅）：**
   - 新增一條 `realRun` 層級的測試：裁判派送沒有被 `s.run` seam 跳過。現有 `service_test.go` 的測試都走 seam，所以要用假的 completerSource，讓 `GetFor(claude-sonnet-5)` 回 `ErrAINotConfigured`、`GetFor(gemini-*)` 回假 completer。
   - 情境 1：只有 Gemini → 試跑成功，裁判是 `gemini-3.7-flash`。
   - 情境 2：有 Claude → 裁判是 `claude-sonnet-5`。
   - 情境 3：兩個都沒有 → `ErrAINotConfigured`。
   - 情境 4：先只有 Gemini，再存入 Claude → 下一次試跑換成 Claude 裁判（熱載）。
   - `TestEstimateAndBudget` 補一個 Gemini 裁判的報價情境。
   - `JudgeNote` 補一條：Claude 版本逐字不變。
   - handler `TestPreviewModel_StatusMapping` 不用改，409 對應照舊；跑一次確認是綠的。
   - 前端不用改。跑 `ModelPicker.spec.tsx`、`useModelPreview.spec.ts` 確認還是綠的。
6. **全綠：** `go test ./...`（apps/api）、`~/go/bin/staticcheck ./...`、`pnpm nx run web:test`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`。
7. **附帶立案：** 在 sprint-status 新增一條 backlog，例如 `disc-2026-10-preview-grade-hides-judge`（P3，Sally 裁定），內容是「試跑評分行沒標裁判；Gemini 當裁判時，分數跟表上不能直接比」。本單不做 UI。
8. **錯誤文字：** 兩個金鑰都沒有時，hint 現在寫「先到 設定 → API 金鑰 存一組 Claude 金鑰」（`model_settings_handler.go:148`、`useModelPreview.ts:44`）。這已經另外有單 `backlog-claude-only-error-copy`，本單不改，但要在那張單的註解補一句：「試跑的 hint 也在範圍內」。

## Tasks / Subtasks

- [x] T1：先寫 AC #5 的測試，看它紅
- [x] T2：做一個決定裁判的函式，試跑和報價共用（AC #1、#3）
- [x] T3：`realRun` 和 `eval.Run` 傳實際的裁判（AC #2）
- [x] T4：`JudgeNoteFor`（AC #4）
- [x] T5：附帶立案、補註解（AC #7、#8）
- [x] T6：全綠（AC #6），開 PR，合併

## Dev Notes

- 派送邏輯在 `services/translation_dispatcher.go:183-199`：`GetFor` 依 `ai.ProviderOf(model)` 分流到 Claude 或 Gemini 的 holder，兩個 holder 每次呼叫都會讀 resolver。所以最簡單的「決定裁判」做法是：先試 `GetFor(ctx, DefaultJudgeModel)`，遇到 `ErrAINotConfigured`（用 `errors.Is`）就改試 `GetFor(ctx, GeminiFallbackJudgeModel)`。報價那邊如果不想真的建立 provider，就用 dispatcher 現有的「這家有沒有設定」方法（例如 `IsConfigured` 這類）判斷。不要另外去讀 `config` 的環境變數。
- 只有 Gemini 時，翻譯和裁判都會用使用者自己的 Gemini 額度。Gemini 的限速在 dispatcher 裡（`DefaultGeminiMinInterval`），裁判每批 20 句（`eval.JudgeBatchSize`），試跑只有一批，不會撞到限速。
- 自評偏誤：試跑的模型本身也是 Gemini 時，就是「Gemini 評 Gemini」。這跟現在「Claude 評 Claude」是同一類問題，用 AC #4 的說明揭露就好，不要另外擋。
- `eval.Run` 在 `JudgeModel` 為空時會補 `DefaultJudgeModel`（`run.go:164-166`），所以一定要明確傳入解析出來的裁判。
- 實際用 Gemini 當裁判打一次分數，要用真的 Gemini 金鑰和真的錢。本機沒有金鑰，所以本單靠假的 completer 測流程。合併、部署之後，Alexyu 可以在正式機只留 Gemini 金鑰（或用測試機）按一次試跑確認。這需要 Alexyu 同意才做，dev 不要自己在正式機做。

### 不要做的事

- 不改 J10 畫面、不加新文字（AC #7 另外立案）。
- 不改 `cmd/grade` 和 `model_ratings.json` 的裁判（等級表維持 Claude 裁判）。
- 不碰 `backlog-claude-only-error-copy` 的其他錯誤文字。

### 設計稿 / Design ref

- J10 `ctRsy`（試跑按鈕四種狀態），F16-D-v2 `gmOt6`、F16-M-v2 `x45wBO`、F19-D-v2 `KThbY`（`ModelPicker.tsx:1`）。本單不改畫面。

### References

- [Source: apps/api/internal/preview/service.go:120-177（NewService、EstimateUSD、BudgetUSD）、:278-330（realRun）]
- [Source: apps/api/internal/eval/judge.go:17-26（DefaultJudgeModel、JudgeNote）、run.go:161-166]
- [Source: apps/api/internal/services/translation_dispatcher.go:23、:183-199；claude_provider_holder.go:118-124]
- [Source: apps/api/internal/handlers/model_settings_handler.go:60、:130-155]
- [Source: apps/api/internal/ai/catalog.go:73-80、budget.go:61-68]
- [Source: apps/web/src/hooks/useModelPreview.ts:43-44；components/subtitle/consent/ModelPicker.tsx:40-44、:269-287]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-21 | 建單（Bob）。裁定：裁判改成「有 Claude 用 Claude，沒有就用 gemini-3.7-flash」，畫面不改；標出裁判另外立案給 Sally。 |
| 2026-10-21 | 完成（Amelia）。裁判每次試跑依當下金鑰決定；報價、預算、報告共用同一個決定函式；JudgeNoteFor 的 Claude 版逐字不變。 |

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，BMAD dev）

### Debug Log References

- 紅燈（先寫測試、加只有簽章的 stub 保留舊行為）：`TestPreview_GeminiOnlyInstall_JudgedByGemini` 失敗訊息就是使用者看到的那個——`AI_NOT_CONFIGURED: ... no Claude API key configured`。另外紅：`TestPreview_ClaudeKeySavedLater_NextPreviewJudgedByClaude`、`TestEstimateAndBudget_GeminiJudgePricesTheButton`（兩種報價一樣）、`TestJudgeNoteFor`、`TestRun_JudgeNoteFollowsTheJudge`、`TestDispatcher_ProviderConfigured_PerProviderAndHotReloaded`，共 6 條。`TestPreview_ClaudeKeyPresent_JudgeStaysClaude`、`TestPreview_NoKeys_NotConfigured` 在舊程式就綠（防退化用）。
- 綠燈：上面 8 條全綠；`go test ./...`（apps/api）全綠、`staticcheck ./...`、`go vet ./...` 無輸出；`pnpm nx run web:test` 298/298 檔綠；`ModelPicker.spec.tsx`＋`useModelPreview.spec.ts` 19/19。

### Completion Notes List

- **決定裁判只有一個函式**：`preview.Service.judgeFor(ctx)`。依序問 dispatcher「Claude 有沒有金鑰」「Gemini 有沒有金鑰」（新方法 `TranslationDispatcher.ProviderConfigured`，走跟翻譯一樣的 KeyResolver，網頁存的和環境變數都算，每次呼叫重讀），回 `claude-sonnet-5`／`gemini-3.7-flash`／`ErrAINotConfigured`。沒有用「先 GetFor 失敗再退」：報價不該為了算錢去建 client，而兩邊共用同一個函式是 AC #3 的要求。
- **試跑**：`realRun` 一開始決定一次裁判，同一個值拿去取 completer、算預算、傳給 `eval.Run`，三者不會不一致。`LocalGrade.JudgeModel` 因此存的是實際裁判。
- **報價**：`EstimateUSD`／`BudgetUSD` 加 ctx 參數（handler 介面與 stub 跟著改），用 `quoteJudge` = `judgeFor`，兩個金鑰都沒有時沿用 Claude 價（此時模型清單本來就是空的，不會顯示按鈕）。順手把報價裡讀在地化等級的 `context.Background()` 換成真的 ctx。
- **偏誤說明**：`eval.JudgeNoteFor(judgeModel)`；Gemini 裁判出 Gemini 版文字，其他（Claude、空字串）回原本的 `JudgeNote` 常數，逐字不變。`eval.Run` 改用它。`cmd/grade` 預設 `--judge-model` 仍是 Claude，`--no-judge` 時 JudgeModel 為空也回原文字，行為不變。
- `eval.GeminiFallbackJudgeModel = "gemini-3.7-flash"` 放在 eval（跟 `DefaultJudgeModel` 同處），註解寫明為什麼選它。
- `docs/development.md` 的評分段落補一句：試跑在只有 Gemini 時用 Gemini 當裁判、分數不能跟等級表直接比；`cmd/grade` 仍固定 Claude 裁判。
- 沒驗到的事：真的用 Gemini 當裁判打一次分（要真金鑰、真的錢）。合併部署後，經 Alexyu 同意，在只留 Gemini 金鑰的機器按一次試跑確認。風險點：Gemini 是會「思考」的模型，裁判回覆上限 4096 token；思考吃掉太多時會回 `ErrAIResponseTruncated`。既有的 `eval.Run` 會把裁判失敗的句子記成「未評、0 分」（`Report.Unjudged`），但 `LocalGrade` 沒帶這個數字，畫面會看到偏高的「不能用」比例。這是原本就有的行為（Claude 裁判也一樣），本單不改；寫在 PR 的「沒做的事」。

### File List

- apps/api/internal/eval/judge.go
- apps/api/internal/eval/judge_test.go
- apps/api/internal/eval/run.go
- apps/api/internal/eval/run_test.go
- apps/api/internal/preview/service.go
- apps/api/internal/preview/service_test.go
- apps/api/internal/services/translation_dispatcher.go
- apps/api/internal/services/translation_dispatcher_test.go
- apps/api/internal/handlers/model_settings_handler.go
- apps/api/internal/handlers/model_settings_handler_test.go
- docs/development.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/disc-2026-10-preview-judge-needs-claude.md
