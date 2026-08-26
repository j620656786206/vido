---
target: homepage
total_score: 27
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 2
timestamp: 2026-08-25T23-30-35Z
slug: apps-web-src-components-homepage
---
Method: dual-agent (A: design-review subagent · B: detector/browser subagent)

# /impeccable critique homepage — 2026-08-26（run 3）

Target: homepage/ + index.tsx + 頁上兩張卡 + AvailabilityBadge。實測降級態＋mock 完整態（含 hover 與 owned 態——B 這輪被明確指派去開這些狀態），1440×900 與 390×844。

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | chip/徽章/琥珀線強；降級骨架 4 秒後整段塌縮、hero 無聲消失（bodyH 2129→900 實測） |
| 2 | Match System / Real World | 3 | 「連接 Plex / Jellyfin 後顯示」指向全 app 不存在的流程 |
| 3 | User Control and Freedom | 3 | 暫停鈕/圓點/Esc 都對；hero 暫停不被記住、CW 板不能收 |
| 4 | Consistency and Standards | 2 | 同頁兩套徽章文法相反（V2 藥丸+tint+底墊 vs 已有 4px 角+實心）；meta 字體 mono vs sans |
| 5 | Error Prevention | 3 | hover-intent、防重複點、retry:1 |
| 6 | Recognition Rather Than Recall | 3 | 徽章用字不用謎圖示；「[FanSub]…」的 fallback 字母是一個大「[」 |
| 7 | Flexibility and Efficiency | 2 | 到 hero 要 ~33 tab stops、無 skip link（已立案，與 rails 併案） |
| 8 | Aesthetic and Minimalist Design | 3 | 配給強調漂亮；空承諾板＋12/24 張實心綠「已有」是兩種反極簡 |
| 9 | Error Recovery | 3 | 琥珀線＋44px 重試；hero 同因失敗卻無交代（已立案） |
| 10 | Help and Documentation | 2 | 最顯眼的一句說明指向不存在的門 |
| **Total** | | **27/40** | **↑ 20→25→27；R2 修正雙評審全數驗證（已有 8.36:1、對齊 264 全線、chip 琥珀同字）** |

## Design Specificity Verdict

有立場的頁面：D3 結構性 IA、例外訊號徽章階梯、hero 融底、對齊紀律實測為真（全區段含 hero 共用 x=264）。但特異性幾乎全住在「誠實層」，視覺構圖是稱職的 Netflix 方言——而唯一「只有 Vido 做得出來」的自有內容區，開頭是一塊不能兌現的預留板。產品核心承諾（字幕閉環）在首頁只剩徽章。（＝已立案的 identity 題，A 第三度按門鈴。）

**Deterministic scan**：CLI 0 findings（含 AvailabilityBadge）。Overlay 自然態 6（4 heading-rhythm FP、1 hash 紫漸層 FP-leaning、1 edge-flush FP）；mock 完整態 +24，其中**實質發現：undersized-ui-text ×16 — 已有/已請求/新增/TMDb 徽章全是 10px 功能文字**（下限 11px）。R2 的 P0 修正實測確認：已有 8.36:1（與程式註解分毫不差）、已請求 5.57:1。

**Visual overlays**：headless，無使用者可見 overlay；live server 起停 3 次皆確認停止。

## Overall Impression

分數第三輪連漲（20→25→27），對比債已清零、對齊與詞彙紀律實測成立。剩下的頭獎全是「誠實層」的殘餘：一枚穿錯衣服又只有 10px 的「已有」、一塊佔著黃金位置的假門、以及降級態每次進頁重演的骨架塌縮劇場。

## What's Working

1. 對齊與尺度紀律實測為真：四個標題全在 x=264；hero 36px 頂在 Display 上限。
2. 狀態誠實系統玩真的：chip 與徽章同字同琥珀（遵守 pending=QUEUED 裁定）；琥珀線＋門是教科書。
3. a11y 工藝：44px 圓點/暫停、焦點陷阱＋還原、group-focus-within 補救、inert 滑動頁。

## Priority Issues

- **[P1] AvailabilityBadge 需要重製（三病一窩）** — ① 「已有」穿實心執行綠（AvailabilityBadge.tsx:20）：綠＝正在發生，「已有」是靜態事實，正是固定詞彙明令禁止的用法，且是全頁最高頻狀態色（12 次）；② 文字 10px（:44，B 的 undersized ×16，含 PosterCard 新增/TMDb 同班）；③ 與同頁 V2 徽章文法相反（4px 角+實心 vs 藥丸+tint+不透明底墊）。修法：已有 → 中性 scrim 藥丸（它是分類不是狀態）；已請求 → 琥珀 tint 藥丸＋V2 式底墊；全窩 11px+。另：每枚徽章帶 role="status" aria-live —— ownership 解析瞬間 12 個「已有」公告同時洗版 SR，改為靜態（或彙總一則）。Suggested: /impeccable harden
- **[P1] 繼續觀看板：黃金位置上的假門** — 「連接 Plex / Jellyfin 後顯示」讀起來是「去連就有」，但全 app 沒有這個門（grep 證實）。牴觸「未來工作不得假裝已存在」。修法：改誠實文案，或 Epic 17 前不渲染（不出現＝你沒要求，才是此刻的真話）。與 identity 立案重疊——需要裁定。
- **[P2] 降級態骨架—塌縮劇場** — TMDb 沒 key（PRODUCT.md：常態）時每次進頁：3 秒骨架 → 5 秒全蒸發只剩琥珀線（bodyH 2129→900）。修法：5xx/未設定不重試、記住上次降級結論直接渲染琥珀線，成功再升級。Suggested: /impeccable harden
- **[P2] Hero 圓點/meta 對比無保底** — 非活動圓點 /50 壓任意劇照實算 1.87:1（非文字需 3:1）；meta 列 --text-secondary 在白劇照下 3.20:1。修法：圓點群組墊 scrim 藥丸（Shapes 修正案明文合法）、meta 升 --text-primary。
- **[P3] 行動端細節** — chip 24px 高（44px 下限）；TrailerModal 關閉鈕 32px；hero 換頁 700ms 疊字殘影（實拍「怪奇物語二部」）；手機初載 tab bar 蓋住 hero 圓點。

## Persona Red Flags

- **Sam**：12 個 aria-live「已有」同時公告＝SR 洗版；~33 tab stops 無 skip（已立案）。
- **Casey**：手機 250px hero 塞五樣東西零餘裕、tab bar 蓋圓點；24px chip 點不準。
- **Jordan（沒填 key 的首跑者，最常見）**：首頁＝假門板＋字母色塊戴徽章＋琥珀線，900px；誠實但沒有一句話告訴他「裝滿之後長什麼樣、為什麼值得去填 key」。

## Minor Observations

- Hero 片名是 h2，與區段標題同級——大綱裡一部片名冒充區段。
- fallback 字母取 title[0]：「[FanSub]…」渲染成大「[」；跳過前導符號即可。
- V2 meta mono 11px vs PosterCard sans 12px——「比較才用等寬」只有一半書架遵守。
- chevron 無可捲動方向時仍渲染（TODO 已自認）。
- 探索 hover 時徽章群 opacity-0（kebab 讓位策略）——owned 徽章 hover 中不可見，程式註解稱刻意，記錄為機械事實。
- 片庫成長後探索列將成「已有」海（mock 已 12/24）——濾掉或折尾？

## Questions to Consider

1. 產品的靈魂在首頁哪裡？「你不在的時候我處理了 N 部片的字幕」這句只有 Vido 說得出的話，至今沒有一格在說。（identity 立案第三度被按門鈴）
2. 片庫 500 部時，探索列是幫你發現新片，還是幫你數已有？
3. 8 秒自轉、暫停不被記住、每輪付 700ms 疊字稅的 hero——對「移除操作者」的產品是不是本質上的異物？
