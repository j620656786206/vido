# Story bugfix：自己上傳的海報看得到了，也不會再被「清除快取」當成快取刪掉

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who uploads a custom poster for a film TMDb got wrong,
I want the poster I uploaded to actually show up in the library and on the detail page, and to survive「清除快取」,
so that fixing a poster by hand is a one-time job instead of something that silently never worked and can be wiped by a cache button.

## Context

升級自 `disc-2026-09-cache-clear-deletes-uploaded-posters`（P1，dsr-3e Task 1 查證快取警告句時發現）。dev 在 2026-09-24 已用程式碼逐條查證（下面每一條都有檔案:行號），SM 建單時再核對一次。

### 🔴 查到的事（main `79f57a56`；行號皆為現況）

1. **`data/posters` 裡只有使用者上傳的海報，沒有任何快取。**
   - 唯一會寫進這個資料夾的是 `images.ImageProcessor.ProcessPoster`（`apps/api/internal/images/processor.go:68`，寫 `<mediaID>.jpg` 與 `<mediaID>-thumb.jpg`，`:96-114`），呼叫端只有 `MetadataEditService.UploadPoster`（`internal/services/metadata_edit_service.go:284`）。
   - TMDb 海報是前端直接從 `image.tmdb.org` 載入（`apps/web/src/lib/image.ts:1,19-23`），後端**不快取任何圖片**。
   - 但同一個資料夾被當成「圖片快取」：`cmd/api/main.go:204-206` 把 `posterDir` 傳給 `NewCacheStatsService` 與 `NewCacheCleanupService`；`cache_stats_service.go:51-56` 把它列成 `type: "image"`／「圖片快取」。
2. **所以兩顆清除鈕都會刪掉使用者上傳的海報：**
   - 設定 → 快取管理 →「圖片快取」的「清除」→ `ClearCacheByType("image")` → `clearAllImages`（`cache_cleanup_service.go:101-107`）＝**全部**上傳海報。
   - 「清除 30 天前的快取」→ `ClearCacheByAge` → `clearOldImages`（`:161-186`）＝mtime 超過 30 天的上傳海報。
   - 不帶參數的 `DELETE /settings/cache` 也會跑 `ValidCacheTypes`（`:26`，含 `image`）。
   - 刪掉後 DB 的 `poster_path` 仍是 `/posters/<id>.jpg`，指向不存在的檔案。
3. **更根本：上傳的海報本來就顯示不出來（Story 3.8 AC3 從未真的上線）。**
   - 上傳成功後 DB 存 `poster_path = "/posters/<id>.jpg"`（`processor.go:226-228` `GetPosterURL`，由 `metadata_edit_service.go:297-301` 寫入）。
   - 前端 `getImageUrl`（`lib/image.ts:19-23`）只認得「絕對網址」或「TMDb 相對路徑」，把 `/posters/<id>.jpg` 拼成 `https://image.tmdb.org/t/p/w342/posters/<id>.jpg` → 破圖；`getImageSrcSet` 同樣拼出三個錯的網址。
   - 後端**沒有任何路由**提供 `/posters/...`（`cmd/api/static.go` 只服務 `/assets` 與 favicon 類；`grep` 全專案無 `/posters` 路由）。dev 模式的 vite 只代理 `/api`（`apps/web/vite.config.mts:21-26`）。
   - 上傳入口是 `LocalDetailV2` 裡的中繼資料編輯器 → `components/metadata-editor/PosterUploader.tsx`；它的預覽直接用原字串 `<img src={preview}>`（`:36, :197`），同樣是破圖。
4. **Story 3.8 的原意**（`3-8-metadata-editor.md:31, 155-156`）是回傳 `/posters/<id>.webp` 並能顯示——「怎麼服務這些檔案」當時沒被做。
5. **正式環境影響未查**：dev 查證時 NAS SSH 逾時（不在家中網路）。dev 開工時先唯讀查：`sqlite3 -readonly /mnt/user/appdata/vido/vido.db "select count(*) from movies where poster_path like '/posters/%'; select count(*) from series where poster_path like '/posters/%';"` 與 `ls /mnt/user/appdata/vido/posters | wc -l`，寫進 Completion Notes。

### ⚖️ 建單裁定（2026-09-24，SM；Alexyu 可在 review 推翻）

1. **完整修好，不只止血。** 只把海報從快取裡拿掉，使用者上傳的海報還是看不到——功能仍是壞的。這張同時做「不再被清掉」與「看得到」。
2. **不搬資料夾、不做資料遷移。** 上傳海報維持存在 `data/posters`、DB 的 `poster_path` 維持 `/posters/<id>.jpg`。改的是：(a) 快取功能不再碰這個資料夾；(b) 後端提供這些檔案；(c) 前端認得這種路徑。既有的上傳海報**不需要任何遷移**就會開始顯示。
3. **「圖片快取」這個分類拿掉**（後端 `image` cache type 刪除）。它從來不是快取；留著一張永遠 0 的卡片只會誤導。快取管理剩四類：AI 解析、TMDb 中繼資料、豆瓣、維基百科。
4. **提供檔案的路由放在 `/api/v1/posters/:file`**（不是根目錄 `/posters`）：dev 的 vite 已代理 `/api`，正式環境也在同一個 API group 下（auth gate 開啟時一併受保護）。前端把 DB 裡的 `/posters/<file>` 對應到 `{API_BASE_URL}/posters/<file>`。**DB 值不改**。

## Acceptance Criteria

1. **快取功能不再碰上傳的海報。**
   - `ValidCacheTypes` 移除 `image`；`CacheStatsService` 不再回報 `image`、不再需要 `imageDir`；`CacheCleanupService` 移除 `clearAllImages`／`clearOldImages` 與 `imageDir`；`ClearCacheByType("image")` 回 `ErrInvalidCacheType`（handler 既有的 400 路徑）。
   - `CacheStatsServiceInterface.GetImageCacheSize` 若無其他呼叫者（建單時查：只有自己的定義）→ 一併移除。
   - `cmd/api/main.go:205-206` 不再傳 `posterDir` 給這兩個服務。
   - 測試（Go）：在一個含 `x.jpg`（mtime 設成 60 天前）的暫存 posters 目錄旁跑 `ClearCacheByAge(ctx, 30)` 與不帶參數的全部清除 → 檔案**仍在**；`GetCacheStats` 的類型清單不含 `image`。
2. **後端提供上傳的海報。**
   - 新路由 `GET /api/v1/posters/:file`，從 `posterDir` 回檔案。
   - **只接受** `^[A-Za-z0-9_-]+(-thumb)?\.jpg$`（UUID 與 `-thumb`）；其他一律 404（**不可**用 `filepath.Join` 直接拼使用者輸入；`..`、`%2e%2e`、子目錄都要擋）。檔案不存在 → 404。
   - 回應帶 `Content-Type: image/jpeg`；快取標頭用 `Cache-Control: no-cache`（同一個 media 重新上傳會覆寫同名檔案，不能 immutable；靠 `Last-Modified` 重新驗證）。
   - 測試（Go handler）：合法檔名 200＋內容；`../vido.db`、`a/b.jpg`、`x.png`、URL 編碼的 `..` → 404；不存在 → 404。
3. **前端認得上傳海報的路徑。**
   - `lib/image.ts`：`getImageUrl` 遇到以 `/posters/` 開頭的值 → 回 `${API_BASE_URL}${path}`（`API_BASE_URL` 與各 service 一致：`import.meta.env.VITE_API_BASE_URL || '/api/v1'`）；**不加** TMDb 尺寸。`getImageSrcSet` 對這種值回 `null`（只有一種尺寸）。其他行為（TMDb 相對路徑、絕對網址）不變。
   - 判斷用前綴 `/posters/` 是安全的：TMDb 路徑是單層檔名（`/abc123.jpg`），不會有子目錄。在 `image.ts` 註解寫明這個理由。
   - `PosterUploader` 的預覽改用 `getImageUrl`（不再直接 `<img src={原字串}>`）；上傳成功後的新預覽同樣。
   - 測試（vitest）：`/posters/u1.jpg` → `/api/v1/posters/u1.jpg`（並有一條 `VITE_API_BASE_URL` 自訂時的案例，或把 base 抽成可測常數）；`getImageSrcSet('/posters/u1.jpg')` → `null`；TMDb 路徑與絕對網址的既有測試不動。
4. **快取管理頁跟著少一類。**
   - 前端 `CacheManagement`／`CacheTypeCard` 本來就依後端清單渲染 → 碼大多不用改；把 spec／夾具裡的 `image` 範例改成其他類型，並更新 `CacheManagement.tsx` 那段提到 `data/posters` 的註解（dsr-3e 寫的）：清 30 天前**不再**碰海報。
   - 「清除 30 天前」的說明條文字（dsr-3e C18）維持不變——「不會影響影片與字幕檔案」仍為真，且現在連海報也不會動；⚖️ **不加字**（要不要補「也不會刪你上傳的海報」由 Sally 決定，列入 Completion Notes 的待確認）。
   - **設計稿**：C11-D／C11-M（`TrU8k`／`aYEWP`）與 C18-D（`dfwSb`）刪掉「圖片快取」那張卡、總計數字改成四類加總；規格註記 `spec-note-dsr-3e` 補一句「圖片快取不是快取（是使用者上傳的海報），已移除」。收尾同 `dsr-3a` AC #1（`problems` 不增、選單 Save、磁碟 grep、只 stage `c11-*`／`c18-d`＋`pen-tokens.json`；**另跑** `python3 scripts/check-design-tokens.py`——`lint:all` 不含它）。
5. **端到端證明「上傳 → 看得到 → 清快取後還在」。**
   - e2e（新增 `tests/e2e/custom-poster.spec.ts` 或追加到既有 metadata editor 的 e2e——dev 先 `grep` 找現有的上傳測試）：對一部片上傳一張小 JPEG（夾具圖放 `tests/support/fixtures/`）→ 詳情頁海報 `<img>` 的 `src` 以 `/api/v1/posters/` 開頭且 `naturalWidth > 0`（真的載入成功）→ 到設定 → 快取管理，按「清除 30 天前的快取」兩次 → 回詳情頁，海報仍然載入成功。
   - 若 e2e 環境的後端無法上傳（例如沒有可編輯的片）→ 以 Go 整合測試＋前端單元測試取代，Completion Notes 說明。
6. **既有行為不准回歸。**
   - TMDb 海報、豆瓣／維基等絕對網址海報照常顯示（`lib/image.ts` 既有測試全綠）。
   - 快取的四類統計與清除、兩段式確認照舊；`CacheManagement.spec`／`CacheTypeCard.spec`／`cache.api.spec.ts`（若斷言 `image`）→ 屬本張刻意變更，Completion Notes 逐條列。
   - 上傳 API（`POST /api/v1/media/:id/poster`）的請求與回應格式不變（`posterUrl` 仍回 `/posters/<id>.jpg`）。
7. **測試與 CI**：紅／守（Rule 16）、每一項修法做 mutation check；`pnpm nx test api`、`pnpm nx test web`、`pnpm run lint:all`、`python3 scripts/check-design-tokens.py` 全綠；視覺夾具 `settings-cache-management`（含 `/confirm`）因少一張卡基準會變——過期 `-linux` 以 `git rm` 交給 CI bootstrap（`project_visual_baseline_intentional_change`）。

## Tasks / Subtasks

- [x] **Task 1 — 正式環境唯讀查證（AC: Context 🔴 #5）**：NAS 上有幾部片用上傳海報、資料夾有幾個檔。連不上就寫明並繼續。
- [x] **Task 2 — 後端：快取功能移除 `image`（AC: #1）**
  - [x] 2.1 `cache_cleanup_service.go`：`ValidCacheTypes`、`ClearCacheByType`、`ClearCacheByAge`、建構子；刪 `clearAllImages`／`clearOldImages`
  - [x] 2.2 `cache_stats_service.go`：拿掉 `image` 類型、`imageDir`、`getImageStats`、（無人用時）`GetImageCacheSize`
  - [x] 2.3 `main.go` 建構呼叫；既有 Go 測試裡的 `image` 案例改寫；新增「清除後上傳海報仍在」測試
- [x] **Task 3 — 後端：`GET /api/v1/posters/:file`（AC: #2）**：handler＋檔名白名單＋標頭＋測試（含路徑穿越）
- [x] **Task 4 — 前端：`getImageUrl`／`getImageSrcSet`／`PosterUploader`（AC: #3）**＋測試
- [x] **Task 5 — 設計稿與快取頁（AC: #4）**：C11／C18 刪卡、規格註記、token 檢查；spec／夾具換掉 `image` 範例；更新註解
- [x] **Task 6 — e2e、mutation、收尾（AC: #5, #6, #7）**

## Dev Notes

### 這張的重點

- **這不是「快取清得太多」而已**：那個資料夾裡根本沒有快取。拿掉「圖片快取」＋讓上傳海報真的能顯示，才是修好。
- **不遷移資料**：DB 值、檔案位置都不動；只改「誰會刪它」與「怎麼拿到它」。
- **路由是新的對外檔案服務**：白名單檔名、不信任使用者輸入、404 一律不洩漏原因。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。`GET /settings/cache` 的回應少一個 `image` 類型屬既有 API 的內容變更（前端本就依清單渲染）；`POST /media/:id/poster` 格式不變。

### 不要做的事

- 不要搬 `data/posters`、不要改 DB 裡的 `poster_path`。
- 不要用 `router.Static("/posters", posterDir)`／`StaticFS`（會把整個目錄連同任何檔名都開出去）；用白名單 handler。
- 不要讓 `getImageUrl` 對 `/posters/` 以外的相對路徑改行為。
- 不要順手做「TMDb 圖片快取」這種新功能。

### 已知陷阱

- **快取頁的「清除 30 天前」說明句是 dsr-3e 查證過的承諾**：改完後它更真了，不需要改字；若要補一句關於海報，是 UX 決定（Sally）。
- **`VITE_API_BASE_URL`**：各 service 各自讀一次同一個 env（例如 `services/serviceStatusService.ts:7`）；`lib/image.ts` 照同一寫法，或抽共用常數——不要寫死 `/api/v1`。
- **Pencil**：變數名先 `GetVariables()` 確認（`project_pen_schema_gotchas` #12：打錯會靜默變裸數字，CI 的 token 檢查會擋）。
- **gh**：用 `GH_TOKEN=$(gh auth token --user j620656786206)`，另一個對話可能把 active account 切走。

### Source tree

```
apps/api/internal/services/cache_cleanup_service.go、cache_stats_service.go（+tests）   ← Task 2
apps/api/cmd/api/main.go                                                            ← Task 2/3
apps/api/internal/handlers/<new or metadata_handler>.go（+test）                     ← Task 3
apps/web/src/lib/image.ts（+spec）、components/metadata-editor/PosterUploader.tsx     ← Task 4
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c11-d,c11-m,c18-d}.png、
apps/web/src/components/settings/CacheManagement.tsx（註解）與 Cache*.spec.tsx、-gallery.fixtures.tsx ← Task 5
tests/e2e/…                                                                         ← Task 6
```

### Cross-Stack Split Check

後端 task 2（Task 2、3）、前端 task 2（Task 4、5 的碼）→ 兩邊都 ≤3，不拆。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（`clearOldImages` 的 mtime 判斷被刪除，不是新增）。

### References

- [Source: `apps/api/internal/images/processor.go:68-114, 226-243`；`internal/services/metadata_edit_service.go:277-318`；`internal/services/cache_cleanup_service.go:26, 48-90, 94-150, 161-186`；`internal/services/cache_stats_service.go:29, 35-56, 115-125`；`cmd/api/main.go:204-206, 339`；`cmd/api/static.go`]
- [Source: `apps/web/src/lib/image.ts:1-35`；`components/metadata-editor/PosterUploader.tsx:16-36, 195-197`；`apps/web/vite.config.mts:21-26`]
- [Source: `_bmad-output/implementation-artifacts/3-8-metadata-editor.md:31, 155-156`（原意）；`dsr-3e-settings-cache-and-logs.md`（Task 1 查證、C18 說明句）；`sprint-status.yaml` → `disc-2026-09-cache-clear-deletes-uploaded-posters`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (1M context)（Amelia / dev-story）

### Debug Log References

- NAS：LAN 位址 SSH 逾時，改走 `~/.ssh/config` 的 `unraid`（Cloudflare tunnel）成功；`sqlite3 -readonly` 查詢（SQL 由 stdin 餵，避免引號被 shell 吃掉）。
- 本機 8080 上一直跑著前幾張單子留下的舊 `api` 程序（跑的是舊程式）→ 先 kill 再用新程式起，否則 e2e 會測到舊行為。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（正式環境）**：NAS 上 `movies`／`series` 的 `poster_path like '/posters/%'` 都是 **0**，`/mnt/user/appdata/vido/posters` 是**空的**（3 月建立後從沒放過檔案）→ **正式環境沒有任何上傳海報被刪過**；這張是預防，不是救資料。
- **Task 2**：`ValidCacheTypes` 移除 `image`（加註解說明原因）；`CacheCleanupService`／`CacheStatsService` 不再持有 `imageDir`，建構子改成只收 `db`；刪 `clearAllImages`／`clearOldImages`／`getImageStats`／`GetImageCacheSize`（介面方法也刪——建單時確認除了自己沒有呼叫者）；`ClearCacheByType("image")` 走既有的 `ErrInvalidCacheType` → 400。`main.go` 不再把 `posterDir` 傳給快取服務。新增 Go 測試：60 天前的上傳海報在「清 30 天前」＋逐類全清之後仍在。
- **Task 3**：新 `handlers/poster_file_handler.go`：`GET /api/v1/posters/:file`，檔名白名單 `^[A-Za-z0-9_-]+(-thumb)?\.jpg$`、不存在或不合規一律 404、`Content-Type: image/jpeg`、`Cache-Control: no-cache`（重新上傳會覆寫同名檔，靠 `Last-Modified` 重驗）。測試含路徑穿越（`..%2F`、`%2e%2e%2f`、`../`）、其他副檔名、帶點的名稱、**資料夾內但不是海報名的檔案**。
- **Task 4**：`lib/image.ts` 新增 `isUploadedPoster`（`/posters/` 前綴）：`getImageUrl` → `${API_BASE_URL}${path}`（`VITE_API_BASE_URL || '/api/v1'`，與各 service 相同寫法）、`getImageSrcSet` → `null`；TMDb 相對路徑與絕對網址行為不變（既有測試全綠）。`PosterUploader` 初始預覽改用 `getImageUrl`（原本直接 `<img src="/posters/…">`，連 TMDb 路徑也是破的）。
- **Task 5**：稿 C11-D／C11-M／C18-D 刪「圖片快取」卡、總計改 599 MB、`spec-note-dsr-3e` 補一句；`check-design-tokens.py` 通過。前端快取頁本來就依後端清單渲染，只改 spec／夾具的 `image` 範例與 `CacheManagement.tsx` 的註解。C18 的說明句**沒改字**（更真了）；要不要補「也不會刪你上傳的海報」→ 待 Sally 決定。
- **Task 6**：新 e2e `tests/e2e/custom-poster.spec.ts`：建一部片 → 上傳 JPEG → API 回 200 `image/jpeg` → 詳情頁 `<img src$="/api/v1/posters/<id>.jpg">` 且 `naturalWidth > 0` → 清 30 天前＋全清＋`/settings/cache/image` 回 400 → 檔案仍 200、重新整理後仍顯示。`--repeat-each=3` 3／3。
- **既有測試改動（刻意，逐條）**：`cache_cleanup_service_test.go`「ClearCacheByType_Image」→ 改成「image 不是快取」（回 `ErrInvalidCacheType`）、「清 30 天前刪舊圖」→ 改成「上傳海報不會被清」、「EmptyImageDir」刪除、`ValidCacheTypes` 期望值少 `image`；`cache_stats_service_test.go` 類型數 5→4、不含 `image`、四條 `GetImageCacheSize`／圖片統計測試刪除；`cache_handler_test.go` 全清的合計 5→4、mock 移除 `GetImageCacheSize`；`e2e/cache.api.spec.ts` 三條（類型清單、標籤、`DELETE /settings/cache/image` 由 200 改為 400）；前端 `CacheManagement.spec`／`CacheTypeCard.spec` 的 `image` 範例換成其他類型。
- 🔗 **AC Drift: FOUND** — `6-2-cache-management` AC #1（「breakdown by type: Image cache (X.X GB), AI parsing cache, Metadata cache, Total」）→ 不再有 Image cache：那個資料夾不是快取（本張 Context 🔴 #1）。`3-8-metadata-editor` AC3（自訂海報上傳）→ 契約不變（回傳 `/posters/<id>.jpg`），本張補上「服務與顯示」這一半。
- 📎 **Contract Stamps: NONE**。
- 🎭 **A11y Pre-Flight: PASS**（`PosterUploader` 只改 src 來源；快取頁少一張卡；jsx-a11y 警告 0 新增）。
- **測試**：`nx test web` **288 files／4292 tests** 綠；`nx test api` 綠；`lint:all`、typecheck、`check-design-tokens.py` 綠；e2e `custom-poster`＋`cache.api`＋`metadata-editor.api` 34／34。
- **Mutation：unit／Go 5／5 紅、e2e 1／1 紅**（前端拿掉 `/posters/` 對應 → 2 條單元＋e2e 紅；拿掉 srcset 排除 → 紅；`ValidCacheTypes` 放回 `image` → Go 紅；白名單放寬成 `.*` → 第一輪**沒紅**（路徑穿越本來就被路由與 Join 擋住）→ 補「資料夾內的非海報檔」測試後紅；快取標頭改 immutable → 紅）。
- **視覺基準**：`settings-cache-management`（default、`/confirm`）少一張卡，darwin 基準更新、過期 `-linux` 以 `git rm` 交給 CI bootstrap。

### Discovery Triage

- N/A — no out-of-scope work discovered（C18 說明句要不要補一句屬本張待 UX 確認，不是新工作）。

### File List

- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-c-search-settings/{c11-d,c11-m,c18-d}.png`
- `apps/api/internal/services/cache_cleanup_service.go`（+test）
- `apps/api/internal/services/cache_stats_service.go`（+test）
- `apps/api/internal/handlers/poster_file_handler.go`（新，+test）
- `apps/api/internal/handlers/cache_handler_test.go`
- `apps/api/cmd/api/main.go`
- `apps/web/src/lib/image.ts`（+spec）
- `apps/web/src/components/metadata-editor/PosterUploader.tsx`（+spec）
- `apps/web/src/components/settings/CacheManagement.tsx`（註解）、`CacheManagement.spec.tsx`、`CacheTypeCard.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/custom-poster.spec.ts`（新）、`tests/e2e/cache.api.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-cache-management/…`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/6-2-cache-management.md`（AC drift reference — see Completion Notes）
- `_bmad-output/implementation-artifacts/3-8-metadata-editor.md`（AC drift reference — see Completion Notes）

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-24 | 正式環境查證（0 張上傳海報）；後端移除 image 快取類型、新增 `/api/v1/posters/:file`；前端認得 `/posters/`；稿刪「圖片快取」卡；e2e 上傳→顯示→清快取後仍在 |
