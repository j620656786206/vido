# Story bugfix：備份把你上傳的海報一起存起來，還原時放回去

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who restores Vido from a backup after the NAS breaks,
I want the posters I uploaded to come back together with the database,
so that a restore does not quietly turn every hand-picked poster into「還沒有海報」.

## Context

升級自 `disc-2026-09-backup-excludes-uploaded-posters`（`bugfix-poster-orphan-sweep` 建單時發現）。上傳的海報**沒有第二份**：它們只存在 `data/posters/`，而備份只打包資料庫。

### 🔴 查到的事（main `cf4e6503`；行號皆為現況）

1. **備份只放兩樣東西**：`createTarGz`（`services/backup_service.go:280-362`）寫入 `vido.db` 與 `manifest.json`；`createManifest`（`:272-278`）只有 schema 版本、時間、app 版本。
2. **同一個 `createTarGz` 也被「還原前自動快照」用**（`createAutoSnapshot`，`:560-629`）——所以改它，快照也會一起帶海報。
3. **還原流程**（`RestoreBackup`，`:421-538`）：驗 checksum → 自動快照 → 解壓到暫存資料夾（`extractTarGz`，`:631-704`，已擋路徑穿越、單檔 10 GB 上限）→ 檢查 schema → `replaceDatabase` → 成功訊息「還原完成，資料庫已恢復」（`:532`）。失敗時 `rollbackFromSnapshot`（`:802`）。
4. **`extractTarGz` 對 `TypeReg` 直接 `os.Create(targetPath)`，沒有先建父資料夾**（`:670-671`）——所以 `posters/x.jpg` 這種項目要嘛前面先寫一個 `TypeDir` 項目，要嘛解壓時 `MkdirAll(filepath.Dir(...))`。
5. **海報檔名規則**已集中在 `images.IsPosterFileName`（PR #545）；上傳中的暫存檔叫 `<name>.jpg.bak`。
6. **`BackupService` 不知道海報資料夾在哪**：`NewBackupService(db, repo, backupDir, schemaVersion)`（`:68`，`main.go:191`）；海報資料夾是 `main.go:204` 的 `posterDir`。
7. **前端**：`BackupManagement.tsx:107` 還原成功時自己顯示固定字串「還原完成，資料庫已恢復」；還原確認框（`RestoreConfirmDialog.tsx`）只說「還原資料」，不必改。設計稿沒有這兩句的稿（查過 `ux-design.pen`）。
8. **孤兒清理**（PR #545）每 45 分鐘把「沒有列指到」的海報刪掉，10 分鐘內的新檔不碰。

### ⚖️ 建單裁定（2026-09-25，SM；Alexyu 可在 review 推翻）

1. **備份一併收 `posters/`**，只收符合 `images.IsPosterFileName` 的檔（不收 `.bak`、不收其他檔名、不進子資料夾、不跟符號連結）。JPEG 已壓縮過，gzip 幫不上忙，但每張海報約數百 KB，數量是「使用者手動上傳的張數」——不會大。
2. **還原採「放回」不採「清空再放」**：備份裡的每張海報寫回 `data/posters/`（同名就覆蓋）；目前資料夾裡**多出來的檔不刪**——還原後的資料庫若沒有列指到它們，孤兒清理會在下一輪收掉。理由：還原失敗要回滾時，「清空」會讓回滾多一件要復原的事；而「多留幾十分鐘的檔」無害。
3. **舊備份（沒有 `posters/` 的）照常能還原**，目前的海報資料夾完全不動；manifest 多一個 `posters` 計數欄位，讀不到就當 0。
4. **放回的海報時間蓋成「現在」**，讓孤兒清理的 10 分鐘保護罩住還原剛結束的那段時間（跟 `parkPoster` 同一個道理）。
5. **成功訊息說實話**：有放回海報時，後端訊息與前端提示改成「還原完成，資料庫與 N 張上傳的海報已恢復」；沒有時維持原句。

## Acceptance Criteria

1. **備份收海報。**
   - `BackupService` 知道海報資料夾（建構子或 setter 注入，`main.go` 傳 `posterDir`；沒設定＝不收，既有測試不受影響）。
   - `createTarGz` 在 `vido.db`、`manifest.json` 之外，加入 `posters/` 目錄項目（`TypeDir`）與每個符合規則的海報檔 `posters/<name>`；manifest 多 `posters: <張數>`。
   - 海報資料夾不存在 → 當 0 張，備份照樣成功。單一海報讀取失敗 → 備份**失敗**（寧可沒備份也不要一個自稱完整、其實缺檔的備份），錯誤訊息寫出檔名。
   - 自動快照（`createAutoSnapshot`）走同一條路，也帶海報。
   - 測試：備份含 `posters/m1.jpg`、`posters/m1-thumb.jpg`，不含 `m1.jpg.bak`、`notes.txt`、子資料夾、符號連結；manifest 的 `posters` 正確；沒有海報資料夾也成功。
2. **還原放回海報。**
   - 解壓後，若暫存資料夾裡有 `posters/`：只取第一層、符合 `IsPosterFileName` 的一般檔，**先寫到海報資料夾裡的暫存名再 rename**（不會留下半張圖），時間蓋成現在；其他項目忽略。
   - 放回海報發生在 `replaceDatabase` **成功之後**；放回失敗 → 記錯誤、結果仍算「完成」但訊息註明「有 N 張海報沒放回」（資料庫已經換好了，不應該為了海報回滾整個資料庫）。
   - 沒有 `posters/` 的舊備份 → 海報資料夾一個檔都不動。
   - `extractTarGz` 處理 `posters/x.jpg` 時父資料夾存在（`TypeDir` 項目先出現；另加 `MkdirAll(filepath.Dir(target))` 當保險）；既有的路徑穿越防護不變，並補一條「`posters/../../x`」的測試。
   - `RestoreResult` 多 `postersRestored int`（JSON `posters_restored`）；訊息依 ⚖️ #5。
   - 測試：新備份還原後海報回來且內容相同；目前多出來的海報沒被刪；舊格式備份還原不動海報；放回時間是「現在」；一張放回失敗時訊息正確、資料庫仍是還原後的。
3. **前端**：`BackupManagement.tsx` 還原成功提示——`posters_restored > 0` 時顯示「還原完成，資料庫與 N 張上傳的海報已恢復」，否則維持「還原完成，資料庫已恢復」；服務層型別補欄位（snake → camel 照既有轉換）。測試兩種情況各一條。
4. **不准回歸**：既有備份／還原／驗證／排程／刪除測試全綠；`VerifyBackup` 仍以整個檔案 checksum 驗證（海報在檔案裡，自然被涵蓋）；e2e `backup*.spec.ts`（dev 先 grep）全綠。
5. **CI**：`pnpm nx test api`、`pnpm nx test web`、`pnpm run lint:all` 綠；紅／守（Rule 16）；mutation check（至少：不收海報、收了 `.bak`、還原時不放回、放回不蓋時間、舊備份還原時動到海報資料夾）。

## Tasks / Subtasks

- [x] **Task 1 — 備份收海報（AC: #1）**：注入海報資料夾、`createTarGz`／manifest、測試
- [x] **Task 2 — 還原放回海報（AC: #2）**：解壓父資料夾、放回（暫存名＋rename＋蓋時間）、`RestoreResult.PostersRestored`、訊息、測試
- [x] **Task 3 — `main.go` 接線（AC: #1）**
- [x] **Task 4 — 前端成功提示（AC: #3, #4, #5）**

## Dev Notes

### 這張的重點

- **上傳的海報沒有第二份**——這張就是給它第二份。
- **還原先顧資料庫**：海報放回失敗不回滾資料庫，只說清楚。
- **跟孤兒清理和平共處**：放回的檔蓋成現在；多出來的舊檔留給清理。

### 上游契約（Rule 20）

- 本張不消費任何 `[@contract-v*]`。`RestoreResult` 多一個欄位屬新增（舊前端忽略即可）；備份檔格式向下相容（舊備份照樣能還原）。

### 不要做的事

- 不要在還原時清空海報資料夾。
- 不要收 `.bak` 或其他檔名。
- 不要因為海報放回失敗而回滾資料庫。
- 不要改還原確認框的文案（它說的是「還原資料」，本來就對）。

### 已知陷阱

- **tar 項目順序**：`posters/` 的 `TypeDir` 要在檔案項目之前寫；解壓時仍加 `MkdirAll` 保險。
- **符號連結**：收檔時用 `os.ReadDir`＋`Type().IsRegular()`。
- **跨裝置 rename**：暫存檔要建在**海報資料夾裡**（同一個檔案系統），不要建在 `os.TempDir()`，否則 rename 會失敗。
- **gh**：`GH_TOKEN=$(gh auth token --user j620656786206)`。

### Source tree

```
apps/api/internal/services/backup_service.go（+test）   ← Task 1, 2
apps/api/internal/models/（RestoreResult）               ← Task 2
apps/api/cmd/api/main.go                                ← Task 3
apps/web/src/components/settings/BackupManagement.tsx（+spec）、services（型別） ← Task 4
```

### Cross-Stack Split Check

後端 3、前端 1 → 不拆。

### Time-dependent visual coverage

- N/A — 前端只改一句成功提示（無視覺夾具變動）；後端時間用實際 `time.Now()`，測試以「在一分鐘內」斷言。

### References

- [Source: `apps/api/internal/services/backup_service.go:68-76, 79-167, 272-362, 421-538, 560-629, 631-704, 802`；`cmd/api/main.go:191, 204`]
- [Source: `apps/api/internal/images/filename.go`（`IsPosterFileName`）；`services/poster_orphan_sweeper.go`（10 分鐘保護）；`services/metadata_edit_service.go`（`parkPoster` 蓋時間）]
- [Source: `apps/web/src/components/settings/BackupManagement.tsx:107`；`RestoreConfirmDialog.tsx`]
- [Source: `sprint-status.yaml` → `disc-2026-09-backup-excludes-uploaded-posters`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1**：`BackupService.SetPosterDir`（沒設定＝不收，既有測試不變）；`createTarGz` 在 `vido.db` 之後寫 `posters/`（`TypeDir`）與每個合規海報（`posterFiles`：第一層、`Type().IsRegular()`、`images.IsPosterFileName`——不收 `.bak`、雜檔、子資料夾、符號連結）；manifest 多 `posters`（`omitempty`，舊檔讀不到＝0）。讀不到某張海報 → 整個備份失敗並寫出檔名。自動快照走同一個 `createTarGz`，自然也帶海報。
- **Task 2**：`extractTarGz` 寫檔前 `MkdirAll(filepath.Dir(target))`（不依賴項目順序；既有路徑穿越防護不變，補「`posters/../../x`」測試）。`replaceDatabase` 成功後 `restorePosters`：只取合規的一般檔，**先寫到海報資料夾裡的暫存名（`.<name>.restoring-*`，同一檔案系統）再 rename**；只新增／覆蓋、不刪多出來的檔；失敗只記錯並計數，不回滾資料庫。`RestoreResult` 多 `posters_restored`／`posters_failed`，訊息依裁定 #5。**偏離裁定 #4**：原想用 `Chtimes` 把放回的檔蓋成「現在」，mutation 證明那行多餘——複製出來的是新檔，時間本來就是現在；已刪掉那行，改成註解寫明「不要沿用備份檔裡的時間」，測試仍斷言時間是現在（防止未來有人加上保留時間）。
- **Task 3**：`main.go` 在 `posterDir` 定義後呼叫 `backupService.SetPosterDir(posterDir)`。
- **Task 4**：`RestoreResult` 型別補 `postersRestored?`／`postersFailed?`；`BackupManagement` 成功提示：有失敗 → `warn`「還原完成，資料庫已恢復；有 N 張上傳的海報沒放回，原因見系統日誌」；有放回 → 「還原完成，資料庫與 N 張上傳的海報已恢復」；否則原句。
- **本機冒煙（真的 API）**：建一部電影 → 上傳海報 → `POST /settings/backups` → 備份檔內容 `vido.db`、`posters/`、`posters/<id>-thumb.jpg`、`posters/<id>.jpg`、`manifest.json`（`"posters": 2`）。✅
- 🚨 **冒煙同時發現既有 P0：還原在真實資料庫上會失敗**。刪掉海報檔後 `POST …/restore` → `RESTORE_DB_FAILED: clear table media_libraries: FOREIGN KEY constraint failed`，自動快照回滾也失敗（`restore_db is already attached`）。失敗點在 `replaceDatabase`（本張沒有改動），發生在放回海報之前。依 Epic 9c AI-2「非小修就立案」→ `disc-2026-09-restore-fails-on-real-database`（P0，原因與修法方向已寫入）。**所以本張「還原放回海報」只有單元測試證明（測試用的資料庫沒有外鍵），真實環境要等那張修好才走得到。** 本機資料庫在交易回滾後完好（之後刪改正常）；冒煙用的 API 已關閉，port 8080／8081／4200 無殘留。
- 🔗 **AC Drift: FOUND** — `6-x` 備份相關 story 的「備份只含資料庫」隱含契約 → 現在也含 `posters/`；還原成功訊息多了海報張數（`RestoreResult` 新增欄位，舊前端忽略即可）。
- 📎 **Contract Stamps: NONE**。
- 🎭 **A11y Pre-Flight: PASS**（只改 `BackupManagement` 的提示文字與 tone；沿用既有的訊息區）。
- 🎨 **UX Verification: SKIPPED**（無設計稿涵蓋這兩句；只改文字，版面不變）。
- **測試**：`nx test api` 綠；`nx test web` **289 files／4303 tests** 綠；`lint:all`（0 error）、typecheck 綠。新增 Go 測試 6、web 測試 3。
- **Mutation 7／7 紅**（第 8 條「放回不蓋時間」存活 → 證明那行多餘，已刪）：備份不收海報、收任何檔、還原不放回、解壓不建父資料夾、失敗不回報、前端不顯示失敗句。

### 🔍 /ship Adversarial Review（2026-09-25）

0 HIGH／0 MEDIUM／2 LOW，都記錄不修：
- **L1** 還原當到一半當機，可能留下 `.<name>.restoring-*` 暫存檔；它不符合海報檔名規則，孤兒清理不會收，但也不會被服務或備份。量極小。
- **L2** 備份進行中剛好有人重新上傳同一張海報（寫檔會先截斷）→ tar 寫入大小不符 → 該次備份失敗（不會產生壞備份）；下次排程或手動重試即可。

### Discovery Triage

- `disc-2026-09-restore-fails-on-real-database`（**P0**，本機冒煙發現，已寫進 sprint-status）。

### File List

- `apps/api/internal/services/backup_service.go`、`backup_posters_test.go`（新）
- `apps/api/internal/models/backup.go`
- `apps/api/cmd/api/main.go`
- `apps/web/src/services/backupService.ts`
- `apps/web/src/components/settings/BackupManagement.tsx`（+spec）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-25 | 建單（SM）：由 `disc-2026-09-backup-excludes-uploaded-posters` 升級；裁定備份收 `posters/`、還原採放回不清空、舊備份相容、放回失敗不回滾資料庫 |
| 2026-09-25 | 實作完成：備份收海報、還原放回、成功提示；本機冒煙發現還原在真實資料庫上會失敗 → 立 P0 disc；狀態 review |
| 2026-09-25 | /ship CR：0H／0M／2L（記錄不修） |
| 2026-09-25 | PR #547 合併（`a98f6bac`），狀態改 done |
