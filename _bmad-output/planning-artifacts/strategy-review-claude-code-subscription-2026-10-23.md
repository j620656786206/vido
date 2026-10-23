# 策略討論：AI 改接 Claude Code（用訂閱額度取代 API 金鑰）？（2026-10-23）

**觸發**：Alexyu 問「有辦法改接 Claude Code、不要用 SDK 嗎？」，參考 [Hao0321/video-autopilot-kit](https://github.com/Hao0321/video-autopilot-kit)；接著追問「改接 Claude Code，使用者會不會變多？」
**方法**：主 agent 先讀程式碼和參考 repo，提出初版方案；再分兩輪找顧問——技術顧問（Fable）審初版方案，Founder／PM／Growth 三個角色回答「使用者會不會變多」。最後主 agent 親自抓官方法律頁核對政策原文。
**結論狀態**：四位顧問結論一致——**不做 `claude -p` provider**。主 agent 原本建議的「先拿檔名解析試做」已撤回。下面列出待 Alexyu 裁定的事項；在他裁定之前，不立 story、不寫程式。

---

## 一句話結論

改接 Claude Code **不會讓使用者變多**（卡點不在金鑰，目標使用者大多沒有 Claude 訂閱），**省錢只省到 Alexyu 自己**，而且**官方條款把這種用法劃在禁止那一側**。如果目的是省錢，有更乾淨的路（批次 API、便宜模型）；如果目的是成長，該先做上架和招測試者。

---

## 討論過程

### 第 0 步：先釐清兩個誤會

1. **vido 用的不是「Claude Agent SDK」**，是 Anthropic 官方的 Go 套件 `github.com/anthropics/anthropic-sdk-go`（`apps/api/go.mod`），直接打 Messages API、按 token 計費。用在：檔名解析、字幕翻譯、術語修正、NFO 在地化、評分工具（eval judge／`cmd/grade`）。真正想換的是這條「按 token 付費」的管道。
2. **video-autopilot-kit 的方向和 vido 相反**：那個 kit 自己完全不呼叫 LLM，是使用者開著 Claude Code／Codex，由 agent 透過 MCP 工具＋skill 去操作它（kit 只負責出計畫、驗收、封存收據）。Claude Code 是主導者。這適合有人在旁邊的剪片流程；vido 的 AI 工作大多是無人值守的背景工作（掃描自動解析、整季字幕過夜批次）。

### 第 1 步：主 agent 初版方案（已撤回）

- 新增一個 `claude-code` 供應商，實作現有的 `ai.TextCompleter`／`ai.Provider` 介面（`apps/api/internal/ai/provider.go`），每次呼叫就執行一次 `claude -p`（自訂系統提示、關工具、單輪、JSON 輸出），認證用 `claude setup-token` 產生的 `CLAUDE_CODE_OAUTH_TOKEN`；API 金鑰路徑保留當備援。
- 自列代價：訂閱有 5 小時／每週額度、會吃 Alexyu 寫程式的額度；失去 1h prompt cache 控制；每次啟動程式慢 1～3 秒；花費追蹤要重做；映像變大；條款不確定。
- 建議先拿檔名解析（量小、不需快取）做 POC。

### 第 2 步：技術顧問（Fable）審查

**結論：不建議做。** 不是技術做不到，而是官方把這條路劃在「請用 API key」那一側，而且解決不了真正的痛點（錢），反而吃掉 Alexyu 自己的額度。

初版方案**誇大**的地方：
- 啟動程式 1～3 秒不是主要問題：一個 10 句 chunk 本來就要 10～20 秒；Gemini 路徑現在每次還刻意等 5 秒（`translation_dispatcher.go:34`）。
- 不是完全失去快取：真正失去的是 **1h TTL**（`provider.go:87-89`，季批次會跨過 5 分鐘）和 breakpoint 位置的控制（`pipeline.go:1370-1385`）。

初版方案**漏掉、而且是致命**的地方：
- **兩層重試打破現有設計**：`claude.go:166-170` 刻意 `WithMaxRetries(0)`，讓重試只發生在 Governor 的預算檢查之下（`claude.go:326-337`、`governor.go:68-80`）。CLI 內部會自己重試、關不掉，預算閘門和限流會坐在黑盒上面。
- **花費會記假帳**：`claude.go:343-345` 每次呼叫把 token × 目錄單價記進 `Budget.RecordLLM`（`budget.go:247`）。訂閱下實際成本是 $0，但會記出假美金、可能觸發 `AIRunBudgetUSD` 暫停（`config.go:110`、`pipeline.go:480-484`）；而「本月 AI 花費」的誠實規則是「沒量到的錢寫次數、不寫 $0」（`subtitle_spend_service.go:16-40`）。整個花費模型需要新增一條「額度」的計法，不是改個數字。
- **錯誤分類退化**：`classifyErr`（`claude.go:252-303`）靠 HTTP 狀態碼分辨 401／404／餘額不足／429；CLI 的 JSON 輸出只給 `is_error` ＋文字。「額度用完、幾點重置」沒有機器可讀的欄位。
- **Alpine 映像**（`Dockerfile:97`）：CLI 在 musl 上需要的相依套件待驗，可能得裝整套 Node。
- **額度是決定性的**：一集約 80 次請求，一季就能吃光 5 小時窗，而且和寫 code 共用。

其他被否決的選項：
- **Agent SDK 給 Go 用**：沒有，只有 Python／TypeScript；其他語言官方說就是用子程序跑 `claude -p`。
- **長駐 Claude Code 程序（stream-json 輸入）**：所有 chunk 會落在同一段對話，前面的句子會漏進後面的翻譯；vido 的 chunk 是刻意無狀態的（`pipeline.go:1359-1400`）。
- **多句併一次呼叫**：batch size 10（`prompts/subtitle_translator.go:42`）和 4096 max_tokens（`translation_service.go:36`）是為了截斷後對半重送調過的，不該為了省啟動時間去動。
- **透過 OpenAI 相容端點接訂閱**：Claude Code 不提供 HTTP 端點；把訂閱轉成 API 的第三方 proxy 正是被禁止、被封過的東西。

Fable 指出唯一合規的「用訂閱」形狀，就是 kit 那種：**Alexyu 自己開 Claude Code，vido 只當 MCP 工具**（例如 `next_untranslated_chunk`／`submit_chunk`／`receipt`），只適用手動、一次性的工作，背景工作仍留在 API 金鑰。

### 第 3 步：Founder／PM／Growth 回答「使用者會不會變多？」

三位都回答**不會**。

**Founder**
- vido 的目標是做給小眾，不是個人工具（`PRODUCT.md`，2026-08-25 確認）；repo 公開、Apache-2.0、GHCR 有公開 image、有 `unraid-template/vido.xml`。
- 現階段的關卡是**留存不是人數**：北極星是「每週自動產出 ≥1 支繁中字幕的安裝數」；10 位外部測試者裡一週後還在用的要 ≥5 人才繼續（`_bmad-output/planning-artifacts/prd/success-criteria.md`）。
- 繁中 NAS 玩家和 Claude Pro/Max 訂閱者的交集只有工程師那一小群，他們本來就弄得到 API key。
- 訂閱額度無法事先報價，會削弱「花多少先講清楚」的定位；核心路徑押在別人一次政策調整上。
- 要成長該押：Gemini 免費額度當第一次啟動的預設路徑、修安裝阻力（`vido.xml` 介紹已過時、沒設 `ENCRYPTION_KEY` 網頁存不了金鑰）、上架 Unraid CA、招滿測試者、長期走本機 Ollama「零金鑰」路線。

**PM**
- 外部安裝數是 0：GitHub 1 star，FB 招募文 26 則留言、0 人安裝（`PRODUCT.md`「Evidence on Hand」）。人在金鑰那步之前就流失了，**沒有數據**顯示金鑰是瓶頸。
- 零花費的路已經有：只用 Gemini 在正式機翻完一整集——568 句、13.7 分鐘、59 次請求全部成功（`sprint-status.yaml` 的 `gemini-translation-dispatch`）。
- 換成訂閱是把一種麻煩換成另一種：`claude setup-token` 要先裝 Claude Code 再登入，比複製一把 Gemini 金鑰難；`creditStop.ts` 的「儲值後繼續」會變成「等 5 小時或下週」。
- 真要做，要改精靈（`N4-D`）、`SpendSection`、同意對話框、`creditStop`，都會動到 `.pen` 和信任文案，不是小改。
- 停損線：2026-11-15 前沒有外部的人裝起來並跑出字幕，就回頭檢視定位、不加功能。
- 最小實驗：私訊測試者時多問三題——卡在哪一步？有沒有 Claude Pro/Max？願不願意申請 Gemini 金鑰？

**Growth**（有上網查）
- 已查證：Claude 在香港、澳門、中國大陸都不能用（[supported-countries](https://www.anthropic.com/supported-countries)），簡中圈直接排除，只剩台灣。
- 推測：台灣 NAS 玩家裡付 Claude Pro/Max 的，大概個位數百分比。
- 沒字幕的片要聽聲音，本來就需要 ElevenLabs 或 OpenAI 金鑰（README），「完全不用金鑰」本來就做不到。
- 競品已經有更順的路：Lingarr 支援十幾家含本機模型、Subtitlarr 已上架 Unraid CA 支援 Ollama／Gemini。自架圈愛聽的是「免費／跑在本機」。
- 前例都是負面的（二手報導，日期未逐一核對）：OpenCode 被要求移除訂閱整合、OpenClaw 在 2026-04 被切斷。
- 測需求：GitHub Discussion 投票（Claude 訂閱／Gemini 免費／本機 Ollama／DeepSeek 這類相容端點）；README 加一行請想要的人按 👍。
- 真正能帶來使用者的管道：上架 Unraid CA（Growth 在 CA 搜尋沒查到 vido，需自行確認）；在 PTT、Mobile01、r/selfhosted 貼翻譯前後對照並附實際花費；主打 ChineseSubFinder 停更後的空位。

### 第 4 步：主 agent 親自核對官方條款

2026-10-23 抓取 [code.claude.com/docs/en/legal-and-compliance](https://code.claude.com/docs/en/legal-and-compliance)，原文重點：

- 「**OAuth authentication** is intended exclusively for purchasers of Claude Free, Pro, Max, Team, and Enterprise subscription plans and is designed to support ordinary use of Claude Code and other native Anthropic applications.」
- 「**Developers** building products or services that interact with Claude's capabilities, including those using the Agent SDK, should use API key authentication … Anthropic does not permit third-party developers to offer Claude.ai login into their own applications, or to route requests through Free, Pro, or Max plan credentials on behalf of their users. Moreover, developers may not collect, store, or intermediate Claude.ai credentials or session tokens」
- 「Advertised usage limits for Pro and Max plans assume ordinary, individual usage of Claude Code and the Agent SDK.」
- 「Anthropic reserves the right to take measures to enforce these restrictions and may do so without prior notice.」
- 允許的例外：使用者用自己的訂閱登入**未修改的 Claude Code 執行檔**（含平台代管 Claude Code 的情況）。

解讀：vido 要儲存 `setup-token` 產生的 token、在背景替整個片庫發請求，落在「不得儲存憑證」「額度以一般個人使用為前提」的範圍。單人自用時「on behalf of their users」字面上有縫，但整體意圖很清楚；一旦 vido 發給別人用，就直接踩線。最壞後果是 Alexyu 拿來開發 vido 的 Claude 帳號被處置。

### 未查證、引用前要再核對的說法

- Fable 提到的 CLI 細節（`--bare` 未來會成為 `-p` 的預設且不讀訂閱登入、`__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__`、版本號）——主 agent 沒有親自核對。
- Fable 給的 Haiku 5.5 價格——沒有核對。
- Growth 引用的 OpenCode／OpenClaw 事件——二手報導。

---

## 顧問建議的替代路線

**如果目的是省錢**
1. 先打開「本月 AI 花費」看實際數字（`parse_usd` ＋ `translated_usd`）。每月 < US$10 就什麼都不用改。
2. 花費較高：過夜翻譯改走 Anthropic Message Batches API（官方定價五折，可和 1h 快取疊加）。代價是字幕流程現在是同步逐 chunk（`translation_service.go:674-713`），要加一條非同步路徑——需要立 story。
3. 拿現有評分考卷（`cmd/grade`、golden-v2）試便宜的模型，零架構變更；評測本身會花一點真錢（約一集的量）。

**如果目的是讓使用者變多**
- 確認／完成 Unraid Community Apps 上架，並更新 `vido.xml` 過時的介紹。
- 修掉沒設 `ENCRYPTION_KEY` 就存不了金鑰、讓第一次安裝卡住的問題。
- 在 PTT、Mobile01、r/selfhosted 貼翻譯前後對照，附實際花費。
- 私訊測試者時加三題（卡在哪一步／有沒有 Claude 訂閱／願不願意申請 Gemini 金鑰）。

**如果就是想用訂閱**
- 唯一合規的形狀：Alexyu 自己開 Claude Code，vido 提供 MCP 工具給它操作，只用在手動、一次性的工作。可立成 P3 story。

---

## 待 Alexyu 裁定

1. **動機是省錢，還是想用訂閱？** 省錢 → 走「替代路線：省錢」；想用訂閱 → 只剩「vido 當 MCP 工具」這一種合規做法，要不要立 P3 story？
2. **接受上面的政策解讀，還是要自己去問 Anthropic（法律頁有 contact-sales 連結）再談？**
3. **要不要授權在 golden-v2 上跑一次便宜模型的評測？**（會花一點真錢，約一集的量）
4. **成長方面**：在停損日 2026-11-15 前，要不要優先做上架 Unraid CA、修 `ENCRYPTION_KEY` 安裝阻力、發對照貼文？
