# Disc：比對失敗、片名是一長串檔名的詳細頁，手機和平板不會再被撐寬、可以左右滑

Status: ready-for-dev

**Priority:** P3（但正式機今天有 10 部電影會中，見下）

**Source:** Rule 24 lane ③，2026-10-19 由字型 PR #840（`disc-2026-10-font-sans-not-noto`）的 dev 回報：「詳細頁 390 寬有橫向捲動，**改動前就有**（`Home.Video.Collection.Vol1` 這種沒有空格的長片名撐出 hero），不是字型造成的」（見該單 Completion Notes「畫面比對」）。
⚖️ **Alexyu 2026-10-19：開單，這張接著進開發。**

**Sally 追認（不擋開工）：** 斷行方式照 B10p-M「片名換行、不截斷」；設計稿沒畫到「整段沒有空格」的情況，本單建議**在字中間斷**（見 Dev Notes「斷行方式」）。dev 在 Step 9 附 390／320 截圖，請 Sally 在 PR 上追認；若 Sally 要改成省略號，只動 AC #1 的 class 與 AC #5 的夾具，測試的「不能左右滑」照留。

## 查到的事（2026-10-19，本 session 讀過的程式碼、設計稿、正式機資料，並在真瀏覽器量過）

### 是哪一個元素撐寬的

- **只有一個元件：`DetailHeroV2`**（`apps/web/src/components/media/DetailHeroV2.tsx`）。
  - 片名 `<h1>`（`:135-137`）：`text-2xl font-bold leading-tight … sm:text-3xl`，**沒有任何斷字設定**。
  - 原文片名 `<p>`（`:138-142`）：同樣沒有斷字設定（原文片名跟片名不一樣時才出現）。
  - 兩者的父層 `<div className="min-w-0 flex-1 pb-1">`（`:121`）**已經有 `min-w-0`**，所以父層本身沒有被撐寬；是文字「溢出」父層、超出畫面，讓整頁可以左右滑。少的是文字本身的 `overflow-wrap`，不是 `min-w-0`。
- **怎麼量的**：把 `dist/apps/web`（字型 PR 的建置）複製到 scratchpad、用 Playwright 開 `/media/movie/<id>`、API 全部 stub（不碰 repo、不碰正在跑的 dev server），逐一檢查每個文字節點的右緣有沒有超過 `document.documentElement.clientWidth`。腳本在 session scratchpad `harness/measure.cjs`，不進 repo。
- **量到的數字**（片名用正式機真實的 `[bitsearch.to] Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLER.mkv`）：

  | 視窗寬 | 改前 `scrollWidth` | 撐寬的元素 | 只在 h1＋原文片名加 `overflow-wrap: break-word` 後 |
  | --- | --- | --- | --- |
  | 320 | **604** | `h1` | 320 |
  | 360 | **604** | `h1` | 360 |
  | 390 | **604** | `h1` | 390 |
  | 640 | **1051** | `h1`（`sm:text-3xl` 變大） | 640 |
  | 768 | **1051** | `h1` | 768 |
  | 1024 | **1051** | `h1` | 1024 |
  | 1280／1440 | 等於視窗寬（沒問題） | — | 不變 |

  → **不只手機：平板和 1024 寬的小筆電也會被撐寬。** `Home.Video.Collection.Vol1`（dev 回報的例子）在 320–390 是 454。原文片名換成沒有空格的長字串（例 `Home.Video.Collection.Vol1.Original`）在 360 也會單獨溢出（右緣 364）。
- **一起量過、沒問題的地方（不改）**：
  - 同一頁「檔案資訊」的路徑列 `<dd className="truncate font-mono …">`（`DetailTechInfoV2.tsx:149`）：文字超出但被 `truncate` 剪掉，不影響頁寬。
  - 媒體庫網格（`PosterCardV2.tsx:176` 的 `line-clamp-2`）、媒體庫列表（失敗列的檔名中段省略、成功列的片名都量過）、首頁：320／360／390 都等於視窗寬。
  - 從詳細頁開的對話框：「管理字幕」sheet 標題（`truncate`，長片名被剪成「管理字幕 — Kimi.no.Na.wa.2016.BDRip…」）、「手動選片」、「修改資訊」：都不讓頁面左右滑。
  - 活動頁列的標題是 `truncate`、說明是 `break-words`（`ActivityRow.tsx:64、:101`）；生成工作區已經有 390 寬的 `scrollWidth` E2E（`tests/e2e/generation-workspace-mobile.spec.ts:135、:204`）；麵包屑只出現在生成工作區，內容是固定字「活動／生成字幕」，沒有片名。
  - 影集詳細頁（`LocalDetailV2` 的影集分支）與 TMDb 詳細頁（`TMDbDetailV2.tsx:93`）用**同一個** `DetailHeroV2`，本單的修正一起生效。
- **正式機真的會碰到**（2026-10-19 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：55 部電影裡 **10 部是比對失敗、片名就是整個檔名**、`original_title` 是空的，例：`[bitsearch.to] Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLER.mkv`（72 字）、`[bitsearch.to] Now.You.See.Me.Now.You.Dont.2025.2160p.WEB-DL.SDR.DDP5.1.H265.MP4-BEN.THE.MEN.mp4`、`[长安的荔枝].The.Lychee.Road.2025.IMAX.2160p.HQ.WEB-DL.HEVC.10bit.HDR10.DTS-X5.1.4.6Audios-QHstudIo.mp4`。中文字之間本來就能斷行，但後面那一大段英數＋句點沒有任何斷點。

### 設計稿怎麼畫

- **B10p-M**（`H2MRl`，`_bmad-output/screenshots/flow-b-detail-v2/b10p-m.png`，比對失敗的手機稿）：片名就是檔名 `[Leopard-Raws] Kimi no Na wa (BD).mkv`，**換成兩行、整段顯示、沒有省略號**。
- **B3p-M**（`SzNRb`，`b3p-m.png`）：一般片名一行。
- DESIGN.md 沒有寫標題的斷行或截斷規則（`grep -n "truncat\|省略\|斷行\|換行\|ellipsis\|overflow" DESIGN.md` 只命中金額寫法那段，無關）。
- → 設計的意思是「換行、不截斷」；**沒畫到的是「一整段沒有空格」時怎麼斷**。本單建議在字中間斷（Sally 追認，見最上面）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要比對失敗、片名是一長串檔名的電影，在手機和平板上打開詳細頁時整頁照樣固定在畫面寬度內、片名整段換行顯示，
這樣我不用左右拖畫面才看得到「手動選片」，也不會一按就滑到旁邊去。

## Acceptance Criteria

1. **片名與原文片名可以在字中間換行。** `DetailHeroV2.tsx` 的 `<h1>`（`:135`）與原文片名 `<p>`（`:139`）加 `break-words`（Tailwind 4.1.18 產生 `overflow-wrap: break-word`；跟 repo 既有寫法一致：`ActivityRow.tsx:64`、`LogEntry.tsx:84`、`DownloadDetailSheet.tsx:64`）。有空格的片名照舊在空格處換行（`break-word` 只在單一個「字」放不下時才在字中間斷），**一般片名的畫面完全不變**。
2. **父層的 `min-w-0` 保留，並加註解說明它是承重的**（`:121`）：拿掉它，`break-word` 就不會生效（flex 子元素的最小寬度會變成那一整段字的寬度）。
3. **不截斷、不加省略號、字級不變。** 片名的完整文字都在畫面上（`toHaveText` 完整字串）；手機仍是 `text-2xl`、`sm` 以上仍是 `text-3xl`。
4. **E2E（真瀏覽器，這張單的主要證據）**——在 `tests/e2e/media-detail.spec.ts` 的 `Media Detail - no metadata` 區塊（`:166` 起）加一組 `long unbreakable title`：
   - 建一部電影：片名用正式機真實形狀 `[bitsearch.to] Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLER.${Date.now()}.mkv`、`parseStatus: 'failed'`。
   - 另建一部**有原文片名**的：片名 `E2E 長原文片名 ${Date.now()}`、原文片名 `Home.Video.Collection.Vol1.Remastered.${Date.now()}`（需要 T3 給 `seedMovie` 加 `originalTitle`；API 的 `CreateMovieRequest` 本來就收 `original_title`，`movie_handler.go:46`）。
   - 對兩部、在 **320×844、360×800、390×844、768×1024** 四個視窗各檢查：
     - `document.documentElement.scrollWidth <= document.documentElement.clientWidth`（整頁不能左右滑）；
     - `h1`（與原文片名那一行）的 `getBoundingClientRect().right <= clientWidth`；
     - `h1` 的文字是完整片名（`toHaveText`），而且在 320 寬時 `h1` 的高度大於一行（> 1.5 × 計算後的 `line-height`），證明是換行、不是縮字或剪掉。
   - **故意改壞會變紅**（dev 要實際做一次、寫進 Completion Notes）：(a) 拿掉 `h1` 的 `break-words` → 四個寬度全紅；(b) 留著 `break-words`、拿掉父層 `min-w-0` → 也要紅；(c) 拿掉原文片名的 `break-words` → 第二部紅。
5. **新的畫廊夾具＋基準圖（只產 darwin）。** `apps/web/src/routes/test/-gallery.fixtures.tsx` 在 `media-detail-hero-v2`（`:1767`）後面加 `media-detail-hero-v2-long-title-mobile`：
   - `component: DetailHeroV2`、`viewport: { width: 390, height: 844 }`、`penNode: 'H2MRl'`（B10p-M）、`statesOnly: ['default']`；
   - props 用正式機真實形狀（Rule 28）：`title: '[bitsearch.to] Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLER.mkv'`、**不給** `originalTitle`（正式機失敗列的 `original_title` 是空的）、`backdropPath: null`、`posterPath: null`、badges 一顆「失敗」（顏色照 `deriveLifecycleStatus`（`utils/libraryStatus.ts:57`）失敗態回傳的 `className`）、actions 放「管理字幕」「修改資訊」兩顆按鈕（同 `media-detail-hero-v2` 夾具的寫法）。
   - 本機產 `tests/visual/components.visual.spec.ts-snapshots/components/media-detail-hero-v2-long-title-mobile/default-visual-darwin.png`；**`-linux.png` 不要自己產**，CI 的 Visual Regression 會開 `chore(visual): bootstrap … -linux baselines` PR 補（CLAUDE.md）。
   - 既有基準圖**預期 0 張改變**（`media-detail-hero-v2` 的片名是「你的名字」，有沒有 `break-words` 都一樣）。若有任何既有基準圖變了，停下來在 Completion Notes 說明是哪張、為什麼。
6. **單元測試**（`DetailHeroV2.spec.tsx`）：加一條「片名 `h1` 與原文片名都帶 `break-words`、兩者的父層帶 `min-w-0`」。jsdom 不排版，這條只防有人手滑刪 class；真正的證明是 AC #4。斷言用具體 class 名（Rule 16）。
7. **檔頭（Rule 21）**：`DetailHeroV2.tsx:1` 的 `Design ref` 補上 `Screen B10p-M (H2MRl)`；`python3 scripts/check-design-refs.py` 綠（`H2MRl` 已在 `_bmad-output/pen-node-ids.json`、`scripts/export-pen-screenshots.py:300`）。
8. `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check` 綠；本機 chromium 跑 `media-detail.spec.ts` 全綠；本機 visual 跑新夾具綠。

## Tasks / Subtasks

- [ ] **T1 元件（AC #1–#3、#7）**：`DetailHeroV2.tsx` 的 `h1`、原文片名 `p` 加 `break-words`；`:121` 加一行註解說明 `min-w-0` 為什麼不能拿掉；檔頭補 B10p-M。
- [ ] **T2 單元測試（AC #6）**：`DetailHeroV2.spec.tsx` 加一條。
- [ ] **T3 E2E（AC #4）**：
  - [ ] `tests/support/helpers/seed-helpers.ts`：`SeedMovieOptions`（`:33`）加 `originalTitle?: string`，`seedMovie` body（`:66-74`）加 `if (opts.originalTitle) body.original_title = opts.originalTitle;`。
  - [ ] `media-detail.spec.ts` 加 AC #4 那組測試（每個寬度用 `page.setViewportSize`；量測寫成一個小函式 `expectNoSideScroll(page)`，寫法參考 `tests/e2e/library-list.spec.ts:93-98`）。
  - [ ] 故意改壞三次（AC #4 a／b／c），確認都會紅，再改回來。
- [ ] **T4 夾具＋基準圖（AC #5）**：加夾具；`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-visual-dummy pnpm run test:visual:update-missing`（只會產新夾具這一張 darwin）；再跑一次 `test:visual -g media-detail-hero-v2` 綠。
- [ ] **T5 Step 9 設計比對**：本機詳細頁截 320、390（手機）與 768（平板）三張，對 `b10p-m.png`：片名整段換行、不截斷、頁面沒有橫向捲軸。附在 PR，請 Sally 追認「沒有空格時在字中間斷」。

## Dev Notes

### 斷行方式（建議，需 Sally 追認）

- **建議：`break-words`（在字中間斷，不截斷）。** 理由：B10p-M 已經畫了「檔名片名換行、整段顯示」；失敗列的片名就是檔名，使用者要靠它認出是哪個檔案、再去「手動選片」，剪掉後半段（`.2160p.MA.WEB-DL…`）等於剪掉判斷依據。
- **為什麼不用省略號（`truncate`／`line-clamp`）**：一，跟 B10p-M 畫的不一樣；二，手機上沒有滑鼠停留，`title` 提示看不到，被剪掉的部分就真的看不到了。
- **為什麼不用 `break-all`**：`break-all` 連有空格的普通英文片名也會在字中間斷（例「The Roundup: Punish-ment」被拆開），一般片名的畫面會變；`break-word` 只在放不下時才斷。
- **`break-words` 還是 `wrap-anywhere`**：兩個在本頁的效果一樣（父層有 `min-w-0`，scratchpad 實測兩種在 320–1024 都等於視窗寬）。選 `break-words` 是因為 repo 已經在用；`wrap-anywhere` 的差別是「父層沒有 `min-w-0` 也有效」——本單用 AC #2 的註解＋AC #4(b) 的測試守住 `min-w-0`，所以不需要。
- 實測換行後的樣子（320 寬、修正後）：`[bitsearch.to]` ／ `Predator.Badlands.2` ／ `025.2160p.MA.WEB-` ／ `DL.DV.HDR.TYMBLER` ／ `.mkv`——在連字號後面會優先斷（瀏覽器本來就把 `-` 當斷點）。

### 不要做的事

- 不要動 `DetailTechInfoV2` 的路徑列、媒體庫卡片／列表、「管理字幕」sheet 標題——都量過、沒有讓頁面左右滑（見「查到的事」）。
- 不要為了這張改手機 hero 的版面（例如拿掉海報縮圖）——見 Discovery Triage。
- 不要用 `overflow-x: hidden` 蓋掉頁面捲動：那只是把溢出的字藏起來，片名後半段會被切掉看不到。
- 不要 commit `-linux.png`、不要跑 `test:visual:update`（會重拍全部）。

### 已知陷阱

- 原文片名只有「跟片名不一樣」才出現（`:138`）；E2E 第二部的片名與原文片名要不一樣，否則量不到原文片名那一行。
- `seedMovie` 沒有 `tmdbId` 時 `parse_status` 是空字串、不會出「沒有找到這部電影的資料」區塊；第一部用 `parseStatus: 'failed'` 才是正式機的真實樣子（失敗列）。
- 視窗寬度切換要 `page.setViewportSize` 後**再** `goto`（或 `reload`），避免沿用上一個寬度的排版；Linux CI 的捲軸會吃寬度，所以一律跟 `clientWidth` 比、不要跟 viewport 數字比（同 `library-list.spec.ts:79-91` 的做法）。
- E2E 有真的後端：片名加 `Date.now()` 避免撞名；`afterEach` 用 `deleteMovies` 清掉（同區塊既有寫法）。

### 對 data-testid／E2E／視覺基準的影響

- **data-testid**：不新增、不改名、不刪除。
- **E2E**：只新增測試；`seed-helpers.ts` 加一個選填欄位，既有呼叫不受影響。
- **視覺基準**：新增 1 張 darwin（AC #5）；既有預期 0 張變化；`-linux` 由 CI bootstrap PR 補。

### 測試標準

- vitest＋Testing Library（Rule 16 具體斷言）；排版只能靠 Playwright（Rule 28「Browser layout」列）。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`。

### Project Structure Notes

- 改：`apps/web/src/components/media/DetailHeroV2.tsx`（＋spec）、`apps/web/src/routes/test/-gallery.fixtures.tsx`、`tests/e2e/media-detail.spec.ts`、`tests/support/helpers/seed-helpers.ts`、新基準圖 1 張。沒有後端改動、沒有新元件。
- **Story Splitting Rule**：只有前端（3 個 task 動程式），不拆。

### Time-dependent visual coverage

- N/A — `DetailHeroV2` 不讀時間。

### References

- [Source: apps/web/src/components/media/DetailHeroV2.tsx:1, 121, 135-142]
- [Source: apps/web/src/components/media/LocalDetailV2.tsx:308-317]、[TMDbDetailV2.tsx:73, 93]
- [Source: apps/web/src/components/media/DetailTechInfoV2.tsx:147-152]、[library/PosterCardV2.tsx:176]、[activity/ActivityRow.tsx:64, 101]
- [Source: _bmad-output/screenshots/flow-b-detail-v2/b10p-m.png（H2MRl）、b3p-m.png（SzNRb）]
- [Source: tests/e2e/media-detail.spec.ts:166-275]、[tests/e2e/library-list.spec.ts:79-98]、[tests/support/helpers/seed-helpers.ts:33-80]
- [Source: apps/api/internal/handlers/movie_handler.go:44-58]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:1767-1801]、[tests/visual/components.visual.spec.ts:236-250]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-font-sans-not-noto.md — Completion Notes「畫面比對」]
- [Source: project-context.md Rule 16、Rule 21、Rule 24、Rule 28]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **③ 建議 parent 決定要不要立案（Bob 建單時發現，未查到既有條目）** — 手機 hero 版面跟設計稿不一樣：B3p-M（`SzNRb`）與 B10p-M（`H2MRl`）手機稿**沒有海報縮圖**，片名佔滿整行；程式碼在手機上有一塊 96px 寬的海報縮圖（`DetailHeroV2.tsx:92-94`，`w-24 … sm:w-40`），片名只剩右邊那一欄。320 寬時「管理字幕」「修改資訊」按鈕的字會折成兩行（scratchpad 截圖）。查過 `dsr-2-flow-b-detail-v2.md` 只處理了縮圖陰影（對照表 #8），沒有裁定手機要不要縮圖。要 Sally 判斷：手機稿省略、還是手機真的不要縮圖。P3。**本單不處理**——修了它，長片名會有更多寬度，但不能取代 AC #1（平板 640–1024 一樣會撐寬）。

### File List

## ⚖️ Sally 追認（2026-10-19，開工前）

- 沒有空格的長片名**在字中間斷**，不用省略號。理由：比對失敗的片，片名就是檔名，使用者要看完整檔名才能判斷是哪一部；截掉反而看不出來。B10p-M 的「換行、不截斷」原則照舊。PR 上不用再等追認。
