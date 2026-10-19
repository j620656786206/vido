# Disc：修字幕清單 B1（前端・列表版面）——字幕狀態緊貼片名、中間不再空一大片，手機也看得到字幕狀態

Status: done

**Source:** 2026-10-19 `/impeccable critique 媒體庫 列表`（21／40，P0 手機看不到字幕狀態、P1 重點擺反、P1 失敗列死路、P2 繁中／中文、P3）＋ Alexyu 看正式機：「左邊有片名、年份跟類別，右手邊是評比星號跟字幕訊息，中間一大片空白，看起來很奇怪」「要不要每張卡片加 border，比較看得懂邊界」。⚖️ **1A**：列表＝修字幕清單；⚖️ **2B**：critique 全部一起處理；ASCII 草圖（表格欄位＋外框一道＋列間細線）Alexyu 回「可以」。
設計稿（定稿，`disc-2026-10-library-list-triage-design` done）：桌機 **A4p-D `b1H71g`**（`_bmad-output/screenshots/flow-a-browse-v2/a4p-d.png`）、手機 **A4p-M `EkB8g`**（`a4p-m.png`）、規則 `spec-note-a4p`、失敗詳情 **B10p-D `p3qEc`**（`flow-b-detail-v2/b10p-d.png`）。

⚖️ **SM 裁定（Bob）：** 原單拆成 A（後端排序）／B1（本單，列表本身）／B2（頁首計數、排序選項、P3 小地方）。切分線是**版面**：B1 只動「列表那個框」裡面的東西（表格、列、手機列），B2 動框外面（頁首、工具列、骨架）。B1 **不依賴 A**，可以先做或並行。這張的規模跟 `disc-2026-10-glossary-season-variants-card-b`（8 項範圍／9 條 AC）相當，再拆會讓同一個元件被兩張單子各動一次。

## 查到的事（2026-10-19，本 session 讀過的程式碼）

- **空白的來源**：`LibraryListRowV2.tsx:91` 片名區 `min-w-0 flex-1` 吃掉所有空間，規格＋字幕（`:103-112`）與評分（`:114-119`）被推到最右邊。
- **P0 手機看不到字幕狀態**：字幕徽章所在的整塊是 `hidden … sm:flex`（`:103`），評分也是 `hidden … sm:flex`（`:115`）。
- **規格是三顆泥金藥丸**：`TechPill` 用 `--accent-tint`／`--accent-text`（`:29-35`），違反 `DESIGN.md:732`「TechBadge 一律中性」；內容是原始值（`:52-54`）。
- **解析度存兩種格式**：ffprobe 存 `寬x高`（`apps/api/internal/services/ffprobe_service.go:244`，例：critique 量到的 `3840x2076`），NFO 存 `4K`／`1440p`／`1080p`…（`nfo_reader_service.go:312-330` 的 `resolveResolution`：高 ≥2160 或寬 ≥3840 且高 ≥1080 → 4K；高 ≥1440 或寬 ≥2560 且高 ≥1080 → 1440p；≥1080 → 1080p；≥720 → 720p；≥480 → 480p；其餘 `${高}p`）。
- **`apps/web/src/utils/resolutionLabel.ts` 是死碼**：只對完全相符的尺寸查表（`:1-15`，`3840x2076`、`3840x1608` 查不到就原樣回傳），全 repo 只有它自己的 spec import 它（`grep -rln resolutionLabel apps/web/src tests` → `resolutionLabel.ts`、`resolutionLabel.spec.ts`）。
- **縮圖沒圖時是空白**：`:87-89` 只有 `bg-tertiary`；海報卡沒圖時用 `filenameToGradient`＋`fallbackInitial`（`PosterCardV2.tsx:59`、`:100-120`）。
- **失敗列**：今天片名位置顯示原始標題（常是檔名），旁邊「失敗」藥丸（`:94-98`，來自 `deriveLifecycleStatus`，`libraryStatus.ts:57-72`）。B10p 的句子在 `DetailNoMetadataV2.tsx:62`、`:91`：「沒有找到這部**電影**的資料」／「…這部**影集**的資料」（名詞跟著類型走）。`filePath` 在列表 JSON 裡有（`types/library.ts:44`、`:112`；Go `file_path,omitempty`，`models/movie.go:309`、`series.go:79`）。
- **字幕徽章規則**（不改）：`deriveSubtitleStatus`（`libraryStatus.ts:139-194`）。「中文」＝⚖️ 2026-10-06 D2，算「有」、青碧（`:151-155`）。處理中（probing／extracting／translating）回 `null`（`:178-181`）——本單起列表寫灰字「不知道」。
- **列表容器**：`LibraryBrowseV2.tsx:825-841`，`flex flex-col gap-0.5`，每列自己 `rounded` ＋ hover 底色（`LibraryListRowV2.tsx:70`）。testid：`library-list-v2`、`list-row-v2-${id}`、`row-select-indicator-${id}`。
- **meta 用等寬字**：`:100` `font-mono`；A4p-D 的 `m` 只指定 Body 字級、沒指定等寬（設計稿提示詞 A-6），截圖也是一般字型。
- **頁面左右邊界**：`LibraryBrowseV2.tsx:556` `px-4 … sm:px-6` → 390 寬手機列表寬 358（設計 story 追認：照 16px，不照稿上的 12px）。
- **手機有工具列**：工具列 `LibraryBrowseV2.tsx:680` 沒有任何斷點隱藏，排序（`SortSelector.tsx:70`）、選取、`ViewToggle`（`:771`）在 <640 都看得到。A4p-M 沒畫這列——但拿掉的話手機就沒辦法切到列表。**本單不動工具列。**
- **列表寬度不只看視窗**：側欄 sm 起就出現、寬 240（`AppShellV2.tsx:63-65`、`AppSidebar.tsx:50`），篩選欄 lg 起 264（`FilterRailShell.tsx:61`）＋ gap 24（`LibraryBrowseV2.tsx:623`）。算下來：1440 開篩選欄 → 列表 864；1280 開篩選欄 → 704；1024 開篩選欄 → 448；768 → 480；640 → 352。A4p-D 只畫了 1440（篩選欄收起的樣子），**中間寬度設計沒畫** → 見下方 ⚖️ SM 提案。
- **container query 有前例**：`CandidateListPanel.tsx:1151-1154`（`@container`，列依清單寬度換版，不看視窗）。
- **提示框**：`components/ui/Tooltip.tsx:20-34`，觸發元素由 Base UI 合併（`render={children}`）。
- **測試現況**：`tests/` 沒有任何 spec 用到 `list-row-v2`／`library-list-v2`／`LibraryListRowV2`（`grep -rn "list-row-v2\|row-select-indicator\|library-list-v2\|LibraryListRowV2" tests` 0 筆）；單元測試 `LibraryBrowseV2.spec.tsx:166-167`、`:437-439` 用到 `library-list-v2`、`list-row-v2-a`／`-b` 與 `aria-pressed`。畫廊沒有列表夾具（`-gallery.fixtures.tsx` 只有 `library-view-toggle`、`library-sort-selector`…）。

⚖️ **SM 提案，Sally 2026-10-19 追認**（理由：規格是最不重要的欄，先讓它退場；字幕欄在任何寬度都在，符合 A4p 的「字幕狀態不藏」。不另補中間寬度的稿）： 列表依「列表自己的寬度」換版（`@container`），不看視窗：
- **≥ 672px（`@2xl`）＝ A4p-D 完整表格**：`48px │ 片名 minmax(200px,360px) │ 字幕 104px │ 評分 64px │ 規格 minmax(120px,1fr) │ › 20px`。片名欄最多 360，空間不夠先縮片名、規格至少留 120。
- **576–671px（`@xl`）＝ 少一欄**：拿掉「規格」欄與欄名，片名欄 `minmax(0,1fr)`，字幕／評分欄照舊。
- **< 576px ＝ A4p-M 三行列**（所有手機、1024 開篩選欄、768）。
理由：A4p-D 的欄寬加起來 700px，1280 開篩選欄就只剩 704、1024 只剩 448——照視窗斷點切一定有寬度會溢出或擠成一團。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要列表檢視一眼就看出每部片的字幕狀態（手機也一樣），比對失敗的片看得懂發生什麼事，
這樣我可以把列表當成「待修字幕清單」一路往下處理。

## Acceptance Criteria

1. **完整表格（列表寬 ≥ 672）。** 整個列表一個外框（1px `--border-subtle`、`--radius-md` 圓角、無底色），列本身**不加框、不加底色**；第一列是欄名列（片名／字幕／評分／規格；Label 12、600、`--text-muted`、高 36），欄名列與各列之間一條 `--border-subtle` 髮絲線；欄序與寬度照上方 ⚖️ 提案；列內距上下 10、左右 12，欄距 16；縮圖 48×72、`--radius-md`；片名 16／600 `--text-primary` 單行截斷；meta 14 `--text-secondary`（一般字型，不是等寬）；hover 底色 `--bg-secondary`。
   **量測（Playwright，1440×900、篩選欄收起）：** 每一列的字幕欄左緣 x 相同；字幕欄左緣 − 片名欄左緣 ≤ 360 + 16；規格文字之後到 › 之前才有空白。
2. **少一欄（576–671）。** 規格欄與欄名「規格」不出現；片名欄自動伸縮；字幕、評分欄位置每列對齊。量測：1200×900、篩選欄打開（列表 1200−240−48−264−24＝624）→ 這一層；畫廊夾具寬 640 也釘住這一層。（1280×900 開篩選欄＝704 → 走第 1 條的完整表格。）
3. **三行列（< 576，含所有手機）。** 沒有欄名列；每列：縮圖 40×60（`--radius-sm`）／第一行片名（14／600 單行截斷）／第二行 meta（12 `--text-secondary`）／第三行「字幕徽章或灰字『不知道』＋★評分」；欄距 12；右側 ›。**390×844：列表外框寬 358、每一列都看得到字幕狀態、`document.documentElement.scrollWidth === clientWidth`（沒有橫向捲動）。** 字幕狀態在任何寬度都不藏（拿掉 `hidden sm:flex`）。
4. **規格一段灰字。** `4K · H.265 · DTS`：`font-mono` 12 `--text-secondary`、不是藥丸、沒有泥金；`TechPill` 刪掉。解析度：`寬x高` 照 Go `resolveResolution` 同一組門檻轉成 4K／1440p／1080p／720p／480p（`3840x2076`→4K、`1920x800`→1080p、`1280x720`→720p），已經是標籤的（`4K`、`1080p`）原樣，`2160p`→4K；三項都沒有 → 「—」（`--text-muted`）。單行截斷，完整內容放 `title`。
5. **字幕欄。** 徽章顏色與字照 `deriveSubtitleStatus`（`libraryStatus.ts` 一行都不改）；藥丸內距上下 4、左右 10，12／600。回 `null`（不知道、處理中）→ 灰字「不知道」（12、`--text-muted`、不是藥丸），**不留白**。「中文」徽章滑過顯示提示框「字幕軌沒標示繁簡」（用 `ui/Tooltip`）。
6. **評分。** 實心星（維持現行 `fill-[var(--warning)]`，設計 story 追認）12×12＋`font-mono` 12 `--text-secondary` 一位小數；沒有評分 → 評分欄空著（欄位仍佔寬，對齊不跑）。
7. **比對失敗的列（`parseStatus === 'failed'`，⚖️ Sally 裁定 7）。** 第一行：與 B10p **同一句**（電影「沒有找到這部電影的資料」／影集「沒有找到這部影集的資料」，從 `DetailNoMetadataV2` 抽出共用函式，兩處讀同一份）＋「失敗」藥丸（`--error-tint`／`--error-text`，內距 2／6，12／600）——表格版在同一行片名後，三行版放第三行最前面。第二行：檔名（`filePath` 最後一段；沒有就用 `title`），`font-mono` 12 `--text-muted`，**從中間截斷、保留最後 12 個字元**（例：`[ItachiUchiha] Chainsaw Man – The Mov…Reze-hen.mkv`、手機 `[ItachiUchiha] Chain…Reze-hen.mkv`），完整檔名放 `title`。縮圖：`--bg-tertiary` 底＋ lucide `Film` 20×20 `--text-muted`。字幕欄照實顯示；沒評分就空；規格照實或「—」。**整列仍是一個連結**，點了進詳細頁（B10p 的手動選片在那裡），列裡不放第二個按鈕。
8. **有比對到但沒海報的縮圖**：用 `filenameToGradient(title)`＋`fallbackInitial(title)`（跟海報卡一樣），不再是空白方塊。
9. **整列的名字（螢幕報讀）。** 連結有 `aria-label`，依欄位順序：
   - 一般：「執法記錄，2026 · 恐怖，字幕：缺中文，評分 6.0，4K · H.265 · DTS」
   - 中文：「…，字幕：中文（字幕軌沒標示繁簡），…」
   - 不知道：「…，字幕：不知道，…」；沒評分、沒規格就略過那一段
   - 失敗：「沒有找到這部電影的資料，比對失敗，檔名 [ItachiUchiha] Chainsaw Man – The Movie – Reze-hen.mkv，字幕：不知道」
   星號、›、縮圖 `aria-hidden`。鍵盤焦點框在外框裁切內仍完整看得到（列用內縮的 focus ring，例如 `focus-visible:outline-offset-[-2px]`）。
10. **選取模式照舊。** 選取時每列最前面多一欄 20px 圓形指示（欄名列同位置留空），點列切換不跳頁、`aria-pressed` 正確；選中底色 `--accent-subtle`。`LibraryBrowseV2.spec.tsx:437-439` 不改仍綠。
11. **「整理中」**（`parseStatus === 'pending'`）的藥丸照舊顯示——表格版在片名後、三行版在第三行最前面；「已入庫」永遠不顯示。
12. **testid 不變**：`library-list-v2`（外框）、`list-row-v2-${id}`（列連結）、`row-select-indicator-${id}`；新增 `library-list-header`（欄名列）、`list-row-subtitle-${id}`、`list-row-spec-${id}`。
13. **畫廊夾具＋基準圖。** 新增三個夾具，資料照 A4p-D／A4p-M（執法記錄 缺中文 6.0 `4K · H.265 · DTS`／鏢人：風起大漠 不知道 8.1 —／Chainsaw Man 失敗列／牧師的最後誘惑 中文 7.0／阿凡達：火與燼 繁中 7.6 `4K · H.265 · E-AC-3`／捍衛任務：復仇芭蕾 繁中 7.3 `AAC`／F1電影 繁中 7.8 `1920x1080`→`1080p`／四海好傢伙 繁中 8.5 `AC-3`），`posterPath: null`（不打網路）：
   - `library-list`（寬 1200，A4p-D `b1H71g`，8 列，第 1 列不需 hover）
   - `library-list-narrow`（寬 640，576–671 那一層）
   - `library-mobile-screens/list`（viewport 390×844，A4p-M `EkB8g`，7 列）
   本機產 `-darwin.png`；**不產、不 commit `-linux.png`**（CI 的 Visual Regression 會開 bootstrap PR）。
14. `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check`、`python3 scripts/check-design-refs.py` 綠；新 E2E 本機跑綠。

## Tasks / Subtasks

- [x] **T1 小工具（AC #4、#7）**
  - [x] 把 `utils/resolutionLabel.ts` 改寫成 `resolutionTier(raw)`（門檻照 Go `resolveResolution`，註解指向 `nfo_reader_service.go:312`），刪掉查表；`audioChannelLabel` 沒人用，一起刪或保留看 lint；spec 改寫（`3840x2076`、`3840x1608`、`1920x800`、`1280x720`、`720x480`、`4K`、`1080p`、`2160p`、空字串）。
  - [x] `formatSpecLine(media)` → `'4K · H.265 · DTS'` 或 `null`（放 `components/library/` 旁或 `utils/`，`// Implements: <utility — no .pen counterpart>`）。
  - [x] 檔名中間截斷：拆成 `head`＋`tail`（最後 12 字元），用兩個 span（`truncate min-w-0`＋`shrink-0`）讓瀏覽器依寬度截，不用量字寬。
  - [x] 從 `DetailNoMetadataV2.tsx:62`、`:91` 抽出 `noMetadataHeading(mediaType)`，詳細頁與列表共用。
- [x] **T2 `LibraryListV2.tsx`（新檔，AC #1–#3、#10、#12）**：外框＋`@container`＋欄名列＋列（`divide-y`），欄寬用一個共用的 grid template 常數（欄名列與每列同一份，才會對齊）；選取模式的指示欄。檔頭 `// Design ref: ux-design.pen Screen A4p-D (b1H71g) · A4p-M (EkB8g)`。
- [x] **T3 改寫 `LibraryListRowV2.tsx`（AC #3–#9、#11）**：表格版／三行版、失敗列、不知道、中文提示框、`aria-label`、縮圖後備、內縮焦點框；刪 `TechPill`；檔頭改 `// Design ref: ux-design.pen Screen A4p-D (b1H71g) · A4p-M (EkB8g)`。
- [x] **T4 接線（AC #1、#10）**：`LibraryBrowseV2.tsx:825-841` 換成 `<LibraryListV2 …>`；`toDisplay`（`:123-149`）視需要多帶 `mediaType`。不動頁首、工具列、骨架（B2）。
- [x] **T5 單元測試（AC #3–#12）**：`resolutionLabel.spec.ts`、`LibraryListRowV2.spec.tsx`（新：每種徽章／不知道／處理中→不知道／失敗列兩種類型／中間截斷保留副檔名／aria-label 四種／中文提示框／沒評分欄位仍在／整理中藥丸）、`LibraryListV2.spec.tsx`（欄名列、選取模式）、`DetailNoMetadataV2.spec` 仍綠。
- [x] **T6 畫廊夾具＋darwin 基準（AC #13）**：照 `library-mobile-screens/skeleton`（`-gallery.fixtures.tsx:3641-3650`）的寫法加三個夾具；`penNode` 填 `b1H71g`／`EkB8g`。
- [x] **T7 E2E（AC #1–#3）**：新 `tests/e2e/library-list.spec.ts`，用 `stubLibraryBaseline`＋`stubLibraryList`（`tests/support/helpers/library-stubs.ts`；`movieItem` 加可選覆寫：`parse_status`、`file_path`、`video_resolution`、`video_codec`、`audio_codec`、`vote_average`）。量：390（外框 358、每列字幕元素可見、沒有橫向捲動）、1200 開篩選欄（624 → 少一欄、沒有規格欄）、1280 開篩選欄（704 → 完整表格、字幕欄 x 對齊）、1440 篩選欄收起（同上＋規格欄出現）。跑：`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-e2e-dummy npx playwright test tests/e2e/library-list.spec.ts --project=chromium --reporter=line`。
- [x] **T8 Step 9 設計比對**：截圖對 `a4p-d.png`、`a4p-m.png`；跑 AC #14 的閘門。

## Dev Notes

### 不要做的事

- 不要改 `libraryStatus.ts`（`deriveSubtitleStatus`／`pickPosterBadge` 的顏色與字，⚖️ 2026-10-06 D2）。「中文」的疑慮只用提示框處理。
- 不要每列加框或加底色（⚖️ Sally 裁定 1、`DESIGN.md:679`「同一層級只准一道框」）。
- 不要在列裡放第二個點擊目標（手動選片按鈕、⋯ 選單）。
- 不要拿掉手機工具列（見「查到的事」）。
- 不要動頁首計數、排序選項、檢視切換大小、載入骨架、網址參數——那是 B2。
- 不要用 `hidden sm:*` 這種**視窗**斷點決定欄位，用 `@container`（理由見 ⚖️ 提案）。
- 不要 commit `-linux.png`；不要跑 `test:visual:update` 去更新 linux 基準。

### 已知陷阱

- **焦點框被外框裁掉**：外框要 `overflow-hidden`（圓角裡的 hover 底色才不會凸出），但全站 `:focus-visible` 是 `outline-offset: 2px`（`apps/web/src/styles.css:552-554`）→ 列的焦點框會被裁。列自己改成內縮。
- **字幕欄要用 grid 對齊**，不是每列各自 flex——flex 會讓每列的字幕欄跟著片名長度跑。
- **處理中的列**：`deriveSubtitleStatus` 回 `null` → 寫「不知道」。這是刻意的（處理中由活動頁負責，`libraryStatus.ts:13-21`），不是 bug。
- **影集的 `filePath` 可能是資料夾**：最後一段沒有副檔名也照樣截，`tail` 就是最後 12 個字元。
- **meta 不再用等寬字**（A4p-D 的 `m` 沒指定等寬）；年份數字用一般字型。規格、評分、檔名才用 `font-mono`。（`font-mono` 目前其實沒吃到 JetBrains Mono，那是另一張單，見 B2 的 Discovery Triage。）
- Tooltip 包在連結裡的 `<span>` 上：滑鼠滑過會開；鍵盤使用者靠整列的 `aria-label` 拿到同一句話。不要給徽章 `tabIndex`（連結裡不能再有可聚焦元素）。

### 測試標準

- vitest＋Testing Library；斷言用具體值（Rule 16，不准 `toBeTruthy`）；className 斷言證明不了 CSS 真的生效，版面一律用 E2E 量（Rule 28「Browser layout」列）。
- 改完 `grep -rn "list-row-v2\|library-list-v2\|row-select-indicator" tests apps/web/src` 再確認一次（本 session 查：`tests/` 0 筆）。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`。

### Project Structure Notes

- 新檔：`apps/web/src/components/library/LibraryListV2.tsx`（＋spec）。改：`LibraryListRowV2.tsx`、`LibraryBrowseV2.tsx`（只換列表那段）、`utils/resolutionLabel.ts`、`components/media/DetailNoMetadataV2.tsx`（抽函式）、`routes/test/-gallery.fixtures.tsx`、`tests/support/helpers/library-stubs.ts`、新 `tests/e2e/library-list.spec.ts`。
- Rule 21：兩個元件檔頭都要指向 `b1H71g`／`EkB8g`（`scripts/export-pen-screenshots.py:282-283` 已有這兩個節點）。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（`LibraryListV2`／`LibraryListRowV2` 不讀時間；`LibraryBrowseV2` 已是 `Time-bomb-exempt`，本單不新增時間讀取）。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-design.md — Sally 裁定 1–8、Dev Agent Record（358、實心星）]
- [Source: .impeccable/critique/2026-10-19T00-18-51Z__ps-web-src-components-library-librarylistrowv2-tsx.md]
- [Source: DESIGN.md:679（一層一道框）、:732（TechBadge 中性）、:368（等寬字用在讀數）]
- [Source: apps/web/src/components/library/LibraryListRowV2.tsx:29-123]、[LibraryBrowseV2.tsx:123-149, 556, 623, 680, 825-841]
- [Source: apps/web/src/utils/libraryStatus.ts:139-194]、[components/media/DetailNoMetadataV2.tsx:62, 91]
- [Source: apps/api/internal/services/nfo_reader_service.go:312-330]、[ffprobe_service.go:244]
- [Source: _bmad-output/pen-tokens.json — Space/lg 16、md 12、sm-plus 10、xs 4、2xs 2；Type BodyLg 16／Body 14／Label 12]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，BMAD dev-story），2026-10-19。

### Debug Log References

- `nx test web`：293 檔／4,587 測試全過；`nx test api`：全過。
- `lint:all`：✖ 170 problems (0 errors, 170 warnings)——全是既有警告，本單改到的檔案 eslint 0 筆；`web:typecheck` 過；`format:check` 過；`check-design-refs.py`：388 個檔頭節點都在。
- E2E `tests/e2e/library-list.spec.ts` 7 個全綠（burn-in `--repeat-each=5` 35/35）；`library-mobile.spec.ts`（共用 stub 有改）11 個仍綠。
- Visual：三個新夾具本機產 `-darwin.png`；全套 `test:visual` 綠（既有基準無差異）。

### Completion Notes List

- 🔗 AC Drift: NONE（checked: `list-row-v2|library-list-v2|row-select-indicator|LibraryListRowV2|resolutionLabel|audioChannelLabel` across `_bmad-output/implementation-artifacts/*.md`、`tests/`、`apps/web/src/routes/test`——命中 ux2-2／ux3-cutover-2 的列表列與 `disc-2026-10-detail-resolution-raw`，皆 REUSE：testid、`aria-pressed` 選取行為不變；`resolutionLabel.ts` 檔名與 `audioChannelLabel` 保留給 detail-resolution-raw 用）。
- 📎 Contract Stamps: NONE（本單不定義也不消費 `[@contract-v*]`；只讀既有列表 JSON 欄位）。
- 🎭 A11y Pre-Flight: PASS（2 components checked，0 jsx-a11y warnings on touched files，0 introduced）。整列一個連結＋依欄位順序的 `aria-label`；星號、›、縮圖、重複的生命週期藥丸 `aria-hidden`；欄名列 `aria-hidden`；中文提示框的觸發元素不可聚焦（Base UI 不加 tabIndex）；焦點框內縮，四角的列跟外框一起圓角。
- **實作：** 一份 DOM、CSS grid；三行版第三行在 `@xl` 變 `display: contents`，字幕／評分兩格落進表格欄。欄寬集中在 `libraryListCols.ts`（欄名列與每列共用）。`@container` 放在外框**外面**的包裝層，門檻量的是整個列表寬（外框 1px 不會讓門檻偏 2px）。
- **偏差 1（片名欄下限）：** 用 `minmax(0,360px)`，不是 `minmax(200px,360px)`——選取模式下列表 672 寬時，20px 指示欄＋16 間距讓片名只剩 174，200 的下限會讓列溢出外框 26px。非選取模式 ≥672 時片名本來就 ≥210，行為不變。
- **偏差 2（解析度）：** AC #4 同時寫「照 Go `resolveResolution`」和「`1920x800`→1080p」，兩者衝突（Go 會判 720p）。照 AC 的例子：在 1080p／720p 兩級補上同型的寬度補判（`寬≥1920 且 高≥720`→1080p、`寬≥1280 且 高≥480`→720p）。正式機 2026-10-19 有 10 部 `1920x80x`／`1920x1036` 的片，不補會顯示 720p。Go 端對齊另立 `disc-2026-10-resolution-tier-scope-encodes`。
- **`resolutionLabel.ts` 保留檔名與 `audioChannelLabel`：** `disc-2026-10-detail-resolution-raw`（blocked-by 本單）要在同檔加 `formatResolutionBadge` 並沿用 `audioChannelLabel`；查表 `resolutionLabel()` 刪除、換成 `resolutionTier()`。
- **檔名中間截斷：** head 用 `whitespace-pre`（不是 `truncate` 的 `nowrap`）——head 常以空白結尾（「… – 」），nowrap 會吃掉區塊尾端空白、把兩半黏在一起。
- **處理中**的列照 Dev Notes 寫「不知道」；「整理中」藥丸表格版在片名後、三行版在第三行最前面，並念進 `aria-label`。
- **對抗式 CR（獨立 reviewer，9 項：0 HIGH／2 MED／7 LOW）已修：** MED-1 程式註解引用的後續單不存在 → 已立案（見下）；MED-2 門檻 E2E 可能沒量到真正的 575／576／671／672 → 改成依 client width 設視窗、`expect(seen).toEqual([575,576,671,672])`；LOW-3 縮圖與 › 在表格版靠自動排位 → 補 `@xl:row-start-1`；LOW-4 最後一列（手機也含第一列）焦點框四角被外框裁掉 → 角落的列跟外框同圓角，E2E 量 `border-bottom-left-radius`；LOW-5 中文提示框靠標籤文字判斷 → 改看 `chineseSubtitle === 'zh'`；LOW-6 兩個單元測試名不副實 → 改名／標成 smoke，「沒評分仍佔欄寬」改由 E2E 量（失敗列的規格 x＝第一列規格 x）；LOW-9 選中列滑過時選取底色消失 → hover 底色只給未選中列。
- **CR 已知、不修：** LOW-7 手機夾具 390 視窗下畫廊自帶 p-8，框寬 326、App 底部分頁列蓋住第 7 列（既有 `library-mobile-screens/skeleton` 同樣情況）；358 與 7 列由 E2E 釘住。LOW-8 檔名「…」後面看起來多一格空白：瀏覽器在最後一個放得下的字後面放「…」，剩下不足一字寬的空間留白；等 `disc-2026-10-font-mono-not-jetbrains` 換成真正的等寬字再看。
- 🎨 UX Verification: PASS — 截圖在 scratchpad `b1-ux/`（390／768／1024 開篩選欄／1200 開篩選欄／1280 開篩選欄／1440 收起）。桌機 1440 對 `a4p-d.png`：外框一道、欄名列、髮絲線、片名 360 → 字幕 104 → 評分 64 → 規格 → ›，空白只在規格後；失敗列句子＋失敗藥丸＋中間截斷檔名＋膠捲縮圖，一致。手機 390 對 `a4p-m.png`：外框 358（照設計 story 追認的 16px 頁邊，不是稿上的 366）、三行、每列都有字幕狀態、失敗列第三行「失敗＋不知道」，一致。差異（皆已裁定）：星號實心（設計 story 追認）；手機工具列保留（Dev Notes）；E2E 截圖是日巡主題、稿是夜行（畫廊基準是夜行）。斷點實測：1440 收起＝1152 完整表格；1280 開＝704 完整表格（片名 242、規格 120 都截斷，照 ⚖️「先縮片名、規格至少 120」）；1200 開＝624 少一欄；1024 開＝448、768＝480、390＝358 三行；門檻兩側 575／576／671／672 各自換版、無橫向捲動。

### Discovery Triage

- **① expand-scope-in-place** — `resolutionLabel.ts` 是死碼且查表方式錯（見「查到的事」）→ T1 改寫成 `resolutionTier`，由 AC #4 追蹤。
- **① expand-scope-in-place** — A4p-M 沒畫手機工具列，但程式碼有、而且是手機切到列表的唯一入口 → 在「不要做的事」與 AC 範圍明訂保留，不另開單。
- **③ backlog-with-carry-forward-link** — `disc-2026-10-detail-resolution-raw`：詳細頁「檔案資訊」的解析度徽章直接顯示原始值（`DetailTechInfoV2.tsx:90-96`，例 `3840x2160`），B10p 畫的是 `4K · 3840×2160`。本單做出 `resolutionTier` 之後可直接套用；不在本單做是因為會動詳細頁的視覺基準。
- **③ backlog** — `disc-2026-10-resolution-tier-scope-encodes`（dev-story 時發現）：Go `resolveResolution` 只有 4K／1440p 有寬度補判，NFO 來源的 1080p／720p 寬銀幕片會被標成低一級，跟前端 `resolutionTier` 不一致；另 `3828x1588`（正式機 1 部）兩邊都判 1440p，是否該算 4K 待裁定。
- **③ backlog** — `disc-2026-10-list-zh-hint-touch`（dev-story 時發現）：「中文」提示框只在滑鼠滑過時出現；手機一按徽章就整列跳到詳細頁，觸控使用者看不到「字幕軌沒標示繁簡」（螢幕報讀有，因為在 aria-label 裡）。

### File List

- `apps/web/src/components/library/LibraryListV2.tsx`（新）
- `apps/web/src/components/library/LibraryListV2.spec.tsx`（新）
- `apps/web/src/components/library/libraryListCols.ts`（新）
- `apps/web/src/components/library/LibraryListRowV2.tsx`（改寫）
- `apps/web/src/components/library/LibraryListRowV2.spec.tsx`（新）
- `apps/web/src/components/library/LibraryBrowseV2.tsx`（只換列表那段）
- `apps/web/src/components/media/DetailNoMetadataV2.tsx`（改用 `noMetadataHeading`）
- `apps/web/src/utils/resolutionLabel.ts`、`resolutionLabel.spec.ts`（查表 → `resolutionTier`）
- `apps/web/src/utils/libraryListRow.ts`、`libraryListRow.spec.ts`（新）
- `apps/web/src/utils/noMetadataHeading.ts`、`noMetadataHeading.spec.ts`（新）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（三個夾具）
- `tests/visual/components.visual.spec.ts-snapshots/components/library-list/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/library-list-narrow/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/library-mobile-screens/list/default-visual-darwin.png`（新）
- `tests/e2e/library-list.spec.ts`（新）
- `tests/support/helpers/library-stubs.ts`（`movieItem` 加可選覆寫）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
