# Story（後端）：每一集算出「有沒有中文字幕」，片內字幕掃一次存起來

Status: review

**Source:** 傘狀單 `disc-2026-10-episode-list-subtitle-badge`（Alexyu 2026-10-05 party mode 裁定；開工條件「看完 S01E02」已在 2026-10-06 達成，見 `eval-see-s01e02-asr-vs-official.md` 狀態列）。前端半張見 `-b`（依賴本單的 AC #1 契約）。
**設計稿：** `J11-D`（`w2Opax`）規則 1、2 → `_bmad-output/screenshots/flow-j-specs/j11-d.png`。

> ⚖️ **SM 裁定（Bob 2026-10-07）：拆成 -a 後端／-b 前端。** 原單後端要補 migration、掃描後寫入、可中斷續跑的背景補齊、API 契約；前端要改圖示、補 Tooltip 的觸控行為、E2E 與視覺基準——合起來比上一張同類單（`disc-2026-10-subtitle-filter-disagrees-with-badges`）大一倍以上。切分線選「API 契約」：-a 出貨後季清單 API 就多兩個欄位（前端暫時不讀，不壞任何畫面），-b 再把圖示接上。依 `feedback_split_oversized_stories.md`。
>
> ⚖️ **SM 裁定：結論用既有的五值，不新增一套。** 原單 AC #1 的 `zh_hant／zh_hans_only／no_chinese／unknown` 是在 `models.ChineseSubtitleVerdict` 存在之前寫的；Dev Notes 的下游依賴已要求「用同一個判斷函式」。所以每集的結論就是 `chinese_subtitle`，值與電影／影集相同（`zh_hant／zh_hans／zh／none／unknown`）。多出來的 `zh`（中文、分不出繁簡，例如片內 `chi` 軌沒有標題）J11-D 沒有畫 → 交給 -b 找 Sally 定圖示。
>
> ⚖️ **SM 裁定：旁邊的字幕檔「現讀＋寫回」。** J11-D 規則 2 要求打開一季時才讀資料夾；下游 `disc-2026-10-subtitle-filter-series-phase-2` 要求結論存進資料庫（SQL 篩選才看得到）。兩個都要：掃描後的背景工作把片內＋旁邊的結果一起存進 `episodes.subtitle_tracks`；打開一季時再讀一次資料夾，跟存的不一樣就寫回。長解，不是二選一（`feedback_architecture_prefer_long_solutions.md`）。

## Story

身為 Vido 的使用者，
我希望 Vido 記得每一集片子裡有哪些字幕，打開一季時再看一眼旁邊的字幕檔，然後告訴前端「這集有繁中／只有簡中／缺中文／還不知道」，以及是根據哪幾個來源，
這樣季清單（-b）才能每集標出正確的圖示，媒體庫的影集篩選（phase-2）之後也有資料可篩。

## 查到的事（2026-10-07，逐一開檔確認）

- 單集沒有任何軌道欄位：`models/episode.go:9-43`；`episodeSelectColumns`（`episode_repository.go:19-23`）。電影的 `subtitle_tracks` 是 JSON 陣列（`services.SubtitleTrack`：片內＋旁邊檔），讀出時用 `ChineseSubtitleVerdict` 算 `chinese_subtitle`（`movie_repository.go:744`）。
- `ChineseSubtitleVerdict(status, language, tracks)`（`models/chinese_subtitle.go:272`）：有中文證據 → 有；軌道可讀且全是已知非中文（含 `[]`）→ `none`；其餘 `unknown`。**`subtitle_tracks` 為 NULL＝沒讀過。**
- 旁邊的字幕檔：`subtitle.ListSidecars`（`inventory.go:128`）一次 `ReadDir`、中文看內容前 100KB；每次呼叫各讀一次資料夾。
- Rule 19：`services` 不能 import `subtitle`；`cmd/api/*_adapter.go` 是橋接層（前例 `asr_adapter.go`）。
- 掃描：影集檔走 `processTVFile` → `MediaIngestService.IngestEpisodeFile`，沒有 ffprobe、沒有大小／修改時間比對（電影才有，`scanner_service.go:660-661`）。掃描完成可以 `AppendOnScanComplete` 掛工作（`main.go:1069` 前例）。
- 開機後一次性背景工作的前例：`CreditsBackfillService.RunAfter`（`credits_backfill_service.go`，`main.go:725-728`）。
- `FFprobeService.Probe` 自帶併發上限（semaphore）與逾時（`ffprobe_service.go:105-130`）。
- 候選分析：單集的 `tracksJSON` 永遠是空的（`generation_candidates.go:1103-1105`），所以每次估價都對每集跑一次 ffprobe（只有路線快取擋著）。電影已經用存好的軌道走免探測的路（`:1245`、`:1325`）。單集時長只在探測時回寫（`:1365`）。

## Acceptance Criteria

1. **[@contract-v1] 季清單 API 每集多兩個欄位**（`GET /api/v1/series/:id/seasons/:n/episodes`，只在 `has_local_file=true` 的集數出現）：
   - `chinese_subtitle`：`zh_hant｜zh_hans｜zh｜none｜unknown`，與電影／影集同一個列舉、同一個判斷函式。
   - `chinese_subtitle_sources`：陣列，列出**是中文的**來源，每項 `{kind, language, label}`：`kind`＝`embedded`（片內）／`sidecar`（旁邊的檔）／`vido`（Vido 自己放的那份）；`language`＝`zh-Hant｜zh-Hans｜zh-unknown`；`label`＝片內軌的標題、旁邊檔的檔名標籤（`See.S01E02.zh-TW.srt` → `zh-TW`），沒有就空字串。沒有中文來源時是 `[]`。
2. **判斷規則**：片內（存好的）＋旁邊（現讀）＋Vido 的紀錄（`subtitle_status`／`subtitle_language`）一起交給 `ChineseSubtitleVerdict`。**片內還沒讀過時**，只有在旁邊或 Vido 紀錄已經證明有中文才下「有」；否則照 Vido 紀錄判斷（通常是 `unknown`）——不能因為旁邊只有英文檔就說「缺中文」。
3. **打開一季不跑 ffprobe，資料夾一個只讀一次**（同資料夾的集數共用一次 `ReadDir`）。旁邊檔讀不到時退回存好的結果，不讓整個清單失敗。
4. **存起來**：migration 044 給 `episodes` 加 `subtitle_tracks`（TEXT，NULL＝沒讀過，格式同電影）與 `subtitle_tracks_file_sig`（TEXT，探測時檔案的「大小:修改時間」）。打開一季時，若片內已讀過而旁邊檔跟存的不一樣，就寫回（窄寫，只動這一欄，不動 `updated_at`）；寫回失敗只記 log。
5. **背景補齊（掃描後＋開機後）**：新服務逐集處理「有檔案、且從沒讀過或檔案大小／修改時間變了」的集數：ffprobe 讀片內軌 → 讀旁邊檔 → 一次寫入 `subtitle_tracks`＋`file_sig`；若 `duration_seconds` 是 NULL 且探測有時長，順手寫入。
   - 開機後延遲啟動（不擋 API）；每次掃描完成觸發一次；正在跑時再觸發 → 跑完再補跑一輪，不會同時跑兩輪。
   - 可中斷（關機時 ctx 取消就停）、可續跑（進度就是資料庫裡的 `file_sig`，重啟後從還沒做的開始）。
   - ffprobe 沒裝 → 整輪不做（集數維持「還不知道」）；單集探測失敗 → 記 log、這個 process 內不再重試這集。
   - log：開始（待處理幾集）、每 25 集一次進度、結束（讀了幾集、失敗幾集、略過幾集）。
6. **候選分析不再重探**：單集列的 `tracksJSON` 改讀 `episodes.subtitle_tracks`，跟電影同一條免探測的路（`backlog-episode-tech-info-parity` 的字幕軌那一半）。
7. **測試用《末日光明》的真實形狀**：S01E01＝`none`（只有英文片內字幕）；S01E02＝`zh_hant`，來源＝旁邊 `zh-TW` 檔＋Vido 生成；S01E03–E08＝`zh_hant`（旁邊 `zh-TW` 檔）；S02＝片內中文——`chi` 標題「繁體中文」→ `zh_hant`、`chi` 沒標題 → `zh`（照 PR #674「分不出繁簡」的規則）。另測：片內沒讀過＋旁邊只有英文 → `unknown`；旁邊檔名寫 `zh-TW` 但內容是簡體 → `zh_hans`。
8. `go build／vet／test ./...`、`staticcheck-2026.1`、migration 測試（NULL 讀回、Up 冪等、有註冊）全綠。

## Tasks / Subtasks

- [x] T1 migration 044＋`Episode` 兩個欄位＋repository（SELECT／scan、`UpdateSubtitleTracks` 窄寫、補齊用的清單查詢）（AC #4）
- [x] T2 `subtitle` 套件：`ListSidecars` 拆出「一次讀資料夾、多個片檔共用」的版本；`cmd/api` 轉接器把結果轉成 `services.SubtitleTrack`（AC #3）
- [x] T3 背景補齊服務＋開機／掃描後接線（AC #5）
- [x] T4 季清單 API 的 `chinese_subtitle`／`chinese_subtitle_sources`＋寫回；候選分析讀存好的軌道（AC #1、#2、#3、#4、#6）
- [x] T5 測試（AC #7、#8）

全是後端，5 項。

## Dev Notes

### 不要做的事

- 不碰前端（-b）。
- 不改電影的 `DetectExternalSubtitles`（檔名標籤）——那是另一件事（原單「一併評估」那條，留在傘狀單）。
- 不改「管理字幕」對話框、影集標頭、季標頭。

### 已知陷阱

- `SubtitleTrack` 加 `FileName`（`omitempty`）給旁邊檔用；電影的 JSON 不受影響。旁邊檔的 `Language` 用 `ListSidecars` 的內容判斷結果，**不要**把檔名塞進 `Title`（`ChineseTitleScript` 會讀 Title，檔名的 `zh-TW` 會蓋過內容判斷）。
- 寫回時別覆蓋背景工作剛寫的片內結果：只在「片內已讀過」時寫，寫的是「存好的片內＋現讀的旁邊」。
- 補齊要 stat 每集（FUSE 上），`aggregateSeriesFileSizes` 每次掃描本來就這樣做，可接受。

### References

- 傘狀單 `disc-2026-10-episode-list-subtitle-badge.md`；`disc-2026-10-subtitle-filter-disagrees-with-badges.md`（判斷規則）
- PR #674（`ListSidecars`）、`credits_backfill_service.go`（背景工作前例）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev）；對抗式 CR 換模型（Fable 5.1）。

### Completion Notes List

- 2026-10-07 Bob（SM）：從傘狀單拆出，三條 SM 裁定見上。
- 2026-10-07 Amelia（dev）：T1–T5 完成。
  - migration 044：`episodes.subtitle_tracks`（格式同電影）＋`subtitle_tracks_file_sig`；`Episode` 兩欄、`episodeSelectColumns`／`scanEpisode` 同步、手寫測試 schema 同步；`Create`／`Update` 不寫這兩欄（掃描 upsert 不會蓋掉）。
  - `subtitle.ListSidecars` 拆出 `sidecarsFromEntries`；新 `subtitle.SidecarTrackReader`（實作 `services.SidecarTrackReader`，一個資料夾讀一次，語言看內容，檔名放 `FileName` 不放 `Title`）。
  - `services.EpisodeSubtitleTracksService`：開機 60 秒後一次＋每次掃描完成觸發；跑的時候再觸發→跑完補一輪；`file_sig` 不同才重探；探測失敗這個 process 不重試同一個檔；時長 NULL 時順手寫入。
  - 季清單 API：`chinese_subtitle`＋`chinese_subtitle_sources`（指標 slice，本地檔集數一定有、沒來源是 `[]`）；片內沒讀過時旁邊只有英文 → `unknown`；寫回用 compare-and-set（`RefreshSubtitleTracks`）。
  - 候選分析：單集 `tracksJSON` 讀 `subtitle_tracks`（同電影的免探測路）。
  - 前端只加 `MergedEpisode` 兩個 optional 型別欄位（契約鏡像），畫面不動。
  - 偏離原單：原單「估價已跑的 ffprobe 順手寫入」改成反方向——背景工作寫軌道與時長，估價讀存好的軌道；效果同樣是「不重複探測」，且軌道只有一個寫入者。
  - 閘門：`go build／vet／test ./...` 全綠、`staticcheck 2026.1` 乾淨、`nx run api:lint` 綠；`-race`：本單新測試 ×3 乾淨；整包 `-race` 只有既有的效能門檻測試 `TestSegmentStore_ReadStaysFastAgainst200kRows` 因 race 模式變慢而紅（非 data race，非本單）。

- 2026-10-07 對抗式 CR（Fable 5.1，換模型）：0H／3M／3L，範圍內 5 項全修、各補測試：
  - **M1** 打開一季會重讀每個旁邊字幕檔的前 100KB（NAS 上慢）→ 旁邊檔的 track 多存 `file_sig`；名字與大小／修改時間沒變就沿用存好的語言，不再開檔。
  - **M2** 開機那一輪若 NAS 還沒掛載會整輪略過、log 看起來像做完 → 分出 `missing_files` 計數；一個檔都摸不到時 10 分鐘後重試，最多 6 次。
  - **M3** 單集存成 `[]`（探測過、確定沒有字幕軌）估價仍會重探 → 單集的 `[]` 算事實（`emptyTracksAreFact`），電影的 `[]` 照舊重探。
  - **L2** 關機時背景工作在 `db.Close()` 之後才停 → 移進關機區塊：取消後等它返回再關資料庫。
  - **L3** 補三條測試：資料夾讀不到只存片內半邊、存的 JSON 壞掉當沒讀過、沒標籤的非中文旁邊檔不算「缺中文」。
  - **L1（不修，寫進 -b）** 片內沒讀過＋Vido 紀錄 `not_found` 會先顯示「缺中文」——這是 `ChineseSubtitleVerdict` 既有規則 2（電影同規則），要改需 Alexyu 裁定；背景工作讀完就會更正。
  - 修後閘門：`go test ./...` 全綠、`staticcheck` 乾淨、`lint:all` 0 errors、prettier 綠、`web:typecheck` 綠；本單測試 `-race` ×2 乾淨。

### Discovery Triage

- **①（就地吸收）** `backlog-episode-tech-info-parity` 的字幕軌那一半、`backlog-episode-duration-warm-cache-gap` 的選項 (a)：背景工作會補時長。兩條都只解一部分，條目留著、加註。
- **③（不修）** 單集按鈕的估價（`transcription_handler.go` 單集 estimate）還沒讀 `episodes.subtitle_tracks`，仍會探測一次；電影那條已讀。留在 `backlog-episode-tech-info-parity` 加註。

### File List

- apps/api/internal/database/migrations/044_add_episode_subtitle_tracks.go（新）＋ _test.go（新）
- apps/api/internal/models/episode.go、apps/api/internal/models/chinese_subtitle.go
- apps/api/internal/repository/episode_repository.go、episode_repository_test.go、episode_subtitle_tracks_test.go（新）
- apps/api/internal/subtitle/inventory.go、sidecar_track_reader_test.go（新）
- apps/api/internal/services/ffprobe_service.go、episode_subtitle_tracks_service.go（新）＋ _test.go（新）、episode_chinese_subtitle.go（新）＋ _test.go（新）、series_season.go、series_service.go、series_season_subtitle_test.go（新）、generation_candidates.go、generation_candidates_test.go
- apps/api/cmd/api/main.go
- apps/web/src/types/library.ts
