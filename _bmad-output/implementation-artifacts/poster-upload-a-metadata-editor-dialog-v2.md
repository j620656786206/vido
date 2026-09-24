# Story poster-upload-a：「修改資訊」對話框照 B′13 重做，類型存對、手機變成底部面板

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who fixes a film's details by hand,
I want the「修改資訊」dialog to look like the rest of the app, keep the genres I already have, and work as a proper sheet on my phone,
so that editing a film does not quietly turn「劇情」into「drama」or leave the page scrollable behind the dialog.

## Context

升級自 `disc-2026-09-poster-upload-no-ui-entry`（上傳入口）＋ `disc-2026-09-metadata-editor-no-design`（對話框沒有稿，已由 B′13 補上）。設計：`ux-design.pen` **B′13 修改資訊・換海報**（桌面 `AFuPx`／手機 `oktn2`）與 **B′14 規格**（`AM0xm`），截圖 `_bmad-output/screenshots/flow-b-detail-v2/{b13p-d,b13p-m,b14-d}.png`（PR #539）。

### ⚖️ SM 裁定：拆成兩張（2026-09-24，Alexyu 可在 review 推翻）

依「該拆就拆」（`feedback_split_oversized_stories`），切分線選**版面**：

- **本張（a）＝對話框外殼＋右欄所有欄位＋左欄只顯示目前海報（唯讀）。** 出貨後使用者得到一個照稿、類型正確、手機可用的編輯器。
- **下一張（b）`poster-upload-b-poster-field`＝左欄海報格的所有互動**（換圖、拖曳、改用網址、九個狀態、儲存順序）＋後端上傳修正＋清快取提醒句。

理由：兩半各動不同的區塊與檔案，各自有測試與價值；「先行為後外觀」會讓同一個元件被動兩次。

### 🔴 查到的事（main `4fe69f75`＋PR #539；行號皆為現況）

1. **類型存錯（真的 bug）。** `MetadataEditorDialog.tsx:54-73` 的 `GENRE_OPTIONS` 用英文 key（`'drama'`），但資料庫與詳情頁的類型是**中文名稱**（`enrichment_service.go:902` 存 TMDb 的 `g.Name`；夾具 `-gallery.fixtures.tsx:2110` 是 `['動作','冒險','科幻']`）。結果：打開對話框時原本的類型**一個都沒被選**；按儲存會把勾選的英文 key（例如 `drama`）寫回資料庫，詳情頁就顯示 `drama`。
2. **對話框是手刻的 fixed overlay**（`:198-246`），自己掛 `keydown`、自己鎖 `body` 捲動；沒有焦點困住、關閉後焦點不會回到「修改資訊」按鈕。專案其他對話框用 `ui/Dialog`（Radix）＋ `components/ui/mobileSheet.tsx` 的 `MOBILE_SHEET_CONTENT`／`SheetGrabber`，例：`ManualMatchDialogV2.tsx:35-68`（含「回到開啟者」的 `openerRef` 寫法）。
3. **標題不一致**：按鈕叫「修改資訊」（`LocalDetailV2.tsx:227`），對話框標題叫「編輯媒體資訊」（`:229`）。稿統一成「修改資訊」。
4. **沒被掛上的舊元件**：`GenreSelector.tsx`、`CastEditor.tsx` 只活在視覺 gallery（`disc-2026-09-unmounted-v1-components`）；對話框是自己內嵌類型與演員 UI。
5. **目前海報沒有傳進來**：`LocalDetailV2.tsx:120-135` `buildEditorMetadata` 沒帶 `posterUrl`，所以「海報圖片網址」欄永遠是空的。
6. **失效的 query key 對不上**：`hooks/useMetadataEditor.ts:24` 失效 `['media', id]`——沒有任何查詢用這個 key（詳情頁是 `detailKeys.localMovie(id)`＝`['details','local-movie',id]`，`hooks/useMediaDetails.ts:15-22`）。現在靠 `LocalDetailV2` 的 `onSuccess` refetch 才更新。

## Acceptance Criteria

1. **改用 `ui/Dialog`，桌面置中、手機底部面板（B′13）。**
   - `Dialog`／`DialogContent`＋`DialogTitle`「修改資訊」；桌面寬 760（`sm:max-w-[760px]`）、`--radius-lg`、`--bg-secondary`、header 56 高帶下框線、footer 帶上框線；手機走 `MOBILE_SHEET_CONTENT`＋`SheetGrabber`，頂角 `--radius-xl`、**footer 固定在底**（取消／儲存兩顆各佔一半、高 44）、中間可捲。
   - 關閉：Esc、關閉鈕（44×44）、取消；**點背景不關**（保留原本理由：編輯表單誤點會丟資料，`:205-207` 註解）。關閉後焦點回到「修改資訊」按鈕（照 `ManualMatchDialogV2` 的 `openerRef`）。
   - 刪掉手刻的 `keydown`／`body.style.overflow` 那段。
2. **版面照 B′13。**
   - 桌面：左欄 184 寬（「海報」標籤＋ 184×276 目前海報，`--radius-md`）、右欄欄位；手機：海報 104×156 與「海報」標籤同一列，下面欄位單欄。
   - 右欄順序：片名＊｜年份（120 寬，同一列）→ 英文片名 → 類型 → 導演 → 演員 → 簡介（4 行）。手機：片名 → 年份｜導演（同一列）→ 英文片名 → 類型 → 演員 → 簡介。
   - 欄位標籤 Label 級 `--text-secondary`；輸入框 `--bg-secondary`＋`--border-subtle`、高 36（手機 44）。
   - 左欄在本張**只顯示目前海報**（`getImageUrl(posterPath,'w342')`；沒有海報時顯示 B′14 ② 的「還沒有海報」佔位）。換圖相關的按鈕、連結、提示句、「新海報・尚未儲存」標籤**都屬 b 張**，本張不畫。
   - 移除「海報圖片網址」文字欄（由 b 張的「改用圖片網址」取代）。⚠️ 本張出貨到 b 張出貨之間，使用者**暫時無法**改海報網址——這個功能目前本來就壞的（#5：欄位永遠是空的、存了也沒人看到預覽），可接受。Completion Notes 寫明。
3. **類型：已選的顯示成 chip，存中文名稱（修 🔴 #1）。**
   - 已選類型顯示成 chip（`--accent-tint` 底、`--accent-text` 字、帶 × 移除，× 的可點區 ≥ 24px 且有 `aria-label="移除類型：劇情"`）；最後一顆「＋ 類型」（外框 chip）打開選單列出其餘類型。選單用既有的 Radix popover／dropdown（dev 先 `grep` `components/ui/` 找現成的，不新增套件）。
   - **值一律用中文名稱**（`動作`、`劇情`…），與資料庫一致；打開時原本的類型要被正確帶出。原本資料庫裡若有不在清單內的類型（例如豆瓣來的），**照樣顯示成已選 chip、可以移除**，不可在儲存時被悄悄丟掉。
   - 測試：`initialData.genres = ['劇情','某個清單外的類型']` → 兩顆 chip 都在；加「動畫」、移除「劇情」→ 送出的 `genres` 正好是 `['某個清單外的類型','動畫']`（順序：保留原順序、新加的接在後面）。
4. **演員：chip ＋「＋ 演員」。** 已有演員顯示成中性 chip（`--bg-tertiary`）帶 ×；「＋ 演員」點下去變成輸入框，Enter 加入、Esc 取消；重複或空白不加入（沿用 `:180-185` 規則）。
5. **舊元件的去留**：若 `GenreSelector`／`CastEditor` 改造後被新對話框使用 → 保留並更新夾具；若沒被用到 → **連同 spec、gallery 夾具、視覺基準一起刪**，並從 `disc-2026-09-unmounted-v1-components` 的清單劃掉。不准留下新的「只活在 gallery」元件。
6. **失效正確的查詢（修 🔴 #6）。** `useUpdateMetadata` 成功後失效 `detailKeys.localMovie(id)` 或 `detailKeys.localSeries(id)`（依 mediaType）與 `libraryKeys.all`（`hooks/useLibrary.ts` 那個 `['library']`，不是 `useMediaLibrary.ts` 的同名常數）。`LocalDetailV2` 的 `onSuccess` refetch 可保留或移除，擇一並說明。
7. **不准回歸。**
   - 必填與年份範圍驗證照舊（`:20-29` zod schema），錯誤訊息顯示在欄位下方並以 `aria-describedby` 連結。
   - 儲存中按鈕顯示「儲存中…」並停用；失敗時錯誤顯示在 footer（沿用 `updateMutation.error`）、對話框不關。
   - `MetadataEditorDialog.spec.tsx`、`LocalDetailV2.spec.tsx`、`tests/e2e/metadata-editor.api.spec.ts` 全綠；因標題改字或結構改變而改的斷言，Completion Notes 逐條列。
8. **設計比對與視覺基準。**
   - 桌面 1440、手機 390×844 各截一張，與 `b13p-d.png`／`b13p-m.png` 並排比對（左欄只比「目前海報」部分）；差異寫進 Completion Notes（Rule：`feedback_design_verification`）。
   - 更新 gallery 夾具 `metadata-editor-metadata-editor-dialog`（桌面＋ `mobile/`），並補一個 `initialData` 含 `posterUrl` 的案例；darwin 基準更新，過期 `-linux` 以 `git rm` 交給 CI bootstrap。
   - 檔頭 Rule 21 註記改成指向 B′13（`AFuPx`／`oktn2`），刪掉 no-screen 變體那段。
9. **測試與 CI**：紅／守（Rule 16）、主要修法做 mutation check（至少：類型值改回英文 key、失效 key 改回 `['media',id]`、點背景會關）；`pnpm nx test web`、`pnpm run lint:all`、typecheck 全綠；a11y 預檢（jsx-a11y 0 新增、Dialog 有標題、chip × 有名稱）。

## Tasks / Subtasks

- [x] **Task 1 — 外殼改 `ui/Dialog`＋手機面板（AC: #1, #7）**：焦點回開啟者、背景不關、固定 footer；刪手刻 keydown／overflow
- [x] **Task 2 — 版面照 B′13（AC: #2）**：左欄目前海報（唯讀、含無海報佔位）、右欄欄位順序、手機單欄；`LocalDetailV2` 的 `buildEditorMetadata` 帶上 `posterUrl: data.posterPath`；移除網址文字欄
- [x] **Task 3 — 類型與演員 chip（AC: #3, #4, #5）**：中文名稱當值、清單外類型保留、＋類型選單、＋演員輸入；舊元件去留
- [x] **Task 4 — 查詢失效、夾具、基準、比對（AC: #6, #8, #9）**

## Dev Notes

### 這張的重點

- **類型那個 bug 是真的在壞資料**：每存一次就把中文類型換成英文 key。先寫會紅的測試再修。
- **左欄在本張是唯讀的**：不要順手做換圖；b 張會在同一格加按鈕與狀態。本張把左欄做成一個乾淨的區塊（例如 `PosterColumn`／`posterSlot`），讓 b 張只換裡面的內容。
- **手機是底部面板**：照 `ManualMatchDialogV2` 那套 class，不要自己發明。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。`PUT` 更新中繼資料的請求格式不變（`genres` 仍是字串陣列，只是值改成中文名稱——那本來就是資料庫的格式）。

### 不要做的事

- 不要做換圖、拖曳、改用網址、上傳（全部是 b 張）。
- 不要改後端。
- 不要讓點背景關閉對話框。
- 不要新增套件。

### 已知陷阱

- **Radix Dialog 的焦點回歸**：開啟者是一般按鈕不是 `DialogTrigger`，要用 `onCloseAutoFocus` ＋ `openerRef`（`ManualMatchDialogV2.tsx:39-57`）。
- **兩個都叫 `libraryKeys`**：`hooks/useLibrary.ts`（`['library']`，媒體庫清單）與 `hooks/useMediaLibrary.ts`（`['media-libraries']`，資料夾設定）。要失效的是前者。
- **手機斷點**：量 390 寬本身（`feedback_measure_the_breakpoint_you_return_to`）；年份｜導演同一列在 390 寬不能擠破。
- **視覺基準**：`-linux` 不能在本機產生；`git rm` 過期的，推上去後觸發 `Visual Regression` workflow 開 bootstrap PR。

### Source tree

```
apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx（+spec）   ← Task 1–3
apps/web/src/components/metadata-editor/GenreSelector.tsx、CastEditor.tsx（改造或刪除，+spec） ← Task 3
apps/web/src/components/media/LocalDetailV2.tsx（buildEditorMetadata 帶 posterUrl）  ← Task 2
apps/web/src/hooks/useMetadataEditor.ts（失效 key）                            ← Task 4
apps/web/src/routes/test/-gallery.fixtures.tsx、tests/visual/…/metadata-editor-*  ← Task 4
```

### Cross-Stack Split Check

後端 0、前端 4 → 不觸發跨棧拆分（已依規模另拆 a／b，見 Context）。

### Time-dependent visual coverage

- `new Date().getFullYear()` 是年份欄的預設值（檔頭第 1 行的 time-bomb 豁免）；夾具一律傳 `year`，照舊。

### References

- [Source: `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx:1-4, 20-43, 54-73, 88-185, 198-246, 454-475`]
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:67, 120-135, 220-229, 391-400`；`components/media/ManualMatchDialogV2.tsx:35-68`；`components/ui/mobileSheet.tsx:21, 40`]
- [Source: `apps/web/src/hooks/useMetadataEditor.ts:18-27`；`hooks/useMediaDetails.ts:15-22`；`hooks/useLibrary.ts:6-8`；`hooks/useMediaLibrary.ts:12-15`]
- [Source: `apps/api/internal/services/enrichment_service.go:902`（類型存中文名稱）]
- [Source: `ux-design.pen` B′13 `AFuPx`／`oktn2`、B′14 `AM0xm`；`sprint-status.yaml` → `disc-2026-09-poster-upload-no-ui-entry`、`disc-2026-09-metadata-editor-no-design`、`disc-2026-09-unmounted-v1-components`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（外殼）**：改 `ui/Dialog`＋`MOBILE_SHEET_CONTENT`＋`SheetGrabber`；桌面 760 寬、`--radius-lg`、有框線；手機底部面板、footer 固定（取消／儲存各半、高 44）。`onPointerDownOutside`／`onInteractOutside` 擋掉點背景；Esc、✕（自畫的 44×44「關閉」，`ui/Dialog` 內建的 16px「Close」以 `closeClassName="hidden"` 關掉）、取消都會關。關閉後焦點回到開啟者（`openerRef`，照 `ManualMatchDialogV2`）。**開啟時焦點放在「片名」**（Radix 預設會放在 ✕ 上，比對截圖時一眼看到金色框框在 ✕，像「要關了」）。手刻的 `keydown`／`body.style.overflow` 刪掉。表單加 `noValidate`：年份的 min／max 原本會被瀏覽器自己的泡泡攔下，看不到 zod 的中文訊息。
- **Task 2（版面）**：左欄「海報」＋目前海報（桌面 184×276、手機 104×156，手機標籤在右），沒有海報或載入失敗都顯示「還沒有海報」佔位；右欄 grid：桌面「片名｜年份(120)」→英文片名→類型→導演→演員→簡介，手機用 `order` 把導演拉到年份旁。`LocalDetailV2.buildEditorMetadata` 帶上 `posterUrl: data.posterPath`。「海報圖片網址」文字欄移除——**b 張出貨前暫時無法在介面改海報網址**（原本那欄一直是空的、存了也看不到預覽，實際上本來就不能用）。
- **Task 3（類型／演員）**：`lib/genres.ts` 新增 `genreNamesFor(movie|series)`，選單是資料庫存的中文名稱（電影 19 種、影集 16 種，來自 `GENRE_MAP`）。已選的是 `--accent-tint` chip＋`移除類型：X`；「＋ 類型」是 Radix DropdownMenu，只列還沒選的。**清單外的類型（例如豆瓣來的）照樣顯示、可移除、儲存時保留**。演員中性 chip＋「＋ 演員」→輸入框（Enter 加、Esc 取消、失焦收起、空白與重複忽略）；Esc 不會連帶關掉對話框（輸入框帶 `data-escape-local`，對話框的 `onEscapeKeyDown` 認得它——Radix 在 document 上先聽到 Esc，React 的 `stopPropagation` 擋不住）。**`GenreSelector`／`CastEditor` 改造後被對話框使用 → 保留**，舊的英文 key `GENRE_OPTIONS` 匯出刪除；兩個元件從 `disc-2026-09-unmounted-v1-components` 清單劃掉（sprint-status 已註記）。
- **Task 4（查詢失效、夾具、基準）**：`useUpdateMetadata`／`useUploadPoster` 成功後失效 `detailKeys.localMovie|localSeries(id)`＋`libraryKeys.all`（`useLibrary` 的 `['library']`）。`LocalDetailV2` 的 `onSuccess` refetch **保留**（它是同一份資料，多一次 refetch 無害，拿掉反而要改它的測試與註解）。gallery：`metadata-editor-metadata-editor-dialog`（桌面，無海報）＋新 `/mobile`（390×844，同源 `placeholder-poster.svg` 當海報——視覺測試會擋掉 TMDb）；類型／演員夾具換成新 props、`penNode` 指向 B′13 的 `Dcf86`／`r7iTr`。darwin 基準更新，過期 `-linux` 7 張 `git rm`，交給 CI bootstrap。
- **既有測試改動（刻意，逐條）**：`MetadataEditorDialog.spec` 整份重寫——標籤「標題（中文）／標題（英文）」→「片名／英文片名」（照稿）、「海報圖片網址」欄已刪、類型從 18 顆全攤的 toggle 改成 chip＋選單、類型值從英文 key 改中文、「年份」不再帶 *（稿只有片名帶 *；驗證照舊）、關閉鈕改用 `關閉` 名稱查；`GenreSelector.spec`／`CastEditor.spec` 因 props 改變重寫；`LocalDetailV2.spec` 的對話框 mock 改成記錄 props（新增一條「把海報交給修改資訊」）。
- 🔗 **AC Drift: FOUND** — `3-8-metadata-editor` AC1（「edit form with all editable fields … Poster (upload or URL)」）→ 本張暫時拿掉網址欄（b 張以「改用圖片網址」補回）；AC1 的類型欄從「18 個 toggle、英文 key」改成「chip＋選單、中文名稱」。
- 📎 **Contract Stamps: NONE**（本張與上游 3-8 皆無 `[@contract-v*]`；`PUT` 更新格式不變，`genres` 值改成資料庫本來的中文名稱）。
- 🎭 **A11y Pre-Flight: PASS**（4 個元件；jsx-a11y 在觸及檔案 0 新增——`autoFocus` 那條已改成 ref＋effect；Dialog 有 `DialogTitle`「修改資訊」、焦點困住與回歸、chip × 有名稱、錯誤以 `aria-describedby`＋`aria-invalid` 連結、footer 錯誤 `role="alert"`）。
- **測試**：`nx test web` **289 files／4297 tests** 綠；`nx test api` 綠；`lint:all`（0 error）、typecheck、prettier 綠。
- **Mutation 6／6 紅**：類型選單改回英文 key → 2 紅；失效 key 改回 `['media',id]` → 3 紅；拿掉點背景防護 → 第一輪**沒紅**（`fireEvent.pointerDown(document.body)` 碰不到 Radix 的外部判斷）→ 改成真的點 scrim 後紅；拿掉 Esc 例外 → 紅；`LocalDetailV2` 不帶海報 → 紅；拿掉海報載入失敗的退路 → 紅。

### 🔍 /ship Adversarial Review（2026-09-24）

0 HIGH／2 MEDIUM／1 LOW，吸收 2M、1L 立案：
- **M1** 演員框裡打了名字沒按 Enter 就按「儲存」→ 名字被失焦丟掉 → 改成**離開輸入框就加入**，只有 Esc 取消（`cancelledRef` 防止 Esc 後的失焦再加回去——jsdom 在元件卸載時不會觸發 blur，這個防護無法用 mutation 證明，屬防禦性寫法）。測試 +3。
- **M2** 清空年份按儲存 → zod 預設英文「Expected number, received nan」→ 改成「請輸入年份」。測試 +1。
- **L1（立案）** 沒有上映日期的片子，年份被預設成今年、儲存就寫進資料庫（舊碼同樣）→ `disc-2026-09-editor-unknown-year-saved-as-this-year`。

### 🎨 UX Verification（對 `b13p-d.png`／`b13p-m.png`）

| Area | Design Spec | Implementation | Match? | Fix Needed |
| --- | --- | --- | --- | --- |
| 外殼 | 760 寬、header 56＋下框線、footer 上框線、radius-lg | 同 | ✅ | — |
| 標題 | 修改資訊 | 修改資訊（原「編輯媒體資訊」） | ✅ | — |
| 左欄 | 海報標籤＋184×276 | 同；新圖標籤、換圖按鈕、提示句屬 b 張 | ✅（本張範圍） | — |
| 右欄順序 | 片名＊｜年份 → 英文片名 → 類型 → 導演 → 演員 → 簡介 | 同 | ✅ | — |
| 類型／演員 | accent-tint chip＋×、外框「＋ 類型」；中性 chip＋「＋ 演員」 | 同 | ✅ | — |
| footer | 取消（Secondary）＋儲存（Primary）；左側提示句 | 按鈕同；提示句屬 b 張 | ✅（本張範圍） | — |
| 手機 | 底部面板、海報 104×156 標籤在右、年份｜導演同列、footer 兩顆各半 | 同 | ✅ | — |
| 焦點 | 稿是靜態、無焦點 | 開啟時「片名」有焦點框 | ⚠️ 刻意 | 🎨 UX Fix：原本焦點框落在 ✕，改到片名 |

### Discovery Triage

- `disc-2026-09-editor-unknown-year-saved-as-this-year`（/ship CR L1，已寫進 sprint-status）。

### File List

- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`（+spec）
- `apps/web/src/components/metadata-editor/GenreSelector.tsx`（+spec）
- `apps/web/src/components/metadata-editor/CastEditor.tsx`（+spec）
- `apps/web/src/components/metadata-editor/index.ts`
- `apps/web/src/components/media/LocalDetailV2.tsx`（+spec）
- `apps/web/src/hooks/useMetadataEditor.ts`、`apps/web/src/hooks/useMetadataEditor.spec.tsx`（新）
- `apps/web/src/lib/genres.ts`（+spec）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/metadata-editor-{metadata-editor-dialog,genre-selector,cast-editor}/…`（darwin 更新、`-linux` 刪除、新增 `metadata-editor-metadata-editor-dialog/mobile/`）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/3-8-metadata-editor.md`（AC drift reference — see Completion Notes）

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-24 | 建單（SM）：由 `disc-2026-09-poster-upload-no-ui-entry` 升級並拆成 a／b；本張為對話框外殼＋欄位 |
| 2026-09-24 | 實作完成：`ui/Dialog`＋手機面板、B′13 版面、類型改存中文名稱（清單外保留）、演員 chip、失效 key 修正、夾具與基準；狀態 review |
| 2026-09-24 | /ship CR：演員框失焦即加入、年份清空的中文訊息；另立年份預設成今年的 disc |
