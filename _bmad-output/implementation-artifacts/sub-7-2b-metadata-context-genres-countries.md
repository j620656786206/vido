# Story sub-7-2b: 比對後真的存下類型與國家；影集補 production_countries 欄（資料側，後端）

Status: ready-for-dev

<!-- SM Bob create-story 2026-09-30，由 sub-7-2 拆出；不依賴 7-2a（兩張都只加訊號）。行號為 main `9f79863d`。 -->

## Story

身為用 Vido 翻字幕的人，
比對到 TMDb 的片，類型（genres）和製作國家不該是空的——
尤其影集：現在連欄位都沒有，導致 sub-7-4「中國內容跳過台灣詞庫」對影集完全不生效。

## 背景（查到的事）

- **genres=[] 的根因**：`metadata/tmdb_provider.go` 完全沒有 `Genres`——TMDb search 結果只有 `genre_ids`，provider 從不對照成名稱，所以 `metadata.MetadataItem.Genres` 永遠空；`enrichment_service.go:1083-1085`／`:483-485` 的 `if len(item.Genres) > 0` 從不成立。只有 NFO（`enrichFromNFOWithTMDb` → `applyTMDbMovieDetails`）、IMDb id、手動比對（`ApplyTMDbMatch` → `applyTMDbMovieDetails`／`applyTMDbSeriesDetails`）這些**有拿 details** 的路徑會寫 genres。一般掃描比對走 search → 沒有。
- **電影 countries**：`movies.production_countries` 欄位存在（migration 006），但只有 scan 的 `services/converters.go:79-87 ConvertTMDbMovieToModel` 會寫；enrichment 的兩個 apply 函式與 `applyTMDbMovieDetails` 都不寫。
- **影集 countries**：`series` 表**沒有** `production_countries` 欄；`models.Series` 沒有對應欄位；`ConvertTMDbSeriesToModel(details *tmdb.TVShowDetails)` 有 details 卻無處可放。TMDb `TVShowDetails.ProductionCountries []Country`、`TVShow.OriginCountry []string` 都有。
- 兩條翻譯腿（`subtitle/media_store.go:seriesContext`、`transcription_service.go:mediaMetadataFor` 影集分支）都留了「series 沒有 countries 欄」的註解等這張。
- `applyTMDbSeriesDetails` 目前在 `enrichment_manual_match.go:419`（手動比對用），沒有 countries。
- 測試用 schema 是**手寫**的：`repository/series_repository_test.go`、`season_repository_test.go`、`episode_repository_test.go`、`media_library_repository_test.go` 與 migrations 006/010/021/024 的 `_test.go` 各自 `CREATE TABLE series`——加欄位要一併補（Rule 15：schema／model／repo／scan 同步，並用真 DB 驗）。

## Acceptance Criteria

1. **series 加欄。** migration 040 `ALTER TABLE series ADD COLUMN production_countries TEXT`（沿 024 的 `columnExists` 樣式）；`models.Series` 加 `ProductionCountriesJSON NullString db:"production_countries"` ＋ `ProductionCountries []ProductionCountry db:"-"` ＋ `Get/SetProductionCountries`（鏡射 movie）；`series_repository.go` 的 `seriesSelectColumns`／Create／UpdateEnrichedMetadata／Upsert／scan 全部帶上；Upsert 對既有值的保留規則同 credits（scan 路徑沒設時不清空）。`TestEverySeriesReadPathReturnsEveryColumn` 綠。
2. **比對後補 details。** `enrichMovieFrom` 與 `enrichSeries` 在 TMDb 比對成功後（`searchResult.Source == TMDb` 且 `TMDbID` 有效）拿一次 `GetMovieDetails`／`GetTVShowDetails`（cache 層 24h），只補「context 欄位」：genres（details 有值就覆蓋 search 的空值）、production countries（影集：`ProductionCountries`，空則退回 `OriginCountry` 代碼）。details 失敗只 `Warn`，比對照常落盤。
3. **既有 details 路徑也寫 countries。** `applyTMDbMovieDetails`／`applyTMDbSeriesDetails`／`ConvertTMDbSeriesToModel` 補 production countries；手動比對與 scan 路徑因此一致。
4. **兩條翻譯腿讀到影集國家。** `seriesContext` 與 `mediaMetadataFor` 影集分支填 `Countries: countryCodes(series.ProductionCountries)`；sub-7-4 的「CN 內容跳過台灣詞庫」對影集開始生效（既有 `IsCNContent`／同類判斷讀 `Countries`——開工時確認實際函式名）。
5. **測試。** repository 真 sqlite：series countries 寫讀往返、Upsert 保留；enrichment：search 比對後 genres／countries 來自 details、details 失敗仍落盤、非 TMDb 來源不打 details；media_store／transcription：影集 countries 進 context；migration 040 up 兩次不炸。api 全量、vet、lint:all 綠。

## Tasks / Subtasks

- [ ] Task 1 — migration 040 ＋ model ＋ repository ＋ 測試 schema 同步（AC #1）
- [ ] Task 2 — enrichment 比對後補 details（AC #2, #3）
- [ ] Task 3 — 兩腿讀影集 countries（AC #4）
- [ ] Task 4 — 測試、全量（AC #5）

## Dev Notes

- 每次 TMDb 比對多一個 details 請求（與 sub-7-3 的 credits 請求同級）；有 24h cache，rate limit 4 req/s 由 client 管。
- 不要在 provider 層用 genre_ids 對照表硬翻——details 才有 countries，而且對照表會隨 TMDb 漂移。
- `MetadataHash` 會因 genres／countries 有值而改變：與 7-2a 同一個語意（不自動重跑，force 才付費），docs 同一段已涵蓋。

### Time-dependent visual coverage

- N/A。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                    |
| ---------- | ----------------------------------------- |
| 2026-09-30 | create-story（SM Bob，自 sub-7-2 拆出）。 |
