# Disc：詳細頁「檔案資訊」的解析度寫成「4K · 3840×2160」，聲道寫成「7.1」，跟設計稿一樣

Status: ready-for-dev

**Blocked-by:** `disc-2026-10-library-list-triage-b1`（b1 的 T1 把 `utils/resolutionLabel.ts` 改寫成 `resolutionTier(raw)`；本單直接用它）。

**Source:** Rule 24 lane ③，2026-10-19 由 `disc-2026-10-library-list-triage-b1` 建單時發現（見 b1 Discovery Triage）。⚖️ **Alexyu 2026-10-19：要修。**
原單描述：詳細頁「檔案資訊」解析度徽章顯示原始值（例 `3840x2160`），B10p 畫 `4K · 3840×2160`。

⚖️ **SM 裁定（Bob）— 等 b1，不把小工具搬來這張：** `resolutionTier` 的規格、測試、門檻（照 Go `resolveResolution`）都已經寫在 b1 的 AC #4／T1，b1 排在本單前面。搬過來的話 b1 要改 AC、改 T1、兩張的 spec 檔會互踩；等 b1 合併的成本是零（本單是 P3、不急）。本單只在同一個檔案**加**一個 `formatResolutionBadge`，不改 `resolutionTier`。如果 Alexyu 想讓本單先做：把 b1 T1 第一個子項整個搬來本單、b1 改成「blocked-by 本單」，兩張的其他內容不變。

## 查到的事（2026-10-19，本 session 讀過的程式碼、設計稿與正式機資料）

- **徽章直接用原始值**：`DetailTechInfoV2.tsx:91-96` 把 `videoResolution`、`videoCodec`、`audioCodec＋聲道`、`hdrFormat` 原樣放進徽章；聲道寫成 `${audioChannels}ch`（`:94`）。徽章是 `font-mono text-xs`、中性底色（`:51-60`，dsr-2 AC #4）。
- **設計稿**：
  - 桌機 B3p-D（`uRGu2`，`_bmad-output/screenshots/flow-b-detail-v2/b3p-d.png`）、B10p-D（`p3qEc`，`b10p-d.png`）：`4K · 3840×2160`、`H.265`、`TrueHD 7.1`、`HDR10`。乘號是 `×`（U+00D7），不是英文字母 x。
  - 手機 B3p-M（`SzNRb`，`b3p-m.png`）、B10p-M（`H2MRl`，`b10p-m.png`）：只寫 `4K`、`H.265`、`TrueHD 7.1`——**兩張手機稿都沒有尺寸**。（節點 id 見 `scripts/export-pen-screenshots.py:291`、`:296`、`:299`、`:300`。）
- **資料的真實形狀**：ffprobe 寫 `寬x高`（`apps/api/internal/services/ffprobe_service.go:243-244`），NFO 寫 `4K`／`1080p` 這類標籤（`nfo_reader_service.go:312-330`）。正式機 2026-10-19 電影的 `video_resolution`：`3840x2160` 17、空 11、`1920x804` 6、`3840x1608` 4、`3840x1600` 3、`3840x2076`／`3840x1606`／`3840x1604`／`1920x1080` 各 2、`3828x1588`／`2880x2160`／`1920x802`／`1920x800`／`1920x780`／`1920x1036` 各 1——**全部都是 ffprobe 形狀**；影集 81 部全空。聲道：2、6、8 三種；影片編碼存的是 `H.265`／`H.264`（`ffprobe_service.go:353-361` 的 `videoCodecMap`），不是 `HEVC`。
- **聲道標籤已經有一份沒人用的對照**：`audioChannelLabel`（`apps/web/src/utils/resolutionLabel.ts:17-26`：1→Mono、2→Stereo、6→5.1、8→7.1），全 repo 只有它自己的 spec 在 import（`grep -rn "resolutionLabel\|audioChannelLabel" apps/web/src tests`）。b1 的 T1 寫「`audioChannelLabel` 沒人用，一起刪或保留看 lint」→ 本單要用它，見 T1。
- **用到這個元件的地方**：只有 `LocalDetailV2.tsx:291-300`；測試 `DetailTechInfoV2.spec.tsx:21` 斷言 `'DTS 6ch'`、`:32` 用 `videoResolution="2160p"`；`LocalDetailV2.spec.tsx:153` 種 `'1080p'`、`:447` 斷言 `toHaveTextContent('1080p')`（標籤原樣，改完仍綠）。`tests/`（E2E）沒有任何地方用 `detail-tech-*` testid（`grep -rn "detail-tech" tests` 0 筆）。
- **畫廊夾具＋基準圖**：只有一個夾具 `media-detail-tech-info-v2`（`routes/test/-gallery.fixtures.tsx:1675-1692`，寬 720，`videoResolution: '2160p'`、`videoCodec: 'HEVC'`、`TrueHD`／8 聲道），基準圖 `tests/visual/components.visual.spec.ts-snapshots/components/media-detail-tech-info-v2/default-visual-darwin.png`＋`default-visual-linux.png`。visual 專案的視窗是 1280×800（`playwright.config.ts:165`），夾具可用 `viewport` 指定手機（例：`-gallery.fixtures.tsx:3641-3650` 的 `library-mobile-screens/skeleton`）。
- 元件檔頭是 `// Design ref: ux-design.pen Screen B3p-D (uRGu2)`（`DetailTechInfoV2.tsx:1`）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要詳細頁的檔案資訊寫「4K · 3840×2160」「TrueHD 7.1」這種我一眼看得懂的寫法，
這樣我不用自己心算 3840x1608 算不算 4K、8ch 是不是 7.1。

## Acceptance Criteria

1. **解析度徽章的字（桌機，視窗 ≥ 640）。** 新函式 `formatResolutionBadge(raw)`（放 `utils/resolutionLabel.ts`，用 b1 的 `resolutionTier`）：
   | 輸入 | 顯示 |
   |---|---|
   | `3840x2160` | `4K · 3840×2160` |
   | `3840x2076` | `4K · 3840×2076` |
   | `2880x2160` | `4K · 2880×2160` |
   | `3828x1588` | `1440p · 3828×1588`（照 Go `resolveResolution`：寬 <3840、高 ≥1440） |
   | `1920x804` | `1080p · 1920×804` |
   | `1280x720` | `720p · 1280×720` |
   | `4K`、`1080p` | 原樣 |
   | `2160p` | `4K` |
   | 空字串／沒有值 | 不出這顆徽章（跟今天一樣） |
   | 認不得的字串（例 `abc`） | 原樣 |
   尺寸用 `×`（U+00D7），兩邊不空格；等級與尺寸之間是 ` · `（U+00B7，前後各一個空格），跟 `LibraryListRowV2` 規格那行同一個分隔符。
2. **手機（視窗 < 640）只寫等級。** 照 B3p-M／B10p-M：`4K`。做法：等級一個 `<span>`、` · 3840×2160` 另一個 `<span className="hidden sm:inline">`；徽章本身的 `title` 放完整字串（`4K · 3840×2160`），滑鼠停留看得到。
3. **聲道寫成聲道配置。** `audioCodec` 後面接 `audioChannelLabel(n)`：2→`2.0`、6→`5.1`、8→`7.1`、1→`1.0`，其他數字維持 `${n}ch`；沒有聲道數就只寫編碼。例：`TrueHD 7.1`、`DTS 5.1`、`AAC 2.0`。（`audioChannelLabel` 的對照從 Mono／Stereo 改成數字，理由：設計稿寫 `7.1`，同一排徽章都是等寬數字讀數；正式機只有 2／6／8 三種。）
4. **其他徽章與事實列不變**：影片編碼、HDR、檔案大小、字幕軌、路徑的內容與樣式一行不改；徽章仍是中性 `bg-tertiary`／`text-secondary`／`rounded-full`（`DetailTechInfoV2.spec.tsx:31-41` 不改仍綠）。
5. **單元測試。** `resolutionLabel.spec.ts` 加 AC #1 整張表與 AC #3 的對照；`DetailTechInfoV2.spec.tsx`：`'DTS 6ch'` 改 `'DTS 5.1'`，新增「`3840x2160` → 徽章文字含 `4K` 與 `3840×2160`、`title` 是 `4K · 3840×2160`、尺寸那個 span 帶 `hidden sm:inline`」；`LocalDetailV2.spec.tsx:447`（`1080p`）不改仍綠。斷言用具體字串（Rule 16）。
6. **畫廊夾具＋基準圖。**
   - `media-detail-tech-info-v2` 的 props 改成正式機的真實形狀（Rule 28）：`videoResolution: '3840x2160'`、`videoCodec: 'H.265'`、`audioCodec: 'TrueHD'`、`audioChannels: 8`、`hdrFormat: 'HDR10'`（其他不動）→ 截圖的徽章列是 `4K · 3840×2160`／`H.265`／`TrueHD 7.1`／`HDR10`，對 `b3p-d.png`。本機重產 `default-visual-darwin.png`，**`git rm` 舊的 `default-visual-linux.png`**（CI 的 Visual Regression 會開 bootstrap PR 補，不要自己產、不要跑 `test:visual:update` 去動 linux）。
   - 新夾具 `media-detail-tech-info-v2-mobile`（同一組 props、`viewport: { width: 390, height: 844 }`、`penNode: 'SzNRb'`）→ 徽章列是 `4K`／`H.265`／`TrueHD 7.1`／`HDR10`；只產 darwin。
7. **Rule 21 檔頭**：`DetailTechInfoV2.tsx:1` 改成 `// Design ref: ux-design.pen Screen B3p-D (uRGu2) · B3p-M (SzNRb)`；`python3 scripts/check-design-refs.py` 綠。
8. `nx test web`、`lint:all`（看「✖ N problems (X errors」那行）、`typecheck`、`format:check` 綠；本機 visual 跑 `media-detail-tech-info-v2` 兩個夾具綠。

## Tasks / Subtasks

- [ ] **T1 小工具（AC #1、#3）**：b1 合併後，在 `utils/resolutionLabel.ts` 加 `formatResolutionBadge(raw): { tier: string; dims: string | null } | null`（或回字串＋另一個回尺寸的函式，擇一，元件要能把尺寸包成另一個 span）；`audioChannelLabel` 的對照改成數字配置。**若 b1 已經把 `audioChannelLabel` 刪掉**，在同檔加回來（同名、同簽名）。spec 照 AC #5。
- [ ] **T2 元件（AC #1–#4、#7）**：`DetailTechInfoV2.tsx` 解析度徽章改用 T1（兩個 span＋`title`）、聲道改用 `audioChannelLabel`；`Badge` 的 `key` 改用穩定字串（今天用內容當 key，`4K · …` 帶 JSX 後不能再當 key——用 `'res'`／`'vcodec'`／`'audio'`／`'hdr'`）；檔頭照 AC #7。
- [ ] **T3 測試＋基準圖（AC #5、#6、#8）**：改兩個 spec；改夾具 props、加手機夾具；`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-visual-dummy pnpm run test:visual --reporter=line -g media-detail-tech-info-v2`（或刪掉舊 darwin 再跑 `test:visual:update-missing`）；`git rm` 舊 `-linux.png`。
- [ ] **T4 Step 9 設計比對**：截圖對 `b3p-d.png`（桌機）與 `b3p-m.png`（手機）的「檔案資訊」徽章列。

## Dev Notes

### 不要做的事

- 不要改 `resolutionTier` 的門檻或行為（b1 的範圍，照 Go `resolveResolution`）。
- 不要把手機的 HDR 徽章或「檔案大小／字幕軌／路徑」拿掉——手機稿沒畫它們，但那是另一個問題（見 Discovery Triage），本單只改解析度與聲道的寫法。
- 不要改徽章顏色、圓角、字級（dsr-2 AC #4 的中性規則）。
- 不要動 `LibraryListRowV2` 的規格那行（b1 用 `resolutionTier` 只寫等級，例 `4K · H.265 · DTS`，列表裡不放尺寸——那是 A4p 的設計）。
- 不要 commit `-linux.png`。

### 已知陷阱

- `×` 與 `·` 都是非 ASCII：測試字串直接寫字元，不要用 `x`／`.` 代替；`font-mono` 目前是系統等寬字，兩個字元都有。（等寬字型之後會換成 JetBrains Mono——`disc-2026-10-font-mono-not-jetbrains`——兩個字元都在它的 latin 子集內，`U+0000-00FF`。）
- `hidden sm:inline` 是**視窗**斷點，jsdom 不會套用 CSS——單元測試只能斷言 className；手機真的只看到 `4K` 由手機夾具的截圖證明（Rule 28「Browser layout」列）。
- 檔案大小：B3p-D 寫 `8.42 GB`、B10p-D 寫 `4.2 GB`——兩張稿自己不一致，程式碼 `toFixed(1)`（`DetailTechInfoV2.tsx:28-33`）跟 B10p 一致，**不要改**。

### 測試標準

- vitest＋Testing Library，斷言具體值（Rule 16）。
- 動到文字前已查：`grep -rn "detail-tech\|6ch" tests` 0 筆；vitest 內 `'DTS 6ch'` 只在 `DetailTechInfoV2.spec.tsx:21`。
- 本機 visual 要帶 `AI_PROVIDER=claude`；macOS 沒有 `timeout`。

### Project Structure Notes

- 改：`apps/web/src/utils/resolutionLabel.ts`（＋spec）、`components/media/DetailTechInfoV2.tsx`（＋spec）、`routes/test/-gallery.fixtures.tsx`、基準圖。沒有新元件、沒有後端改動。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（`DetailTechInfoV2` 不讀時間）。

### References

- [Source: _bmad-output/screenshots/flow-b-detail-v2/b3p-d.png、b10p-d.png、b3p-m.png、b10p-m.png]
- [Source: apps/web/src/components/media/DetailTechInfoV2.tsx:1, 28-33, 51-60, 89-98]
- [Source: apps/web/src/utils/resolutionLabel.ts:1-26]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-b1.md — AC #4、T1（`resolutionTier`）]
- [Source: apps/api/internal/services/ffprobe_service.go:243-251, 353-361]、[nfo_reader_service.go:312-330]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:1675-1692]、[playwright.config.ts:165]
- [Source: project-context.md Rule 16、Rule 21、Rule 24、Rule 28]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **① expand-scope-in-place** — 聲道寫成 `8ch`，設計稿寫 `7.1`（同一排徽章、同一張基準圖）→ AC #3 追蹤。
- **③ backlog-with-carry-forward-link** — `disc-2026-10-detail-techinfo-mobile-vs-design`（Bob 建單時發現；**需要 parent 在 sprint-status 新增這一行**）：手機稿 B3p-M／B10p-M 的「檔案資訊」只有三顆徽章（沒有 HDR10、沒有檔案大小／字幕軌／路徑三列），程式碼在手機上全部顯示（`DetailTechInfoV2.tsx` 沒有任何斷點）。要 Sally 判斷：是手機稿省略沒畫，還是手機真的要精簡。P3。

### File List
