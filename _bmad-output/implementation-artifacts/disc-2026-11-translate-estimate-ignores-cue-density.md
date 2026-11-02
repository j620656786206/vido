# Story disc-2026-11-translate-estimate-ignores-cue-density：英文字幕翻譯的預估改用「句數」算，不再偏低 2.5 倍

Status: done

**Source:** sprint-status 同名待辦（P2，PR #948 建單）。⚖️ Alexyu 2026-11-02「接著做英文字幕預估偏低那張」。待辦寫「修法方向要 Bob 定」——決定寫在下面。

## 問題

《天蠍行動》S01E01–E03 翻片內英文字幕（claude-sonnet-5），預估每集 $0.19，實際 $0.44、$0.49、$0.44。預估是「片長 × 每分鐘費率」（`services/generation_candidates.go` `translationUSDPerMinuteByModel`，sonnet-5 = $0.00462／分鐘，eval-1 量的、那批片平均每分鐘約 14 句），但花費跟著**句數**走：天蠍每分鐘約 36 句。

正式機資料（2026-11-02，sonnet-5、translate、完成、沒有快取、句數 >100）：
- **每千句**：30 筆，平均 US$0.393，大多 0.28～0.50，p75 約 0.44，最高 0.71。
- **每分鐘**（有片長的 23 筆）：平均 US$0.0072（範圍 0.005～0.0118），句數每分鐘 13～37。程式寫的 0.00462 平均就低了 1.56 倍。

## 決定（Bob）

1. **能知道句數就用句數算**：預估 = 句數 × 每千句費率。
2. **句數從哪來（都不花錢、不解碼）**：
   - 片內文字字幕軌：ffprobe 的軌道標籤 `NUMBER_OF_FRAMES`（mkvmerge 寫的統計標籤，也可能是 `NUMBER_OF_FRAMES-eng`）；MP4 用 stream 的 `nb_frames`。《天蠍》E2 英文軌標籤 1,609、實際翻 1,502 句。
   - 片旁英文 .srt：直接數字幕條數。
   - 都拿不到才退回「片長 × 每分鐘費率」。
3. **費率寧可高一點，不能低**（成本確認畫面的原則：絕不低報）：
   - 每千句：sonnet-5 用 **US$0.45**（正式機 p75）。其他模型照現有做法，用 sonnet 的費率乘上「混合價格比」換算（`ratePerMinuteFrom` 那套）。
   - 退回用的每分鐘費率：sonnet-5 改成 **US$0.0085**（正式機分布偏高的那段），其他已量測的模型同比例上修，未量測的照舊用 anchor 換算。
4. 只改**翻英文字幕**這條路（extract／sidecar）。語音辨識那條（asr）不在本張：它的句數要聽完才知道。
5. 估價 API 的回傳欄位不改；可以在既有欄位允許的範圍內多帶「這次用句數還是片長算」的內部 log，不新增對外欄位。

## Acceptance Criteria

1. [x] 路線判斷（`subtitle/predict_route.go` 的 Probe／FromTracks）能把選中的英文文字軌的句數帶出來（標籤或 nb_frames；讀不到就 0）；片旁 .srt 路線能數條數。
2. [x] 估價（單集 `TranscriptionEstimateService`、批次 `GenerationCandidateService`、開跑時記帳的 `runStartEstimate`——三者共用同一套）有句數時用句數 × 每千句費率，沒有時用新的每分鐘費率。
3. [x] 費率表：sonnet-5 每千句 0.45、每分鐘 0.0085；其他模型按比例；測試鎖住數字與換算。
4. [x] 驗證：用《天蠍》E2 的實際標籤 1,609 句 → sonnet-5 預估約 US$0.72（≥ 實際 $0.49，不低報）；30 筆正式機資料用新算法，低報的筆數與比例寫進 Dev Agent Record。
5. [x] `go build／vet／test`、`staticcheck`、`gofmt` 乾淨；前端不改。

## Dev Agent Record

### 改了什麼

- **句數從 ffprobe 拿**（`services/ffprobe_service.go`）：`SubtitleTrack` 加 `CueCount`（JSON `cue_count,omitempty`）。解析 ffprobe 時讀軌道標籤裡任何 `NUMBER_OF_FRAMES` 開頭的鍵（含 `NUMBER_OF_FRAMES-eng`，不分大小寫），沒有就用 stream 的 `nb_frames`（MP4）；都不是正整數就 0。不解碼、不多跑 ffprobe。舊資料沒有這個欄位讀成 0，0 不會寫進 JSON，所以舊列與沒標籤的軌道存起來一個字都不變。
- **路線判斷帶出句數**（`subtitle/predict_route.go`）：新增 `PredictCuesFromTracks`（`SelectCandidates` 選出的軌道裡最大的句數——管線可能用正片軌或 SDH 軌，取大的才不會低報）和 `Router.PredictRouteWithCues`（同一次 probe 回路線、片長、句數；`PredictRouteWithDuration` 改成呼叫它，probe 只有一條路）。`cmd/api/route_predictor_adapter.go` 實作新的選用介面 `services.RouteCuePredictor`（`CuesFromTracks`／`ProbeWithCues`），並加編譯期檢查。
- **費率與算法**（`services/generation_candidates.go`）：
  - 新表 `translationUSDPer1kCuesByModel`（sonnet-5 0.45、haiku 0.293）＋ anchor 0.782，走同一個 `ratePerMinuteFrom`（有量測用量測，沒有就 anchor × 混合價格比，沒價格列就 anchor）。
  - 每分鐘表（退回用）：sonnet-5 0.0085、haiku 0.00554、anchor 0.0148。
  - `extractTranslationUSD(minutes, cues, model)`：有句數用句數 × 每千句費率，否則片長 × 每分鐘費率。`estimateUSDByCues` 是帶句數的版本，`estimateUSD` 維持原簽名（= 句數 0）；`estimateUSDWithCredit` 多一個 cues 參數。語音辨識那條忽略句數。
- **三個估價共用同一套**：
  - 批次 `GenerationCandidateService`：`classify` 多回傳句數（存起來的字幕軌用 `CuesFromTracks`，要 probe 的用 `ProbeWithCues`），每個模型的價錢都用同一個句數。
  - 路線快取（`services/route_cache.go`）：值改成 `"extract|1609"`（沒句數就照舊只有路線），`routeVersion` 1 → 2，舊的快取全部作廢、重新 probe 一次——不然 30 天內快取命中的集數都會照片長算。讀到看不懂的值（未知路線、壞掉或負的句數）當沒命中。
  - 單集 `TranscriptionEstimateService`：存著的字幕軌、片長那次 probe、補的路線 probe 都帶出句數；片旁英文 .srt 路線（plan `sidecar`）讀那個檔數條數（只讀檔、上限 8MB，讀不到就照片長）。開跑時記帳的 `runStartEstimate` 走的就是這個 `Estimate`，自動一樣。
  - 多一行 Debug log「translation quote basis」（basis = cues／runtime、句數、片長、模型、金額），不新增對外欄位。

### 決定與跟 Bob 決定的對照（請確認）

Bob 定了 sonnet-5 的兩個數字（每千句 0.45、每分鐘 0.0085）；其他模型我照「同比例」的原則推，有兩處跟字面不完全一樣，理由都是「絕不低報」：

1. **Haiku 每千句 0.293，不是 sonnet × 價格比的 0.15**。Haiku 每次只送 10 句，每批的系統提示要多付五倍次數，eval-1 量到它是 Sonnet 的 0.65 倍（不是價格比的 0.33 倍）；用價格比會比它自己量到的成本還低。做法：Sonnet 從量測值（See S01E04–06：每千句 0.33269）到 0.45 是 ×1.3526，Haiku 的 eval-1 量測值（每千句 0.21632）也 ×1.3526 → 0.2926，無條件進位 0.293。
2. **anchor 也跟著上修**：每分鐘 anchor 0.00804 → **0.0148**（同 ×1.8398）；每千句 anchor 用 eval-1 量到的 Sonnet 10 句一批（每千句 0.57754）×1.3526 → **0.782**。原本程式的規則是「沒量測過的模型一律用 Sonnet 10 句一批的成本當基準，不能用 50 句一批的低價」；如果 anchor 留在 0.00804，它會比 Sonnet 5 新的 0.0085 還低，Sonnet 4.6（同價、10 句一批）會被報得比 Sonnet 5 便宜，舊測試「unknown 必須高於 Sonnet」也會失敗。所以我把「未量測的照舊用 anchor 換算」讀成「換算方法照舊、anchor 值一起修正同一個句數密度誤差」。Opus 每千句 1.30、每分鐘 0.0247。
- Haiku 每分鐘照 Bob 的「同比例上修」：0.00301 × 1.8398 = 0.005538 → 0.00554。

### ⚠️ 還沒涵蓋的：已經存起來的字幕軌沒有句數

- 批次清單讀的是集數／電影**已存的** `subtitle_tracks`（集數由背景清點、電影由掃描時寫），那些是在這張之前 probe 的，沒有 `cue_count`，所以會照**新的每分鐘費率**算，直到那個檔重新 probe（換檔、或新掃到）。以《天蠍》為例，批次清單會是 41 分鐘 × 0.0085 ≈ $0.35，仍低於實際 $0.44–0.49；單集對話框（集數一定會 probe）和開跑記帳則會是 $0.72。
- 建議另開一張：讓集數字幕軌背景清點對「有英文文字軌但沒有句數」的集數重新 probe 一次（每個行程一次，避免沒標籤的檔一直重讀），電影的補讀同理。本張沒做，因為不在 AC 裡、也牽涉清點服務的判斷。
- 另外：花費摘要的「略過估算省下」也是用每分鐘費率算（`subtitle_spend_service.go`），費率從 0.00462 變 0.0085 後，這個「省下」的估算會跟著變大約 1.84 倍；它是估算、本來就寫明，但數字會變，告知一下。

### AC #4 驗證

- 《天蠍》S01E02：標籤 1,609 句 × $0.45/1k = $0.724 → **$0.72**（實際 $0.4859，不低報；舊算法 $0.19）。
- 正式機 30 筆（2026-11-02 唯讀查 `/mnt/cache/appdata/vido/vido.db`：sonnet-5、translate、完成、沒有快取、句數 >100），用「實際翻的句數」× 0.45 算（容器標籤通常比實際翻的多，真正的報價會更高）：**低報 5 筆，16.7%**——《末日光明》S1E2、S1E3（這兩筆是關思考之前的舊執行，每千句 0.60、0.71）、《和平使者》S1E2、S1E3、S1E5；最多低報 $0.11。其餘 25 筆都不低於實際。
- 同一批有片長的 23 筆：舊的每分鐘 0.00462 **23 筆全部低報**；新的退回費率 0.0085 低報 3 筆（《天蠍》三集，正是句數算法要接手的）。
- 這 30 筆寫成 table-driven 測試鎖住（`TestCueEstimate_ProductionRuns`）。

### 測試證據

- `services/translate_estimate_cues_test.go`（新）：每千句費率與換算（sonnet 0.45、haiku 由量測推、anchor、Opus／Sonnet 4.6／未知模型都高於 sonnet-5）；有句數用句數、沒有用片長、語音辨識不吃句數；《天蠍》E2 = $0.72；30 筆正式機資料；批次：存著的字幕軌與 probe 各自帶句數、存著的不重 probe、沒句數退回片長、快取存 `extract|1609` 且第二次掃描命中時金額一樣；快取值編解碼（含壞值）與版本 = 2；單集：三種來源的句數、沒句數退回、語音辨識不受影響、片旁 .srt 數條數（讀不到退回片長）；ffprobe：`NUMBER_OF_FRAMES`、`-eng`、`nb_frames`、標籤優先、沒有／亂寫／0；舊 JSON 讀得回來、0 不寫進 JSON。
- `services/generation_candidates_test.go`：改寫每分鐘費率的測試（新數字＋比例＋anchor 仍高於 sonnet）。`generation_candidates_resume_test.go`：補 cues 參數。
- `subtitle/predict_route_test.go`：`PredictCuesFromTracks`（取最大、圖片軌／片旁檔／und 不算、中文優先跟執行時一樣）、`PredictRouteWithCues` 一次 probe、不抽取、錯誤會回傳。
- `cd apps/api && go build ./... && go vet ./... && go test ./...`：44 個套件全 ok；`~/go/bin/staticcheck ./...` 無輸出、exit 0；改到的檔案 `gofmt -l` 乾淨。前端沒有改。

### Bob 覆核（2026-11-02）

- 接受 dev 推的兩個費率：Haiku 每千句 0.293（用它自己 eval-1 量測值同倍數上修，因為它每批只送 10 句、照價格比會低報）、未量測模型的基準每分鐘 0.0148（跟著上修，維持「未知模型不低於 Sonnet」）。
- 批次清單對舊資料要等重新 probe 才有句數 → 另開 `disc-2026-11-backfill-subtitle-cue-counts`。

## Change Log

- 2026-11-02：建立（Bob 決定費率與句數來源）。
- 2026-11-02：Amelia 實作完成（句數來源、新費率、三個估價共用、路線快取 v2），送 review；其他模型費率的推法與「已存字幕軌沒有句數」待確認。
