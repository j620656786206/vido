# Bugfix: 「管理字幕」列出這部片真正有的字幕，不再說假話

Status: review

**Source:** Alexyu 2026-10-05 在 NAS 上打開《末日光明》S01E02 的管理字幕，視窗寫「尚無字幕／此影片目前沒有任何字幕軌」——但這集片內有英文字幕軌，旁邊還有一份 `.zh-TW.srt`。⚖️ Alexyu 選 B：「打開視窗時，真的去看這一集」。

## Story

身為 Vido 的使用者，
我希望打開某一集（或某部電影）的「管理字幕」時，看到它**真正**有哪些字幕（片內的、旁邊另外放的、Vido 做的），
這樣我才知道要不要花錢生成，也不會被一句假話誤導。

## 查到的事（2026-10-05，Bob 逐一開檔確認）

- **單集完全沒有字幕資訊：** `apps/web/src/components/media/SeasonAccordion.tsx:135-140` 註解寫明「subtitleTracks is deliberately NOT passed」——展開一季時逐集讀檔會讓硬碟忙不過來（red line 3）。對話框的 `buildTrackRows`（`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:104-140`）因此回傳空陣列，畫出「尚無字幕／此影片目前沒有任何字幕軌」（`:365-376`）。
- **電影有資料但是舊的：** 掃描時把片內軌＋旁邊的字幕檔合在一起存進 `subtitle_tracks`（`apps/api/internal/services/enrichment_service.go:756-757`，`DetectExternalSubtitles` 在 `ffprobe_service.go:241-290`）。之後新增、刪除的字幕檔不會更新；字幕檔語言照檔名標籤（`ffprobe_service.go:272-279`），違反「語言看內容、不看檔名」的規則（`project-context.md` 字幕規則第一條）。
- **來源標錯：** 對話框把每一條都標「本地檔案」（`ManageSubtitleDialogV2.tsx:132`），片內軌也一樣。
- **片內中文軌分不出繁簡：** ffprobe 只給 `chi`／`zho`，前端 `subtitleLangLabel` 把它顯示成原字 `chi`（`apps/web/src/utils/libraryStatus.ts:97-104`）；`zh` 則被歸成繁中（`:82`），不能拿來代表「不知道是繁是簡」。ffprobe 的 `title` 標籤目前沒被讀出（`ffprobe_service.go:207-216` 只取 `language`）。
- **現成的偵測：** `subtitle.Detect`（`apps/api/internal/subtitle/detector.go:61`，最多讀 100KB，`:26`）依內容判斷繁／簡／混合／非中文。檔名過濾的前例：`mine.IsOfficialZhSidecar` 排除 `.bak`、`.tmp.`（`apps/api/internal/subtitle/mine/sources.go:51-72`）。
- **設計稿：** F1-D-v2（`r1EY9`）／F1-M-v2（`JkdfH`）已畫「檔案列」：語言藥丸＋來源（已生成／本地檔案／線上下載）＋檔名。**沒畫**片內軌、載入中、讀取失敗。
- **NAS 實例：** S01E02 片內有英文、英文 SDH 等約 30 種語言的軌，旁邊有 `.zh-TW.srt`；S02 的 4K 檔片內有 3 條 `chi` 軌。

## 設計

**只在打開某一部片的對話框時才去讀**（一次一個檔，不會在展開整季時逐集讀）。讀分兩段，**便宜的那段可以單獨拿來用**（之後「季清單每集有／缺字幕」的單子會用到）：

1. **便宜：列出旁邊的字幕檔**——讀一次資料夾，挑出同檔名開頭的 `.srt／.ass／.ssa／.vtt`（排除 `.bak`、`.tmp.`）。語言：中文**一律看內容**（`Detect`），非中文才用檔名標籤；標出哪一份是 Vido 自己放的（路徑等於資料庫的 `subtitle_path`）。
2. **較貴：讀片內字幕軌**——ffprobe 一個檔。中文軌用 `title` 標籤猜繁／簡／粵（含「繁、Traditional、Hant、TW、HK」→ 繁中；「简、簡、Simplified、Hans、CN」→ 簡中；「粵、Cantonese、yue」→ 粵語）；猜不到就說「中文」，不假裝知道。

**顯示：**
- Vido 做的那份照舊是「字幕引擎／已生成」那一列（不重複列）。
- 旁邊的字幕檔：來源「本地檔案」，下面一行是檔名。
- 片內字幕軌：來源「片內字幕」，下面一行是軌道名稱（沒有名稱就寫「第 N 軌」）。**只列中文和英文**；其他語言收成一行「另有 N 種其他語言的片內字幕」（S01E02 有約 30 種，全列太吵）。
- 讀取中：「正在讀取這部片的字幕…」。
- 讀不到（整個失敗）：「讀不到這部片的字幕資訊」＋「重試」，**不再說「沒有字幕」**。
- 片內那段讀不到、但字幕檔讀得到：照常列字幕檔，加一行「片內字幕讀不到」。
- 兩段都讀到、真的什麼都沒有：才顯示「尚無字幕／這部片目前沒有任何字幕」。
- 影集層級（不是單集）的對話框不變。

## Acceptance Criteria

1. **後端便宜段：** `subtitle.ListSidecars(mediaPath, ownSubtitlePath)` 只讀一次資料夾＋每個字幕檔前 100KB；回傳每個檔的檔名、語言（中文看內容、其他看檔名、都沒有就 `und`）、格式、是否為 Vido 自己放的。不呼叫 ffprobe。
2. **後端貴段：** ffprobe 讀出 `title`；片內軌語言正規化成 `zh-Hant`／`zh-Hans`／`yue`／`zh-unknown`（中文但分不出）／`en`／原標籤。
3. **API：** `GET /api/v1/movies/:id/subtitles/inventory`、`GET /api/v1/episodes/:id/subtitles/inventory`，回 `{sidecars:{status,files[]}, embedded:{status,tracks[]}}`；`status` 是 `ok`／`unavailable`（沒裝 ffprobe）／`failed`。找不到片 404；片沒有檔案路徑或檔案不見 400（同 transcribe 的文字）。一段失敗不影響另一段（照樣 200）。
4. **前端：** 電影與單集的管理字幕改用這支 API（打開時才查）；依「顯示」那段的規則畫列與狀態；片內軌來源「片內字幕」；中文分不出繁簡顯示「中文」。
5. **季清單不變：** 展開一季不會多打任何 API。
6. **設計稿：** 新增獨立規格畫面，畫片內字幕列、「另有 N 種」、讀取中、讀不到、片內讀不到五種狀態（含手機寬度說明）；匯出截圖並登記到 `SCREENS`。
7. **測試：** Go：`ListSidecars`（內容蓋過檔名、排除 .bak／其他影片／不同檔名開頭、標出 Vido 自己的）、片內語言正規化、handler（電影／單集 200、404、ffprobe 沒裝時片內 `unavailable` 但字幕檔照列）。前端：載入、失敗＋重試、真的沒有字幕、Vido 那份不重複、只列中英＋其他數量、中文分不出顯示「中文」。
8. 全部檢查綠：`go build／vet／test ./...`、`staticcheck-2026.1`、`pnpm nx test web`、`pnpm run lint:all`、`check-design-tokens.py`。

## Tasks / Subtasks

- [x] T1 後端：`ListSidecars`＋ffprobe `title`＋片內語言正規化＋測試（AC #1、#2）
- [x] T2 後端：inventory handler＋路由＋main 接線＋測試（AC #3）
- [x] T3 前端：型別、service、hook（AC #4）
- [x] T4 前端：對話框列與狀態＋spec；SeasonAccordion 註解更新（AC #4、#5）
- [x] T5 設計稿規格畫面＋截圖（AC #6）
- [x] T6 檢查（AC #7、#8）

後端 2 項、前端 2 項 → 不拆單。

## Dev Notes

### 不要做的事

- 不要在季清單展開時打這支 API（red line 3）。
- 不要改掃描時寫 `subtitle_tracks` 的邏輯（媒體庫徽章、候選分析都讀它，範圍外）。
- 不要信檔名的中文標籤（`.zh-TW.srt` 內容可能是簡體，前例見 `subtitle-v4-replan-and-feasibility-audit-2026-06.md:179`）。

### 已知陷阱

- `subtitleLangLabel('zh')` 會說「繁中」——所以「分不出繁簡」要用另一個值（`zh-unknown`），由對話框自己對應成「中文」。
- ffprobe 預設 10 秒逾時（`ffprobe_service.go:69-71`）；4K 檔通常一兩秒，逾時就回 `failed`，不要讓整個請求失敗。
- 字幕檔路徑比對要 `filepath.Clean` 兩邊。

### Time-dependent visual coverage

N/A — 不讀時鐘。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

### Completion Notes List

- 2026-10-05 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created
- **前提更正（建單時查證）：** 交辦說「電影也不列旁邊的字幕檔」——不對。電影在掃描時有把字幕檔合進 `subtitle_tracks`（`enrichment_service.go:756-757`），只是之後不更新、而且語言照檔名。單集才是完全沒有。本單兩者都改成打開時現讀。
- 🔗 AC Drift: FOUND — dsr-6b AC #3（對話框 track 列「本地檔案」來源、`subtitleTracks` JSON）→ 電影／單集改讀 inventory，片內軌改標「片內字幕」，空狀態文字改成「這部片目前沒有任何字幕」（只在兩半都讀到且真的沒有時）。影集層級的對話框維持舊行為。spec 的舊 tracks 測試改用 `inventoryFromTracks` 轉成字幕檔餵入，測的事不變。
- 📎 Contract Stamps: NONE (新 API，未加 stamp；既有 API 不變)
- 🎭 A11y Pre-Flight: PASS (1 component checked, 0 jsx-a11y warnings introduced；讀取中 `aria-live=polite`、讀不到 `role=alert`、重試鈕 44px 觸控高；lint 抓到重試字用 `--accent-primary` 低於 AA → 改 `--accent-text`，設計稿同步)
- 🎨 UX Verification: PASS — 三張對話框視覺夾具（桌機、手機、untranslated）重產 darwin 基準，列的排法與 F1-D-v2 一致（藥丸＋來源＋Mono 檔名同一列）；新狀態依新規格畫面 `F1-SPEC-INVENTORY`（`VJTsJ`）。-linux 基準已 `git rm`，由 CI bootstrap。
- T1：`subtitle/inventory.go`：`ListSidecars`（一次 `ReadDir`＋每檔前 100KB `Detect`；中文看內容、中文標籤但沒有中文字 → `und`；排除 `.bak`、`.tmp.`；`IsVidoOutput` 用 `filepath.Clean` 比對 `subtitle_path`）、`embeddedLanguage`（`title` 標籤猜繁／簡／粵；猜不到 `zh-unknown`）、`BuildInventory`（兩半各自 status）。`services.SubtitleTrack` 多 `Title`（`omitempty`，掃描存的 JSON 只會多欄位）。
- T2：`handlers/subtitle_inventory_handler.go`＋`main.go` 接 `repos.Movies`、`repos.Episodes`、`ffprobeService`。本機啟動 API 確認路由沒有衝突（打不存在的 id 回 404）。
- T3：`subtitleService.getInventory`、`hooks/useSubtitleInventory.ts`（只在對話框打開、且不是影集層級時啟用；`retry:false`）。
- T4：對話框：只列中英片內軌，其他收成「另有 N 種其他語言的片內字幕」；Vido 自己那份把檔名借給「字幕引擎」那列、不重複；讀取中／讀不到＋重試／半邊讀不到的說明行；生成結束或線上下載後重新讀。`SeasonAccordion` 註解更新。gallery 三個夾具 seed inventory。
- T5：`.pen` 新增 `F1-SPEC-INVENTORY`（`VJTsJ`，放在 `JzmvC`「F · 規格」右側），`SCREENS` 登記 `f1-spec-inventory`；匯出後只提交新圖＋`pen-tokens.json`，其餘 13 張重繪噪音還原。
- 驗證：`go vet／test ./...` 全綠、`staticcheck-2026.1` 零輸出、`pnpm nx test web` 297 檔／4574 測試、`pnpm run lint:all` 0 errors（警告數不變）、`check-design-tokens.py` 一致、本機 visual 只有三張刻意刪掉的基準「writing actual」、`test:cleanup` 無殘留。

### Discovery Triage

- **③** 季清單每集標「有／缺字幕」（Alexyu 2026-10-05 要求）→ `disc-2026-10-episode-list-subtitle-badge`。本單的 `ListSidecars` 刻意設計成「一次讀資料夾、不跑 ffprobe」，那張單可直接拿來做整季摘要。
- **③** 掃描時存的電影字幕檔語言照檔名標籤（`ffprobe_service.go:272-279`）→ 影響媒體庫徽章；本單不改，記在 `disc-2026-10-episode-list-subtitle-badge` 一併評估（同一個「怎麼判斷有沒有中文字幕」的問題）。

### File List

- apps/api/internal/subtitle/inventory.go（新）
- apps/api/internal/subtitle/inventory_test.go（新）
- apps/api/internal/handlers/subtitle_inventory_handler.go（新）
- apps/api/internal/handlers/subtitle_inventory_handler_test.go（新）
- apps/api/internal/services/ffprobe_service.go
- apps/api/internal/services/ffprobe_service_test.go
- apps/api/cmd/api/main.go
- apps/web/src/services/subtitleService.ts
- apps/web/src/services/subtitleService.spec.ts
- apps/web/src/hooks/useSubtitleInventory.ts（新）
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx
- apps/web/src/components/media/SeasonAccordion.tsx（註解）
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/subtitle-manage-subtitle-dialog-v2{,-mobile,-untranslated}/default-visual-darwin.png（重產）、default-visual-linux.png（刪，待 CI bootstrap）
- ux-design.pen（`VJTsJ`）
- scripts/export-pen-screenshots.py
- _bmad-output/screenshots/flow-f-subtitle-v2/f1-spec-inventory.png（新）
- _bmad-output/pen-tokens.json
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-10-05 建立（Bob create-story，Alexyu 選 B）＋實作 T1–T6（Opus 5.5 dev-story），status → review。
