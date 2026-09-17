# Story DSR.6a：按下去會花錢的「生成字幕」，按鈕上先寫多少錢（後端估價＋前端按鈕＋設計稿）

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 「管理字幕」 on one movie or one episode,
I want 「生成字幕」（和失敗後的「重試」）按鈕上直接寫這次大概要花多少錢，算不出來就不給按,
so that 我在按下去的那一刻就知道要花多少錢，不會像 2026-08-07 那次一樣錢花掉了才知道。

## Context

`epic-dsr` 的 `dsr-6`（Flow F 字幕，最大的一塊）拆出來的**第一張**。⚖️ Alexyu 2026-09-17：「把 `disc-2026-09-single-item-cost-estimate` 和 `disc-2026-09-no-cost-bearing-component` 合成一個 story，當作 dsr-6 拆出來的第一塊」。兩張都是 **P0**。

- **產品規則**（`DESIGN.md:312-324`，⚖️ Alexyu 2026-09-10 A 案 ＋ PM 追加裁定）：會花錢的控制項一定帶預估金額；`$` 就是記號，不加硬幣圖示；**沒有金額就不給按**；估價還在路上 → 金額位置放骨架、暫時不可點；估價失敗 → 停用＋說明列寫原因；`≈` 只表示「片長是假設的」；一律兩位小數走 `usd()`；`$0.00` 不准寫「免費」。
- **今天的狀況**：`ManageSubtitleDialogV2.tsx:495-509` 的「生成字幕」**一按就直接開跑、直接花錢**（`startGeneration` → `trigger.mutate()`，`:232-235`），全檔沒有任何金額字串，也沒有確認步驟。後端沒有單片估價端點。
- **驗收基準**：`J9-D`（`Ls4GO`）六個狀態的定稿文案、`Component/ButtonCost/*` 三個母版的節點值。PNG 只是參考。

### 🔴 建單時查到的事

1. **按鈕做的事，跟候選清單估價器的「路線」不是同一件事。** 單片「生成字幕」永遠是「抽音訊 → 語音辨識 → 翻譯」（`transcription_service.go:566-690` 的 `runPipeline`，完全不看內嵌字幕軌）。這是 ⚖️ **2026-08-06 裁定 A**（`backlog-dialog-helper-verb-drift`：按鈕留在語音辨識路線）。唯一例外：`subtitle_status=untranslated` 且英文 SRT 還在 → **只翻譯**（`tryTranslateOnlyResume`，`:804-823`）。所以**不能**沿用候選清單的 `route`（extract／asr／skip）或 `estimated_usd`——有內嵌字幕的片會被估成「只翻譯」而低估好幾倍，`skip` 的片會被估成 $0 但按下去照樣付語音辨識費。
2. **候選快照不能當來源。** `GET /subtitles/generation-candidates` 是存在記憶體的整庫掃描結果：重啟就沒、只有 `status=ready` 才有 `result`、`per_candidate` 只用 `media_id` 當鍵（`generation_candidates.go:331-351, 543-588`）。
3. **模型。** 單片觸發不帶 `model_id`，實際扣的是 `claudeHolder.EffectiveModel()`（`CLAUDE_MODEL` 或 `claude-sonnet-5`，`claude_provider_holder.go:172-177`）。**不要用** `ModelCatalogService.DefaultModel`——`CLAUDE_MODEL` 不在目錄裡時它會退到清單第一個（`model_catalog.go:97-112`），估的就是別的模型的價。
4. **片長。** 候選估價的階梯是 `duration_seconds`（ffprobe 實測）→ `runtime`（TMDb 分鐘）→ 45 分假設（`candidateRow.runtimeMinutes`，`generation_candidates.go:1081-1089`）。但**影集的 `episodes.runtime` 沒有任何生產程式在寫**，`episodes.duration_seconds` 只有候選掃描探測過才有（`:775-778`）。不現場探測的話，影集幾乎每一集都是 `≈`。電影在 NFO 短路 enrichment 時也可能沒有 `duration_seconds`。
5. **哪些東西會讓金額變。** 自架語音辨識（`ai.IsSelfHostedASRBaseURL`）只讓**語音辨識那一半**變 0（`ai/budget.go:154-159`）；沒有 Claude 金鑰時 run 只做英文，**不收翻譯費**（`translateAndPersist`，`:850`）。所以 `$0.00` 只在「自架語音辨識＋沒有翻譯」時成立。
6. **設計稿有兩張違反 2026-08-06 裁定。** `F2-D-v2`（`$0.04`／「抽取內嵌字幕＋AI 翻譯，約需數分鐘」）與 `F1-M-v2`（同上）畫的是按鈕根本不會走的路線。`F1-D-v2` 是對的（`$0.27`／「語音辨識＋AI 翻譯，約需數分鐘」）。**`F1-M-v2` 與 `F1-D-v2` 是同一集（怪奇物語 S04E07），卻寫了不同路線、不同金額。**
7. **會花錢的按鈕不只一顆**：`action-generate-subtitle`、觸發失敗後的 `generation-trigger-retry`（`ManageSubtitleDialogV2.tsx:478-485`）、生成失敗面板的 `gen-retry`（`GenerationProgressV2.tsx:199-208`；`onRetry` 全 app 只有管理字幕對話框在傳）。設計稿 `F4-D-v2` footer 的「重試」（`iNhNz`，ref `Component/Button/Primary`）也沒有金額。
8. **金鑰未設定（J9-D ⑥）今天要按了才知道**：503 `TRANSCRIPTION_DISABLED` 回來才切到 `notConfigured` 畫面（`:427-465`）。
9. **電影的 `untranslated` 也會只翻譯**（`transcription_handler.go:95`），但說明列只對影集顯示便宜那句（`ManageSubtitleDialogV2.tsx:517`）。
10. **`Button.tsx` 幾乎沒人用**（4 個生產檔），字幕區三處手刻同一串 class（`ManageSubtitleDialogV2:500`、`ConfirmGenerationDialog:211`、`CandidateListPanel:1141`）。**ESLint Rule 21 的名稱不收 `/`**（`eslint-rules/implements-pen-node-id.js:43` 是 `Component\/[A-Za-z0-9-]+`），但 2026-09-10 起母版都是斜線名（`Component/ButtonCost/Default`，`disc-2026-09-component-variant-naming`）——新元件的檔頭照實寫會 lint 紅。
11. **送出中會在按鈕裡塞 `Loader2`**（`:501-506`），按鈕變寬；J9-D ④ 規定按鈕不得跳寬。

### 設計稿節點

| 代號 | 節點 | 內容 | 本張 |
| --- | --- | --- | --- |
| `Component/ButtonCost/Default` | `qAERt` | 泥金底 `$accent-primary`、圓角 `$radius-md`(8)、padding 8／20、gap 8；label「生成字幕」Noto Sans TC 14／600 `$text-on-accent`；amount `$0.42` **JetBrains Mono** 14／600 `$text-on-accent` | 元件的 ready 態 |
| `Component/ButtonCost/Loading` | `zhIx7` | 同上，amount 換成 40×12 骨架（`$text-on-accent`、opacity 0.25、`$radius-sm`） | loading 態 |
| `Component/ButtonCost/Disabled` | `dqE4G` | `$bg-tertiary` 底＋內側 1px `$border-subtle`；label `$text-disabled`；**沒有 amount** | unavailable 態 |
| `J9-D` | `Ls4GO` | 裁定、六個狀態定稿文案、邊界情況、§D 現況（`T8apb`／`igqGt`／`QG6G5`） | §D 改寫（AC #1） |
| `F1-D-v2` | `r1EY9` | 按鈕 `MruTd`（`$0.27`）、說明 `X7exGq`「語音辨識＋AI 翻譯，約需數分鐘」、條件文案註記 | 對照用，不改 |
| `F2-D-v2` | `S9Rbrq` | 按鈕 `N1qsH`（`$0.04`）、說明 `TXYYF`「抽取內嵌字幕＋AI 翻譯…」 | ✅ 改（AC #1） |
| `F1-M-v2` | `JkdfH` | 按鈕 `k0H8wR`（`$0.04`）、說明 `qR6hi`「抽取內嵌字幕＋AI 翻譯…」 | ✅ 改（AC #1） |
| `F4-D-v2` | `U8rRtv` | footer `dg5rH` 的 `iNhNz`（`Component/Button/Primary`「重試」）；錯誤句 `pjXCe`「翻譯失敗：AI 服務逾時，已保留轉錄結果，可直接重試翻譯」 | ✅ 改（AC #1） |

以上由 SM 以 Pencil MCP 讀出（2026-09-17）；動手前再 `Get` 一次。

**J9-D 六個狀態（定稿，逐字）：**

| # | 狀態 | 按鈕 | 說明列 |
| --- | --- | --- | --- |
| ① | 正常（片長實測） | `生成字幕 $0.42`，可點 | 語音辨識＋AI 翻譯，約需數分鐘 |
| ② | 片長未知（45 分假設） | `生成字幕 ≈ $0.42`，可點 | 片長未知（估 45 分）——實際費用依內容長度而定 |
| ③ | 零花費（自架語音辨識） | `生成字幕 $0.00`，可點 | 語音辨識：自架（不另計費） |
| ④ | 估價還在路上 | 金額位置骨架，暫時不可點 | （說明列不變） |
| ⑤ | 估價失敗 | 停用，無金額 | 暫時算不出費用，因此先不開放。重新整理或稍後再試。 |
| ⑥ | 金鑰未設定 | 停用，無金額 | 生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。 |

### 程式碼地圖

```
apps/api
  cmd/api/main.go:600-606      claudeHolder、modelCatalog
  cmd/api/main.go:929, 1167    transcriptionHandler 建立與註冊
  cmd/api/main.go:990-1019     generationCandidateService（routePredictorAdapter、selfHostedASR、episode duration writer）
  internal/handlers/transcription_handler.go   POST /movies/:id/transcribe、/episodes/:id/transcribe ← 本張加兩條 GET …/estimate
  internal/services/transcription_service.go   IsAvailable :317、canResumeTranslateOnly :795、translateAndPersist :847
  internal/services/generation_candidates.go   estimateUSD :1009、roundUSD :1041、runtimeMinutes :1081、translationRatePerMinute、RouteDurationPredictor :70
  internal/ai/budget.go                        EstimatedASRPerMinute、whisperPerMinuteUSD
apps/web/src
  components/subtitle/ManageSubtitleDialogV2.tsx   ← 本張：三顆付費按鈕、說明列
  components/subtitle/GenerationProgressV2.tsx     ← 本張：gen-retry 帶金額
  components/ui/ButtonCost.tsx                     ← 本張新增
  services/transcriptionService.ts                 ← 本張：估價呼叫
  lib/currency.ts、components/subtitle/consent/consentSelection.ts:371   usd()、usdWithEstimate()
  eslint-rules/implements-pen-node-id.js           ← 本張：名稱收 `/`
```

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP）。**
   - 母版的子節點：`Component/ButtonCost/Default` 的 label 是 `StNiU`、amount 是 `ASE9e`。
   - `F2-D-v2`：`TXYYF` → 「語音辨識＋AI 翻譯，約需數分鐘」；`N1qsH` 的 amount（`ASE9e`）`$0.04` → `$0.67`（S01E01 以 48 分估：48 ×（0.006＋0.00804））。
   - `F1-M-v2`：`qR6hi` → 「語音辨識＋AI 翻譯，約需數分鐘」；`k0H8wR` 的 amount → `$0.27`（跟 `F1-D-v2` 同一集，同一個數）。
   - `F4-D-v2`：footer（`dg5rH`）的 `iNhNz`（`ref otvKh`，padding `[$Space/md, $Space/xl]`）換成 `Component/ButtonCost/Default` instance，`StNiU`「重試」、`ASE9e` `$0.39`（錯誤句寫「已保留轉錄結果」→ 只翻譯，以 48 分 × 0.00804 估），padding 沿用原值。**用 `Replace` 換 instance，不要 `Insert` 新 frame**（Insert 會位移，見已知陷阱）。
   - `J9-D` §D 三段（`T8apb`／`igqGt`／`QG6G5`）改寫成本張完成後的現況：付費入口「生成字幕」與兩顆「重試」都帶金額；單片估價端點是 `GET /api/v1/{movies|episodes}/:id/transcribe/estimate`；元件是 `components/ui/ButtonCost.tsx`。§D 標題的日期改成 2026-09-17。
   - `J9-D` §C 末尾 **Copy** 一個既有段落（例如 `Q9GlUk`）補一段「建單補充（SM 2026-09-17）」，逐字寫進 AC #5 裡三句 J9-D 原本沒有的文案與適用情況：①「語音辨識：自架（不另計費）。僅能產生英文字幕——尚未設定翻譯金鑰」（`$0.00`）；②只差翻譯但沒有翻譯金鑰 → 停用＋「尚未設定翻譯金鑰」；③影片檔讀不到 → 停用＋「讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。」；以及「說明列優先順序較高時，按鈕上的 ≈ 不另加說明」。
   - 每張改完用 `ctx.problems` 掃裁切。🚨 存檔要確認 ` M ux-design.pen`（`.claude/memory/feedback_verify_pen_saved_before_commit.md`）。匯出後**只 stage** `flow-f-subtitle-v2/{f2-d-v2,f1-m-v2,f4-d-v2}.png`、`flow-j-specs/j9-d.png`，**以及 `_bmad-output/pen-tokens.json`**（`check-design-tokens.py:157-164` 比對 `.pen` 的 sha256，CI 會跑；**不准還原它**）。其他重繪雜訊 `git checkout` 還原（`SCREENS` 四筆都已經在，不用加）。

2. **[@contract-v1] 單片估價端點（後端）。** 在 `TranscriptionHandler` 加：
   - `GET /api/v1/movies/:id/transcribe/estimate`
   - `GET /api/v1/episodes/:id/transcribe/estimate`（另外還要 `episodeService != nil`，同觸發路由）
   - **注入方式**：`NewTranscriptionHandler` 有 16 個呼叫點（main＋3 支測試檔），**不要改建構子**。加 setter（例如 `SetEstimator`）；**沒設定就不掛這兩條 GET**（→ 404），既有測試不用動。

   200 回應（Rule 3 信封，`data` 欄位名逐字）：
   ```json
   {
     "media_id": "uuid",
     "media_type": "movie",
     "plan": "full",
     "asr_available": true,
     "self_hosted_asr": false,
     "translation_configured": true,
     "model_id": "claude-sonnet-5",
     "runtime_minutes": 48.5,
     "runtime_known": true,
     "runtime_source": "ffprobe",
     "estimated_usd": 0.68
   }
   ```
   - `media_type`：`movie` | `episode`。`plan`：`full` | `translate_only`。`runtime_source`：`ffprobe` | `tmdb` | `fallback`（沿用 `generation_candidates.go:130-141` 的 `RuntimeSource*` 常數；confirmed against [@contract-v1] (Story sub-4-1 AC #7)）。`model_id`：`translation_configured=false` 時是 `""`。
   - **`estimated_usd` 是 JSON number**：Go 欄位 `float64`，由 `decimal` 經 `.InexactFloat64()` 轉出（同 `generation_candidates.go:246` 的做法）。**不要**直接放 `decimal.Decimal`（會序列化成字串 `"0.68"`）。
   - 錯誤：找不到 → `NotFoundError`（`DB_NOT_FOUND`，404）；影集查詢遇到「找不到」以外的錯誤 → 500 `INTERNAL_ERROR`（照 `TranscribeEpisode:208-219` 的分法）；沒有 `file_path` 或檔案讀不到 → 400 `VALIDATION_REQUIRED_FIELD`，**訊息沿用同檔觸發路由的字句**。**不新增任何 Rule 7 錯誤碼或前綴。**
   - `asr_available=false` 時照樣 200、照樣算金額（前端負責停用）。
   - Swaggo 註解補齊（`@Summary`／`@Description`／`@Tags subtitles`／`@Param`／`@Success 200`／`@Failure`／`@Router`）。⛔ **不要跑 `swag init`**（`disc-2026-09-swagger-docs-stale`）。
   - `main.go` 接線；Rule 15 自檢（路由真的註冊、`GET /movies/:id` 與 `POST /movies/:id/transcribe` 不受影響）。

3. **估價跟 run 用同一份判斷（後端）。** 新增服務（名稱自訂，放 `internal/services`——跟 `estimateUSD`、`roundUSD`、`candidateRow` 同一個 package，這些未匯出的函式才用得到）。服務**直接持有 `*TranscriptionService`**，不要自己另外拿一份 translation service：
   - **`plan`**：跟觸發路由用**同一個**續跑判斷（`canResumeTranslateOnly(ctx, mediaType, id)`）。`true` → `translate_only`，否則 `full`。（對話框的電影觸發一律帶 `?translate=true`、影集一律翻譯，所以估價固定以「要翻譯」為前提；在 godoc 寫明。）
   - **`translation_configured`**：把 `translateAndPersist:850` 裡「translation service 非 nil 且 `IsConfigured()`」**抽成 `TranscriptionService` 的一個方法**，run 與估價都呼叫它（不准兩處各寫一次）。**`asr_available`** = `IsAvailable()`。**`self_hosted_asr`** = 與 `main.go:1001` 同一個來源 `ai.IsSelfHostedASRBaseURL(cfg.ASRBaseURL)`。
   - **`model_id`** = `claudeHolder.EffectiveModel()`（🔴 #3）。
   - **金額一律呼叫既有函式組出來，不准抄公式或常數**（`asrRate = ai.EstimatedASRPerMinute(selfHosted)`）：

     | plan | 有翻譯 | 金額 |
     | --- | --- | --- |
     | `full` | 是 | `estimateUSD(RouteASR, 分鐘, asrRate, model)` |
     | `full` | 否 | `roundUSD(分鐘 × asrRate)` |
     | `translate_only` | 是 | `estimateUSD(RouteExtract, 分鐘, asrRate, model)`（extract 就是「只收翻譯費」） |
     | `translate_only` | 否 | `0` |

   - **片長階梯**：① 已存的 `duration_seconds > 0` → ② 沒有的話現場探測（`RouteDurationPredictor.ProbeWithDuration`），秒數 > 0 就用（`runtime_source=ffprobe`）→ ③ `runtime > 0`（`tmdb`）→ ④ 45 分（`fallback`，`runtime_known=false`）。**探測失敗記 Warn 繼續往下走，絕不回 500。** 重用 `candidateRow.runtimeMinutes()`（先把量到的秒數填進 `durationSeconds` 再呼叫），不要寫第二份階梯。
   - **`main.go` 接線**：`routePredictorAdapter` 今天是寫在 `:997` 的行內值（沒有變數名），而且比 `transcriptionHandler`（`:929`）晚建。把它**提到 `:929` 之前**成為變數（例如 `routePredictor := routePredictorAdapter{...}`，它需要的 `ffprobeService` 在 `:435` 就有了），候選服務與估價服務都用這個變數。真正要共用的是底下的 `ffprobeService`（3 格 semaphore）。
   - **只有影集把量到的秒數寫回**（`UpdateDurationSeconds`，同 `rememberEpisodeDuration` 的規則；寫回失敗只記 Debug）。電影不寫（`duration_seconds` 的主人是 enrichment）。
   - **估價不花錢、不佔工作**：不呼叫 `acquireJob`、不送 SSE、除了影集片長之外不寫任何東西。

4. **`ButtonCost` 元件（前端，新檔 `components/ui/ButtonCost.tsx`）。**
   - 檔頭兩行：`// Implements: Component/ButtonCost/Default (qAERt) + Component/ButtonCost/Loading (zhIx7) + Component/ButtonCost/Disabled (dqE4G)`、`// Source: ux-design.pen (Pencil app)`。
   - **ESLint Rule 21 名稱收 `/`**（🔴 #10）：`implements-pen-node-id.js:43` 的兩處 `Component\/[A-Za-z0-9-]+` 改成 `Component\/[A-Za-z0-9-]+(?:\/[A-Za-z0-9-]+)*`（斜線只能出現在兩段名字之間），註解同步；`implements-pen-node-id.spec.ts` 加：`Component/ButtonCost/Default (qAERt)` 合法、`Component//X (abc)` 不合法、`Component/X/ (abc)` 不合法。`project-context.md` Rule 21 的 `{Name}` 說明補一句「可含 `/`（2026-09-10 起母版用斜線命名空間）」。
   - props：`label: string`、`cost: { status: 'ready'; usd: number; approximate: boolean } | { status: 'loading' } | { status: 'unavailable' }`、`onClick`、`busy?: boolean`、`className?`、其餘 button 屬性（含 `data-testid`、`aria-describedby`）往下傳。
   - **ready**：泥金實心、`label` ＋ 金額。金額用 `font-mono font-semibold tabular-nums`，**顏色跟 label 一樣**（不另外上色）。文字用 `usdWithEstimate(usd, approximate)` → `$0.42`／`≈ $0.42`／`$0.00`。`usdWithEstimate` 從 `consentSelection.ts:371` **搬到 `lib/currency.ts`**，`consentSelection.ts` **必須 re-export**（`consentSelection.spec.ts:19`、`CandidateListPanel.tsx:35`、`ConfirmGenerationDialog.tsx:33` 都從它 import，這三個檔本張不准動）。
   - **loading**：外觀同 ready，金額位置換成骨架 `inline-block h-3 w-[5ch] rounded-[var(--radius-sm)] bg-[var(--text-on-accent)] opacity-25`；`aria-disabled="true"`、點了不呼叫 `onClick`；有 sr-only 文字「正在估算費用」。
   - **unavailable**：`disabled`；`bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] text-[var(--text-disabled)]`；**不渲染金額**。
   - **busy**（觸發請求送出中）：`disabled` ＋ `aria-busy="true"`，**內容不變、不塞轉圈**（🔴 #11）。
   - 金額欄位 `min-w-[5ch]`，骨架換成數字時按鈕不跳寬（J9-D ④）。
   - 尺寸：`min-h-[44px]`（觸控；母版 36 是元件預設，畫面上的 instance 是 47／51）、`px-5 gap-2`、`rounded-[var(--radius-md)]`、`text-sm font-semibold`、hover `bg-[var(--accent-pressed)]`。**不加硬幣或任何圖示。**

5. **管理字幕對話框接上估價。**
   - `transcriptionService` 加 `getTranscriptionEstimate(mediaType: 'movie' | 'episode', id)` → 呼叫 AC #2 的端點，回傳 camelCase 型別 `TranscriptionEstimate`（`mediaId`、`mediaType`、`plan: 'full' | 'translate_only'`、`asrAvailable`、`selfHostedAsr`、`translationConfigured`、`modelId`、`runtimeMinutes`、`runtimeKnown`、`runtimeSource`、`estimatedUsd`）。（confirmed against [@contract-v1] (Story dsr-6a AC #2)）
   - `transcriptionService` 的估價呼叫丟 `lib/apiError.ts` 的 `ApiError`（要讀 `code`），並把 TanStack Query 的 `signal` 傳給 `fetch`（關掉對話框時取消還在排隊等 ffprobe 的請求）。
   - hook `useTranscriptionEstimate`：key `['subtitles', 'transcription-estimate', mediaType, mediaId]`、**用全域預設 staleTime（不要設 0）**、`enabled: open && (isMovie || isEpisode)`。**對話框關閉時 `queryClient.removeQueries` 這個 key**——每次打開都重新估價，而且視覺夾具預塞的資料不會被立刻重抓（CI 的視覺測試沒有後端）。**影集（series）不發請求。**
   - **先看有沒有資料，再看錯誤**：`data` 存在 → 用 `data`（背景重抓失敗也不換掉）；沒有 `data` 且 `isError` → 失敗；其他 → 載入中。`isFetching` **不准**把已經有價錢的按鈕變回骨架。
   - 把「按鈕狀態＋說明列」寫成**一個純函式**（輸入：估價查詢結果、`mediaType`、`subtitleStatus`；輸出：`cost`、說明文字、語氣、要不要「前往設定」連結），主按鈕與兩顆重試共用，單元測試直接測這個函式。
   - **按鈕狀態**（`action-generate-subtitle`）：

     | 情況 | 按鈕 | 說明列 |
     | --- | --- | --- |
     | series | `ButtonCost` unavailable | 「請於下方分集清單逐集生成」（現有） |
     | 估價載入中 | loading | 影集或電影 `subtitleStatus === 'untranslated'` →「僅需翻譯，不再重跑語音辨識——這次很快也很便宜」；其他 →「語音辨識＋AI 翻譯，約需數分鐘」 |
     | 估價失敗，`code === 'VALIDATION_REQUIRED_FIELD'` | unavailable | 「讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。」，`--text-secondary` |
     | 估價失敗，其他 | unavailable | J9-D ⑤，`--text-secondary` |
     | `plan=full` 且 `asrAvailable=false` | unavailable | J9-D ⑥ ＋「前往設定」（`/settings/keys`，沿用 `go-to-settings` 的做法），`--text-secondary` |
     | `plan=translate_only` 且 `translationConfigured=false` | unavailable | 「尚未設定翻譯金鑰」＋「前往設定」，`--text-secondary`（這時按下去會續跑、跳過翻譯、什麼都沒產生——不能給一顆能按的 `$0.00`） |
     | 其他 | ready，金額 `estimatedUsd`，`approximate = runtimeSource === 'fallback'`（語意同 `isRuntimeApproximate`） | 見下 |

   - **ready 時說明列的優先順序**（可點時維持今天的 `--text-muted`）：
     1. `plan=translate_only` →「僅需翻譯，不再重跑語音辨識——這次很快也很便宜」（**電影也適用**，🔴 #9）
     2. `translationConfigured=false` →「僅能產生英文字幕——尚未設定翻譯金鑰」＋「前往設定」（現有 `helper-goto-settings`）；若同時 `selfHostedAsr=true`（這時金額是 `$0.00`），句首加「語音辨識：自架（不另計費）。」
     3. `runtimeSource=fallback` → J9-D ②
     4. 其他 →「語音辨識＋AI 翻譯，約需數分鐘」

     第 1、2 條命中時按鈕上若有 `≈`，不另加說明（一列最多一個 ≈；已寫進 J9-D §C 的建單補充，AC #1）。
   - **無障礙**：按鈕用 `aria-describedby` 指向說明列（三種狀態都要）。
   - `translationDegraded` 改讀估價的 `translationConfigured`；`useKeySettings` 在這個檔案只用在這裡，**拿掉**（兩個來源會分岔）。
   - **兩顆「重試」也帶金額**（🔴 #7）：`generation-trigger-retry` 與 `gen-retry` 都改用 `ButtonCost`（label「重試」），狀態來自同一個純函式。**兩個面板裡都沒有說明列**，所以：`unavailable` 或 `approximate` 時，在重試按鈕下方渲染那句說明（⑥ 與沒有翻譯金鑰時含「前往設定」），並用 `aria-describedby` 連上；ready 且不是 ≈ 時不多渲染。`GenerationProgressV2` 的 props 改成 `onRetry`、`retryCost`、`retryNote?: ReactNode` 一組（`onRetry` 與 `retryCost` 型別上必須一起給），拿掉 `RotateCcw` 圖示；testid 不變。`generation-trigger-retry` 原本是文字連結樣式，換成實心按鈕沒有對應的設計稿——記在 Completion Notes 給 Sally 確認，不另立案。
   - **價格要跟著狀態更新**：觸發失敗時、以及 `generation.progress.phase` 變成 `failed` 或 `complete` 時，invalidate 估價 key（一次失敗的 run 可能已經留下英文 SRT，重試就變成只翻譯、價錢變便宜）。`useGenerationProgress` 只有 `onComplete`（`:183-186`），失敗要在對話框用 `useEffect` 盯 `generation.progress.phase`＋`useQueryClient`（對話框今天兩個都沒有）。
   - 點擊：只有 ready 會呼叫 `trigger.mutate()`；觸發的端點與請求**完全不變**。`trigger.isPending` → `busy`。按下後回 503 仍然走既有的 `notConfigured` 畫面（估價與按下之間金鑰被刪的情況）。
   - 檔頭：`ManageSubtitleDialogV2.tsx` 維持 `Screen F1-D-v2 (r1EY9)`，加 `+ Screen F2-D-v2 (S9Rbrq) + Screen F1-M-v2 (JkdfH)`；`GenerationProgressV2.tsx` 檔頭不變。

6. **後端測試**（testify ＋ 手寫 stub，照 `generation_candidates_test.go` 的寫法，不用 SQLite）：
   - 金額（AC #3 表格四格＋自架兩種）：`full` 有翻譯／`full` 沒翻譯（只有語音辨識費）／`translate_only` 有翻譯（只有翻譯費）／`translate_only` 沒翻譯（`0`）／自架＋有翻譯（只有翻譯費）／自架＋沒翻譯（`0`）。
   - 對照：`full`＋有翻譯 ＝ `estimateUSD(RouteASR, …)`。
   - 模型：holder 的 `EffectiveModel` 設成 **`claude-haiku-4-5`**（費率 0.00301，跟預設 Sonnet 的 0.00804 不同——用目錄外的 id 測不出來，因為未知模型也是算 Sonnet 價），斷言翻譯那一半用的是 Haiku 價。
   - 片長四階：已存 → 探測 → TMDb → 45；探測回錯誤時往下走不回錯；影集寫回、電影不寫回。
   - 不佔工作：估價後 `IsInProgress(id) == false`，沒有 SSE。
   - 抽出來的「有沒有翻譯」方法：`translateAndPersist` 既有測試照綠。
   - handler：200 的 `data` 鍵名逐字（用 `map[string]any` 斷言，抓大小寫與拼字），`estimated_usd` 斷言是 `float64`；404；影集查詢一般錯誤 500；沒有 `file_path` 的 400；**沒呼叫 setter 時兩條 GET 都是 404**；`episodeService == nil` 時影集那條 404。

7. **前端測試。**
   - `ButtonCost.spec.tsx`：三態＋busy；`$0.42`／`≈ $0.42`／`$0.00`；loading 與 unavailable 點了不呼叫 `onClick`；busy 有 `aria-busy` 且沒有 `.animate-spin`；unavailable 不渲染金額。
   - `lib/currency` spec：`usdWithEstimate` 搬家後輸出不變。
   - ESLint rule spec：斜線名（合法一條、不合法兩條）。
   - 按鈕狀態純函式的 spec：AC #5 表格每一列＋說明列四條優先順序＋「有 data 但背景重抓失敗仍是 ready」＋「`isFetching` 不變回 loading」。
   - `ManageSubtitleDialogV2.spec.tsx`：**既有的點擊測試全部先把估價 mock 成 ready 再點**（否則按鈕是骨架）；既有的 `useKeySettings` mock（`:86-93`）改成 mock 估價。新增：載入中不可點；估價失敗 → ⑤ 且停用；讀不到檔案 → 那句且停用；`asrAvailable=false` → ⑥＋連結、點了不觸發；`translate_only`＋沒翻譯金鑰 → 停用；電影 `translate_only` 說明列；沒翻譯金鑰說明列（＋自架時的前綴與 `$0.00`）；`fallback` → `≈` ＋ ②；series 不發估價請求；兩顆重試帶金額、停用時下方有原因；生成 `failed` 後估價被重抓；關閉對話框後估價快取被移除；`aria-describedby` 指向說明列。金額**逐字**斷言（`toHaveTextContent` 是子字串比對）。
   - `GenerationProgressV2.spec.tsx`：重試帶金額；`retryNote` 有給才渲染。

8. **視覺夾具與基準線。**
   - `subtitle-manage-subtitle-dialog-v2`（`-gallery.fixtures.tsx:4260-4318`）：`seedQueries` 補估價 key（`plan: 'full'`、`asrAvailable: true`、`translationConfigured: true`、`runtimeSource: 'ffprobe'`、`estimatedUsd: 0.42`）。**基準線會變**（按鈕多了 `$0.42`）——這是預期的。CI 的視覺測試只起前端、沒有後端（`visual-regression.yml:320`）：夾具截圖**必須**是 `$0.42`，不是骨架也不是 ⑤——若出現後兩者，代表 AC #5 的 staleTime 或「先看資料」沒做對。
   - `generation-progress-v2/失敗`：補 `retryCost`（ready，`usd: 0.39`）。基準線會變。
   - 新增 `ui-button-cost`：一張夾具把四種樣子直排（`$0.42`／`≈ $0.42`／loading／unavailable），不打網路、不含日期、`statesOnly: ['default']`。
   - 基準線照 dsr-2 AC #14 的五步流程（`.claude/memory/project_visual_baseline_intentional_change.md`）。⛔ 不要本機產 `-linux.png`。

9. **文件與單子。**
   - `DESIGN.md:326` 的「⚠️ 現況（2026-09-10）」整段改寫成完成後的現況（與 AC #1 的 J9-D §D 同一套說法）。
   - `DESIGN.md:324` 的母版名稱 `Component/ButtonCost` / `-Loading` / `-Disabled` 改成 `.pen` 現在的名字 `Component/ButtonCost/Default` / `/Loading` / `/Disabled`。
   - 收單時：`disc-2026-09-single-item-cost-estimate` → `done`；`disc-2026-09-no-cost-bearing-component` → `done`；`dsr-6-flow-f-subtitle-v2` 的 ②（F16 金額標籤）以外，註記「生成字幕／重試帶金額已由 dsr-6a 完成」。

10. **既有測試保留通過**，只有下列是本張刻意改的：`ManageSubtitleDialogV2.spec.tsx`（點擊前先 mock 估價、說明列來源改讀估價、`useKeySettings` mock 換掉）、`GenerationProgressV2.spec.tsx`（重試 props）、兩張夾具基準線。特別守住：批次同意流程（`GenerationConsentView`／`CandidateListPanel`／`ConfirmGenerationDialog` 的 spec 與 `tests/e2e/batch-subtitle.spec.ts`）一行不改仍綠——`usdWithEstimate` 搬家不能改變它們。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`、Go 的 `gofmt`。⛔ 絕不用 `run_in_background` 跑測試。⛔ 局部 vitest 綠之後**一定要跑 typecheck**。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] F2-D-v2、F1-M-v2 文案與金額；F4-D-v2 的重試 `Replace` 成 `ButtonCost` instance
  - [x] J9-D §D 改寫、§C 建單補充；`ctx.problems`；確認 ` M ux-design.pen`；stage 四張圖＋`pen-tokens.json`
- [x] **Task 2 — 後端估價服務（AC: #3, #6 服務部分）**
  - [x] 先寫紅測試：六種金額、對照 `estimateUSD`、Haiku 模型、片長四階、寫回規則、不佔工作
  - [x] 把「有沒有翻譯」抽成 `TranscriptionService` 方法（`translateAndPersist` 改呼叫它）
  - [x] 服務（持有 `*TranscriptionService`）；金額照 AC #3 表格呼叫既有函式；重用 `runtimeMinutes`
- [x] **Task 3 — 後端路由（AC: #2, #6 handler 部分）**
  - [x] handler setter＋兩條 GET＋測試；`main.go`：`routePredictor` 變數提到 `:929` 之前、`repos.Episodes`、`claudeHolder.EffectiveModel`、`ai.IsSelfHostedASRBaseURL(cfg.ASRBaseURL)`；Swaggo 註解（不跑 `swag init`）；`gofmt`
- [x] **Task 4 — `ButtonCost` 元件（AC: #4, #7 元件部分）**
  - [x] ESLint 規則收 `/`（嚴格版 regex）＋spec；`project-context.md` Rule 21 一句
  - [x] `usdWithEstimate` 搬到 `lib/currency.ts`，`consentSelection.ts` re-export
  - [x] 元件＋spec
- [x] **Task 5 — 對話框接線（AC: #5, #7 對話框部分）**
  - [x] 先改既有 spec（點擊前 mock 估價 ready、換掉 `useKeySettings` mock），再寫新的紅測試
  - [x] service（`ApiError`＋`signal`）＋hook（預設 staleTime、關閉時 `removeQueries`）
  - [x] 按鈕狀態純函式＋spec；三顆按鈕；重試下方的原因；`aria-describedby`；failed／complete invalidate；`GenerationProgressV2` 的 `retryCost`／`retryNote`
- [x] **Task 6 — 夾具與基準線（AC: #8）**
- [x] **Task 7 — 收尾（AC: #9, #10, #11）**
  - [x] `DESIGN.md:326`；全套閘門；dev-story Step 9 截圖比對（`j9-d`、`f1-d-v2`、`f2-d-v2`、`f1-m-v2`、`f4-d-v2`）
  - [x] 收單時關兩張 disc、更新 `dsr-6` 註記

## Dev Notes

### 這張的重點

- **價錢要估「按鈕真的會做的事」**，不是候選清單的路線（🔴 #1）。最容易犯的錯就是去讀候選快照或 `estimate_usd`。
- **一份判斷兩個地方用**：估價的 `plan`、有沒有翻譯、模型、費率，全部要跟 `runPipeline` 讀同一個函式。對照測試（AC #3）就是在守這件事。
- **沒有金額就不給按。** 前端任何「不知道價錢」的狀態都只能是 loading（不可點）或 unavailable（停用），**不准退回一顆沒有金額但能按的按鈕**。停用一定要看得到原因——主按鈕看說明列，兩顆重試看按鈕下方那句。
- **SM 建單裁定（2026-09-17，J9-D 沒畫到的三種情況）**：①自架語音辨識＋沒有翻譯金鑰 → `$0.00`，說明列兩句接起來；②只差翻譯但沒有翻譯金鑰 → 停用（按下去什麼都不會產生，不能給一顆能按的 `$0.00`）；③影片檔讀不到 → 停用＋專用句子（⑤ 的「稍後再試」對檔案不見的情況不是實話）。三句都寫進 J9-D §C（AC #1），Sally／Alexyu 可以在 review 時推翻。

### 不要做的事

- **不要改觸發端點或它的請求**（`POST …/transcribe`），不要加 `model_id` 或預算參數。
- **不要讓單片按鈕改走內嵌字幕抽取／`/subtitles/pipeline/run`**——⚖️ 2026-08-06 裁定 A。
- **不要動批次同意流程**（`GenerationConsentView`、`CandidateListPanel`、`ConfirmGenerationDialog`、`ModelPicker`）的行為；`disc-2026-09-candidate-usd-missing-guard` 是另一張單子。
- **不要改 `Button.tsx`**；新元件獨立一個檔。
- **不要加硬幣圖示、不要給金額上色、不要用 `formatUsdShort`**（DESIGN.md:308, 320）。
- **不要跑 `swag init`**。
- **不要在估價端點裡佔 `inProgress`、送 SSE、或寫電影的 `duration_seconds`。**

### 已知陷阱

- **Pencil `Insert` 新 frame 會位移約 50px、內容被裁掉**（dsr-8 Debug Log）。換 instance 用 `Replace`，其他一律 `Copy` 既有節點再改。
- **Radix Dialog 走 Portal**：單元測試找對話框用 `screen`，不是 `container`。
- **`toHaveTextContent('生成字幕')` 是子字串比對**（spec `:194`），加了金額照樣會過——新測試要**逐字**斷言金額文字。
- **視覺 CI 沒有後端**（`visual-regression.yml:320`）、截圖前也不等網路靜止（`components.visual.spec.ts:220-222`）。估價查詢只要一重抓就會失敗——這就是 AC #5 不准 `staleTime: 0`、而且要「先看資料」的原因。
- **`check-design-tokens.py` 會比對 `.pen` 的 sha256 與 `pen-tokens.json`**（`:157-164`）。改了 `.pen` 就一定要一起 commit 匯出產生的 `pen-tokens.json`。
- **探測可能要等**：ffprobe 是 3 格 semaphore＋每次 10 秒逾時（`main.go:435`）。候選掃描正在跑時 3 格可能全被佔，估價要排隊。前端要能長時間停在 loading，不要把 loading 當錯誤。
- **`duration_seconds` 是整數秒**（float 截斷存進去），`runtime` 是分鐘。
- **`input_per_1m` 轉 camelCase 會變 `inputPer_1m`**（`subtitleService.ts:183-190`）——本張的欄位沒有數字，但新型別要看一下 `snakeToCamel` 的結果。
- **行號以建單時為準**（2026-09-17，main `9b1c2581`）。

### Source tree

```
apps/api/internal/services/<新估價服務>.go(+_test.go)          ← Task 2（新）
apps/api/internal/handlers/transcription_handler.go(+_test.go) ← Task 3
apps/api/cmd/api/main.go                                       ← Task 3
apps/web/src/components/ui/ButtonCost.tsx(+spec)               ← Task 4（新）
apps/web/src/eslint-rules/implements-pen-node-id.js(+spec)     ← Task 4
apps/web/src/lib/currency.ts(+spec)、components/subtitle/consent/consentSelection.ts ← Task 4
apps/web/src/services/transcriptionService.ts                  ← Task 5
apps/web/src/hooks/useTranscriptionEstimate.ts(+spec)          ← Task 5（新，名稱可調）
apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx(+spec) ← Task 5
apps/web/src/components/subtitle/GenerationProgressV2.tsx(+spec)   ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx                 ← Task 6
tests/visual/components.visual.spec.ts-snapshots/components/{subtitle-manage-subtitle-dialog-v2,generation-progress-v2,ui-button-cost}/** ← Task 6
apps/web/src/components/subtitle/<按鈕狀態純函式>.ts(+spec)       ← Task 5（新，名稱可調）
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f2-d-v2,f1-m-v2,f4-d-v2}.png、flow-j-specs/j9-d.png ← Task 1
DESIGN.md、project-context.md                                  ← Task 4, 7
```

### Cross-Stack Split Check

後端 task **2 個**（Task 2、3），前端／設計 task 5 個。後端 ≤ 3 → **本張不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `ButtonCost` 與改動的兩個元件都不讀時鐘；新夾具不含日期。

### References

- [Source: `ux-design.pen` `qAERt`／`zhIx7`／`dqE4G`／`Ls4GO`（J9-D）／`r1EY9`（F1-D-v2）／`S9Rbrq`（F2-D-v2）／`JkdfH`（F1-M-v2）／`U8rRtv`（F4-D-v2）] — Pencil MCP 讀出
- [Source: `DESIGN.md:308-326`] — 金錢不上色、會花錢的動作要有記號、金額寫法、現況
- [Source: `apps/api/internal/services/transcription_service.go:317, 337-372, 566-690, 777-823, 847-863`] — 可用性、單片 run、續跑、翻譯條件
- [Source: `apps/api/internal/handlers/transcription_handler.go:62-164`] — 觸發路由、錯誤字句、`translate` 參數
- [Source: `apps/api/internal/services/generation_candidates.go:53-80, 404-480, 760-800, 1009-1089, 1277-1330`] — 估價函式、片長階梯、探測、寫回
- [Source: `apps/api/internal/services/claude_provider_holder.go:94-104, 172-177`、`model_catalog.go:97-112`、`ai/budget.go:101, 154-159`、`ai/whisper.go:125-127`] — 模型與費率
- [Source: `apps/api/cmd/api/main.go:600-606, 929, 990-1019, 1167`] — 接線
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:131-235, 427-544`、`GenerationProgressV2.tsx:58, 118, 188-210`、`services/transcriptionService.ts:47-84`、`lib/currency.ts:48`、`subtitle/consent/consentSelection.ts:72-74, 371-373`、`eslint-rules/implements-pen-node-id.js:43`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:4065-4135, 4260-4318`、`tests/visual/components.visual.spec.ts:252-253, 313-319`]
- [Source: `sprint-status.yaml` → `backlog-dialog-helper-verb-drift`（⚖️ 2026-08-06 裁定 A）]
- [Source: `_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md` AC #5–#7、`sub-6-8a-per-run-model-selection-backend.md`、`sub-6-12-consent-money-traps.md` AC #4] — 候選估價的來歷（本張刻意不共用它的路線）
- [Source: `_bmad-output/implementation-artifacts/dsr-2-flow-b-detail-v2.md` AC #14、`dsr-8-flow-i-discover-v2.md` Debug Log] — 基準線流程、Pencil Insert 陷阱
- [Source: project-context.md#Rule 3 / #Rule 7 / #Rule 11 / #Rule 15 / #Rule 19 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-17

### Debug Log References

- `pnpm nx test api --skip-nx-cache`：PASS（35 個 package ok）。新測試：`services/transcription_estimate_test.go` 13 條、`handlers/transcription_estimate_handler_test.go` 7 條（含子測試）；紅燈先確認（編譯失敗＝紅）。
- `pnpm nx test web --skip-nx-cache`：**3545 / 3545 passed**（266 files）。「關閉後移除價錢」與「失敗後重新估價」兩條另外做了反向驗證：把對應的程式碼拿掉，測試確實變紅，再還原。
- `pnpm run lint:all`：**0 errors**（128 warnings，與前一張 dsr-2b-b 記錄的數字相同，本張改過的檔單獨 eslint 0 problems）· `web:typecheck`：PASS · `format:check`：PASS · `check-design-tokens.py`：一致（82 變數、194 張畫面、73 母版）· `gofmt -l` 本張的 Go 檔：無輸出。
- **本機 API 煙霧測試**（`go build` 後以空資料夾啟動）：`GET /api/v1/movies/<uuid>/transcribe/estimate` 與 episodes 版都回 `DB_NOT_FOUND` 的 JSON 信封（路由真的有掛上，Rule 15）；打錯的路徑是 gin 的 404。
- **視覺**：`nx serve web` ＋ `CI=1 VISUAL_BUCKETS=6 --workers=6 --update-snapshots=all`（1.8 分鐘）→ 185 張 re-render 雜訊全數還原，只留 `subtitle-manage-subtitle-dialog-v2`、`generation-progress-v2/失敗` 兩張更新與新夾具 `ui-button-cost`；兩張舊的 `-linux` 已 `git rm`（交給 CI bootstrap）。再跑一次比對模式：本張三張全過，紅的只剩 `glossary-panel-v2/seeded`、`parse-floating-parse-progress-card`、`retry-retry-notifications`——正是 `preexisting-fail-visual-darwin-three-stale-baselines` 點名的三張。
- **Pencil**：F2-D-v2／F1-M-v2 用 `Update`（文字與 instance 的 `descendants`）；F4-D-v2 的重試用 `Replace` 換成 `Component/ButtonCost/Default` instance（明確補 `enabled:true`）；J9-D §C 用 `Copy` 既有段落。`ctx.problems` 四張皆空。AppleScript File ▸ Save → ` M ux-design.pen`，並 grep 磁碟檔確認新段落真的落盤。匯出 194 張：Flow I 四張 re-render 雜訊還原，留下 `f1-m-v2`／`f2-d-v2`／`f4-d-v2`／`j9-d` 與 `pen-tokens.json`。
- ⚠️ Pencil 小怪事：F4-D-v2 新換上的重試 instance 在 Pencil 裡量出來高 36（母版高度），同樣 padding 的 `MruTd`（F1-D-v2）卻量出 47；磁碟上兩個節點的 JSON 除了 label 覆寫之外完全一樣。匯出圖看得出重試比旁邊的「稍後再試」矮一點。沒有再去調——只是稿上的高度，不影響程式碼（程式碼一律 `min-h-[44px]`）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 **AC Drift: FOUND**（`grep -r "約需數分鐘\|僅能產生英文字幕\|usdWithEstimate\|僅需翻譯，不再重跑\|action-generate-subtitle\|gen-retry"` 全部 story 檔）
  - `sub-2-2d-f5-cta-code-strings` AC #2（與 `sub-2-2b-untranslated-badge-frontend` 同一條）→ 「沒有翻譯金鑰」的訊號從 `GET /settings/keys` 的 `claude.configured` 改成估價回應的 `translation_configured`（同一個 run 自己的判斷）；「CTA 一直可按」改成「有價錢才可按」——載入中是骨架、估價失敗停用。沒有翻譯金鑰時仍然可按（degraded ≠ blocked 保留）。
  - `9R-UX-episode-row-cta-design`（`untranslated` 便宜說明只給影集）→ 電影能續跑只翻譯時也顯示。
  - `sub-6-12-consent-money-traps` AC #4（`usdWithEstimate`）→ 函式搬到 `lib/currency.ts`，`consentSelection.ts` re-export，批次同意畫面行為不變（`consentSelection.spec.ts` 一行未改仍綠）。
  - `ux3-subtitle-v2`（`GenerationProgressV2` 的 `onRetry` 可以單獨給）→ `onRetry` 與 `retryCost` 型別上必須一起給；拿掉 `RotateCcw` 圖示。
- 📎 **Contract Stamps: FOUND**（本張定義 AC #2 `[@contract-v1]`；消費 `sub-4-1` AC #7 `[@contract-v1]` 的 `runtime_source` 詞彙，ack 寫在 AC #2 與 `transcription_estimate.go` 的欄位註解；前端型別也標了 `confirmed against [@contract-v1] (Story dsr-6a AC #2)`。沒有任何 bump。）
- 🎭 **A11y Pre-Flight: PASS**（3 個元件：`ButtonCost`、`ManageSubtitleDialogV2`、`GenerationProgressV2`；改過的檔 jsx-a11y 0 warning。①圖片：無；②modal：對話框仍是 Radix，焦點行為不變；③非同步揭露：價錢載入中有 sr-only「正在估算費用」，送出中 `aria-busy`；④自訂 widget：三顆付費按鈕都用 `aria-describedby` 指到說明列／重試下方的原因，停用時仍念得到原因。）
- ✅ **Pre-existing failures: NONE new**（視覺那三張是已立案的 `preexisting-fail-visual-darwin-three-stale-baselines`）。
- **跟 story 不同的地方（有理由）**：
  - 估價服務持有的是一個窄介面 `transcriptionPlanSource`（`IsAvailable`／`translationEnabled`／`canResumeTranslateOnly`），不是直接寫 `*TranscriptionService` 型別——生產環境傳的就是 `*TranscriptionService`（有測試用型別斷言守著），介面只為了測試能控制三個答案。三個方法都是 run 自己用的那一份，沒有第二份判斷。
  - 觸發路由原本的「查片＋檢查檔案」搬成 `lookupMovieFile`／`lookupEpisodeFile` 兩個 helper，觸發與估價共用——這樣兩邊對「檔案不見」一定回同一個狀態碼、同一句話。觸發路由的檢查順序（先可用性閘門、再查片）沒變，既有 handler 測試全綠。
  - `GenerationProgressV2` 在 `onRetry` 有給但 `retryCost` 沒給時**不畫重試**（型別已經擋，這是執行期的第二道）。
- **視覺結果**：`ui-button-cost` 四種樣子與 J9-D B 欄一致；對話框按鈕是「生成字幕 $0.42」，說明列「語音辨識＋AI 翻譯，約需數分鐘」；失敗面板的重試是「重試 $0.39」。

#### 🎨 UX Verification（對照 `flow-j-specs/j9-d`、`flow-f-subtitle-v2/f1-d-v2`、`f2-d-v2`、`f1-m-v2`、`f4-d-v2` 與夾具截圖）

| 區域 | 設計稿 | 實作 | 一致？ | 處置 |
| --- | --- | --- | --- | --- |
| ① 可點 | 泥金底、「生成字幕」＋ `$0.42` 等寬、同色 | 同（`font-mono tabular-nums`，不另上色） | ✅ | — |
| ② ≈ | `≈ $0.42` | 同 | ✅ | — |
| ④ 載入中 | 金額位置 40×12 骨架、25% 透明 | `w-[5ch] h-3 opacity-25`（約 42×12） | ✅ | — |
| ⑤⑥ 停用 | `$bg-tertiary`＋髮絲框、字 `$text-disabled`、無金額 | 同 | ✅ | — |
| 按鈕高度 | 母版 36；F1-D-v2 上的 instance 47 | `min-h-[44px]` | ⚠️ | 觸控規格，AC #4 已定；不改 |
| 內距 | 母版 8／20、gap 8 | `px-5 gap-2` | ✅ | — |
| 說明列 | ①「語音辨識＋AI 翻譯，約需數分鐘」等六句 | 同（逐字，純函式 spec 守著） | ✅ | — |
| F2-D-v2 | 稿已改成語音辨識路線 | 程式碼本來就是這條路線 | ✅ | — |
| F4-D-v2 重試 | 在對話框 footer，旁邊有「稍後再試」 | 在失敗面板裡、沒有「稍後再試」 | ⚠️ | 版位差異早就存在，屬 `dsr-6-flow-f-subtitle-v2` 其餘範圍（已補註記）；本張只補金額 |
| F1-M-v2 手機 | 按鈕滿版 | 跟桌機一樣是內容寬 | ⚠️ | 同上，已記在 `dsr-6-flow-f-subtitle-v2` |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - `disc-2026-09-single-item-cost-estimate` → **AC #2, #3, #6**（本張主體）
  - `disc-2026-09-no-cost-bearing-component` → **AC #4, #5, #7, #8**（本張主體）
  - 設計稿 F2-D-v2／F1-M-v2 畫了按鈕不會走的抽取路線（🔴 #6）→ **AC #1**
  - 兩顆「重試」也會花錢但沒金額，F4-D-v2 同（🔴 #7）→ **AC #1, #5**
  - 金鑰未設定要按了才知道（🔴 #8）→ **AC #5**
  - 電影 `untranslated` 沒顯示「只翻譯」說明（🔴 #9）→ **AC #5**
  - ESLint Rule 21 名稱不收斜線（🔴 #10）→ **AC #4**
  - 送出中按鈕跳寬（🔴 #11）→ **AC #4**
  - 只差翻譯但沒有翻譯金鑰時會出現能按卻什麼都不產生的 `$0.00`（建單後驗證發現）→ **AC #5**（停用）＋ **AC #1**（J9-D §C 補充）
  - 影片檔讀不到時 ⑤「稍後再試」不是實話（建單後驗證發現）→ **AC #5** ＋ **AC #1**
  - `DESIGN.md:324` 的母版舊名（建單後驗證發現）→ **AC #9**

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**
  - **`disc-2026-09-asr-unavailable-copy-ignores-ffmpeg`**（/ship CR 立）— ⑥ 的說明只講 ASR 金鑰，但「不能用」也可能是缺 FFmpeg。
  - **`disc-2026-09-asr-route-translation-rate-unmeasured`**（/ship CR 立）— 翻譯費率可能不是在語音辨識路線量的，可能少報。
  - **`disc-2026-09-solo-run-unwritable-folder-pays-asr`**（建單時立）— 單片 run 在語音辨識**付完錢之後**才寫 `.en.srt`（`transcription_service.go:648, 656-658`），資料夾不能寫就白付；批次管線有 sub-6-1 的寫入預檢，這條路沒有。
  - **`disc-2026-09-solo-run-timeout-shorter-than-translation`**（建單時立）— 單片與批次的 `RunTranscription` 都用 `s.timeout = 5 分鐘`（`:176, 366, 400`），但 eval-1 量到 Sonnet 翻譯要片長的 17%（`generation_candidates.go:218-221`）——45 分的集數翻譯約 7.6 分鐘。推算，未在 NAS 實測。

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/api/internal/services/transcription_estimate.go`（＋`_test.go`）
- `apps/api/internal/handlers/transcription_estimate_handler_test.go`
- `apps/web/src/components/ui/ButtonCost.tsx`（＋spec）
- `apps/web/src/components/subtitle/generateCostView.ts`（＋spec）
- `apps/web/src/hooks/useTranscriptionEstimate.ts`（＋spec）
- `tests/visual/components.visual.spec.ts-snapshots/components/ui-button-cost/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6a-cost-on-paid-generate-buttons.md`（本檔）

**修改：**
- `apps/api/internal/services/transcription_service.go` — 抽出 `translationEnabled()`，`translateAndPersist` 改呼叫它
- `apps/api/internal/handlers/transcription_handler.go` — `SetEstimator`、兩條 GET、`lookupMovieFile`／`lookupEpisodeFile` 共用
- `apps/api/cmd/api/main.go` — `routePredictor` 提成變數、估價服務接線
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`（＋spec）— 三顆付費按鈕、說明列、重新估價、關閉時移除價錢、拿掉 `useKeySettings`、檔頭
- `apps/web/src/components/subtitle/GenerationProgressV2.tsx`（＋spec）— 重試改 `ButtonCost`、`retryCost`／`retryNote`
- `apps/web/src/services/transcriptionService.ts` — `TranscriptionEstimate` 型別、`getTranscriptionEstimate`
- `apps/web/src/lib/currency.ts`（＋spec）— `usdWithEstimate` 搬進來
- `apps/web/src/components/subtitle/consent/consentSelection.ts` — re-export `usdWithEstimate`
- `apps/web/src/eslint-rules/implements-pen-node-id.js`（＋spec）— 名稱收斜線
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 新夾具、對話框預塞價錢、失敗面板補 `retryCost`
- `tests/visual/components.visual.spec.ts-snapshots/components/{subtitle-manage-subtitle-dialog-v2,generation-progress-v2/失敗}/default-visual-darwin.png`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f1-m-v2,f2-d-v2,f4-d-v2}.png`、`_bmad-output/screenshots/flow-j-specs/j9-d.png`
- `DESIGN.md` — 母版名稱、現況段落
- `project-context.md` — Rule 21 `{Name}` 可含 `/`
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 本張 → review；兩張 disc → done；`dsr-6` 註記；/ship CR 立的兩張 backlog

**刪除：**
- `tests/visual/components.visual.spec.ts-snapshots/components/{subtitle-manage-subtitle-dialog-v2,generation-progress-v2/失敗}/default-visual-linux.png`（改動過的畫面，交給 CI bootstrap 重產）

**AC drift reference（未修改）：** `sub-2-2d-f5-cta-code-strings.md`、`sub-2-2b-untranslated-badge-frontend.md`、`9R-UX-episode-row-cta-design.md`、`sub-6-12-consent-money-traps.md`

## 對抗式 Code Review（/ship，2026-09-17）

獨立 reviewer（fresh context，只讀；自己跑過 Go 三個 package 與 7 支相關 spec）回報 **0 HIGH、1 MED、9 LOW、3 項無法驗證**。**修 7、立案 2、不修 2**。每一條修正都先寫會紅的測試。修完後 web **3551 / 3551**、api PASS、lint 0 errors、typecheck PASS。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| M1 | MED | 在同一個對話框裡下載線上字幕後，片子從 `untranslated` 變 `found`，但估價還在快取裡——按鈕仍是便宜的「只翻譯」價，按下去卻跑完整語音辨識（**唯一一條少報價錢的路**） | 下載成功立刻重新估價；`subtitleStatus` 在對話框開著時改變也重新估價；兩條測試 |
| L1 | LOW | ⑥ 的說明叫人去設 ASR 金鑰，但「不能用」也可能是缺 FFmpeg（自架語音辨識的人去設定頁修不好） | 同一句話以前就是 503 面板的文案，本張只是提早顯示；立 `disc-2026-09-asr-unavailable-copy-ignores-ffmpeg`（要 Sally 定文案） |
| L2 | LOW | 背景重抓說「讀不到檔案」時，「先看資料」讓舊價錢留著、按鈕還能按 | 檔案不見是確定的答案，覆蓋舊價錢；測試 |
| L3 | LOW | 兩顆重試在「只能產生英文」（含 `$0.00`）時沒有說明 | 重試下方也顯示那句；測試 |
| L4 | LOW | 失敗面板的重試沒有 busy 狀態，連點會送兩次 | `retryBusy`；測試 |
| L5 | LOW | 按鈕寬度仍會跳：骨架的 `5ch` 不是用等寬字量的；停用態多 1px 框 | 骨架加 `font-mono`；每個狀態都有框（非停用時透明）；測試＋三張基準線重拍 |
| L6 | LOW | `main.go` 兩段註解過時／理由寫錯 | 改寫 |
| L7 | LOW | 關對話框取消請求時，每次記一筆 Warn「duration probe failed」 | 請求已取消時改記 Debug；測試。查電影失敗記 Error 與觸發路由一致，不改 |
| L8 | LOW | 測試強度：失敗後重新估價斷言的是同一個數字；`translationEnabled()` 為真的分支沒測；續跑判斷只用 stub | 失敗後改成不同價錢並斷言新數字；補真假兩個分支；補一條接真正 `TranscriptionService`（影集、英文 SRT 在／不在）的測試 |
| L9 | LOW | 關閉對話框的離場動畫中，按鈕會先閃成骨架 | **不修**：只在淡出的那一瞬間；改成「開啟時才移除」會讓上一次的舊價錢先出現，更糟 |
| U1 | — | 翻譯費率可能是在抽取路線量的，語音辨識路線的翻譯可能被少報（候選清單同一個函式，既有問題） | 立 `disc-2026-09-asr-route-translation-rate-unmeasured`（先查 eval-1 紀錄） |

查過不成立：扣款模型（單片觸發不帶 `WithModelID`，holder 退回 `EffectiveModel()`，與估價一致）、自架語音辨識的判斷來源（ASR holder 也是開機時的 `cfg.ASRBaseURL`）、續跑判斷兩邊只差一次 ReadFile、四格金額都走既有函式、handler 重構順序與字句逐字相同、`SetEstimator` 在 `RegisterRoutes` 之前、沒有「能按但沒金額」的狀態、`aria-describedby` 參照的元素一定存在、TanStack 的 `removeQueries`／`invalidate` 不會造成重抓迴圈、視覺夾具用 app 的 queryClient（5 分鐘 staleTime，CI 沒有後端也是 `$0.42`）、ESLint 新 regex 嚴格包含舊的、`usdWithEstimate` 沒有循環 import、文案逐字相符。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | ✅ 收單 —— PR #455 合併進 main（commit 7340763b），**CI 17 項全綠**（含 4 個 e2e shard 與 4 個視覺 shard）。`-linux` 基準線：手動觸發 Visual Regression → bootstrap PR #456（3 張 linux）合回分支後轉綠；看過其中兩張，對話框是「生成字幕 $0.42」、不是骨架。 |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／1 MED／9 LOW，修 7、立案 2、不修 2。最重要的一條：在同一個對話框裡下載線上字幕之後，按鈕還留著「只翻譯」的便宜價，按下去卻會跑完整語音辨識——現在下載成功、或片子的字幕狀態一變，就立刻重新估價。另外：讀不到檔案時不再讓舊價錢留著；兩顆重試在只能產生英文時也寫出原因；失敗面板的重試不能連點；按鈕在各狀態之間不再變寬。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。api PASS、web 3545/3545、lint 0 errors、typecheck、prettier、token 一致；視覺比對只剩三張既有紅。管理字幕的「生成字幕」和兩顆「重試」第一次在按下去之前就寫出要花多少錢：後端新增單片估價（估的是這顆按鈕真的會做的事），前端新增 `ButtonCost`（有價錢／估價中／停用三種樣子），算不出價或沒有語音辨識金鑰時按鈕停用並寫出原因。設計稿 F2-D-v2／F1-M-v2 改回語音辨識路線、F4-D-v2 重試帶金額、J9-D 補上現況與三種補充情況。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀，抽查約 30 個行號全對）：4 項 CRITICAL、16 項 SHOULD FIX，**全部併入**。最重要的四項：① 照原本的 staging 規則會漏 commit `pen-tokens.json`，CI 的 token 檢查必紅（補進 AC #1）；② 估價查詢設 `staleTime: 0` 會讓視覺夾具在沒有後端的 CI 上重抓失敗，截圖時好時壞（改成預設 staleTime＋關閉時移除快取＋「先看資料再看錯誤」）；③ 「只差翻譯但沒有翻譯金鑰」會出現一顆能按、`$0.00`、按了什麼都不會產生的按鈕（改成停用）；④ 兩顆重試停用時沒有地方寫原因（按鈕下方補一句）。另外：金額改成呼叫既有 `estimateUSD`／`roundUSD` 組出來、`estimated_usd` 定為 JSON number、「有沒有翻譯」抽成一個方法兩處共用、handler 用 setter 注入（不動 16 個建構子呼叫點）、`routePredictorAdapter` 提成變數、模型測試改用 Haiku（用目錄外 id 測不出差別）、`usdWithEstimate` 必須 re-export、ESLint regex 改嚴格版、讀不到檔案的專用句子、`DESIGN.md:324` 母版舊名、sub-4-1 AC #7 契約 ack、`aria-describedby`、影集查詢一般錯誤回 500、補 Pencil 子節點 id。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。由 `dsr-6-flow-f-subtitle-v2` 拆出的第一張，合併 `disc-2026-09-single-item-cost-estimate` 與 `disc-2026-09-no-cost-bearing-component`（皆 P0，⚖️ Alexyu 2026-09-17）。Pencil MCP 讀出 ButtonCost 三個母版、J9-D、F1-D-v2／F2-D-v2／F1-M-v2／F4-D-v2；盤點單片 run、候選估價、模型、片長、對話框三顆付費按鈕。新立兩個 backlog。 |
