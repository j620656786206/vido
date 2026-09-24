# Story poster-upload-b：在「修改資訊」裡換海報——選圖、拖曳、貼網址，按「儲存」才換上

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person whose film got the wrong poster from TMDb,
I want to pick my own image (or paste an image link) in「修改資訊」and have it replace the poster everywhere when I press 儲存,
so that fixing a poster is one obvious action instead of something only the API could do.

## Context

`disc-2026-09-poster-upload-no-ui-entry` 拆出的第二張。**依賴 `poster-upload-a-metadata-editor-dialog-v2`**（對話框外殼、左欄區塊、`LocalDetailV2` 已帶入 `posterUrl`）。設計：`ux-design.pen` **B′13**（桌面 `AFuPx`／手機 `oktn2`）與 **B′14 換海報・狀態與規則 Spec**（`AM0xm`），截圖 `_bmad-output/screenshots/flow-b-detail-v2/{b13p-d,b13p-m,b14-d}.png`。後端的檔案服務（`GET /api/v1/posters/:file`）與前端 `/posters/` 對應已在 PR #536 上線。

### ⚖️ 已定案（Sally 設計裁定＋ Sally × John 共同裁定，2026-09-24）

1. 入口只有一個：「修改資訊」→ 海報欄。不在選單另加「換海報」。
2. **選圖不上傳，按「儲存」才上傳並換上**；「取消」或關閉，什麼都不變。
3. 儲存時**先上傳海報，成功才存其他欄位**；任一步失敗，對話框不關，錯誤寫在出錯的那一格。
4. 存好後，詳情頁與媒體庫卡片**立刻**看到新海報，不用重新整理——**重新上傳同一部片也要換得掉**。
5. 手機：點海報或「換一張圖片」開系統選圖，沒有拖曳；按鈕高 44。
6. 這次不做：還原成 TMDb 海報、裁切／旋轉、批次換海報。
7. **清快取提醒句跟本張一起改**（Sally × John）：「…但不會影響影片**、字幕與你上傳的海報**。」理由：沒有上傳入口時講「你上傳的海報」只會讓人困惑；入口上線這句才是真的安心。

### 🔴 查到的事（main `4fe69f75`；行號皆為現況）——後端的上傳路徑有三個真 bug

1. **影集的海報上傳會「假成功」。** 前端把 `mediaType` 放在 multipart body（`apps/web/src/services/metadata.ts:172-174`），後端卻只讀 query（`handlers/metadata_handler.go:423` `c.DefaultQuery("mediaType","movie")`）→ 永遠當成電影 → 在 `movies` 表找不到這個 id → `updatePosterPath` 失敗只記 Warn（`services/metadata_edit_service.go:299-305`「Don't fail」）→ **API 回 200、檔案寫進資料夾、資料庫沒改**，使用者什麼都看不到。（e2e 的 helper 用 query 傳，`tests/support/helpers/api-helpers.ts:551`，所以沒測出來。）
2. **格式與大小檢查從來沒跑。** `UploadPosterRequest.Validate()`（`services/metadata_service.go:308-334`，JPG／PNG／WebP、5 MB）**沒有任何呼叫者**；handler 對 `ErrPosterInvalidFormat`／`ErrPosterTooLarge` 的 400 分支（`metadata_handler.go:459-470`）因此永遠走不到——傳一個 .gif 會變成 500「failed to process image」，傳 50 MB 會整個讀進記憶體。另外 `file.Read(fileData)`（`:436-437`）只讀一次，大檔可能讀不完整（應 `io.ReadFull`／`io.ReadAll`）。
3. **先寫檔、後查片子**（`metadata_edit_service.go:284-307`）：對不存在的 id 上傳會留下沒人指到的檔案（`disc-2026-09-poster-orphan-files` 的主因）。
4. **同一個網址換不掉圖**：海報路徑固定是 `/posters/<id>.jpg`（`images/processor.go` `GetPosterURL`），重新上傳後資料庫值不變 → React 的 `<img src>` 不變 → 不會重畫；瀏覽器雖然會以 `no-cache` 重驗，但沒有新的請求就沒有機會。
5. 前端：`useUploadPoster`（`hooks/useMetadataEditor.ts:40-51`）沒有呼叫者，失效的 `['media', id]` 對不上任何查詢；舊的 `PosterUploader.tsx` 只活在 gallery（`disc-2026-09-unmounted-v1-components`）。

## Acceptance Criteria

1. **後端：上傳路徑修正（🔴 #1–#3）。**
   - `mediaType` 先讀 multipart 欄位，沒有才讀 query（兩種都要有測試）；不是 `movie`／`series` → 400。
   - 用 `http.MaxBytesReader`（上限略大於 5 MB，例如 6 MB）限制 body；讀檔改 `io.ReadAll`；呼叫 `Validate()`；超過上限或 `ErrPosterTooLarge` → 400 `POSTER_TOO_LARGE`，格式不對 → 400 `POSTER_INVALID_FORMAT`。
   - **先確認片子存在**，不存在 → 404 `POSTER_UPLOAD_NOT_FOUND`，且**沒有寫任何檔案**。寫檔後資料庫更新失敗 → 刪掉剛寫的檔（`ImageProcessor.DeletePoster`）並回 500，**不再「Warn 然後回成功」**。
   - Go 測試：影集以 multipart 欄位上傳 → `series.poster_path` 真的更新；.gif → 400；6 MB+ → 400 且沒有檔案；不存在的 id → 404 且資料夾裡沒有新檔；DB 更新失敗（mock）→ 500 且檔案被刪。
2. **後端：換圖後網址會變（🔴 #4）。**
   - 上傳成功寫進資料庫與回應的路徑帶版本：`/posters/<id>.jpg?v=<unix 毫秒>`（縮圖同）。`GET /api/v1/posters/:file` 不受影響（`:file` 不含 query）；前端 `getImageUrl` 直接前綴 API base、保留 query（加一條單元測試）。
   - 舊資料（沒有 `?v=`）照常顯示，不遷移。
   - 測試：連續上傳兩次，兩次回應的 `poster_url` 不同、且都能 `GET` 到 200。
   - ⚠️ `tests/e2e/custom-poster.spec.ts` 目前斷言 `src$=".jpg"`，要改成「路徑部分」比對（屬本張刻意變更，Completion Notes 列出）。
3. **前端：海報格（B′13 左欄＋ B′14 九個狀態）。**
   - 新元件（例如 `metadata-editor/PosterField.tsx`）放進 a 張留下的左欄區塊，**取代**舊 `PosterUploader`（舊元件、spec、gallery 夾具、視覺基準一併刪除，並從 `disc-2026-09-unmounted-v1-components` 清單劃掉）。
   - 九個狀態逐一照 B′14：① 目前海報 ② 還沒有海報（按鈕字改「上傳圖片」）③ 拖曳到海報上（只有桌面；整格金色外框＋ scrim＋「放開就用這張」）④ 已選新圖・尚未儲存（新圖預覽、金色外框、`--accent-tint` 標籤「新海報・尚未儲存」）⑤ 儲存中（scrim＋「上傳中…」，整個對話框停用，儲存鈕「儲存中…」）⑥ 格式不對 ⑦ 太大（選的當下就擋，不送出）⑧ 上傳失敗（預覽留新圖、`--error` 外框）⑨ 改用圖片網址（按鈕區換成網址欄＋「改用上傳」連結；預覽跟著網址換）。錯誤文案**逐字**照 B′14。
   - 「換一張圖片」是真的 `<button>`，觸發隱藏的 `<input type="file" accept="image/jpeg,image/png,image/webp">`；拖曳區不是唯一入口（鍵盤只用按鈕就能完成）。錯誤訊息放在 `aria-live="polite"` 區、以 `aria-describedby` 連到按鈕。
   - 桌面提示句：「JPG、PNG 或 WebP，5 MB 以內。也可以直接把圖片拖到海報上。」；手機：「JPG、PNG 或 WebP，5 MB 以內。新海報會在按「儲存」後換上。」；桌面 footer 左側：「新海報會在按「儲存」後換上；按「取消」就不會動到目前的海報。」（只在有待存的新圖時顯示）。
   - 網址模式：預覽 `<img>` 載入失敗 → 顯示「這個網址打不開圖片。」且**儲存鈕停用**，直到網址改好或切回上傳。網址照舊寫進 `posterUrl`（不在伺服器抓圖，`FetchPosterFromURL` 不在本張範圍）。
4. **前端：儲存順序（定案 #2、#3）。**
   - 有待存的新圖：先 `uploadPoster`（`mediaType` 放 multipart 欄位）→ 成功才 `updateMetadata`（**不帶** `posterUrl`，避免蓋掉剛上傳的路徑）。上傳成功後清掉「待存」狀態，所以若接著欄位儲存失敗、再按一次「儲存」，**不會重傳圖**。
   - 上傳失敗 → 狀態 ⑧、對話框不關、欄位也沒存；欄位儲存失敗 → 錯誤在 footer（a 張既有行為）。
   - 取消／關閉 → 丟掉待存的圖（`URL.revokeObjectURL` 釋放預覽）。
   - 成功後失效 `detailKeys.localMovie/localSeries(id)` 與 `libraryKeys.all`（`hooks/useLibrary.ts` 那個）；`useUploadPoster` 同步修正或併入。
   - 測試（vitest，用真的 QueryClient）：上傳→更新的呼叫順序；上傳失敗時沒有呼叫更新；欄位失敗後重試只呼叫更新；取消後沒有任何呼叫；網址模式載入失敗時儲存停用。
5. **清快取提醒句（定案 #7）。**
   - `components/settings/CacheManagement.tsx:134-135` 改成「再按一次才會真的清除。這會刪掉 30 天前的所有快取，之後第一次瀏覽會比較慢，但不會影響影片、字幕與你上傳的海報。」＋對應 spec。
   - 設計稿 C18-D（`dfwSb`）的文字節點 `aldeK` 同步改字；重出 `flow-c-search-settings/c18-d.png`，只 stage 這張＋ `pen-tokens.json`；跑 `python3 scripts/check-design-tokens.py`。
   - 視覺基準 `settings-cache-management/confirm` 會變 → 過期 `-linux` 以 `git rm` 交給 CI bootstrap。
6. **端到端：真的從介面換海報。**
   - e2e（追加到 `tests/e2e/custom-poster.spec.ts`）：詳情頁按「修改資訊」→ 用 `setInputFiles` 選一張 JPEG → 看到「新海報・尚未儲存」→ 按「取消」→ 詳情頁海報沒變；再開、選圖、按「儲存」→ 對話框關閉、詳情頁海報 `src` 指向 `/api/v1/posters/<id>.jpg?v=…` 且 `naturalWidth > 0`；**再換一次**→ `src` 改變、不用重新整理。
   - 至少一條影集的上傳（API 層即可）證明 🔴 #1 修好。
7. **不准回歸**：a 張的所有測試、`lib/image.ts` 既有測試、`custom-poster`／`cache.api`／`metadata-editor.api` e2e 全綠；TMDb 海報與絕對網址海報照常顯示。
8. **設計比對、測試與 CI**：九個狀態各有 gallery 夾具（桌面，③ 只有桌面；另加手機 ④），與 `b14-d.png` 逐格比對；`b13p-d`／`b13p-m` 整體比對。紅／守（Rule 16）、mutation check（至少：拿掉版本參數、`mediaType` 改回只讀 query、存在檢查移到寫檔後、上傳前不擋 5 MB、更新時帶上 `posterUrl`）。`pnpm nx test api`、`pnpm nx test web`、`pnpm run lint:all`、typecheck、`check-design-tokens.py` 全綠；a11y 預檢。

## Tasks / Subtasks

- [x] **Task 1 — 後端：上傳 handler 修正（AC: #1）**：`mediaType` 來源、body 上限、`io.ReadAll`、`Validate()`、錯誤對應＋測試
- [x] **Task 2 — 後端：先查片子、失敗清檔（AC: #1）**＋測試
- [x] **Task 3 — 後端：版本化海報路徑（AC: #2）**＋測試；`custom-poster.spec.ts` 斷言調整
- [x] **Task 4 — 前端：`PosterField` 九個狀態、拖曳、網址模式（AC: #3）**；刪舊 `PosterUploader`
- [x] **Task 5 — 前端：儲存順序與查詢失效（AC: #4）**
- [x] **Task 6 — 清快取提醒句＋設計稿＋e2e＋基準＋收尾（AC: #5–#8）**

## Dev Notes

### 這張的重點

- **後端那三個 bug 要先修**：不修的話，介面做得再好，影集換海報照樣「顯示成功、什麼都沒變」。
- **「按儲存才換」是這張的核心承諾**：所有測試都圍著「取消不會動到任何東西」與「上傳成功才存欄位」。
- **網址要會變**：不然第二次換圖時使用者會以為壞了。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。`POST /media/:id/poster` 的回應欄位不變，**值**多了 `?v=` 查詢參數；唯一消費者是前端 `getImageUrl`（照樣前綴）。`mediaType` 多接受 multipart 欄位，query 照舊可用（e2e helper 不用改）。

### 不要做的事

- 不要在伺服器端抓網址圖片（`FetchPosterFromURL` 維持沒人呼叫）。
- 不要做還原 TMDb 海報、裁切、批次。
- 不要遷移舊的 `poster_path`。
- 不要動 a 張已完成的右欄欄位（只在左欄區塊與 footer 提示句）。

### 已知陷阱

- **`http.MaxBytesReader` 超限時**，`FormFile` 回的錯誤要辨識成「太大」（`*http.MaxBytesError`），不要變成「File is required」。
- **Blob 預覽**：`URL.createObjectURL` 要在換圖、取消、卸載時 `revokeObjectURL`。
- **拖曳事件**：`dragleave` 會在子元素間觸發；用計數器或 `relatedTarget` 判斷，避免狀態 ③ 閃爍。
- **TanStack Query**：用真的 QueryClient 測（`project_tanstack_refetch_no_data_resets_pending`）。
- **Pencil**：改 `aldeK` 後用選單 Save、grep 磁碟確認新字（`feedback_verify_pen_saved_before_commit`）；截圖只 stage `c18-d.png`。
- **視覺基準**：`-linux` 交給 CI bootstrap。
- **gh**：`GH_TOKEN=$(gh auth token --user j620656786206)`。

### Source tree

```
apps/api/internal/handlers/metadata_handler.go（+test）                         ← Task 1
apps/api/internal/services/metadata_edit_service.go、metadata_service.go（+test） ← Task 2
apps/api/internal/images/processor.go（GetPosterURL／GetThumbnailURL，+test）    ← Task 3
apps/web/src/components/metadata-editor/PosterField.tsx（新，+spec）、刪 PosterUploader.tsx（+spec） ← Task 4
apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx、hooks/useMetadataEditor.ts、services/metadata.ts ← Task 5
apps/web/src/lib/image.ts（+spec：保留 query）                                  ← Task 3
apps/web/src/components/settings/CacheManagement.tsx（+spec）、ux-design.pen、c18-d.png、pen-tokens.json ← Task 6
tests/e2e/custom-poster.spec.ts、-gallery.fixtures.tsx、tests/visual/…            ← Task 6
```

### Cross-Stack Split Check

後端 3（Task 1–3）、前端 3（Task 4–6）→ 兩邊都 ≤3，不拆。

### Time-dependent visual coverage

- `?v=<unix 毫秒>` 由後端產生；gallery 夾具一律用固定字串（例如 `/posters/abc.jpg?v=1`），不可在夾具裡讀現在時間。

### References

- [Source: `apps/api/internal/handlers/metadata_handler.go:399-480, 500`；`internal/services/metadata_edit_service.go:277-338`；`internal/services/metadata_service.go:300-350`；`internal/images/processor.go`（`ProcessPoster`、`GetPosterURL`、`DeletePoster`）]
- [Source: `apps/web/src/services/metadata.ts:163-193`；`hooks/useMetadataEditor.ts:34-51`；`components/metadata-editor/PosterUploader.tsx`；`lib/image.ts`；`components/settings/CacheManagement.tsx:130-137`]
- [Source: `tests/e2e/custom-poster.spec.ts`；`tests/support/helpers/api-helpers.ts:544-560`]
- [Source: `ux-design.pen` B′13 `AFuPx`／`oktn2`、B′14 `AM0xm`、C18-D `dfwSb`／`aldeK`；`sprint-status.yaml` → `disc-2026-09-poster-upload-no-ui-entry`、`disc-2026-09-poster-orphan-files`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- ⚠️ **建單事實更正（dev 查證）**：Context 🔴 #2「`Validate()` 沒有任何呼叫者」**不對**——`MetadataService.UploadPoster`（`metadata_service.go:887-935`）會先 `Validate()`（格式、5 MB）、再用 `posterUploader.Exists` 查片子，所以 .gif 本來就回 400、不存在的 id 本來就回 404 且不寫檔。SM 建單時只 grep 了 handler 與 `MetadataEditService`。**真的壞的是**：`Exists` 不分類型（電影或影集任一存在就過）→ 影集以電影身分上傳會先寫檔、再在 movies 表找不到、只記 Warn 回 200（🔴 #1 成立）；以及 handler 在讀檔前沒有上限、`file.Read` 只讀一次。AC #1 照實際缺口做，沒有重做已存在的檢查。
- **Task 1（handler）**：`mediaType` 先讀 multipart 欄位、沒有才讀 query、不是 movie／series → 400；`http.MaxBytesReader`（6 MB）超限 → 400 `POSTER_TOO_LARGE`，且 service 根本不會被呼叫；`io.ReadAll` 讀完整個檔，`FileSize` 改用實際讀到的長度。測試 +3。
- **Task 2（service）**：`MetadataEditService.UploadPoster` 先用 **該類型** 的 repo 查片子（`findByType`），找不到 → `ErrUploadPosterNotFound`、不寫任何檔；寫檔前把既有的 `<id>.jpg`／`-thumb.jpg` 改名成 `.bak`（白名單不會服務這種檔名），DB 寫入失敗 → 刪新檔、放回舊檔、回錯誤（不再「Warn 然後回成功」）；成功 → 刪 `.bak`。**多做了一步**：原單子只寫「失敗就刪檔」，但重新上傳會覆寫同名檔，直接刪會把資料庫仍指著的舊海報一起刪掉——所以改成暫存再還原。測試 +5（含「失敗的重新上傳保留舊海報、資料夾只剩兩個檔」）。
- **Task 3（版本化路徑）**：上傳寫進資料庫與回應的路徑帶 `?v=<unix 毫秒>`（縮圖同）；`GET /api/v1/posters/:file` 不受影響；`lib/image` 保留 query（+1 測試）；`custom-poster.spec.ts` 的路徑斷言改成比對 `?v=`（刻意變更）。
- **Task 4（`PosterField`）**：新元件取代 `PosterUploader`（元件、spec、gallery 夾具、6 張基準刪除）。九個狀態照 B′14；錯誤文案逐字照稿（集中在 `POSTER_COPY`）；拖曳用 enter／leave 計數避免閃爍；blob 預覽在 effect 裡建立與釋放（StrictMode 會重跑 effect，memo 版會被第一次 cleanup 釋放掉）；新圖／失敗的金色／硃砂框畫在圖片**上層**（inset ring 畫在盒子本身會被圖片蓋住——看基準圖才發現）；網址打不開時顯示空框而不是破圖示。「換一張圖片」是真的按鈕、觸發隱藏的 file input；錯誤在 `aria-live` 區並以 `aria-describedby` 連到按鈕。gallery 夾具提供 `initialState`（拖曳中、格式錯、太大）以拍到只有互動才看得到的狀態。
- **Task 5（儲存順序）**：選了新圖 → 先 `uploadPoster`（`mediaType` 在 multipart 欄位）→ 成功才存欄位；**只換海報時不呼叫欄位更新**（那個寫入會把片子標成「手動編輯」）；更新不帶 `posterUrl`；上傳成功後記住，欄位失敗再按儲存只重存欄位；上傳失敗 → ⑧、欄位不存、對話框不關；網址模式：載入成功才可儲存，經由欄位更新寫 `posterUrl`；上傳中整個欄位區 `fieldset disabled`、儲存鈕「儲存中…」。footer 在有待存新圖時顯示「新海報會在按「儲存」後換上；…」（僅桌面，照稿）。
- **Task 6**：清快取提醒句改成「…但不會影響影片、字幕與你上傳的海報。」（程式、spec、設計稿 `aldeK`、`c18-d.png`）；`check-design-tokens.py` 綠；`settings-cache-management/confirm` 基準因差異低於門檻**不會自動重拍**——手動刪掉 darwin 重產、`-linux` `git rm`（否則基準圖會留著舊句子）。e2e：從介面「修改資訊」選圖→取消不變→儲存後詳情頁顯示 `?v=` 新圖且真的解碼→再換一次 `src` 改變不用重新整理；影集以 multipart 欄位上傳真的寫進 `series.poster_path`。`--repeat-each=3` 9／9。
- 🔗 **AC Drift: FOUND** — `bugfix-custom-posters-served-and-not-cache` AC #6「上傳 API 的回應格式不變（`posterUrl` 仍回 `/posters/<id>.jpg`）」→ 值多了 `?v=`（欄位與前端處理不變）；`3-8-metadata-editor` AC3 → 上傳入口終於存在（B′13）。
- 📎 **Contract Stamps: NONE**（相關 story 皆無 `[@contract-v*]`）。
- 🎭 **A11y Pre-Flight: PASS**（`PosterField`、`MetadataEditorDialog`、`CacheManagement`；jsx-a11y 0 新增；隱藏的 file input `tabIndex=-1`＋`aria-hidden`，可見按鈕是唯一的鍵盤入口；錯誤 `aria-live="polite"`；網址欄有 sr-only label 與 `aria-invalid`）。
- **測試**：`nx test web` **289 files／4300 tests**、`nx test api`、`lint:all`（0 error）、typecheck、prettier、`check-design-tokens.py` 全綠；e2e `custom-poster`＋`metadata-editor.api`＋`cache.api` 全綠。
- **Mutation 8／8 紅**：拿掉版本參數、`mediaType` 只讀 query、拿掉類型存在檢查、拿掉失敗時的清檔／還原、拿掉 body 上限、拿掉前端 5 MB 檢查、更新時帶上 `posterUrl`、重試時重傳圖。

### 🔍 /ship Adversarial Review（2026-09-24）

0 HIGH／1 MEDIUM／2 LOW：
- **M1（修）** 海報上傳成功、欄位儲存失敗時，footer 仍寫「按『取消』就不會動到目前的海報」——但海報已經換了，取消也收不回來 → 改成顯示「新海報已經換上了；其他欄位還沒存，請再按一次『儲存』。」（`role="status"`），並拿掉那句承諾。測試 +2 斷言。
- **L1（不修，記錄）** 同一情境下海報格的標籤仍是「新海報・尚未儲存」；footer 的新句子已說清楚，標籤要改得動 `PosterField` 的 phase 語意，收益小。
- **L2（不修，記錄）** 欄位儲存中（非上傳中）時，海報格的按鈕仍可點；上傳中已整格鎖住。欄位儲存通常 <1 秒。

### 🎨 UX Verification（對 `b13p-d.png`／`b13p-m.png`／`b14-d.png`）

| Area | Design Spec | Implementation | Match? | Fix Needed |
| --- | --- | --- | --- | --- |
| 左欄（桌面） | 海報標籤、（新圖時）標籤、184×276、換一張圖片、改用圖片網址、提示句 | 同 | ✅ | — |
| 左欄（手機） | 104×156 在左，標籤／按鈕／連結在右，提示句在下 | 同 | ✅ | — |
| ① 目前／② 沒有 | 目前海報；「還沒有海報」＋「上傳圖片」 | 同 | ✅ | — |
| ③ 拖曳 | 金框＋scrim＋「放開就用這張」 | 同 | ✅ | — |
| ④ 已選 | 新圖＋金框＋「新海報・尚未儲存」 | 同 | ✅ | 🎨 UX Fix：框原本被圖片蓋住，改畫在上層 |
| ⑤ 上傳中 | scrim＋「上傳中…」、儲存中… | 同 | ✅ | — |
| ⑥⑦ 錯誤 | 原海報＋按鈕下的錯誤句 | 同（逐字） | ✅ | — |
| ⑧ 失敗 | 新圖＋硃砂框＋錯誤句 | 同 | ✅ | 🎨 UX Fix：同 ④ |
| ⑨ 網址 | 網址欄＋改用上傳；打不開的錯誤句 | 同；打不開時顯示空框 | ✅ | 🎨 UX Fix：原本顯示破圖示＋alt 文字 |
| footer | 左側「新海報會在按「儲存」後換上…」 | 有待存新圖時顯示（桌面） | ✅ | — |
| C18 | 「…不會影響影片、字幕與你上傳的海報。」 | 同（程式＋稿） | ✅ | — |

### Discovery Triage

- N/A — no out-of-scope work discovered（`disc-2026-09-poster-orphan-files` 的主因已由本張修掉，剩「刪片子不刪海報」與 e2e 留檔，維持原條目）。

### File List

- `apps/api/internal/handlers/metadata_handler.go`（+test）
- `apps/api/internal/services/metadata_edit_service.go`（+test）
- `apps/web/src/components/metadata-editor/PosterField.tsx`（新，+spec）
- `apps/web/src/components/metadata-editor/PosterUploader.tsx`、`PosterUploader.spec.tsx`（刪除）
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`（+spec）、`index.ts`
- `apps/web/src/components/settings/CacheManagement.tsx`（+spec）
- `apps/web/src/lib/image.spec.ts`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/custom-poster.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/{metadata-editor-poster-field/**（新）,metadata-editor-poster-uploader（刪）,metadata-editor-metadata-editor-dialog/**,settings-cache-management/confirm}`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-c-search-settings/c18-d.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/bugfix-custom-posters-served-and-not-cache.md`（AC drift reference — see Completion Notes）

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-24 | 建單（SM）：由 `disc-2026-09-poster-upload-no-ui-entry` 升級並拆出；本張為海報格互動＋後端上傳修正＋清快取提醒句。依賴 poster-upload-a |
| 2026-09-24 | 實作完成：上傳 handler／service 修正（類型、上限、失敗還原）、版本化路徑、`PosterField` 九狀態、按儲存才上傳、清快取提醒句；更正建單的 `Validate()` 事實；狀態 review |
| 2026-09-24 | /ship CR：海報已上傳但欄位失敗時，footer 改說實話 |
| 2026-09-25 | PR #542 合併（`66adf26b`），狀態改 done |
