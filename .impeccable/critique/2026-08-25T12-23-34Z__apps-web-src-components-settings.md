---
target: settings（三 P1 修復後第三跑）
total_score: 29
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
timestamp: 2026-08-25T12-23-34Z
slug: apps-web-src-components-settings
---
Method: dual-agent (A: 設計檢視 · B: 偵測器＋瀏覽器取證) · 第三跑（19 → 24 → 本次）
（A 中途被機器睡眠打斷，喚醒後以原 context 完成 —— 量測未重跑即有效）

## Design Health Score

| # | 原則 | R1 | R2 | R3 | 本次主要理由 |
|---|---|---:|---:|---:|---|
| 1 | 系統狀態可見性 | 2 | 3 | **4** | 每張卡無條件時間戳；loading/aria-live 齊備 |
| 2 | 真實世界相符 | 3 | 3 | 3 | zh-TW 自然；Base Path/ENCRYPTION_KEY 邊緣術語 |
| 3 | 控制與自由 | 2 | 3 | 3 | 確認皆可取消；搜尋要 Enter 無提示、掃描不可中止 |
| 4 | 一致性 | 1 | 2 | **2** | 8 個同級 tab 3 套頁首系統；圓角 4/6/8/full 並存；確認語法兩套 |
| 5 | 錯誤預防 | 2 | 2 | **4** | 二段確認拿到行為鐵證（首擊零 API 請求）；還原 modal；HTTP 明文 ack |
| 6 | 辨識優於回憶 | 2 | 3 | 3 | 分組只活在 aria-label 和分隔線 —— 螢幕閱讀器拿到的 IA 比明眼人多 |
| 7 | 彈性與效率 | 1 | 2 | 2 | 18,041 筆 ÷ 50 = 361 頁只有上下頁；零快捷鍵 |
| 8 | 美感與極簡 | 2 | 2 | **2** | A 給 3；照前兩跑同一依據下修 —— 型階 1.6:1 實測仍在 4 頁（B），加上兩個死 tab 佔 285px |
| 9 | 錯誤復原 | 2 | 3 | **4** | 解密橫幅「self-hosted 工具裡見過最好的錯誤訊息」（A 原話） |
| 10 | 說明與文件 | 2 | 2 | 2 | 金鑰頁行內提示好；他處零文件路徑 |
| **總分** | | **19** | **24** | **29/40** | **Good 邊緣 —— 弱軸是一致性(4)、效率(7)、文件(10)** |

## Design Specificity Verdict

**A（未受偵測器影響）**：「有作者的，而作者性活在文字和狀態機裡，不在版面。」骨架（tab 條＋卡片）是可換的後台家具；但裡面的東西沒有別的產品能原樣使用 —— 解密失敗橫幅「指認原因、駁斥誤讀、給兩條出路」；金鑰頁拒絕三種謊言；完成態刻意穿中性、且有行內註解在守。夜行調色盤（墨綠×宣紙×金）套用連貫。通用殘渣：備份回饋字串裡的 `✅`/`⚠️` emoji（全 App 唯一非 lucide 圖形）。**總結：誠實是訂製的，外殼是現貨的 —— 而在設定頁，現貨外殼是可以接受的。**

**Deterministic scan（B）**：CLI 3 筆 advisory（11px ×3）；routes 乾淨。覆蓋層六頁：**connection/keys 0**、其餘四頁各 1（同一筆型階 1.6:1）。**對比全面掃射零失敗，最差 6.26**。hex 0、token 611 次、`no-hardcoded-palette` lint exit 0。**二段確認的行為鐵證**：首擊零網路請求 → 上膛（label 翻轉＋朱砂填色＋取消浮現）→ 取消零請求 → 確認才發 `DELETE …older_than_days=30`。

**三個 P1 修正全數被兩位獨立確認**：語意色（最差 2.40 → 全過最差 6.26）、二段確認（行為驗證）、綠色詞彙（已清除橫幅實測中性 6.45、完成態穿中性有註解在守）。

## 本跑的新發現（前兩跑都沒抓到的）

1. **`rounded-md` 其實是 6px** —— 沒有 `@theme` 區塊把 `--radius-md: 8px` 映進 Tailwind，所以 `rounded-md` 靜默渲染成 Tailwind 預設 6px，4/6/8/full 四種圓角並存。token 存在但沒接上 —— 跟色彩債同構的形狀債。
2. **備份的 verify/restore 訊息不分結果套色**：「⚠️ 校驗碼不符，可能已損壞」穿金色（你在這裡）、「✅ 還原完成」穿琥珀（你要求了但沒發生）—— **最高風險的兩個時刻，顏色在說謊**。
3. **CacheManagement 的註解說「anywhere else disarms」但程式碼沒有 outside-click handler** —— 只有取消能退膛。註解與行為不符（#307 的殘留）。
4. **狀態頁只有單向修復連結**：qB 錯誤 → 金鑰頁有連結；但狀態卡上 TMDb=錯誤/qB=已斷線**沒有**「前往設定」—— 對「查狀態、修好、走人」的核心 persona，修法在一個 tab 外但沒路標。
5. **0 B 快取上渲染活的清除鈕**，按了會上膛一個 no-op 的破壞性確認 —— 違反自家「停用要留著並說明」模式。

## What's Working（A 的三強）

1. **降級狀態是一級設計**：解密失敗用橫幅取代表單而非蓋在空表單上（註解：「Better no readout than a flattering one」）；唯讀金鑰可見＋停用＋原因；讀取失敗徽章寫「無法確認」而非「尚未設定」。
2. **詞彙在漂移發生的地方被強制 —— code review 層**：三個元件帶著固定詞彙註解，讓「完成穿中性」能活過未來的編輯者。
3. **Tab 條是誠實的工程**：10 個 tab 全部實測 44.0px、分組進 accessible name、router 的 `data-status`、fade 讓截斷看得見、scroll-into-view 實測 auto-centered。

## Priority Issues（0 P0 · 3 P1 · 2 P2）

### [P1] 8 個同級 tab、3 套頁首系統
24→20→18px、有無 icon、有無描述 —— 切 tab 像換了產品。統一成 connection/keys 的最新裁決（h1 24px＋一行描述）。→ `/impeccable polish`
### [P1] 備份回饋與 INFO 徽章的詞彙違規
verify/restore 依結果路由套色（不符→warning、成功→中性、錯誤→error）；emoji 換 lucide；LogEntry/LogFilters 的 INFO 從金改 `--info-*`。→ `/impeccable colorize`
### [P1] 手機上操作控制項 28px（不是 44）
測試連線 28、per-type 清除 28、log 展開 16、區塊排序 28 —— Operate 介面的「動手」控制項全部低於承諾。tab 條已證明模式。→ `/impeccable adapt`
### [P2] BackupTable 在 390px 動作欄被 overflow-hidden 裁掉
610px 固定欄寬塞 342px 容器 —— 「NAS 掛了人在外面」的還原場景在手機上不可能。（fixture 零備份，源碼級查證。）→ `/impeccable adapt`
### [P2] 日誌 361 頁不可導航＋圓角漂移
日期篩選＋跳頁；`@theme` 映射 `--radius-*` 讓 `rounded-md` 說 token 的話。→ `/impeccable harden` / `shape`

## Persona Red Flags（精華）

- **Alex**：零快捷鍵、361 頁、單選等級篩選、無匯出 —— 「會直接 SSH 進去 grep，UI 輸掉了自己受眾的專家層」。
- **Sam**：大體強（aria 齊備、金色 focus ring、停用可達且有原因）；旗子：通知 5 秒自動消失無法暫停（WCAG 2.2.1）。
- **繁中 NAS 玩家**：狀態卡的判決到修法之間沒有路。
- **降級不失效**：0 B 快取的活清除鈕。

## Minor Observations

回饋橫幅永不消失（陳舊的「已清除 0 筆」會蓋在後續操作下）· 掃描排程 select 是全表面唯一零成功回饋的 mutation · 半數次要按鈕 hover 無視覺變化（hover 色=本色）· unconfigured 用 `--text-muted/10` 當底色（該用 bg-tertiary）· 停用測試鈕的原因字串每列渲染兩次（title＋可見文字）。

## Questions to Consider

1. 入口 redirect 到 qBittorrent —— 產品的心臟是金鑰（字幕要 Claude/TMDb），不是下載器。第一眼該是誰？
2. 兩個「尚未開放」每次造訪都花掉 285px 的黃金橫條寬度 —— parked 多久變成債？
3. 型階 1.6:1 第三跑仍在 —— 這是唯一連續三跑未動的實測發現。它是不是下一個「還債」？
