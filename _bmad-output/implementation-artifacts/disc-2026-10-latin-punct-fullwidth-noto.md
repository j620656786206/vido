# Disc：英文簡介裡的 ’ “ ” 不再跟後面的字空一大格（Frieren’s 不會變成 Frieren’ s），中文標點照舊

Status: done（⚖️ **2026-10-22 關單：真實資料沒出現，不修**——Alexyu 選 A）

## ⚖️ 關單紀錄（2026-10-22）

- **為什麼不做：** 寫 AC #6 的 E2E 時照 Rule 28 去找「含彎引號的真實 TMDb 英文簡介」，**找不到**：TMDb 熱門電影 100 部（`/movie?page=1..5`）、熱門影集 60 部（`/tv?page=1..3`）、加上 30 部經典片的英文簡介，**0 部**含 `‘’“”`，全部是直的 `'`（直撇號在 Noto 本來就是正常寬度）。正式機 2026-10-19 的 2 段英文簡介也沒有彎引號。最初的 `Frieren’s` 來自測試用畫廊夾具，不是真實畫面（TMDb 上 Frieren 209867 的英文簡介用的是直的 `humanity's`）。為一個真實資料碰不到的情況多加一個字型依賴不划算。
- **Sally 已追認的做法（留給以後）：** B1。DESIGN.md 句子定稿：「**英文段落的引號：** 標了 `lang="en"` 的英文段落裡，`‘’“”` 這類彎引號（`U+2018-201F`，共 8 個）改由 Source Sans 3 畫。這個字型隨 Vido 打包，只含這幾個字，畫面上出現時才下載，放大 110% 讓它跟 Noto 的字母一樣大。改用它是因為 Noto Sans TC 的這幾個字是全形，`Frieren’s` 會多出一格空白。字母仍由 Noto Sans TC 負責；中文段落的“”、…，以及分隔點 `·`，維持 Noto 的全形。」附帶條件：粗體 h1 的引號要截圖確認；原文片名「君の名は。 · Your Name」這種混排不標 `lang`。
- **當時寫到哪：** `utils/textLang.ts`（有拉丁字母、且沒有任何漢字／假名／諺文／注音／`U+3000-303F`／`U+FF00-FFEF` 才回 `'en'`）、`DetailHeroV2` 片名與原文片名、`LocalDetailV2`／`TMDbDetailV2` 簡介標 `lang`、`styles.css` 的 `@font-face`＋`@layer base` 規則、授權檔、`styles-fonts.spec.ts` 4 條、元件測試，vitest 79 條綠；建置確認只多 1 個 28,740 bytes 的 woff2。全部撤掉，沒有合併。
- **什麼時候重開：** 在真實畫面（正式機或 TMDb 即時資料）看到英文簡介／片名裡的 `’ s` 空隙時。重開直接照上面 B1 與 AC #1–#9 做；E2E 用當時找到的那段真實簡介。

（以下為 2026-10-19 建單時的原文，保留供重開時參考。）

原狀態：ready-for-dev（**開工前 Sally 追認**推薦做法，見下）

**Priority:** P3

**Source:** Rule 24 lane ③，2026-10-19 由字型 PR #840（`disc-2026-10-font-sans-not-noto`）的 dev 回報：Noto Sans TC 的 `’` `“` `”` `…` `·` 都是全形寬；中文畫面沒問題（也符合設計稿），但英文段落裡的彎引號會出現明顯空隙（例 `Frieren’s`）。字型單寫明「不要開 `font-feature-settings`（例 `palt`）——設計稿沒有，會改變中文標點寬度」（該單 Dev Notes）。
⚖️ **Alexyu 2026-10-19：開單。**

**⚠️ 開工前 Sally 追認：** 這是字型／設計系統決策——推薦做法會在 Vido 裡多放一個字型檔（只拿來畫英文段落裡的 4 個引號），並在 DESIGN.md §Typography 加一句。Sally 同意推薦做法（或改選表上另一個）之後才開工；她選別的，只要改 AC #1–#3，其餘 AC 的量測方式不變。

## 查到的事（2026-10-19，本 session 讀過的程式碼、字型檔、正式機資料，並在真瀏覽器量過）

### 字型檔本身

- Vido 打包的 Noto Sans TC latin 區塊（`node_modules/@fontsource-variable/noto-sans-tc/files/noto-sans-tc-latin-wght-normal.woff2`，`unicode-range` 含 `U+0000-00FF`、`U+2000-206F`）用 fontTools 讀：`’` `‘` `“` `”` `…` `·` 的字寬全部是 **1000／1000 em（全形）**；`'`（直的撇號）231、字母 `r` 339。
- 這個字型**沒有** `pwid`（比例寬）功能：GSUB 只有 `ccmp liga locl vert vrt2`；GPOS 有 `halt palt kern`。`palt`／`halt` 只把 `‘’“”·` 縮成 500（半形），**`…` 不在內**。→ CSS 的 `font-variant-east-asian: proportional-width`（＝`pwid`）在這個字型上沒有作用。
- 真瀏覽器實測（Chromium，16px，scratchpad `harness/punct.cjs`，打包後的 Vido CSS）：

  | 寫法 | `’` 寬 | `“`／`”` 寬 | `…` 寬 | `·` 寬 |
  | --- | --- | --- | --- | --- |
  | 現況（中文頁面） | 16 | 16 | 16 | 16 |
  | 只加 `lang="en"` | 16 | 16 | 17 | 16 |
  | `lang="en"`＋`palt`（或 `halt`） | **8** | 8 | 17 | 8 |
  | `lang="en"`＋Source Sans 3 引號子集（推薦） | **4.4** | 7.5 | 17 | 16 |
  | `lang="en"`＋系統字型 `local()` 引號子集 | 4.5 | 6.8 | 17 | 16 |
  | 整段改用系統字型（-apple-system） | 3.4 | 7 | 12.6 | 4.5 |

  並排截圖（scratchpad `harness/punct-compare.png`）：`palt` 之後 `Frieren’ s`、`didn’ t` **仍看得到空格**；Source Sans 3 那行看起來跟一般英文排版一樣。`…` 在每一種寫法裡看起來都正常（英文字型的 `…` 本來就接近 1em，Source Sans 3 也是 0.924em）——**不用處理**。

### 哪裡會出現英文長文字

| 位置 | 程式碼 | 什麼時候是英文 |
| --- | --- | --- |
| 本機片的「簡介」 | `LocalDetailV2.tsx:335` | TMDb 沒有繁中／簡中簡介時，後端的語言後備 `zh-TW → zh-CN → en`（`apps/api/internal/tmdb/fallback.go:10`；逐欄位補，`:257-283`）會寫入英文簡介 |
| TMDb 詳細頁的「簡介」 | `TMDbDetailV2.tsx:124` | 同上（探索、搜尋、推薦點進去的非片庫影片） |
| 片名 `h1` | `DetailHeroV2.tsx:135-137` | TMDb 沒有中文片名時是英文片名；比對失敗時是整個檔名 |
| 原文片名 | `DetailHeroV2.tsx:138-142` | 英語片的原文片名（例 `The Roundup: Punishment` 的原文是韓文，`Mag Mag` 是中文——原文片名不一定是英文） |
| TMDb 標示文字 | `TmdbAttribution.tsx:120` | 固定英文，**已經標了 `lang="en"`**，內容沒有彎引號 |

- 不算在內：檔名、路徑是 `font-mono`（JetBrains Mono，引號本來就是等寬字的寬度）；`SearchResultCard.tsx:119`（手動搜尋結果卡的簡介）只被 `ParseFailureCard` 用，而 `ParseFailureCard` 只出現在測試路由 `routes/test/manual-search.tsx:20`——正式畫面看不到；dev 回報的 `Frieren: Beyond Journey’s End` 就是這個測試用畫廊夾具（`-gallery.fixtures.tsx:2590`，`manual-search-search-results-grid`）。影集的分集清單沒有簡介。演員角色名太短、不處理。
- **正式機今天的實際狀況**（2026-10-19 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：
  - 電影 44 部有簡介，**2 部是英文**（`Mag Mag`、`The Roundup: Punishment`），兩段都只用直的 `'`，**沒有彎引號**；影集 81 部，0 部英文簡介。片名、原文片名含 `’` `“`：0 筆。
  - 反過來，**中文簡介大量使用 `“” …`**：電影 11 部、影集 10 部，例「象竟是“天字第一號逃犯”知世郎……」「監視他們的，遠不止現場的攝影機…」「“教父”維托·唐·柯里昂」——這些在 Noto 是全形，**是對的、要保持**。
  - `original_language` 欄位 55 部全部是空的 → 不能靠它判斷原文片名是什麼語言（TMDb 詳細頁的 `MovieDetails.originalLanguage` 才有值，`types/tmdb.ts:120` 起）。
  - → 問題是真的，但今天主要會出現在 TMDb 即時資料（探索／搜尋／推薦點進去的詳細頁）的英文簡介，片庫裡暫時 0 筆。P3 合理。

### 可行做法比較

| | 做法 | 要不要多打包字型 | 對中文標點 | 對 `·` 分隔點（設計稿） | 英文引號效果 | 其他代價 |
| --- | --- | --- | --- | --- | --- | --- |
| A | 全站 `--font-sans` 最前面疊一個只含 `U+2018-201F` 的拉丁字型（不分語言） | 是（同 B1） | **會壞**：中文簡介的“天字第一號逃犯”變窄，跟設計稿與台灣排版不一致（正式機 21 段中文簡介受影響） | 不受影響（不含 `U+00B7`） | 好 | 不建議 |
| **B1（推薦）** | 英文段落標 `lang="en"`，CSS 只對「有 `lang` 且不是中日韓」的元素在字型堆疊最前面加一個 **Source Sans 3 引號子集**（`unicode-range: U+2018-201F`，`size-adjust: 110%`） | 是：`@fontsource-variable/source-sans-3@5.3.0` 的 latin woff2 **28,740 bytes**（1 個檔）；**只有畫面上真的出現英文引號時才會下載** | 不受影響（中文元素沒有 `lang`，堆疊不含這個字型） | 不受影響（範圍不含 `U+00B7`，`…` 也不含） | 好（`’` 4.4px／16px，跟一般英文排版一致） | 字母仍是 Noto（設計稿的字），只換 4 個引號；Source Sans 3 就是 Noto Sans TC 拉丁字母的同一家族（Source Han Sans 的拉丁字母取自 Source Sans；同 `upm` 下 Noto 的拉丁字寬約是 Source Sans 3 的 1.10 倍，`a` 536／486、`n` 580／526，所以 `size-adjust: 110%`）；要附 OFL 授權檔 |
| B2 | 同 B1，但子集用 `src: local('Helvetica Neue'), local('Arial'), local('Roboto'), local('Segoe UI')` | 否（0 byte） | 不受影響 | 不受影響 | Mac／Windows／Android 好；**Linux（CI）沒有這些字型 → 退回 Noto 全形**，CI 證明不了；每台機器的引號長相不同 | 違反字型單「離線 NAS 也跟設計稿一樣、不靠系統字」的方向 |
| B3 | 英文段落整段改用系統字型或 Noto Sans（拉丁） | Noto Sans latin 35,820 bytes；系統字 0 | 不受影響 | 英文段落裡的 `·` 會變窄（罕見） | 好 | **字母也換字型**，跟 DESIGN.md:336「英數字也由它（Noto Sans TC）負責」衝突；同一頁中英段落字母長得不一樣 |
| C | 英文段落 `lang="en"`＋`font-feature-settings: 'palt'` | 否 | 不受影響（只套英文段落） | 英文段落裡的 `·` 也被縮成半形（罕見） | **只改善一半**：`’` 16→8px，`Frieren’ s` 仍有空隙（截圖）；`…` 不變 | 字型單寫明不開 `palt`（雖然那句是怕中文標點被改，這裡不會） |
| D | 顯示前把 `’` 換成 `'` | 否 | 不受影響 | 不受影響 | 只修 `’`，`“”` 不修 | 改了資料內容（複製、搜尋拿到的字不一樣），不建議 |

**推薦 B1。** 理由：唯一一個「英文引號真的變正常、中文標點與 `·` 一個像素都不動、離線 NAS 與 CI 都一致」的做法；代價是 28.7 KB、只在需要時下載。C 是零成本的退路，但截圖上空隙還在，修不乾淨。

### 其他已確認的事

- Vite 會解析 CSS `url()` 裡的套件路徑：scratchpad 用 repo 的 Vite 7.3.1 實測 `src: url('@fontsource-variable/…/files/…woff2')`，建置後輸出到 `assets/` 並改寫網址（沒有動到 repo）。⚠️ 未查證（dev 建置後確認）：在 repo 裡走過 `@tailwindcss/vite` 之後結果相同。若不同，退路是在 `main.tsx` 用 `import url from '…woff2?url'`＋`new FontFace(…)` 註冊（`unicodeRange`、`sizeAdjust` 兩個描述子都要帶）。
- `size-adjust` 描述子：Chrome／Edge 92+、Firefox 92+、Safari 17+；舊 Safari 會忽略它，引號只是小 10%，不會壞。
- 字型授權：Source Sans 3 名稱表寫「© 2023 Adobe, with Reserved Font Name ‘Source’」、授權欄空白 → 跟 Noto 一樣要另附授權檔（OFL 條件 2）。直接用套件裡沒改過的 woff2，不涉及「修改版」。
- 現有的守門：`apps/web/src/styles-fonts.spec.ts`（讀 `styles.css` 原始碼斷言字型堆疊，`:43-118`）、`tests/e2e/font-sans.spec.ts`（chromium，媒體庫頁所有字型請求都來自本機、Noto 預算 2,000,000 bytes）。媒體庫頁沒有英文引號 → 不會多下載這個檔，既有預算斷言不受影響。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要 TMDb 給的英文簡介裡 `Frieren’s`、`“I didn’t know”` 這種引號跟一般英文書一樣緊貼著字，
這樣讀英文簡介時不會一直卡在「這裡怎麼多一格空白」，而中文簡介的“引號”和……照舊是全形、跟設計稿一樣。

## Acceptance Criteria

1. **打包引號子集（B1，Sally 追認後）。** 根目錄 `package.json` 加 `@fontsource-variable/source-sans-3` 鎖 `5.3.0`（跟 `jetbrains-mono`／`noto-sans-tc` 同一批 2026-07-19 發布；`5.3.1` 是 2026-10-11，太新）。**不 import 套件的 `wght.css`**（那會讓整段英文改用 Source Sans 3 的字母）；改在 `styles.css` 自己寫一條 `@font-face`：
   - `font-family: 'Vido Latin Quotes'`、`src: url('@fontsource-variable/source-sans-3/files/source-sans-3-latin-wght-normal.woff2') format('woff2-variations')`、`font-weight: 200 900`、`font-display: swap`、**`unicode-range: U+2018-201F`**（`‘’‚‛“”„‟` 八個，**不含** `U+2026 …`、`U+00B7 ·`）、`size-adjust: 110%`。
   - 建置後 `dist/apps/web/assets/` 多 1 個 `source-sans-3-latin-wght-normal-<hash>.woff2`（28,740 bytes），沒有其他 Source Sans 3 檔。
2. **CSS 規則只套「標了非中日韓語言」的元素。** 在 `@layer base`（`styles.css:546` 起）加：
   ```css
   [lang]:not(:lang(zh)):not(:lang(ja)):not(:lang(ko)) {
     font-family: 'Vido Latin Quotes', var(--font-sans);
   }
   ```
   放在 `@layer base` 是刻意的：`font-mono` 等工具類在後面的 utilities 層，一定贏，等寬字不受影響。`<html lang="zh-Hant-TW">` 不會中。`TmdbAttribution.tsx:120` 的 `lang="en"` 段落會中（內容沒有彎引號，畫面不變）。
3. **英文文字標 `lang="en"`。** 新增 `apps/web/src/utils/textLang.ts`：`latinTextLang(text?: string | null): 'en' | undefined`——文字裡有拉丁字母、**而且沒有任何**漢字／假名／諺文／注音／全形標點（`U+3000-303F`、`U+FF00-FFEF`）時回 `'en'`，其他（含空字串、純數字）回 `undefined`（沿用頁面的 `zh-Hant-TW`）。用在：
   - `LocalDetailV2.tsx:335`、`TMDbDetailV2.tsx:124` 的簡介 `<p lang={latinTextLang(overview)}>`；
   - `DetailHeroV2.tsx:135` 的 `h1` 與 `:139` 的原文片名 `<p>`。
   - 不用 `originalLanguage` 當提示：正式機 55 部電影這欄全空；法文等其他拉丁語系標成 `en` 對讀螢幕軟體仍比今天的 `zh-Hant-TW` 好。
4. **可觀察的結果（真瀏覽器，16px 內文）**：
   - 英文簡介裡的 `’` 寬度 **< 6px**（今天 16px）；`“` `”` **< 9px**。
   - 中文簡介（例「象竟是“天字第一號逃犯”知世郎……」）的 `“` `”` `…` 仍是 **16px ± 0.5**（全形，跟今天一樣）。
   - 詳細頁 meta 列「2016 · 107 分」的 `·` 仍是 **1em**（設計稿的寬度，`a4p-m.png`／`b3p-m.png`）。
   - 英文段落的字母仍是 Noto Sans TC（`getComputedStyle` 的堆疊第二個是 `"Noto Sans TC Variable"`；字母 `r` 的寬度跟沒有 `lang` 時相同）。
   - 只開中文頁面時**不會**下載 Source Sans 3 的 woff2；開有英文引號的頁面時下載 1 次、來自本機、`font/woff2`、200。
5. **單元測試。**
   - `utils/textLang.spec.ts`：`'Frieren’s journey begins…'`→`'en'`；`'The Roundup: Punishment'`→`'en'`；`'象竟是“天字第一號逃犯”知世郎'`→`undefined`；`'君の名は。'`、`'범죄도시 4'`→`undefined`；`'[长安的荔枝].The.Lychee.Road.2025…mp4'`（正式機真實檔名，含漢字）→`undefined`；`'2016'`、`''`、`undefined`→`undefined`；`'Ｗｉｄｅ'`（全形英文）→`undefined`。
   - `styles-fonts.spec.ts` 加一組：`@font-face` 的 `font-family` 是 `'Vido Latin Quotes'`、`unicode-range` **正好是** `U+2018-201F`（多一個 `U+00B7` 或 `U+2026` 就紅）、有 `size-adjust: 110%`；`@layer base` 裡有 AC #2 的選擇器、堆疊是 `'Vido Latin Quotes', var(--font-sans)`；`main.tsx` **沒有** import `@fontsource-variable/source-sans-3`（防有人順手把整套字母換掉）。
   - `DetailHeroV2.spec.tsx`、`LocalDetailV2.spec.tsx`：英文片名／原文片名／簡介帶 `lang="en"`，中文的沒有 `lang` 屬性（斷言具體值，Rule 16）。
6. **E2E（chromium，新檔 `tests/e2e/font-latin-quotes.spec.ts`，寫法參考 `tests/e2e/font-sans.spec.ts`）。** 用 `seedMovie` 建兩部（`overview` 欄位 helper 已支援）：
   - 英文：`overview` 用一段**從 TMDb 拿的真實英文簡介、含 `’` 或 `“”`**（Rule 28；⚠️ 未查證哪一部有——dev 用 `GET /3/movie/{id}?language=en-US` 找一段，貼進測試並註明來源 id）；
   - 中文：`overview` 用正式機那段「…象竟是“天字第一號逃犯”知世郎……」（鏢人：風起大漠）。
   - 用 `Range` 量單一字元寬度，斷言 AC #4 的每一條；記錄字型請求，斷言 AC #4 最後一條。
   - **故意改壞會變紅**（dev 實際做、寫進 Completion Notes）：(a) 拿掉 AC #2 的 CSS 規則 → 英文 `’` 回到 16px，紅；(b) 留規則、拿掉簡介的 `lang` → 紅；(c) 把 `unicode-range` 改成含 `U+2026` → `styles-fonts.spec.ts` 紅。
7. **授權與文件。**
   - `apps/web/public/licenses/SourceSans3-OFL.txt`（套件的 `LICENSE` 原文，4,314 bytes）、`docs/licenses/source-sans-3.md`（照 `docs/licenses/noto-sans-tc.md` 的格式：套件、版本、用途「只畫英文段落裡的 ‘’“” 等 8 個引號」、Reserved Font Name ‘Source’、檔案沒有修改）。
   - **DESIGN.md §Typography（`:336` 那行之後）加一句**（Sally 定稿字句）：英文段落（標 `lang="en"`）裡的 `‘’“”` 由 Source Sans 3 畫，因為 Noto Sans TC 的這幾個字是全形；中文段落的引號、`…`、`·` 仍是 Noto 的全形。
8. **視覺基準：預期 0 張變化。** 現有畫廊夾具沒有一個會被標成 `lang="en"` 又含彎引號（`media-detail-hero-v2` 是「你的名字／君の名は。」；含 `Journey’s` 的 `manual-search-search-results-grid` 用的是 `SearchResultsGrid`，本單不碰它）。跑一次完整 `test:visual`（darwin），若有任何基準圖變了，停下來說明是哪張、為什麼；**不要**為了這張新增夾具（理由見 Dev Notes）。
9. `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check` 綠；`font-sans.spec.ts`、`font-mono.spec.ts` 仍綠。

## Tasks / Subtasks

- [ ] **T0 Sally 追認**：把「可行做法比較」表與 `harness/punct-compare.png` 的結論給 Sally；記錄她的裁定（選 B1／改選別的）與 DESIGN.md 那句的定稿字句，寫進本檔最上面。
- [ ] **T1 字型與 CSS（AC #1、#2）**：裝 `@fontsource-variable/source-sans-3@5.3.0`；`styles.css` 寫 `@font-face`＋`@layer base` 規則，更新 `:521-533` 那段註解（說明這個子集為什麼存在、為什麼不含 `…`／`·`）；`pnpm nx build web` 後確認 `dist/apps/web/assets/` 多且只多 1 個 Source Sans 3 woff2。
- [ ] **T2 標語言（AC #3）**：`utils/textLang.ts`；接到 `DetailHeroV2`、`LocalDetailV2`、`TMDbDetailV2` 四處。
- [ ] **T3 測試（AC #5、#6）**：三個 vitest 檔＋新 E2E 檔；故意改壞三次確認會紅。
- [ ] **T4 授權與文件（AC #7）**。
- [ ] **T5 視覺基準確認＋Step 9（AC #8）**：完整 `test:visual` 一次；本機開一部英文簡介的片（例：正式機 `The Roundup: Punishment` 的形狀，簡介改一段含 `’` 的）與一部中文簡介的片，截 1440／390 對照，附在 PR 給 Sally。

## Dev Notes

### 不要做的事

- 不要 import `@fontsource-variable/source-sans-3/wght.css`：那條 `@font-face` 的範圍是整個 latin，英文段落的字母會全部換成 Source Sans 3（比 Noto 小 10%、跟設計稿不一樣）。
- 不要把子集放進全站的 `--font-sans`（做法 A）：正式機 21 段中文簡介的“引號”會變窄。
- 不要把 `U+2026 …` 或 `U+00B7 ·` 放進 `unicode-range`：`…` 在英文段落裡本來就正常；`·` 是設計稿的全形分隔點（字型單 Debug Log：14px 字級量到 14px，對照 `a4p-m.png`）。
- 不要開 `palt`／`halt`（做法 C）——除非 Sally 改選 C。
- 不要碰 `SearchResultCard`／`SearchResultsGrid`（正式畫面沒在用）。

### 已知陷阱

- `unicode-range` 是「畫到那個字才下載」：E2E 量寬度前要 `await page.evaluate(() => document.fonts.ready)`，必要時等 `document.fonts.check('16px "Vido Latin Quotes"', '’')` 為真再量。
- `[lang]:not(:lang(zh))…` 只會選到**帶 `lang` 屬性的那個元素**，子元素靠繼承拿到 `font-family`；如果英文段落裡有 `font-mono` 的子元素，工具類照樣贏（utilities 層在後）。
- `font-display: swap`：字型還沒到的那一瞬間，引號先用 Noto 全形畫、到了再縮回去——只發生在第一次看到英文引號時，之後一年快取。
- 字型 PR 已經處理好 woff2 不 gzip、`font/woff2` 型別、一年快取（`apps/api/cmd/api/static.go`），本單不用動後端。

### 為什麼不加畫廊夾具

- 正式機片名／原文片名含彎引號的是 0 筆，TMDb 片名幾乎都用直的 `'`；做一個「英文片名含 `’`」的夾具不是真實形狀（Rule 28）。簡介區塊在 `LocalDetailV2`／`TMDbDetailV2` 裡，不是獨立元件，沒有現成夾具。證據改由 AC #6 的 E2E 量字寬＋T5 的截圖負責。

### 對 data-testid／E2E／視覺基準的影響

- **data-testid**：不新增、不改名、不刪除。
- **E2E**：新增 `font-latin-quotes.spec.ts`；既有 `font-sans.spec.ts`（媒體庫頁只抓本機字型、Noto 預算）不受影響——該頁沒有英文引號，不會下載新檔。
- **視覺基準**：預期 0 張變化（AC #8）。

### 測試標準

- vitest＋Testing Library（Rule 16 具體斷言）；字寬與字型下載只能在真瀏覽器證明（Rule 28「Browser layout」列），所以 E2E 是主要證據。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`。

### Project Structure Notes

- 改：`package.json`／`pnpm-lock.yaml`、`apps/web/src/styles.css`、`apps/web/src/utils/textLang.ts`（新，＋spec）、`components/media/DetailHeroV2.tsx`、`LocalDetailV2.tsx`、`TMDbDetailV2.tsx`（＋各自 spec）、`styles-fonts.spec.ts`、`tests/e2e/font-latin-quotes.spec.ts`（新）、`apps/web/public/licenses/SourceSans3-OFL.txt`（新）、`docs/licenses/source-sans-3.md`（新）、`DESIGN.md`。沒有後端改動。
- **Story Splitting Rule**：只有前端，5 個 task（其中 T0 是決策），不拆。
- **跟 `disc-2026-10-detail-mobile-long-title-overflow` 的關係**：兩張都改 `DetailHeroV2.tsx` 的 `h1` 與原文片名那兩行（那張加 `break-words`，這張加 `lang`）。那張先做；這張開工時 rebase 到 main，兩邊的 class 與屬性都要留著。

### Time-dependent visual coverage

- N/A — 不碰任何讀時間的元件。

### References

- [Source: node_modules/@fontsource-variable/noto-sans-tc/index.css:1041-1049（latin 區塊）、files/noto-sans-tc-latin-wght-normal.woff2（fontTools：hmtx、GSUB／GPOS 功能表）]
- [Source: npm `@fontsource-variable/source-sans-3@5.3.0`（2026-07-19）— `npm pack` 到 scratchpad：files/source-sans-3-latin-wght-normal.woff2 28,740 bytes、LICENSE 4,314 bytes、名稱表版權列]
- [Source: apps/web/src/styles.css:521-543, 546]、[apps/web/src/main.tsx:5-6]、[apps/web/index.html（`lang="zh-Hant-TW"`）]
- [Source: apps/web/src/components/media/DetailHeroV2.tsx:17-29, 135-142]、[LocalDetailV2.tsx:308-317, 332-337]、[TMDbDetailV2.tsx:73, 93, 121-126]、[ui/TmdbAttribution.tsx:30-31, 120]
- [Source: apps/api/internal/tmdb/fallback.go:10, 257-283]
- [Source: apps/web/src/styles-fonts.spec.ts:43-118]、[tests/e2e/font-sans.spec.ts]、[tests/support/helpers/seed-helpers.ts:33-80]
- [Source: DESIGN.md:336-341]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-font-sans-not-noto.md — Dev Notes「不要開 font-feature-settings」、Debug Log「Noto 的 · 是全形寬」]
- [Source: _bmad-output/screenshots/flow-a-browse-v2/a4p-m.png、flow-b-detail-v2/b3p-m.png（全形 `·` 分隔點）]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **③ 建議 parent 決定要不要立案（Bob 建單時發現，P3）** — `components/manual-search/`（`ManualSearchDialog`、`SearchResultsGrid`、`SearchResultCard`…）與 `components/library/ParseFailureCard.tsx` 在正式程式碼裡只剩彼此互相 import，最外層的 `ParseFailureCard` 只被測試路由 `routes/test/manual-search.tsx:20` 用（`git grep` 2026-10-19）；正式的手動選片已經是 `ManualMatchDialogV2`。看起來是 `chore(v1-cleanup)`（#752）漏掉的舊元件，還各自有畫廊夾具與基準圖（例 `manual-search-search-results-grid`）。Bob 沒有查是否有 E2E 依賴測試路由 `/test/manual-search`；刪之前要先 grep `tests/`。

### File List
