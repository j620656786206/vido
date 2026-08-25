---
target: homepage
total_score: 25
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 3
timestamp: 2026-08-25T17-38-03Z
slug: apps-web-src-components-homepage
---
Method: dual-agent (A: design-review subagent · B: detector/browser subagent; A resumed once after an API drop)

# /impeccable critique homepage — 2026-08-26（run 2）

Target: `apps/web/src/components/homepage/` + routes/index.tsx + 渲染在頁上的 PosterCardV2/PosterCard。實測 http://localhost:8090/（降級態＋mock 完整態），1440×900 與 390×844。A 以 0–10 評分，總分 66/100；依 playbook 換算為 0–4 制（×0.4 四捨五入）。

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | 降級列/chip/徽章俱佳；Hero 失敗仍無聲消失（HeroBanner.tsx:186-188） |
| 2 | Match System / Real World | 2 | 「查看更多」仍落無過濾 /search；▶ 播放覆蓋層暗示不存在的播放能力 |
| 3 | User Control and Freedom | 2 | 暫停鈕/Esc/焦點還原都在；但輪播每 8s 把鍵盤焦點打回 BODY（實測 8.6s） |
| 4 | Consistency and Standards | 2 | 兩套卡片 hover 文法、兩套橫向捲動、Hero 左邊線偏 24px、rounded-full ×10 待裁定 |
| 5 | Error Prevention | 3 | 想要防重複、hover-intent 防請求風暴、ownership 批次化 |
| 6 | Recognition Rather Than Recall | 3 | 資訊就地呈現、無隱藏語意 |
| 7 | Flexibility and Efficiency | 2 | 穿越探索區 ~120 tab stops、無 skip；hover-only 資訊手機拿不到 |
| 8 | Aesthetic and Minimalist Design | 2 | 密度克制；但首屏第一格仍是永久空殼（CW slot） |
| 9 | Error Recovery | 3 | 琥珀列附門、44px 重試；預告片空狀態是死巷 |
| 10 | Help and Documentation | 3 | placeholder 自我解釋、chip 連到活動中心 |
| **Total** | | **25/40** | **↑ from 20 — R1 修正全數驗證有效；剩下的是移植區的世界觀問題** |

## Design Specificity Verdict

夜行色盤、例外訊號徽章、降級琥珀列是有立場的設計；但 Hero+Explore 仍是「換上夜行顏色的 Netflix 直譯」——顏色歸化了，世界觀（誠實讀數）還沒有。R1 的五個修正雙評審皆獨立確認：對比 22 組全過（最低 5.08:1，R1 最低 1.1:1）、輪播 44px dots＋暫停鈕＋native stretched link、降級列被 A 評為「誠實的讀數的教科書實作」。

**Deterministic scan**：CLI 0 findings（這次含兩張卡片檔）。瀏覽器 overlay 6 findings：heading-rhythm ×4（誤報：海報 caption）、ai-color-palette ×1（誤報：hash 佔位圖）、edge-flush ×1（真的但輕微：捲軸 0px 邊緣內距）。**雙方都漏了 hover 態**——A 用 mock 資料看到 hover/owned 態才抓到本輪 P0。

**Visual overlays**：headless harness 無使用者可見瀏覽器，以量測數據替代；live server 已確認停止。

## Overall Impression

R1 修的東西是真的修好了；本輪的新頭獎是 token-debt 同類的最後一窩（白字壓翡翠綠 2.17:1），以及兩個「說謊的 affordance」（▶ 播放、綠 chip vs 琥珀徽章自相矛盾）。骨架已經及格，剩下的是移植區要不要歸化的問題。

## What's Working（A 特別標注「不能改壞」）

1. 降級通知模式（ExploreBlocksList）— 琥珀 tint、role="status"、附唯一能修好它的門。
2. 例外訊號徽章 — steady state 不渲染，資訊噪音控制極好。
3. 輪播 a11y — inert 清 slide、44px dots、暫停鈕、native anchor；R1 修正是真修。
4. D3 own-above-external、per-section fail-soft、手機 0px 水平溢出、reduced-motion。

## Priority Issues

- **[P0] AvailabilityBadge 一窩白字壓翡翠綠** — `AvailabilityBadge.tsx:16` `bg-[var(--success)] text-white` 實算 **2.17:1**（獨立驗算確認；正確解 `--text-on-accent` 壓 --success = 8.36:1）。同檔同類：PosterCard「新增」徽章 :231-236 同 2.17、metadataSource 徽章 :238-241 ≈2.71。token-debt 四波修正漏掉的最後一窩，就在首頁探索卡上。Suggested: /impeccable harden
- **[P1] 輪播每 8s 沒收鍵盤焦點** — 焦點在 title link 上，slide 轉 inert 時整棵子樹被踢出，activeElement 落回 BODY（實測 8.6s）。滑鼠有 hover-pause，鍵盤沒有 focus-within pause。修法：onFocusCapture/onBlurCapture 對等設 pause。Suggested: /impeccable harden
- **[P1] 固定詞彙自相矛盾（需裁定）** — 同一批 parseStatus=pending 項目：卡上戴琥珀「整理中」、標題列 chip 穿綠「進行中」。兩者必有一個在說謊——pending 是「排隊未跑」（琥珀對）還是「管線正在消化」（綠對）？需要一次裁定、兩處對齊。
- **[P1] ▶ 播放覆蓋層是說謊的 affordance** — PosterCard.tsx:263-272 hover 出大圓 ▶，點了導去詳情頁；產品今天不存在播放路徑。誠實優先的產品裡，首頁最大的一句假話。修法：換成「查看詳情」語意的 affordance 或拿掉。
- **[P2] 移植區歸化批** — Hero 左邊線偏 24px（264 vs 288）；最近新增排無 chevron/scrim 而探索排有（同型不同文法，首卡被裁切無 affordance）；「查看更多」TODO 未還；預告片空狀態死巷＋bg-black/85 硬編；區標題 text-xl vs text-lg md:text-xl（20px 不在字級表）。Suggested: /impeccable polish
- **[P2] 藥丸形需裁定** — DESIGN.md Shapes「絕不做成藥丸形」仍是現行法，首頁 rounded-full ≥10 處。要嘛夜行修法（DESIGN 補裁定），要嘛還原。Sally 的裁決題，不該放任漂移。

## Persona Red Flags

- **Sam（鍵盤/SR）**：焦點沒收＋~120 tab stops＋無 h1 — 首頁對這群人接近不可用。
- **低視力**：2.17:1 的「已有」徽章直接看不見。
- **Casey（手機）**：想要/片長/kebab/chevron 全 hover-only；任務可繞詳情頁完成，但首頁是次級體驗。
- **Jordan（新手）**：第一屏指向不存在的 Plex 設定頁——第一分鐘就教會使用者「這介面說的話不一定算數」（已立案 disc-2026-08-home-identity-hero）。

## Minor Observations

- 全頁無 h1（outline 起自 H2）。
- 手機 Hero 年份/評分列落在漸層 ≈25% 不透明處，壓亮部海報是對比抽獎。
- CW 空框同時有色階又有邊框（色調優先違規）。
- 手機探索卡拿不到片長/季數（hover-intent 才抓）。
- edge-flush：捲軸 0px 邊緣內距（B 實測，輕微）。

## Questions to Consider

1. 首頁是使用者的，還是 roadmap 的？誠實讀數的產品，首屏沒有一格在讀「字幕做到哪了」。（已立案）
2. 綠 chip 與琥珀徽章，哪個在說謊？同屏兩色等於宣告詞彙可以協商。
3. 藥丸形是夜行的新法，還是 Epic 10 的走私品？二選一，不能懸著。
