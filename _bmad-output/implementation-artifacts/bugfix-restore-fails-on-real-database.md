# Story bugfix：「還原備份」在真正的資料庫上也能成功，失敗時也退得回去

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who keeps Vido's backups for the day the NAS breaks,
I want 還原 to actually bring my library back — including backups made by an older version of Vido —
so that a backup is something I can rely on, not a file that fails the moment I need it.

## Context

升級自 `disc-2026-09-restore-fails-on-real-database`（**P0**，`bugfix-backup-includes-uploaded-posters` 本機冒煙發現）。

### 🔴 查到的事（main `c258c5a1`；行號皆為現況）

1. **實測重現**（本機開發資料庫，有媒體庫與電影，走 API）：建立備份 → 還原同一個備份 → `RESTORE_DB_FAILED: clear table media_libraries: constraint failed: FOREIGN KEY constraint failed (787)`，接著自動快照回滾 `RESTORE_ROLLBACK_FAILED: attach backup db: SQL logic error: database restore_db is already attached`。資料庫本身因交易回滾而完好。
2. **現行還原做法**（`replaceDatabase`，`services/backup_service.go:720-800` 左右）：在一個交易裡 `ATTACH` 備份檔，對備份裡的每張表 `DELETE FROM main.<t>` 再 `INSERT INTO main.<t> SELECT * FROM restore_db.<t>`。三個洞：
   - **外鍵**：連線開著 `foreign_keys=ON`（`internal/database/database_test.go:213-218` 驗證過），表的順序是 `sqlite_master` 的順序 → 先清父表就撞外鍵。
   - **舊版備份的欄位數不同**：`SELECT *` 對上欄位較少的舊表 → 「table has N columns but M values」。舊版本做的備份**一定**會失敗。
   - **錯誤路徑沒有 DETACH**：`ATTACH` 留在連線池某條連線上，回滾再 `ATTACH` 同名就失敗（重現中的第二個錯誤）。
   - 另外：備份裡沒有、但目前 schema 有的表（新版加的表）會保留**目前的**資料，還原後資料混在一起。
3. **既有測試為什麼一直綠**：`backup_service_test.go` 的 `createTestDB`／`createTestBackupArchive` 只建一張沒有外鍵的 `test_data` 表。
4. **「schema 版本」是假的**：`main.go:191` `NewBackupService(…, 17)` 寫死 17；manifest 與檔名的 `v17` 都來自它，實際 migration 已到 **038**（`internal/database/migrations/038_…`）。還原時的「備份比目前新就拒絕」（`:630` 附近）因此永遠比不出東西。
5. **可用的正解**：`modernc.org/sqlite v1.44.0` 的連線有 `NewRestore(srcUri) (*Backup, error)`（`conn.go:951`；`Backup.Step(-1)`／`Finish()`，`backup.go`）——這是 SQLite 的 **online backup API** 反向使用：以**頁**為單位把整個備份資料庫（含 schema）覆蓋到目前的資料庫，不經過 SQL，外鍵、欄位數、表的順序都不是問題。透過 `(*sql.DB).Conn(ctx)` ＋ `Conn.Raw(func(driverConn any) error)` 取得底層連線呼叫。
6. **Migration 執行器**：`migrations.NewRunner(db)`＋`RegisterAll(migrations.GetAll())`＋`Up(ctx)`（`main.go:78-97` 的開機流程），會把舊 schema 補到最新；`schema_migrations(version,…)` 記錄已套用的版本。
7. 還原後的放回海報（PR #547）接在 `replaceDatabase` 成功之後，不受本張影響。

### ⚖️ 建單裁定（2026-09-25，SM；Alexyu 可在 review 推翻）

1. **換掉整個「逐表 DELETE＋INSERT」做法，改用 SQLite online backup API 整庫覆蓋，覆蓋後立刻跑 migration 補到最新。** 理由：逐表複製要自己處理外鍵、欄位差異、表的增減、順序——每個 schema 變更都可能再弄壞它；整庫覆蓋是 SQLite 本身提供的原子操作，而「舊 schema 補到新」本來就是 migration 的工作（開機時就這樣做）。這是長解（`feedback_architecture_prefer_long_solutions`）。
2. **自動快照回滾走同一條路**——修好一個就修好兩個。
3. **schema 版本改成真的**：備份與 manifest 用目前資料庫 `schema_migrations` 的最大版本；還原前的「備份比目前新就拒絕」改成**讀解壓出來的備份資料庫本身**的 `schema_migrations`（舊備份 manifest 寫 17 不可信）。拒絕時錯誤寫出兩個版本號。
4. **還原中的寫入**：還原是使用者明確按下、且有確認框的動作；覆蓋期間 SQLite 會鎖住資料庫，其他背景工作會等或失敗重試——**不另外停掉排程器**（本張不擴大範圍），在 Dev Notes 記錄。

## Acceptance Criteria

1. **還原用 online backup API 整庫覆蓋。**
   - `replaceDatabase` 改為：固定一條連線（`s.db.Conn(ctx)`）→ `Raw` 取得底層連線 → `NewRestore(<解壓出來的 vido.db 路徑>)` → `Step(-1)` → `Finish()`；任何一步失敗都要 `Finish`／關閉，回傳錯誤。
   - 覆蓋成功後，對同一個 `*sql.DB` 跑 migration（`NewRunner`＋`RegisterAll(GetAll())`＋`Up`）。migration 失敗 → 當作還原失敗 → 走回滾。
   - 覆蓋成功且 migration 成功後，`PRAGMA foreign_key_check` 應回空；若不為空只記 Warn（不回滾——那是備份當時的資料狀態）。
   - **不再使用 `ATTACH`／`DETACH`**；刪掉那段程式。
2. **回滾走同一條路**（`rollbackFromSnapshot` 呼叫同一個 `replaceDatabase`，天然套用）。
3. **真實 schema 版本。**
   - `BackupService` 不再收寫死的 `17`：建構子改讀（或注入一個讀取函式）目前資料庫 `SELECT MAX(version) FROM schema_migrations`。備份檔名、manifest、`models.Backup.SchemaVersion` 都用它。
   - 還原前讀**解壓出來的** `vido.db` 的 `MAX(version)`（唯讀開啟）；比目前新 → `RESTORE_INCOMPATIBLE_VERSION: backup schema version X is newer than current Y`，不動資料庫。讀不到 `schema_migrations`（不是 Vido 的資料庫）→ 還原失敗、不動資料庫。
4. **測試（Go）——必須用真的 migration 鏈建庫，不能再用 `test_data` 單表。**
   - **外鍵重現**：建一個跑完全部 migration 的**檔案型**（非 `:memory:`）資料庫，插入 `media_libraries`＋有外鍵指向它的列（例如 `media_library_paths` 或 `movies.library_id`，dev 先查實際外鍵），`CreateBackup` → 改動資料（刪一部片、改一個標題、新增一筆）→ `RestoreBackup` → 狀態 completed、資料回到備份當下。**這條在修改前必須是紅的**（外鍵錯誤）。
   - **舊版備份**：用 `newMigratedDBUpTo`（`migrations/037_…_test.go` 的寫法）建一個只跑到較舊版本的資料庫、插資料、`VACUUM INTO` 打包成備份 → 還原到最新 schema 的服務 → 成功、資料在、`schema_migrations` 最大版本等於目前、新版才有的欄位有預設值。
   - **比目前新的備份**：備份資料庫的 `schema_migrations` 多一筆比目前大的版本 → 拒絕、目前資料不變。
   - **回滾**：讓還原在覆蓋後的 migration 失敗（注入一個會失敗的 migration 或假的 runner）→ 回滾成功、資料回到還原前、訊息含 `RESTORE_ROLLBACK_SUCCESS`；連續兩次還原都不會出現「already attached」。
   - **海報仍放回**：沿用 PR #547 的測試，改成真實 schema 後仍綠。
   - 既有的 `test_data` 型測試若因新做法不再適用（例如沒有 `schema_migrations`），改寫成真實 schema；刻意變更逐條寫進 Completion Notes。
5. **本機冒煙（必做）**：用 `apps/api/vido-data`（有媒體庫、電影）照 🔴 #1 的步驟重跑：建備份 → 刪一部片、刪其海報 → 還原 → 片子回來、海報回來（`GET /api/v1/posters/<id>.jpg` 200）、`RestoreResult` 訊息正確。結果寫進 Completion Notes；冒煙用的 API 關掉。
6. **CI**：`pnpm nx test api`、`pnpm run lint:all` 綠；紅／守（Rule 16）；mutation check（至少：拿掉 migration 那步、拿掉版本檢查、`Finish` 前不 `Step`、改回讀 manifest 的版本）。

## Tasks / Subtasks

- [x] **Task 1 — 測試先紅：真實 schema 的還原測試夾具＋外鍵重現（AC: #4）**
- [x] **Task 2 — `replaceDatabase` 改用 online backup API＋覆蓋後跑 migration＋外鍵檢查（AC: #1, #2）**
- [x] **Task 3 — 真實 schema 版本：建構子、manifest、還原前讀備份資料庫的版本（AC: #3）**＋ `main.go`
- [x] **Task 4 — 舊版備份／新版拒絕／回滾測試、本機冒煙、收尾（AC: #4, #5, #6）**

## Dev Notes

### 這張的重點

- **不要修補逐表複製**——整個換掉。外鍵、欄位數、表的增減都是它的結構性問題。
- **測試夾具才是這個 bug 活這麼久的原因**：一定要用真的 migration 鏈。

### 上游契約（Rule 20）

- 本張不消費任何 `[@contract-v*]`。API 回應格式不變；`schema_version` 的**值**從固定 17 變成真的 migration 版本（前端只顯示、不做判斷——dev 先 grep 確認）。

### 不要做的事

- 不要在還原時關掉 `foreign_keys` 然後繼續用逐表複製（那是短解）。
- 不要停掉排程器或改其他服務（裁定 #4）。
- 不要改前端。

### 已知陷阱

- **`Conn.Raw` 的型別斷言**：modernc 的連線型別未匯出，用介面斷言 `interface{ NewRestore(string) (*sqlite.Backup, error) }`；`*sqlite.Backup` 是匯出的（`modernc.org/sqlite`）。
- **`:memory:` 不適用**：online backup 覆蓋、WAL、多連線行為都要用檔案型資料庫測；`t.TempDir()` 下開檔。
- **WAL**：online backup 在目的地為 WAL 模式時要求頁大小相同——兩邊都是 Vido 建的，一致；測試資料庫也要設成 WAL（照 `internal/database` 的開法）。
- **migration 在覆蓋後的同一個 `*sql.DB` 上跑**：Runner 會自己 `CREATE TABLE IF NOT EXISTS schema_migrations`，已套用的版本會被跳過。
- **備份用 `VACUUM INTO`**（`sqliteBackup`）不變。
- **gh**：`GH_TOKEN=$(gh auth token --user j620656786206)`。

### Source tree

```
apps/api/internal/services/backup_service.go（+ backup_service_test.go、backup_posters_test.go）  ← Task 1–4
apps/api/cmd/api/main.go                                                                        ← Task 3
```

### Cross-Stack Split Check

後端 4、前端 0 → 不拆。

### Time-dependent visual coverage

- N/A — 沒有前端。

### References

- [Source: `apps/api/internal/services/backup_service.go`（`CreateBackup`、`sqliteBackup`、`createManifest`、`RestoreBackup`、`replaceDatabase`、`rollbackFromSnapshot`）；`cmd/api/main.go:78-97, 191`]
- [Source: `modernc.org/sqlite@v1.44.0/conn.go:937-975`（`NewBackup`／`NewRestore`）、`backup.go`（`Step`／`Finish`）]
- [Source: `apps/api/internal/database/migrations/runner.go:29-110, 283`；`migrations/037_glossary_seed_marks_and_foreign_tmdb_ids_test.go:15`（`newMigratedDBUpTo`）；`internal/database/database_test.go:213-218`（`foreign_keys=ON`）]
- [Source: `sprint-status.yaml` → `disc-2026-09-restore-fails-on-real-database`；`bugfix-backup-includes-uploaded-posters.md` Completion Notes（冒煙紀錄）]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（先紅）**：測試夾具改成真的：`openVidoTestDB`（檔案型、`foreign_keys(on)`、WAL、跑 migration 到指定版本）；`createTestDB` 改成完整 migration＋原本的 `test_data` 表；`createTestBackupArchive(…, schemaVersion, …)` 現在做出「真的是那個版本」的 Vido 資料庫（比目前舊就只跑到那版；比目前新就多塞一筆未來版本），用 `VACUUM INTO` 打包。新測試檔 `backup_restore_real_test.go`。**把舊的 `replaceDatabase` 換回去跑，三條新測試全紅，而且錯誤訊息與本機重現一字不差**：`FOREIGN KEY constraint failed (787) | RESTORE_ROLLBACK_FAILED: … restore_db is already in use`，以及舊版備份的 `table main.movies has 45 columns but 35 values were supplied`——後者證明**舊版 Vido 做的備份過去一定還原失敗**。
- **Task 2**：`replaceDatabase` 改成固定一條連線 → `Conn.Raw` → modernc 的 `NewRestore("file:<vido.db>?mode=ro")` → `Step(-1)` → `Finish()`（整庫以頁覆蓋，不經 SQL）；成功後跑 `migrateUp`（預設 `runAllMigrations`，與開機同一套）；再 `pragma_foreign_key_check`，有問題只記 Warn。`ATTACH`／`DETACH` 全部刪除。回滾（`rollbackFromSnapshot`）呼叫同一個函式，一併修好。
- **Task 3**：`NewBackupService(db, repo, backupDir)`——拿掉寫死的 `17`；備份檔名、`Backup.SchemaVersion`、manifest、自動快照都用 `schemaVersionOf(db)`（`MAX(version) FROM schema_migrations`，本機現為 **38**）。還原前改讀**解壓出來的資料庫本身**的版本（`backupSchemaVersion`，唯讀開啟）；讀不到 → `RESTORE_EXTRACT_FAILED: not a Vido database`；比目前新 → `RESTORE_INCOMPATIBLE_VERSION: backup schema version X is newer than current Y`，資料庫不動。`vido.db` 是否存在的檢查移到版本檢查之前。manifest 仍寫入但不再用來做判斷（`readManifest` 保留、有測試）。
- **Task 4**：新測試——真實 schema＋外鍵的還原（改、刪、增後還原，資料回到備份當下、`foreign_key_check` 為空）；連續兩次還原；舊版（v20）備份還原後資料在、`schema_migrations` 升到最新；比目前新的備份被拒且資料不變；覆蓋後 migration 失敗 → 回滾成功（`RESTORE_ROLLBACK_SUCCESS`）、資料回到還原前、沒有「already attached」；備份用真的 schema 版本。既有的 `test_data` 型測試因夾具變真實而**全部照樣綠**（不需改斷言），僅刪掉 `NewBackupService(…, 17)` 的版本參數。
- **本機冒煙（真的 API，`apps/api/vido-data`）**：建片 → 上傳海報 → 建備份（檔名 `…-v38.tar.gz`）→ 刪片（204）、刪海報檔、片子 404 → 還原 → `completed`「還原完成，資料庫與 2 張上傳的海報已恢復」、`posters_restored=2` → 片子回來（標題正確）、海報 `GET /api/v1/posters/<id>.jpg` **200**。✅ 這也是 PR #547「還原放回海報」第一次在真實資料庫上跑通。冒煙後刪掉測試片、關掉 API，port 8080／8081／4200 無殘留。
- ⚠️ **沒有停掉排程器**（裁定 #4）：覆蓋期間 SQLite 會鎖住資料庫，其他背景寫入會等（`busy_timeout`）或失敗重試。
- 🔗 **AC Drift: FOUND** — Story 6-x 備份還原（`RestoreBackup` 的「逐表還原」行為、manifest 的 `schema_version` 用來判斷相容性）→ 改為整庫覆蓋＋migration、相容性讀備份資料庫本身；`schema_version` 的值從固定 17 變成真的 migration 版本（API 欄位不變）。
- 📎 **Contract Stamps: NONE**。
- 🎭 **A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）**。
- 🎨 **UX Verification: SKIPPED — no UI changes in this story**。
- **測試**：`nx test api` 綠、`lint:all`（0 error）綠。
- **Mutation 4／4 紅**＋「換回舊實作 3／3 紅」：拿掉 migration、拿掉版本檢查、`Step(0)` 不複製、版本寫死 17。

### 🔍 /ship Adversarial Review（2026-09-25）

0 HIGH／0 MEDIUM／2 LOW：
- **L1（加測試，不需改碼）** 按下還原時，背景剛好有寫入交易 → 整庫覆蓋會不會直接失敗？補 `TestRestore_WaitsForAConcurrentWriter`（另一條連線持有寫入交易 700ms）：覆蓋會等（連線的 `busy_timeout`），還原成功、資料正確。
- **L2（記錄不修）** 「備份比目前新」的拒絕發生在自動快照**之後**（既有順序），被拒時會多留一份快照檔；資料庫不受影響。

### Discovery Triage

- N/A — no out-of-scope work discovered.

### File List

- `apps/api/internal/services/backup_service.go`
- `apps/api/internal/services/backup_service_test.go`、`backup_posters_test.go`、`backup_restore_real_test.go`（新）
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-25 | 建單（SM）：由 `disc-2026-09-restore-fails-on-real-database`（P0）升級；裁定改用 SQLite online backup API 整庫覆蓋＋覆蓋後跑 migration；schema 版本改成真的並讀備份資料庫本身 |
| 2026-09-25 | 實作完成：online backup API 整庫覆蓋＋migration、真實 schema 版本、真實 schema 測試夾具；本機冒煙還原（含海報）成功；狀態 review |
| 2026-09-25 | /ship CR：0H／0M／2L；補並行寫入測試 |
| 2026-09-25 | PR #549 合併（`69ee7078`），狀態改 done |
