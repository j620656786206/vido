---
target: homepage
total_score: 20
max_score: 40
na_heuristics: 
p0_count: 2
p1_count: 2
timestamp: 2026-08-25T14-20-07Z
slug: apps-web-src-components-homepage
---
Method: dual-agent (A: design-review subagent · B: detector/browser subagent)

# /impeccable critique homepage — 2026-08-25（run 1 for this slug）

Target: `apps/web/src/components/homepage/`（HomeBrowseV2 + HeroBanner + ContinueWatchingSlot + ExploreBlock(s) + RecentlyAddedRowV2 + TrailerModal）＋ `routes/index.tsx`。實測 http://localhost:8090/（真實降級態：TMDb 500）＋ mock 完整態，桌機 1440×900、手機 390×844。

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | 進行中 · N 只數「最近 20 筆」內的 pending（RecentlyAddedRowV2.tsx:70）；TMDb 掛時使用者配置的 Explore 區塊無聲消失 |
| 2 | Match System / Real World | 3 | 徽章以「怎麼救」命名（極好）；但「查看更多」連到不帶 filter 的 /search（ExploreBlock.tsx:260-267） |
| 3 | User Control and Freedom | 2 | 輪播 8s 自轉僅 hover 可暫停；手機永不停（WCAG 2.2.2 fail） |
| 4 | Consistency and Standards | 1 | 同頁兩套海報卡（PosterCardV2 vs 舊 PosterCard）：圓角、評分位置、標題行數全相反 |
| 5 | Error Prevention | 3 | 無破壞性操作；fail-soft；TrailerModal focus trap + Esc |
| 6 | Recognition Rather Than Recall | 2 | 10 個徽章詞彙無圖例、無 tooltip |
| 7 | Flexibility and Efficiency | 2 | hover 預取、lazy 區塊好；輪播圓點 8px 高，遠低於 44px |
| 8 | Aesthetic and Minimalist Design | 2 | 下半部三個 surface 都是同一份 TMDb trending；影集列每卡再蓋「影集」chip ×8 純噪音 |
| 9 | Error Recovery | 2 | 最近新增錯誤帶是模範；hero/Explore 卻 skeleton-then-vanish |
| 10 | Help and Documentation | 1 | 繼續觀看 說「連接 Plex / Jellyfin 後顯示」但無任何入口（且功能 blocked on Epic 17） |
| **Total** | | **20/40** | **及格邊緣：上半誠實、下半樣板** |

## Design Specificity Verdict

**上下兩個世界。** 上半（own-content）是真的為這個產品作的：exception-only 海報徽章以「復原方式」分類（libraryStatus.ts:186-196）、own-above-external 有 DOM-order 測試保護。下半（Hero + Explore）可整段搬去任何 Netflix clone：HeroBanner 通篇 hardcode text-white/bg-white/bg-black（HeroBanner.tsx:74,116），夜行 token 零出場。產品唯一護城河（無人值守字幕閉環）在首頁只分到 11px 徽章。

**Deterministic scan**：CLI 對 homepage/ 目錄本身 0 findings —— 但這是範圍假象：真正的機械缺陷全在 `components/library/PosterCardV2.tsx`（由 RecentlyAddedRowV2 渲染，掃描範圍外）。瀏覽器 overlay 18 findings：low-contrast ×13（12 個是同一顆評分藥丸）、heading-rhythm ×4（誤報：海報卡 caption 的緊上鬆下是正確節奏）、ai-color-palette ×1（誤報：hash 漸層佔位圖，非設計色盤）、edge-flush ×1（誤報：捲軸自身內容邊）。**偵測器漏抓**：整理中 藥丸 4.31:1（B 自己的探針量到；styles.css:72-76 的註解甚至早就記載這個 defect class）。

**Visual overlays**：本 harness 無使用者可見瀏覽器（headless），無 overlay 可看——以量測數據替代。live server 已確認停止。

## Overall Impression

上半頁在講真話、下半頁在演 Netflix。兩個 P0 都是「一行 token 用錯」等級的修復，但它們正在讓 AA 硬閘門每天被違反。最大的機會不是修 bug，是身分：首頁最大的 surface 在推銷你還沒擁有的內容，而產品真正的承諾（字幕閉環）沒有自己的一塊面。

## What's Working

1. **pickPosterBadge 的例外訊號設計**（libraryStatus.ts:186-196）——穩態零徽章、in-flight 交給活動中心、以「還救不救得回來」區分兩種缺字幕。誠實讀數寫進了型別系統。
2. **D3 own-above-external 法則**（HomeBrowseV2.tsx:33-40）——你的片庫永遠壓在 TMDb 行銷上，且有測試保護。
3. **最近新增的錯誤帶**（RecentlyAddedRowV2.tsx:117-135）——inline、role=alert、44px 重試、單區塊獨立降級。全頁其他區塊該長這樣。

## Priority Issues

- **[P0] 評分數字隱形（1.07–1.20:1）** — PosterCardV2.tsx:117 把 `--text-on-accent`（#14161a，設計給金底的深墨）放在 `--overlay-scrim`（70% 黑）上。雙評審獨立確認。修法：改 `--text-primary`（→11.94:1），一行。Suggested: /impeccable harden
- **[P0] 海報徽章 sub-AA 且底色半透明壓任意畫面** — libraryStatus.ts:36-40 TINT map 用原始 `--warning`/`--success`/`--info` 而非 `*-text` AA 變體（styles.css 自己記載 sub-AA）；12% tint 壓在亮色 hash 佔位圖上實測 1.58:1、洋紅上 1.80:1。修法：底改不透明面、文字一律 `*-text`。Suggested: /impeccable harden
- **[P1] 降級態對「使用者配置過的東西」不誠實** — Explore 區塊內容 500 後靜默 return null（ExploreBlock.tsx:64、ExploreBlocksList.tsx:44），skeleton 先閃後蒸發。按固定詞彙這是琥珀場景（你要求了但沒發生），卻被渲染成「不存在」。修法：頁級一條 inline 說明＋前往設定。Suggested: /impeccable harden
- **[P1] 輪播可及性三連** — 無暫停控制（手機永轉，WCAG 2.2.2）；圓點 8px 高（HeroBanner.tsx:229）；slide 是 role="link" 內嵌 button 與 Link（HeroBanner.tsx:45-63）ARIA 結構違規。修法：暫停鈕＋圓點 44px 觸控面＋slide 改非互動容器。Suggested: /impeccable adapt
- **[P2] 一頁兩套卡片 + hero 全 hardcode** — Explore 用舊 PosterCard（clip-path 8px、text-white:327、評分右下）與 PosterCardV2 並排分裂；hero 的白 focus ring 是全產品唯一白環。修法：統一到 V2、hero 過渡夜行 token、「查看更多」帶真 filter 或先藏。Suggested: /impeccable polish

## Persona Red Flags

- **Sam（SR/鍵盤）**：role="link" 嵌套互動元素；輪播無 aria-live 策略、朗讀中內容自換；評分數字對低視力不存在。
- **Casey（手機）**：輪播永不暫停；8×8 圓點點不到；想要/chevron desktop-only——這頁在手機上是純唯讀，與「手機完成完整任務」的產品事實有距離。
- **Jordan（首次安裝）**：沒 TMDb key 時首屏 1/3 是虛空（593/900px 有內容）；繼續觀看 是「未來的支票」——說連 Plex 卻無門把、功能也還沒做。

## Minor Observations

- 進行中 chip 穿金色（RecentlyAddedRowV2.tsx:90）——固定詞彙說綠＝正在發生；且 chip 不可點，無路通活動中心。
- Star 圖示日常穿 `--warning`（HeroBanner.tsx:89 等）——最承重的狀態色被裝飾性稀釋。
- hero 標題 48px 超過 Display 層（36px、僅詳情頁）。
- 頁面無 h1；hero h2 與區塊 h2 平級。
- ExploreBlocksList loading 是空白 min-h-[200px]（:47-53），與 skeleton 語言不一致。
- PosterCardV2 無年份時 meta 行收合、基線跳動；舊卡反而有   保位——修過的 bug 沒帶到新卡。
- 熱門影集列每卡「影集」chip ×8（PosterCard.tsx:235）＝已知資訊三次重複。

## Questions to Consider

1. 「片庫繁中覆蓋率 42/55」這種一眼定心的讀數，不才是這個首頁的 hero 嗎？為什麼最大 surface 在推銷你還沒擁有的內容？
2. 繼續觀看 佔每份安裝的第一格、等一個 blocked on Epic 17 的功能——是誠實的預留，還是首屏最貴位置上的一句道歉？
3. 當 TMDb 掛掉（降級是常態），首頁只剩兩列。首頁的身分該押在「多數使用者不會設定」的外部 API 上嗎？
