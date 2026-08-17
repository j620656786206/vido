# Story 5.2: ASR 金鑰免重啟熱載 —— ASRProviderHolder、可用性改為每次呼叫判定、文案回收「重啟」謊言

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** `epic-subtitle-pipeline-m3`（M3 第一波，D 群組）· **Risk: 🟠 MED（活體 client 重接線＋四處已上線文案反轉）** · **後端為主（BE 4 task / FE 1 task）**
**Source:** sprint-status `epic-subtitle-pipeline-m3` seed（D:ASR 金鑰免重啟熱載 —— 鏡射 sub-2-1a Claude keyResolver 先例）
**Promotes:** `backlog-asr-runtime-key-resolution`（lane ③,由 sub-2-2c 於 2026-08-06 copy-verification 時立案,雙向）
**Cross-stack split check:** backend tasks = 4, frontend tasks = 1 → 單一 story（未觸發 >3/>3 門檻）

---

## Story

As a NAS owner,
I want the cloud ASR key I save in the settings page to take effect immediately,
so that turning on subtitle generation does not require SSH-ing into the box and restarting the container.

---

## Context — 這個 story 為什麼存在

sub-2-1a 為 **Claude** 修好了兩個 break（金鑰只讀 env／provider 只在開機建一次）。**ASR 那條腿原封不動地保留了 break 2**：

```go
// main.go:508-533 —— 現況
if cfg.HasOpenAIKey() && audioExtractorService.IsAvailable() {
    whisperClient := ai.NewWhisperClient(cfg.GetOpenAIAPIKey(), whisperOpts...)   // ← 開機建一次，讀 env
    transcriptionService = services.NewTranscriptionService(..., whisperClient, ...)
    transcriptionService.SetRunBudgetUSD(...)      // ← 這四個 setter
    transcriptionService.SetGlossaryRepository(...) //    只活在 if 分支裡
    transcriptionService.SetOpenCCConverter(...)    //
    transcriptionService.SetPlacer(...)             //
} else {
    transcriptionService = services.NewTranscriptionService(..., nil, ...)  // ← asr = nil，永遠 nil
}
```

`keyResolver` 已經在 `main.go:551-555` 建好，`KeyOpenAI` / `SecretNameOpenAI` 也已經在 `key_resolver.go:22,43` 定義好，`/settings/keys` 的「雲端 ASR」列早在 sub-2-1b 就能存了——**但存進去的值沒有任何 runtime 消費者**。使用者存了金鑰、看到「已設定」、`source=secret`，然後 `POST /movies/{id}/transcribe` 繼續回 503。

sub-2-2c 當時的處理是**誠實面對**：γ 把 F5 面板、503 訊息、金鑰頁 hint 全部改寫成「請重啟伺服器」，並在同一天立案 `backlog-asr-runtime-key-resolution`，明文寫下 **「Restart-clause coupling: if backlog-asr-runtime-key-resolution ships a holder, THAT story updates the copy.」** 本 story 就是那支 story——所以文案回收是**本 story 的義務,不是附帶**。

### 🚨 authoring 盤點挖到的兩個額外事實

**(1) keyless boot 不只少了 client,還少了四個 pipeline 零件。** 上面那段 `if/else` 裡,`SetRunBudgetUSD` / `SetGlossaryRepository` / `SetOpenCCConverter` / `SetPlacer` 全部只在 **有金鑰的分支**執行。就算本 story 只把 client 換成 holder,keyless boot 起來的 `TranscriptionService` 仍然沒有預算上限、沒有 per-show 詞彙庫、沒有 OpenCC 安全網、沒有原子 placer——熱載金鑰後跑出來的是一條**降級管線**,而且沒有任何訊號。這正是 sub-2-1a Break 2 的同構體（「keyless boot 讓 terminology/translation 永遠是 nil」）,必須在同一次修掉（AC #2）。

**(2) 自架 ASR 目前根本跑不起來,除非你偽造一把 OpenAI 金鑰。** `main.go:510` 的閘門是 `cfg.HasOpenAIKey()`,`whisper.go:164` 又有 `if c.apiKey == "" { return ErrWhisperNotConfigured }`。sub-5-1 才剛讓自架 ASR 的記帳歸零（`IsSelfHostedASRBaseURL`）、`docs/deployment.md:115` 也把自架寫成受支援的部署——但 `docs/development.md:158` 對 `ASR_BASE_URL` 只字未提「你還得設一把用不到的 `OPENAI_API_KEY`」。「ASR 有沒有設定」這個述詞正是本 story 要重寫的東西,帶著一個已知錯誤的述詞去寫 holder 是不可接受的 → **Rule 24 lane ① 吸收為 AC #3**。

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` `ASRProviderHolder`：per-call 解析、fingerprint 快取、Governor 跨重建共用

新增 `apps/api/internal/services/asr_provider_holder.go`,**逐條鏡射 `claude_provider_holder.go`**：

```go
// [@contract-v1] — 消費者：TranscriptionService（AC #2）、main.go 接線。
// 命名說明：backlog 條目寫的是 WhisperProviderHolder，但它持有的 port 是
// ai.ASRProvider（引擎無關，9R-9 起自架引擎就走同一個介面），故以 port 命名。
type ASRProviderHolder struct{ ... }

var _ ai.ASRProvider = (*ASRProviderHolder)(nil)   // 編譯期證明（claude holder 的雙 assertion 先例）

func NewASRProviderHolder(resolver KeyResolver, baseURL, model string, logger *slog.Logger, opts ...ai.WhisperOption) *ASRProviderHolder
func (h *ASRProviderHolder) Get(ctx context.Context) (ai.ASRProvider, error)
func (h *ASRProviderHolder) IsConfigured(ctx context.Context) bool
func (h *ASRProviderHolder) Transcribe(ctx context.Context, audioPath string) (string, error)
func (h *ASRProviderHolder) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error)
```

- **解析點**：`resolver.Get(ctx, services.KeyOpenAI)`——secret > env,順序由 KeyResolver `[@contract-v1]` 固定,本 story 不動。
- **fingerprint**：`key + "|" + baseURL + "|" + model`（`claude_provider_holder.go:80` 的 `key|model` 先例加上 baseURL；baseURL/model 目前是 env-only 常數,但放進 fingerprint 讓未來它們可設定時不必改語意）。
- **opts 只捕捉一次、每次重建重播**——這是 `WithWhisperGovernor(aiGovernor)` 在換金鑰後**仍是同一個 Governor 實例**的機制（`claude_provider_holder.go:52-55` 的理由逐字適用：重建不等於新的預算池／新的節流池）。
- **NFR-S1**：任何 log 不得輸出金鑰或其前綴（照抄 claude holder `:96-97` 的 `key_source` + `model_override` 形狀,加 `self_hosted`）。
- **未設定時**回 `ai.ErrWhisperNotConfigured`（既有 sentinel,`whisper.go:38`）——不要新造錯誤型別,消費者的分類邏輯不變。

### AC #2 — TranscriptionService 無條件建構；可用性改為每次呼叫判定（含四個 setter 的解放）

1. `main.go` 的 `if cfg.HasOpenAIKey() && ...` **整段拆掉**。`keyResolver` 的建立（現在 `:551-555`）**上移到 transcription 區塊之前**（`secretsService` 早已存在,無循環）,`transcriptionService` 一律以 holder 建構。
2. **四個 setter 全部無條件執行**（`SetRunBudgetUSD` / `SetGlossaryRepository` / `SetOpenCCConverter` / `SetPlacer`）——見「額外事實 (1)」。既有的 `SetSubtitleStatusWriter` / `SetSubtitleStateReader` / `SetEpisode*` 四個本來就在 if 外,維持不動。
3. `TranscriptionService.IsAvailable()`（`transcription_service.go:235-236`）加上探針,**簽名不變**：

```go
func (s *TranscriptionService) IsAvailable() bool {
	if s.audioExtractor == nil || !s.audioExtractor.IsAvailable() || s.asr == nil {
		return false
	}
	if probe, ok := s.asr.(interface{ IsConfigured(ctx context.Context) bool }); ok {
		return probe.IsConfigured(context.Background())
	}
	return true   // 素樸 provider（測試 fake、直接注入的 client）行為完全不變
}
```

**這是 `TranslationService.IsConfigured`（`translation_service.go:101-118`）的逐字同構體**——連「素樸 provider 沒有探針就維持 configured」這條相容性條款都一樣。抄它,不要自創。

4. 三個下游可用性消費者因此**零改動**、自動變誠實：
   - `transcription_handler.go:67` 的 503 閘門（FR12 手動單項）
   - `pipelineASRAdapter.Available`（`cmd/api/asr_adapter.go:28`）→ worker-pool 的 `WithASRAvailability` sweep 閘門
   - `RouteCGenerationRunner.IsAvailable`（`generation_batch_runner.go:29`）→ `GenerationBatchProcessor.IsAvailable` → legacy 模式批次 503

   `handlers.TranscriptionServiceInterface`（`transcription_handler.go:23`）的 `IsAvailable() bool` 簽名不動 ⇒ **無 Rule 20 bump**。

5. 驗收語意：**keyless boot → 存金鑰 → 不重啟 → `IsAvailable()` 由 false 翻 true**,且該次 run 拿得到 budget／glossary／OpenCC／placer（不是降級管線）。

### AC #3 — 自架 ASR 免金鑰（Rule 24 lane ① 吸收）

「ASR 已設定」的定義改為：**`ai.IsSelfHostedASRBaseURL(baseURL)` 為真,或 `KeyOpenAI` 解析出非空值**。

- `ASRProviderHolder.IsConfigured` 依此判定；`Get` 在自架且無金鑰時,以空金鑰建 client 並照樣回傳。
- `whisper.go` 兩處配合：
  - `TranscribeWithLanguage:164-166` 的 `if c.apiKey == ""` 提前返回,改為**只在非自架時**返回 `ErrWhisperNotConfigured`（`c.isSelfHosted()` 已存在,`:126-128`）。
  - `:231` 的 `req.Header.Set("Authorization", "Bearer "+c.apiKey)` 改為 **金鑰為空時不設該 header**（送出 `Bearer ` 空值會被部分自架 server 判為格式錯誤；不設比送空好）。hosted 路徑 byte 不變。
- 判定一律走 `ai.IsSelfHostedASRBaseURL`——sub-5-1 CR M1 的裁定（兩個獨立偵測器會在「明示設成官方端點」時分歧）在這裡同樣適用,**不得再長出第三個判定式**。
- 這條讓自架部署第一次真的能在零金鑰下跑 ASR；hosted 部署的行為完全不變（無 baseURL override ⇒ `isSelfHosted()` 為 false ⇒ 空金鑰仍然 `ErrWhisperNotConfigured`）。

### AC #4 — 後端文案回收「重啟」＋文件同步

sub-2-2c/2-2d 上線的重啟句子在本 story 合併後成為**謊言**,同一次改掉：

| 位置 | 現況（節錄） | 改為 |
| --- | --- | --- |
| `transcription_handler.go:74` | 「…儲存雲端 ASR 金鑰,並重啟伺服器。」 | 「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（/settings/keys）儲存雲端 ASR 金鑰,儲存後立即生效。」 |
| `generation_batch_handler.go:89` | 「…；若儲存後仍無法使用,請重啟伺服器。FFmpeg 已內建…」 | 「請至金鑰設定（/settings/keys）確認所需的 AI 金鑰已儲存（翻譯需 Claude 金鑰、語音辨識需雲端 ASR 金鑰）,儲存後立即生效。FFmpeg 已內建於 Docker 映像檔。」 |

- 兩處的 in-code 註解（現在寫著「the WhisperClient is boot-built…the restart clause is the truth」）一併改成本 story 的事實。
- **既有測試從正向斷言翻為負向守衛**：`transcription_handler_test.go:182` 與 `generation_batch_handler_test.go:115` 的 `assert.Contains(t, body, "重啟伺服器")` → `assert.NotContains(...)` ＋ 對新字串的 `Contains`（sub-2-2d L2「symmetric NotContains guards」先例）。
- `docs/deployment.md`：`OPENAI_API_KEY` 現在可由設定頁熱載、自架 ASR 免金鑰；**`ASR_BASE_URL` / `ASR_MODEL` 仍是 env-only 且仍需重啟**——`:123-124` 那句「these are environment variables and a restart is required」要精確化,不能整句刪掉。`docs/development.md:158` 的 ASR 列補上免金鑰事實。
- Rule 17：`docs/deployment.md` 無 zh-TW 雙生（既有債,`backlog-deployment-doc-zh-tw-twin` 已在追）——本 story **不新增**違規,沿用 sub-5-1 的處置與記錄方式。

> 🚧 **不得誤傷的「重啟」字串（全 repo 掃描後的白名單,只有上表兩處該改）。** 一次 `grep 重啟` 的機械替換會弄壞四句**仍然為真**的話：
> - `subtitle_pipeline_handler.go:113,154`（`VIDO_SUBTITLE_PIPELINE_MODE` 是 env-only,仍需重啟）
> - `key_settings_handler.go:119`（`ENCRYPTION_KEY` env,仍需重啟）／`:191`（`CLAUDE_MODEL` env,仍需重啟）
> - `ApiKeysForm.tsx:61` TMDb 列 hint（`backlog-tmdb-runtime-key-resolution` 未解,仍需重啟）＋守著它的 `ApiKeysForm.spec.tsx:136`
> - `ApiKeysForm.tsx:247`／`ApiKeysForm.spec.tsx:223`（`ENCRYPTION_KEY` 未設,仍需重啟）

### AC #5 — 前端文案（FE 唯一 task）

| 檔案 | 改為 |
| --- | --- |
| `ManageSubtitleDialogV2.tsx:391`（F5 未設定面板） | 「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。」 |
| `ApiKeysForm.tsx:68`（雲端 ASR 列 hint） | 「選配：雲端語音辨識。儲存後立即生效,無需重啟伺服器。未設定時仍可使用內建的字幕來源。」 |

- 標題「語音辨識尚未設定」、`data-testid="generation-not-configured"`、`data-testid="go-to-settings"`、按鈕與導向 `/settings/keys` **全部不動**（TestSprite/e2e selector 相依）。
- 「儲存後立即生效,無需重啟伺服器。」是 **Claude 列（`ApiKeysForm.tsx:51`）已上線且已核定的字串**,ASR 列直接複用 ⇒ 不是新設計語彙,不需要新一輪設計核定。
- 但 **`.pen` 的 F5-D-v2（`f6ZxY`）body 仍帶著重啟句**（sub-2-2c γ 核定）→ **設計與實作漂移**。`.pen` 只能由 Sally + Pencil Inline-Agent 流程改（dev agent 不得直接編輯）⇒ authoring 時立 lane ③ `backlog-f5-asr-restart-copy-pen-resync`（雙向）。**裁定：程式碼先講真話**——把一句已成假的指示留在產品裡,會讓使用者去重啟 NAS,傷害大於短暫漂移；`.pen` 由該條目補齊。
- 無新互動面、無版面變動 ⇒ **無視覺基準線重產**（`settings/*` 與該 dialog 皆無 gallery fixture,同 Rule 22 邊界）。

### AC #6 — 契約姿態與「不抽泛型 holder」裁定

- **消費**：`KeyResolver` `[@contract-v1]`（sub-2-1a AC #1）—— 只是新增一個消費者,零改動 ⇒ **記 ack,不 bump**。
- **產生**：`ASRProviderHolder` 標 `[@contract-v1]`（AC #1 的型別與語意面）。
- `ai.ASRProvider`（9R-9,repo 內 port,未 stamp）簽名不動；`handlers.TranscriptionServiceInterface`、`GenerationRunner`、`subtitle.SpeechTranscriber`、D2/D6/`transcription_*` SSE 全部不動 ⇒ **本 story 產生 0 個 bump,無下游 stale-mark 義務**。
- ⚖️ **裁定：本 story 不做泛型 provider-holder 抽取。** `backlog-asr-runtime-key-resolution` 提到 ADR Decision 3 的「等第三個消費者」,但那句把「已立案」當成「已存在」——本 story 合併後**實際存在的 holder 只有兩個**（Claude、ASR）,TMDb 仍只是 backlog。第二個具體實作就抽共用型別是過早泛化（ADR `adr-external-api-integration-standard.md` Decision 3 的 YAGNI 精神）。**抽取的裁定權交給 `backlog-tmdb-runtime-key-resolution`**——它落地時是第三個,屆時再決定要不要 `providerHolder[T]`。本裁定寫進 `asr_provider_holder.go` 檔頭,免得下一位 reviewer 重問。

### AC #7 — 測試

至少：

- **(a) 整合測試（真 `:memory:` DB ＋ 真 AES-256-GCM）**：照 `key_resolution_integration_test.go` 的 `newRealSecretsService` 模式 —— keyless boot 的 `TranscriptionService.IsAvailable()` 為 false → 經 `KeySettingsService.Save` 存入 `openai` → **不重建任何物件** → `IsAvailable()` 翻 true。這是本 story 的頭條場景,必須是真 secrets 服務不是 fake（Rule 15／bugfix-20-1 先例）。
- **(b) holder 單元測試**：fingerprint 命中不重建（同 key 兩次 `Get` 回同一指標,`assert.Same`）／換 key 重建／**Governor 跨重建仍是同一實例**（claude holder 測試模板）／未設定回 `ErrWhisperNotConfigured`／log 不含金鑰。
- **(c) AC #3 三態**：自架＋無金鑰 ⇒ configured 且能送出請求（httptest server 斷言 **無 `Authorization` header**）／hosted＋無金鑰 ⇒ 仍 `ErrWhisperNotConfigured`／自架＋有金鑰 ⇒ header 照送。`ai.IsSelfHostedASRBaseURL` 單一判定式的 agreement 斷言沿用 sub-5-1 的 `TestSelfHostedJudgment_SingleSource`。
- **(d) 可用性探針**：素樸 fake provider（無 `IsConfigured`）維持 configured ＝ 既有測試全數不動的證明；holder 注入時隨解析結果翻轉。
- **(e) main.go 接線的可證性**：四個 setter 無條件執行 —— 以 keyless 建構路徑斷言 `runBudgetUSD`／placer／opencc／glossary 皆已注入（若欄位私有,以行為斷言：keyless 建構後的 service 走完一次 run 會經過 placer fake）。
- **(f) 文案負向守衛**：AC #4 的兩處（`transcription_handler_test.go:182`、`generation_batch_handler_test.go:115`）翻為 `NotContains("重啟")` ＋ 新字串 `Contains`；FE **`ManageSubtitleDialogV2.spec.tsx:301` 逐字斷言舊字串,必須同步翻面**；`ApiKeysForm.spec.tsx` 新增雲端 ASR 列的「立即生效」斷言,而 `:136`（TMDb 列仍需重啟）與 `:223`（ENCRYPTION_KEY）**保持不動**——它們是白名單的守衛。
- 全回歸閘門照常：`go test ./...`、`pnpm nx test web`、`pnpm run lint:all`、`format:check`。

---

## Tasks / Subtasks

- [x] **Task 1 — `ASRProviderHolder`（AC: #1, #6）** 🔴 BE
  - [x] `services/asr_provider_holder.go`：resolver→fingerprint→快取重建；`ai.ASRProvider` 編譯期 assertion；檔頭寫入「不抽泛型」裁定
  - [x] `IsConfigured` / `Get` / `Transcribe` / `TranscribeWithLanguage` 四個方法；NFR-S1 log 形狀

- [x] **Task 2 — main.go 接線 ＋ 可用性探針（AC: #2）** 🔴 BE
  - [x] `keyResolver` 建立上移；拆掉 `if cfg.HasOpenAIKey()`；四個 setter 移出 if
  - [x] `TranscriptionService.IsAvailable()` 加 `IsConfigured(ctx)` 探針（抄 `translation_service.go:101-118`）
  - [x] 確認三個下游消費者零改動（handler 503／sweep 閘門／legacy 批次）

- [x] **Task 3 — 自架免金鑰（AC: #3）** 🔴 BE
  - [x] `whisper.go`：`ErrWhisperNotConfigured` 只在非自架時返回；空金鑰不設 `Authorization` header
  - [x] holder 的 configured 判定納入 `ai.IsSelfHostedASRBaseURL`（單一判定式,不新增偵測器）

- [x] **Task 4 — 後端文案與文件（AC: #4）** 🔴 BE
  - [x] 兩處 503 suggestion ＋ in-code 註解改寫；兩處既有測試翻為負向守衛
  - [x] `docs/deployment.md`（精確化 env-only/重啟句,不整段刪）＋ `docs/development.md:158`

- [x] **Task 5 — 前端文案（AC: #5）** 🟡 FE
  - [x] `ManageSubtitleDialogV2.tsx:391` ＋ `ApiKeysForm.tsx:68`；testid／按鈕／導向全部不動；**`ManageSubtitleDialogV2.spec.tsx:301` 同步翻面**（AC #4 白名單其餘字串一律不動）
  - [x] 立案 `backlog-f5-asr-restart-copy-pen-resync`（雙向）——`.pen` f6ZxY 由 Sally 補

- [x] **Task 6 — 測試與回歸（AC: #7）**
  - [x] 整合測試（真 secrets）＋ holder 單元 ＋ 三態 ＋ 探針 ＋ 接線 ＋ 文案守衛
  - [x] 契約清點（KeyResolver ack、0 bump）＋ 全回歸閘門

（後端 task 4 個、前端 1 個 —— 未觸發跨端拆分門檻。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| holder 骨架（fingerprint／opts 重播／Governor 共用／log 形狀） | `services/claude_provider_holder.go`（**逐條鏡射,勿自創**） |
| 金鑰解析 | `services.KeyResolver` ＋ `KeyOpenAI` / `SecretNameOpenAI`（`key_resolver.go:22,43`）——已存在,零改動 |
| 「provider 可能是 lazy holder」探針 | `translation_service.go:101-118`（含素樸 provider 相容條款） |
| per-call 可用性先例 | `nfo_localizer_service.go:50-52`（sub-2-1a CR M1 修出來的形狀） |
| 自架判定 | `ai.IsSelfHostedASRBaseURL`（`whisper.go:117-119`,sub-5-1 CR M1 唯一判定式） |
| Whisper option 機制 | `WithWhisperBaseURL` / `WithWhisperModel` / `WithWhisperGovernor`（`whisper.go:65-107`） |
| 真 secrets 整合測試模式 | `services/key_resolution_integration_test.go`（`newRealSecretsService`,migration-005 schema 逐字） |
| 文案負向守衛先例 | sub-2-2d CR L2（symmetric `NotContains` guards） |

### 關鍵決策（authoring 已裁）

- **命名 `ASRProviderHolder` 而非 backlog 寫的 `WhisperProviderHolder`**：持有的 port 是引擎無關的 `ai.ASRProvider`（9R-9 起自架引擎共用）。不是漂移,是修正。
- **不抽泛型 holder**（AC #6）：本 story 後只有兩個具體 holder；抽取裁定權交給 `backlog-tmdb-runtime-key-resolution`。
- **自架免金鑰吸收進本 story**（lane ①）而非另立 backlog：holder 的存在理由就是回答「ASR 有沒有設定」,帶著已知錯誤的述詞交付等於明知故犯。
- **程式碼文案先講真話,`.pen` 由 lane ③ 補**：留一句會讓使用者去重啟 NAS 的假指示,傷害大於短暫的設計-實作漂移；且新字串是 Claude 列已核定語彙的複用,非新設計。
- **`ASR_BASE_URL` / `ASR_MODEL` 維持 env-only**：它們不是秘密、不在 `/settings/keys` 的鍵集合裡,把 endpoint 設定搬進設定頁是另一個產品面（未立案,本 story 不擴張）。fingerprint 已預留欄位。
- **`IsAvailable()` 簽名不變**：改成吃 ctx 會波及 handler 介面與四個消費者,換來的只是把 `context.Background()` 往上推一層——`subtitleCapabilityGate`（`main.go:591`）與 `TranslationService.IsConfigured` 都是同樣的取捨,保持一致。

### seam 資料層觸及（retro-m2-AI3 慣例）

- `ASRProviderHolder`：純記憶體 ＋ 透過 KeyResolver 讀 `secrets` 表（既有讀路徑,零新表、零 migration）。
- `TranscriptionService`：既有觸及不變（`movies`／`episodes`／`show_glossary`／SSE）；本 story 只改「asr 從哪來」與「可用性怎麼問」。
- `WhisperClient`：外部 HTTP；AC #3 只改 header 與前置檢查,不動計帳（sub-5-1 的 `RecordASRWithRate` 路徑 byte 不變）。
- **零 migration、零新 Rule 7 error code**（prefix 數維持 16）。

### 已知限制（記錄,不在本 story 解）

- `/settings/keys/test` 仍只探 Claude；雲端 ASR 列的「測試」維持 disabled ＋ 既有理由字串 → lane ③ `backlog-asr-key-test-probe`（真實 ASR 探測要嘛花錢、要嘛只能對自架打 `/v1/models`,值得獨立裁定）。
- TMDb 仍是重啟才生效（`backlog-tmdb-runtime-key-resolution`,`ApiKeysForm.tsx:61` 的 hint 維持不動——本 story 只改 ASR 列）。
- `.pen` F5-D-v2 `f6ZxY` 在 lane ③ 補齊前,設計與實作在該句話上不一致（已立案、雙向）。
- 熱載只覆蓋**金鑰**；`ASR_BASE_URL`／`ASR_MODEL` 換值仍需重啟。

### 契約姿態（Rule 20）

- 消費 `KeyResolver` `[@contract-v1]`（sub-2-1a AC #1）：新增消費者,零改動 ⇒ ack ＋ Change Log,不 bump。
- 產生 `ASRProviderHolder` `[@contract-v1]`。
- `ai.ASRProvider` / `handlers.TranscriptionServiceInterface` / `GenerationRunner` / `subtitle.SpeechTranscriber` / D2 / D6 / `transcription_*` SSE：全部不動 ⇒ 0 bump ⇒ 無 stale-mark 義務。

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.`（FE 僅兩個靜態字串。）

### References

- [Source: sprint-status `epic-subtitle-pipeline-m3` seed] — D 群組:ASR 金鑰免重啟熱載,鏡射 sub-2-1a
- [Source: sprint-status `backlog-asr-runtime-key-resolution`] — 立案事實＋「holder 落地時由該 story 更新文案」的耦合條款
- [Source: `apps/api/cmd/api/main.go:508-533,551-557,591`] — boot-gated 建構、四個 setter、resolver 位置、capability gate 先例
- [Source: `apps/api/internal/services/claude_provider_holder.go`] — 逐條鏡射對象
- [Source: `apps/api/internal/services/key_resolver.go:12-23,40-44,121-160`] — KeyOpenAI／SecretNameOpenAI／解析順序
- [Source: `apps/api/internal/services/translation_service.go:101-118`] — lazy-holder 探針（逐字同構）
- [Source: `apps/api/internal/services/transcription_service.go:104-155,235-236`] — asr 欄位、建構子、IsAvailable
- [Source: `apps/api/internal/ai/whisper.go:38,117-128,163-166,231`] — sentinel、自架判定、空金鑰前置檢查、Authorization header
- [Source: `apps/api/cmd/api/asr_adapter.go:25-28`] — sweep 側可用性探針
- [Source: `apps/api/internal/services/generation_batch_runner.go:28-31`] — legacy 批次可用性
- [Source: `apps/api/internal/handlers/transcription_handler.go:23,67-75`] — 介面簽名與 503 文案
- [Source: `apps/api/internal/handlers/generation_batch_handler.go:79-91`] — 批次 503 文案
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:374-400`] — F5 面板
- [Source: `apps/web/src/components/settings/ApiKeysForm.tsx:46-71`] — 三列 hint（Claude 列＝已核定的「立即生效」語彙來源）
- [Source: `apps/api/internal/services/key_resolution_integration_test.go:1-60`] — 真 secrets 整合測試模式
- [Source: `docs/deployment.md:112-124`, `docs/development.md:158`] — 待同步的部署事實
- [Source: `project-context.md`] — Rule 3/7/11/13/14/17/19/20/24/27

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- Full API suite: `go test ./... -count=1` — 34 packages ok, 0 failures.
- Full web suite: `pnpm nx test web -- --run` — 233 files / 2620 tests green.
- `go build ./...` + `go vet ./...` clean; `gofmt -l` clean on all touched files.
- `pnpm run lint:all` — 0 errors, 119 pre-existing warnings (retro-11-AI1b batch; **0 on files this story touched** — scoped eslint over the 4 touched `apps/web` files returned empty). `prettier --check` green after formatting `docs/development.md` (the table-alignment reflow the new cell widths forced).
- Guard falsification: `internal/asr_hot_reload_test.go` was temporarily re-pointed at a pattern that DOES occur in `main.go` (and the construction-count expectation flipped to 99) — both tests went red with 6 findings, then were restored green. The guards detect, they do not merely pass.

### Completion Notes List

- **🔗 AC Drift: FOUND — sanctioned, and pre-authorised by the tracking entry.** sub-2-2c (γ ratified string table) and **sub-2-2d AC #1/#3** shipped the restart clause as the deliberate, *verified-true* copy — 「請至金鑰設定儲存，並重啟伺服器後再試。」 and the two 503 suggestions. This story **reverses that observable behaviour** because the fact underneath it changed. This is not an unnoticed drift: `backlog-asr-runtime-key-resolution` recorded the coupling in advance — *"if backlog-asr-runtime-key-resolution ships a holder, THAT story updates the copy"* — and this story is that story. Grep sweep: `重啟伺服器` / `boot-built` / `restart clause` / `HasOpenAIKey` across `_bmad-output/implementation-artifacts/*.md` → hits in `sub-1-7a`, `sub-2-1b`, `sub-2-2c`, `sub-2-2d`, `9-2a`; every one read. Only sub-2-2c/2-2d assert the ASR restart clause (DRIFT, handled here); sub-1-7a and sub-2-1b concern the **TMDb** row and the ENCRYPTION_KEY notice, which stay true and are **untouched** (REUSE); 9-2a is the original boot-built wiring this story replaces (REUSE of its pipeline semantics, only the construction site moves).
- **📎 Contract Stamps: FOUND (3 stamped ACs across 2 files — this story stamps `[@contract-v1]` on AC #1 `ASRProviderHolder`; upstream `[@contract-v1]` Story sub-2-1a AC #1 `KeyResolver` referenced).** **confirmed against `[@contract-v1]` (Story sub-2-1a AC #1)** — the `KeyResolver` interface, the FIXED secret > env resolution order, and `KeyOpenAI`/`SecretNameOpenAI` are consumed **unchanged**: this story adds a consumer and edits nothing on the contract surface, so **0 bumps produced ⇒ no downstream stale-mark obligation**. sub-2-1a AC #2 (`ClaudeProviderHolder`, also `[@contract-v1]`) is **mirrored, not consumed** — `ASRProviderHolder` is a sibling implementation of the same shape and touches no Claude code. Unstamped surfaces verified untouched: `ai.ASRProvider`, `handlers.TranscriptionServiceInterface.IsAvailable() bool`, `GenerationRunner`, `subtitle.SpeechTranscriber`, D2/D6/`transcription_*` SSE.
- **🎭 A11y Pre-Flight: PASS** (2 components checked — `ApiKeysForm`, `ManageSubtitleDialogV2`; **0** jsx-a11y warnings on touched files, 0 introduced). Both changes are static text inside existing elements: no new interactive surface, no image, no modal/focus change, no async-revealed content, no lazy-load request-count claim. The F5 panel's structure (`data-testid`, the 前往設定 button, the `/settings/keys` navigation) is byte-identical.
- **🎨 UX Verification: ONE KNOWN DRIFT, ruled at authoring and tracked** — see the dedicated note below.
- **AC #1** — `services/asr_provider_holder.go`: `ASRProviderHolder` stamped `[@contract-v1]`, a per-line mirror of `ClaudeProviderHolder`. Fingerprint is `key|baseURL|model` (the endpoint and model are part of "which engine is in force", not decoration — pinned by `TestASRProviderHolder_FingerprintCoversBaseURLAndModel`). `opts` captured once and replayed on rebuild, which is what keeps the **shared 9R-11 Governor** alive across a key change (`TestASRProviderHolder_GovernorSurvivesRebuild`, asserting `assert.Same` on the instance). Unconfigured returns the **existing** `ai.ErrWhisperNotConfigured` sentinel — no new error type, no new Rule 7 code. Logs carry `key_source` / `self_hosted` / `base_url_override` / `model_override` and never the key (NFR-S1). Naming rationale (port, not engine) and the ⚖️ **no-generic-extraction** ruling are both written into the file header so a reviewer does not re-litigate them.
- **AC #2** — `main.go`: the `if cfg.HasOpenAIKey() && audioExtractorService.IsAvailable()` block is **gone**; `keyResolver`/`claudeHolder` moved up ahead of the transcription wiring, `asrHolder` built from `cfg.ASRBaseURL`/`cfg.ASRModel` + the shared `aiGovernor`, and the service constructed **once**. The four pipeline setters (`SetRunBudgetUSD` / `SetGlossaryRepository` / `SetOpenCCConverter` / `SetPlacer`) are no longer **key-gated** — they run on every boot regardless of key state (precision note added at CR L3: `SetOpenCCConverter` keeps its pre-existing `if subtitleConverter != nil` guard, which is converter-existence, not key state — a nil concrete pointer through the interface would defeat the service's own nil-check). `TranscriptionService.IsAvailable()` gained the `IsConfigured(ctx)` probe, a verbatim isomorph of `TranslationService.IsConfigured` **including** its compatibility clause (a plain provider without the probe stays available — which is why every pre-existing transcription test is untouched and still green). The three downstream availability consumers (`transcription_handler.go` 503, `pipelineASRAdapter.Available`, `RouteCGenerationRunner.IsAvailable`) needed **zero edits** and became truthful automatically; `IsAvailable() bool` keeps its signature ⇒ no interface change, no Rule 20 bump.
- **AC #2 (structural guard, beyond the literal AC)** — new `internal/asr_hot_reload_test.go` in the repo-invariant package (`internal/cost_consent_test.go` precedent): `.HasOpenAIKey(` must have **no production caller**, and `main.go` must construct `TranscriptionService` **exactly once** with **exactly one** call to each of the four setters. This bug shipped twice (Claude in sub-2-1a, ASR here) and the second half of it — the else-branch that dropped the setters — was invisible to every existing test. The invariant is now structural rather than a comment. The scanner skips comment lines so the explanatory references to the removed gate do not self-trip.
- **AC #3** — self-hosted ASR runs with **no key at all**. `whisper.go` returns `ErrWhisperNotConfigured` only when the key is empty **and** `!c.isSelfHosted()`, and omits the `Authorization` header entirely rather than sending a bare `Bearer ` (a malformed credential some engines reject). `ASRProviderHolder.IsConfigured` = `IsSelfHostedASRBaseURL(baseURL) || resolver.Has(KeyOpenAI)`. Every self-hosted judgment routes through the **one** predicate `ai.IsSelfHostedASRBaseURL` (sub-5-1 CR M1) — the trap value (`ASR_BASE_URL` explicitly set to the official endpoint) is re-pinned on the availability side by `TestASRProviderHolder_ExplicitOfficialURLIsNotSelfHosted`, because a second "non-empty ⇒ self-hosted" detector here would hand out a keyless client that 401s on every call. The paid hosted path is unchanged in every respect, including sub-5-1's metering.
- **AC #4** — both 503 suggestions now end 「儲存後立即生效」; their in-code comments state the new fact instead of the old one. The two existing `assert.Contains(t, body, "重啟伺服器")` assertions were **inverted** into `NotContains("重啟")` + a positive assertion on the replacement (sub-2-2d CR L2 symmetric-guard precedent), so the lie cannot come back silently. `docs/deployment.md` gained two precise bullets (which keys hot-reload — with TMDb named as the exception — and that self-hosted ASR needs no key) and the blanket "these are environment variables and a restart is required" sentence was **narrowed** rather than deleted: `ASR_BASE_URL`, `ASR_MODEL`, `AI_RUN_BUDGET_USD` and `VIDO_SUBTITLE_PIPELINE_MODE` genuinely still need one. `docs/development.md`'s ASR rows say the same. **Rule 17:** `docs/deployment.md` still has no zh-TW twin — pre-existing debt tracked by `backlog-deployment-doc-zh-tw-twin`; this story does not newly violate it (the sub-4-2 / sub-5-1 handling).
- **AC #4 (scope fence honoured)** — the story's 「不得誤傷」 whitelist was checked line by line and **every** still-true restart string is untouched: `subtitle_pipeline_handler.go:113,154` (pipeline-mode env), `key_settings_handler.go` (ENCRYPTION_KEY, CLAUDE_MODEL), the `ApiKeysForm` TMDb row + its spec guard, and the ENCRYPTION_KEY notice + its spec guard. A blanket `grep 重啟` replacement would have broken all four.
- **AC #5** — F5 panel and the 雲端 ASR row updated; title, `data-testid="generation-not-configured"`, `data-testid="go-to-settings"`, the button and the `/settings/keys` navigation are byte-identical (TestSprite/e2e selectors depend on them). The ASR row's hint also now names the self-hosted case, so an operator who configured `ASR_BASE_URL` stops hunting for a key to paste. **Test-authoring catch:** the first negative guard asserted `not.toHaveTextContent('需重啟')` and failed — the new copy legitimately contains 「無**需重啟**伺服器」. The guard now pins the TMDb row's *affirmative* string instead, which is the thing that must not leak over; the comment records why the naive form is wrong.
- **AC #6** — 0 bumps produced, 1 ack recorded (above). ⚖️ The **no-generic-holder** ruling stands and is now documented at the code: with this file there are exactly **two** concrete holders, not three (TMDb is a backlog entry, not an implementation), so extraction would be premature generalisation per ADR `adr-external-api-integration-standard` Decision 3. The decision is delegated to `backlog-tmdb-runtime-key-resolution`, whose sprint-status entry already carries the hand-off note.
- **AC #7** — 25 new tests (count corrected at CR — the original note said 22/holder-14; the honest tally is holder 16, integration 4, whisper 3, guards 2): `asr_provider_holder_test.go` ×16 (fingerprint identity, rebuild-on-key-change, Governor survival, the NFR-S1 log-capture test added at CR H1, self-hosted trio incl. the official-URL trap, per-call delegation, constructor plumbing, and the four `TranscriptionService` availability cases including the plain-provider compatibility clause), `asr_key_resolution_integration_test.go` ×4 (real `:memory:` DB + real AES-256-GCM: save→un-gate with nothing rebuilt, secret-outranks-env, clear→re-gate, self-hosted available with the key row still honestly reporting `none`), `whisper_test.go` ×3 (keyless self-hosted transcribes with **no** `Authorization` header; hosted keyless still errors; self-hosted **with** a key still sends it), `asr_hot_reload_test.go` ×2 structural guards. RED was observed before each implementation step (compile failure for the holder; `ErrWhisperNotConfigured` for the self-hosted transcribe).
- **Additional hardening not in the literal AC:** `ai.WhisperClient` gained `Governor()` and `BaseURL()` accessors (the `ClaudeProvider.Governor()` precedent, with the same doc rationale) so the rebuild assertions read real state instead of trusting wiring; and `config.HasOpenAIKey` — orphaned by this story — carries a ⚠️ doc warning that it is **not** the answer to "is ASR configured", so the next reader does not re-introduce the gate. The accessor itself is kept, matching how sub-2-1a left `HasClaudeKey` in place.
- **Pre-existing failures: NONE encountered.** Both suites were green before and after; nothing was skipped, filed or waived.

### 🎨 UX Verification — one known drift, ruled and tracked

| Area | Design spec (`.pen` F5-D-v2 `f6ZxY`) | Implementation | Match? | Fix |
| --- | --- | --- | --- | --- |
| Panel title | 語音辨識尚未設定 | 語音辨識尚未設定 | ✅ | — |
| Panel body | 「…請至金鑰設定儲存，**並重啟伺服器後再試**。」 | 「…請至金鑰設定**儲存後即可使用**。」 | ❌ | **Not fixed here — by ruling** |
| CTA button / target | 前往設定 → `/settings/keys` | identical | ✅ | — |
| Tint / icon / layout | `$warning-tint`, Settings icon | identical | ✅ | — |
| 雲端 ASR row hint | (no `.pen` surface — settings pages have none) | 「儲存後立即生效，無需重啟伺服器。」 | n/a | reuses the ratified Claude-row wording |

The single mismatch is **deliberate and was ruled at authoring**: the `.pen` body was ratified by sub-2-2c when the restart requirement was *true*, and this story is what makes it false. `.pen` edits go through Sally + the Pencil Inline-Agent flow and are not a dev agent's to make, so shipping the lie in code until a design round completes was rejected — it would send users to reboot a NAS for nothing. Tracked bidirectionally as **`backlog-f5-asr-restart-copy-pen-resync`**. No `.pen` file was opened or modified. No visual baselines apply (`settings/*` and this dialog have no gallery fixtures — the same Rule 22 boundary as 1-7b AC #5), so no screenshot regeneration was required or performed.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time（2026-08-13）：
    - **① expand-scope-in-place** — 自架 ASR 部署在無 `OPENAI_API_KEY` 時完全無法啟用（`main.go:510` 閘門 ＋ `whisper.go:164` 前置檢查）,而 sub-5-1 才剛把自架記帳歸零、文件也把自架寫成受支援 → 吸收為 **AC #3**（本 story 重寫的正是「ASR 有沒有設定」這個述詞）。
    - **① expand-scope-in-place** — keyless boot 的 `TranscriptionService` 連 `SetRunBudgetUSD`／`SetGlossaryRepository`／`SetOpenCCConverter`／`SetPlacer` 都沒跑（四個 setter 困在 `if cfg.HasOpenAIKey()` 分支內）,熱載金鑰後會得到一條無聲降級的管線 → 吸收為 **AC #2 第 2 點**。
    - **③ backlog-with-carry-forward-link** — `backlog-f5-asr-restart-copy-pen-resync`：`.pen` F5-D-v2（`f6ZxY`）body 仍是 sub-2-2c γ 核定的重啟句,本 story 只改程式碼字串（`.pen` 需 Sally ＋ Inline-Agent 流程）。非阻塞。
    - **③ backlog-with-carry-forward-link** — `backlog-asr-key-test-probe`：`POST /settings/keys/test` 仍 claude-only,雲端 ASR 列的「測試」維持 disabled；真實 ASR 探測的成本／自架-only `/v1/models` 取捨需獨立裁定。非阻塞。
  - **實作階段(2026-08-13)：無新增發現。** 兩條 lane ① 缺口皆已於 AC #3／AC #2 內交付並測試釘住；兩條 lane ③ 條目在 authoring 時即已立案(雙向連結已驗)。`config.HasOpenAIKey` 因本 story 而失去生產消費者 —— **不算 lane ③**:它仍是合法的 env accessor、仍有自己的測試,且已在 in-code 加上「這不是 ASR 是否設定的答案」警語(比照 sub-2-1a 保留 `HasClaudeKey` 的處置),沒有留下未追蹤的工作。

### File List

**Backend — new**

- `apps/api/internal/services/asr_provider_holder.go` — AC #1 `[@contract-v1]` `ASRProviderHolder`: `key|baseURL|model` fingerprint cache (Rule 14), options replayed on rebuild so the shared Governor survives, `IsConfigured` folding in the self-hosted predicate, per-call delegation of both `ai.ASRProvider` methods, compile-time `var _` assertion, header ruling on no-generic-extraction
- `apps/api/internal/services/asr_provider_holder_test.go` — 14 tests (fingerprint identity / rebuild / Governor survival / self-hosted trio incl. the official-URL trap / delegation / 4 availability cases)
- `apps/api/internal/services/asr_key_resolution_integration_test.go` — AC #7(a) 4 tests on a real `:memory:` DB + real AES-256-GCM secrets service
- `apps/api/internal/asr_hot_reload_test.go` — repo-invariant guards: no production caller of `.HasOpenAIKey(`; exactly one `TranscriptionService` construction site with exactly one call to each of the four pipeline setters

**Backend — modified**

- `apps/api/cmd/api/main.go` — key resolver + Claude holder moved ahead of transcription wiring; `if cfg.HasOpenAIKey()` gate removed; ASR holder wired with `cfg.ASRBaseURL`/`cfg.ASRModel` + shared `aiGovernor`; the four pipeline setters no longer key-gated; boot log reports resolved availability
- `apps/api/internal/subtitle/worker_pool.go` — (CR M2) `sweepEligible` takes a per-sweep `asrOK` snapshot instead of probing per item — the probe became a secrets read + AES decrypt once keys went hot-reloadable; also makes `WithASRAvailability`'s "checked per sweep, not per item" doc comment true
- `apps/api/internal/services/transcription_service.go` — `IsAvailable()` gains the `IsConfigured(ctx)` probe (signature unchanged; plain-provider compatibility clause preserved)
- `apps/api/internal/ai/whisper.go` — AC #3: `ErrWhisperNotConfigured` only on the hosted path; empty key omits the `Authorization` header; `Governor()` + `BaseURL()` accessors for rebuild assertions
- `apps/api/internal/ai/whisper_test.go` — 3 AC #3 tests + `authProbeServer`/`wavFixture` helpers
- `apps/api/internal/config/api_keys.go` — ⚠️ doc warning on `HasOpenAIKey` (not the ASR-configured predicate any more)
- `apps/api/internal/handlers/transcription_handler.go` — AC #4 503 suggestion + comment
- `apps/api/internal/handlers/transcription_handler_test.go` — restart assertion inverted to a negative guard + positive replacement
- `apps/api/internal/handlers/generation_batch_handler.go` — AC #4 503 suggestion + comment (mode-neutral framing preserved)
- `apps/api/internal/handlers/generation_batch_handler_test.go` — same inversion

**Frontend**

- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` — F5 panel body (restart clause retired); testids/CTA/navigation untouched
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx` — copy assertion updated + `not.toHaveTextContent('重啟')` guard
- `apps/web/src/components/settings/ApiKeysForm.tsx` — 雲端 ASR row hint (immediate effect + self-hosted note)
- `apps/web/src/components/settings/ApiKeysForm.spec.tsx` — new ASR-row test; TMDb/ENCRYPTION_KEY guards deliberately untouched

**Docs / tracking**

- `docs/deployment.md` — hot-reloadable keys (TMDb named as the exception) + self-hosted-needs-no-key; the blanket restart sentence narrowed, not deleted
- `docs/development.md` — `OPENAI_API_KEY` / `ASR_BASE_URL` rows (Prettier reflowed the table)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — sub-5-2 ready-for-dev → in-progress → review
- `_bmad-output/implementation-artifacts/sub-2-2c-f5-asr-copy-design.md` — (AC drift reference — see Completion Notes; file NOT modified)
- `_bmad-output/implementation-artifacts/sub-2-2d-f5-cta-code-strings.md` — (AC drift reference — see Completion Notes; file NOT modified)

---

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-13 | create-story：M3 D 群組建檔。PROMOTES `backlog-asr-runtime-key-resolution`。authoring 盤點吸收 2 個 lane ① 缺口（自架免金鑰、keyless boot 的四個 setter）＋ 立 2 條 lane ③（`.pen` 文案 resync、ASR 金鑰測試探針）。 |
| 2026-08-13 | Task 1 (AC #1, #6)：`ASRProviderHolder` `[@contract-v1]` —— `key\|baseURL\|model` fingerprint 快取、opts 重播讓共用 Governor 跨重建存活、未設定回既有 `ErrWhisperNotConfigured`、NFR-S1 log 形狀；不抽泛型 holder 的裁定寫進檔頭。confirmed against `[@contract-v1]` (Story sub-2-1a AC #1) —— KeyResolver 零改動,0 bump。 |
| 2026-08-13 | Task 2 (AC #2)：main.go 拆掉 `if cfg.HasOpenAIKey()`,resolver/holder 上移,transcription service 單一建構點且四個 pipeline setter 無條件執行（原本困在 key 分支內 ⇒ keyless boot 只能復原成降級管線）；`IsAvailable()` 加 `IsConfigured(ctx)` 探針（簽名不變 ⇒ handler 介面 0 bump,三個下游閘門零改動自動變誠實）。新增 repo-invariant 守衛把「不得再有開機期金鑰閘門」結構化。 |
| 2026-08-13 | Task 3 (AC #3)：自架 ASR 免金鑰 —— `ErrWhisperNotConfigured` 只在 hosted 路徑返回,空金鑰不送 `Authorization` header；configured 判定併入 `ai.IsSelfHostedASRBaseURL`（sub-5-1 CR M1 唯一判定式,官方 URL 明示設定的陷阱值在可用性側再釘一次）。hosted 路徑行為與計帳完全不變。 |
| 2026-08-13 | Task 4 (AC #4)：兩處 503 suggestion 的「重啟」句回收為「儲存後立即生效」,既有正向斷言翻為 `NotContains` 負向守衛；`deployment.md`／`development.md` 同步（blanket 重啟句收窄而非刪除 —— `ASR_BASE_URL`/`ASR_MODEL`/`AI_RUN_BUDGET_USD`/`VIDO_SUBTITLE_PIPELINE_MODE` 仍需重啟）。白名單四處仍為真的重啟字串逐一確認未動。 |
| 2026-08-13 | Task 5 (AC #5)：F5 面板與雲端 ASR 列文案更新（testid／CTA／導向 byte 不變）；`.pen` f6ZxY 的漂移依 authoring 裁定不在此修,由 `backlog-f5-asr-restart-copy-pen-resync` 追蹤。 |
| 2026-08-13 | Task 6 (AC #7)：新測試（holder／真 secrets 整合 4／whisper 三態 3／結構守衛 2，其中守衛經 falsification 驗證會紅）；全回歸綠 —— api 34 packages、web 233 files/2620 tests、lint 0 errors、gofmt/prettier clean。Status in-progress → review。（測試計數於 CR 修正：實為 25 非 22。） |
| 2026-08-13 | Senior Developer Review (Fable 5 adversarial, 換模型慣例 — implementation by Opus 5) — 1H/2M/3L, all adjudicated and fixed in-session. H1: AC #7(b) 明列的 NFR-S1「log 不含金鑰」測試不存在 → 補 log-capture test（gemini_test MissingUsageMetadata 模式，雙向斷言 key_source 在、金鑰與其前綴不在）。M1: ApiKeysForm ASR hint 偏離 AC #5 核定字串且新增句把自架 ASR 與降級 fallback 混為一談（讀起來像「自架 ⇒ ASR 不可用」）→ 回歸核定字串，自架免金鑰事實留在 docs（in-code 註解記錄裁定）。M2: sweepEligible 逐列呼叫 asrRecoverable()，該鏈自本 story 起是 secrets DB 讀＋AES 解密 per item（原為 nil-check；EnqueueMissing 目前零生產呼叫者故為 latent，但 WithASRAvailability 的 doc 本來就宣告 per-sweep）→ 改為每 sweep 快照一次，doc 與實作對齊。L1: 測試計數不實（22→25、holder 14→16）→ 修正。L2: FingerprintCoversBaseURLAndModel 測試名不符實（兩個獨立 holder 必然不同 client，無法否證 fingerprint 主張）→ 更名 BaseURLAndModelReachTheClient ＋ 誠實註解（baseURL/model 維度在單一 holder 生命期內不可變，屬 future-proofing）。L3:「四個 setter 無條件執行」措辭精確化（OpenCC 仍在 converter-existence guard 內，非 key 閘門）。修後全綠：api 34 pkg、web 233/2620、gofmt/prettier clean。Status review → done。 |

## Senior Developer Review (AI)

**Reviewer model:** Claude Fable 5（換模型 adversarial CR 慣例 — implementation by Opus 5）· **Date:** 2026-08-13 · **Outcome:** Changes Requested → all findings adjudicated and fixed in-session → **Approve (done)**

**Mandatory checks:** 🔒 Rule 7 Wire Format: PASS（in-review Go 檔 0 個新 error-code 常數；兩處 503 重用已註冊的 `TRANSCRIPTION_DISABLED`，prefix 數維持 16）· 🔒 Rule 20 Contract Bump: N/A（0 bump；本 story 新 stamp `[@contract-v1]` ×1 ＋ 消費側 ack ×1，ack 行 `confirmed against [@contract-v1] (Story sub-2-1a AC #1)` 於 Completion Notes 驗證存在）· 🔒 Rule 25 Mega-line: N/A（project-context.md 未觸及）· Git vs File List: 0 discrepancies（21 檔全數對帳；CR 修復另 +1 檔 worker_pool.go，已補入 File List）。

### Findings & resolutions（1H / 2M / 3L）

- [x] **[H1] AC #7(b) 明列的 NFR-S1「log 不含金鑰」測試不存在。** Task 6 標 [x] 且 Completion Notes 宣稱 AC #7 交付，但 `asr_provider_holder_test.go` 零 log 斷言——NFR-S1 只靠人工目視。FIXED：`TestASRProviderHolder_RebuildLogNeverContainsTheKey`（gemini_test 的 log-capture 模式）雙向釘住：rebuild 行有出、`key_source` 在、金鑰與其 6 字前綴皆不在。
- [x] **[M1] ApiKeysForm ASR hint 偏離 AC #5 核定字串，且新增子句語意錯誤。** 「使用自架 ASR 或未設定時，仍可使用內建的字幕來源」把自架 ASR 併進降級 fallback 句——對自架 operator 讀起來是「你的 ASR 不可用」，與事實相反（自架恰恰不需要這把金鑰）。意圖（告知自架免金鑰）可嘉、句子錯誤、且偏離未經設計輪。FIXED：回歸 AC #5 逐字核定字串；自架免金鑰事實留在 `docs/deployment.md`／`development.md`（該處已由 AC #4 交付）；in-code 註解記錄本裁定防回歸。
- [x] **[M2] `sweepEligible` 的可用性探針成本模型被本 story 隱形改變：nil-check → 每列一次 secrets DB 讀＋AES-256-GCM 解密。** `asrRecoverable()` → `Available()` → `IsAvailable()` → holder `IsConfigured` → `secrets.Retrieve`，在兩個列舉迴圈內逐列呼叫；`WithASRAvailability` 的 doc 反而一直宣告「Checked per sweep, not per item」——實作早就漂移，本 story 讓漂移變貴。緩解事實：`EnqueueMissing` 目前零生產呼叫者（cost-consent guard），屬 latent。FIXED：`EnqueueMissing` 開頭快照一次 `asrOK`，`sweepEligible(s, asrOK)` 傳值；語意正確性由既有的「over-enumeration 安全」註解背書（per-item entry gate 仍是權威閘門）。
- [x] **[L1] 測試計數不實。** Completion Notes／Change Log 宣稱「22 新測試（holder 14…）」，實數為 25（holder 16 含 H1 新增、整合 4、whisper 3、守衛 2）。FIXED：兩處更正。
- [x] **[L2] `TestASRProviderHolder_FingerprintCoversBaseURLAndModel` 測試名不符實。** 兩個**獨立** holder 的 client 必然不同指標，斷言無法否證「fingerprint 涵蓋 baseURL/model」——它實際驗的是建構子 option plumbing；且 baseURL/model 在單一 holder 生命期內不可變（constructor-fixed），該 fingerprint 維度今天本就不可否證。FIXED：更名 `TestASRProviderHolder_BaseURLAndModelReachTheClient` ＋ 誠實註解（future-proofing 定位，key 維度才是今天可否證的，已由 rebuild 測試涵蓋）。
- [x] **[L3] 「四個 setter 全部無條件執行」措辭過強。** `SetOpenCCConverter` 仍在 `if subtitleConverter != nil` 內（正確——converter-existence guard，typed-nil interface 陷阱防護，非 key 閘門）；且結構守衛只數呼叫次數、無法偵測條件包裹。FIXED：Completion Notes 措辭精確化為「不再 key-gated」＋ 記錄 guard 侷限。

### Reviewer verifications that held（未再由 orchestrator 重查）

三個下游可用性消費者確為零 diff（handler 介面簽名 byte 不變）· AC #4 白名單四處仍為真的重啟字串逐一確認未動（含兩個 spec guard）· `route-c-poc` 走 `os.Getenv` 非 `.HasOpenAIKey(`（守衛不誤傷）· `docs/development.md` 無 zh-TW 雙生（與 deployment.md 同為既有債，Rule 17 無新違規）· hosted 路徑 byte 不變（header 只在空金鑰時省略；metering 路徑未觸）· 守衛 falsification 紀錄屬實（Debug Log 有紅證）· `.pen` 漂移為 authoring 裁定且 `backlog-f5-asr-restart-copy-pen-resync` 雙向存在 · 修後全綠：api 34 packages、web 233/2620、gofmt/prettier clean。
