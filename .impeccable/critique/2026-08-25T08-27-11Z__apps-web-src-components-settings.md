---
target: settings（夜行複測）
total_score: 24
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
timestamp: 2026-08-25T08-27-11Z
slug: apps-web-src-components-settings
---
Method: dual-agent (A: 設計檢視 · B: 偵測器＋瀏覽器取證) · 夜行換皮後複測（前次 19/40）

## Design Health Score

| # | 啟發式原則 | 分數 | 主要問題 |
|---|---|---|---|
| 1 | 系統狀態可見性 | 3 | 檢查於／spinner／active tab 皆到位；清除結果 banner 不自動消退 |
| 2 | 系統與真實世界相符 | 3 | zh-TW 道地；日誌內容仍全英文 |
| 3 | 使用者控制與自由 | 2 | 兩個清除不可逆且無 undo、無出口 |
| 4 | 一致性與標準 | 2 | 8 個同級 tab 出現 4 種頁首模式；確認模式三套；按鈕圓角三種 |
| 5 | 錯誤預防 | 2 | 一鍵清 14,741 筆日誌零確認；同頁 CacheTypeCard 卻有二段確認 |
| 6 | 辨識優於回憶 | 3 | 全文字 tab、停用附原因、錯誤 banner 內建跳轉 |
| 7 | 彈性與效率 | 2 | 搜尋要按 Enter 無提示；重複錯誤不摺疊；無鍵盤捷徑 |
| 8 | 美感與極簡設計 | 2 | 密度克制，但實測型階比僅 1.6:1（11–18px 擠在一起，3 頁中招）＋日誌噪音牆 |
| 9 | 錯誤復原 | 3 | 解密錯誤 banner 教科書等級；saveMutation 原樣拋 message |
| 10 | 說明與文件 | 2 | 欄位一行式說明好；無深入求助路徑 |
| **總分** | | **24/40** | **可接受 — 距 Good 差的是一致性與防護，不是根基。** |

> 註：第 8 項由 A 的 3 下修為 2 —— 與前次同一依據：偵測器實測型階比（這次 1.6:1，比上次 1.8:1 更扁）是事實不是品味。與前次評分基準一致。

## Design Specificity Verdict

**A（未受偵測器影響）**：「調色盤是有作者的，構圖是通用的——而在 Operate 型設定頁，這大致是對的取捨。」夜行墨綠×宣紙白×金確實傳達出武俠劇愛好者的暗房閱讀環境，文案（「設定已經存在，不是沒設定過」）是誠實哲學的直接落地。但個性由 palette 和 copy 扛，composition 零貢獻；且系統自己的簽名（等寬讀數、單一圓角、固定詞彙）恰在此執行最鬆。**是主題外衣罩在通用構圖上，但外衣是真品。**

**Deterministic scan**：CLI 3 筆 advisory（11px 徽章 ×3）；routes 乾淨。瀏覽器覆蓋層 5 頁注入成功：**connection 0、keys 0**（前次 4/3）、logs/status 各 1（型階扁平）、**backup 3**（型階＋兩顆 **2.40:1 金底白字按鈕**）。`local/no-hardcoded-palette` 實跑 exit 0 —— token 紀律由 lint 強制中。原始 hex **0**、token 使用 594 次。

**兩位檢查員的一個分歧，已裁決**：B 量到 390px 時 active tab 未捲入視野；A 對全部 8 條路由逐一實測（scrollLeft 0–502 各自正確、全部 fully visible）。A 的證據更完整，判 B 抓到的是 effect 執行前的一幀 —— 慢裝置上可能短暫可見，列為觀察不列為缺陷。

**A 的一個事實錯誤，由 B 修正**：A 稱「無一處閱讀文字壓在實心強調色上」——B 找到兩顆：`建立備份`／`匯出` 按鈕 `text-white` 壓金底 **2.40:1**（BackupManagement/MetadataExport）。金比舊藍亮，白字比換皮前（3.68）更糟。QBittorrentForm 同款已在 #292 修過，這兩顆漏了。

## 八條 Named Rule（4 PASS / 4 FAIL）

| 規則 | 判定 | 證據 |
|---|---|---|
| 固定詞彙 | FAIL | 綠色仍說「做完了」：QBittorrentForm.tsx:227、CacheManagement.tsx:79、LogsViewer.tsx:145 |
| 配給強調 | **PASS** | active tab 用 15% 淡洗；金色實心只給可按控制項（新的！前次 FAIL） |
| 兩種藍（語意分工） | FAIL | `--error` 當字 9+ 處：已斷線 **3.00**、ERROR badge **3.17**、warning 字 4.30；`*-text` token 全套備好，元件沒接 |
| 預設小字 | PASS | 14/12px 紀律完好 |
| 比較才用等寬 | FAIL | 時間戳、0 B · 0 筆 仍是 sans；mono 只在路徑 |
| 貼齊側欄 | **PASS** | strip 貼齊、內容靠左、多餘寬度收右 |
| 色調優先 | **PASS**（前次 FAIL） | 三階背景分層、tinted 面零邊框（#298 的成果被確認） |
| 單一圓角 | FAIL | 按鈕 4/8/12px 三種並存；LogFilters 藥丸形仍在 |

## Overall Impression

**19 → 24。** 修的東西全數被獨立確認：對比反轉修好（active 11.50）、誠實橫幅被 A 稱為教科書等級、tab 條被列為第一 strength（44px 全達標、自動捲入、fade）、tinted 邊框歸零、token 紀律由 lint 守著。**剩下的分數卡在兩件事**：語意色被當文字讀（token 備好了、元件沒接 —— 純機械修）、和防護不一致（兩個無確認的清除）。這兩件修完，28+ 在望。

## What's Working

1. **Tab 條是真正下過功夫的導覽**：44px 全達標、8 條路由 active 自動捲入各自正確、fade 讓截斷可見、nav-of-links 而非假 tablist、停用項留在畫面附原因。
2. **誠實錯誤文案落地**：解密失敗那段，多數商業產品寫不出來。
3. **強調配給執行到位**：金色稀有到每次出現都有意義。（惟 B 抓到兩顆白字金底按鈕例外，見 P1。）

## Priority Issues

### [P1] 語意色當文字讀 —— 11+ 處 sub-AA，含兩顆 2.40:1 按鈕
已斷線 3.00、ERROR badge 3.17、warning 4.30、**建立備份/匯出白字壓金 2.40**。`*-text` token 全套已 gate 通過，元件層沒接上。修法：ServiceStatusCard:31,37,43、LogEntry:8-10、CacheManagement:35、LogsViewer:55、BackupManagement:98,145,175、QBittorrentForm:231 換 `*-text`；兩顆按鈕的 `text-white` 換 `text-[var(--text-on-accent)]`（墨字 6.79:1）。→ `/impeccable harden`

### [P1] 兩個無確認的破壞性清除，與同頁模式矛盾
LogsViewer.tsx:36、CacheManagement.tsx:19 一鍵即清；同頁 CacheTypeCard 有二段確認。統一採現成的 inline 二段。→ `/impeccable harden`

### [P1] 綠色說「做完了」，固定詞彙貶值
詞彙鬆動則「已連線」的綠開始不可信 —— 唯一不准貶值的資產。完成回饋改中性。→ `/impeccable clarify`

### [P2] 4 種頁首模式、僅 2 頁有 h1
抽 SettingsPageHeader。→ `/impeccable distill`

### [P2] 形狀與觸控漂移
按鈕三種圓角、chips 26px、清除鈕 24–30px、死 hover ×2。→ `/impeccable polish`

## Persona Red Flags

- **Sam**：P1 對比；10 個 tab 全在鍵盤序列（含 2 停用），走完要 10 下。
- **Alex**：搜尋要 Enter 無提示、重複錯誤不摺疊、零快捷鍵。
- **Casey**：破壞鈕全在拇指區外頂端、小控制項低於 44px、filter 狀態不保留。
- **繁中 NAS 自架玩家**：出事時偏要讀英文 log 才能自救 —— LogEntry 的 hint 欄位是正確方向，該讓它載繁中診斷。

## Minor Observations

unconfigured 拿 `--text-muted` 當面色（該有中性 tint）· 桌機不溢出時 fade 仍渲染且 pr-10 吃右緣 · 停用 tab 用 muted 反而過度可讀 · corrupted 與 pending 共用一色一形 · keys 頁三鑰共用一顆捲動後才可見的儲存 · 「目前僅支援 Claude 金鑰測試」是教科書級 disabled · B 的 390px 時序觀察：慢裝置上 active tab 捲入前可能短暫在視野外。

## Questions to Consider

1. 綠色連「已儲存」都能說，「已連線」的綠還剩多少可信度？
2. 北極星是誠實讀數的產品，最重要的讀數頁（日誌）為何是唯一沒被設計過的頁（無 .pen、英文內容、噪音無摺疊）？
3. 兩個「尚未開放」要掛多久 —— parked 是誠實還是永久 IA 債？
