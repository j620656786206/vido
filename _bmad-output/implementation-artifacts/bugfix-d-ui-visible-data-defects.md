# Story: Bugfix D — UI 可見資料缺陷四連發（簡體 genre／poster_path 三格式／year 全空／title 對調）

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Source:** NAS BUG HUNT（2026-07-13, party-mode）P1 bundle，全數經 live NAS 資料實證；2026-08-19 What'Sub 對抗重審列為無悔四項之 N3（README 第一句承諾「把繁體中文字幕這件事做到底」，genre 欄卻顯示「动作/冒险」）。
**Risk: 🟡 MED** —— 四個獨立根因的資料管線 bundle；修 code path 讓**新掃描**正確。既有 5,922 列的髒資料回填屬 bugfix-c（data migration），明確不在本案。

---

## Story

As a zh-TW NAS owner browsing my library,
I want genres in Traditional Chinese, posters that load, years that display, and titles the right way round,
so that the library page—the product's face—stops contradicting its own core promise on every row.

---

## Context —— 四個缺陷（live NAS 實證，sprint-status bugfix-d 條目原文）

| # | 缺陷 | 實證 |
|---|---|---|
| D1 | genre 是**簡體**（动作/冒险），且 `/library/genres` 對 5,922 部電影只回 **2 個** genre | `TMDB_DEFAULT_LANGUAGE=zh-TW` 環境下仍發生 → ~~某條 TMDb 取值路徑沒帶語言參數~~ **假設經查核推翻，見下方根因段** |
| D2 | `poster_path` 一欄三格式：222 列抽樣 = 103 絕對 TMDb URL / 1 相對路徑 / 118 空 | FE 前綴 base URL 的地方遇到絕對 URL → 壞圖 |
| D3 | `year` 全部 null，`release_date` 卻有值（如 `2026-02-17`） | year 推導從未執行或寫入點缺席 |
| D4 | 部分列 title/original_title 對調（title:"Mag Mag" / original_title:"禍禍女"） | zh-TW 標題落在 original_title → 清單顯示羅馬拼音 |

## 🔍 開工前根因查核（2026-08-20）—— story 的 D1 假設被推翻，D3 前提無法重現

### D1＋D4 是**同一個根因**，且不是「沒帶語言參數」

取值路徑**有**帶 zh-TW：`DefaultFallbackLanguages = ["zh-TW","zh-CN","en"]`（`tmdb/fallback.go:10`）。真正的缺陷是 **全有全無判定**：`hasLocalizedMovieDetails`（`:693`）要求 title **和** overview **都**非空才算 localized。TMDb 對冷門片常有 zh-TW 標題但 **zh-TW overview 空白** → 判定失敗 → 迴圈丟棄整包 zh-TW payload、改用下一個語言的**整包**結果，於是：

- `Genres[].Name` 一起變简体（动作／冒险）→ **D1**
- 再往下掉到鏈尾 `en` 時，`Title` 變英文（"Mag Mag"）而 `OriginalTitle` 留著中文（"禍禍女"）→ **D4**（story 描述的「對調」其實是**整包語言掉鏈**，不是欄位映射錯誤）

**⚖️ Alexyu 裁定（2026-08-20）：逐欄合併，維基另案。** 以 zh-TW 為底，只有空白欄位才從下一個語言補；genre 因為每個語言都會回傳，zh-TW 版本必定勝出。另外查證：`tmdb_provider.go` 從不設定 `TitleZhTW`，所以 `applyMetadataToMovie` 的 zh-TW 優先分支對 TMDb item 永遠走不到 —— 逐欄合併在**來源端**解決，該分支維持給豆瓣／維基使用。

**維基補 overview 已評估、另案處理**：`metadata/partial.go` 的 `MergePartialResults` 已有逐欄補洞機制、`wikipedia_provider.go:374` 已能產生中文簡介 —— 可行但屬 metadata orchestrator 層的獨立工程；另發現 `wikipedia/client.go:19` 的 `BaseURL` **未帶 `variant=zh-tw`**，故 `:376` 註解「Wikipedia zh content is in Traditional Chinese」不準確（一併留待該案處理）。

### D2 完全確認 —— 兩條寫入路徑各寫各的格式

- `applyTMDbMovieDetails`（`enrichment_service.go:636`）寫 `*details.PosterPath` → **相對路徑**
- `applyMetadataToMovie`（`:708`）寫 `item.PosterURL`＝`buildImageURL()` 產物 → **絕對 URL**（該行原註解「TMDB returns full URL like "/poster.jpg"」自相矛盾，正是這個 bug 的護身符）
- FE `lib/image.ts:7` 無條件前綴 base → 絕對 URL render 成 `.../w342https://image.tmdb.org/...` → 壞圖

### D3 前提無法重現 —— **本案不做，deferred**

全庫 grep：`movies` 表（`001_create_movies_table.go`）、`models.Movie`、所有 repository **都沒有 `year` 欄位**。FE 的年份是 `PosterCard.tsx:78` 用 `new Date(releaseDate).getFullYear()` **端算**的。因此「`year` 全部 null」找不到對應的可修物件。Alexyu 要求連 NAS（`192.168.50.52:8088`）查證，但**本 session 跑在雲端容器、無法連入區網**（實測 curl 逾時）→ D3 從本案切出，等實際查詢結果再定案。

## Acceptance Criteria

_（風險分層：本 bundle 為 lean AC —— 每缺陷一條可驗收行為＋一條回歸釘；根因定位屬 Task。）_

### AC #1 — D1 genre 繁中化
- 新掃描/enrichment 後，genre 以 zh-TW 寫入（动作→動作）；`/library/genres` 回傳數量與 TMDb zh-TW genre 清單一致（電影 19 類上限，實際依庫內容）。
- ~~根因修在**取值路徑**（帶 `language=zh-TW`）~~ → **依查核改為**：根因修在 **fallback 合併語意**（逐欄補洞取代全有全無）。仍**不**用事後 OpenCC 補救（genre 是 TMDb 官方詞表，不該進轉換器）—— 這條原則維持。
- ~~回歸釘：enrichment 單元測試斷言 genre 請求含 zh-TW 語言參數~~ → 語言參數本來就有；改釘**合併結果**：zh-TW 缺 overview 時，genre／標題／artwork 仍為 zh-TW，只有 overview 從下一語言借。電影與影集各一案。

### AC #2 — D2 poster_path 單一格式
- 裁定並文件化**單一儲存格式**（建議：一律存 TMDb 相對路徑，FE 統一前綴 —— 與既有多數 FE 假設一致；若現況 FE 已假設絕對 URL 則反向裁定，擇一，寫入 story Dev Notes）。
- 寫入路徑全部收斂到該格式；FE 顯示層對兩種歷史格式**容錯讀取**（防 bugfix-c 回填前的舊列壞圖）。
- 回歸釘：寫入層測試斷言格式；FE 測試斷言兩種輸入都渲染出合法 img src。

### AC #3 — D3 year 推導 ⏸️ **DEFERRED（本案不做，見上方根因查核）**
- 有 `release_date`（或 episode/series 對應日期欄）的列，掃描/enrichment 後 `year` 非 null；FE 年份重新出現。
- 回歸釘：解析測試 `2026-02-17 → 2026`；無日期 → year 維持 null（不發明資料）。

### AC #4 — D4 title 方向
- 寫入規則釘死：`title` = zh-TW 顯示標題（TMDb `title` with language=zh-TW），`original_title` = 原文。對調的根因（欄位映射或語言 fallback 順序）修正。
- 回歸釘：映射測試以「禍禍女／Mag Mag」為 fixture。

### AC #5 — 範圍紅線
- 既有髒資料列**不回填**（bugfix-c 的職權；本案修 code path，新掃描正確即過）。
- 不動 TMDb rate-limit／快取層行為（Rule 27 Five Pillars 既有機制）。

## Tasks / Subtasks

- [x] Task 1: D1 根因定位＋修復 —— 查核推翻原假設；改修 fallback 合併語意（`fillEmptyMovieFields` / `fillEmptyTVShowFields`）
- [x] Task 2: D2 格式裁定＋寫入收斂＋FE 容錯
- [ ] Task 3: ~~D3 year 推導寫入~~ ⏸️ **DEFERRED** —— 前提無法重現（無 year 欄位），且無法連入 NAS 查證
- [x] Task 4: D4 title 映射修正 —— 與 D1 同源，由同一個合併修復解決
- [x] Task 5: 回歸釘＋全綠驗證（D1/D2/D4）

## Dev Agent Record

### D2 格式裁定（AC #2 要求的單一格式）

**正規儲存格式＝TMDb 相對路徑**（`/abc.jpg`）。理由（證據導向，非偏好）：

1. FE `lib/image.ts` 全站無條件前綴 `https://image.tmdb.org/t/p/{size}` —— **多數假設已是相對路徑**
2. 尺寸段（`w342`/`w500`）是 **render 時的決定**，寫進 DB 等於把響應式圖片鎖死
3. 影像主機可能改變，不該固化在資料列裡

**但單一格式無法涵蓋全部**：orchestrator 會回傳豆瓣／維基來源的項目，它們的圖片在**自己的主機上、根本沒有相對路徑**可言。因此完整裁定是：

> 相對路徑＝TMDb 圖片（正規形式）；絕對 URL＝非 TMDb 來源（必要的例外），FE 原樣輸出不加前綴。

實作上 `MetadataItem` 新增 `PosterPath`/`BackdropPath`（僅 TMDb provider 填寫），寫入點優先取相對路徑、缺才退回絕對 URL。FE `getImageUrl` 以 `isAbsoluteUrl()` 分流 —— 這同時是 AC #2 要求的**歷史格式容錯**（bugfix-c 回填前的舊列不再壞圖），也永久修好了豆瓣／維基來源的圖片。

`getImageSrcSet`/`getBackdropSrcSet` 對絕對 URL 回傳 `null`（單一固定尺寸沒有 size ladder 可給），兩個消費端本來就以 `|| undefined` / `?? undefined` 處理，無回歸。

### 既有測試更新（釘住舊行為者）

`TestLanguageFallbackClient_GetMovieDetailsWithFallback` 的 "falls back when no overview" 案**釘的正是缺陷本身**（zh-TW 缺 overview 就期待整包 `en`）。已改寫為 "borrows only the missing overview, keeps the zh-TW title"，並為 table 新增 `wantOverview` 欄位補強斷言。

### 範圍紅線遵守

- 既有髒資料**未回填**（AC #5，屬 bugfix-c）—— 本案只修 code path
- **TMDb 呼叫次數未增加**：fallback 鏈本來就會逐語言發請求直到判定成功，改動只是**改用**先前被丟棄的回應，而非多發請求
- `MapQBState` 式的對映層未動；快取層（`cache.go` 以 `tmdb:movie/%d` 為 key、語言無關）語意不變

### 驗證

- Go 34 packages 全綠 · gofmt / go vet / staticcheck 2026.1（tmdb／metadata／services）乾淨
- web：`lib` + `components/media` 共 32 檔 445 測試全過 · `pnpm run lint:all` 0 errors · **`pnpm run format:check` 全 repo 綠**（上一個 story 的 CI 教訓：`nx lint web` 不涵蓋根目錄也不跑 prettier）
- 新增 7 測試：BE fallback 逐欄合併 4 案（電影 3＋影集 1）、BE 寫入層 3 案（相對路徑／絕對 URL 保留／zh-TW 標題優先）；FE 6 案（絕對 URL 直通／相對路徑仍前綴／srcset 行為）

## Change Log

| 日期 | 變更 |
| --- | --- |
| 2026-08-20 | 開工前根因查核推翻 story 的 D1 假設（語言參數本來就有，真因是 fallback 全有全無判定）並確認 D1＋D4 同源；Alexyu 裁定「逐欄合併，維基另案」。實作 `fillEmptyMovieFields`／`fillEmptyTVShowFields`（電影＋影集），D2 以 `MetadataItem.PosterPath` 收斂寫入格式＋FE `isAbsoluteUrl` 容錯，D4 由同一個合併修復解決。D3 前提無法重現（全庫無 `year` 欄位、FE 端算）且無法連入 NAS 查證 → **DEFERRED**。+7 測試，1 個釘住舊行為的既有測試改寫。Status → review。 |
