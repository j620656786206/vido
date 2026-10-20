# Disc（後端）：TMDb 沒有台灣譯名時，片名和簡介不會再顯示成簡體字

Status: ready-for-dev

**Source:** `backlog-metadata-zh-fallback-simplified`。⚖️ Alexyu 2026-10-20 選 A：先修這張，再拍 FB 貼文的片庫截圖。這張排在 `disc-2026-10-generate-keeps-user-en-srt-b1` 前面。

**為什麼要做：** 2026-10-20 用 5 部 Blender 開放授權片做 FB 截圖用的展示片庫（全新容器、全新資料庫、第一次掃描）。5 部裡有 3 部片名是簡體：「钢铁之泪」「寻龙记」「大象之梦」。《Tears of Steel》的詳細頁，簡介也整段是簡體。

- Vido 的定位是「繁中優先」，台灣使用者第一次掃描片庫，第一眼看到的卻可能是簡體字。
- FB 截圖因為這個 bug 暫停，要修好才能拍。

## 查到的事（2026-10-20，`origin/main`）

- **後備順序**：`tmdb/fallback.go:10` `DefaultFallbackLanguages = {"zh-TW", "zh-CN", "en"}`。另外 `config.go:235` `TMDB_FALLBACK_LANGUAGES` 也可以用環境變數覆寫。
- **逐欄合併**：`fillEmptyMovieFields`（`:257`）、`fillEmptyTVShowFields`（`:342`）、`fillEmptySeasonFields`（`:440`，含每集）。zh-TW 沒有的欄位，就拿下一個語言的值補上。bugfix-d 之後是逐欄補，不是整包換掉。但從 zh-CN 補進來的 `Title`／`Overview`／`Tagline`／`Name`，**原樣存進資料庫**，沒有轉成繁體。
- **列表類也一樣**：搜尋、熱門、探索、推薦、相似（`SearchMoviesWithFallback` 等，`:68`～`:793`）。只要 zh-TW 沒有在地化內容，就整包用 zh-CN 的結果。
- **已經有的轉換零件**：`zhtw.Finalize(conv, text, countries)`（`zhtw/finalize.go:56`）。
  - 陸／港／澳片只轉字（s2tw），其他片轉成台灣用語（s2twp）並套台灣詞庫。這是 `project_cn_subtitle_policy` 的裁定（2026-10-05），字幕的轉換路徑已經在用。
  - `zhtw` 只依賴 `ai/prompts`，`tmdb` 套件可以直接 import。轉換器（Converter）要從外面注入。
- **類型名稱早就處理了**：`models/genre_names.go`。所有寫入都經過 `GenresJSON`，會換成台灣名稱。這張不碰類型。
- **已經存進資料庫的簡體不會自己變好**：正式機的《Tears of Steel》（TMDb 133701）存的就是「钢铁之泪」。

## Story

身為台灣的 NAS 使用者，
我想要片庫裡的片名和簡介都是繁體中文，就算 TMDb 只有中國的譯名，
這樣我打開 Vido 看到的就是一個繁中工具。

## Acceptance Criteria

1. **從 zh-CN 補進來的文字，一律轉繁體再存。**
   - 範圍：詳細資料（電影、影集、季、每一集）的 `Title`／`Name`、`Overview`、`Tagline`，加上列表類結果（搜尋、熱門、探索、推薦、相似）的片名和簡介。
   - 只要文字來自 `zh-CN`（或任何簡體語言碼，例如 `zh-SG`），都要經過 `zhtw.Finalize`。
   - 來自 `zh-TW` 和 `en` 的值**不轉**：一來避免 s2twp 改到原本就是繁體的句子，二來英文不用轉。
2. **用語規則跟字幕一樣。**
   - 詳細資料用 `ProductionCountries`。陸／港／澳片只轉字：例如「寻龙记」→「尋龍記」，不會改成台灣說法。其他片轉成台灣用語並套詞庫。
   - 列表類結果沒有出品國，就用 `OriginalLanguage` 判斷：`zh`／`cn`／`yue` 當成「保留原本說法」，其他照台灣用語轉。這個近似判斷要在程式註解寫清楚。
3. **轉換器怎麼來。** `LanguageFallbackClient` 多一個可選的轉換器（`zhtw.Converter`），用跟字幕同一個實作，在 `cmd/api` 接線。
   - 沒接轉換器時，要記一筆 Warn log，照原樣存，**不能讓抓資料失敗**（Rule 13：失敗要降級，不能整個壞掉）。
   - 轉換出錯時也一樣：記 log、存原文。
4. **已經存進去的簡體要補轉。** 開機時跑一次補轉（冪等：重跑結果一樣），範圍是 `movies`、`series`、`seasons`、`episodes` 的片名、簡介、標語。
   - 用 `subtitle` 套件現有的文字字體判斷（`Detect`）。判成簡體的才轉，繁體、英文、判不出來的一律不動。
   - **使用者手動改過的欄位不碰**：如果有「手動編輯」標記，就照標記；沒有標記的話，要在 Completion Notes 寫明你怎麼確認的。
   - 補轉完要記一筆 log：每張表各轉了幾筆。
   - 這段補轉可以用 `cache_entries` 或設定表記「已跑過」。
5. **測試（Rule 28 Real-Shape）。**
   - `tmdb` 層：用假 client 回「zh-TW 片名空、zh-CN 有簡體片名和簡介」，結果要是繁體。電影、影集、季（含集）各一個。
   - 列表：zh-TW 沒有在地化內容、整包改用 zh-CN 時，結果是繁體。
   - 陸片只轉字、外國片轉台灣用語，各一個案例。用真的轉換器跑，不要 mock 掉 `Finalize`。
   - zh-TW 原本的繁體值和 en 的值，一個字都不變。
   - 沒有轉換器、轉換器出錯時，存原文、不報錯。
   - 補轉：真 SQLite（`t.TempDir()`）放簡體、繁體、英文、手動改過的各一筆，只有簡體那筆被轉；重跑第二次，什麼都不變。
6. **正式機驗證。**
   - 合併、NAS 更新映像後，正式機的《Tears of Steel》要顯示「鋼鐵之淚」，簡介是繁體。
   - 展示片庫（`/mnt/user/data/tmp-vido-demo/`）重新掃描後，5 部都沒有簡體。
   - Completion Notes 要附 API 回傳的 `title` 和簡介前 30 字，修前、修後各一份。
7. `go test ./...`、`~/go/bin/staticcheck ./...`、`lint:all`（看「✖ N problems (X errors」那行）都要綠。

## 不做

- 後備順序要不要加 `zh-HK`（香港譯名是繁體，但用語和譯名跟台灣差很多）→ 不在本單。要做的話另開單問 Alexyu。
- 類型名稱 → 已經由 `disc-2026-10-genre-names-simplified` 處理。
- 前端顯示層 → 不改。存進去就是繁體，前端照樣顯示。
- 人名、角色名（演職員）→ 不在本單，先看實際有多少簡體再說。

## Dev Agent Record

### Completion Notes

（dev 填）
