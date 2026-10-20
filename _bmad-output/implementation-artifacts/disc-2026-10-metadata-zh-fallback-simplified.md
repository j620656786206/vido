# Disc（後端）：TMDb 沒有台灣譯名時，片名和簡介不會再顯示成簡體字

Status: done

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

- **TMDb 層（AC #1～#3）：** 新增 `tmdb/simplified.go`。`LanguageFallbackClient` 每一個 `c.client.*` 呼叫之後，都多一行 `result = c.localize*(result, lang)`，總共 15 處，包含 discover 由呼叫端指定語言的那條路。
  - 回應語言是簡體（`zh-CN`／`zh-SG`／`zh-Hans*`）時，先複製一份再轉換，不寫回 client 給的物件。
  - 轉換範圍：詳細資料的片名、簡介、標語，季名、季簡介、每集名稱和簡介，列表的片名和簡介。
  - 用語規則：電影和影集用出品國決定；列表電影用 `original_language`；列表影集先看 `origin_country`，再看 `original_language`。
  - 季的回應沒有國家資訊，所以一律只轉字（`keepOwnWording`）。這比 story 寫的更保守，理由是寧可外國影集的集名少轉一些說法，也不要改掉陸劇原本的用語。
  - 一次回應裡的所有欄位用分隔字串接成一段，丟給 OpenCC 跑一次，再切回去（`zhtw.ConvertJoined`）。
  - 沒有接轉換器時只記一次 Warn；轉換失敗時記 log、保留原文，查詢本身不會失敗。
  - 接線：`main.go` 把 `subtitle.NewConverter()` 移到 TMDb 服務建好之後、任何查詢開始之前，OpenCC 可用時就呼叫 `tmdbService.SetChineseConverter`，所以不會有 data race。
- **⚠️ 偏離 story：「是不是簡體」不用 `Detect` 判斷，改用 `zhtw.SimplifiedFields`。** 規則是：s2tw 會改到這段文字，而 t2s 完全沒改到，或 t2s 改到的字數不到 s2tw 的一半。後半句是為了接住「簡體簡介裡引了一個繁體名字」這種情況。
  - 先試過 `Detect`：它的表只有大約 200 個字，「大象之梦」「霸王别姬」都判成「不確定」，等於漏掉。
  - 再試過「只要 s2tw 會改到就轉」：在正式機資料庫的副本上，這會把 25 部電影和 23 部影集的繁體字改壞，例如「蘇里」變「蘇裡」、「好傢伙」變「好傢夥」、「神祕」變「神秘」、「社區」變「社群」、「淀治」變「澱治」。
  - 所以多加一道 t2s 檢查：繁體字會被 t2s 改到，就不算簡體。TMDb 層也用同一道檢查，因為 TMDb 的 zh-CN 欄位有時是使用者用繁體填的。
  - 為了這個，`subtitle.Converter` 多了 `ConvertT2S`。正式映像的 `/usr/share/opencc/` 已經有 `t2s.json`（Dockerfile 會複製整個資料夾）。
- **補轉（AC #4）：** `services/chinese_metadata_backfill.go`，在伺服器開始接請求之前同步跑，上限 2 分鐘。
  - **只跑一次**：所有表都轉成功，才在設定表寫 `metadata.zh_hant_backfill_done`。之後開機看到這個標記就直接結束；之後新進來的資料，由 TMDb 層負責轉。
  - 如果跑失敗，就不寫標記，下次開機會再跑一次。重跑不會把已經轉好的資料再改一遍。
  - 寫入方式：每張表包在一個 transaction 裡；只寫有變的欄位，所以空的簡介會原樣保留。
  - **範圍比 story 窄：** 只轉 `metadata_source = 'tmdb'` 的電影和影集，以及這些影集底下的季和集。
  - 不轉 manual（使用者手動改的）和 nfo（使用者自己的檔）：這兩種是 story 本來就要求的。
  - 也不轉沒有來源的列：沒比對成功的片，片名就是檔名。
  - 也不轉豆瓣來源的列：豆瓣在抓的時候已經自己轉過繁體。
  - 跑成功時也會清掉 TMDb 快取，因為修好之前存進快取的答案可能還是簡體。
  - log 會寫每張表改了幾筆。
- **正式機資料庫副本實跑**（真 OpenCC，`.backup` 下來在本機跑）：
  - 第一次：電影 2 筆、影集 3 筆、季 0、集 0。其中包含 钢铁之泪→鋼鐵之淚（簡介也轉了），另外 2 部陸劇的片名和簡介轉成繁體（只轉字），還有 1 筆測試資料。這份文件是公開的，所以不寫出片庫裡其他片的片名。
  - 第二次跑：看到標記就直接結束，什麼都沒做。整個流程含編譯不到 2 秒。
  - 剩一個誤判：一部片名裡的「凶」變成「兇」。OpenCC 把「凶」當成「兇」的簡體寫法，這部片名很可能本來就是台灣譯名。兩種寫法在台灣都有人用，影響正式片庫 57 部裡的 1 部，記錄在這裡，不另外處理。
- **測試：**
  - `tmdb/simplified_test.go`：電影、影集、季與集、列表、陸片只轉字、外國片轉台灣說法、HK／US 影集用出品國判斷、zh-TW 和 en 不轉、zh-CN 裡填的是繁體就不轉、沒有轉換器、轉換失敗、語言碼判斷。
  - `tmdb/simplified_opencc_test.go`：走伺服器實際用的 `subtitle.Converter`，真的開 OpenCC 處理程序（repo 的 stand-in 腳本），確認分隔字串來回一趟不會壞。
  - `zhtw/script_test.go`：判斷規則、混合字體、錯誤處理、分隔字串被吃掉時拒絕套用。
  - `services/chinese_metadata_backfill_test.go`：真 SQLite（`t.TempDir()`、跑完整 migration），只有簡體的列被轉；繁體、英文、檔名、manual、nfo、手動改過影集底下的集都不動；陸片只轉字；快取只清一次；第二次跑完全沒變化；轉換失敗不寫入；沒有轉換器就什麼都不做。
- **跨模型 CR（Sonnet，2026-10-20）：** 沒有 blocker。四項 should-fix 都處理了：
  - (1) 「簡體裡夾一個繁體字」原本整段會被判成不是簡體，現在用改動字數比較，見上面的規則。
  - (2) 呼叫端明確要求 zh-CN 時，也照樣轉成繁體。這是刻意的，程式裡有註解。
  - (3) 補轉原本每次開機都掃一遍，也沒有包 transaction。現在改成只跑一次，每張表一個 transaction。
  - (4) 原本兩個欄位都寫回，空字串會被改成 NULL。現在只寫有變的欄位。
  - nit 不改：季和集只轉字、不套台灣用語。理由寫在上面「TMDb 層」那段。
- **檢查：** `go test ./...`、`go vet`、`~/go/bin/staticcheck ./...` 全部乾淨；`lint:all` 是「✖ 170 problems (0 errors」。
- **正式機驗證（AC #6，2026-10-20 11:02，映像 revision `458bf2b8`）：** 通過。
  - 開機 log：`Simplified metadata backfill finished movies=2 series=3 seasons=0 episodes=0`，接著 `cleared cached TMDb answers entries=63`。補轉從 OpenCC 啟動到結束不到 1 秒。
  - 《Tears of Steel》，`GET /api/v1/movies/:id`：
    - 修前：`title` 是「钢铁之泪」；簡介前 30 字「Blender 基金会的第四部开源电影 Tears of Steel 问世，与」。
    - 修後：`title` 是「鋼鐵之淚」；簡介前 30 字「Blender 基金會的第四部開源電影 Tears of Steel 問世，與」。
  - 展示片庫（5 部 Blender 片，全新資料庫、第一次掃描，走 TMDb 層而不是補轉）：鋼鐵之淚、大雄兔、尋龍記、大象之夢，加上 Caminandes: Gran Dillama（TMDb 只有英文）。5 部都沒有簡體。截圖給 FB 貼文用，存在作者的電腦，不放進 repo。
