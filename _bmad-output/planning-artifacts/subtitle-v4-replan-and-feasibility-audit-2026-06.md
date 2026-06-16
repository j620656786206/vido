# 字幕核心 V4 重規劃 ＋ V4 外部依賴可行性稽核

**日期：** 2026-06-16
**作者：** Party-mode 規劃會議（引導式）— Mary／Winston／John／Murat／Bob
**觸發點：** Live POC 證明字幕「Route A」（抓取現成人工字幕）在真實世界對繁中是結構性不可行的。負責人（Alexyu）提出流程問責：可行性必須在**規劃當下**驗證，而不是事後反應式才發現。詳見 `.claude/memory/feedback_feasibility_gate_before_spec.md` 與 `project_subtitle_route_a_poc.md`。

> **新工作守則 —— 可行性閘（Feasibility Gate）：** 任何依賴外部資源（網站／API／模型／工具／第三方帳號）的需求，在被寫進 spec 當「定案」之前，必須先有一支薄薄的 **live spike** 證明 happy path 真的通。沒過 live POC 的外部依賴需求，不准標成 `ready`。SM 在 DoD 守這道閘；分析師在分析時插「可行性紅旗」。

---

## PART A — V4 外部依賴可行性矩陣（主動掃描）

2026-06-16 live 探測。從 `apps/api` 清點出 12 個 HTTP host ＋ 數個外部 binary。

| 依賴 | 類型 | 憑證／設定 | Live 實測 | 判定 |
|---|---|---|---|---|
| TMDB API (`api.themoviedb.org/3`) | REST | `TMDB_API_KEY` | HTTP 401（活著、key-gated） | ✅ 可靠 —— 官方免費 key、易取 |
| TMDB 圖片 CDN (`image.tmdb.org`) | CDN | 無 | 可達 | ✅ |
| Wikipedia (`zh.wikipedia.org/w/api.php`) | REST | 無 | HTTP 200、真 JSON | ✅ 可靠 |
| **Douban 豆瓣** (`search.douban.com`) | 爬蟲 | 無（`ENABLE_DOUBAN=false`） | HTTP 200 但**靜態 HTML 為 JS 渲染、無結果標記** | ⚠️ **脆弱 —— 需 spike。** 豆瓣強反爬＋搜尋結果 JS 渲染，goquery 爬蟲很可能只拿到空殼。預設關閉本身就代表低信心。 |
| Assrt 射手 (`api.assrt.net/v1`) | REST | `assrt_api_key` | 活著，但 token 註冊是半棄置「射手(伪)」鏡像上壞掉的 SPA | ❌ **token 實測無法取得**（使用者 2026-06-16 親自驗證：根本無法註冊）→ 對此使用者**不可用** |
| Zimuku 字幕庫 (`zimuku.org`) | 爬蟲 | 無 | **Yunsuo WAF、HTTP 404 ＋ JS 挑戰頁** → 每筆查詢回 `ErrCaptchaDetected` | ❌ WAF 牆死（POC 已證）。硬編域名也過時。 |
| OpenSubtitles (`api.opensubtitles.com/api/v1`) | REST | key ＋ 登入 | 活著 | 🔑 可用但**繁中覆蓋薄**（亞洲字幕組不上傳 —— 負責人領域判斷）。不適合當繁中主來源。 |
| Gemini (`generativelanguage.googleapis.com`) | REST | `GEMINI_API_KEY` | HTTP 403（活著、key-gated） | ✅ 可靠 |
| Claude (`api.anthropic.com/v1`) | REST | `CLAUDE_API_KEY` | HTTP 405（活著、key-gated） | ✅ 可靠 |
| OpenAI Whisper (`api.openai.com`) | REST | `OPENAI_API_KEY` | HTTP 401（活著、key-gated） | ✅ 可靠（雲端、付費） |
| ffmpeg／ffprobe | binary | 須在 PATH | ❌ **本機開發機未安裝** | ⚠️ 部署依賴 —— Route C（抽音軌）與技術 metadata 必需。須確保 Docker image 內含。缺了會靜默降級。 |
| OpenCC | Go lib（in-process） | 無 | `longbridgeapp/opencc` 純 Go | ✅ 簡→繁 OK，免 CLI |
| qBittorrent | user REST | host ＋ 帳密 | 用戶環境 | 👤 需在用戶自己的環境驗證 |

### A1. 核心診斷

- **脆弱性不是散布在整個 V4，而是局部的。** 西方／商用依賴（TMDB、Wikipedia、三家 LLM、qBT、ffmpeg）全都健康或可控。
- **每一個不可控／被擋的依賴，無一例外都是「華語社群字幕／metadata 來源」**（Zimuku、Assrt、Douban）。這正好是 V3 時代的產品身分所在那一層。
- **真正重要的分野：可控 vs 不可控。**
  - 可控且可靠：TMDB、Wikipedia、LLM APIs（付費即穩）、ffmpeg（自己 bundle）、OpenCC（in-process）。
  - 不可控且脆弱：Zimuku／Assrt／Douban（WAF 軍備競賽、註冊牆、反爬 JS）。
- **主動揪到的新隱患：** Douban 在地化 fallback（P1-004）很可能根本抓不到。任何在地化設計依賴它之前，先補一支 spike。

---

## PART B — 字幕核心重規劃（feasibility-first）

### B0. POC 驗證結果（2026-06-16 — Route C 端到端實測 PASSED）

用 `apps/api/cmd/route-c-poc/main.go`（串接 vido 真實零件）對真實 4K 檔
`The.Boys.S01E01 ...mkv`（7.4GB、SMB 掛載 /Volumes/data）跑前 240 秒：

- ✅ **➊ ffmpeg/ffprobe 抽音軌**：偵測 3 軌、正確選英文軌（eac3, 6ch），4 分鐘音檔 <1s 抽完。
- ✅ **➋ Whisper 轉錄**：87 句、~20s，正確英文逐字稿。
- ✅ **➌ Claude Haiku 4.5 翻譯**：40 句、~8s，**道地台灣繁中**（"Yo, I am so psyched for Invisible Force 2." → 「哎呀，我超期待看《隱形戰士2》。」）。
- ✅ **➍ OpenCC s2twp + Placer**：原子寫檔、`.bak` 備份、BCP 47 命名。

**結論：Route C「可控核心」紙上推論 → 實測通過。** 成本 3 次 <$0.10。

**live POC 揪出 2 個 mock 測試抓不到的正式碼 bug（→ backlog 修復項）：**
1. 🔴 `ai/claude.go:18` 預設模型 `claude-3-5-haiku-latest` 已被 Anthropic 下架 → 打過去 **404 not_found**。POC 以 `WithClaudeModel("claude-haiku-4-5-20251001")` 覆寫；**正式碼預設值需更新**。
2. 🔴 `ai/whisper.go` 未帶 `language` 參數 → Whisper 自動偵測**不穩定**：同一段英文音軌曾被誤判成中文、把英文對白聽寫成中文亂碼。**已修**：新增 `WithWhisperLanguage`（把 ffprobe 音軌語言 `eng→en` 餵入）。正式管線應由 `AudioExtractorService` 選定的音軌語言自動帶入。

**順手證實 keystone 必要性**：AI 自行把片名 *Invisible Force 2* 掰成《隱形戰士2》（與官方譯名不一致）→ 正是 backlog #2（術語表）+ #3（metadata context）要解的跨集譯名一致性問題。

### B1. 修正（改了什麼、為什麼）

- POC 證明 Route A（抓現成繁中人工字幕）結構性脆弱：Zimuku WAF 死、Assrt 註冊牆、OpenSubtitles 繁中薄、Douban 反爬。
- 現有風險文件（`prd/project-scoping-phased-development.md`）**確實**把「Assrt API 穩定性／Zimuku 爬蟲維護」列為高風險 —— 但它**斷言了 mitigation（「多來源冗餘」、「Assrt 官方 API 當可靠主來源」）卻沒驗證**。現實剛好相反：所有繁中來源**共享同一個失敗模式**（被 gated／被擋），所以冗餘是假象，而那個「可靠」的 Assrt 反而最難取得。**從頭到尾沒有任何 WAF／反爬／註冊牆風險被記錄下來。**
- **護城河重定位：** 從「繁中字幕 *抓取* 量大」→「可控的繁中 *生成* ＋ 跨集術語一致性 ＋ 媒體庫全面在地化」。抓取從來就不可防禦（任何人的 WAF 都能弄死它）；生成／在地化這套則活在我們自己的程式碼邊界內。

### B2. 重構的架構 —— 可控核心（Route A 對繁中已封死，Route C 為唯一路徑）

- **核心（只准用可控依賴）—— Route C 生成堆疊：**
  ffmpeg（bundle）→ Whisper 轉錄（本地 faster-whisper *或* 雲端 OpenAI）→ LLM 翻譯（Claude／Gemini，key-gated、可靠）→ **per-show 術語表** → OpenCC（in-process）。這裡每一個依賴，不是 bundle、就是 in-process、要不就是付費可靠的商用 API。
- **~~補充~~ → 已封死 —— Route A 抓取：**
  繁中三來源全斷（Zimuku WAF 死、**Assrt token 實測無法取得（使用者已驗證）**、OpenSubtitles 繁中薄）。**Route A 不再是繁中策略的一部分。** Zimuku 移除；Assrt/OpenSubtitles 程式碼留 dormant 但不列入依賴、UI 不呈現為可用路徑。**→ Route C 是繁中字幕的唯一路徑。**
- **差異化 —— metadata 在地化（greenfield，Section E）：**
  復用**同一套**生成基建（LLM ＋ 術語表）把 .nfo 的劇情／分集標題／角色翻成繁中，以**只增不覆蓋**的方式回寫成一份平行的繁中 .nfo，供 Kodi／Jellyfin／Plex 刮取。任何單機字幕工具（如 Subtitle Studio）都給不了這個。

### B2.1 架構決策 —— 引擎可插拔（租 commodity 引擎、自建繁中層）

**背景**：負責人提出兩個策略疑慮——(1) 自己刻 Whisper 轉錄是不是在重刻 OSS 早就做好的事？(2) Whisper 現在開源，未來若 OpenAI 搞 vendor lock-in，轉字幕功能會不會被廢？這節是對這兩點的架構性回答。

**先澄清一個關鍵混淆**：
- **Whisper「模型」**＝ 2022 年 OpenAI 以 **MIT 授權開源釋出**的權重，已下載到全世界、**收不回去**，可用 faster-whisper / whisper.cpp / OpenVINO 永遠本地跑。
- **Whisper「API」**＝ 付費雲端服務（`api.openai.com`）。**只有這個有漲價/限制/下架風險。**
- vido 現在打的是後者。**所以 lock-in 風險只在 API、不在能力本身。**

**核心原則**：
> **轉錄、甚至基礎翻譯都是 commodity —— 藏在介面後面當「可插拔引擎」，誰好用就插誰（雲端 API / 本地 faster-whisper / OpenVINO）。只自己「擁有」繁中差異化層（術語表 + metadata 在地化 + 媒體庫整合）。** 這一條同時解決「不重刻輪子」與「免疫 Whisper lock-in」。

**三層分工（租 vs 自建）**：

| 層 | 策略 | 候選（皆 OSS / OpenAI 相容） | 授權 | 硬體 |
|---|---|---|---|---|
| ASR 轉錄 | **租 · 可插拔** | Speaches（前 faster-whisper-server）、WhisperLive（含 **OpenVINO** 後端，吃 Intel iGPU）、hwdsl2/whisper-server；或雲端 OpenAI | 多為 MIT（採用前核對 LICENSE） | base/small 可在弱 NAS CPU 跑；large 需 GPU |
| LLM 翻譯 | **租 · 可插拔** | OpenAI 相容端點（Claude/Gemini 雲端，或本地 Ollama/LM Studio）；參考 llm-subtrans、subtitle-translator(rockbenben) | 多為 MIT | 雲端為主（繁中品質關鍵，弱 NAS 本地不夠） |
| **繁中在地化層** | **自建 · 唯一擁有** | 無人代勞：per-show 術語表、metadata .nfo 在地化、跨集一致性、媒體庫整合 | 你的 | — |

**授權核對（2026-06-16，逐一查 LICENSE）**：Speaches / WhisperLive / Subgen / faster-whisper / llm-subtrans / subtitle-translator(rockbenben) **全部 MIT、可商用、零 copyleft 義務**；whisper.cpp MIT、OpenVINO Apache-2.0、Whisper 權重 MIT。**→ 「租引擎」整層授權無虞。** **避開**：Bazarr(GPL-3.0)、LibreTranslate(AGPL-3.0) 等 copyleft（打包有 share-alike 義務）；pyannote 語者模型為 HF gated。**加碼**：Subgen 本身提供 OpenAI 相容 `/v1/audio/{transcriptions,translations}` 且已整合 Plex/Jellyfin/Emby/Bazarr → **可當 vido 直接指向的 drop-in ASR 後端，不只是參考**。

**關鍵省力點**：vido 的 `ai/whisper.go` 本來就打 OpenAI 標準 `/v1/audio/transcriptions`、且**已有 `WithWhisperBaseURL` option** → **把雲端換成本地 Speaches/WhisperLive，幾乎只要改一個 base URL**，不用重寫。lock-in 保險大半已就緒。

**lock-in 緩解結論**：轉字幕能力**不會因 OpenAI 怎樣而被廢**——本地 faster-whisper / OpenVINO / 其他託管 Whisper / 其他 ASR（Parakeet、Deepgram…）皆可頂上，只要它在 `ASRProvider` 介面後面。

**ArcSub 這類整包 app 不整合**：它是 React+Express 的獨立 web app、無對外 API、技術棧與 vido(Go) 不同 → **研究其做法（VAD/diarization/OpenVINO 選型）可，硬塞進來不可**。要插的是「引擎」，不是「整車」。

**新增 backlog 項（架構）**：
- A1. 定義 `ASRProvider` 介面（`Transcribe(audio) → SRT`），把 `whisper.go` 收斂成其一實作；新增「本地 Speaches/WhisperLive」實作（base URL 可設）。
- A2. 轉錄 provider + base URL 做成設定（雲端 / 本地 二選一，類似現有 `AI_PROVIDER`）。
- A3. 評估 OpenVINO 後端在目標 Intel NAS 的可行性（併入 spike S2）。

### B3. 優雅的連鎖反應 —— 最難的缺口自動降級

把 Route A 降級，也就把**時間軸對齊**問題（原稽核 #7／#8／#9：整體偏移／漸進漂移／ffsubsync/alass／手動 UI）一起降級了。AI 生成的字幕是**從音訊本身轉錄出來的** → 天生就對齊。所以一旦生成變成核心，這個單一最難的技術缺口就降為低優先。不必去打 ffsubsync/alass 那場軍備競賽。

### B4. Feasibility-gated backlog（優先順序）

每項標 `[PROVEN]`（基建已存在／已驗證）或 `[SPIKE]`（需先做可行性 spike）＋ 依賴類別。

1. `[PROVEN 基建 · 可控]` **AI 成本/配額控制** （`internal/ai/`）—— backoff、retry、throttle、token 計量。**任何批次 AI 功能的前置條件**（目前只有 429 偵測、無控制）。沒有它，批次翻譯/轉錄會燒錢失控＋撞 rate limit 整批失敗。
2. `[PROVEN 基建 · 可控]` **術語表 schema**（`show_glossary` 表）＋ 把 `TranslationService` 泛化成 `TranslationRequest{Fields, Glossary}`（Winston 的「兩刀」）。同時服務 Route-C 品質**與**在地化。**keystone（基石）。**
3. `[PROVEN 路徑 · 可控]` **翻譯吃 metadata context** —— 把劇名／劇情／角色譯名表餵進翻譯 prompt。便宜、品質 ROI 高；把辨識層接到翻譯層。
4. `[SPIKE · 部署]` **Whisper 本地 vs 雲端決策** —— spike faster-whisper（本地、免費、隱私，像 Subtitle Studio）vs OpenAI（雲端、付費）。**確保 Docker image 內含 ffmpeg**（已證開發機沒裝）。
5. `[PROVEN 路徑 · 可控]` **接線 Route C 自動 fallback** —— Route A 無結果時，轉錄 → 翻譯 → 落檔。定義明確觸發條件（稽核發現目前未定義）。
6. `[PROVEN 基建 · 可控]` **metadata 在地化（Section E）** —— 用共享基建把 .nfo 劇情/分集/角色翻成繁中；以只增不覆蓋方式回寫平行繁中 .nfo；保留原檔（絕不覆蓋）。類別級差異化。
7. `[已封死 · 不可控]` **Route A 對繁中確認無可用來源** —— 三條全斷：Zimuku WAF 死、**Assrt token 實測無法取得（使用者已驗證，2026-06-16）**、OpenSubtitles 繁中薄且不適用。**結論：Route A 不再是繁中策略的一部分。** 動作：**移除 Zimuku**（D3）；Assrt/OpenSubtitles provider 程式碼可留作 dormant（萬一未來取得憑證），但**不列入任何規劃依賴、UI 不呈現為可用路徑**。`assrt.go:22` 限流 bug 變 moot（無 token）。**→ Route C 成為繁中字幕的唯一路徑。**
8. `[SPIKE · 不可控]` **Douban 在地化 fallback** —— 依賴前先 spike（探針顯示 JS 渲染、爬蟲很可能瞎）。失敗就從在地化 fallback 鏈移除。
9. `[延後]` **時間軸對齊（ffsubsync/alass ＋ 手動 UI）** —— 只跟抓取的字幕（Route A）有關；已被 B3 降級。只有當 Route A 哪天變可靠才重訪。

### B5. 決策（2026-06-16 由負責人 Alexyu 鎖定）

- **D1 —— metadata 在地化（Section E）：先 SPIKE。** spike S1 過了才決定是否納入 V4 範圍。（最強差異化，但目前零 spec。）
- **D2 —— Whisper 本地 vs 雲端：HYBRID、把兩個模型解耦、＋ spike。**
  - **翻譯（LLM）留在雲端**（Claude／Gemini）—— 品質關鍵；無顯卡的 NAS 比不上雲端繁中品質/速度。這是差異化步驟，不在弱硬體上犧牲。
  - **Whisper 轉錄：預設雲端，本地 faster-whisper 當 opt-in**（隱私/離線；用戶接受慢）。
  - 理由：**「可控」≠「本地」** —— 付費商用 API 就是可控的（不受 WAF/棄置鏡像擺佈）。NAS 無獨顯；iGPU/QuickSync **不會**加速 LLM/Whisper。Subtitle Studio 的本地 Whisper 跑在 Apple Silicon 的 Mac 上 —— 與 NAS 硬體不可比。
  - **整條產線實作要點：** 影片 →【NAS 本地】ffmpeg 抽音 → ☁️ 上傳音訊給 Whisper API、收回原文字幕 → ☁️ 把字幕文字送給 Claude/Gemini、收回繁中 →【NAS 本地】OpenCC 校正 → 寫檔。NAS 只做「抽音、發 HTTP、寫檔」三件輕事；兩個重模型都在雲端 GPU 跑，**NAS 硬體弱不影響**。本地 opt-in = 加一個 `whisper_provider = cloud|local` 開關（類似現有 `AI_PROVIDER`）；翻譯永遠雲端。
  - **預設值 gated on spike S2**（在真實 NAS 上實測 faster-whisper）再定。
- **D3 —— Zimuku provider：移除。** WAF 死（Yunsuo）、硬編域名過時。保留＝死碼＋給 UI 假希望。移除 provider ＋ engine 註冊 ＋ 測試。
- **D4 —— 定位：主打「繁中 AI 生成 ＋ 在地化媒體庫」。** 可控、可防禦的護城河成為頭號賣點 —— 在更高的層次找回 V3 的繁中身分（不是「我們什麼都抓得到」）。

### B5.1 該跑的 spike（feasibility-gated —— 依賴它的 spec 寫之前必須先過）

| Spike | 它回答的問題 | 通過標準 |
|---|---|---|
| **S1 · .nfo 在地化**（鎖 D1） | 能不能用 LLM 在地化 .nfo ＋ 回寫，讓 Kodi/Jellyfin/Plex 顯示繁中？ | 真實播放器刮到只增的繁中 .nfo 並顯示翻譯後的劇情/角色；原 .nfo 保留 |
| **S2 · NAS Whisper benchmark**（鎖 D2 預設值） | faster-whisper `base`/`small` 在目標 NAS CPU 上的真實吞吐量？ | 量到每集分鐘數 → 決定雲端預設 vs 本地 opt-in 是否可行 |
| **S3 · Douban 在地化 fallback** | Douban 爬蟲是否真能 live 抓回繁中 metadata（它 JS 渲染）？ | 端到端解析到真結果；否則從在地化 fallback 鏈移除 |

### B6. 它接在哪裡

- Phase-1 後端（Epic 8/9）已 DONE，但是建在如今被證偽的「抓取優先」假設上。本重規劃**不丟棄**它 —— Route A 程式碼留作 best-effort 補充。
- Phase-3 UX3 Epic 6（`ux3-subtitle-v2`）與 Epic 7（`ux3-ai-subtitle`）仍是骨架 —— **依本重規劃（生成核心）來寫它們的 stories**，而非舊的抓取核心 brief。
- 從現在起，對每一個外部依賴 story 套用**可行性閘**。

---

## 附錄 A — Route C 價值實證：人工字幕 vs AI 生成（2026-06-16）

把整集 POC 產出的 AI 繁中字幕，與該影集旁既有的「人工」字幕（`The.Boys.S01E01...zh-TW.srt`）逐面向比對。測試素材：The Boys S01E01（4K、SMB）。

> ⚠️ 先決發現：那份標 `.zh-TW` 的「人工字幕」**實際內容是簡體中文＋大陸用語**（「你无法杀死」「算法」「接口」），檔名誤標。也就是說，使用者「手上現成」的繁中字幕，連語言都不對。

| 面向 | 人工字幕（既有） | AI 生成（Route C POC） |
|---|---|---|
| 語言 | ❌ 其實是**簡體**＋大陸用語 | ✅ **道地繁體**＋台灣口語 |
| 涵蓋範圍 | ✅ 含**畫面文字**（劇集卡、招牌、地點、人物註解） | ❌ **只有口白**（從音訊轉，看不到畫面文字） |
| 專有名詞 | ✅ 認得角色名（深海/The Deep、沃特/Vought）、較官式片名 | ⚠️ **自行掰**（隱形特務2、洶湧狂潮），把 "The Deep" 直譯成「深海怪物」 |
| 對白品質 | 簡體、部分生硬 | ✅ **自然口語**（「根本爛透了」「輾壓」「老二」） |
| 時間軸 | ✅ 精準到 frame | ⚠️ Whisper 估的、較粗（多整秒邊界） |
| 幻覺 | ✅ 無 | ❌ 片尾**掰出「點讚訂閱」**（靜音段 Whisper 幻覺） |
| 句數 | 1029（精煉） | 1082（含幻覺尾＋較細切） |
| 取得 | 要找、品質參差 | ✅ **全自動、即時、風格一致** |

### 結論（Route C 價值）

1. **AI 在使用者最在乎的維度（繁中）直接勝出**——既有「人工」版根本是簡體。這正是產品價值主張：一座真的「說繁中」的媒體庫。
2. **AI 的弱點，剛好全是本重規劃 backlog 已要解的差異化層**：
   - 掰專有名詞 / 不認角色名 → backlog #2（per-show 術語表）+ #3（metadata context）。
   - 幻覺尾巴 → 新增 backlog 項：**VAD / 尾段幻覺過濾**（Whisper 通病，與引擎無關）。
3. **唯一結構性差距是畫面文字**（招牌/字卡/地點）——從音訊轉錄先天做不到（需 OCR，超出範圍）。
4. **淨判斷**：Route C + 術語表 + VAD 過濾的產出，會**優於使用者目前能隨手取得的字幕**，且是真繁中。

### 附帶：POC 揪出的 6 個 live-only 正式碼問題（mock 測試皆抓不到）

1. `ai/claude.go:18` 預設模型 `claude-3-5-haiku-latest` → 404（下架）。**正式預設值待更新**。
2. `ai/whisper.go` 無 `language` 參數 → 語言誤判（英文聽寫成中文亂碼）。**已加 `WithWhisperLanguage`**；正式管線應由選定音軌語言帶入。
3. `ai/whisper.go` chunking 判準不一致：`NeedsChunking`（size）vs `SplitAudioChunks`/`getWAVDuration`（duration，且誤解析 ffmpeg WAV header）→ 回傳超大檔 → **413**。**待修**（POC 改用 ffmpeg segment muxer 繞過）。
4. `ai/whisper.go` 無 retry → 單次 transient timeout 即整條陣亡。**待加 retry/backoff**（POC 已加 3 次重試，整集實測救回一次）。
5. Whisper 幻覺尾巴（靜音段無中生有）→ 需 VAD/後處理過濾。
6. 專有名詞跨次飄移（同片名每跑一次譯名不同：隱形戰士↔隱形特務、深海之潮↔洶湧狂潮、透視人↔透明人）→ 術語表鐵證。

POC 腳本：`apps/api/cmd/route-c-poc/main.go`（成本：整集 ≈ $0.48；前 4 分鐘測試 ≈ $0.03）。
