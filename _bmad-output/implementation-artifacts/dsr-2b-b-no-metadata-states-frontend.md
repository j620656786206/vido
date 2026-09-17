# Story DSR.2b-b：沒認出來的片，詳情頁說實話並給你真的能走的路（前端＋設計稿）

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Depends on: `dsr-2b-a-manual-match-backend`**（套用、單筆重新比對、修改資訊三個後端端點要先變成真的。⛔ `dsr-2b-a` 沒合併前不要開工。）

## Story

As the person who clicks a poster in 媒體庫 that the scanner could not match,
I want 詳情頁直接告訴我「沒找到這部片的資料」或「還在整理」，並讓我在同一頁選對的那部片或立刻比對,
so that 我不用對著一頁空白漸層與「失敗」徽章猜發生了什麼，而且按下去的每一顆按鈕都真的有用。

## Context

`epic-dsr` 的 `dsr-2b`，拆成兩張（⚖️ Alexyu 2026-09-17，見 `dsr-2b-a` 裁定紀錄）。**本張＝前端與設計稿**，後端在 `dsr-2b-a`。

⚠️ **這是補功能，不是對齊。** v2 詳情頁從來沒做過這兩個狀態；v1 的 `FallbackFailed`／`FallbackPending` 只活在視覺夾具裡，而且它們的出路（`搜尋 Metadata` 跳去 TMDb 搜尋頁、`手動編輯`）一條都救不回片子。設計稿 B6-M／B7-M 是 v1 手機抽屜的形狀、沒有桌機版——**要先重畫**。

⚠️ **驗收基準是 `.pen` 節點值，不是 PNG。** Flow B 兩個資料夾已在 `READABLE_FLOWS`（dsr-2）。

### 🔴 建單時查到的事

1. **判斷條件要用 `parseStatus`，不能沿用 v1 的 `tmdbId > 0`。** v1（5-11）用 `hasMetadata = tmdbId > 0`——豆瓣／Wikipedia／NFO 比對成功、或手動填過資料的片**沒有 `tmdb_id`**，v1 會對它們說「我們找不到這部電影的資料」。
2. **今天一部沒認出來的片長這樣**（`LocalDetailV2`）：hero 片名是原始檔名、海報首字是 **`[`**（`DetailHeroV2.tsx:110` 用 `title.slice(0, 1)`；`PosterCardV2.tsx:45-49` 早就有跳過符號的 `fallbackInitial`）、徽章「失敗」或「整理中」，下面只剩「檔案資訊」。**影集更慘**：`SeasonAccordion` 在 `tmdbId <= 0` 時整塊不渲染（`:159`），一集都看不到。
3. **「在地化資訊」會把檔名當片名翻譯並寫進 NFO**（`nfo_localizer_service.go:74-93, 128-131`）。
4. **「整理中」可能永遠不會結束。** 自動比對只在「有變動的掃描」之後跑（`main.go:465-486`），批次重新解析設完 pending 也沒人跑（`disc-2026-09-batch-reparse-never-runs`）。v1 的轉圈圈與 60% 假進度條（`FallbackPending.tsx:35`）是在宣稱「正在跑」——DESIGN.md:514「會動的元件就是在宣稱現在有工作在跑」。
5. **手動選片的舊對話框有三個 bug**：`titleZhTW` 大小寫錯（後端 `title_zh_tw` 經 `snakeToCamel` 變 `titleZhTw`，`caseTransform.ts:13`；型別寫 `titleZhTW`，`services/metadata.ts:21`）→ **正式 API 回來的中文片名永遠是 undefined**；套用成功後 invalidate `['media', id]`（`useManualSearch.ts:44-48`），詳情頁的 key 是 `['details','local-movie',id]`（`useMediaDetails.ts:21-22`）→ **套用完頁面不刷新**；`services/metadata.ts` 的 `fetchApi` 丟純 `Error`、沒有 `status`／`code` → **做不出錯誤碼膠囊與 409 的說明**。
6. **舊對話框不能直接改成 v2。** `tests/e2e/manual-search.spec.ts` 綁死了它的 DOM：點 `[data-testid="dialog-backdrop"]` 關閉（`:375-391`，Radix overlay 是 pointerdown 而且沒有這個 testid）、標題「手動搜尋 Metadata」（`:285`）、影集按鈕的 `bg-[var(--accent-primary)]` class（`:348-360`）、原生 `<select>`（`:334`）；`SearchResultCard.spec`／`SearchResultsGrid.spec` 綁死來源徽章與「來源：」。所以**本張另做一個 v2 對話框**，舊的留給 `/test/manual-search`。
7. **`''`（空字串）不是「整理中」。** API 建立的片是 `''`，其中很多已經有完整 TMDb 資料（`media-detail.spec.ts` 所有種子都是）；`deriveLifecycleStatus('')` 本來就不給徽章（`libraryStatus.ts:67-68`）。把 `''` 當整理中會對好好的片說「資料還在整理」，按「立即比對」反而把它打成 failed。

### 設計稿節點

| 代號 | 節點 | 內容 | 本張 |
| --- | --- | --- | --- |
| `B6-M` | `2m1Pv` | v1 手機抽屜「我們找不到這部電影的資料」 | 由新稿取代後**刪**（AC #9） |
| `B7-M` | `7UnDy` | v1 手機抽屜「正在搜尋電影資訊⋯」 | 由新稿取代後**刪**（AC #9） |
| `B3p-D` ／ `B3p-M` | `uRGu2` ／ `SzNRb` | v2 電影詳情桌機／手機 | 新稿的**底**（Copy，不要 Insert——見已知陷阱） |
| `B6p-D` | `Z42zy` | 找不到這部影片（置中狀態版面） | 狀態區塊的版面參考 |
| `Component/DialogFrame` | `m6KMPr` | 對話框母版 | 手動選片桌機用 |
| `Component/BottomSheet` | `SG1ln` | 手機 sheet 母版 | 手動選片手機用 |
| `Component/PosterCard-v2/Unmatched` | `n6Crb` | 媒體庫的未比對卡片 | ⛔ 不動（`disc-2026-09-unmatched-two-words`） |
| **新** `B10p-D` ／ `B10p-M` | — | 比對失敗（桌機／手機） | ✅ AC #1 |
| **新** `B11p-D` ／ `B11p-M` | — | 資料整理中（桌機／手機） | ✅ AC #1 |
| **新** `B12p-D` ／ `B12p-M` | — | 手動選片（桌機 dialog／手機 sheet，含確認步驟） | ✅ AC #1 |

`m6KMPr`、`SG1ln`、caption `WzXrL`（B6-M）、`tJIVc`（B7-M）由 SM 以 Pencil MCP 讀出；動手前再 `Get` 一次確認。

### 程式碼地圖

```
routes/media/$type.$id.tsx → LocalDetailV2（UUID）
  ├─ DetailHeroV2          片名、首字、徽章、動作列（管理字幕／修改資訊／在地化資訊／複製路徑）
  ├─ （新）DetailNoMetadataV2   ← 本張：比對失敗／整理中兩個狀態區塊
  ├─ DetailTechInfoV2      檔案資訊（沒有任何技術值、大小、路徑時整塊不渲染，:103）
  ├─ SeasonAccordion       tmdbId <= 0 時不渲染
  ├─ MetadataEditorDialog  修改資訊（dsr-2b-a AC #3 之後存檔會清掉「失敗」）
  └─ （新）ManualMatchDialogV2  ← 本張：手動選片（重用 useManualSearch／useApplyMetadata）
manual-search/ManualSearchDialog（v1）→ 只給 library/ParseFailureCard →/test/manual-search 與夾具；本張只修型別與檔頭
media/FallbackFailed.tsx、FallbackPending.tsx   ← v1，只活在夾具，AC #9 刪
```

---

## Acceptance Criteria

1. **設計稿先畫（Pencil MCP，全部 Copy 既有節點再改）。**
   - **`B10p-D`／`B10p-M` 比對失敗**：以 `B3p-D`／`B3p-M` 為底——hero 片名是檔名、海報是片名雜湊漸層＋第一個字母或中文字（不是 `[`）、徽章「失敗」（`$error-tint`／`$error-text`，`libraryStatus.ts:67`）；**hero 動作列拿掉「在地化資訊」**（AC #5），「管理字幕」改次要樣式（一個畫面只留一顆泥金實心＝狀態區塊的主要按鈕）。hero 下方是狀態區塊（AC #2 文案），接著「檔案資訊」。電影、影集共用一張稿，影集差異用註記說明。
   - **`B11p-D`／`B11p-M` 資料整理中**：同上，徽章「整理中」，狀態區塊用 AC #3 文案。**不准畫轉圈圈或進度條**；「比對中…」只畫在註記裡（按下按鈕後才出現）。
   - **`B12p-D`／`B12p-M` 手動選片**：桌機 `Component/DialogFrame`、手機 `Component/BottomSheet`。標題「手動選片」、搜尋框（預填清理過的檔名）、結果列表（海報／中文片名／原名／年份）、選中後的確認步驟（AC #4 文案）。**不畫**來源選擇、電影／影集切換、來源徽章。
   - 每張畫完用 `ctx.problems` 掃裁切；位置照 `DESIGN.md` §怎麼新增一張設計稿（Flow B 群組內、標題在上、不重疊）。
   - `scripts/export-pen-screenshots.py` 的 `SCREENS` 加 6 筆（`flow-b-detail-v2`，代碼 `b10p-d`／`b10p-m`／`b11p-d`／`b11p-m`／`b12p-d`／`b12p-m`）。
   - 🚨 存檔要確認 ` M ux-design.pen`（同值 `Update` 標髒後 AppleScript 存檔，`.claude/memory/feedback_verify_pen_saved_before_commit.md`）。

2. **比對失敗（`parseStatus === 'failed'`）。** 新元件 `media/DetailNoMetadataV2.tsx`（`variant: 'failed' | 'pending'`），渲染在 hero 之下、其他區塊之上，`data-testid="detail-no-metadata"`、`data-variant`。檔頭：`// Design ref: ux-design.pen Screen B10p-D (<id>) + Screen B11p-D (<id>) + Screen B10p-M (<id>) + Screen B11p-M (<id>)`（一行、以 `)` 結尾）。
   - 標題（h2）：電影「沒有找到這部電影的資料」、影集「沒有找到這部影集的資料」。
   - 說明：「自動比對沒有找到符合的作品。你可以自己選對的那一部。」
   - 按鈕：**「手動選片」**（主要，泥金實心，`data-testid="no-metadata-manual-match"`）→ 開 AC #4 的對話框；**「重新比對」**（次要，`no-metadata-rematch`）→ AC #3 的比對流程。
   - 兩行中性灰小字：「上次如果是網路或 TMDb 暫時出錯，可以再比對一次。」「也可以用上方的「修改資訊」自己填片名與年份。」（誠實：AI 解析結果依檔名快取，因解析錯而失敗的片重新比對通常還是失敗，見 `dsr-2b-a` 已知陷阱。）
   - 區塊本身不帶 `role="alert"`（這是頁面內容，不是剛發生的錯誤）。
   - **影集**：同一個區塊；`SeasonAccordion` 維持不渲染，**不要**為沒資料的影集另做分集清單。
   - **一個畫面只有一顆主要按鈕**：狀態區塊出現時，hero 的「管理字幕」改成次要樣式（`bg-[var(--bg-secondary)] text-[var(--text-primary)]`，同「修改資訊」）。
   - `LocalDetailV2.tsx` 第 1 行檔頭加 `+ Screen B10p-D (<id>) + Screen B11p-D (<id>)`。

3. **資料整理中（`parseStatus === 'pending'`，只有這個值）。** 同一個元件 `variant="pending"`。
   - 標題：「這部片的資料還在整理」。
   - 說明：「新加入的檔案會在下次掃描後自動比對。不想等的話，可以現在就比對。」
   - 按鈕：**「立即比對」**（主要，`no-metadata-rematch`）。
   - **`''` 與任何其他值 → 不顯示區塊**（🔴 #7），頁面照舊。`success` 不論有沒有 `tmdbId` 都不顯示。
   - **重新比對／立即比對的流程（兩個 variant 共用）**：
     - 呼叫 `POST /library/{movies|series}/:id/reparse`（confirmed against [@contract-v1] (Story dsr-2b-a AC #2)）。按鈕停用並寫「比對中…」，這時才可以用 `Loader2` 轉圈。
     - 200 → invalidate `detailKeys.localMovie/localSeries(id)` 與 `libraryKeys.all`。`parse_status: success` → 頁面自然變回正常詳情頁；`failed` → 變成 AC #2 的失敗區塊，並在區塊內顯示「重新比對完成，還是沒有找到。」（`role="status"`）。
     - 409 `ENRICHMENT_ALREADY_RUNNING` → 「媒體庫正在比對其他檔案，請稍後再試。」；504 `METADATA_TIMEOUT` → 「比對花太久，已經停止。請稍後再試。」；其他 → 「比對失敗，請稍後再試。」＋錯誤碼膠囊（同 dsr-1／dsr-2 的 class）。三句都在區塊內、`role="alert"`。
   - `libraryService.reparseMovie/reparseSeries` 回傳型別改成 `{ id, parseStatus, title, tmdbId }`；`useReparseItem` 補 `onSuccess` invalidate（今天完全沒有）。`libraryService` 的 `fetchApi` 已經丟 `ApiError`（`libraryService.ts:25-48`），直接讀 `status`／`code`。

4. **手動選片：新的 `media/ManualMatchDialogV2.tsx`。** 不改舊的 `manual-search/ManualSearchDialog` 的 DOM（🔴 #6）。
   - **外殼**：`ui/Dialog`（Radix）；`<sm` 時 bottom-sheet 定位，照 `ManageSubtitleDialogV2.tsx:1-5, 50, 295` 的做法。focus trap、Escape、捲動鎖交給 Radix，不要手刻。
   - **props**：`open`、`onOpenChange`、`mediaId`、`mediaType: 'movie' | 'series'`、`initialQuery`。搜尋固定 `source: 'tmdb'`、`mediaType` 由 `movie → 'movie'`、`series → 'tv'` 推出；**沒有**類型切換與來源選擇。
   - **重用** `useManualSearch`（300ms debounce 照舊）與 `useApplyMetadata`；結果列表自己寫（不要重用 `SearchResultCard`——它帶來源徽章，而且它的 spec 鎖死舊行為）。每列：海報（`posterUrl` 沒有或載入失敗時用片名漸層＋`fallbackInitial`）、中文片名（`titleZhTw || title`）、原名（不同時才顯示）、年份。空結果：「找不到符合的作品，換個關鍵字試試。」
   - **預填搜尋字**：有 `filePath` 時用它的檔名、沒有時用 `title`（套用或編輯過之後 `title` 已不是檔名），再經 AC #4 的清理函式。清理函式放 `utils/`：去副檔名、去方括號與圓括號內容、`.`／`_` 換空白、去解析度與編碼字樣（`1080p`、`2160p`、`x264`、`x265`、`HEVC`、`BluRay`、`WEB-DL`、`BD` 等），收斂空白；單元測試至少含 `[Leopard-Raws] Kimi no Na wa (BD).mkv` → `Kimi no Na wa`、`The.Matrix.1999.1080p.BluRay.x264.mkv` → `The Matrix 1999`、純中文檔名原樣保留。
   - **確認步驟**：「套用「{中文片名或原名}（{年份}）」？這會用它的片名、海報、簡介取代目前的資料，之後自動比對不會再改它。」＋「取消」「確認套用」。套用中「套用中…」停用。
   - **套用請求**補 `selectedItem.mediaType`（confirmed against [@contract-v1] (Story dsr-2b-a AC #1)）。成功 → 關閉對話框（頁面由 invalidate 刷新）。
   - **錯誤**（確認步驟內、`role="alert"`）：409 同 AC #3；404 `TMDB_NOT_FOUND` → 「TMDb 上找不到這部作品，請換一筆。」；其他 → 「套用失敗，請稍後再試。」＋錯誤碼膠囊。**不准直接印 `error.message` 的英文原文。**
   - **修三個 bug**（🔴 #5）：
     - `ManualSearchResultItem.titleZhTW` → `titleZhTw`，連同 `SearchResultCard.tsx:128-130`、`ManualSearchDialog.tsx:333-334`、兩支 spec、`-gallery.fixtures.tsx:1857-1906` 的 mock（只改鍵名，夾具像素不變）；
     - `useApplyMetadata.onSuccess` 改 invalidate `detailKeys.localMovie/localSeries(mediaId)`（依 `mediaType`）與 `libraryKeys.all`；
     - `services/metadata.ts` 的 `fetchApi` 改丟 `lib/apiError.ts` 的 `ApiError`（`message` 不變，舊呼叫端不受影響）；`ApplyMetadataParams.selectedItem` 補 `mediaType`、`ApplyMetadataResponse` 補 `tmdbId`、`parseStatus`。
   - **檔頭**：`ManualMatchDialogV2.tsx` → `// Design ref: ux-design.pen Screen B12p-D (<id>) + Screen B12p-M (<id>)`。`manual-search/` 的 `ManualSearchDialog.tsx`、`SearchResultsGrid.tsx`、`SearchResultCard.tsx`、`FallbackStatusDisplay.tsx` 目前指向 `QTqcC`（那是 **E4-D 媒體庫未比對篩選**，不是對話框）→ 改成 no-screen 變體，理由寫同一行：「v1 手動搜尋，只掛在 /test/manual-search；v2 見 ManualMatchDialogV2，去留見 disc-2026-09-unmounted-v1-components」。**只改檔頭，不改元件。**
   - 在 `LocalDetailV2` 掛載：只在狀態區塊出現時可開啟。

5. **沒資料的片不顯示「在地化資訊」。** `LocalDetailV2` 在 `parseStatus` 是 `failed` 或 `pending` 時不渲染 `NfoLocalizeAction`（🔴 #3）。其他情況照舊。**單元測試要給 `filePath`**——`NfoLocalizeAction` 沒有路徑時本來就隱藏（`NfoLocalizeAction.tsx:54-55`），不給路徑的測試會假綠。

6. **海報首字跳過符號。** 把 `PosterCardV2.tsx:45-49` 的 `fallbackInitial` 搬到 `utils/fallbackInitial.ts` 並匯出，`PosterCardV2`、`DetailHeroV2:110`、`ManualMatchDialogV2` 共用；`DetailHeroV2.spec` 加一條 `[FanSub] 未知電影` → `F`。檢查既有夾具 `media-detail-hero-v2` 的片名第一個字：若是符號，它的基準線會變（列入 AC #8）；不是就沒有像素差異。

7. **e2e 重新上線（`tests/e2e/media-detail.spec.ts:167-245`）。** 四條 `test.skip` 改寫成 v2，**不准只是解除 skip**：
   - `:187` → 沒有海報的片顯示 `detail-poster-fallback`，且首字不是 `[`（種子標題用 `[E2E] 無資料電影 …`）。
   - `:202` 比對失敗 → 種一部「亂碼標題、無路徑」的片，用 `dsr-2b-a` 的 `reparseMovie` helper（有 409 重試）讓它變 `failed`，再開詳情頁：`detail-no-metadata[data-variant=failed]`、標題文字、`no-metadata-manual-match`、`no-metadata-rematch`。**不要斷言「檔案資訊」**（API 種的片沒有路徑與大小，`DetailTechInfoV2` 整塊不渲染）；「在地化資訊」隱藏由 AC #5 的單元測試證明。
   - `:226` 整理中 → `seedMovie` 之後呼叫 `POST /library/batch/reparse`（新增 helper `batchReparse`，設成 pending、沒有人會跑）：`detail-no-metadata[data-variant=pending]`、「立即比對」可見、**頁面上沒有 `.animate-spin`**。刪掉 `:219-225` 那段過期註解。
   - `:230` → 手動選片：點按鈕 → 對話框開啟、搜尋框預填清理過的片名、**看不到**類型切換與來源選擇。
   - **新增一條真的套用**：失敗的片 → 手動選片 → 搜 `Fight Club` → 選第一筆 → 確認套用 → `detail-no-metadata` 消失、片名不是亂碼。CI 的 e2e 後端有 `TMDB_API_KEY`。
   - **既有種子不受影響**：`media-detail.spec.ts` 其他測試種的片是 `''`，不會出現區塊（🔴 #7）；確認它們仍綠。
   - describe 名稱的 `@story-5-11` 換成 `@dsr-2b-b`。

8. **視覺夾具與基準線。**
   - 新增（`routes/test/-gallery.fixtures.tsx`，全部不打網路、不含日期、`statesOnly: ['default']`）：
     - `media-detail-no-metadata-failed-v2`、`media-detail-no-metadata-pending-v2`；
     - `media-manual-match-dialog-v2`：`ManualMatchDialogV2` 開啟、`mediaType="movie"`，用 `seedQueries` 預塞 `metadataKeys.manualSearch(...)` 3 筆結果（`posterUrl: undefined`）。Radix 走 Portal——既有的 `manual-search-manual-search-dialog` 夾具特地寫「NOT Radix portal」（`:2963`），**先看 `ManageSubtitleDialogV2` 有沒有夾具、怎麼截的**再決定；截不到時記進 Dev Notes 並改用確認步驟以外的可控呈現，**不准跳過這個夾具**。
   - **既有基準線**：`manual-search-*` 三個夾具只改 mock 鍵名，**應該零像素差異**——若有差異就是改錯了；`media-detail-hero-v2` 視 AC #6 檢查結果。
   - 基準線照 dsr-2 AC #14 的五步流程（`.claude/memory/project_visual_baseline_intentional_change.md`）。⛔ 不要本機產 `-linux.png`。

9. **v1 遺留：有後繼版本就刪（⚖️ 2026-09-10 判準）。**
   - 刪 `media/FallbackFailed.tsx`、`FallbackPending.tsx` 與兩支 spec、兩個夾具（`media-fallback-failed`、`media-fallback-pending`，`-gallery.fixtures.tsx:1196, 1214`）、四張基準線（`-darwin`＋`-linux`）。
   - 刪設計稿 `B6-M`（`2m1Pv`）、`B7-M`（`7UnDy`）與 caption（`WzXrL`、`tJIVc`）；`SCREENS` 移除 `b6-m`／`b7-m`；`git rm` 兩張 PNG。刪之前確認沒有其他 instance 引用。
   - `CLAUDE.md` 的 `flow-b-detail-interaction/` 說明拿掉「Fallbacks (mobile)」；`flow-b-detail-v2/` 說明補「比對失敗／資料整理中／手動選片 (B10p–B12p)」。
   - 在 `disc-2026-09-unmounted-v1-components` 補一行：兩個 fallback 元件已刪（清單少 2）。
   - **`ParseFailureCard`、`/test/manual-search`、`manual-search/` 元件、`ColorPlaceholder` 不刪。**

10. **既有測試保留通過**，只有下列是本張刻意改的：`media-detail.spec.ts:167-245`（AC #7）；`SearchResultCard.spec.tsx`／`SearchResultsGrid.spec.tsx` 的 `titleZhTW` 鍵名（AC #4，只改鍵名）；`useManualSearch` 的 invalidate 斷言（若有）；`PosterCardV2.spec`（若 import 路徑變）；被 AC #9 刪掉的兩支 spec。特別守住：`LocalDetailV2.spec.tsx` 既有全部、**`tests/e2e/manual-search.spec.ts` 全部（一行都不准改）**、`LibraryBrowseV2` 批次重新解析。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試。⛔ 局部 vitest 綠之後**一定要跑 typecheck**。

## Tasks / Subtasks

- [x] **Task 0 — 確認 `dsr-2b-a` 已合併**（`POST /library/movies/:id/reparse` 回 `parse_status` 而不是 `reparse_queued`；`/metadata/apply` 要求 `selected_item.media_type`）
- [x] **Task 1 — 設計稿（AC: #1, #9 稿的部分）**
  - [x] B10p-D／M、B11p-D／M、B12p-D／M（Copy 既有節點）
  - [x] 刪 B6-M／B7-M 與 caption；`SCREENS` 增刪
  - [x] `ctx.problems` 掃裁切；確認 ` M ux-design.pen`
- [x] **Task 2 — 狀態區塊（AC: #2, #3, #5, #6）**
  - [x] 先寫紅測試：failed／pending／`''`／success 無 tmdbId、主要按鈕只一顆、在地化資訊（給 `filePath`）、比對三種結果與三種錯誤
  - [x] `DetailNoMetadataV2`；`LocalDetailV2` 接線；`reparse` 型別＋`useReparseItem` invalidate
  - [x] `fallbackInitial` 搬 `utils/` 共用
- [x] **Task 3 — 手動選片（AC: #4）**
  - [x] 清理檔名純函式＋單元測試
  - [x] `ManualMatchDialogV2`（先紅：預填、鎖類型、確認文案、三種錯誤、成功關閉）
  - [x] 三個 bug（`titleZhTw`、invalidate、`ApiError`）
  - [x] 檔頭（新元件＋`manual-search/` 四個 no-screen）
- [x] **Task 4 — e2e（AC: #7）**（含 `batchReparse` helper）
- [x] **Task 5 — 視覺夾具、基準線、刪 v1（AC: #8, #9）**
- [x] **Task 6 — 收尾（AC: #10, #11）**
  - [x] 全套閘門；匯出截圖只 stage Flow B 變動
  - [x] 收單時：關 `disc-2026-07-v2-detail-fallback-states`；更新 `disc-2026-09-unmounted-v1-components`

## Dev Notes

### 這張的重點

- **最有感的是 AC #4＋AC #2 的組合**：沒認出來的片，第一次可以在 app 裡被救回來。
- **「整理中」不能畫成在跑。** 大部分時候根本沒有東西在跑；轉圈圈只准出現在使用者按下「立即比對」之後。徽章本身是泥金（⚖️ 2026-09-11「整理中正在發生」），這跟「多半沒在跑」有衝突，屬詞彙裁定，已立 `disc-2026-09-pending-badge-claims-running`，本張不改徽章。
- **判斷條件看 `parseStatus`**（🔴 #1、#7）：`failed` → 失敗區塊、`pending` → 整理中區塊、其他一律不顯示。

### 不要做的事

- **不要做「跳去 `/search`」的出路**（TMDb 探索搜尋，選了也不會回寫這個檔案）。
- **不要改舊的 `ManualSearchDialog` 的 DOM 或行為**（🔴 #6），只准改檔頭與 `titleZhTw` 鍵名。
- **不要改媒體庫的「未匹配」篩選或卡片**（`disc-2026-09-unmatched-two-words`）。
- **不要刪 `ParseFailureCard`、`/test/manual-search`、`manual-search/` 元件、`ColorPlaceholder`**。
- **不要為沒資料的影集做分集清單。**
- **不要動任何後端檔案**——後端行為跟 AC 寫的不一樣時，停下來回報，不要在前端繞過去。
- **不要支援豆瓣／Wikipedia 的套用**（`disc-2026-09-apply-non-tmdb-sources`）。
- **不要改 `MetadataEditorDialog`**（它對沒有日期的片預填今年，`disc-2026-09-metadata-editor-prefills-current-year`）。

### 已知陷阱

- **Pencil `Insert` 新 frame 會位移約 50px、內容被裁掉**（dsr-8 Debug Log）。一律 `Copy` 既有節點再改；spacer 用固定高度。
- **Radix Dialog 走 Portal**：單元測試找對話框用 `screen`，不是 `container`。
- **`seedMovie` 種出來的片是 `''`**，不會出現區塊；整理中要靠 `batchReparse`、失敗要靠 `dsr-2b-a` 的單筆重新比對（會真的打 TMDb）。
- **e2e 平行跑、共用後端**：chromium 與 webkit-core 兩個專案、2 個 worker（`playwright.config.ts:120-130`）。重新比對與套用在批次比對進行中會 409，helper 要重試。
- **`DetailTechInfoV2` 沒有任何值時整塊不渲染**（`:103`）——API 種的片沒有路徑與大小。
- **換片會取代使用者上傳的海報**——確認文案寫「取代目前的資料」。
- **行號以建單時（2026-09-17，main `2a172a3a`）為準**；`dsr-2b-a` 合併後後端行號會變，前端行號應大致不變。

### Source tree

```
apps/web/src/components/media/DetailNoMetadataV2.tsx(+spec)   ← Task 2（新）
apps/web/src/components/media/ManualMatchDialogV2.tsx(+spec)  ← Task 3（新）
apps/web/src/components/media/LocalDetailV2.tsx(+spec)        ← Task 2, 3
apps/web/src/components/media/DetailHeroV2.tsx(+spec)         ← Task 2（首字）
apps/web/src/components/library/PosterCardV2.tsx              ← Task 2（fallbackInitial 搬走）
apps/web/src/utils/fallbackInitial.ts(+spec)                  ← Task 2（新）
apps/web/src/utils/cleanFilenameForSearch.ts(+spec)           ← Task 3（新，名稱可調）
apps/web/src/components/manual-search/*.tsx                   ← Task 3（只改檔頭與 titleZhTw）
apps/web/src/services/metadata.ts、services/libraryService.ts ← Task 2, 3
apps/web/src/hooks/useManualSearch.ts、hooks/useLibrary.ts    ← Task 2, 3
apps/web/src/components/media/FallbackFailed.tsx、FallbackPending.tsx(+spec) ← Task 5（刪）
apps/web/src/routes/test/-gallery.fixtures.tsx                ← Task 3, 5
tests/e2e/media-detail.spec.ts、tests/support/helpers/*       ← Task 4
tests/visual/components.visual.spec.ts-snapshots/components/** ← Task 5
ux-design.pen、scripts/export-pen-screenshots.py、_bmad-output/screenshots/flow-b-*、CLAUDE.md ← Task 1, 5
```

### Cross-Stack Split Check

後端 task **0 個**（全部在 `dsr-2b-a`）、前端／設計 task 6 個。後端 ≤ 3 → **本張不再拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 新元件與改動都不讀時鐘；AC #9 刪掉的 `FallbackFailed.tsx:94` 的 `new Date(createdAt)` 是解析 prop。三個新夾具不含日期。

### References

- [Source: `ux-design.pen` `2m1Pv`（B6-M）／`7UnDy`（B7-M）／`uRGu2`／`SzNRb`／`Z42zy`／`m6KMPr`／`SG1ln`／`n6Crb`] — Pencil MCP 讀出（B6-M「我們找不到這部電影的資料」「你可以手動搜尋，或等待系統自動比對」「搜尋中繼資料」「手動編輯」「比對失敗」；B7-M「正在搜尋電影資訊⋯」「系統正在比對檔案名稱與 TMDb 資料庫」）
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:1, 78-99, 133-150, 176-215, 257-341`] — 資料 hook、錯誤分流、動作列、區塊順序
- [Source: `apps/web/src/components/media/DetailHeroV2.tsx:101, 110`、`library/PosterCardV2.tsx:45-49`、`media/DetailTechInfoV2.tsx:103`、`media/NfoLocalizeAction.tsx:54-55`、`media/SeasonAccordion.tsx:159`]
- [Source: `apps/web/src/utils/libraryStatus.ts:55-70`] — 徽章與顏色裁定、`''` 不給徽章
- [Source: `apps/web/src/components/manual-search/ManualSearchDialog.tsx`、`SearchResultCard.tsx:1-23, 128-130`、`services/metadata.ts:10-74`、`hooks/useManualSearch.ts:36-49`、`utils/caseTransform.ts:13`、`hooks/useMediaDetails.ts:15-22`、`hooks/useLibrary.ts:95-108`、`services/libraryService.ts:25-48, 110-120`、`lib/apiError.ts`] — 對話框、三個 bug、型別
- [Source: `tests/e2e/manual-search.spec.ts:285, 334, 348-360, 375-391`] — 舊對話框被 e2e 綁死的地方
- [Source: `apps/web/src/components/ui/Dialog.tsx`、`subtitle/ManageSubtitleDialogV2.tsx:1-5, 50, 295`] — v2 對話框做法
- [Source: `apps/web/src/components/media/FallbackFailed.tsx`、`FallbackPending.tsx`、`routes/test/-gallery.fixtures.tsx:1196, 1214, 1857-1906, 2963`] — v1 遺留、夾具
- [Source: `apps/api/internal/services/nfo_localizer_service.go:74-93, 128-131`、`apps/api/cmd/api/main.go:465-486`] — 在地化資訊翻檔名、自動比對何時跑
- [Source: `tests/e2e/media-detail.spec.ts:167-245`、`tests/support/helpers/seed-helpers.ts`、`playwright.config.ts:120-130`]
- [Source: `DESIGN.md:223, 249-256, 514, 641`] — 固定詞彙、會動＝在跑、Primary
- [Source: `_bmad-output/implementation-artifacts/5-11-fallback-ui-enhancement.md`] — v1 規格（`''` 當失敗、`tmdbId > 0`——本張刻意不沿用）
- [Source: `_bmad-output/implementation-artifacts/dsr-2b-a-manual-match-backend.md` AC #1／#2 [@contract-v1]] — 後端契約
- [Source: `_bmad-output/implementation-artifacts/dsr-2-flow-b-detail-v2.md` AC #14、`dsr-8-flow-i-discover-v2.md` Debug Log] — 基準線流程、Pencil Insert 陷阱
- [Source: project-context.md#Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]
- [Source: `sprint-status.yaml` → `dsr-2b-a-manual-match-backend` / `disc-2026-07-v2-detail-fallback-states` / `disc-2026-09-unmounted-v1-components` / `disc-2026-09-unmatched-two-words` / `disc-2026-09-apply-non-tmdb-sources` / `disc-2026-09-pending-badge-claims-running` / `disc-2026-09-metadata-editor-prefills-current-year` / `disc-2026-09-batch-reparse-never-runs`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-17

### Debug Log References

- `pnpm nx test web --skip-nx-cache`：**3487 / 3487 passed**（263 files）；/ship CR 修正後重跑見下
- `pnpm nx test api`：PASS（本張沒動後端）
- `pnpm run lint:all`：**0 errors**（128／126 warnings 皆既有）；改過的元件單獨 eslint：0 problems · `web:typecheck`：PASS · prettier：PASS · `check-design-tokens.py`：一致（82 變數、194 張畫面、73 母版）
- **e2e 對真後端＋真 TMDb**（本機 `vido-api`＋`nx serve web`）：`media-detail.spec.ts`＋`manual-search.spec.ts` chromium **42 / 42 passed**，含新的 5 條 `@dsr-2b-b`（其中一條真的搜 Fight Club、套用、頁面變回一般詳情頁）。webkit-core 本機沒裝 WebKit，交給 CI。
- **視覺**：`CI=1 VISUAL_BUCKETS=6 --workers=6 --update-snapshots=all` 57 秒；191 張 re-render 雜訊全部還原，只留 3 個新夾具的 darwin 圖；再跑一次比對模式只剩 `retry-retry-notifications`、`glossary-panel-v2/seeded` 兩張紅——都是 `preexisting-fail-visual-darwin-three-stale-baselines` 點名的既有本機問題。刪掉兩個 v1 夾具的 4 張基準線（darwin＋linux）。
- **Pencil**：這次改用「Copy 整張畫面再改」＋對已經不要的節點 `Delete`（不是 `enabled:false`——停用的節點會觸發 fill_container 警告）。`Insert` 一個 ref 進 dialog body 之後又碰到版面錯亂（footer 被算到 y=881），刪掉 ref、把 footer 刪掉再從原稿 Copy 一份回來就正常；搜尋框改 Copy 既有的 SearchInput instance。gofmt 那種坑這裡沒有，但 Pencil 會在 `fill_container` 的父層不是 flex 時警告。
- `.pen` 存檔：同值 `Update` 標髒 → AppleScript File › Save → ` M ux-design.pen`。匯出 194 張，Flow I 四張 re-render 雜訊還原，留下 Flow B 的 6 張新圖、`b9-d`（改了 case A 的指向文字）與刪除 `b6-m`／`b7-m`。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 **AC Drift: FOUND**（`grep -rn "FallbackFailed\|FallbackPending\|no-metadata\|ManualSearchDialog\|titleZhTW"` 全部 story 檔）
  - `5-11-fallback-ui-enhancement` AC #2／#3／#5 → **v2 取代**：判斷改看 `parseStatus`（原本 `tmdbId > 0`，會對豆瓣／NFO 比對成功的片說找不到）；`''` 不再當失敗（API 建的片多半已有資料）；出路從「跳 `/search`＋手動編輯」改成「手動選片（真的套用）＋重新比對」；整理中不再轉圈。
  - `3-7-manual-metadata-search-and-selection` AC3 → 舊對話框的套用請求補送 `selectedItem.mediaType`（dsr-2b-a 之後必填；不補的話 `/test/manual-search` 開發頁套用會 400）、`titleZhTW` 更正為 `titleZhTw`（中文片名原本永遠讀不到）。**這是 AC #4「不改舊對話框行為」的一個刻意例外**：只多送一個欄位，DOM 與 e2e 綁定的東西都沒動，`manual-search.spec.ts` 全綠。
  - `ux2-3-detail-v2`／`9R-13b`（動作列）→ 沒有資料的片：「管理字幕」降為次要樣式、「在地化資訊」不顯示；有資料的片不變。
  - `story-20-2-e2e-test-data-seeding` → 它留下的 pending skip 與「無法種 pending」的說明已不成立（改用批次重新解析種）。
- 📎 **Contract Stamps: FOUND**（本張消費 `dsr-2b-a` AC #1、AC #2 的 [@contract-v1]；兩行 `confirmed against` 在 AC #3／#4，程式碼註解也標在 `ReparseResult` 與 `ApplyMetadataParams.selectedItem.mediaType`。本張不定義新契約。）
- 🎭 **A11y Pre-Flight: PASS**（改過的 7 個元件 eslint 0 problems。①圖片：結果列海報 `alt=""`（片名就在旁邊的文字裡）、載入失敗換片名漸層；②modal：`ManualMatchDialogV2` 用 Radix Dialog（焦點困住、Escape）；因為不是由 `DialogTrigger` 開啟，關閉時用 `onCloseAutoFocus` 自己還原焦點（CR #4）；③非同步揭露：重新比對結果 `role="status"`、錯誤 `role="alert"`、搜尋中 `aria-busy`；④自訂 widget：結果列是 `button` 帶 `aria-pressed`，搜尋框 `type="search"` 有 `aria-label`。）
- ✅ **Pre-existing failures: NONE**（視覺那兩張是已立案的本機既有問題）
- **主要按鈕只留一顆**：有沒有資料的區塊時，「管理字幕」改成 `bg-secondary`；單元測試斷言整頁泥金實心按鈕只剩 `no-metadata-manual-match`。
- **重新比對的結果只在本頁顯示**：`useReparseItem` 的實例在切換詳情頁之間會留著，所以「還是沒有找到」與錯誤句都只在 `variables.id === id` 時顯示（有測試）。
- **設計稿**：`B10p-D` 與 `B11p-D` 高度 960（註記寫得比 900 長）；兩張手機稿沒放註記（空間不夠，桌機註記涵蓋）。`B12p-D` 畫的是「已選一筆」的狀態（確認列出現）；夾具截的是「還沒選」的狀態，兩者都是真的畫面。`B9-D` 的「case A 見 B6-M／B7-M」改指 B′10／B′11。
- **對話框搜尋框**：`type="search"` 的瀏覽器原生清除鈕（藍色 ×）用 `[&::-webkit-search-cancel-button]:appearance-none` 關掉，設計稿沒有它。

#### 🎨 UX Verification（對照 `flow-b-detail-v2/b10p-d`、`b11p-d`、`b12p-d` 與夾具截圖）

| 區域 | 設計稿 | 實作 | 一致？ | 處置 |
| --- | --- | --- | --- | --- |
| 失敗區塊標題／說明 | 沒有找到這部電影的資料／自動比對沒有找到… | 同 | ✅ | — |
| 失敗區塊按鈕 | 手動選片（泥金實心）＋重新比對（次要） | 同 | ✅ | — |
| 失敗區塊小字 | 兩行中性灰 | 同 | ✅ | — |
| 整理中區塊 | 標題、說明、只有「立即比對」、不轉圈 | 同 | ✅ | — |
| hero 徽章 | 失敗＝error tint、整理中＝accent tint | `deriveLifecycleStatus` 同 | ✅ | — |
| hero 動作列 | 沒有在地化資訊、管理字幕次要 | 同 | ✅ | — |
| 海報首字 | 跳過 `[` → L | `fallbackInitial` | ✅ | — |
| 手動選片標題／搜尋框 | 手動選片、預填清理過的檔名 | 同 | ✅ | — |
| 結果列 | 海報縮圖（漸層＋首字）、中文片名、原名 · 年份 | 同 | ✅ | — |
| 選中列 | 泥金外框＋淡底＋勾 | 泥金 ring＋`accent-subtle` 底，**沒有勾** | ⚠️ | 狀態用 `aria-pressed` 與外框表達；勾號是裝飾，記在這裡不另立案 |
| 確認列 | 確認文案＋取消＋確認套用 | 同 | ✅ | — |
| 手機 | bottom sheet | `<sm` bottom-sheet 定位（同管理字幕） | ✅ | — |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - `disc-2026-07-v2-detail-fallback-states` → **AC #2, #3, #7**（本張主體；收單時關掉）
  - 在地化資訊把檔名當片名翻譯 → **AC #5**
  - 海報首字是 `[` → **AC #6**
  - `titleZhTW` 大小寫、套用後 invalidate 錯 key、`metadata.ts` 丟不帶碼的錯誤 → **AC #4**
  - `manual-search/` 四個檔頭指向 E4-D → **AC #4**
  - `FallbackFailed`／`FallbackPending` 的去留 → **AC #9**

- **② spawn-blocking-story**：**`dsr-2b-a-manual-match-backend`**（本張依賴它）。

- **③ backlog-with-carry-forward-link**
  - **`disc-2026-09-pending-badge-claims-running`**（建單驗證時立）— 「整理中」徽章是泥金（「正在跑」），但 pending 片多半沒有任何工作在跑。
  - **`disc-2026-09-metadata-editor-prefills-current-year`**（建單驗證時立）— 「修改資訊」對沒有日期的片預填今年，本張的失敗區塊會把人導去那裡。

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/web/src/components/media/DetailNoMetadataV2.tsx`（＋spec，11 條）
- `apps/web/src/components/media/ManualMatchDialogV2.tsx`（＋spec，12 條）
- `apps/web/src/utils/fallbackInitial.ts`（＋spec）、`apps/web/src/utils/cleanFilenameForSearch.ts`（＋spec）
- `apps/web/src/hooks/useManualSearch.spec.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/{media-detail-no-metadata-failed-v2,media-detail-no-metadata-pending-v2,media-manual-match-dialog-v2}/default-visual-darwin.png`
- `_bmad-output/screenshots/flow-b-detail-v2/{b10p-d,b10p-m,b11p-d,b11p-m,b12p-d,b12p-m}.png`

**修改：**
- `apps/web/src/components/media/LocalDetailV2.tsx`（＋spec，+11 條）— 狀態區塊、主要按鈕降級、隱藏在地化資訊、掛手動選片、檔頭
- `apps/web/src/components/media/DetailHeroV2.tsx`（＋spec）— 首字跳過符號
- `apps/web/src/components/library/PosterCardV2.tsx` — `fallbackInitial` 搬到 utils
- `apps/web/src/services/libraryService.ts`（＋spec）— `ReparseResult`
- `apps/web/src/hooks/useLibrary.ts`（＋spec）— `useReparseItem` invalidate
- `apps/web/src/services/metadata.ts` — `ApiError`、`titleZhTw`、`selectedItem.mediaType`、回應型別
- `apps/web/src/hooks/useManualSearch.ts` — 套用後 invalidate 正確的 key
- `apps/web/src/components/manual-search/{ManualSearchDialog,SearchResultCard,SearchResultsGrid,FallbackStatusDisplay}.tsx`（＋兩支 spec）— 檔頭 no-screen、`titleZhTw`、舊對話框補送 `mediaType`
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 三個新夾具、刪兩個 v1 夾具、`titleZhTw`
- `tests/e2e/media-detail.spec.ts` — 四條 skip 改寫成 v2＋一條真的套用
- `tests/support/helpers/api-helpers.ts`（`batchReparse`）、`seed-helpers.ts`（註解）
- `ux-design.pen` — 新增 B10p／B11p／B12p 桌機與手機、刪 B6-M／B7-M 與 caption、B3p-M 左移補位、B9-D 指向文字、Flow B 描述
- `scripts/export-pen-screenshots.py` — `SCREENS` 增 6 刪 2
- `_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-b-detail-interaction/b9-d.png`
- `CLAUDE.md` — Flow B 兩個截圖資料夾的說明
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉；關 `disc-2026-07-v2-detail-fallback-states`；更新 `disc-2026-09-unmounted-v1-components`
- `_bmad-output/implementation-artifacts/dsr-2b-b-no-metadata-states-frontend.md` — 本檔

**刪除：**
- `apps/web/src/components/media/FallbackFailed.tsx`、`FallbackPending.tsx`（＋兩支 spec）
- `tests/visual/components.visual.spec.ts-snapshots/components/{media-fallback-failed,media-fallback-pending}/default-visual-{darwin,linux}.png`
- `_bmad-output/screenshots/flow-b-detail-interaction/{b6-m,b7-m}.png`
- AC drift reference（未修改）：`5-11-fallback-ui-enhancement.md`、`3-7-manual-metadata-search-and-selection.md`、`ux2-3-detail-v2.md`、`story-20-2-e2e-test-data-seeding.md`

## 對抗式 Code Review（/ship，2026-09-17）

獨立 reviewer（fresh context，只讀）回報 **0 HIGH、3 MED、5 LOW**。**修 6、立案 3（含 1 個既有模式）**。新增的 5 條元件測試都在「把修正拿掉」的程式碼上確認會紅。修完後 web **3502 / 3502**、lint 0 errors、typecheck PASS。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| 1 | MED | 預填搜尋字碰到常見下載檔名會留雜訊：`The.Matrix.1999.1080p.BluRay.x264-SPARKS` → `The Matrix 1999 -SPARKS`、單獨的 `HDR` 刪不掉、`H.265` 變 `H 265`、`Charlotte's.Web` 的 Web 被刪 | 改成「在第一個發行標籤處截斷」＋去掉尾端年份（TMDb 用片名字詞比對，帶年份反而找不到）；只剝真的影片副檔名（`The.Last.of.Us` 不再被吃掉 `Us`）；`web`／`bd`／`dv` 這類也是片名字詞的不當截斷點。測試從 5 條擴到 16 條真實檔名。⚠️ 與 AC #4 的例子 `The Matrix 1999` 不同，改成 `The Matrix`——記在這裡 |
| 2 | MED | 影集的預填字用了 `file_path`，但影集的路徑是**資料夾**（扁平目錄下是媒體庫根目錄 → 預填 `TV`） | 影集改用片名（後端的影集重新比對也是用片名）；測試 |
| 3 | MED | TMDb 斷線時後端手動搜尋回 200＋空結果，對話框會說「找不到符合的作品，換個關鍵字試試」 | 後端問題，立 `disc-2026-09-manual-search-hides-source-errors` |
| 4 | LOW-MED | 對話框由普通按鈕開，Radix 找不到 trigger，關閉後鍵盤焦點掉到 body；A11y 記錄寫「關閉還原焦點」不正確 | `onCloseAutoFocus` 還原到開啟前的焦點（開啟者還在時）；「重新比對」改 `aria-disabled`（`disabled` 會在按下瞬間丟焦點）；兩條測試。同樣寫法的其他對話框立 `disc-2026-09-dialogs-without-trigger-lose-focus` |
| 5 | LOW | 套用失敗的錯誤句在改選另一筆後仍留著 | 選列與換搜尋字時 `apply.reset()`；測試 |
| 6 | LOW | 「整理中」按立即比對後、頁面重抓前，會在「資料還在整理」下面出現「還是沒有找到」 | 只在頁面本身已是 failed 時顯示；測試 |
| 7 | LOW | TestSprite 計畫與種子腳本仍指著已刪的 testid | 種子腳本註解已改；TestSprite 計畫立 `disc-2026-09-testsprite-fallback-testids-stale` |
| 8 | LOW | 「整理中」e2e 只看 `batch.success`，某個 id 失敗時也是 200 | 補斷言 `success_count === 1` |

查過不成立：請求／回應大小寫、錯誤碼送達、套用成功後對話框被卸載時 body 的 pointer-events、手機 class、Escape、夾具隔離與 key 雜湊、e2e 共用後端（沒有 spec 觸發真的批次比對）、記錄裡的數字。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | ✅ 收單 —— PR #452 合併進 main（commit b4faf515），**CI 17 項全綠**（含 4 個 e2e shard 與 4 個視覺 shard）。`-linux` 基準線：手動觸發 Visual Regression → bootstrap PR #453（3 張，只加圖）合回分支後轉綠。 |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／3 MED／5 LOW，修 6、立案 3。最重要的兩條：① 預填搜尋字對真實下載檔名會留下壓制組、HDR、H 265 之類的雜訊——改成在第一個發行標籤處截斷；② 影集的「檔案路徑」其實是資料夾，預填會變成 `TV`——影集改用片名。另外關閉對話框後焦點會回到按鈕、錯誤句不再黏著、整理中不會提早說「還是沒有找到」。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。web 3487/3487、api PASS、lint 0 errors、typecheck、token 一致；e2e（真後端＋真 TMDb）42/42。沒認出來的片，詳情頁第一次說實話並給出真的能走的路：「比對失敗」有手動選片（套用後頁面變回正常詳情頁）與重新比對，「資料整理中」只有立即比對、不轉圈；舊的 v1 fallback 元件、夾具、兩張 v1 手機抽屜稿一起刪掉。設計稿新增 6 張（B10p／B11p／B12p 桌機與手機）。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀）：本張相關 2 項 CRITICAL、6 項 SHOULD FIX，**全部併入**。最重要的兩項：① 原本要把舊的手動搜尋對話框改成 v2——但 `tests/e2e/manual-search.spec.ts` 綁死了它的背景點擊關閉、標題、class 與原生下拉，同時又規定那支 spec 一行不准改，兩條互相矛盾（改成另做 `ManualMatchDialogV2`，舊的只修型別與檔頭）；② 「失敗」e2e 原本斷言看得到「檔案資訊」與看不到「在地化資訊」——API 種的片沒有路徑，前者永遠紅、後者不做 AC #5 也會綠（前者拿掉，後者移到給了路徑的單元測試）。另外：`''` 原本算整理中，會對所有 TMDb 種子片說「還在整理」，按下去還會把好片打成失敗（改成只有 `pending`，e2e 用批次重新解析種 pending）；`metadata.ts` 的錯誤沒有碼（改丟 `ApiError`）；預填搜尋字要用檔案路徑的檔名；「重新比對」文案不能暗示一定有用；補檔頭、`CLAUDE.md` 說明、兩個新 backlog。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。由 `dsr-2b-flow-b-no-metadata-states` 拆出的前端半張（⚖️ Alexyu 2026-09-17 裁定拆兩張）。Pencil MCP 讀出 B6-M／B7-M（v1 手機抽屜、無桌機）；盤點 `LocalDetailV2`、`manual-search/`、v1 fallback、e2e。 |
