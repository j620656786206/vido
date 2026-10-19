# Disc：裁掉黑邊的寬銀幕 4K 片（例 3828×1588）算 4K；前端和 Go 用同一套解析度等級規則

Status: done

**Source:** Rule 24 lane ③，2026-10-19 由 `disc-2026-10-library-list-triage-b1`（dev-story）發現：Go `resolveResolution` 只對 4K／1440p 做寬度補判，b1 在前端 `resolutionTier` 多補了 1080p／720p 兩級，兩邊從那天起規則不一樣。
⚖️ **Alexyu 2026-10-19：`3828x1588` 要算 4K**（寬 3828＝4K 片裁掉上下黑邊，左右也裁了幾個像素）→ 範圍改成前後端一起改，優先度 P3→P2。門檻由 dev 定，並列出正式機所有片子改前改後的等級。

## 查到的事（2026-10-19，本 session 讀過的程式碼與正式機唯讀資料）

### 誰在判定等級

| 位置 | 吃什麼 | 現在的規則 | 結果存在哪 |
|---|---|---|---|
| 前端 `resolutionTier` → `tierFromSize`（`apps/web/src/utils/resolutionLabel.ts:27-48`） | DB 裡的 `video_resolution` 字串；`寬x高` 才換算，`4K`／`1080p` 這類標籤原樣（`2160p`→`4K`） | **以高為主**，寬度補判四級：`高≥2160 或 (寬≥3840 且 高≥1080)`→4K、`高≥1440 或 (寬≥2560 且 高≥1080)`→1440p、`高≥1080 或 (寬≥1920 且 高≥720)`→1080p、`高≥720 或 (寬≥1280 且 高≥480)`→720p、`高≥480`→480p、其餘 `${高}p` | 不存，每次畫面算。用的人：媒體庫列表「規格」那行（`utils/libraryListRow.ts:18`）、詳細頁「檔案資訊」解析度徽章（`formatResolutionBadge`，`resolutionLabel.ts:70`，#833） |
| Go `resolveResolution`（`apps/api/internal/services/nfo_reader_service.go:312-330`） | NFO `<streamdetails><video><width>/<height>`（`:291-294`） | 以高為主，寬度補判**只有 4K、1440p 兩級**：`1920x800`→**720p**、`1280x536`→**480p** | **存進 DB**：`movies.video_resolution` 寫標籤（`4K`、`720p`…），原始寬高丟掉（`enrichment_service.go:946-947`） |
| ffprobe（`ffprobe_service.go:243-244`） | 實際檔案 | **不判定**，原樣寫 `寬x高` | 存進 DB `movies.video_resolution`／`series.video_resolution`（`enrichment_service.go:797-798`） |

- **媒體庫沒有解析度篩選**：`library_handler.go:78-218` 的 query 參數只有 `genres`／`year_min`／`year_max`／`unmatched`／`subtitle_status`／`chinese_subtitle`／`limit`…，沒有 resolution；前端 `LibraryBrowseV2` 也沒有解析度篩選（`grep -rn "resolution" apps/web/src/components/library` 只剩 A7p-D 文案註解裡的「（4K）」例子）。→ 原任務說的「篩選 4K 會出現它」**目前做不到也不用對齊**：沒有這個篩選。設計稿 A7p-D 的空狀態文案拿「4K」當例子，但篩選本身沒排進任何 story。
- **第三個在讀解析度的地方（本單不改，立案）**：字幕搜尋把詳細頁的 `videoResolution` 原樣送給後端（`LocalDetailV2.tsx:451` → `ManageSubtitleDialogV2.tsx:674`／`SubtitleSearchDialog.tsx:125`），Go 字幕評分 `normalizeResolution`（`apps/api/internal/subtitle/scorer.go:171-187`）只認 `2160p`／`4k`／`1080p`… 這類標籤，`3840x2160` 原樣回傳、永遠不等於字幕標的 `2160p` → **有標解析度的字幕一律拿 0.2 分，比沒標的（0.5）還低**。見 Discovery Triage。

### 正式機資料（2026-10-19 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）

- 影集 81 部 `video_resolution` 全空。電影：全部是 ffprobe 的 `寬x高` 形狀或空，**沒有任何 NFO 標籤**（所以 Go 規則的差異目前在正式機看不到，但下一次掃到帶 NFO 的片就會）。
- 逐一改前改後（「改前」＝今天前端顯示；Go 欄＝同樣寬高若從 NFO 進來、今天 Go 會存的標籤）：

| `video_resolution` | 片數 | 片名 | 前端改前 | Go 改前 | **改後（兩邊）** |
|---|---|---|---|---|---|
| `3840x2160` | 17 | — | 4K | 4K | 4K |
| `3840x2076` | 2 | 阿凡達：火與燼、真人快打II（1995） | 4K | 4K | 4K |
| `3840x1608` | 4 | 驚聲尖笑、驚聲尖叫7、動物方城市2、犯罪都市４（UHD） | 4K | 4K | 4K |
| `3840x1606` | 2 | 真人快打II、沙丘：第二部 | 4K | 4K | 4K |
| `3840x1604` | 2 | There's No Way Out、劇場版 鏈鋸人 蕾潔篇 | 4K | 4K | 4K |
| `3840x1600` | 3 | 尋秦記、鬥陣俱樂部、名模大間諜 2 | 4K | 4K | 4K |
| **`3828x1588`** | **1** | **捍衛任務：復仇芭蕾（Ballerina）** | **1440p** | **1440p** | **4K** ← 唯一會變的一部 |
| `2880x2160` | 1 | 牧師的最後誘惑（First Reformed，4:3 片幅） | 4K | 4K | 4K |
| `1920x1080` | 2 | — | 1080p | 1080p | 1080p |
| `1920x1036` | 1 | 極限返航（IMAX 比例） | 1080p | **720p** | 1080p |
| `1920x804` | 6 | 狸想世界、The Roundup: Punishment、犯罪都市４、認賊作爸媽、凶器、重返沉默之丘 | 1080p | **720p** | 1080p |
| `1920x802` | 1 | 橫衝直闖 | 1080p | **720p** | 1080p |
| `1920x800` | 1 | F1電影 | 1080p | **720p** | 1080p |
| `1920x780` | 1 | 超人（2025） | 1080p | **720p** | 1080p |
| 空 | 11 | — | 不顯示 | — | 不顯示 |

→ 使用者看得到的差別：**只有 Ballerina 一部從 1440p 變 4K**（列表規格行、詳細頁徽章）。Go 那欄的 720p 是「如果它們從 NFO 進來」的潛在錯，今天沒有片子中。

### 存下來的判定結果要不要重算？

- 不用遷移。ffprobe 那條存的是原始寬高，前端每次重算，改完立刻生效。
- NFO 那條存的是 Go 算好的標籤、原始寬高沒存——**正式機 0 筆**，沒有舊資料可以重算；改 Go 只影響之後掃進來的片。（要讓 NFO 也存原始寬高、讓前端一處判定，是另一個資料模型決定，本單不做，見「不要做的事」。）

## 規則（dev 定的門檻）

**一句話：把畫面轉成橫的（長邊當寬），長邊或短邊「任一」達到該級標準 16:9 尺寸的 90%，就算那一級；從 4K 往下找第一個成立的。**

| 等級 | 標準尺寸 | 長邊 ≥ | 或 短邊 ≥ |
|---|---|---|---|
| 4K | 3840×2160 | 3456 | 1944 |
| 1440p | 2560×1440 | 2304 | 1296 |
| 1080p | 1920×1080 | 1728 | 972 |
| 720p | 1280×720 | 1152 | 648 |
| 480p | 854×480 | 769 | 432 |
| 都不到 | — | — | 寫 `${短邊}p`（例 `640x360`→`360p`） |

為什麼這樣定：
- **以寬度為主是對的**：電影裁黑邊只裁高（`3840x1608`），寬才是片源等級。Sonarr／Radarr 判片源解析度也是「寬或高任一達標」（專案的 qBT 狀態對照已經以它們為準，見 memory `project_qbt_state_mapping`）。
- **留 10% 容差**：裁黑邊時常順手裁掉左右幾個像素（`3828`、`3832`、`3816`）；4:3 片幅（`2880x2160`）或裁過的（`2880x2152`）靠短邊進 4K。10% 的界線剛好把 21:9 帶魚螢幕錄影 `3440x1440`（長邊 3440 < 3456）留在 1440p，不會被誤升 4K。
- **直式影片**：先轉橫。`1080x1920` 今天被判成 1440p（高 1920≥1440），改後 1080p；`2160x3840`→4K；`720x1280`→720p。
- 480p 用 16:9 的 854 當標準寬，`720x480`／`720x576`（DVD）靠短邊、`640x360` 兩邊都不到→維持 `360p`（今天的行為）。

邊界對照（前後端共用 `apps/api/internal/services/testdata/resolution_tiers.json`，見 AC #3）：

| 輸入 | 今天前端 | 今天 Go | 改後 |
|---|---|---|---|
| `3840x2160`／`4096x2160`（DCI） | 4K | 4K | 4K |
| `3840x1608`／`3840x1600`／`4096x1716` | 4K | 4K | 4K |
| `3828x1588` | 1440p | 1440p | **4K** |
| `3456x1440`（4K 長邊門檻） | 1440p | 1440p | **4K** |
| `3455x1440`、`3440x1440`（帶魚螢幕） | 1440p | 1440p | 1440p |
| `2880x2160`、`2880x2152` | 4K、1440p | 4K、1440p | 4K、**4K** |
| `2560x1440`、`2560x1080`、`2560x1068` | 1440p、1440p、1080p | 1440p、1440p、**720p** | 1440p、1440p、**1440p** |
| `1920x1080`、`1920x1036`、`1920x800`、`1920x780` | 1080p | 1080p、**720p**、**720p**、**720p** | 1080p |
| `2048x858`（DCI 2K 寬銀幕）、`1440x1080`（4:3） | 1080p | 720p、1080p | 1080p |
| `1280x720`、`1280x536`、`960x720` | 720p | 720p、**480p**、720p | 720p |
| `720x576`、`720x480` | 480p | 480p | 480p |
| `640x360` | 360p | 360p | 360p |
| `1080x1920`（直式） | 1440p | 1440p | **1080p** |
| `2160x3840`、`720x1280`（直式） | 4K、1080p | 4K、1080p | 4K、**720p** |
| `0x1080`（寬未知）、`3840x0`（高未知） | 1080p、`0p` | 1080p、`0p` | 1080p、**4K**（有一邊未知就不轉橫） |
| `0x0` | 空 | 空 | 空 |

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要裁掉黑邊的 4K 片（例 Ballerina 的 3828×1588）在列表和詳細頁都寫「4K」，
而且不管片子的資訊是 ffprobe 量的還是 NFO 寫的，同一個寬高都得到同一個等級，
這樣我不會以為自己抓錯了片源、也不會同一個尺寸在兩處看到兩種說法。

## Acceptance Criteria

1. **前端 `resolutionTier` 照「規則」表判定**（`utils/resolutionLabel.ts`）：長短邊、90% 門檻、`${短邊}p` 備援；標籤字串（`4K`／`1080p`／`SD`）與 `2160p`→`4K` 的行為不變。`formatResolutionBadge` 跟著變（`3828x1588` → `{ tier: '4K', dims: '3828×1588', full: '4K · 3828×1588' }`），不改它本身。
2. **Go `resolveResolution` 照同一張表判定**（`nfo_reader_service.go`），輸出同樣的字（`4K`、`1440p`、`1080p`、`720p`、`480p`、`${短邊}p`、兩邊都 ≤0 時空字串）。
3. **同一張對照表測兩邊**：新增 `apps/api/internal/services/testdata/resolution_tiers.json`（`[{ "width": 3828, "height": 1588, "tier": "4K", "why": "…" }, …]`），涵蓋上面「邊界對照」每一列＋正式機每一種寬高＋每級門檻的「剛好到／差 1」兩側。Go `TestResolveResolution_SharedTable` 與前端 `resolutionLabel.spec.ts` 都讀這個檔逐列斷言；任一邊改規則沒改表、或改表沒改另一邊，該邊的測試會紅。
4. **故意改壞會變紅**：把任一邊的 4K 長邊門檻改成 3840（回到舊行為）時，兩邊各自的共用表測試都失敗（`3828x1588` 那列）；dev 實際改壞跑一次、記在 Debug Log。
5. **沒有東西要遷移**：DB 不改；不重算舊資料（正式機 NFO 標籤 0 筆）。
6. **畫面**：本機把一部片的 `video_resolution` 寫成 `3828x1588`，媒體庫列表（1440、390）規格行寫 `4K`、詳細頁（1440）徽章 `4K · 3828×1588`、手機（390）`4K`。
7. 檔頭與註解更新：`resolutionLabel.ts` 的說明改成新規則、拿掉「One step past Go」那段；`resolveResolution` 的註解寫同一規則並指回共用表。
8. `nx test web`（或相關 vitest）、`lint:all`（看「✖ N problems (X errors」那行）、typecheck、format:check、`go test ./...`、`staticcheck ./...` 綠。

## Tasks / Subtasks

- [x] **T1 共用對照表（AC #3）**：寫 `testdata/resolution_tiers.json`。
- [x] **T2 Go（AC #2、#4、#7）**：改 `resolveResolution`；`nfo_reader_service_test.go` 新增讀共用表的測試（舊的 6 列 `TestResolveResolution` 併入共用表）；故意改壞驗證。
- [x] **T3 前端（AC #1、#4、#7）**：改 `tierFromSize`；`resolutionLabel.spec.ts` 讀共用表；`3828x1588` 的徽章列改成 4K；故意改壞驗證。
- [x] **T4 Step 9 畫面比對（AC #6）**：本機截圖。

拆單判斷（Story Splitting Rule：跨前後端、單邊 >3 task 才拆）：前端 1 個 task、後端 1 個 task＋共用表 → **不拆**。

## Dev Notes

### 不要做的事

- 不要讓 NFO 改存原始寬高、也不要把判定搬到後端一處（資料模型決定，另議；目前兩邊用共用表鎖住一致）。
- 不要改字幕評分 `normalizeResolution`（另立案，見 Discovery Triage）。
- 不要新增媒體庫解析度篩選（沒有這個功能，要做是新功能，先過 party mode）。
- 不要動 visual 基準：畫廊夾具只用 `3840x2160`／`1920x1080`，E2E `library-list.spec.ts` 用 `3840x2160`／`1920x800`，改前改後等級都一樣。
- 不要動 DB／遷移。

### 已知陷阱

- 前端 spec 用 `readFileSync(join(__dirname, …))` 讀 Go 的 testdata（先例：`styles-contrast.spec.ts:15`、`tmdb-logo-asset.spec.ts:36`）。路徑：`apps/web/src/utils` → `../../../api/internal/services/testdata/resolution_tiers.json`。
- 門檻寫成整數（3456、1944…），不要寫 `× 0.9` 讓兩種語言各自做浮點數（854×0.9＝768.6，兩邊取整方式不同就會分歧）。
- 同 package 的 `setup_service.go` 在 working tree 有別的 session 未提交的修改——只 stage 本單的檔案。

### 測試標準

- vitest／Go testify，斷言具體字串（Rule 16）。
- 改 testid 前 grep `tests/e2e`：本單不動 testid。

### Project Structure Notes

- 改：`apps/web/src/utils/resolutionLabel.ts`（＋spec）、`apps/api/internal/services/nfo_reader_service.go`（＋test）。新增：`apps/api/internal/services/testdata/resolution_tiers.json`。

### Time-dependent visual coverage

- N/A — 不讀時間。

### References

- [Source: apps/web/src/utils/resolutionLabel.ts:1-48]、[apps/web/src/utils/libraryListRow.ts:18]、[apps/web/src/components/media/DetailTechInfoV2.tsx:98]
- [Source: apps/api/internal/services/nfo_reader_service.go:283-330]、[ffprobe_service.go:243-244]、[enrichment_service.go:797-798, 946-947]
- [Source: apps/api/internal/handlers/library_handler.go:78-218]（沒有解析度篩選）
- [Source: apps/api/internal/subtitle/scorer.go:153-187]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-detail-resolution-raw.md、disc-2026-10-library-list-triage-b1.md AC #4]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（SM＋dev agent，2026-10-19）

### Debug Log References

- 正式機唯讀（2026-10-19）：`select video_resolution, count(*) from movies group by 1`；`series` 同；非 `3840x2160`／`1920x1080` 的逐片列出片名（表見上）。
- 故意改壞（AC #4）：前端把 4K `minLong` 3456→3840 → `resolutionLabel.spec.ts` 4 個失敗（共用表 `3828x1588`、`3456x1440` 兩列＋舊表與徽章表的 `3828x1588`）；Go 同樣改 → `TestResolveResolution_SharedTable/3828x1588`、`/3456x1440` 失敗。改回後全綠。
- 本機畫面（AC #6）：`serve-test-env.sh` 種子 DB 把 沙丘:第二部 寫成 `3828x1588`、星際效應 `3840x1608`、全面啟動 `1920x800`；截圖在 scratchpad `res-tier-ux/`。列表 1440：`4K · H.265 · DTS`／`4K · H.265`／`1080p · H.264`；詳細頁 1440 徽章 `4K · 3828×1588`、390 只寫 `4K`。手機列表本來就不顯示規格行（A4p-M），無變化。

### Completion Notes List

- T1：共用表 55 列（`apps/api/internal/services/testdata/resolution_tiers.json`）：正式機每一種寬高、每級門檻「剛好到／差 1」兩側、直式、DCI、寬或高未知。
- T2：Go `resolveResolution` 改成表驅動（`resolutionTiers`），舊的 6 列測試併入共用表；另加一個 NFO 解析測試（3828×1588 的 NFO 存成 `4K`）。
- T3：前端 `tierFromSize` 同一張表（`RESOLUTION_TIERS`），整數門檻；spec 讀共用表；`3828x1588` 的舊表與徽章表改成 4K；拿掉「One step past Go」說明。
- 對抗式自我審查（就地修）：①`640x0` 會寫出 `0p` → 兩邊改成空字串（不出徽章），表加一列；②前端 spec 讀 Go 的 testdata，但 `nx test web` 的快取輸入只看 `apps/web` → `apps/web/project.json` 的 test inputs 加上那個 JSON；③Go 註解講清楚寬高為 0 的分支只為了跟前端答同一張表（NFO 呼叫端本來就要求兩邊 >0）；④共用表筆數下限 40→55。未修（記在 PR）：已經從 NFO 存成標籤的舊列不會重算（正式機 0 筆）、字幕評分（另立案）、正方形畫面（`1440x1440`→1440p）沒有特別規則。
- 測試：web vitest 4,771 pass；`go test ./...` 42 個套件 ok；staticcheck、go vet、eslint（0 errors）、typecheck、`prettier --check .` 綠。
- ⚠️ 過程中另一個 session（setup-wizard-byok）把本分支的未提交修改 stash 起來切走 working tree，約 25 分鐘後還原；期間的檢查改在 scratchpad 的 `git archive` 匯出副本上跑（不是 git worktree），還原後逐檔 `cmp` 確認與副本相同才 commit。

### Discovery Triage

- **③ backlog** — `disc-2026-10-subtitle-score-frame-size`（P3）：字幕搜尋送 `3840x2160` 這種原始寬高給 Go 評分，`normalizeResolution` 認不得 → 有標解析度的字幕一律 0.2 分、低於沒標的 0.5，排序被反過來。修法候選：評分端先把寬高換成等級（要用同一套規則，Go 的 `resolveResolution` 在 `services` 套件、評分在 `subtitle` 套件，得先搬到共用處）。

### File List

- _bmad-output/implementation-artifacts/disc-2026-10-resolution-tier-scope-encodes.md（新增）
- _bmad-output/implementation-artifacts/sprint-status.yaml
- apps/api/internal/services/nfo_reader_service.go
- apps/api/internal/services/nfo_reader_service_test.go
- apps/api/internal/services/testdata/resolution_tiers.json（新增）
- apps/web/src/utils/resolutionLabel.ts
- apps/web/src/utils/resolutionLabel.spec.ts
- apps/web/project.json
