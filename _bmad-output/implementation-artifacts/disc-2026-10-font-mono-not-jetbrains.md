# Disc：把 JetBrains Mono 字型放進 Vido，數字、路徑、檔名真的用設計稿那套等寬字（離線 NAS 也一樣）

Status: ready-for-dev

**Source:** Rule 24 lane ③，2026-10-19 由 `disc-2026-10-library-list-triage-b2` 建單時發現（見 b2「查到的事」與 Discovery Triage）；critique P3「font-mono 沒吃到 JetBrains Mono」（`.impeccable/critique/2026-10-19T00-18-51Z__ps-web-src-components-library-librarylistrowv2-tsx.md`）。
⚖️ **Alexyu 2026-10-19 選 (a)：把字型放進 Vido**（自帶字型檔＋`@theme --font-mono`）。不選 (b)「只修 cascade 不載字型」（Mac 會從 SF Mono 變成 Monaco，更糟）、不選 (c)「改 DESIGN.md 承認用系統等寬字」。

**排程（不是硬性 blocked-by）：** 本單會重產大量視覺基準圖。**不要跟任何會動基準圖的單子同時在途**（目前：`disc-2026-10-library-list-triage-b1`、`-b2`、`disc-2026-10-detail-resolution-raw`）。建議排在它們都合併之後——這樣本單的 PR 只有「換字型」一種差異，CI 的 `-linux` bootstrap PR 也不會跟別張單子的搶同一批檔案。

## 查到的事（2026-10-19，本 session 讀過的程式碼與套件）

- **cascade 輸了**：`apps/web/src/styles.css:547-549` 的 `.font-mono { font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace; }` 寫在 `@layer base`（`:522` 開始）。Tailwind v4（4.1.18）宣告層順序 `theme, base, components, utilities`（`node_modules/tailwindcss/index.css:1`），它自己的 `font-mono` 工具類在 `utilities`、值是預設 `--font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace`（`index.css:9-11`）→ base 那條永遠輸。實際建置結果（`dist/apps/web/assets/index-DHZ_qeAc.css`，2026-10-14 build）：位置 8836 `.font-mono{font-family:JetBrains Mono,Consolas,Monaco,monospace}` 在 `@layer base` 裡，位置 47533 `.font-mono{font-family:var(--font-mono)}` 在 `@layer utilities` 裡。
- **字型本身也沒有載入**：`apps/web/index.html` 沒有字型連結（只有 `/src/styles.css` 與 `main.tsx`）；`styles.css` 沒有 `@font-face`；根目錄 `package.json` 沒有任何 `@fontsource`（`grep -rn "fontsource\|@font-face\|fonts.googleapis" apps/web/index.html apps/web/src package.json` 0 筆）；`apps/web/public/` 只有 `favicon.ico`、`images/`。所以就算 cascade 對了，也只有本機剛好裝了 JetBrains Mono 的人看得到。
- **用量**：`font-mono` 在 `apps/web/src` 非測試檔 73 個檔、182 處；搭配字重：預設 400、`text-xs` 預設 500（`styles.css:509-512`，`--text-xs--font-weight: 500`）、`font-medium` 3、`font-semibold` 33、`font-bold` 2 → 需要 400／500／600／700。`<code>`／`<pre>`／`<kbd>` 11 處，吃 Tailwind preflight 的 `--default-mono-font-family`＝`--theme(--font-mono)`（`index.css:463-467`、`:600-606`）→ 改 `--font-mono` 會一起換掉，這是要的。
- **沒有其他地方寫死字型**：`grep -rn "JetBrains\|fontFamily\|font-family" apps/web/src` 只有 `styles.css:539`、`:548` 與一個測試名稱（`FilterPanel.spec.tsx:111`，斷言 className，不受影響）。
- **設計文件**：`DESIGN.md:337`「**讀數字體：** JetBrains Mono（後備 Consolas、Monaco、monospace）」、`:368`「等寬（JetBrains Mono）不是獨立的字級階」——跟本單要做的一致，只有後備清單與「隨 Vido 打包」要補一句。
- **怎麼出貨**：Docker 建置時 `npx nx build web` 把 `dist/apps/web` 複製到 `/app/public`（`Dockerfile:41`、`:119`）；Go 用 `router.Static("/assets", …)` 供應、`Cache-Control: public, max-age=31536000, immutable`（`apps/api/cmd/api/static.go:17`、`:42-46`），並加 `X-Content-Type-Options: nosniff`（`static.go:118`）。Vite 會把 CSS 引用的 `.woff2` 打成 `/assets/<名稱>-<hash>.woff2`。Go 內建 MIME 表沒有 `.woff2`（`$(go env GOROOT)/src/mime/type.go` 只到 `.wasm`），執行映像是 `alpine:3.21` 只裝 `ca-certificates tzdata ffmpeg su-exec`（`Dockerfile:97-100`）、沒有 `/etc/mime.types`，所以會走內容嗅探——Go 的嗅探表認得 `wOF2` → `font/woff2`（`$(go env GOROOT)/src/net/http/sniff.go:178`）。`nosniff` 依 Fetch 規範只管 script／style，不擋字型。**執行時不連外**：字型在映像裡、`pnpm install` 只在建置階段。
- **套件**（2026-10-19 `npm view`／`npm pack` 查）：`@fontsource-variable/jetbrains-mono@5.3.0`（2026-07-19 發布，OFL-1.1）。內容：`wght.css`（6 個子集各一條 `@font-face`，`font-family: 'JetBrains Mono Variable'`、`font-weight: 100 800`、`font-display: swap`、各自 `unicode-range`）＋6 個 woff2，合計 92 KB；其中 latin 子集 `jetbrains-mono-latin-wght-normal.woff2` 40,404 bytes，`unicode-range: U+0000-00FF, …, U+2000-206F, …`（含 `×` U+00D7、`·` U+00B7、`…` U+2026、`—` U+2014）。對照：靜態版 `@fontsource/jetbrains-mono@5.3.0` latin 每個字重約 21 KB，四個字重 ≈ 86 KB——**可變字型一個檔 40 KB 涵蓋 400–700，比較小**。瀏覽器只下載用得到的子集（unicode-range），一般畫面只會抓 latin 那個。
- **授權**：OFL-1.1 允許跟軟體一起打包、散布（套件 `LICENSE` 開頭：「Copyright 2020 The JetBrains Mono Project Authors … licensed under the SIL Open Font License, Version 1.1」）；條件是每份拷貝要附著作權聲明與授權全文（可以是獨立文字檔），而且不能單獨販售字型。Vido 的授權盤點放在 `docs/licenses/`（`opencc.md`、`runtime-copyleft.md`；`NOTICE:10-11` 指向這個資料夾）。
- **視覺測試等字型**：截圖前有 `await page.evaluate(() => document.fonts.ready)`（`tests/visual/components.visual.spec.ts:244-245`），而且是在畫廊區塊可見之後才等（`:243`）——unicode-range 延遲載入的字型會被等到。門檻 `maxDiffPixelRatio: 0.001`（`playwright.config.ts:62-63`）。基準圖現況：`components/` 下 152 個夾具資料夾、`-darwin.png` 357 張、`-linux.png` 357 張。
- **JetBrains Mono 預設開連字**（`calt`）：`->`、`=>`、`==`、`!=`、`<=`、`--`、`...`、`::` 會被畫成合字。Vido 的 `font-mono` 用在檔名、路徑、數字（例：檔名常見 `--`、`...`），合字會讓原文看起來不是原文。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要 Vido 的數字、檔名、路徑用設計稿指定的 JetBrains Mono 顯示，不管我用哪台電腦、NAS 有沒有連網，
這樣畫面上的讀數長得跟設計一樣，每台裝置看起來也一致。

## Acceptance Criteria

1. **字型隨 Vido 打包，不連外。** 根目錄 `package.json` 的 `dependencies` 加 `"@fontsource-variable/jetbrains-mono": "5.3.0"`（**鎖精確版本**），`pnpm-lock.yaml` 跟著更新；在 `apps/web/src/main.tsx` 最上面 `import '@fontsource-variable/jetbrains-mono/wght.css';`。`nx build web` 後 `dist/apps/web/assets/` 有 `jetbrains-mono-*-wght-normal-<hash>.woff2`；建置後的 CSS 與 JS 裡**沒有** `fonts.googleapis`、`fonts.gstatic`、`cdn.jsdelivr`、`unpkg` 等外部網址（`grep` 證明）。只載直立體（`wght.css`），不載 `wght-italic.css`。
2. **`font-mono` 真的是 JetBrains Mono。** `styles.css` 加
   ```css
   @theme {
     --font-mono: 'JetBrains Mono Variable', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
     --font-mono--font-feature-settings: 'calt' 0, 'liga' 0;
   }
   ```
   （一般 `@theme`，不是 `@theme inline`——值是字面量，兩者結果相同，跟檔內既有兩個 `@theme` 一致。）**刪掉** `@layer base` 裡 `:547-549` 那條死規則 `.font-mono { … }`。建置後的 CSS：只剩一條 `.font-mono`、在 `@layer utilities`、`--font-mono` 以 `'JetBrains Mono Variable'` 開頭。
3. **合字關掉。** 在瀏覽器裡（AC #5 的 E2E）某個 `font-mono` 元素的 `getComputedStyle(el).fontFeatureSettings` 含 `"calt" 0`。⚠️ 未查證（dev 開工時先確認）：Tailwind 4.1 的 `font-mono` 工具類是否會一併輸出 `font-feature-settings: var(--font-mono--font-feature-settings)`——看建置後的 CSS；**如果不會**，改在 `@layer base` 加 `code, kbd, samp, pre, .font-mono { font-variant-ligatures: none; }`（工具類不設這個屬性，所以 base 層不會被蓋掉），並在 Completion Notes 寫明走哪一條。
4. **CSS 守門測試。** `apps/web/src/styles-type-scale.spec.ts`（或新 `styles-fonts.spec.ts`，照 `styles-motion.spec.ts:13` 讀 `styles.css` 的寫法）斷言：`@theme` 的 `--font-mono` 第一個是 `'JetBrains Mono Variable'`；`@layer base` 裡沒有 `.font-mono {` 宣告 `font-family`；`main.tsx` 有 import `@fontsource-variable/jetbrains-mono/wght.css`。
5. **瀏覽器實測（Rule 28「Browser layout」列）。** 新 `tests/e2e/font-mono.spec.ts`：打開 `/library`（用 `tests/support/helpers/library-stubs.ts` 的 stub）或畫廊頁，找一個有 `font-mono` 的元素：
   - `getComputedStyle(el).fontFamily` 以 `"JetBrains Mono Variable"` 開頭；
   - `await document.fonts.ready` 之後 `document.fonts.check('500 12px "JetBrains Mono Variable"')` 為 `true`；
   - 這次頁面載入**沒有任何**對非 `localhost` 主機的字型請求（`page.on('request')` 收集 `resourceType() === 'font'` 的網址）。
6. **Docker 映像實測（離線 NAS）。** 本機 `docker build -t vido:font-check .` → `docker run --rm -p 8099:8080 …`（拋棄式容器；Go API 預設 `AI_PROVIDER=gemini`、沒金鑰會直接退出，所以至少帶 `-e AI_PROVIDER=claude -e CLAUDE_API_KEY=sk-ant-dummy`——見 `.claude/memory/project_local_visual_run_env.md`）→ `curl -sI http://localhost:8099/assets/<latin woff2 檔名>`：`200`、`Content-Type: font/woff2`、`Cache-Control: public, max-age=31536000, immutable`。Completion Notes 貼這三行。（若本機 Docker 不能用，說明原因，並改用 `VIDO_PUBLIC_DIR=dist/apps/web go run ./cmd/api` 打同一個請求。）
7. **授權跟著走。**
   - `apps/web/public/licenses/JetBrainsMono-OFL.txt`：套件 `LICENSE` 全文原樣複製（Vite 會把 `public/` 原樣放進 `dist/apps/web/`，所以映像的 `/app/public/licenses/` 有這份——字型的每一份拷貝都附授權，OFL 條件 2）。
   - 新 `docs/licenses/jetbrains-mono.md`（照 `docs/licenses/opencc.md` 的格式、zh-TW）：套件名與版本、授權 OFL-1.1、出處 <https://github.com/JetBrains/JetBrainsMono>、映像內位置（`/app/public/assets/*.woff2`、`/app/public/licenses/JetBrainsMono-OFL.txt`）、「不單獨販售字型、未修改字型檔」。
8. **DESIGN.md 跟程式碼一致。** `DESIGN.md:337` 改成：「**讀數字體：** JetBrains Mono（隨 Vido 打包：`@fontsource-variable/jetbrains-mono`，可變字重，不連外；關閉連字；後備 `ui-monospace`、SF Mono、Menlo、Monaco、Consolas、monospace）」。`:368` 不用改。
9. **大小有量。** Completion Notes 貼改動前後（同一台機器、同一個 commit 前後各 build 一次）：`dist/apps/web/assets/` 的 CSS 大小（原始／gzip）、JS 大小、新增 woff2 檔數與總大小，以及「一般頁面實際下載的字型」（AC #5 收集到的字型請求與大小）。預期：CSS +1～2 KB（6 條 `@font-face`）、woff2 6 個共約 92 KB 在映像裡、第一次開頁面約下載 40 KB（latin），之後一年快取。
10. **視覺基準圖全部重產。**
    1. 改完先跑 `AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-visual-dummy pnpm run test:visual --reporter=line`，記下**所有**失敗的夾具／狀態（＝受字型影響的基準圖）。
    2. 對每一張失敗的：刪掉 `-darwin.png`、`git rm` 對應的 `-linux.png`。
    3. `pnpm run test:visual:update-missing` 產新的 darwin（會 exit 1 並印 "writing actual"，正常）；再跑一次 `test:visual` 全綠。
    4. 隨機打開至少 5 張新的 darwin 圖，肉眼確認是 JetBrains Mono（例：數字 `0` 中間有點、小寫 `l` 有彎尾），Completion Notes 列出看了哪幾張。
    5. **不要**自己產 `-linux.png`、不要跑 `test:visual:update`；PR 的 CI Visual Regression 會開 `chore(visual): bootstrap N missing -linux baselines` PR，核對檔案清單＝第 2 步刪掉的那批之後合併。
    6. Completion Notes 寫受影響張數（darwin 改了幾張、linux 刪了幾張，兩個數字要相等）。
11. **沒有別的東西壞掉。** `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check`、`check-design-refs.py` 綠；全套 E2E 本機跑一次（字寬改變可能影響量版面的 E2E），失敗的要逐一說明是「字寬變了、斷言要更新」還是真的壞了。

## Tasks / Subtasks

- [ ] **T1 套件＋載入（AC #1）**：`pnpm add -w @fontsource-variable/jetbrains-mono@5.3.0`（確認 `package.json` 寫成精確版本，不是 `^`）；`main.tsx` import。
- [ ] **T2 主題（AC #2、#3）**：`styles.css` 加 `@theme`、刪 `:547-549`；`nx build web` 後檢查 CSS（`grep -o '\.font-mono{[^}]*}'`、`--font-mono:`），確認 feature-settings 有沒有跟著出來，決定 AC #3 走哪條。
- [ ] **T3 守門測試＋E2E（AC #4、#5）**。
- [ ] **T4 授權與文件（AC #7、#8）**。
- [ ] **T5 量大小＋Docker 實測（AC #6、#9）**：改動前先 build 一次記數字（本 session 看到的 2026-10-14 build：CSS 105,954 bytes／gzip 18,150、JS 1,518,691 bytes，**只能當參考，要自己在改動前重 build**）。
- [ ] **T6 基準圖（AC #10）**：照 AC #10 的六步。
- [ ] **T7 全部閘門＋E2E 全套（AC #11）**。

## Dev Notes

### 不要做的事

- 不要用 Google Fonts 或任何 CDN 連結——NAS 可能不能連外（⚖️ Alexyu 選 (a) 的前提就是字型在映像裡）。
- 不要自己寫 `@font-face`、不要把 woff2 複製進 repo——用套件，升級只改版本號。
- 不要載入斜體、不要載入靜態字重的 `@fontsource/jetbrains-mono`（四個字重 86 KB > 可變字型 40 KB）。
- 不要改內文字型（`.font-primary`／Noto Sans TC）——那是另一張單（見 Discovery Triage），本單只管等寬字。
- 不要改任何元件的 `font-mono` 用法、字級、字重；不要「順便」把某處改成不用等寬。
- 不要 commit `-linux.png`；不要跑 `test:visual:update`。

### 已知陷阱

- **`@import` 放 `main.tsx`，不要放 `styles.css`**：`styles.css` 由 `index.html` 直接連結、經 `@tailwindcss/postcss` 處理；在 CSS 裡 `@import` 套件檔，url 改寫要靠 Tailwind 的 import 解析，多一層風險。放 `main.tsx` 是 fontsource 的標準用法，Vite 會把它併進建置後的 CSS 並改寫 woff2 路徑。
- **`font-display: swap`**（套件預設）：第一次開頁面會先用系統等寬字、字型到了再換（區網上通常 <100 ms），之後一年快取不會再發生。不要改成 `block`（字型沒到前數字是空白）或 `optional`（第一次載入可能整次都不用）。
- **CJK 落在等寬字裡**（例：路徑 `/media/movies/你的名字 (2016)/…`）：JetBrains Mono 沒有中文字，中文字照舊落到後備鏈、由系統補字；跟今天一樣，不是 bug。
- **E2E 字寬**：JetBrains Mono 的字寬是 0.6 em，跟 SF Mono／Menlo 很接近，但不是完全相同；量寬度的 E2E 若只差幾 px，更新斷言前先確認版面在 390 寬沒有橫向捲動。
- **CI 上的 linux 基準圖會更接近 darwin**：以前 linux 用系統的 DejaVu／Liberation Mono、darwin 用 SF Mono；之後兩邊都是同一個 woff2。這是好事，但第一次 bootstrap 的張數會很多。

### 測試標準

- vitest 斷言具體值（Rule 16）；字型是不是真的套上一律用 E2E 在真瀏覽器量（jsdom 不算 CSS）。
- E2E：`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-e2e-dummy npx playwright test tests/e2e/font-mono.spec.ts --project=chromium --reporter=line`。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`；visual 不要背景跑。

### Project Structure Notes

- 改：根目錄 `package.json`、`pnpm-lock.yaml`、`apps/web/src/main.tsx`、`apps/web/src/styles.css`、`DESIGN.md`、守門 spec、大量 `tests/visual/components.visual.spec.ts-snapshots/**`。新：`tests/e2e/font-mono.spec.ts`、`apps/web/public/licenses/JetBrainsMono-OFL.txt`、`docs/licenses/jetbrains-mono.md`。
- Rule 17（雙語）：`docs/licenses/` 現有兩份都只有 zh-TW、屬於內部盤點文件，照現況不另寫英文版。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（只改字型與 CSS 主題）。

### References

- [Source: apps/web/src/styles.css:487-519, 522, 536-549]
- [Source: node_modules/tailwindcss/index.css:1, 9-11, 463-467, 600-606]（4.1.18）
- [Source: apps/web/index.html]、[apps/web/src/main.tsx:1-16]
- [Source: DESIGN.md:336-337, 341, 368]
- [Source: Dockerfile:17-41, 97-119]、[apps/api/cmd/api/static.go:12-62, 118]
- [Source: tests/visual/components.visual.spec.ts:243-245]、[playwright.config.ts:62-63, 165]
- [Source: docs/licenses/opencc.md]、[NOTICE:10-11]
- [Source: npm `@fontsource-variable/jetbrains-mono@5.3.0` — `wght.css`、`LICENSE`、`files/`（2026-10-19 `npm pack` 檢視）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-b2.md — 查到的事「font-mono 沒吃到 JetBrains Mono」、Discovery Triage]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **③ backlog-with-carry-forward-link** — `disc-2026-10-font-sans-not-noto`（Bob 建單時發現；**需要 parent 在 sprint-status 新增這一行**）：內文字型也一樣沒照 DESIGN.md。`DESIGN.md:336`、`:341` 寫內文與 CJK 一律 Noto Sans TC，但 `.font-primary`（`styles.css:537-545`）全站沒有任何元件使用（`grep -rn "font-primary" apps/web/src` 只命中 `styles.css` 自己），`body` 吃 Tailwind 預設 `--font-sans`（`ui-sans-serif, system-ui, …`，`node_modules/tailwindcss/index.css:5-7`），Noto Sans TC 也沒有載入。Mac 上實際是蘋方、Windows 是微軟正黑。要 Alexyu 選：(a) 自帶 Noto Sans TC（繁中子集每個字重仍是 MB 級，映像與首次載入都會明顯變大）／(b) 改 DESIGN.md 承認用系統中文字型、刪掉死規則 `.font-primary`。Bob 傾向 (b)，理由是成本差兩個數量級、而且系統中文字型在各平台都已經是為螢幕調過的字。P3。

### File List
