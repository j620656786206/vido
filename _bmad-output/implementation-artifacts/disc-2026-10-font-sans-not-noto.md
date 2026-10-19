# Disc：把 Noto Sans TC 放進 Vido，畫面上的中文真的用設計稿那套字（離線 NAS 也一樣）

Status: ready-for-dev

**Source:** Rule 24 lane ③，2026-10-19 由 `disc-2026-10-font-mono-not-jetbrains` 建單時發現（見該單 Discovery Triage）。
⚖️ **Alexyu 2026-10-19 選 (a)：把 Noto Sans TC 放進 Vido，只放有用到的字重。** 不選 (b)「改 DESIGN.md 承認用系統中文字型、刪死規則」。

**跟等寬字那張單一起做（Bob 建議，見下方「跟 mono 單怎麼排」）：** 本單與 `disc-2026-10-font-mono-not-jetbrains` 在**同一條分支、同一個 PR** 完成——先做 mono 的程式碼、再做本單的程式碼、最後**只重產一次**視覺基準圖。兩張單都會讓幾乎每一張基準圖改變，分兩個 PR 就要重產兩次、CI 開兩次 `-linux` bootstrap PR、Alexyu 看兩輪「全部截圖都變了」。

**排程（不是硬性 blocked-by）：** 跟 mono 單一樣，排在會動基準圖的 `disc-2026-10-library-list-triage-b1`、`-b2`、`disc-2026-10-detail-resolution-raw` 都合併之後。開工前 `gh pr list` 確認沒有別的 PR 正在改 `tests/visual/**-snapshots/`。

## 查到的事（2026-10-19，本 session 讀過的程式碼與套件）

### 現況：中文根本沒用 Noto Sans TC

- `apps/web/src/styles.css:537-545` 的 `.font-primary { font-family: 'Noto Sans TC', -apple-system, … }` 在 `@layer base`（`:522` 開始），全站沒有元件用它（`grep -rn "font-primary" apps/web/src` 只命中 `styles.css:536-537` 自己）。
- `body` 的字型來自 Tailwind preflight：`html, :host`（`node_modules/tailwindcss/index.css:514-515`）的 `font-family: --theme(--default-font-family, …)`（`:519`），而 `--default-font-family: --theme(--font-sans, initial)`（`:455`）；`--font-sans` 是預設的 `ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"`（`:5-7`，Tailwind 4.1.18）。→ **改 `@theme` 的 `--font-sans` 就會換掉整站內文**，不用動任何元件。Mac 上今天實際是蘋方、Windows 是微軟正黑。
- `font-sans` 工具類只有一處：`components/settings/BackupTable.tsx:278`（把狀態膠囊從 `font-mono` 列裡拉回內文字型）——改完自動變 Noto，正是要的。
- 字型沒有載入：`apps/web/index.html:1-37` 沒有字型連結；`apps/web/src/main.tsx:1-16` 沒有字型 import；根目錄 `package.json` 沒有任何 `@fontsource`（`grep -n fontsource package.json pnpm-lock.yaml` 0 筆，mono 單也還沒實作）。

### 用到哪些字重（⚖️ 只放用到的）

| 字重 | 證據 | 用在哪 |
| ---- | ---- | ------ |
| **400** | 沒有 `font-*` 的文字全吃預設 400；`font-normal` 4 處（例 `CandidateListPanel.tsx:234`、`EpisodeList.tsx:282`）；`DESIGN.md:358-359` Body／Body Large 400；`.pen` `Type/Body/Weight`、`Type/BodyLg/Weight` = 400（`_bmad-output/pen-tokens.json:103-118`） | 全站內文、按鈕 |
| **500** | `font-medium` 264 處／125 檔；`text-xs` 預設 500（`styles.css:511` `--text-xs--font-weight: 500`）；`DESIGN.md:360` Label 500；`.pen` `Type/Label/Weight` = 500 | 海報片名（`PosterCardV2.tsx:176`）、列表片名（`LibraryListRowV2.tsx:93`）、所有徽章與殼層小字 |
| **600** | `font-semibold` 253 處／115 檔；`DESIGN.md:356-357` H3／H4 600；`.pen` `Type/H3/Weight`、`Type/H4/Weight` = 600 | 區塊與卡片標題 |
| **700** | `font-bold` 34 處／28 檔；preflight `b, strong { font-weight: bolder }`（`index.css:589-591`，`<strong>` 2 處：`RestoreConfirmDialog.tsx:127`、`routes/test/gallery.tsx:313`）；`<th>` 4 處（瀏覽器預設粗體）；`DESIGN.md:353-355` Display／H1／H2 700；`.pen` `Type/Display|H1|H2/Weight` = 700 | **每一頁的頁面大標**：`ActivityHub.tsx:396`「活動」、`DiscoverBrowseV2.tsx:177`「探索」、`routes/search.tsx:81`「搜尋媒體」、`DetailHeroV2.tsx:135` 詳細頁片名、`HeroBanner.tsx:209` 首頁 hero 片名、`SettingsPageHeader.tsx:35` |

- 100／200／300／800／900：0 處（`font-thin|extralight|light|extrabold|black` 全 0；任意值 `font-[…]` 0；CSS／inline `font-weight` 除 `styles.css:511` 外 0）。
- 量法：`grep -rhoE "font-<w>([^a-zA-Z0-9-]|$)" apps/web/src --include='*.tsx' --include='*.ts' --exclude='*.spec.*' --exclude='*.test.*'`。
- **最少用的是 700**（34 處），但它是每一頁的 H1／H2。併到 600 會讓頁面大標跟 H3／H4 卡片標題一樣粗，DESIGN.md 的層級（`:351-360`）就塌了——**不併**。而且下一段選的可變字型是「一組檔案涵蓋所有字重」，少放一個字重**一個 byte 都省不到**，所以沒有「為了省空間併字重」的問題。

### 套件比較（2026-10-19 `npm view`／`npm pack` 到 scratchpad 實測，不在 repo 裡）

| | `@fontsource-variable/noto-sans-tc@5.3.0`（**選這個**） | `@fontsource/noto-sans-tc@5.3.0`（靜態，只取 400–700） |
| --- | --- | --- |
| 發布 | 2026-07-19（跟 mono 單鎖的 `@fontsource-variable/jetbrains-mono@5.3.0` 同一批） | 同 |
| 授權 | OFL-1.1 | OFL-1.1 |
| 檔案 | `wght.css`：105 條 `@font-face`，`font-family: 'Noto Sans TC Variable'`、`font-weight: 100 900`、`font-display: swap`、各自 `unicode-range`；`files/` 105 個 woff2 | 每個字重一支 CSS（`400.css` …），每支 106 條 `@font-face`，**`src` 同時列 woff2 與 woff** |
| 放進映像的大小 | **105 個 woff2 共 4,194,768 bytes（4.2 MB）**，涵蓋 100–900 全部字重 | 400／500／600／700 的 woff2 共 13,146,508 bytes（424 檔）；因為 CSS 也引用 woff，Vite 會一起打包 → 再加 woff 16,994,524 bytes → **約 30 MB、848 檔** |
| 一頁 zh-TW 畫面實際下載（估算上限，見下） | 媒體庫（介面字＋60 部片名）**≈1.44 MB**；詳細頁（介面字＋一段簡介＋演員）**≈1.47 MB** | 同頁要分別下載每個字重用到的區塊：媒體庫 ≈2.1 MB、詳細頁 ≈2.4 MB（上限 3.1 MB） |
| 整個 App 所有介面字串 | 1,005 個不同字元、落在 25 個區塊、**1.49 MB**——逛過一輪幾乎全部進快取 | — |

- **估算方法**：從 `apps/web/src` 非測試檔抽出介面字串（去掉註解）＋60 部常見繁中片名＋一段芙莉蓮簡介，對照 `wght.css` 每條 `unicode-range` 算出會被下載的區塊、加總 woff2 大小（腳本在 scratchpad，不進 repo）。這是**上限**：有幾個數字區塊（例 `[119]`）的 range 也含 ASCII，但瀏覽器會先用後宣告的 `latin` 區塊（25,136 bytes），通常不會為了英數字去抓它們。實際數字由 AC #5／#10 在真瀏覽器量。
- **為什麼可變字型比較小**：可變字型一組檔案 4.2 MB 就有 100–900，比靜態「只取四個字重」的 woff2（13.1 MB）小三倍；而且一頁只需要抓一份字形、不必每個字重各抓一份。**這就是「只放用到的字重」最省的做法**——雖然檔案裡技術上含 100–900，但沒有任何選項能用更少的 bytes 提供 400–700。
- **試過把軸裁到 400–700**（fontTools `instancer`，scratchpad 實測）：4,194,768 → 3,906,768 bytes，只省 **7%（288 KB）**，代價是 repo 要多一道 Python 字型處理步驟、自己 commit 處理過的字型檔，而且依 OFL 這算「修改版」（字型名稱表寫著 `(c) 2014-2021 Adobe … with Reserved Font Name 'Source'`）。**不做。**
- **版本選 5.3.0、不選最新 5.3.1**：5.3.1 是 2026-10-11 發布（8 天前），字型版本 v40（5.3.0 是 v39），多 407 個罕用字（CJK 擴展 A，例 㐁㑁）、總大小 4,276,832 bytes。介面字串與 60 部片名在兩版都**沒有缺字**（缺的只有 🎬🔍🔴 這類 emoji，本來就交給系統 emoji 字型）。選 5.3.0：跟 mono 單同一批發布、已經放三個月；之後要升只改版本號。
- **三個極小區塊會被內嵌**：`files/` 有 3 個檔 <4,096 bytes（`[97]` 1,656、`[98]` 3,052、`[92]` 3,584），Vite 預設 `assetsInlineLimit` 4096 會把它們轉成 base64 塞進 CSS（`apps/web/vite.config.mts:46-53` 沒有覆寫這個設定）。⚠️ 未查證（dev 建置後確認）：`dist/apps/web/assets/` 的 Noto woff2 應該是 102 個，不是 105。
- **CSS 會變大**：`wght.css` 108,576 bytes（gzip 33,505）＋3 個內嵌區塊約 11 KB base64；mono 單記錄的 2026-10-14 建置 CSS 是 105,954 bytes（gzip 18,150）→ 預期 CSS 大約翻倍。這份 CSS 會擋住首次繪製，但走區網、而且一年快取，可以接受；數字要在 AC #10 量。
- **授權檔**：套件 `LICENSE`（4,314 bytes）開頭 `Google Inc.` 接 OFL-1.1 全文；woff2 名稱表的版權是 Adobe（見上），授權欄（name ID 13）是空的——**字型檔本身沒帶授權文字，所以一定要另外附授權檔**（OFL 條件 2）。

### 字型堆疊順序（英數字誰來畫）

- **英數字也用 Noto Sans TC。** `DESIGN.md:336`「內文字體：Noto Sans TC（後備 `-apple-system` …）」把 Noto 放第一位；`:339`「一套人文主義無襯線扛下所有閱讀工作」；`DESIGN.md:40`、`:45` …（frontmatter 的 12 個 `typography.*.fontFamily`）全部 `Noto Sans TC, -apple-system, …`；`.pen` `Type/Family/Primary` = `Noto Sans TC`（`pen-tokens.json:152`）。設計稿上的英文本來就是 Noto Sans TC 畫的——如果 Latin 先走系統字，實作反而跟設計稿不一樣。Noto 的 `latin` 區塊只有 25 KB。
- **後備加 `PingFang TC`、`Microsoft JhengHei`**：字型還沒下載完（`swap` 的那一瞬間）或遇到 Noto 沒有的罕用字時，中文要落到**繁體**系統字。`index.html:2` 是 `<html lang="en">`，只寫 `-apple-system` 的話，中文落到哪套系統字是瀏覽器依語系猜的。明寫 `PingFang TC`（Mac／iOS）、`Microsoft JhengHei`（Windows）放在 `sans-serif` 前面，就不靠猜。
- **emoji 保留**：Tailwind 預設尾端的四個 emoji 字型照留（介面字串裡有 🎬🔍🔴）。
- **等寬字照舊 JetBrains Mono**（mono 單）；但路徑、檔名裡的中文（例：`/media/movies/你的名字 (2016)/…`）JetBrains Mono 沒有，`DESIGN.md:341`「CJK 一律走 Noto Sans TC」→ 在 `--font-mono` 的 `monospace` 前面補 `'Noto Sans TC Variable'`（AC #3）。SF Mono／Menlo／Consolas 都沒有中文，所以中文會一路落到 Noto。

### `font-display` 與畫面跳動

- 套件寫死 `font-display: swap`（`wght.css` 每一條）。第一次開某頁：先用系統字畫、Noto 區塊到了再換。
- **媒體庫格狀檢視不會跳**：海報片名是 `line-clamp-2 min-h-[2.75em]`（`PosterCardV2.tsx:176`），高度固定兩行，換字只換字形、不改卡片高度。列表片名是 `truncate` 單行（`LibraryListRowV2.tsx:93`）。
- **詳細頁可能跳一次**：片名 H1（`DetailHeroV2.tsx:135`）與簡介段落換字後斷行可能不同，下面的內容會移一次。只發生在「這台裝置第一次看到這幾個字」——之後一年快取（`static.go:17` `cacheImmutable`）。行高全部是比例值（`styles.css:509-519`），所以字型本身的 ascent／descent 不會改變行框高度。
- **不改成 `optional`**：第一次載入可能整次不用 Noto，同一頁一下 Noto 一下蘋方。**不改成 `block`**：中文最多 3 秒是空白。改 `font-display` 也得自己寫 `@font-face`（違反「用套件」）。
- **不 preload**：要 preload 的是哪幾個區塊取決於頁面上有哪些字，而且檔名有 Vite hash，得寫 Vite 外掛；區網載入本來就快、之後全走快取，划不來。

### 離線 NAS 與伺服器

- Docker：`pnpm install --frozen-lockfile`（`Dockerfile:31`）→ `nx build web`（`:41`）→ `dist/apps/web` 複製到 `/app/public`（`:119`）。執行映像 `alpine:3.21`、只裝 `ca-certificates tzdata ffmpeg su-exec`（`:97-100`）。**執行時不連外**。
- Go：`router.Static("/assets", …)`＋`Cache-Control: public, max-age=31536000, immutable`（`apps/api/cmd/api/static.go:17`、`:42-46`、`:64-70`）；`X-Content-Type-Options: nosniff`（`static.go` `securityHeadersMiddleware`）。Go 內建 MIME 表沒有 `.woff2`（`$(go env GOROOT)/src/mime/type.go:116` 只到 `.wasm`），alpine 沒 `/etc/mime.types` → 走內容嗅探，`wOF2` → `font/woff2`（`$(go env GOROOT)/src/net/http/sniff.go:178`，go1.26.3）。同 mono 單結論。
- **woff2 被 gzip 了一次（新發現，本單 AC #6 一起修）**：`apps/api/cmd/api/main.go:1413-1415` `router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths(…)))`；`gin-contrib/gzip@v1.2.5` 預設只排除 `.png .gif .jpeg .jpg`（`options.go:18-20`）。woff2 本身已經是 Brotli 壓縮——實測 `[115]` 區塊 74,332 bytes，gzip 後 74,410（**反而大 78 bytes**），白白讓 NAS 的 CPU 每次首訪壓 1.5 MB。注意 `WithExcludedExtensions` 是**取代**預設清單（`options.go:55-59`），不是追加。
- `/licenses/*` 不會被 HTTP 供應：`registerStaticRoutes` 只開 `/assets` 與根目錄三個固定檔名（`static.go:42-57`），其他路徑走 SPA fallback 回 `index.html`（`static.go:75-102`）。授權檔放在映像的 `/app/public/licenses/` 是為了「字型的每一份拷貝附授權」（映像就是那份拷貝），不是要給瀏覽器看——跟 mono 單 AC #7 一樣。

### 視覺測試

- 截圖前有 `await page.evaluate(() => document.fonts.ready)`（`tests/visual/components.visual.spec.ts:244-245`），在畫廊區塊可見之後（`:243`）。unicode-range 是**看到字才下載**：同一個區塊裡之後才出現的狀態（`:247-250` 逐一截每個 `data-gallery-state`）如果帶出新字，那個區塊會晚一點才到。`toHaveScreenshot` 會重拍到兩張一樣才比對，通常會自己穩住，但要實測（AC #11 跑兩次）。
- 門檻 `maxDiffPixelRatio: 0.001`（`playwright.config.ts:62-63`）；`visual` 專案 1280×800、深色（`:163-166`）；本機 `webServer` 是 `npx nx serve web`（`playwright.config.ts:240-245`），字型由 Vite 從 `node_modules` 供應，CI 另行啟動（`:224-226`）。
- 基準圖：`tests/visual` 下 `-darwin.png` 357 張、`-linux.png` 357 張（`find tests/visual -name '*-darwin.png' | wc -l`）。換內文字型 → **幾乎每一張都會變**。CI 用版本固定的 `ubuntu-24.04`，因為 Linux 的字型渲染決定基準像素（`.github/workflows/visual-regression.yml:40-42`）；改完之後 linux 與 darwin 都用同一份 woff2，兩邊會更接近。

⚖️ **Alexyu 2026-10-19 另裁定三件：** ① 用可變字型（`@fontsource-variable/noto-sans-tc`，4.2 MB 一組涵蓋全部字重；比只取四個字重的靜態版 30 MB 省）；② 跟 `disc-2026-10-font-mono-not-jetbrains` 同一條分支、同一個 PR，基準圖只重產一次；③ `<html lang>` 一起改成繁中（AC #13）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要 Vido 的中文（和內文裡的英數字）用設計稿指定的 Noto Sans TC 顯示，不管我用 Mac、Windows 還是手機、NAS 有沒有連網，
這樣畫面跟設計稿長得一樣，每台裝置的字也一致，不會這台蘋方、那台微軟正黑。

## Acceptance Criteria

1. **字型隨 Vido 打包，不連外。** 根目錄 `package.json` 的 `dependencies` 加 `"@fontsource-variable/noto-sans-tc": "5.3.0"`（**精確版本，不是 `^`**），`pnpm-lock.yaml` 跟著更新；`apps/web/src/main.tsx` 最上面（跟 mono 單的 import 放一起）`import '@fontsource-variable/noto-sans-tc/wght.css';`。**不裝**靜態版 `@fontsource/noto-sans-tc`。`nx build web` 後 `dist/apps/web/assets/` 有 `noto-sans-tc-*-wght-normal-<hash>.woff2`；建置後的 CSS 與 JS 裡沒有 `fonts.googleapis`、`fonts.gstatic`、`cdn.jsdelivr`、`unpkg`（`grep` 證明）。
2. **整站內文是 Noto Sans TC。** `styles.css` 的 `@theme` 加：
   ```css
   --font-sans: 'Noto Sans TC Variable', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang TC', 'Microsoft JhengHei', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';
   ```
   （一般 `@theme`，可跟 mono 單的 `--font-mono` 放在同一個區塊。）**刪掉** `@layer base` 的死規則 `.font-primary { … }`（`styles.css:537-545`），並把 `:536` 的註解改成只講 `font-mono`／`font-sans` 由 `@theme` 決定。建置後的 CSS：`--font-sans` 以 `'Noto Sans TC Variable'`（或壓縮後的 `Noto Sans TC Variable`）開頭；全檔沒有 `.font-primary`。
3. **等寬字裡的中文也走 Noto。** mono 單 AC #2 的 `--font-mono` 在 `monospace` 前面補 `'Noto Sans TC Variable'`：`'JetBrains Mono Variable', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Noto Sans TC Variable', monospace`。（mono 的程式碼必須已經在同一條分支上；見 Dev Notes「跟 mono 單怎麼排」。）
4. **CSS 守門測試。** 跟 mono 單 AC #4 同一個 spec 檔（建議 `apps/web/src/styles-fonts.spec.ts`，照 `styles-motion.spec.ts:13` 讀 `styles.css`）加斷言：`@theme` 的 `--font-sans` 第一個是 `'Noto Sans TC Variable'`、清單裡有 `'PingFang TC'` 與 `'Microsoft JhengHei'` 且都在 `sans-serif` 前面；`--font-mono` 含 `'Noto Sans TC Variable'` 且在 `monospace` 前面；`styles.css` 沒有 `.font-primary`；`main.tsx` import 了 `@fontsource-variable/noto-sans-tc/wght.css`。
5. **瀏覽器實測（Rule 28「Browser layout」列）。** 新 `tests/e2e/font-sans.spec.ts`（chromium）：打開 `/library`（用 `tests/support/helpers/library-stubs.ts` 的 stub，片名要有中文），`await document.fonts.ready` 之後：
   - `getComputedStyle(document.body).fontFamily` 以 `"Noto Sans TC Variable"` 開頭；
   - `[...document.fonts]` 裡 `family` 為 `Noto Sans TC Variable`（注意引號可能被保留，比對前去掉）且 `status === 'loaded'` 的至少 1 個；
   - 用 `page.on('response')` 收集 `resourceType() === 'font'`：**每一個**網址的主機都是本機（`localhost`／`127.0.0.1`），至少一個網址含 `noto-sans-tc-`，而且這次頁面載入抓的 Noto 字型總大小 **< 2,000,000 bytes**（測試把實際數字 `console.log` 出來，貼進 Completion Notes）；
   - ⚠️ 不要用 `document.fonts.check()` 當主斷言：字型族名不存在時它也可能回 `true`（行為依瀏覽器版本，未查證），會變成空洞通過。
6. **woff2 不再被 gzip 一次。** `apps/api/cmd/api/main.go:1413` 的 `gzip.Gzip(…)` 加 `gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".woff2"})`——**要把原本四個預設副檔名一起寫上**（這個選項是取代、不是追加，`gin-contrib/gzip@v1.2.5/options.go:55-59`）。驗收看 AC #7 的第二個 `curl`。
7. **Docker 映像實測（離線 NAS）。** 照 mono 單 AC #6 的方式起拋棄式容器（`-e AI_PROVIDER=claude -e CLAUDE_API_KEY=sk-ant-dummy`），挑一個 Noto 區塊：
   - `curl -sI http://localhost:8099/assets/<noto 區塊檔名>` → `200`、`Content-Type: font/woff2`、`Cache-Control: public, max-age=31536000, immutable`；
   - `curl -s -D - -o /dev/null -H 'Accept-Encoding: gzip' http://localhost:8099/assets/<同一檔>` → **沒有** `Content-Encoding: gzip`；同樣打 `/assets/index-<hash>.css` → **有** `Content-Encoding: gzip`（證明沒把 CSS／JS 的壓縮一起關掉）；
   - `docker image ls` 改動前後映像大小，差距約 +4.2 MB（Noto）＋約 0.1 MB（mono）。
   Completion Notes 貼這幾行。本機 Docker 不能用時，改用 `VIDO_PUBLIC_DIR=dist/apps/web go run ./cmd/api` 打同樣的請求並說明原因。
8. **授權跟著走。**
   - `apps/web/public/licenses/NotoSansTC-OFL.txt`：套件 `LICENSE` 全文原樣複製（Vite 原樣放進 `dist/apps/web/licenses/`，映像裡就在 `/app/public/licenses/`）。
   - 新 `docs/licenses/noto-sans-tc.md`（照 `docs/licenses/opencc.md` 的格式、zh-TW）：套件名與版本 `@fontsource-variable/noto-sans-tc@5.3.0`、授權 OFL-1.1、字型版權 `(c) 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'`（取自字型名稱表）、出處 <https://github.com/notofonts/noto-cjk> 與 Google Fonts、映像內位置（`/app/public/assets/noto-sans-tc-*.woff2`、`/app/public/licenses/NotoSansTC-OFL.txt`）、「未修改字型檔（沒有自己裁字、裁字重）、不單獨販售字型」。
9. **DESIGN.md 跟程式碼一致。** `DESIGN.md:336` 改成：「**內文字體：** Noto Sans TC（隨 Vido 打包：`@fontsource-variable/noto-sans-tc`，可變字重，只會用到 400／500／600／700，不連外；英數字也由它負責；後備 `-apple-system`、`BlinkMacSystemFont`、`Segoe UI`、`Roboto`、`PingFang TC`、`Microsoft JhengHei`、sans-serif）」。`:341` 不用改（本單就是在實現它）。frontmatter 的 `typography.*.fontFamily`（`:40` 起）不動——那是設計字族，不是 CSS 堆疊。
10. **大小有量。** Completion Notes 一張表，同一台機器、改動前後各 build 一次（mono＋sans 都改完算「後」，另外記一欄「只有 mono」若方便）：`dist/apps/web/assets/` 的 CSS 原始／gzip、JS、woff2 檔數與總大小；AC #5 量到的「媒體庫首次載入抓的 Noto 字型大小」。預期：Noto woff2 約 102 個、約 4.2 MB；CSS 約 +120 KB（gzip 約 +40 KB）；媒體庫首次載入 Noto 約 0.2–1.5 MB（依 stub 片名多寡）。
11. **視覺基準圖全部重產——跟 mono 單合併成一次。** 等 mono 與本單的程式碼都在分支上之後才做（mono 單 AC #10 的六步**只執行一次**，兩張單共用結果）：
    1. `AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-visual-dummy pnpm run test:visual --reporter=line`，記下所有失敗的夾具／狀態。
    2. 每一張失敗的：刪 `-darwin.png`、`git rm` 對應的 `-linux.png`。
    3. `pnpm run test:visual:update-missing` 產新的 darwin（會 exit 1 並印 "writing actual"，正常）。
    4. **再跑兩次** `test:visual`，兩次都全綠（確認 unicode-range 晚到的區塊不會讓截圖忽好忽壞；若有忽好忽壞的，在 `components.visual.spec.ts:250` 的狀態迴圈裡、每次截圖前再 `await page.evaluate(() => document.fonts.ready)`，並寫進 Completion Notes）。
    5. 隨機打開至少 5 張新的 darwin 圖、其中至少 2 張有頁面大標（700）與海報片名（500），跟 git 裡的舊圖並排，肉眼確認中文字形確實換了、粗細層級看得出來（大標 700 ＞ 卡片標題 600 ＞ 片名 500 ＞ 內文 400），Completion Notes 列出看了哪幾張。
    6. 不要自己產 `-linux.png`、不要跑 `test:visual:update`；CI 會開 `chore(visual): bootstrap N missing -linux baselines` PR，核對檔案清單＝第 2 步刪的那批之後合併（這種 PR 沒有 checks，不用輪詢）。
    7. Completion Notes 寫受影響張數（darwin 改幾張、linux 刪幾張，兩數相等）。
13. **網頁語言改成繁中（⚖️ Alexyu 2026-10-19「一起改」，lane ①）。** `apps/web/index.html:2` 從 `<html lang="en">` 改成 `<html lang="zh-Hant-TW">`；`TmdbAttribution.tsx:120` 的 `lang="en"` 段落保留。vitest 或 E2E 斷言 `document.documentElement.lang === 'zh-Hant-TW'`。基準圖變化併進 AC #11 那一次重產，不另跑。
14. **沒有別的東西壞掉。** `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check`、`check-design-refs.py`、Go：`go test ./...` 與 `~/go/bin/staticcheck ./...`（AC #6 動了 Go）都綠；**全套 E2E** 本機跑一次——Noto 的英數字寬度跟系統字不同（⚠️ 未查證寬窄），量寬度、`truncate`、390 寬不能橫向捲動的 E2E 可能受影響，失敗的逐一說明是「字寬變了、斷言要更新」（更新前先確認 390 寬沒有橫向捲動）還是真的壞了。

## Tasks / Subtasks

- [ ] **T0 確認順序**：分支上 mono 單的 T1–T5（程式碼、守門測試、E2E、授權、量大小）已完成；`gh pr list` 沒有別的 PR 在改 `tests/visual/**-snapshots/`。
- [ ] **T1 套件＋載入（AC #1）**：改動前先 `nx build web` 記數字（AC #10）；`pnpm add -w @fontsource-variable/noto-sans-tc@5.3.0`（確認 `package.json` 是精確版本）；`main.tsx` import。
- [ ] **T2 主題（AC #2、#3）**：`@theme` 加 `--font-sans`、補 `--font-mono` 尾端；刪 `.font-primary`；`nx build web` 後 `grep -o -- '--font-sans:[^;]*' dist/apps/web/assets/*.css`、`grep -c 'font-primary'`（應為 0）、數 woff2。
- [ ] **T3 守門測試＋E2E（AC #4、#5）**。
- [ ] **T4 Go：woff2 不 gzip（AC #6）**：改 `main.go:1413`；`go test ./...`、`staticcheck ./...`。
- [ ] **T5 授權與文件（AC #8、#9）**。
- [ ] **T6 量大小＋Docker 實測（AC #7、#10）**：mono 單的 Docker 實測可以在這裡一起做（同一個映像、多打一個 JetBrains 區塊），兩張單的 Completion Notes 各貼自己的那幾行。
- [ ] **T7 基準圖（AC #11）**：兩張單合併執行一次。
- [ ] **T8 全部閘門＋E2E 全套（AC #12）**。

## Dev Notes

### 跟 mono 單怎麼排（Bob 建議：同一條分支、同一個 PR）

- **為什麼合併**：兩張單的大頭都是「幾乎每一張基準圖重產」。分開做＝重產兩次（本機一次約十分鐘以上）、CI 開兩次 bootstrap PR、Alexyu 看兩次「357 張截圖全變了」；而且第二張的 PR 一定要等第一張的 bootstrap PR 合併後才能開，否則 `-linux` 基準互相打架。合併做只有一次。
- **為什麼還是兩張單**：兩張的驗收不同（等寬字要關連字、內文字要選字重與 gzip），分開寫比較好查；PR 描述把兩張都列出來。
- **分支上的順序**：
  1. mono 的程式碼 commit（mono 單 T1–T5）；
  2. 本單的程式碼 commit（本單 T1–T6）；
  3. 一個 `chore(visual)` commit：重產的 darwin 基準＋`git rm` 的 linux 基準（AC #11）。
- 分支名建議 `feat/bundle-fonts`；commit 範例 `feat(fonts): 內文改用 Vido 自帶的 Noto Sans TC，每台裝置的中文長得一樣`。PR 標題講結果（例：「中文與數字字型改用 Vido 自帶的檔案，跟設計稿一致、NAS 不連網也一樣」）。
- **如果 Alexyu 不同意合併**：本單排在 mono 單（含它的 bootstrap PR）**合併之後**才開工，AC #3 就改成在本單裡改 `--font-mono`；基準圖照 AC #11 自己再跑一輪。

### 不要做的事

- 不要用 Google Fonts 或任何 CDN——NAS 可能不能連外（⚖️ 選 (a) 的前提）。
- 不要裝靜態版 `@fontsource/noto-sans-tc`（四個字重約 30 MB，每頁要抓的也更多）；不要用 fontTools 自己裁軸或裁字、不要把 woff2 複製進 repo、不要自己寫 `@font-face`。
- 不要改 `font-display`、不要加 `<link rel="preload">`（理由見「查到的事」）。
- 不要改任何元件的字重、字級、`font-*` 類別；不要「順便」把 700 改成 600。
- 不要開 `font-feature-settings`（例 `palt` 比例寬標點）——設計稿沒有，會改變中文標點寬度。
- 不要改 `index.html` 的 `lang`（見 Discovery Triage，等 Alexyu 決定）。
- 不要 commit `-linux.png`；不要跑 `test:visual:update`。

### 已知陷阱

- **可變字型的預設軸是 100**（字型 `fvar` 的 `wght` 預設值 100，scratchpad 用 fontTools 讀出）：只要元素有 `font-weight`（所有元素都有，預設 400），瀏覽器就會把軸設到對應值，沒問題；但**不要**寫 `font-variation-settings: normal` 之類的覆寫，也不要在 Canvas／SVG 裡直接引用這個字族而不給字重。
- **import 放 `main.tsx`，不要放 `styles.css`**（理由同 mono 單「已知陷阱」第一條）。
- **英數字寬度會變**（⚠️ 未查證寬窄方向，dev 跑 E2E 時以實測為準）：Noto Sans TC 的 Latin 字形跟 SF／Segoe 不同，片名 `truncate`、按鈕寬度、固定寬欄位（例：列表的 360／104／64 欄）可能多截幾個字——這是預期，但 390 寬不准出現橫向捲動（`feedback_measure_the_breakpoint_you_return_to`：要量斷點本身）。
- **CI 第一次跑會比較慢一點**：每個 E2E／visual 測試是新的瀏覽器 context，每頁要從本機 dev server 抓約 1 MB 字型；本機傳輸可忽略，但如果 E2E 有很緊的逾時（<1 秒等文字出現），要注意。
- **`document.fonts.ready` 只等「已經開始載入」的字**：新出現的文字帶出的區塊是之後才開始下載（AC #11 第 4 步）。
- **gzip 選項是取代**：只寫 `[".woff2"]` 會讓 PNG／JPEG 開始被 gzip。

### 測試標準

- vitest 斷言具體值（Rule 16）；字型有沒有真的套上一律在真瀏覽器量（jsdom 不算 CSS）。
- E2E：`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-e2e-dummy npx playwright test tests/e2e/font-sans.spec.ts --project=chromium --reporter=line`。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`；visual 與測試都不要背景跑。
- Go 改動 push 前跑 `~/go/bin/staticcheck ./...`（`lint:all` 不含它）。

### Project Structure Notes

- 改：根目錄 `package.json`、`pnpm-lock.yaml`、`apps/web/src/main.tsx`、`apps/web/src/styles.css`、`apps/api/cmd/api/main.go`、`DESIGN.md`、守門 spec、大量 `tests/visual/components.visual.spec.ts-snapshots/**`。新：`tests/e2e/font-sans.spec.ts`、`apps/web/public/licenses/NotoSansTC-OFL.txt`、`docs/licenses/noto-sans-tc.md`。
- Cross-stack 檢查：後端 1 個 task（T4）、前端 ≤3 個主要 task → 不用拆。
- Rule 17（雙語）：`docs/licenses/` 現有兩份都只有 zh-TW、屬內部盤點文件，照現況不另寫英文版（同 mono 單）。
- 不動 `ux-design.pen`：`.pen` 本來就是 Noto Sans TC（`pen-tokens.json:152`），本單是讓程式碼追上設計稿。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（只改字型、CSS 主題與一個 gzip 選項）。

### References

- [Source: apps/web/src/styles.css:509-519, 522, 536-549]
- [Source: node_modules/tailwindcss/index.css:1, 5-7, 455, 464, 514-519, 589-591]（4.1.18）
- [Source: apps/web/index.html:1-37]、[apps/web/src/main.tsx:1-16]、[apps/web/vite.config.mts:46-53]
- [Source: DESIGN.md:40-95（frontmatter typography）, 336-341, 351-360, 368]
- [Source: _bmad-output/pen-tokens.json:103-118, 140-155, 177-255（Type/*/Weight、Type/Family/*）]
- [Source: apps/web/src/components/library/PosterCardV2.tsx:176]、[LibraryListRowV2.tsx:93]、[media/DetailHeroV2.tsx:135]、[activity/ActivityHub.tsx:396]、[settings/BackupTable.tsx:278]
- [Source: Dockerfile:27-41, 97-100, 119]、[apps/api/cmd/api/static.go:12-21, 34-70, 75-102]、[apps/api/cmd/api/main.go:1411-1415]
- [Source: ~/go/pkg/mod/github.com/gin-contrib/gzip@v1.2.5/options.go:15-20, 55-59]
- [Source: $(go env GOROOT)/src/mime/type.go:116、src/net/http/sniff.go:177-178]（go1.26.3）
- [Source: tests/visual/components.visual.spec.ts:236-250]、[playwright.config.ts:62-63, 163-166, 224-246]、[.github/workflows/visual-regression.yml:40-42]
- [Source: docs/licenses/opencc.md]、[NOTICE:10-11]
- [Source: npm `@fontsource-variable/noto-sans-tc@5.3.0`／`@5.3.1`、`@fontsource/noto-sans-tc@5.3.0` — `npm view`、`npm pack` 到 scratchpad 檢視 `wght.css`、`400.css`、`metadata.json`、`LICENSE`、`files/`（2026-10-19）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-font-mono-not-jetbrains.md — AC #2、#4、#6、#7、#10、Discovery Triage]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **① 已併入本單 AC #13（⚖️ Alexyu 2026-10-19「一起改」）** — 原 `disc-2026-10-html-lang-en`（Bob 建單時發現；**需要 parent 在 sprint-status 新增這一行**）：`apps/web/index.html:2` 是 `<html lang="en">`，但整個介面是繁體中文。影響：螢幕報讀器用英文發音念中文；Chrome 可能跳「要翻譯這個英文網頁嗎」；字型還沒載入或遇到罕用字時，瀏覽器依 `en` 猜中文要用哪套系統字（本單用明寫 `PingFang TC`／`Microsoft JhengHei` 繞過了字型這一點）。`TmdbAttribution.tsx:120` 已經對英文段落標 `lang="en"`，表示預期整頁是中文。改成 `zh-Hant-TW` 可能改變中文斷行與部分預設樣式 → **又是一次全部基準圖重產**。Bob 建議：**併進本單同一個 PR（lane ①，加一條 AC）**，搭這次重產的便車；若 Alexyu 不要，就照 lane ③ 開單、P3。
- **① expand-scope-in-place** — woff2 被 gzip 一次（`main.go:1413-1415`＋`gin-contrib/gzip` 預設清單）→ 已收進 **AC #6**（並由 AC #7 驗收）。

### File List
