# Story DL-IMPORT.1: 下載清單說得出「進媒體庫了沒」（後端）

Status: done

## Story

As a self-hoster whose downloads are imported by Sonarr and Radarr,
I want each finished download in Vido to say whether it was imported and whether Vido has it,
so that I can tell at a glance what is done, what is still waiting for Sonarr/Radarr, and what Vido has not picked up.

## Context

由 `disc-2026-09-downloads-v2-no-parse-status` 拆出的第二張（第一張 13-6 已合併，PR #434）。

⚖️ **Alexyu 2026-09-15 兩個裁定**：不補舊的「解析狀態」（那條管線從沒接線，見 `disc-2026-09-parse-pipeline-never-wired`），改做「入庫狀態」；判斷方式**走 Sonarr/Radarr 的匯入紀錄**，不用檔名或片名猜。

依據（2026-09-15 唯讀查線上環境）：

- Sonarr/Radarr 匯入時會改名：`/mnt/user/data/torrents/{movies,tv}` 717 個影片檔只有 31 個檔名與媒體庫相同，檔名比對不可行。
- Sonarr 4.0.19、Radarr 6.3 的 `GET /api/v3/history`：
  - `downloadId` 是**大寫**種子 hash，而且大小寫敏感（小寫查不到）；
  - `eventType=1` 是 grabbed、`eventType=3` 是 downloadFolderImported；
  - `includeSeries=true&includeEpisode=true`（Sonarr）／`includeMovie=true`（Radarr）會把 series／episode／movie 帶進每筆紀錄，裡面有 `tmdbId`、`seasonNumber`、`episodeNumber`；
  - `pageSize=1000` 有效；線上 Sonarr 581 筆 grabbed＋197 筆 import，Radarr 43＋42。
- Vido 的 movies／series 有 `tmdb_id`（有索引）與 `FindByTMDbID`；episodes 有 `FindBySeriesSeasonEpisode` 與 `file_path`。

## Acceptance Criteria

1. **[@contract-v1] 回應欄位**（dl-import-2 消費）：`GET /api/v1/downloads` 的每個 item 可選帶 `import_status`：
   - `state`：`in_library`（Sonarr/Radarr 匯入了，而且 Vido 有：這部電影任何一份沒被移除、有檔案的版本；或這個種子匯入的每一集）｜`awaiting_scan`（匯入了，但 Vido 沒有）｜`awaiting_import`（抓了，之後沒有別的事）｜`import_failed`（Sonarr/Radarr 標記下載失敗）｜`import_ignored`（有人叫 Sonarr/Radarr 略過）
   - `source`：`radarr`｜`sonarr`；`media_type`：`movie`｜`tv`
   - `media_id`：Vido 有這部片／這部劇時的 id（部分入庫的季包也給，方便連結）
   - Sonarr 另有 `episodes_imported`、`episodes_in_library`（電影不帶）
   - 兩個 plugin 都不認得的種子、兩個都沒設定、或匯入紀錄沒有 TMDb id（沒辦法對媒體庫）：**不帶欄位**（不猜）。
2. 只對**下載完（進度 100%）**的種子查，不管它現在是做種、暫停、排隊或出錯（缺檔）；下載中的不查。
3. **不拖慢清單**：每個 plugin 的歷史依 hash 建索引，一分鐘最多更新一次，而且在背景更新——手上已經有一份時請求**不等** Sonarr/Radarr；第一次最多等 3 秒，還沒好就先不帶欄位。同時間多個請求共用一次抓取，抓取不跟著任何一個請求被取消。
4. **失敗不擋清單**：
   - 連不上時沿用上一份（匯入事件不會「取消發生」，只會晚、不會錯），最多 30 分鐘，太舊就不說；
   - 連續失敗會退避（1、2、4… 分鐘，最多 5 分鐘），而且健康檢查說連不上的 plugin 根本不在請求裡去打；
   - 第一次失敗記 WARN，之後降成 Debug；
   - 關掉或沒設定的 plugin 立刻忘掉它的舊資料；設定換到另一台伺服器時也丟掉舊資料。
5. **媒體庫比對說真話**：
   - 已被移除的電影／影集不算；電影列存在但沒有檔案（例如只是被請求過）不算；同一個 TMDb id 有多列時取最新的一列。
   - 影集以「檔案」為單位：一個檔案裝兩集（S01E01E02）時，Vido 只建一列也算兩集都有；Vido 用不同編號存（例如動畫的絕對集數）時，用匯入後的檔名比對。
   - 同一頁清單裡，同一部劇只讀一次集數。
6. 單元測試（client 解析、狀態判斷、快取、失敗）、handler 測試（只查完成的種子、欄位形狀、沒接服務時不變）、go vet／staticcheck 全綠。

## Tasks / Subtasks

- [x] **Task 1 — plugin 層（AC #1）**：`plugins.ImportHistoryRecord`（含 `Date`、`ImportedPath`）、`HistoryEvent*`（grabbed／downloadFolderImported／downloadFailed／downloadIgnored）、`ImportHistoryReader`（不放進 `DVRPlugin`，照 `ProfileLister` 的前例）；Radarr／Sonarr client 的 `GetImportHistory`：先 `GET /movie`／`/series` 一次拿 id→TMDb 對照（取代逐筆內嵌，線上實測內嵌讓回應大 2～3 倍），再依事件類型分頁抓歷史（新到舊排序，碰到 20 頁上限時丟最舊的並 WARN），hash 轉大寫
- [x] **Task 2 — 服務（AC #1, #3, #4, #5）**：`services/import_status_service.go`
- [x] **Task 3 — repository（AC #5）**：`MovieRepository.FindWithFileByTMDbID`（沒移除、有檔案、最新一列）、`SeriesRepository.FindActiveByTMDbID`（沒移除、最新一列）；介面與既有 mock 補上
- [x] **Task 4 — handler＋接線（AC #1, #2）**：`DownloadItem.ImportStatus`、`SetImportStatusService`、每頁一次 `Resolve`、依進度篩；`cmd/api/main.go` 用 `pluginManager`＋`repos.Movies/Series/Episodes` 接上
- [x] **Task 5 — 測試（AC #6）**：Radarr client 3 個、Sonarr client 2 個、服務 19 個（含 `-race` 連跑 3 次）、repository 2 個、handler 2 個
- [x] **Task 6 — 真實資料驗證**：暫時的 probe 測試透過 SSH 通道打線上 Sonarr/Radarr，媒體庫用線上 DB 匯出的 tmdb／集數／檔名（唯讀，已過濾移除的列），跑完即刪。CR 修正後重跑：155 個下載裡 154 個解得出來（1 個匯入紀錄沒有 TMDb id，照規則不說）：

  | 來源 | in_library | awaiting_scan | awaiting_import |
  | --- | --- | --- | --- |
  | Radarr | 28 | 14 | 2 |
  | Sonarr | 73 | 13 | 24 |

  Sonarr 的 in_library 從第一版的 68 升到 73：改用檔名比對後，Vido 用不同編號存的集數也對得上了。（這是整份歷史裡的所有下載，不只是目前還在 qBittorrent 裡的；剩下的 awaiting_scan 多半是之後被升級或刪掉的舊檔。線上沒有任何 failed／ignored 事件，那兩個狀態只有單元測試。）

- [x] **Task 7 — 對抗式 CR（乾淨 context agent）**：3H／7M／7L，見 Completion Notes

## Dev Notes

### 不要做的事

- **不做畫面**：卡片／表格顯示是 `dl-import-2`。
- **不接舊的解析管線**：`disc-2026-09-parse-pipeline-never-wired`。
- **不在清單請求裡對每個種子打 *arr API**：以線上 155 個下載、Sonarr/Radarr 10 req/s 的限流算，一頁 500 筆會卡上分鐘級。

### Rule 27 五柱

- ① 限流：沿用 plugin client 的 `rate.Limiter`；背景抓取有自己的 90 秒逾時。
- ② 快取：每個 plugin 的歷史索引 60 秒，stale-while-revalidate；請求內另有媒體庫查詢快取。
- ③ 降級：先看健康檢查；失敗退避；沿用上一份最多 30 分鐘。
- ④ 錯誤碼：不新增；client 沿用 `DVR_*`，服務層不往外丟錯。
- ⑤ 金鑰：只在 client 標頭，log 不出現。

### References

- 13-4a／13-4b（plugin 管理與 client）、13-6（設定畫面）
- `disc-2026-09-downloads-v2-no-parse-status`（裁定與查證紀錄）

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context)

### Completion Notes List

對抗式 CR 結果與處理：

- **H1 共用抓取綁在第一個請求的 context 上**（使用者離開頁面就讓大家的抓取一起失敗）→ 抓取在背景跑、有自己的逾時；請求只等結果，不擁有抓取。附測試。
- **H2 在清單請求裡同步抓歷史、失敗沒有退避**（*arr 睡著時每次開清單卡 10–20 秒）→ stale-while-revalidate、第一次最多等 3 秒、健康檢查擋掉連不上的 plugin、失敗退避。附測試。
- **H3 已移除或重複的電影列被當成已入庫** → 新 repository 方法過濾移除、要求有檔案、固定取最新。附 repository 測試。
- **M1 一檔多集永遠到不了 in_library** → 以檔案為單位＋檔名比對。附測試（也涵蓋絕對集數）。
- **M2 awaiting_scan 包含 *arr 已經沒有的檔案、tmdb=0 永遠卡住** → tmdb=0 不帶欄位；電影改成「任何一份」即算入庫（升級後換檔也對）。*arr 端刪檔無法便宜判斷，記為限制：前端文案只說「Vido 還沒有」，這句永遠是真的。
- **M3 失敗／略過的下載永遠顯示等匯入** → 加抓 downloadFailed／downloadIgnored，以最新事件為準。
- **M4 每頁資料庫查詢沒上限** → 請求內快取；每部劇只讀一次集數。
- **M5 每次請求解密金鑰兩次** → 改用記憶體裡的健康狀態，只在真的要更新時才取 client。
- **M6 失敗只記 Debug** → 第一次 WARN。
- **M7 歷史越來越大** → 拿掉逐筆內嵌、新到舊排序、截斷會 WARN；增量抓取（`history/since`）未做，記為後續優化。
- **L 已處理**：換伺服器時丟掉舊資料（L1）、舊資料最長 30 分鐘並在關掉 plugin 時丟掉（L2）、有匯入的一方優先（L3）、tmdb 取最新匯入那筆（L4）、依進度篩（L5）。
- **L 記錄不修**：狀態型別放在 services（L6）、兩個 client 程式碼相近（L7，照既有慣例）。

### File List

- `apps/api/internal/plugins/plugin.go`
- `apps/api/internal/plugins/radarr/client.go`＋`client_test.go`
- `apps/api/internal/plugins/sonarr/client.go`＋`client_test.go`
- `apps/api/internal/services/import_status_service.go`（新）＋`_test.go`（新）
- `apps/api/internal/repository/movie_repository.go`、`series_repository.go`、`interfaces.go`＋兩個 repository 測試
- `apps/api/internal/testutil/mocks.go`、`internal/services/*_test.go` 裡的 repo mock（補新方法）
- `apps/api/internal/handlers/download_handler.go`＋`download_handler_test.go`
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-15 | 建立並實作；真實資料驗證；對抗式 CR 3H／7M 全處理（M7 增量抓取留作後續）後重跑驗證 154/155。狀態 → review。 |
| 2026-09-15 | PR #436 合併進 main（7b580212）。CI 全綠，純後端沒有視覺變更。狀態 → done。 |
