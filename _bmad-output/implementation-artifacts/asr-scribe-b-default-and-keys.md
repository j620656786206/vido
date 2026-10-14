# Story asr-scribe-b-default-and-keys: 填了 ElevenLabs 金鑰，聽寫就預設走 Scribe；沒填照舊用 Whisper

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-14。接 asr-scribe-a-provider。Alexyu 2026-10-14 裁定 A：預設換成 Scribe，Whisper 保留可選。
     行號為 main 591b7403。 -->

## Story

身為 Vido 使用者，
我要在「金鑰」設定頁多填一把 ElevenLabs 的金鑰，之後沒有字幕軌的片就自動用 Scribe 聽寫、估價和實際扣款都照 Scribe 的價錢，
沒填的人什麼都不用改，照舊用 OpenAI Whisper。

## 規則（裁定）

1. **選擇順序**：有 ElevenLabs 金鑰 → Scribe；沒有但有 OpenAI 金鑰 → Whisper；自架端點（`ASR_BASE_URL` 指到自己的）→ 照舊自架 Whisper 相容端點。設定頁可以手動選「用 Whisper」蓋過順序（下拉或單選：自動／ElevenLabs／OpenAI）。
2. **金鑰一律選用**，不進首次設定精靈。
3. **文案**：估價和同意畫面寫「語音辨識：ElevenLabs Scribe」或「OpenAI Whisper」，價錢跟著走。不能出現「免費」（ElevenLabs 免費額度不能商用、要標註來源）。

## 查到的事（2026-10-14）

- 金鑰種類是封閉集合：`key_resolver.go:16-23`（`claude`、`tmdb`、`openai`），secret 名 `:45-49`，`EnvKeys` `:54-58`，兩個 switch `:123`／`:136`；設定存檔比 env 優先（`:166-188`，`key_source` 由此來）。**不用 migration**（secret 按名字存：`elevenlabs.api_key`）。
- 加一種金鑰要動：`key_resolver.go`、`key_settings_service.go:57` `orderedIDs`、`key_settings_handler.go:67-71`／`:110-114`、`config.go`／`api_keys.go`、`main.go:234-238`。前端 `keySettingsService.ts:17` `KeyName`、`ApiKeysForm.tsx:58-90` `KEY_ROWS`（openai 那列現在寫「雲端 ASR（選配）」）、`:92` `EMPTY_VALUES`。「測試金鑰」只支援 claude（`key_settings_handler.go:157`）——本單不擴。
- 選 provider 的地方：`asr_provider_holder.go` `cached` 是 `*ai.WhisperClient`（`:46`）要改成 `ai.ASRProvider`；`Get` `:98-136` 只看 `KeyOpenAI`；`IsConfigured` `:142-144`。`main.go:677-679` 建 holder、`:772` 印 `asr_configured`。
- 估價只有一個 `selfHosted bool`：`generation_candidates.go:472`、`:513-526`、`:773`（`asrRate`）、`:908`、`:940`；`transcription_estimate.go:119`、`:134`、`:249-271`；`main.go:1184`、`:1276` 開機時用 `IsSelfHostedASRBaseURL` 算死。要改成「問 holder 現在的 provider 每分鐘多少錢」（金鑰可以在執行中加）。活動頁花費讀的是實際記錄（`subtitle_spend_service.go`），不用改。
- 回應欄位 `self_hosted_asr`（`generation_candidates.go:358-361`、`transcription_estimate.go:58`）前端用來寫「自架（不另計費）」vs「OpenAI」：`consentSelection.ts:491-509`、`GenerationConsentView.tsx:170-174`／`:207`／`:353`／`:616`、`CandidateListPanel.tsx:152-153`／`:723`／`:976`、`generateCostView.ts:52`／`:159`。要加 `asr_provider`（`openai|elevenlabs|self_hosted`）欄位，前端照它寫字；`self_hosted_asr` 保留不刪（舊前端）。
- 缺金鑰的標籤 `"asr"`：`generation_batch_handler.go:62-73`、`transcription_service.go:507-517`；前端 `creditStop.ts:13` provider 名稱對照表要加 `elevenlabs`。
- 文件要改：`README.md:31,80,84,102-103`、`docs/development.md:53,185-188`、`docs/deployment.md:140,148,163-167,211`、`docker-compose.yml:59-63`、`.env.example:135-138,181-189,205`。

## Acceptance Criteria

1. 金鑰種類加 `elevenlabs`（env `ELEVENLABS_API_KEY`，secret `elevenlabs.api_key`）；設定頁多一列「雲端聽寫 ElevenLabs（建議）」，說明寫「填了就用 Scribe 聽寫，比 Whisper 準、每小時 US$0.22；留空用 OpenAI Whisper」。
2. 設定頁多一個「聽寫服務」選擇：自動（預設）／ElevenLabs／OpenAI，存在 settings；holder 照規則 1 選，金鑰或選擇改了下一次聽寫就生效（不用重啟）。
3. 估價（單部、批次、resume 扣錢）與實際扣款都用目前 provider 的費率；回應多 `asr_provider` 欄位；同意畫面與單部估價文案照 provider 寫「ElevenLabs Scribe」／「OpenAI Whisper」／「自架（不另計費）」。
4. 批次因 ElevenLabs 額度用完停下時，畫面說是 ElevenLabs、連到它的儲值頁（`creditStop.ts`）。
5. `asr_configured` 開機 log 印出目前 provider；缺金鑰的錯誤訊息說清楚缺哪一家。
6. 文件：README 的「要用 AI 字幕先確認三件事」與費用表加 ElevenLabs；`.env.example`、`docker-compose.yml` 註解、deployment／development 同步。
7. 測試：holder 三種選擇各一條（含執行中加金鑰）；估價用 Scribe 費率一條；前端文案三種各一條；既有 Whisper 路徑測試不改。
8. **正式機驗證**：NAS 填 ElevenLabs 金鑰，用 `vido-eval-clip` 的丟棄容器跑 See S01E02 10 分鐘片段 → 字幕打熊段對、花費記成 Scribe 價；再跑一整集看「長片整段漏掉」的疑慮。

## Tasks / Subtasks

- [ ] T1 金鑰種類＋設定頁列
- [ ] T2 holder 改 `ai.ASRProvider`、選擇規則、「聽寫服務」設定
- [ ] T3 估價／扣款費率跟 provider 走、`asr_provider` 欄位
- [ ] T4 前端文案、creditStop
- [ ] T5 文件
- [ ] T6 測試、正式機驗證

### Time-dependent visual coverage

設定頁多一列、同意畫面文案改 → 相關 visual 基準會變（故意的）。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List
