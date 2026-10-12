# Story disc-2026-09-generation-resume-c-estimate-deduction：被預算擋下的片，再按「生成字幕」時，估價扣掉已經付過的部分

Status: done

**Source:** 2026-09-18 SM 拆 `disc-2026-09-generation-resume-from-checkpoint` 時建的第三張（A 翻譯續跑 #479、B 聽寫續跑 #477 都已上線）。2026-10-12 Alexyu 從 backlog 選定。

## 查到的事（2026-10-12，Bob 逐一開檔確認，main `09784cc4`）

### 現在的估價

- 「管理字幕」對話框的價錢來自 `GET /movies/:id/transcribe/estimate`、`/episodes/:id/transcribe/estimate` → `TranscriptionEstimateService.Estimate`（`apps/api/internal/services/transcription_estimate.go:158`）。
- 價錢 = 片長分鐘 × 費率（`priceUSD`，`:211`）。完全不看「之前做到哪」：
  - `full`（聽寫＋翻譯）：分鐘 × (聽寫費率＋翻譯費率)。
  - `translate_only`（之前已經聽寫完、英文字幕還在，只差翻譯）：分鐘 × 翻譯費率。
- 結果：一部 57 分鐘的片聽寫到第 9 段（共 16 段）被擋下，再打開對話框還是報全價。實際上那 8 段不用再付錢，按下去實際花的比報的少很多。你看到全價，可能就不按了。

### 已經存著的進度

- **聽寫（B，#477）：** 每段聽寫完就存在 `cache_entries`（`asr_chunk_store.go`）。另有一份 manifest（清單）記錄這部片哪些段已經存了：
  - key = `asrManifestKey(mediaID, 檔案 size, mtime, 聽寫端點指紋)`
  - 內容 = `{track, lang, chunk_seconds, done: [段起點秒數…]}`
  - 段的 key 可以從 manifest 算出來（`clearChunkCache` 就是這樣算，`transcription_service.go:1640-1660`）。
  - 現在切段會對齊靜音（disc-2026-10-asr-chunk-at-silence），但 key 和 manifest 還是用格子起點 `i × chunk_seconds`。
- **翻譯（A，#479）：** 每句翻完就存，key = `segkey.SegmentKey(英文句子, RunVersion)`。`RunVersion` = 片子資料 hash＋名詞表 hash＋提示詞版本＋模型（`translationRunVersion`，`transcription_service.go:1756`）。`translateSRT`（`:1981`）在開跑前一次 `GetMany`，命中的句子不送模型。
- 這兩份進度 30 天後過期；做完整部會刪掉聽寫段（翻譯句子留著）。

### 估價和實際要一致

- 估價說「這幾句已經翻過」，實際跑的時候必須真的命中。所以估價算 key 的材料要和實際跑的時候**同一段程式**算出來，不能另寫一份。

## Story

身為被預算擋下過的使用者，
我希望再打開「管理字幕」時，價錢只算還沒做的部分，
這樣我看到的數字接近實際會花的錢，不會因為報全價就放棄。

## 拆單（2026-10-12）

原單有兩半：單部估價、批次同意清單。同意清單那半要讓 `GenerationCandidateService` 每部片多做一次檔案 stat＋快取查詢，清單上還要加一行說明（要 Sally 看）。依「該拆就拆」拆出 `disc-2026-09-generation-resume-c2-consent-list-deduction`（backlog）。**這張只做單部估價**，純後端。

## Acceptance Criteria

1. **聽寫扣減（plan=`full`）：** 估價讀這部片的 manifest，再一次 `GetMany` 確認 manifest 列的段真的還在。
   - 已完成分鐘 = 還在的段數 × `chunk_seconds` ÷ 60；`chunk_seconds` 為 0（整檔一段）時，有存就等於整部。
   - 上限是片長（最後一段可能比較短）。
   - 價錢 = (片長 − 已完成分鐘) × 聽寫費率 ＋ 片長 × 翻譯費率。翻譯還沒開始，所以照全片算。
2. **翻譯扣減（plan=`translate_only`）：** 讀那份英文字幕，切成句子，用**跟實際翻譯同一個函式**算出 RunVersion 和每句的 key，一次 `GetMany`。
   - 命中比例 = 命中句數 ÷ 總句數。
   - 價錢 = 片長 × 翻譯費率 × (1 − 命中比例)。
3. **回應多兩個欄位（只加不改，`[@contract-v2]` 加寬）：**
   - `resumed_asr_minutes`（float，已完成的聽寫分鐘，沒有就省略）
   - `cached_translation_share`（float 0–1，翻譯已有的比例，沒有就省略）
   - `runtime_minutes`、`plan` 的意思不變。前端不用改。
4. **一致性測試：** 同一份英文字幕、同一部片，估價算出的翻譯 key 集合 == `translateSRT` 實際 `GetMany` 的 key 集合。
5. **任何讀取失敗都當成「沒有進度」，報全價**（往高估，不會讓使用者嚇到）。估價本身不能因為快取壞掉而失敗。
6. **`extract` plan 不動**（片內字幕軌的翻譯走字幕管線，不在這張範圍）。沒有聽寫快取或翻譯快取接上時（測試、舊設定），行為跟現在一樣。
7. **檢查：** `go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 `TranscriptionService`：把 `translateSRT` 裡算 glossary／metadata／level／RunVersion 的部分抽成一個函式，實際翻譯和估價共用（AC #2、#4）
- [x] T2 `TranscriptionService.resumeCredit`：聽寫已完成分鐘（manifest＋段存在檢查）、翻譯命中比例（AC #1、#2、#5）
- [x] T3 `TranscriptionEstimateService`：介面加 `resumeCredit`、`priceUSD` 扣減、回應兩個新欄位（AC #1–#3、#6）
- [x] T4 測試：扣減計算、讀取失敗報全價、一致性測試（AC #4、#5）
- [x] T5 檢查全綠（AC #7）

## Dev Notes

- 聽寫段只有「切點跟這次一樣」才會被重用（`transcribeAudio` 比 `StartMS`／`DurationMS`）。估價時不切音訊，所以假設切點一樣（同一個檔案、同一套靜音偵測，結果是固定的）。真的不一樣時，實際花的錢會比估價多，但最多回到全價。
- manifest 不分音軌和語言（key 只有檔案和端點），內容才記。估價不知道這次會選哪條音軌，直接用 manifest 記的值算段的 key。音軌選擇對同一個檔案是固定的。
- 費率不重寫，沿用 `ai.EstimatedASRPerMinute`、`asrLegTranslationRatePerMinute`。

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- `transcription_service.go`：把 `translateSRT` 裡算 glossary／metadata／level／RunVersion 那段抽成 `translationInputsFor`，回傳 `translationInputs`。翻譯本身只是改成呼叫它，行為不變。
- 新檔 `transcription_resume_credit.go`：
  - `resumedASRMinutes`：讀 manifest → 用 manifest 記的 track／lang／chunk_seconds 算出各段 key → 一次 `GetMany` 數還在的段 → 段數 × chunk_seconds ÷ 60，上限片長；chunk_seconds 為 0 時整部。
  - `cachedTranslationShare`：讀 `resumeSource` 那份英文字幕 → `ParseSRTToTranslationBlocks` → `translationCacheKeys(blocks, translationInputsFor(...))` → 一次 `GetMany` → 命中比例。
  - 任何失敗都回 0（報全價），只記 Debug。
- `transcription_estimate.go`：
  - `transcriptionPlanSource` 介面加這兩個方法。
  - `Estimate`：plan=full 且聽寫可用時問聽寫進度；plan=translate_only 且有翻譯時問翻譯進度。翻譯那個用 `ai.WithModelID(ctx, model)`，跟報價用的模型一致（key 裡有模型）。
  - `priceUSD` 多一個 `resumeCredit` 參數：聽寫分鐘只扣聽寫那半；翻譯比例只扣 translate_only。沒有進度時走原本的 `estimateUSD`，跟同意清單的價錢一樣。
  - 回應加 `resumed_asr_minutes`、`cached_translation_share`（omitempty）。

### 驗證

- 新測試檔 `transcription_resume_credit_test.go`：
  - 價錢：聽寫做一半 $1.66 → $1.48、只要英文的、聽寫全做完只剩翻譯、翻譯 75% 已有 → $0.32、全部已有 → $0、沒翻譯 key／聽寫不能用／extract 路線都不去問。
  - `resumedASRMinutes`：只算還在的段、上限片長、短檔一段、key 用 manifest 的音軌語言、各種讀不到都是 0、檔案重新編碼就不算。
  - `cachedTranslationShare`：用跑的那個版本算命中、別的模型的不算、讀不到都是 0。
  - **一致性（AC #4）：** 估價查的 key 列表 == `translateAndPersist` 實際查的 key 列表。
- 突變檢查：拿掉「段還在」的檢查 → 3 條紅；估價的 key 多加一個空白 → 3 條紅（含一致性那條）。
- `go vet ./...`、`~/go/bin/staticcheck ./...`、`go test ./...` 全綠；gofmt 乾淨。

### 範圍外（沒修）

- 同意清單那半 → `disc-2026-09-generation-resume-c2-consent-list-deduction`。
- `extract` 路線（翻片內字幕軌）的快取扣減：走字幕管線，不在這張。
- 估價不切音訊，假設切點跟上次一樣。真的不一樣時，實際花費會比估價高，但不會超過全價。
- 前端沒有顯示「已扣掉多少」，只是價錢變低。要不要多一行說明，跟 c2 一起請 Sally 看。
