# Story: Bugfix E — qBT 錯誤/暫停種子被計成「排隊中」（Activity 誠實分桶）

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Source:** NAS BUG HUNT（2026-07-13, party-mode）P1；2026-08-19 What'Sub 對抗重審列為無悔四項之 N2。
**Risk: 🟢 LOW** —— 單點算式修正＋additive 欄位；比照 `project_qbt_state_mapping`（Sonarr/Radarr 業界標準）。

---

## Story

As a NAS owner opening the Activity hub every day,
I want errored and paused torrents counted in their own buckets instead of being swept into「排隊中」,
so that 3,068 個壞掉的種子不再偽裝成正常排隊 —— Activity 與 `/downloads/counts` 對同一批種子說同一句話。

---

## Context —— 缺陷機制（live NAS 實證）

`ActivityService.downloadsSection`（`activity_service.go:190-193`）：

```go
queued := c.All - c.Downloading - c.Completed - c.Seeding
```

減式漏掉 `Error` 與 `Paused` → ERRORED / PAUSED 全數落入 queued。live NAS 實測：activity 回報 queued **3102**（= 3068 error + 34 paused），而 `/downloads/counts` 同時正確回報 `error: 3068` —— **兩個端點對同一批 3,641 顆種子自相矛盾**。`DownloadCounts` 結構早就有 `Paused`/`Error` 欄位（`qbittorrent/torrent.go:130-137`），資料源齊備，純算式與呈現層缺陷。

## ⚖️ 範圍裁定（Alexyu 2026-08-20）—— 法 B「補 Queued 桶」

開工前的事實查核推翻了 AC #4 的前提，裁定改採**法 B**：

1. **推翻點**：`GetDownloadCounts`（`download_service.go:190`）把 `StatusQueued` 併進 `counts.Paused`，所以 `/downloads/counts` **並非「本來就是對的」**——它的 `paused` 虛胖（含佇列），而且下載頁「已暫停」tab 的**數字**（counts，含佇列）與**清單**（qBT `FilterPaused`，不含佇列）本來就對不上，是同一家族的誠實性缺陷。
2. **若照原 AC #1 減式**：`queued` 會恆為 ~0（各桶互斥且總和＝All，減完無殘值）＝**死標籤**，真正排隊的種子改顯示成「暫停」——用一個謊換另一個謊。
3. **法 B 定案**：`DownloadCounts` 新增 `Queued` 桶（`StatusQueued` 不再併入 `Paused`），activity 各桶**直接讀自己的計數**，不再靠減式推導。一次修好三件事：排隊中維持活標籤、暫停數不再虛胖、下載頁 tab 數字與清單一致。
4. **AC #4 紅線調整**：`/downloads/counts` 從「不動」改為「**additive `queued` key ＋ `paused` 語意收窄**」，並帶 `[@contract-v1]` 首次形式化（該端點原本無 stamp，故無既有 bump 欠務）。qBT 對映層（`MapQBState`）仍**一行未動**，原紅線的實質意圖保留。

## Acceptance Criteria

### AC #1 — BE 分桶修正（依裁定改為法 B）
- ~~`queued := c.All - c.Downloading - c.Completed - c.Seeding - c.Error - c.Paused`~~ → **各桶直接讀計數**：`Queued: c.Queued`、`Errored: c.Error`、`Paused: c.Paused`。減式與 clamp 一併移除——各桶互斥且總和＝`All`（`DownloadCounts` 註解載明、counts 測試釘住），值本來就 ≥ 0。
- `DownloadCounts`（`qbittorrent/torrent.go:130`）新增 `Queued int \`json:"queued"\``；`GetDownloadCounts` 的 `StatusQueued` 由 `Paused++` 改為 `Queued++`。
- `DownloadsSection`（`activity_service.go:93-99`）新增 additive 欄位 `Errored int \`json:"errored"\``、`Paused int \`json:"paused"\``。既有 key（`status/downloading/queued/total/error`）**一個不動**（`error` 是字串錯誤訊息欄位，新計數欄位取名 `errored` 避開撞名）。additive、不 bump；wire-shape 測試釘 snake_case key ＋ 釘 `error`（訊息）在健康 section 缺席。

### AC #2 — FE Activity hub 誠實呈現
- `ActivityHub.tsx:198` detail 字串擴為：`{downloading} 個進行中 · {queued} 個排隊` ＋ **當 `errored > 0` 時**追加 ` · {errored} 個錯誤`（`--error` 色調，與既有 badge token 一致；`paused > 0` 同理顯示「暫停」，`--text-muted`）。零值不顯示 —— 健康系統的畫面 byte-unchanged。
- 側欄狀態 strip（Epic 0 F2）若同源消費此 section，同步繼承；若另有算式，本 story 僅修 ActivityHub，strip 差異記入 Dev Notes 供 follow-up。

### AC #3 — 測試
- BE：table-driven 分桶測試 —— error/paused/混合/全零四案；queued 不再含 error/paused；additive 欄位序列化 key 斷言。
- FE：`ActivityHub.spec.tsx` —— errored>0 顯示、=0 不顯示（回歸釘）。
- 既有 activity/downloads 測試全綠。

### AC #4 — 範圍紅線
- ~~不動 `/downloads/counts`（它本來就是對的）~~ → **裁定改寫**：可動，但僅限 additive `queued` key ＋ `paused` 語意收窄（見上方裁定 §4）。
- 不動 qBT state→bucket 對映本身（`project_qbt_state_mapping` 已定，對映層非本案）。
- 效能問題（>1.1s）屬 bugfix-f，不在此案。

## Tasks / Subtasks

- [x] Task 1: AC #1 分桶修正 —— `DownloadCounts.Queued` 桶＋`GetDownloadCounts` 改判＋`DownloadsSection` additive 欄位＋各桶直讀（取代減式）
- [x] Task 2: AC #2 ActivityHub 呈現（零值不顯示；`ActivityRow.detail` 由 `string` 放寬為 `ReactNode` 以承載分色 segment）
- [x] Task 3: AC #3 測試（BE table-driven 5 案＋wire-shape 2 案；FE 2 案＋健康行回歸釘；2 個釘住舊行為的既有測試已更新）

## Dev Agent Record

### Context / 開工前事實查核（Rule 24 discovery triage）

驗證了 story 引用的每一處程式碼快照 —— 全部屬實（`activity_service.go:190` 減式、`DownloadsSection` 形狀、`DownloadCounts` 已有 Paused/Error、`ActivityHub.tsx:198` detail 字串）。**但**多查到三件 story 未預見、且改變作法的事：

1. **`StatusQueued` 被併進 `counts.Paused`**（`download_service.go:190`，且 `TestDownloadService_GetDownloadCounts_UnmappedStatusesNotCounted` 明確釘住 `queuedDL → paused`）→ 原 AC #1 減式會讓「排隊中」變死標籤。**→ 呈報裁定，採法 B**（見上方裁定段）。
2. **下載頁「已暫停」tab 的數字與清單本來就不一致**：count 來自 `counts.Paused`（含佇列），list 來自 qBT `FilterPaused`（不含佇列）。法 B 順帶修好。
3. **各桶互斥且總和＝All**：`GetDownloadCounts` 是 `switch`（每顆種子只進一桶），`MapQBState` 對未知 state 有 `default` → 沒有未計入的殘值。這是「各桶直讀、免 clamp」成立的前提，已寫進 `DownloadCounts` 註解並由 counts 測試的 `sum == All` 斷言釘住。

### 側欄狀態 strip（AC #2 第二條）

查核 `status_summary_service.go:135-146`：`queueSection` 只消費 `counts.Downloading` 與 `counts.All`，**沒有 queued 算式**，故無「同源繼承」可做、也無差異需要 follow-up。本 story 僅修 ActivityHub，符合 AC #2 的第二分支。

### Rule 24 ① —— 順手修掉的既有缺陷

`GenerationWorkspaceV2.tsx:494` 的 `activeItemProgress` 字面值缺 `partial`/`englishKeptBlocks`（bugfix-j 加進 `GenerationProgressState` 時漏更新此處）。`nx typecheck web` 抓到（該 target 在乾淨 main 上已有 148 個既有錯誤、非 CI 閘門，故 #239 CI 綠不代表無此錯）。屬本人前一個 story 引入、2 行可修 → **FIX**：補上 not-partial 預設值（`partial: false`、`englishKeptBlocks: null`）。修後 148 → 147，本 story 觸及的檔案零 typecheck 錯誤。

### Rule 20 —— Contract

- **新 stamp**：`DownloadCounts` 帶 `[@contract-v1]`（`GET /api/v1/downloads/counts` 的 wire shape 首次形式化 —— 查核過該端點與 handler 原本**無任何 stamp**，故非 bump、無下游 stale-mark 欠務）。
- **`DownloadsSection`**：additive 欄位（`errored`/`paused`），既有 key 零變更 → 不 bump。

### 已知留白（FILE，不在本案）

- 下載頁**無「佇列中」tab**：qBT API 沒有 server-side `queued` filter，加 tab 需 client-side 過濾 → 超出本案。佇列種子仍可在「全部」看到。
- `TestGenerationBatch_RequestedBudgetOverridesDefault` 在全套跑時偶發失敗（隔離跑過）—— 既有已立案 flake（`preexisting-flake-generation-batch-sse-drain`），與本案無關。

### File List

- apps/api/internal/qbittorrent/torrent.go（`DownloadCounts.Queued` ＋ 互斥/總和契約註解 ＋ `[@contract-v1]`）
- apps/api/internal/services/download_service.go（`StatusQueued → counts.Queued++`）
- apps/api/internal/services/activity_service.go（`DownloadsSection` additive 欄位；`downloadsSection` 各桶直讀取代減式）
- apps/api/internal/services/activity_service_test.go（+3 測試；1 個既有測試的 fixture 改為自洽）
- apps/api/internal/services/download_service_test.go（釘住 `queuedDL → Queued` 的新語意）
- apps/web/src/services/activityService.ts（`DownloadsSection` 型別）
- apps/web/src/services/downloadService.ts（`DownloadCounts.queued`）
- apps/web/src/components/activity/ActivityRow.tsx（`detail: string → ReactNode`）
- apps/web/src/components/activity/ActivityHub.tsx（分色 segment，零值不顯示）
- apps/web/src/components/activity/ActivityHub.spec.tsx（+2 測試＋健康行回歸釘＋fixture）
- apps/web/src/components/downloads/DownloadFilterTabs.spec.tsx（fixture）
- apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx（Rule 24 ① 修復）

### 驗證

- Go 34 packages 全綠（唯一失敗＝上述既有 flake，隔離跑過）· gofmt / go vet / staticcheck 2026.1（`internal/services` + `internal/qbittorrent`）皆乾淨
- web：activity 12 測試、downloads 13 檔 130 測試全過 · `nx lint web` 綠 · typecheck 觸及檔案零錯誤

## Change Log

| 日期 | 變更 |
| --- | --- |
| 2026-08-20 | 開工前事實查核推翻 AC #4 前提（`/downloads/counts` 的 `paused` 虛胖含佇列、tab 數字與清單不一致）→ 呈報 Alexyu，裁定採**法 B 補 Queued 桶**。實作：`DownloadCounts.Queued` 新桶 `[@contract-v1]`、activity 各桶直讀取代減式、`errored`/`paused` additive 欄位、ActivityHub 分色 segment（零值不顯示）。+7 測試（BE 5＋2、FE 2）＋2 個舊行為測試更新。Rule 24 ① 順手修掉 bugfix-j 遺留的 `GenerationWorkspaceV2` 型別缺口。Status → review。 |
