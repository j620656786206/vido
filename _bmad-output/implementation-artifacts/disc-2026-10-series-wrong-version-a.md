# disc-2026-10-series-wrong-version-a：影集會讀資料夾裡的 tvshow.nfo，也會用檔名年份挑版本，不再把 2026 動畫版認成 2018 真人版

Status: review

<!-- SM Bob create-story 2026-10-23；行號為 main `65a795f1`。傘狀條目 disc-2026-10-series-wrong-version 拆成 -a（比對，本單）／-b（已比對的片也能手動選片，要先畫稿）／-c（資料夾改名後舊影集不消失）。 -->

**Source:** Alexyu 2026-10-23 回報「將夜總共只有 19 集，為什麼 Vido 顯示 60 集？」。查證後 Alexyu 選 A（治本：讀 NFO＋用年份＋已比對的也能手動選片），⚖️ SM 裁定拆三張，切分線見上。

## 為什麼要做

NAS 實查（`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`，2026-10-23 13:26 寫入，新資料）：

- 資料夾 `/media/tv/將夜/Season01/` 有 19 個檔，檔名 `[将夜].Ever.Night.2026.S01E01…`，是 **2026 年的動畫版**（TMDb 282136，bilibili，20 集）。
- Vido 把它比對成 **2018 年的真人劇**（TMDb 83612，騰訊，第 1 季 60 集），所以畫面寫 60 集。
- 資料夾裡的 `tvshow.nfo` 已經寫著 `<uniqueid default="true" type="tmdb">282136</uniqueid>`，但 Vido 只有電影會讀 NFO，影集不讀。
- 用「將夜」搜 TMDb，第一個是 2018 版、第二個才是 2026 版。Vido 永遠拿第一個；檔名裡的 2026 沒有被拿來挑。

翻拍、重製、同名的劇很常見（將夜、流星花園、三國……）。認錯了，集數、海報、簡介、演員、字幕的人名詞彙表全都是另一部的。

## Acceptance Criteria

1. **讀 tvshow.nfo**：影集資料夾裡有 `tvshow.nfo`，裡面有 TMDb 編號 → 直接用這個編號跟 TMDb 拿資料，不搜尋。`metadata_source = nfo`，`parse_status = success`，演員照常抓。只有 IMDb 編號 → 用 TMDb 的「用外部編號找」找影集，找到就照上面做。
   - 第一次比對（掃描後）和「重新解析」都適用。所以將夜修好之後，按一次「重新解析」就會變成 2026 版。
2. **不在媒體庫根目錄讀**：影集的資料夾就是媒體庫根目錄時（平鋪的擺法），不讀那裡的 `tvshow.nfo`、不用那裡的資料夾名稱或檔案猜年份——那些不屬於這一部劇。
3. **用年份挑版本**：沒有可用的 NFO 編號時，照下面的順序找一個年份，找到就在搜尋結果裡挑**第一個跟第一名同名、而且首播年份相同的**（同一個名字的另一個版本）；都沒有，照舊拿第一個。同名的限制是防止檔名裡的年份其實是後面某一季的年份，結果把對的劇換成剛好那年播出的另一部。
   1. `tvshow.nfo` 的 `<year>`
   2. 資料夾名稱結尾的年份：`將夜 (2026)`、`將夜.2026`
   3. 影集名稱結尾的年份（掃描時從檔名解析出來的，例如 `Ever Night 2026`）
   4. 資料夾裡（含季資料夾）第一個影片檔檔名裡、緊接在 `S01E01` 前面的年份
   - 名稱結尾的年份也從搜尋字串拿掉（`Ever Night 2026` 用 `Ever Night` 搜）。
   - 整個名稱就是年份（例如影集 `1923`）不算年份。
4. **不蓋掉使用者的選擇**：`metadata_source = manual` 照舊不動（現在的行為）。NFO 優先序照 `ShouldOverwrite`（manual 100 > nfo 80 > tmdb 60）。
5. **出錯照舊**：NFO 讀不到、壞掉、TMDb 拿不到那個編號 → 記 log，退回搜尋（加上 AC 3 的年份）。不會因此變成比對失敗。
6. **不用 AI、不多打 TMDb**：找年份只用正規表示式和讀資料夾，不呼叫 AI 解析；讀資料夾最多看兩層、有上限。
7. **測試**：
   - NFO 有 TMDb 編號 → 用編號、不搜尋、source=nfo；只有 IMDb 編號 → 外部編號找影集。
   - 影集資料夾＝媒體庫根目錄 → 不讀 NFO。
   - NFO 壞掉／TMDb 失敗 → 退回搜尋。
   - 年份四種來源各一條；有年份時挑對的那一個；沒有相同年份時拿第一個；`1923` 不被當成年份。
   - manual 的影集不讀 NFO。
   - 將夜的實際檔名：`[将夜].Ever.Night.2026.S01E01.2160p…mp4` 解析出 2026。

## 不要做的事

- 不改掃描時的影集名稱（`TVParser` 的 `Title` 照舊），只在比對時另外拆年份。
- 不改電影的比對（電影的年份其實也沒傳給 TMDb 搜尋，另外立案 `disc-2026-10-tmdb-search-ignores-year`）。
- 不做「已比對的片也能手動選片」（-b）、不處理資料夾改名後的舊影集（-c）。
- 不改 NFO 在地化（9R-13a）寫出的內容。

## 已知風險

- 如果某個 `tvshow.nfo` 本身寫錯編號，Vido 會照它比對（NFO 的優先序高於搜尋）。手動選片（manual，-b 會讓已比對的片也能用）優先序最高，可以蓋過去。

## Tasks

- [x] Task 1 — `parser.SplitTrailingYear`：名稱結尾的年份拆出來（AC 3、7）
- [x] Task 2 — `enrichSeries` 讀 `tvshow.nfo`（TMDb／IMDb 編號），根目錄防護、失敗退回（AC 1、2、4、5）
- [x] Task 3 — 年份提示四種來源＋搜尋結果挑年份（AC 3、6）
- [x] Task 4 — `main.go` 接上媒體庫根目錄；測試、vet、staticcheck（AC 7）
- [x] Task 5 — NAS 驗證：將夜按「重新解析」後變成 TMDb 282136、20 集

## Dev Notes

- 影集比對：`apps/api/internal/services/enrichment_service.go:390`（`enrichSeries`，拿 `Items[0]`）。
- 電影的 NFO 流程可以照抄：`tryNFOEnrichment`（`:848`）、`enrichFromNFOWithTMDb`（`:923`）、`enrichFromIMDbID`（`:1119`）。
- 寫入影集：`applyTMDbSeriesDetails`、`persistSeriesMatch`（`enrichment_manual_match.go:437、482`）。
- NFO 解析已支援 `<tvshow>`：`nfo_reader_service.go:200`（`TMDbID`、`IMDbID`、`Year`）。
- 搜尋結果的年份：`metadata.MetadataItem.Year`（`tmdb_provider.go:207`，從 `first_air_date` 來）。
- 季資料夾的判斷：`media_ingest_service.go` 的 `seasonDirPattern`。
- 媒體庫根目錄：`MediaLibraryRepository.GetAllPaths`（`MediaLibraryPath.Path`）。
- `TVParser` 對將夜的檔名回 `Title="Ever Night 2026"`、`Year=0`（實測）。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-23。

### Completion Notes List

- 新檔 `enrichment_series_identity.go`：讀 `tvshow.nfo`（TMDb 編號優先，沒有才用 IMDb 透過 TMDb find）、媒體庫根目錄防護（根目錄讀不到也不信任資料夾）、年份提示四種來源、`pickSeriesMatch`。`enrichSeries` 只多三段：先試 NFO、搜尋字串拆掉結尾年份、用年份挑結果。
- （CR 自審）挑年份加了同名限制：檔名年份若是後面某一季的發行年，不會把對的劇換成那年播的另一部（`TestPickSeriesMatch_OnlyTradesForTheSameNamedShow`）。
- NFO 比對換成另一部時，抓不到新演員就寫空的演員表，不留舊那部的（跟手動選片同一條規則）。
- 年份也帶進 `SearchMetadataRequest.Year`（電影本來就有帶；TMDb provider 目前沒用它，只影響重試佇列與 keyword fallback 的請求）。
- `main.go`：根目錄 = 媒體庫路徑＋`VIDO_MEDIA_DIRS` 備援路徑。
- 反向驗證：`pickSeriesMatch` 換回 `Items[0]` → 年份 4 個案例紅；拿掉 NFO 那段 → NFO 兩個測試紅。
- **NAS 實測（2026-10-23，正式資料庫的備份＋臨時容器，媒體唯讀）**：將夜（`8f55e12f…`）重新解析前 `83612|tmdb|2018-10-31`，重新解析後 `282136|nfo|2026-04-23`；log `series matched from tvshow.nfo`、詞彙表用 `tmdb:tv:282136` 重新種了 4 個人名。測完容器與備份都刪了。正式機要等部署後再按一次「重新解析」。
- `go test ./internal/services/... ./internal/parser/... ./internal/handlers/... ./cmd/api/...` 全過；vet、staticcheck 乾淨。

### File List

- `apps/api/internal/parser/trailing_year.go`（新）
- `apps/api/internal/parser/trailing_year_test.go`（新）
- `apps/api/internal/services/enrichment_series_identity.go`（新）
- `apps/api/internal/services/enrichment_series_identity_test.go`（新）
- `apps/api/internal/services/enrichment_service.go`
- `apps/api/cmd/api/main.go`
