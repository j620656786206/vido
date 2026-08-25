---
target: settings
total_score: 19
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 4
timestamp: 2026-08-25T02-57-48Z
slug: apps-web-src-components-settings
---
Method: dual-agent (A: 設計檢視 · B: 偵測器＋瀏覽器取證)

## Design Health Score

| # | 啟發式原則 | 分數 | 主要問題 |
|---|---|---|---|
| 1 | 系統狀態可見性 | 2 | `QBittorrentForm.tsx:20` 只解構 `{ data, isLoading }`，從不看 `isError`。設定 API 回 500，畫面卻渲染乾淨空表單。`ServiceStatusCard.tsx:133` 的 `lastCheckAt` 藏在「顯示詳情」裡，而 connected／unconfigured 兩種狀態預設不展開。 |
| 2 | 系統與真實世界相符 | 3 | zh-TW 用詞道地、領域正確。扣分：中文介面裡夾生英文 log（`Failed to decrypt qBittorrent password`）、`Coming Soon`、手機上同畫面出現兩個「首頁」。 |
| 3 | 使用者控制與自由 | 2 | `BackupTable.tsx:116` 刪除備份無確認、無復原。`CacheManagement.tsx:61` 大量清除立即執行。風險梯度顛倒。 |
| 4 | 一致性與標準 | 1 | 9 條路由 3 種頁首樣式；同一畫面兩種 active nav 樣式；1 個元件用 design token、12 個不用；姊妹頁面內容寬度 768 vs 928。 |
| 5 | 錯誤預防 | 2 | 兩條不可逆路徑無防護；連線表單接受自由文字 host 無格式驗證。`CacheTypeCard` 的兩段式確認與 `RestoreConfirmDialog` 做得好。 |
| 6 | 辨識優於回憶 | 2 | 手機隱藏 10 個分類中的 5 個且無任何提示。`服務狀態` 顯示「未設定」卻不連到 `金鑰設定`。ENCRYPTION_KEY 橫幅要求離開 App、記住變數名、再回來。 |
| 7 | 彈性與效率 | 1 | 全面沒有鍵盤快捷鍵。日誌 1,029 筆 / 21 頁只有上一頁下一頁。沒有批次操作，沒有「測試全部服務」。 |
| 8 | 美感與極簡設計 | 2 | 密度克制是真的。但偵測器實測型階比只有 **1.8:1**（7 種字級擠在不到 2 倍範圍內），`/settings/cache` 是 5 排一模一樣的 `0 B · 0 筆` 加 6 顆清除鈕。 |
| 9 | 錯誤復原 | 2 | 原始 server 字串直接當文案：`CacheManagement.tsx:37` `{error.message}`、`BackupManagement.tsx:41-46` `err.message`。講了故障、沒講解法。 |
| 10 | 說明與文件 | 2 | 欄位輔助文案具體誠實，停用控制項附帶原因。但沒有文件連結，JSON/YAML/NFO 匯出各只有一行說明。 |
| **總分** | | **19/40** | **可接受 — 貼著下緣。需要顯著改善。** |

> 註：第 8 項從設計檢視員原本的 3 分下修為 2。他稱讚克制，但同一份報告裡又說同一批頁面「沒有視覺層級、沒有主要動作」。偵測器實測的 1.8:1 型階比把這件事從品味判斷變成事實，所以以事實為準。

## Design Specificity Verdict

**判決：分裂，而且斷點可以標日期。** 這個介面大約 20% 是為繁中 NAS 自架使用者寫的，80% 是通用的深色後台面板。

**LLM 評估（未受偵測器影響）**

真正屬於這個產品的部分：詞彙是母語不是翻譯（`qBittorrent`、`TMDb 中繼資料`、`豆瓣快取` 不加註解，因為受眾本來就懂）。`ApiKeysForm.tsx` 區分 Claude 的「儲存後立即生效，無需重啟伺服器」和 TMDB 的「儲存後需重啟伺服器才會生效」— 這句話只有知道 Go 後端各家金鑰載入生命週期不同的人寫得出來，偽造不了。`ScannerSettings` 顯示「尚未執行過掃描」而不是破折號或假時間戳，是 PRODUCT.md 誠實原則的字面執行。

可被任何產品替換的部分：**外框是現成的。** 圖示＋標題＋副標的頁首、`border + bg` 的卡片、右對齊的 ghost 按鈕、10 項的平面導航。把「快取管理」換成 Cache，這就是 Grafana、Portainer、Uptime Kuma，或任何一套 Tailwind 後台模板。**顏色系統是 Tailwind 的，不是 Vido 的** — `styles.css` 定義了完整調色盤和 `--radius-*`，設定頁卻繞過它：`bg-red-900/30`、`bg-green-400/10`、`bg-yellow-900/20`、`text-amber-400`。

**最欠缺的角色是「錢」和「時間」。** PRODUCT.md 說成本可見性「是使用場景的一部分，不是選配」，`AI_RUN_BUDGET_USD` 預設 5.0。而貼上這些會花錢的 API 金鑰的那一頁，**完全沒有**提到預算、花費或用量。`服務狀態` 則是一個沒有時間軸的狀態頁。

**結論：文案是被創作的，介面不是。** 一個設計師寫了句子，一個工程師用預設值組了盒子，兩邊沒碰過面。唯一在 DESIGN.md 之後才寫的元件（`ApiKeysForm`）證明系統能用，只是沒回頭改舊的。

### 八條 Named Rule 合規（2 PASS / 6 FAIL）

| 規則 | 判定 | 證據 |
|---|---|---|
| 固定詞彙 | **FAIL** | 綠色用在「做完了」— 規則明文禁止。`CacheManagement.tsx:80`、`LogsViewer.tsx:144`、`BackupTable.tsx:8`（`completed: '完成' → text-[var(--success)]`）。另外 `LogEntry.tsx:10` 把品牌藍徵召成 INFO 狀態色，而 `--info`（提示青）閒置。 |
| 配給強調 | **FAIL** | `SettingsLayout.tsx:117` active 用 `border-blue-400 bg-[var(--bg-tertiary)]`，規格要求的是 15% `--accent-subtle` 淡層。同一畫面上 app rail 的 active 和設定導航的 active 長得不一樣。 |
| 兩種藍 | **FAIL（實測）** | active 導航 3.04:1、`錯誤`／`已斷線` 3.50:1，都低於 AA 4.5:1。`--accent-text`／`--error-text` 早就存在，全頁只用了一次。DESIGN.md 逐字預言過這件事。 |
| 預設小字 | **PASS** | 14px/12px/11px 主導，最大內文是 24px h1。9 條路由都沒有膨脹字級。 |
| 比較才用等寬 | **FAIL（實測）** | 日誌時間戳是 `ui-sans-serif` 12px — 一整欄 50 個值要垂直掃描。`CacheTypeCard.tsx:65` 的 `0 B · 0 筆` 也是 sans。`font-mono` 全用在檔案路徑（正確），**沒有一個數字讀數用它**。 |
| 貼齊側欄 | **PASS** | 實測 app rail `0→240`、設定導航 `240→464`、內容 `464→1440`，**間隙 = 0px**。1920 寬同樣成立。`SettingsLayout.tsx:180-186` 的註解就寫在下一個人會想加 `mx-auto` 的位置。 |
| 色調優先 | **FAIL** | 直接命中「別給已經有色階的填色面再加邊框」：`border-subtle` + `bg-secondary` 同時出現在 **17 個檔案**。`connection.tsx:15` 和 `CacheTypeCard.tsx:56` 用 `bg-secondary/50` 把色階砍半再補邊框，等於把規則反過來做。 |
| 單一圓角 | **FAIL** | `SettingsLayout.tsx:160`、`LogFilters.tsx:59,73` 用 `rounded-full`，規格說「絕不做成藥丸形」。全目錄 6 種圓角、123 次使用中只有 **1 次**讀 token。 |

兩條 PASS 分別是「有過回歸所以被寫進測試」的版面規則，和「不需要 token 紀律就能守」的字級規則。**凡是需要伸手去拿 CSS 變數的規則，全部 FAIL。**

### Deterministic scan（偵測器）

CLI 掃描：`components/settings` 退出碼 2，**4 筆**；`routes/settings` 退出碼 0，**乾淨**。

| 規則 | 嚴重度 | 位置 |
|---|---|---|
| `design-system-font-size` | advisory | `ApiKeysForm.tsx:331`（`text-[11px]`） |
| `design-system-font-size` | advisory | `BackupTable.tsx:77`（`text-[11px]`） |
| `design-system-font-size` | advisory | `SettingsLayout.tsx:133`（`text-[10px]`） |
| `side-tab` | warning | `LibraryEditModal.tsx:287`（`border-l-4`） |

瀏覽器覆蓋層在 5 條路由注入成功，**26 筆真實發現**：低對比 8、字級過小 10、灰字壓彩底 5、型階扁平 3。

硬數字（排除 `*.spec.tsx`）：

| 指標 | 數字 |
|---|---|
| 原始 hex 色碼 | **0** |
| `var(--…)` token 使用 | 510 |
| **硬寫 Tailwind 色階**（`bg-red-900` 等） | **94 次 / 20 個檔案** |
| 圓角種類 | 6 種（僅 1/123 次讀 token） |
| 陰影 | 5 次，全在 modal，卡片零陰影 |
| `text-sm` / `text-xs` | 122 / 47 —— **佔全部字級決策 91%** |
| `onClick` 掛在非互動標籤 | **0** |
| 純圖示按鈕缺無障礙名稱 | **0 / 53 顆按鈕** |
| `data-testid` | 163；22/22 元件都有 co-located spec |
| Rule 21 `Design ref:` | 元件層 **22/22 有**；路由層 **0/12 有** |

「0 個 hex」這個數字有美化效果。真正的問題是 94 次硬寫的 Tailwind 色階，而且集中在**語意狀態色**（紅／綠／黃）上 —— 明明 `--error`（29 次）、`--success`（9 次）、`--warning`（8 次）都存在。狀態系統是精神分裂的：一半 token 化，一半硬寫。

### 誤報（已剔除）

- **19 筆 `text-occlusion` 全是誤報。** 覆蓋層在偵測它自己的標籤。證明是算術的：自動首掃 26 筆，手動再掃 45 筆，差 19 筆，逐頁對上 text-occlusion 的 2/3/3/3/8。`detail` 字串直接引用 impeccable 自己的規則名（`span "low contrast text" is 100% covered…`），而這個介面全是繁中，沒有這些英文字串。
- **3 筆 `design-system-font-size` 的權威性是循環的** —— DESIGN.md 一個 commit 前才從這份程式碼自動產生。同一行 10px 用覆蓋層的 `undersized-ui-text`（外部 11px 可讀性底線）當依據比較站得住腳，而且它不給「那就把 10px 加進色階」這條逃生門。
- **`side-tab` 是弱發現。** 那是有 token、有配色底的行內提示框（admonition），不是卡片，而且是整個 `apps/web/src` 樹裡**唯一**的 `border-l-4`。但它旁邊的側欄用 `border-l-2`，所以仍是一處一次性不一致 → 降為 P3。

### 覆蓋層畫面

覆蓋層曾在瀏覽器中實際注入並執行，但那個工作階段已結束、live server 已停止（port 8400 已釋放）。**現在沒有可看的覆蓋層畫面。** 上面的數字是當時 console 讀回來的。

## Overall Impression

好消息：這個表面的**語意**做對了。53 顆按鈕沒有一顆缺無障礙名稱，沒有一個 `onClick` 掛在 `div` 上，每個元件都有測試，文案是母語且誠實。骨頭是好的。

壞消息：**像素**沒跟上。而且不是隨機的醜 —— 是一個很具體的機制在漏水。設計系統定義好了、寫在 `styles.css` 裡了，然後 13 個元件裡有 12 個繞過它，直接抓 Tailwind 的預設色階。八條 Named Rule 有六條 FAIL，六條全部 FAIL 在同一件事上：**沒有伸手去拿 CSS 變數。**

最大的單一機會不是任何一個視覺修正，是**回頭把那 12 個元件改成用 token**。它會一次解掉對比失敗、狀態詞彙稀釋、圓角雜訊、卡片邊框冗餘 —— 這四項現在看起來是四個問題，其實是一個問題的四個症狀。`ApiKeysForm.tsx` 已經是現成的參考實作。

## What's Working

**1. 貼齊側欄的修正是真的，而且註解寫在會被讀到的地方。** 實測 `0→240 / 240→464 / 464→1440`，間隙 0px，1920 寬同樣成立。真正好的不是數字，是 `SettingsLayout.tsx:180-186` 那段註解 —— 它就坐在下一個工程師會想加 `mx-auto` 的那一行旁邊，解釋加了會壞掉什麼。那個 200px 死空隙不可能悄悄回來了。

**2. `ApiKeysForm` 是誠實原則真的被執行，不是被宣告。** 三件事同時做對：輔助文案反映各家後端真實行為；停用的「測試」按鈕**留在畫面上**並在旁邊寫明原因（「目前僅支援 Claude 金鑰測試」）而不是消失；阻擋條件用琥珀橫幅點名確切的環境變數。它也剛好是唯一使用 tint token 的元件。

**3. `/settings/scanner` 的密度是對的。** 等寬字的完整 NAS 路徑（這是「比較才用等寬」的正確用法 —— 這些路徑**確實**會被互相比較）、每個媒體庫的可達性圓點、「尚未執行過掃描」而不是假日期、一個全寬不會看漏的主要動作。另外八頁應該長這樣。

## Priority Issues

**沒有 P0。** 沒有任何一條路徑完全阻斷任務完成。這是誠實的發現，不是漏檢。

### [P1] 兩個最吃重的訊號對比不足 —— 違反產品自己的硬性門檻

**問題**：active 設定導航「連線設定」是 `#3b82f6` 壓 `#2e3b56` → **3.04:1**（`SettingsLayout.tsx:117`）。錯誤狀態「錯誤」／「已斷線」是 `#ef4444` 壓 `#24304a` → **3.50:1**（`ServiceStatusCard.tsx:36,41`）。每一個**非** active 的導航項都是 7.47:1 通過 —— 唯一失敗的是那個必須跳出來的。兩個獨立方法測到同一個數字（設計檢視員量 3.04，偵測器覆蓋層量 3.0）。

**為什麼重要**：PRODUCT.md 寫 WCAG AA 4.5:1 是硬性要求，不是理想值，而且記錄過 `--text-disabled` 因為只有 3.55:1 被否決掉。這兩個比被否決的那個**還低**。在昏暗房間（產品自述的使用情境）裡，「我在哪」和「什麼壞了」正是最需要活下來的兩個讀數。

**怎麼修**：凡是這個顏色被**讀**（而不是被按或被填）的地方，`text-[var(--accent-primary)]` → `text-[var(--accent-text)]`（#60a5fa）、`text-[var(--error)]` → `text-[var(--error-text)]`（#f87171）。兩個 token 都已經存在。然後把 active 導航的 `bg-tertiary` + `border-blue-400` 換成規格指定的 15% `--accent-subtle` 淡層。

**建議指令**：`/impeccable harden`

### [P1] 兩個不可逆動作沒防護，可逆的那個反而有 modal

**問題**：`BackupTable.tsx:116` → `onDelete(backup.id)` → `deleteBackup.mutateAsync(id)`。沒確認、沒復原、16px 垃圾桶圖示。`CacheManagement.tsx:61` 一鍵清掉全部五種快取。同時「還原備份」有完整的 `RestoreConfirmDialog`，單一快取類型清除有兩段式確認。

**為什麼重要**：刪掉備份會銷毀使用者資料庫狀態的唯一副本；還原只是覆蓋當前狀態，備份檔還活著。**儀式感的分配是反的。** 手機上那個垃圾桶是 28×28，緊鄰下載和驗證。「使用者會不會為這個聯絡客服？」—— 一指誤觸刪掉唯一的備份之後，會。

**怎麼修**：把刪除備份接上現成的 `RestoreConfirmDialog` 模式。大量清除快取套用 `CacheTypeCard` 已經寫好的兩段式確認 —— 兩個模式都在同一個資料夾裡。兩者都套 `--error` 填色（Destructive 按鈕規格保留紅色正是給這種）。

**建議指令**：`/impeccable harden`

### [P1] 連線頁吞掉伺服器錯誤，然後渲染一個令人安心的謊言

**問題**：`QBittorrentForm.tsx:20` 是 `const { data: config, isLoading } = useQBittorrentConfig();` —— `isError` 從沒被解構或處理。設定端點目前回 500（每次載入 `/settings/connection` 都捕捉到 2 個 console 錯誤；App 自己的日誌檢視器顯示 `Failed to get qBittorrent config` 和 `Failed to decrypt qBittorrent password`）。使用者看到的是一個乾淨的、有 placeholder 的表單。偵測器獨立確認了 runtime 行為：HTTP 500、頁面上找不到任何錯誤字眼、兩顆 CTA 載入時都是 disabled。

**為什麼重要**：這正是 PRODUCT.md 的北極星要防的那件事 ——「會諂媚的讀數比沒有讀數更糟」。一個設定存在但解不開密的使用者，在視覺上被告知他從沒設定過。他會重打一次本來就在那裡的帳密，而底層的解密失敗繼續藏著。更糟的是 `/settings/status` 正確顯示 qBittorrent「已斷線」—— **App 在兩個分頁之間自我矛盾。**

**怎麼修**：處理 `isError`，用故障紅橫幅寫出真實狀況和成因（「無法讀取 qBittorrent 設定：密碼解密失敗」），並連到 `金鑰設定`（ENCRYPTION_KEY 的說明在那）。**永遠不要把空表單當成載入失敗的狀態** —— 空表單是一個「這裡什麼都沒存」的斷言。

**建議指令**：`/impeccable clarify`

### [P1] 手機藏掉一半的設定 IA，而且每個觸控目標都太小

**問題**：390px 實測，分頁列 `scrollWidth 742` vs `clientWidth 390`。**「日誌」「狀態」「備份」「匯出」「效能」在畫面右外側**，沒有漸層、沒有箭頭、沒有捲軸、沒有露出半個分頁 —— 第五個分頁「快取」結束在 x=377，所以整條看起來是完整的。分頁高 **30px**。其餘低於 44px 的：日誌展開鈕 **16×16（×50）**、首頁列動作 **28×28（×12）**、快取「清除」28px、「測試連線」28px、「顯示詳情」16px、備份 radio **13×13**。

**為什麼重要**：PRODUCT.md（2026-08-25 確認）寫「手機與桌機同等重要…手機上要能完成完整任務」。使用者在手機上沒發現那個看不見的滑動手勢，就到不了「備份與還原」或「服務狀態」。DESIGN.md 規定手機最小 44px；這個表面上沒有任何一個達標。

**怎麼修**：讓第五個分頁明顯被裁切（縮內距讓「日誌」露出一角）加邊緣漸層，並在掛載時把 active 分頁自動捲進視野。藥丸提高到 44px，圖示按鈕用 padding 把點擊區撐到 44×44、字形維持 16px。

**建議指令**：`/impeccable adapt`

### [P2] 設計系統定義好了，然後被 13 個元件中的 12 個繞過

**問題**：`styles.css` 依 DESIGN.md 定義了完整調色盤、tint 組和 `--radius-*`。設定頁接著用原生 Tailwind 手刻：`bg-red-900/30`、`bg-green-400/10`、`bg-yellow-900/20`、`border-blue-800`、`text-blue-300`、`text-amber-400`。`ApiKeysForm.tsx` 是唯一使用 `--success-tint`／`--info-tint`／`--error-tint`／`--warning-tint` 的檔案；`var(--radius-*)` 全目錄出現一次。偵測器獨立數到 94 次硬寫色階、6 種圓角、只有 1/123 次讀 token。

**為什麼重要**：**這是前面四個問題的機制，不是另一個美學抱怨。** 狀態詞彙是 PRODUCT.md 說的護城河 ——「這組詞彙比任何程度的精緻都值錢」—— 而它正在被一次一個 `bg-green-400/10` 稀釋掉。整個表面九種近似的紅，等於沒有一種紅有意義。

**怎麼修**：把 `ApiKeysForm.tsx` 當參考實作，回頭改那 12 個元件。從語意 tint 開始（每次編輯換到的意義最多），再來是卡片的 `border`+`bg` 配對，最後圓角。**順便修掉這裡浮出來的 IA 矛盾**：`匯出/匯入` 在導航裡被停用並標「Coming Soon」，但一個完全可用的 `匯出媒體資料`（JSON/YAML/NFO）就活在 `/settings/backup` 上。

**建議指令**：`/impeccable polish`

### [P3] `LibraryEditModal.tsx:287` 的 `border-l-4`

整個 `apps/web/src` 樹裡唯一的 `border-l-4`，用 token、有配色底、是標準的行內提示框樣式 —— 不是重複性的壞習慣。但它旁邊的側欄用 `border-l-2`。一次性不一致，順手改掉即可。

## Persona Red Flags

**Alex（沒耐性的重度使用者）** —— 最接近真實受眾
- 全表面零鍵盤快捷鍵。
- `/settings/logs`：1,029 筆、21 頁，**只有上一頁／下一頁**。沒有跳頁、沒有每頁筆數、沒有日期篩選。要找昨天的錯誤是 20 次點擊。
- 沒有批次操作：不能多選備份刪除，首頁區塊只能一次上移／下移一格（把第 3 塊移到第 1 位要兩次來回）。
- `/settings/status` 五個服務、五顆「測試連線」，沒有「全部測試」。
- 瑣碎的快取清除有多餘的確認；刪除備份**零**確認。他會被第二個燙到，正因為第一個教會他「這個 App 會確認」。

**Sam（依賴無障礙功能）** —— 敗得最慘，而且敗在產品自己訂的硬門檻上
- active 導航 3.04:1、錯誤狀態 3.50:1，都低於 AA。Sam 無法可靠地判斷自己在哪一頁設定。
- `/settings/status` 只用顏色＋圖示傳達健康度 ——「錯誤」和「已斷線」都是紅色不同字形。文字標籤有（好），但對比不足削弱了它。
- `LogEntry` 展開鈕 **16×16**，每頁 50 個。
- 200% 縮放：設定導航是固定 `w-56` 不換行不收合；1440/2 = 720px 有效寬度會跨過 `md` 斷點掉進分頁列，然後分頁列藏掉一半分類。
- **要給的功勞**：圖示按鈕的 `aria-label` 都在且具描述性（`上移 熱門電影`、`刪除 熱門影集`），`ApiKeysForm.tsx:515` 用了 `role="status" aria-live="polite"`。偵測器也確認 53 顆按鈕零缺名、零假互動元素。**語意做完了，像素沒做。**

**「阿哲」—— 繁中 NAS 自架者**（依 PRODUCT.md 受眾推導）
- **`金鑰設定` 從不提錢。** 他貼上 Claude 金鑰和 ASR 金鑰，沒有任何跡象顯示這些會花真的美金，沒有預算欄位，沒有連到 `AI_RUN_BUDGET_USD`。PRODUCT.md 原則三（「花錢的事先問」）在這個唯一啟用花費的畫面上零體現。
- **ENCRYPTION_KEY 橫幅對他特別是死路。** 它點名了變數，沒說放哪 —— 沒有路徑、沒有 `docker run -e` 範例、沒有文件連結。他知道什麼是環境變數，他不知道**這個** App 的檔案配置。
- 日誌內容是原始英文，夾在全中文介面裡。他最需要診斷的地方，正是唯一停止說他語言的地方。

**「回訪者」—— 離開後回來查的人**（PRODUCT.md：「使用者會關掉視窗走開，之後回來查」）
- **`服務狀態` 沒有時間軸。** `ServiceStatusCard.tsx:133` 只在「顯示詳情」裡渲染 `lastCheckAt`，而 connected 和 unconfigured 兩種狀態預設不展開。一個綠色服務永遠無法告訴他這是 2 秒前還是 6 小時前驗證的。
- **`/settings/logs` 預設「全部」**，所以他「我不在時發生了什麼」的查詢回傳一整面 `Log batch inserted`。共 1,029 筆，沒有錯誤筆數的分解。
- **`/settings/connection` 主動誤導他** —— 顯示一個看起來沒設定過的表單，而設定其實存在只是解不開密。
- 任何動作之後唯一的成功訊號是一行不會自動消失的小綠字，回來時無法和一小時前產生的那行區分。

## Minor Observations

- **姊妹頁面的內容寬度不一致。** `connection.tsx:10` 蓋在 `max-w-3xl`（實測 h1 寬 **768px**），`SettingsLayout.tsx:184` 把其他全部蓋在 `max-w-5xl`（實測內容 **928px**）。在設定分頁間切換時，卡片寬度會肉眼可見地跳動。
- **設定導航不會固定。** `SettingsLayout.tsx:89` 是 `hidden w-56 shrink-0 border-r md:block` —— 沒有 `sticky`、沒有自捲。而 app rail **是** `sticky top-0 h-screen`。在 `/settings/logs`（2,420px 高）捲到底時子導航不見了、app rail 還在，要切分類得整個捲回頂端。
- **`/settings/cache` 在零狀態下是個沒用的空狀態。** 五排 `0 B · 0 筆` 加六顆什麼都不會做的清除鈕，而且**沒有停用**。總計為 0 時應該收合成一行誠實的字。
- **`/settings/backup` 的空狀態是惰性的。** 「尚未建立任何備份」置中在一個有邊框的盒子裡，盒子裡沒有 CTA；「建立備份」按鈕在盒子外面上方。動作應該放進那個空洞裡。
- **頁面叫「備份與還原」，但零備份時「還原」完全不可見** —— 標題有一半在講一個畫面上沒有代表物的能力。
- **`/settings/export` 只能靠打網址進去。** 導航項是 `<span>` 不是 `Link`，但路由渲染了一個真的 placeholder，沒人navigate得到。
- **`ScannerSettings.tsx:151` 用了原生 `<select>`** 配 `bg-[var(--bg-primary)]` —— 比表面上其他所有輸入框（用 `bg-secondary`）明顯更暗，而且渲染 OS 預設的下拉箭頭。在截圖裡明顯出系統。
- **`✅` / `⚠️` emoji 被當狀態字形用**（`BackupManagement.tsx` 的驗證／還原訊息），而其他所有狀態都用 lucide 圖示。
- **手機同一畫面兩個「首頁」** —— 設定分頁（「自訂首頁」縮成「首頁」）和底部導航的 home。
- 停用導航項的 `title="此功能尚未實作"` 只在 hover 出現 —— 觸控裝置上看不到，而那正是最模糊的地方。
- **Rule 21 追溯性有乾淨的斷層**：元件層 22/22 都有 `Design ref:` 標頭，路由層 0/12 有。路由是薄的組合層，這可能是刻意的政策而非漂移 —— 值得確認 Rule 21 的範圍是否只涵蓋元件。
- 桌面九條路由都沒有水平溢出（`scrollWidth 1440 === clientWidth 1440`）。乾淨。

## Questions to Consider

1. **這個 redirect 把每一個設定訪客先送到 qBittorrent 帳密。** 產品的目的是中文字幕，下載客戶端是配角。如果 `/settings/` 落在一個狀態摘要（金鑰已設、媒體庫可達、上次掃描、本月花費）而不是最不核心那個整合的表單，會怎樣？
2. **`金鑰設定` 收集會花真錢的金鑰，卻從不提錢。** 如果預算就住在消耗它的那把金鑰旁邊，這一頁會長什麼樣？
3. **你為可復原的動作蓋了 `RestoreConfirmDialog`，為兩個不可復原的動作什麼都沒蓋。** 如果把這個表面上每個動作依「誤觸的話使用者失去什麼」排序，現在的確認對話框分佈撐得過那個排序嗎？
4. **`ApiKeysForm` 用 design token，12 個姊妹元件不用。它也是最新的檔案。** DESIGN.md 記錄的是這個 codebase **是**什麼，還是一個近期元件**變成**了什麼？什麼機制能讓下一個元件長得像 `ApiKeysForm` 而不是像 `BackupTable`？
5. **DESIGN.md 說強調色要配給、狀態色是固定詞彙。設定頁出了九種紅、五種綠，還用綠色表示「完成」—— 詞彙表明文禁止的。** 如果使用者從這個表面學這個 App 的顏色語言，他會學到什麼？
6. **一個說不出「什麼時候」的狀態頁，到底在斷言什麼？** 對一個護城河是「無人值守的信任」的產品，一個沒有時間戳的綠點，比沒有點更誠實嗎？
7. **手機上 10 個設定分類有 5 個看不見，也沒有任何存在的暗示。** 水平分頁列是 10 個項目的正確模式嗎，還是這其實是一個 sheet／accordion，因為桌面有側欄所以被做成了分頁？
8. **連線表單現在告訴使用者他沒設定過一個他設定過的東西。兩次點擊外的日誌頁說的相反。** 如果介面能在兩個分頁之間自我矛盾，那個定位所宣稱的「單一真相來源」是什麼？
