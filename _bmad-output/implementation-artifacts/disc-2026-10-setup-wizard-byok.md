# Disc：首次設定精靈講清楚「AI 費用付給誰」，沒有 Claude 金鑰的人也能填 Gemini 試

Status: review

**Source:** 2026-10-19 party mode（founder-pm／founder-growth／John／Mary／Sally／Bob）。⚖️ **Alexyu 2026-10-19 選 B：先改精靈，改完再在 FB 招募文底下公開邀請**（不選 A「先發文、精靈之後再說」）。
**為什麼：** v0.2.0（10-14）發版五天，查不到任何外人裝起來（GitHub 14 天 13 個不同訪客、`0.2.0` 映像 6 次下載、外人 issue 0）。FB 招募文 42 則留言裡：黃柏銘把「報價」讀成「付給測試者的工資」；Edward Fang「直接用 NAS 的 whisper，完全不燒 Claude API」；林琮耀問「是走 OpenAI API 嗎」。大部分 NAS 玩家**沒有 Claude 金鑰**，而精靈的金鑰那一步只給 Claude 一個欄位、只寫「用於字幕翻譯與 AI 檔名解析」，沒說錢付給誰、一集多少、沒金鑰怎麼辦。Gemini 的免費額度這條路，只有進「設定 → 金鑰設定」或設環境變數才看得到。
**停損關聯：** `release-0-2-tag-and-notes` 的停損是 2026-11-15 前要有一個外人裝起來、跑出一支字幕。本單是那之前唯一排定的產品改動；做完立刻發 FB 公開邀請（不是本單範圍）。

## 查到的事（2026-10-19，本 session 讀過的程式碼）

- **精靈的金鑰步驟**：`apps/web/src/components/setup/ApiKeysStep.tsx`（Design ref N4-D `D990CP`）。欄位只有 TMDb、Claude；Claude hint「用於字幕翻譯與 AI 檔名解析」；說明句「設定 API 金鑰以啟用進階功能。可以稍後在設定頁面新增。」；`nothingEntered` 時的跳過提示只講 Claude 功能受限。
- **「Gemini 只能用環境變數」這句註解已經過期**（`ApiKeysStep.tsx:38-41`、`models/settings.go:43-44`）：`services/key_gemini.go` 有 `KeyGemini`，`key_resolver.go:139-157` 會從加密儲存讀 Gemini、沒有才退回 env；`translation_dispatcher.go:79,94` 用 resolver 取 Gemini 金鑰；`key_settings_handler.go:79` 的 PUT 已收 `gemini`；設定頁 `ApiKeysForm.tsx:69-77` 早就有 Gemini 列，hint 寫「儲存後立即生效」。
- **精靈存金鑰的路**：前端 `SetupWizard.tsx:50`（`clearOnSkip: ['tmdbApiKey', 'claudeApiKey']`）、`:88`、`:139` → `services/setupService.ts:26`（`claudeApiKey` → JSON `claude_api_key`）→ Go `models.SetupConfig`（`settings.go:35-50`）→ `setup_service.go:81-90 setupKeyUpdates(tmdbKey, claudeKey)` → `:123` 寫入前先檢查 `keysWritable()`（沒 `ENCRYPTION_KEY` 就整個拒絕，`ErrKeysNotWritable`）。另有 `setup_service.go:386` 從 map 讀 `claude_api_key`（dev 確認這條是哪個路徑在用）。
- **設定頁 Gemini 文案的界線**（`ApiKeysForm.tsx:73-76` 註解）：永遠不說「免費」兩個字單獨出現，要說「用你自己的 Gemini 額度」；預設翻譯模型仍是 Claude。2026-10-14 裁定：只有 Gemini 金鑰時預設改走 Gemini（`.claude/memory` project_byok_moat_direction）。
- **README 的錢**（`README.md:93-105`，2026-10-15 量）：影集一集（約一小時、片內有英文字幕）US$0.15～0.3；電影一部 US$0.3～0.6；沒字幕要先聽聲音約 US$0.8。
- **設計稿**：`flow-n-setup-wizard/n4-d.png`（`D990CP`）。精靈沒有 N4 的手機稿（只有 `n3-m`）。

## ⚠️ 未查證（dev 開工第一件事，結果寫進 Completion Notes）

1. ~~只填 Gemini 時一集翻得完嗎？~~ **已查證（建單後同日）：** `gemini-translation-dispatch` AC #9 正式機驗證（2026-10-14 15:08，sprint-status）——See S01E02 整集 568 句、`gemini-3.5-flash-lite`、13 秒一批共 13.7 分鐘、59 次請求全部成功、帳記 US$0.08（目錄價）。免費方案 5 RPM；3.8-flash 每天 20 次會不夠，但只有 Gemini 金鑰時預設是 flash-lite。dev 只需在 AC #3 的 E2E 後於 NAS 再跑一次「精靈只填 Gemini → 直接產生字幕」確認走得通。
2. ~~只填 Gemini 時預設翻譯模型是不是 Gemini？~~ **已查證：** 同一張單的實作紀錄——「只有 Gemini 金鑰的安裝預設 `gemini-3.5-flash-lite`」（`gemini-translation-dispatch.md:65`）。
3. **AI 檔名解析**：`ai/factory.go:47` 讀的是 env 的 `GEMINI_API_KEY`。只在精靈填 Gemini 時，檔名解析會不會用到？不會的話 Gemini 欄位的 hint 就**不能**寫「也用於 AI 檔名解析」。

## Story

身為第一次裝 Vido、手上沒有 Claude 金鑰的 NAS 玩家，
我想要在設定精靈就知道「AI 費用是付給誰、一集大概多少、沒有 Claude 金鑰還能用什麼」，
這樣我不會以為要付錢給作者，也不會卡在一個我沒有的金鑰前面就放棄。

## 設計（⛔ 先做，Sally）

改 `ux-design.pen` 的 N4-D（`D990CP`），照 `CLAUDE.md` 的流程重產 `n4-d.png`、刷新 `pen-node-ids.json`，**只 commit 真的變了的那張**。Sally 2026-10-19 定稿（交給 Pencil inline agent 的提示詞就是這幾句；畫完以 `.pen` 為準）：

- **步驟說明 `nsDeY`：** 「Vido 不收錢。AI 字幕用你自己的金鑰，費用直接記在你的 Claude 或 Gemini 帳號，不經過 Vido；每次按下去之前，Vido 會先告訴你大約花多少。兩把都可以先空著，之後到設定頁再填。」
- **Claude 欄位 hint `w6DvG8`：** 「預設的翻譯模型。一集影集（約一小時、有英文字幕）大約 US$0.15～0.3。」下面一行連結「怎麼申請 Claude 金鑰 ↗」（`$accent-text`，新分頁開 Anthropic Console）。
- **新的 Gemini 欄位（複製 `SBCDC`）：** 標籤「Gemini 金鑰（選填）」、placeholder「輸入 Gemini API 金鑰...」、hint「沒有 Claude 金鑰？用你自己的 Gemini 免費額度也能翻；每分鐘 5 次，Vido 會自動放慢。」、連結「怎麼申請 Gemini 金鑰 ↗」（Google AI Studio）。**不寫**「也用於 AI 檔名解析」（未查證 #3）。
- **跳過提示 `lidJ4`：** 「Claude 和 Gemini 金鑰都沒填的話，就不能做 AI 字幕翻譯和檔名解析；電影和影集的元資料會使用內建的 TMDb 金鑰。」（`tmdbBundled` 為 false 時的那句，dev 照同樣句型改：把 TMDb 那半句換成既有的「自動取得元資料」。）
- 不加新的顏色或元件；連結用既有的文字連結樣式。

## Acceptance Criteria

1. **精靈多一個 Gemini 欄位，存得進去、用得到。**
   - 前端：`ApiKeysStep` 在 Claude 下面加 `WizardTextField`（`id="gemini-api-key"`、`type="password"`、`testId="gemini-key-input"`、placeholder `AIza…`）；`SetupWizard` 的資料型別、`clearOnSkip`、送出都帶 `geminiApiKey`；`setupService.ts` 對應成 JSON `gemini_api_key`。
   - 後端：`models.SetupConfig` 加 `GeminiApiKey string \`json:"gemini_api_key,omitempty"\``；`setupKeyUpdates` 收第三個參數，非空白就寫 `KeyGemini`；`keysWritable()` 的拒絕規則照舊（三把任一有填、又沒 `ENCRYPTION_KEY` → 整個拒絕、什麼都不寫）。
   - 刪掉／改正兩處過期註解（`ApiKeysStep.tsx`、`models/settings.go`）。
   - Go 測試：只填 Gemini → resolver `Get(KeyGemini)` 拿到值、source 是 secret；沒 `ENCRYPTION_KEY` 只填 Gemini → `ErrKeysNotWritable` 且沒有任何設定被寫入。**用真的遷移鏈與真的 secret store**（Rule 28），不要 mock resolver。
   - 前端 spec：`setupService.spec.ts` 斷言 `gemini_api_key` 有送出；`ApiKeysStep.spec.tsx` 有 Gemini 欄位、跳過時清掉。
2. **文案照 Sally 定稿的 N4-D。** 說明句、兩個 hint、兩個「怎麼申請」連結（`target="_blank" rel="noreferrer"`）、跳過提示的條件改成「Claude 與 Gemini 都沒填」。`tmdbBundled` 的兩種 TMDb 文案不動。
3. **只填 Gemini 的人走得完。** E2E（`tests/e2e/` 既有精靈 spec 旁邊）：精靈只填 Gemini → 完成 → `GET /api/v1/settings/keys` 的 `gemini` 為 configured、source `secret`。另外把「未查證 #1、#2」的 NAS 實測結果寫進 Completion Notes（模型、句數、花多久、有沒有撞額度）。
4. **設計對得上。** 本機截圖跟 `flow-n-setup-wizard/n4-d.png` 並排比對（Rule：dev 完成前必做 UX 截圖比對）；受影響的視覺基準只刪 darwin＋`git rm` 對應 `-linux`，不產 `-linux.png`（CI 會開 bootstrap PR）。
5. **沒有別的東西壞掉。** `nx test web`、`go test ./...`、`~/go/bin/staticcheck ./...`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check`、`check-design-refs.py` 綠。拿掉或改名 `data-testid` 前先 grep `tests/e2e`。

## 不做（刻意排除）

- README、FB 招募文、GitHub About 的改寫——是 founder-growth 的事，本單合併後馬上做。
- 精靈裡的聽寫金鑰（ElevenLabs／OpenAI）——「沒字幕要先聽聲音」的人比較少，先不擋路。
- 本機模型（Ollama／自架 whisper）——Edward Fang 的留言是真訊號，但那是定位問題，要另開 party mode，不塞進本單。

## Dev Agent Record

### Completion Notes

- **設計**：N4-D 由 Alexyu 的 Pencil inline agent 依 Sally 提示詞改完並 commit（3c94192a，`n4-d.png`）；Sally 用 MCP 唯讀逐字比對文案、`ctx.problems` 全空。CR 後兩句文案再改（Gemini 提示拿掉「每分鐘 5 次」、警告拿掉「和檔名解析」），連同 N5-D 新增「Gemini 金鑰」一列，第二輪提示詞交給 inline agent。
- **實作**：前端 `ApiKeysStep`（說明、Claude 提示＋申請連結、Gemini 欄位＋申請連結、跳過提示兩種句子）、`WizardTextField` 加選填 `link`（新分頁、`↗` aria-hidden＋sr-only「（開新分頁）」）、`SetupWizard`（clearOnSkip、validate-step、complete 都帶 `geminiApiKey`）、`setupService` 型別、`CompleteStep` 摘要多「Gemini 金鑰」一列。後端 `models.SetupConfig.GeminiApiKey`、`setupKeyUpdates` 第三參數寫 `KeyGemini`、`validateApiKeysStep` 讀 `gemini_api_key`、完成 log 多 `has_gemini_key`；兩處過期註解改正。
- **未查證 #3 結果**：AI 檔名解析（`services.NewAIService`）只讀環境變數（`config/api_keys.go` 的 `GetGeminiAPIKey`／`GetClaudeAPIKey`），精靈或設定頁存的金鑰都用不到 → 精靈文案不提檔名解析；設定頁兩列 hint 仍寫「也用於 AI 檔名解析」，開 backlog（見下）。
- **AC #3 偏離**：E2E 用 stub 的 API（開發機的實例早就完成設定，`/setup` 進不去），只證明瀏覽器送出 `gemini_api_key`；「存進去、翻譯用得到」由 Go 整合測試 `TestSetupWizard_GeminiOnlyKeyReachesTranslation` 以真的 AES-256-GCM 金鑰庫＋`TranslationDispatcher` 證明（只填 Gemini → 預設 provider 是 Gemini）。NAS 整集實測沿用 `gemini-translation-dispatch` AC #9（10-14，568 句全成功）。
- **測試**：`go test ./...` 全綠、vet、staticcheck 乾淨；`nx test web` 297 檔 4,721 測試全綠（CR 修正後重跑 setup 範圍）；typecheck、format:check、`lint:all` 0 errors；新 E2E `tests/e2e/setup-wizard-gemini.spec.ts` chromium＋mobile-chrome 通過；本機截圖 N4（桌機 1440、手機 390）與 N5 跟設計稿並排比對一致。
- **CR（Fable，跨模型）**：H1 `setup-api-keys-step`／`setup-complete-step` 視覺基準過期 → 刪 darwin 重產、`git rm` `-linux` 等 CI bootstrap；M2 警告寫「檔名解析」→ 拿掉；M3「每分鐘 5 次」過期（10-14 重量後是 5 秒一批＝每分鐘 12 次）→ 精靈改「免費方案有速率限制」；M4 本節；L5 E2E 只驗線路 → 如上說明；L6 `↗` 無障礙 → sr-only；L7 其他 handler 錯誤訊息只提 Claude → backlog；L8 只填 TMDb 時警告會消失 → 既有條件，不改（來源建置那句提到元資料，改條件會說錯話）。

### Discovery（Rule 24 lane ③，已寫進 sprint-status backlog）

- `backlog-ai-parse-reads-env-keys-only`：AI 檔名解析只讀環境變數；設定頁 Claude／Gemini 兩列 hint 說「也用於 AI 檔名解析」是錯的；設定頁 Gemini hint「每分鐘 5 次」過期。
- `backlog-claude-only-error-copy`：`subtitle_pipeline_handler.go:112`、`nfo_localizer_handler.go:136/183/228`、`key_settings_handler.go:222` 的錯誤訊息只叫人設定 Claude，沒提 Gemini（閘門本身已用 `translationDispatcher.IsConfigured`，行為正確）。
- `backlog-unraid-template-main-tag`：`unraid-template/vido.xml` 指向 `:main`（每次合併都變的開發版）；上架 Community Apps 前要改 `:latest`，並補 `ca_profile.xml`。
