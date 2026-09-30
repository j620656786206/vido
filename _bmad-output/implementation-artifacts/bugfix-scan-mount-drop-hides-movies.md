# Story: bugfix-scan-mount-drop-hides-movies — NAS 資料夾暫時連不到時，電影不會從媒體庫永久消失

Status: done

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一張」。調查 disc-2026-09-removed-item-detail-still-served 時發現、本機 :8090 實測重現。
     行號為 main `9a6ce84f`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為把片庫放在 NAS 上的人，
當 NAS 的資料夾暫時連不到（重開機、掛載點掉了、排程掃描剛好碰上）時，
我的電影不能因此從媒體庫消失，
而且資料夾回來之後，下一次掃描就要全部回來。

## 背景（查到的事）

- **實測重現（:8090 種子資料）**：媒體庫 15 部電影 → 把電影資料夾暫時移走、掃描 → **0 部** → 放回資料夾、掃描 → **還是 0 部**，再掃一次仍是 0。
- 掃描最後會跑 `detectRemovedFiles`（`apps/api/internal/services/scanner_service.go:782-821`）：每部電影 `os.Stat` 檔案，**找不到就 `MarkRemoved`**（`is_removed = 1`，`repository/scan_state_update.go:72-79`）。它不管整個資料夾是不是根本連不到——掛載點掉了就把全部標成已移除。
- **標了就回不來**：檔案回來後，掃描走到 `processVideoFile` 的「已經有這筆」分支（`scanner_service.go:589-614`），大小和時間都沒變 → 算「略過」直接 return，`is_removed` 永遠不會被清掉。整個 codebase 沒有任何地方把 `is_removed` 改回 0（`git grep "is_removed = 0,"` 只有查詢條件）。
- 媒體庫、首頁、搜尋都用 `is_removed = 0` 過濾（`movie_repository.go:568,610-614,983,1034,1115,1144`），所以這些片在 app 裡整個看不見。
- Docker 的常見情況：NAS 分享沒掛上時，容器裡的掛載點是**存在但空的資料夾**——所以「資料夾存在」不代表「片子真的被刪了」。
- 只有電影會被標移除（`MarkRemoved` 只有 movies 有）；影集不在本張。

## Acceptance Criteria

1. **檔案回來就恢復**：掃描遇到已標移除的電影、而檔案確實在，就清掉移除標記（只寫這一欄），算在「更新」裡；檔案大小有變時照原本流程重新解析。
2. **連不到的資料夾不判刑**：這次掃描裡**連不到**的媒體資料夾，底下的電影一律不標移除。
3. **空的掛載點不判刑**：資料夾存在、但這次在裡面**一個影片都沒找到**，而資料庫記得它底下有電影 → 視為掛載問題，底下的電影不標移除，並寫一行警告 log。
4. 其他情況維持原本行為：資料夾好好的、只有某部片的檔案被刪掉 → 那一部照樣標移除；不在任何媒體資料夾底下的電影（例如資料夾已從設定移除）照舊處理。
5. 測試（先紅後綠）：恢復、資料夾連不到、空掛載點、單一檔案被刪仍標移除；本機 :8090 重跑上面的實測：資料夾移走再放回 → 15 部都在（移走期間也不會變 0）。
6. 已經被誤標的片：修好後下一次掃描會自動恢復（因為 AC #1），不需要另外的修復工具。

## Tasks / Subtasks

- [x] Task 1 — repository `RestoreRemoved`＋mock（AC #1）
- [x] Task 2 — scanner：恢復、記錄每個資料夾「連得到嗎／找到幾個」、`detectRemovedFiles` 跳過連不到與空的資料夾＋測試（AC #1–#4）
- [x] Task 3 — 本機 :8090 實測；全量測試、lint

## Dev Notes

- 判斷「電影在哪個資料夾底下」：用路徑前綴（`filepath.Clean`＋分隔符），資料夾若連得到也比對解析 symlink 後的路徑（電影的 `file_path` 存的是解析過的路徑，macOS 的 `/var` → `/private/var` 就是例子）。
- 不改 `detectRemovedFiles` 對「不在任何媒體資料夾底下」的處理。
- `disc-2026-09-removed-item-detail-still-served`（詳情頁要不要對已移除的片顯示）留在 backlog，需要產品裁定；本張修的是「根本不該被標移除」。

### Time-dependent visual coverage

- N/A — backend only.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED：資料夾連不到的測試在舊碼上直接 panic（`MarkRemoved` 被呼叫）；恢復測試在舊碼上失敗（檔案沒變就「略過」）。
- GREEN：`RestoreRemoved`（窄寫一欄）；掃描記錄每個資料夾「連得到嗎、看到幾個影片、有沒有讀取錯誤」；`detectRemovedFiles` 對不可信資料夾（依路徑**和**媒體庫 id）底下的電影不標移除，並寫一行 `SCANNER_ROOT_UNTRUSTED` 警告。
- 本機 :8090 重跑實測：15 → 資料夾移走 15 → 空掛載點 15 → 放回 15 → 刪一部 14 → 放回 15（修前：15 → 0 → 0）。
- 種子資料：`is_removed` 的兩部片原本也寫了檔案，修好「檔案在就恢復」後會被第一次掃描恢復——種子改成不寫（並刪掉舊種子留下的檔案）。
- 對抗式 review（subagent）：無 High。修掉 2 條 Med＋1 條 Low：
  - M1 巢狀資料夾（`/media` 與 `/media/movies` 都設定）時內層被誤判為空，真的刪掉的片永遠不標——改成「看到的影片」包含已被外層算過的。
  - M2 資料夾內是指向 NAS 的 symlink、NAS 掉線時路徑比對不到——改成同時依電影所屬媒體庫判斷。
  - L3「掃到一半出錯」原本觸發不了（walk callback 吞錯）——改成資料夾內任何讀取錯誤或失效 symlink 都讓該資料夾不可信。
  - 不修（記錄）：兩個資料夾之間的時間差（第一個掃完後才掉線）仍可能被標移除，但下一次掃描會恢復；資料夾裡**最後一部**片被刪時不會被標移除（空資料夾視為掛載問題），只會每次掃描寫警告——刻意取捨，寧可多留不可誤藏。
- 全量：`pnpm nx test api` 綠（含 `-race` 的掃描測試）；`lint:all` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（無 API 變動）。
- 🎭 A11y Pre-Flight: N/A（後端）。🎨 UX Verification: N/A。

### Discovery Triage

- 無新立案。原本的 `disc-2026-09-removed-item-detail-still-served` 仍在 backlog（詳情頁對已移除的片要不要顯示，需產品裁定）。

### File List

- apps/api/internal/services/scanner_service.go、scanner_mount_drop_test.go（新）
- apps/api/internal/repository/scan_state_update.go、scan_state_update_test.go、interfaces.go
- apps/api/internal/testutil/mocks.go
- apps/api/internal/services/enrichment_nfo_test.go、parse_queue_service_test.go（mock 補方法）
- apps/api/cmd/seed/main.go
- _bmad-output/implementation-artifacts/bugfix-scan-mount-drop-hides-movies.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-30 | create-story（SM Bob）。 |
| 2026-09-30 | dev-story（Amelia）＋對抗式 review → review。 |
| 2026-09-30 | PR #617 合併 → done。 |
