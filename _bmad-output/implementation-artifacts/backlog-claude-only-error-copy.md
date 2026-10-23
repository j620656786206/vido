# backlog-claude-only-error-copy：沒放金鑰時，提示說「Claude 或 Gemini 都可以」，不再只叫人去拿 Claude 金鑰

Status: in-progress

<!-- SM Bob create-story 2026-10-23；行號為 main `3e476650`；.pen 節點 id 由 Sally 用 Pencil MCP 唯讀讀取。 -->

**Source:** Rule 24 lane ③，`disc-2026-10-setup-wizard-byok` CR L7 立案（2026-10-19）；10-21 `disc-2026-10-preview-judge-needs-claude` AC #8 補進試跑的提示。2026-10-23 founder-pm 判定這是停損日（11-15）前唯一值得做的一張，Alexyu 選 A。

## 為什麼要做

FB 招募文的留言告訴我們：多數人沒有 Claude 金鑰，想用自己的 Gemini 免費額度。精靈（N4-D）已經寫「Claude 或 Gemini 都可以」，Vido 也真的只要其中一把就能翻譯。

但新用戶兩把都還沒放、第一次打開「批次產生字幕」時，提示寫的是「批次產生字幕需要翻譯（Claude）金鑰」。沒有 Claude 金鑰的人看到這句，會以為一定要去申請 Claude、付錢，然後就走了。試跑模型的提示、API 回的建議也一樣只講 Claude。

閘門本身沒問題（`translationDispatcher.IsConfigured`，Gemini 一把就會開），只是字沒跟上。

## Acceptance Criteria

1. **批次產生字幕的提示（F28）**：伺服器說缺的是翻譯金鑰時（`missing_key: claude`），寫「批次產生字幕需要一把翻譯金鑰（Claude 或 Gemini）。到金鑰設定儲存後就能開始；清單可以先看。」不知道缺哪把時，寫「批次產生字幕需要翻譯金鑰（Claude 或 Gemini）與語音辨識（ASR）金鑰。…」。ASR 那句不變。
2. **試跑模型的提示**：兩把都沒有時（`AI_NOT_CONFIGURED`），寫「先到 設定 → API 金鑰 存一組 Claude 或 Gemini 金鑰」。前端和後端的 hint 用同一句。
3. **API 的建議文字**：
   - `POST /subtitles/pipeline/run` 的 `AI_NOT_CONFIGURED`：「請到 設定 → API 金鑰 儲存 Claude 或 Gemini 金鑰，或設定 CLAUDE_API_KEY／GEMINI_API_KEY 環境變數。」
   - NFO 在地化的 `NFO_LOCALIZE_DISABLED`：提到 `CLAUDE_API_KEY or GEMINI_API_KEY`。
   - 錯誤代碼、HTTP 狀態都不變。
4. **設計稿一致**：F28-D-v2 的說明文字（`pMCK7`）和 J10 的註解（`vl0KX`）改成 AC 1、2 的句子（Task 0）。畫廊 `generation-consent/not-configured` 跟著換字，更新 `-darwin` 基準、刪掉舊的 `-linux`。
5. **測試**：前端 AC 1 三種情況、AC 2 一條；後端 pipeline 的 suggestion 同時含 `CLAUDE_API_KEY` 和 `GEMINI_API_KEY`、model preview 的 hint 含 Gemini。舊的「只講 Claude」字串在 `apps/web/src` 和 handlers 裡不再出現（測試除外）。

## 不要做的事

- 不改閘門、不改 `missing_key` 的值（`claude` 照舊代表「翻譯金鑰」，這是 `[@contract-v1]`）。
- 不改「測試金鑰」按鈕的訊息（`key_settings_handler.go:222`）：那個按鈕本來就只測 Claude 金鑰，`tester` 是 nil 代表沒接線，不是使用者沒放金鑰。
- 不改「使用：Claude（你的金鑰）」這類顯示目前用哪把的字（那些已經依實際的金鑰顯示）。
- 不碰精靈（N4-D 已經是對的）。

## Tasks

- [ ] **Task 0（設計，Alexyu 執行）**— 把下方提示詞貼給 Pencil 的 Inline AI Agent；⌘S；`python3 scripts/export-pen-screenshots.py`；只留 `f28-d-v2.png` 和 J10 的截圖；回報完成。**不要切分支、不要 commit，存檔後跟我說。**
- [x] Task 1 — 後端三處文字＋測試（AC 2、3、5）
- [x] Task 2 — 前端兩處文字＋測試（AC 1、2、5）
- [x] Task 3 — 畫廊基準；lint、typecheck、相關測試（AC 4）

## Sally 的 Pencil 提示詞（Task 0，整段貼給 Inline AI Agent）

```
在 ux-design.pen 只改兩段文字，其他都不要動。

1. 設計稿「F28-D-v2 · 批次生成 — 尚未設定金鑰」（節點 N6dxid）裡的說明文字 pMCK7（名稱 wp-sub）。
   目前內容：批次產生字幕需要翻譯（Claude）金鑰。到金鑰設定儲存後就能開始；清單可以先看。
   改成：批次產生字幕需要一把翻譯金鑰（Claude 或 Gemini）。到金鑰設定儲存後就能開始；清單可以先看。
   只改文字，字級、顏色、寬度設定都不要動。它的欄位（wp-col，MzW4o）有 744px 寬，放得下一行。

2. 規格稿「J10 · 模型列三態（試跑 20 句）」（節點 ctRsy）裡的註解文字 vl0KX（名稱 note）。
   目前內容：金鑰未設定 → 按鈕停用，說明「先到 設定 → API 金鑰 存一組 Claude 金鑰」（同 J9 ⑥）。
   改成：金鑰未設定 → 按鈕停用，說明「先到 設定 → API 金鑰 存一組 Claude 或 Gemini 金鑰」（同 J9 ⑥）。
   只改文字。

3. 存檔（File ▸ Save）。
```

## Dev Notes

- 前端：`apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:724-733`（`notConfiguredBody`）、`apps/web/src/hooks/useModelPreview.ts:44`。對應測試 `CandidateListPanel.spec.tsx:1601-1623`、`useModelPreview.spec.ts:14`。
- 後端：`subtitle_pipeline_handler.go:112`（測試 `subtitle_pipeline_handler_test.go:213` 只斷言含 `CLAUDE_API_KEY`）、`model_settings_handler.go:147`、`nfo_localizer_handler.go` 兩處 `NFO_LOCALIZE_DISABLED`（前端 `NfoLocalizeAction` 對這個代碼顯示自己的狀態，不讀這句）。
- Gemini 的環境變數是 `GEMINI_API_KEY`（`config/config.go:204`）。
- 畫廊：`-gallery.fixtures.tsx` 的 `generation-consent/not-configured`；換字幅度可能低於快照門檻，照 /ship 的做法：刪 `-darwin` 再 `test:visual:update-missing`。

### Time-dependent visual coverage

- N/A — 只換文字。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5，Amelia），2026-10-23。

### Completion Notes List

- 程式改完，等 Task 0（設計稿）。
- 後端：pipeline 的建議同時寫 Claude 或 Gemini 和兩個環境變數；試跑的 hint 跟前端同一句；NFO 兩處改成兩個環境變數。RED → GREEN：把兩個 handler 改回舊字，`TestRunPipeline_409AndNoEnqueueWhenTranslationKeyMissing`、`TestPreviewModel_StatusMapping` 會紅。
- 前端：F28 兩種說明、試跑提示換字；`CandidateListPanel.spec.tsx`、`useModelPreview.spec.ts` 130 條綠。
- 畫廊：刪舊 `-darwin`／`-linux`，`test:visual:update-missing` 重產 `-darwin`，畫面上是新句子、其他沒變。
- gofmt、go vet、staticcheck、eslint（只有既有的 1 個 warning）、Prettier 過。
- 範圍外（不修，記下）：同一個面板第二行「使用：Claude（尚未設定金鑰）」在兩把都沒有時仍寫 Claude——那是「預設會用哪個模型」，跟提示不衝突，但沒 Claude 的人可能還是會疑惑。等有人真的卡住再處理。

### File List

- apps/api/internal/handlers/subtitle_pipeline_handler.go、subtitle_pipeline_handler_test.go
- apps/api/internal/handlers/model_settings_handler.go、model_settings_handler_test.go
- apps/api/internal/handlers/nfo_localizer_handler.go
- apps/web/src/components/subtitle/consent/CandidateListPanel.tsx、CandidateListPanel.spec.tsx
- apps/web/src/hooks/useModelPreview.ts、useModelPreview.spec.ts
- tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/not-configured/default-visual-darwin.png（更新；刪 `-linux` 讓 CI 重產）
- 本檔、sprint-status.yaml

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-23 | create-story（SM Bob）＋ Sally 的 Pencil 提示詞（節點用 MCP 唯讀確認）。 |
