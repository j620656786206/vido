# Bugfix H: 資料庫裡的時間 SQLite 自己讀得懂（全部 UTC、一種格式）

Status: done

**Merges:** `bugfix-h-nanosecond-timestamps-break-sqlite-dates`（P2）＋ `backlog-sqlite-timestamps-carry-go-monotonic-suffix`（Rule 24 ③，sub-7-1 立案）。兩張講的是同一個病。

## Story

As the owner of a NAS library that is starting to grow date-based features (monthly spend, "最近 N 天", log retention),
I want every timestamp the app stores to be a plain UTC time SQLite itself can read,
so that `date()`/`datetime()`/`strftime()` and "newer than" comparisons stop silently returning NULL or the wrong rows.

## 病情（2026-10-02 正式庫副本實測，SQLite `.backup` 拉回本機，唯讀）

- 21 張表、所有由 Go 寫入的時間欄位：`datetime(col)` **100% 是 NULL**。例：`system_logs.created_at` 509,690/509,690、`cache_entries.*` 20,433/20,433、`episodes.*` 2,406/2,406。
- 存的是 Go `time.Time.String()`：`2026-08-24 10:56:26.303854294 +0800 CST m=+26.660278123`——本機時區名稱＋單調時鐘垃圾 `m=+…`。
- 時區混雜：同一張 `series.updated_at` 有 `+0800 CST` 也有 `+0000 UTC`（容器 TZ 曾經換過）。字串比大小（`ORDER BY created_at`、`WHERE expires_at < ?`）跨時區就會差 8 小時。
- 原票說「SQLite 只吃 3 位小數」**不成立**：`datetime('2026-10-02 23:10:50.110881115+08:00')` 正常。真正的兇手是後面那段 ` +0800 CST m=+…`。
- 根因：modernc `sqlite` driver 沒設 `_time_format` 時，綁定 `time.Time` 一律用 `t.String()` 序列化（`conn.go formatTime` 的註解自己承認是為了相容舊版）。
- Go 讀回來還能用，是因為 driver 的 `parseTimeString` 會先切掉 `m=`——靠運氣。

## Acceptance Criteria

1. **一處修，全部吃到。** App 的資料庫連線（`database.New`）綁定任何 `time.Time`（含 `sql.NullTime`、`*time.Time` 經 `driver.Valuer`）時，一律寫成 UTC、固定寬度 `YYYY-MM-DD HH:MM:SS.fffffffffZ`。不改 54 個 repo 呼叫點；做在 driver 層（包一層 `database/sql/driver`，只加 `CheckNamedValue`）。
2. **讀不受影響。** 新格式、舊 `String()` 格式、SQLite `CURRENT_TIMESTAMP` 格式，Go Scan 成 `time.Time` 都照舊成功、時間點不變。
3. **線上還原照舊。** `BackupService.replaceDatabase` 用的 `NewRestore`（`conn.Raw` 斷言）穿過包裝層仍可用。
4. **既有資料一次正規化。** Migration 042：所有非虛擬表、宣告型別為 `TIMESTAMP`/`DATETIME`/`DATE` 的欄位裡，Go `String()` 形狀的值改寫成 AC #1 的格式（UTC）。只動符合形狀且 parse 得了的值；NULL、空字串、`CURRENT_TIMESTAMP`、RFC3339、其他文字不動。可重跑（第二次 0 列）。
5. **釘住。** 測試：經 `database.New` 寫入 `time.Now()`（帶 monotonic、本機時區）後，`datetime(col)` 不為 NULL、字串以 `Z` 結尾、不含 `m=`、跟原時間點相等。
6. **正式庫副本乾跑。** 在 2026-10-02 的副本跑 migration：所有時間欄位「非 NULL 但 `datetime()` 是 NULL」的列數 → 0；記下耗時（`system_logs` 50 萬列）。
7. **順手一致化。** `ai_cache` 改傳 `time.Time`（原本自己 `Format(RFC3339)` 成本機時區字串，跟其他表的 UTC 空格格式比大小會錯）；讀取端行為不變。

## Tasks / Subtasks

- [x] T1 `internal/database/utc_driver.go`：註冊 `sqlite-utc` driver，包 modernc 已註冊的 `*sqlite.Driver`（保留全域 UDF／collation），conn 嵌入介面轉發所有方法＋`NewBackup`/`NewRestore`，只多 `CheckNamedValue`。`FormatTime` 匯出給 migration 共用。（AC #1 #3）
- [x] T2 `database.connect` 改用新 driver 名稱。（AC #1）
- [x] T3 Migration 042 `normalize_go_string_timestamps`（AC #4）
- [x] T4 測試：driver（AC #1 #2 #5，含 NullTime／nil／非時間值原樣）、還原（AC #3）、migration（AC #4，含冪等與不動 RFC3339）。
- [x] T5 `ai/cache.go` 寫 `time.Time`（AC #7）
- [x] T6 正式庫副本乾跑（AC #6），結果寫進 Dev Agent Record。

## Dev Notes

- 不設 DSN `_time_format=sqlite` 而是自己轉字串：driver 的 `sqlite` 格式保留本機時區偏移（`+08:00`）、小數位數不固定，字串比較仍會跨時區錯。轉成 UTC 固定寬度字串才同時解決「讀得懂」和「比得對」。
- 為什麼不在 repo 層改 `time.Now().UTC()`：54 處 × 19 檔，新程式碼一忘就回到原點；driver 層是唯一出口。
- 不動 `time.Local`：備份排程「每天幾點」依賴本機時區。
- mattn `sqlite3` 只在部分舊測試用，不走 app 連線，不受影響。
- Rule 24：不擴大成「RFC3339 欄位也統一」——`ai_cache` 是唯一寫 RFC3339 的表（正式庫 0 列），T5 直接改寫入端即可。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-02）

### Completion Notes List

- 共用格式放在葉節點 `internal/database/dbtime`：migration 直接 import `database` 會在 `persistence_test.go` 形成測試期 import cycle。
- Driver 包裝層拿的是 modernc 已註冊的那個 `*sqlite.Driver`（`sql.Open("sqlite","").Driver()`），不是 `&sqlite.Driver{}`——後者沒有全域 UDF／collation。
- `CheckNamedValue` 先跑 `driver.DefaultParameterConverter`（modernc 自己沒有 checker，原本跑的就是它），所以非時間值行為完全不變；`sql.NullTime`／`*time.Time` 經 Valuer 後一樣被轉。
- 正式庫副本乾跑（AC #6，Mac 上）：54 個時間欄位、577,257 列改寫、**2.9 秒**（`system_logs` 509,690 列占 3 秒中的大半；NAS 預估 10–20 秒，只發生一次）。改寫後「非 NULL 但 `datetime()` 讀不懂」＝ **0**。改寫後 app 新寫入一筆：`2026-10-02 06:12:07.528351000Z → datetime 2026-10-02 06:12:07`。
- 讀回不受影響：測試涵蓋舊 `String()`、`CURRENT_TIMESTAMP`、新格式三種都 Scan 回同一時間點；`+08:00` 的參數跟 UTC 存值比大小正確。
- 線上還原：`conn.Raw` 斷言 `NewRestore` 穿過包裝層仍成功（測試實跑一次 restore）。
- 前端連帶：時間改存 UTC 後，`RequestRow` 原本 `requestedAt.slice(0,10)` 會把台北早上 8 點前的請求顯示成前一天；改成用瀏覽器本地日期。全 web 只有這一處切 ISO 字串。
- 沒有任何 SQL 用 `date()`/`datetime()`/`strftime()`，也沒有 `WHERE x_at = ?` 等值比對——沒有程式依賴舊的壞行為。

- **Adversarial CR（2026-10-02，獨立 agent，用 `go test -overlay` 實測）**：
  - ✅ 修：`ai/cache.go` 的 `ClearExpired`/`Stats` 還在綁本機 RFC3339 字串；存值改 UTC 空格格式後，同一天的比較在分隔字元（空白 < `T`）就分出勝負，會把「今天稍晚才過期」的項目提早刪掉。改綁 `time.Now()`；`ai` 測試改走 app driver，並加一支在 UTC 也會紅的回歸測試（先在舊碼上確認會紅）。
  - ✅ 修：`RequestRow` 新測試在 UTC 的 CI 上分不出新舊寫法；改成測試內釘 `TZ=Asia/Taipei`，舊寫法下確認會紅。
  - 📝 不修：`ai_cache` 舊的 RFC3339 列 042 不動（正式庫 0 列）；42 支 repo 測試仍用原生 `sqlite` driver（行為與修前相同，格式由 `utc_driver_test` 釘住）；movies 每列 3 個時間欄位各一次 UPDATE 會觸發 3 次 FTS trigger（實測 2 萬列 0.5 秒，可接受）；讀回的 `time.Time` 現在是 UTC 位置，JSON 輸出變 `Z`——全前端只有 `RequestRow` 切字串，已修。
  - ✅ 驗證無誤：`db.Exec`／`Prepare`／`tx.Exec`／`tx.Stmt`／`sql.Named`／`*time.Time` 全走到 `CheckNamedValue`；modernc stmt 沒有自己的 checker；包裝層沒丟掉 app 用到的任何 modernc 方法；跟 `datetime('now')`／`CURRENT_TIMESTAMP` 的比較（cache、log、douban、offline_cache）修前差 8 小時、修後正確。

### File List

- apps/api/internal/database/dbtime/dbtime.go（新）
- apps/api/internal/database/utc_driver.go（新）
- apps/api/internal/database/utc_driver_test.go（新）
- apps/api/internal/database/database.go
- apps/api/internal/database/migrations/042_normalize_go_string_timestamps.go（新）
- apps/api/internal/database/migrations/042_normalize_go_string_timestamps_test.go（新）
- apps/api/internal/ai/cache.go
- apps/api/internal/ai/cache_test.go
- apps/web/src/components/requests/RequestRow.tsx
- apps/web/src/components/requests/RequestRow.spec.tsx

## Change Log

- 2026-10-02 建立（Bob；合併兩張 backlog 票；病情以正式庫副本實測為準）。
- 2026-10-02 Dev 完成（Opus 5.5）→ review。
