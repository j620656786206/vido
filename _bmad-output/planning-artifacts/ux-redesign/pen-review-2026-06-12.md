# Vido `ux-design.pen` 全檔 UX Review — 2026-06-12

> **方法**：4 個並行代理對整份 `ux-design.pen`（A–J 流程 + Design System Reference / Component
> Library + 畫布標註層）做純 review，**未修改任何畫布節點**。本檔以 zh-TW 忠實保存（含中文 UI
> 字串與節點 ID），作為 UX Redesign Initiative **Phase 0 的主要證據輸入**。
> **分工**：代理一 = 流程 A+C；代理二 = 流程 B；代理三 = D/E/F/G；Sally(主持) = H/I/J + Design System + 標註層。

---

## Part 1 — 跨流程綜合 Triage（最重要的分析輸出）

### 🔑 六大系統性根因（4/4 代理交叉印證）
1. **中文字型 DM Sans / Inter** — 無 CJK 字形，實際 fallback 不可控（全 4 流程都中；元件 `ButtonPrimary`/`ButtonSecondary`/`FilterChip` 是根源）。
2. **內容裁切到看不見** — A/B/C/E 都有 HIGH（sheet 塌陷、按鈕被切、面板溢出）。
3. **語意色硬編碼 / token 漂移** — D/E/F/G + B 大量（status tint、accent tint、fallback 漸層）。
4. **觸控目標 < 44px** — 所有手機畫面。
5. **`--text-muted` 小字對比不足 WCAG AA**（≈4.0:1 on `--bg-primary`，用於 11–12px）。
6. **半形標點 / 台灣用語** — 全站；已有 backlog `chore-zhtw-error-string-punctuation-sweep`。

### 分流（能不能修）
**🟢 Tier 1 — 立即修（高信心、機械式、唯一正解）**
- 中文字型 → Noto Sans TC（**改元件一次 = 全站生效** + 補各畫面硬編 DM Sans 標題）。
- 所有 HIGH 內容裁切/溢出（C3-M、A6-M、B4-D、B8-D/M、B2-M、B5-M、E2-M、D1-D 黑字）— 皆 `textGrowth`/`fill_container`/sheet `y` 低風險修正。
- 元件層 bug：`EmptyLibrary-NoQBT` 子節點溢出；建 `Component/TabMobile`（手機 tab 列手刻三份）。
- spec 註記移出 mockup（H badgeDemo、F1-D design-note、B9 inline）→ 獨立框。
- caption 間距 30→45px（Update y）。
- 已確認決策的 stale 畫面：H2-M emoji→lucide（H5-D AC3 已拍板）、F3-M 補第4 tab、G3-M 補金鑰說明。
- 殘留/矛盾 mock：D2-D 5.21>5.20、C5-D autoText enabled:false。

**🟡 Tier 2 — 值得修，但先要拍板設計決策（改的是設計系統本身，含 styles.css）**
- **A. Token 化語意色**：新增 `accent-subtle`/`success·error·warning-tint`/`overlay-scrim`/`chip-bg`/`fallback-gradient`/`text-on-accent`，同步 `.pen ↔ styles.css`。⚠️ 白字按鈕 `#FFFFFF`：`--text-inverse`(#1b2336) 是深色**不適用**藍底按鈕，應對 `--text-primary`(#f2f2f2) 或新增 `--text-on-accent`。
- **B. 對比 a11y**：(a) `--text-muted` 提亮至 ~`#8A93A6`（全站含 code）／(b) 規定 muted 只用 ≥14px、小字改 `--text-secondary`。
- **C. 觸控目標**：(a) 視覺真的放大 ≥44px／(b) 視覺維持、外擴熱區 + spec 註記。

**🟤 Tier 3 — 低優先 / 併 backlog**
- 半形標點 + 台灣用語（儲存路徑 / 遞增·遞減 / 磁力連結 / 「…」）→ 併入 zh-TW sweep backlog，擴及 .pen。
- 8pt 間距 / 圓角 drift（gap 5/6/10/14、radius 6/10/14/20）→ 機會性清理。
- `s11main` → `Component/AdvSearchMain`。

**🔵 Tier 4 — 不是「修」，是「新設計」**
- 缺狀態畫面：error / empty / no-result / loading（A/D/E/F/G 普遍缺）、破壞性操作確認 modal（C2 批次刪除、D 下載操作）。
- **⭐ Epic 12 詳情頁大改**：DoubanSection(#61)、預告片(#60)、串流(#58)、推薦(#57) 全無 .pen 畫面；460px 詳情面板 IA 需重排。

**⚪ 報告提到，但建議「不要修」（避免過度修正）**
- 海報網格背景半切（B1/B2、A2/A3 row clip）= 故意的捲動示意。
- B1 hover 手機版 = 觸控無 hover，已由 bottom sheet 覆蓋。
- C1-D radio vs A6-M segmented = 可接受平台差異。

---

## Part 2 — 流程 A（瀏覽）+ 流程 C（搜尋·設定）

**總評**：token 採用率高、桌機/手機 IA 同步良好、無簡體殘留。最嚴重集中在兩處內容裁切到不可見（C3-M 設定 sheet 塌陷、A6-M 套用/重置 被裁）+ 元件層中文字型誤用 Inter（一處修正全域生效）。

### 主表（依嚴重度）
| 畫面碼 | 問題 | 嚴重度 | 修法 |
|--------|------|--------|------|
| C3-M | 設定 sheet 完全塌陷：`settings-sheet`(RqAc7) 未設高度、子 `settings-content`(rwInP) 設 `fill_container` 循環依賴 → sheet 只剩 70px，三設定項全不可見 | 高 | RqAc7 給固定高 ~380–420px（自 y≈464），或 rwInP 改 `fit_content` |
| A6-M | 篩選 sheet 主動作被裁：固定高 600、內容超高，「媒體類別」切半、套用/重置(xhUXt) 全裁 | 高 | 減少預設晶片排數(3排+顯示更多)或加高 sheet；`fActions` 移出捲動區固定底部 |
| A4-D | 斑馬紋規則錯亂：row-4/6/8 有 `$bg-secondary`，row-1~3/5/7/9 沒有 | 中 | 統一偶數列上色，或全移除只留分隔線 |
| A4-D | 表頭排序指示矛盾：評分欄圖示 `$accent-primary` 但文字 muted；新增日期欄文字+圖示皆 accent | 中 | 評分欄圖示改 `$text-muted`，僅「新增日期」為作用中 |
| A4-D | 日期格式不一致：列表「Jan 15」英文月份 vs C5-D「2026/03/20 14:00」 | 中 | 統一 `2026/01/15` 數字格式 + JetBrains Mono |
| A3-M, C2-M | 海報長寬比失真：poster 高固定 330、手機卡寬 173 → 1:1.9（應 2:3 = 1:1.5）過度裁切 | 中 | 手機 grid override poster 高 ≈260；長期以比例約束取代固定高 |
| A1-M, A3-M, C2-M, C4-M | 觸控 <44px：按鈕~39、分頁40、晶片35、Checkbox20、純文字小目標 | 中 | 手機按鈕統一 44；分頁列 48；小鈕加 padding |
| C2-D, C2-M | 批次「刪除」破壞性操作但無確認對話框 | 中 | 補刪除確認 modal（將刪除 N 項 + 取消/確認刪除） |
| 全部 -M | 手機 `mobile-tabs` 手刻、未複用 Tab 元件，A1/A2/A3-M 複製三份 | 中 | 建 `Component/TabMobile`（active/inactive） |
| A1-D / A3-D / A4-D | 同流程工具列不一致：A3-D 有「待解析 3」badge + 搜尋框手刻；其餘用 SearchInput | 中 | A4-D 補 parse-badge；A3-D 搜尋框改用 `6MxLT` 元件 |
| C4-M | 分類晶片列被右緣裁切：「備份」切半、「匯出/效能」全不可見，無捲動暗示 | 中 | 右緣加漸層 fade 或最後一顆露半 |
| A5-M | 提示 11px `$text-muted` 對比~4.0:1 未達 AA 且過小 | 中 | 12px + `$text-secondary` |
| 多畫面 | 摺線下內容被 clip（A3-D row-2、A4-D row-9 等） | 低 | 可接受；最後一列露半表達可捲 |
| C5-D | `autoText`(自動備份:關閉) `enabled:false` 且與右側開關(開啟)矛盾，殘留節點 | 低 | 刪除或更新為「每日 03:00」並啟用 |
| C1-D | 桌機媒體類別 radio、A6-M 手機 segmented，同功能異模式 | 低 | 可接受平台差異 |
| A2-D, A2-M | 載入骨架無 shimmer/動畫標註 | 低 | 骨架旁加 note 說明 pulse/shimmer |
| C5-D | 備份「操作」欄僅 16px 圖示無文字無 tooltip；刪除與下載同 muted 色 | 低 | 刪除 hover `$error`；spec 註 aria-label/tooltip |

### 設計系統偏移
**字型**：`ButtonPrimary`(DaLcd)/`ButtonSecondary`(L9cIf)/`FilterChip`(v1GYz) = Inter（中文內容）→ Noto Sans TC；C2-D `deleteLabel`/C2-M `deleteL` Inter → Noto Sans TC；C5-D 檔名/大小/時間 + C4 主機位址 placeholder → JetBrains Mono。
**色彩硬編碼**：`#3B82F626`→`accent-subtle`；`#3B82F610`→`accent-faint`；`#2E3B564D`(晶片底 30+處)→`chip-bg`；`#22C55E20`→`success-subtle`；`#1B2336`(=bg-primary)→直接引用 `$bg-primary`；`#FFFFFF`(按鈕字/勾/segmented)→`text-on-accent`；`#000000`50%→`overlay-scrim`；PosterCardHover `#000000AA/66`/`#FFFFFFDD/99/AA`→`overlay-*`。
**對比**：`$text-muted` #808080 on bg-primary≈4.0:1、on bg-secondary≈3.3:1，12px 以下全不達 AA（搜尋 placeholder、表頭、年份、A5-M、C3-D）。提亮 ~`#8A93A6` 或限 ≥14px。
**8pt**：gap 6/10/14、padding 14/[4,10]/[3,8]/[5,12]/3；主版面 16/24/32 正確。

### 覆蓋缺口
A 無載入失敗/連線錯誤畫面；C1-D/A6-M 套用後 0 結果 + 搜尋無結果 缺；批次刪除確認 modal 缺；C5 備份手機版(C5-M) 缺；C1 進階篩選手機版缺；A4 列表手機版(A4-M) 缺或註明桌機限定；備份非成功狀態(進行中/失敗/還原) 缺；C4 連線測試成功/失敗回饋缺。

### 繁中
✅ 無簡體、全形括號正確。⚠️「...」→「…」(低)；「升序/降序」「匯出/匯入」半形斜線；「升序/降序」陸用語→「遞增/遞減」(併 backlog)。
**修復優先序**：①C3-M+A6-M ②元件 Inter→Noto Sans TC ③A4-D 斑馬/排序 ④token 化 ⑤觸控+對比 ⑥補缺口。

---

## Part 3 — 流程 B（詳情與互動）

### 主表
| 畫面碼 | 問題 | 嚴重度 | 修法 |
|--------|------|--------|------|
| B4-D | 詳情面板內容嚴重溢出：`episode-list`(kkMgW) 切半，`加入清單`(kcfJT)/divider2/`file-info`(fQB3r) 全裁 | 高 | 縮減示意集數至 3、synopsis 縮短，或 panel gap 16→12 |
| B8-D | `tech-badges`(UsaEG) 溢出：「英文」半切、「日文」全切（~481>412） | 高 | 徽章拆兩列（Pencil flexbox 不自動換行） |
| B8-D | backdrop `bd-img`(Zyo97)+`overlay`(MN2jI) 用 `fill_container(0)` 但父 `layout:none` → 塌陷 0、漸層沒渲染 | 高 | 固定 `width:460 height:240`（同 B3-D 做法） |
| B8-M | `tech-badges`(ZT4jq)「英文」全切（417>350） | 高 | 拆兩列 |
| B2-M | `context-sheet`(wQcxW) 底超出 54px、「取消」被切 | 高 | y 494→440 |
| B5-M | `detail-ctx-sheet`(y5WYJ) 底超 55px、「取消」被切 | 高 | y 584→529 |
| B3-D | `file-info`(cMrM6) 底切，101px 只露 23px | 中 | synopsis 縮短或 gap 20→16 |
| B3-D/B5-D/B8-D/B3-M/B8-M/B9-M | 中文標題「你的名字」用 DM Sans，與 B4-D 的 Noto Sans TC 不一致 | 中 | 統一 Noto Sans TC 28/700 |
| B3-D/B5-D/B3-M/B9-M | 按鈕 label（ButtonPrimary/Secondary）Inter，中文 fallback | 中 | 元件 label 改 Noto Sans TC |
| B3-M | 與桌面不同步：缺原文「Your Name」、meta 缺年份、缺檔案資訊 | 中 | 補 original-title/年份/精簡檔案資訊 |
| B3-M/B6-M/B7-M/B8-M/B9-M | `menu-btn` 32×32<44；字幕按鈕 36；手動編輯 32 | 中 | 44×44 熱區 |
| 全部 B | `$text-muted` 對比~3.5:1 用於 11–12px | 中 | ≤12px 改 `$text-secondary` |
| B7-M | `file-hint` 9px、B6-M 10px 過低 | 中 | 手機最小 11px、檔名 truncate |
| B6-D/B6-M | 「手動編輯」純文字無 affordance | 低 | 加 pencil icon / underline / ghost |
| B4-D | 季數 chip 高~29px | 低 | 手機版 ≥36px |
| B9-D | spec 半形標點 | 低 | 併 zh-TW sweep |
| B9-M | `sheet-content` 底溢 5px | 低 | gap 16→14 |
| B7-D | `pending-desc` 固定 width 300 | 低 | `fixed-width`+`fill_container`+center |

**亮點（保留）**：B2/B5 桌機手機完全同步（含刪除 `$error` + 分隔線）；B6「搜尋 Metadata」CTA 44px、層級正確；B3-D 視覺層級健康（28/700→14→meta→13）；無簡體。

### 設計系統偏移
Fallback 漸層 `#4338CA→#6D28D9→#7C3AED` 重複 6 次（B9 自稱 token 但文件無）→ 新增 `fallback-gradient-1/2/3`；B8 backdrop `#1E3A5F→#2D1B69`、overlay 終點 `#1A1A2E` ≠ `$bg-primary`(#1B2336) 有接縫 → 改 bg-primary；同值未綁變數（B9-M frame #1B2336、handleBar #808080、parse-badge #1B2336）；TechBadge 色 `#60A5FA/C084FC/4ADE80/FBBF24`+18% → badge 語意變數；透明色 `#00000066/AA`、`#FFFFFF18/CC` 散落；間距 drift gap5/6/10/14、`panel-content` gap20+padding[20,24] 影響最廣；尺寸 drift B4-D subtitle-action 37px、字幕按鈕桌34 vs 手36；B9-D spec 底 `#0F1420` 非 token（spec 豁免）。

### 覆蓋缺口
- **B4 影集詳情手機版(B4-M)** — 流程 B 最關鍵缺口，季數選擇器+集數清單手機排佈，優先補。
- **Epic 12 新功能無稿**：DoubanSection(#61)/預告片(#60)/串流(#58)/推薦含「已有」徽章(#57)；460px 面板 IA 建議重排序：標題→meta→徽章→動作→預告片→簡介→串流→豆瓣短評→推薦→檔案資訊。
- 外連 affordance：詳情面板無 TMDB/豆瓣外連入口（「TMDb」只是純文字）。
- 外部資料 fail-soft 示意缺（建議比照 B9 做 spec screen）。
- 網路層錯誤態（TMDb 逾時/API 錯誤可重試）缺。
- 無障礙標註：星等 aria-label、徽章文字替代、32px 圓鈕的 44px 熱區規則入 spec。
- B1 hover 手機版：**不需補**（觸控無 hover，已由 B2-M bottom sheet 覆蓋）。

---

## Part 4 — 流程 D（下載）/ E（掃描）/ F（字幕）/ G（AI 字幕）

### 主表
| 畫面碼 | 問題 | 嚴重度 | 修法 |
|--------|------|--------|------|
| D1-D | 多處 `fill:#000000` 深底不可見（search-icon qvxDb、placeholder N4oxE、副標 ESohm、chevron NUIiw/u4Y27、省略號 p9Kd9、「每頁」2mRNV、「筆」xllL9、chevron MThKY） | 高 | 改 `$text-secondary`/`$text-muted` |
| D1-D/D2-D/D1-M | 下載卡無「暫停/繼續/刪除/重試」按鈕，僅展開 chevron；錯誤卡無重試入口 | 高 | 卡片右側加 icon 組：暫停/繼續(secondary)、刪除(`$error` 需確認)、錯誤卡「重試」(primary) |
| D1-M | chips padding[5,10] fontSize11 高~26px；分頁~26px <44 | 高 | chips padding[10,12]+；分頁 ≥40×40 |
| E2-M | 掃描進度 sheet(qhdBM) 懸空：y484 高289 底773、距 844 縫 71px | 高 | 高度補足或 y 下移貼齊 844 |
| F1-D | `design-note`(y5mCj) 疊在 dialog 上，違反 spec 獨立 | 高 | 移出或建 F 系 spec 獨立畫面 |
| G1-D/G1-M/G2-D/G2-M/G3-D/G3-M | 中文標題用 DM Sans（「AI 用語校正預覽」「英文音軌轉錄」等）→ CJK fallback | 高 | 全部改 Noto Sans TC |
| D1-D | placeholder「搜尋磁連結...」疑漏字、且與 D2-D「搜尋種子...」不一致 | 中 | 統一 |
| D2-D | 展開卡 stroke `#14352B` 深綠無 token 語意不明 | 中 | `$border-subtle` 或 accent |
| D1-M | tab「下載中」active 僅藍字無底線；E1-M/E4-M 有 2px indicator 不一致 | 中 | 補 2px indicator |
| E1-D | 表單只有正常態，資料夾無效/排程停用未呈現 | 中 | folder-row 錯誤變體 |
| E2-D | 62%(dTuV5) absolute 疊放脆弱（F3-D 用 flex） | 中 | 改 F3-D `ltsF8` flex 結構 |
| E2-D | 「正在處理:/預估剩餘:」半形冒號（F3-D 全形）；檔名(eCG0e) `$text-muted` 11px | 中 | 統一全形「：」；檔名改 `$text-secondary` |
| E1-M | folder 編輯/刪除 icon 14px 無容器 | 中 | 32–44px 容器（刪除 `$error` ✓） |
| F1-D | HI badge `#F59E0B25/40` 字 8px | 中 | ≥10px；tint token |
| F1-D/F2-D | 數值欄（92%、1,247）Inter，應 JetBrains Mono | 中 | 統一 Mono |
| F2-D | 重試按鈕(NcmTu) 僅 10px icon 無 label | 中 | icon+「重試」文字 |
| F3-D/F3-M | 「失敗 2」無法點看清單/重試 | 中 | 失敗統計改可點連結 |
| F3-M | 頂部 tab 只 3 個，缺「待解析」 | 中 | 補第 4 tab |
| F1-M | 搜尋按鈕/框 36px <44 | 中 | 提高至 44 |
| G2-D | 副標「星際效應 (2014)」Inter | 中 | Noto Sans TC |
| G1-D/G1-M | diff 勾選 18px 無容器；「全部選取」純文字 12px | 中 | 勾選 44px 熱區 |
| G3-M | 缺「(需要 Claude API 金鑰)」說明，與 G3-D 不同步 | 中 | 補 |
| D2-D | 已下載 5.21GB > 總 5.20GB 矛盾 | 低 | mock 改 ≤ |
| D3-M | 「主機:/狀態:」半形冒號 | 低 | 全形「：」 |
| E3-D/E3-M | toast 良好；auto-dismiss 進度條無暫停 | 低 | 註記 hover 暫停 |
| E4-D/E4-M | 「狀態: 未比對」半形冒號 | 低 | 全形「：」 |
| F2-D | 列底 `#22C55E08/#3B82F610/#EF444408` 硬編碼 | 低 | tint token |
| F3-D/F3-M | 背景 logo「Vido」Inter（他處「vido」DM Sans） | 低 | 統一 vido/DM Sans 700 |
| F3-M | batch-sheet y484 高352 底836 距 8px | 低 | 高度 +8 |
| G2-D | 主進度僅步驟器內 4px 細條 | 低 | 面板頂加整體進度或註記 |
| G2-M | 62% Inter（E2/F3 用 Mono） | 低 | 統一 Mono |
| G3-D | 「稍後決定」G3-D 僅文字（G3-M 已 48px ✓） | 低 | G3-D 比照 44px |

### 設計系統偏移
**狀態徽章/進度色硬編碼且偏離 token**：下載中 `#1E3A5F`/`#60A5FA`→accent-hover/primary+tint；已完成 `#14352B`/`#34D399`→`$success`#22C55E（#34D399 非 token）；錯誤 `#4C1D1D`/`#F87171`/`#7F1D1D`→`$error`#EF4444；已暫停 `#3B3520`/`#FBBF24`→`$warning`#F59E0B；進度軌 `#1E293B`→`$bg-tertiary`#2E3B56 → **新增 `success/error/warning/accent-tint` + `-fg` 亮階**。
**tint 散落**：`#3B82F620`、`#22C55E40/30/20/08`、`#F59E0B40/30/25`、`#EF444440/30/20/08`。
**圓角 drift**：6（幾十處）/10/14/20；只 F 用 `$radius-*` 變數，D/E/G 裸數字 → 變數化。
**間距 drift**：gap10/6/3/2、padding[5,10]/[6,14]/[3,8]/[2,6]/[10,14]/14。
**字型 drift**：G 中文 DM Sans/Inter；數值 Inter vs Mono 雙軌；分隔點「·」掛 Inter；F3 logo Vido Inter。
**對比**：`$text-muted`#808080 11–12px 未達 AA → 提亮 ~`#8C95A8` 或小字 `$text-secondary`。
**繁中標點**：半形冒號 D2-D/D3-M/E2-D/E2-M/E4；「保存路徑」→「儲存路徑」；「磁連結」→「磁力連結」；「无码」「星際穿越」簡體 mock（檔名情境可接受）。

### 覆蓋缺口
- **D**：D3 只有手機 → 缺 D3-D；D2 只有桌面 → 缺 D2-M；無載入骨架/空態；卡片無操作。
- **E**：掃描整體失敗態缺；E1 驗證錯誤/零資料夾空態缺。
- **F**：搜尋無結果空態、搜尋中 loading、provider 全失敗、F3 批次完成總結+失敗清單+重試 缺。
- **G**：AI 失敗態全缺（轉錄/翻譯/金鑰無效）；G1「0 處修正」空態缺；G2 步驟 error 缺。
- 響應式：D2/D3 單端缺口、F3-M 少 tab、G3-M 少金鑰說明。
- **優先**：①D1-D 黑字 ②E2-M 懸空 ③G 中文 DM Sans ④下載卡操作+D2-M/D3-D ⑤token 化。

---

## Part 5 — 流程 H（首頁）/ I（進階搜尋）/ J（規格）+ Design System + 標註層

### 主表
| 畫面碼 | 問題 | 嚴重度 | 修法 |
|--------|------|--------|------|
| H/I/J 全部 | 系統性字型違規：畫面內所有中文（hero「你的名字」、「熱門電影」、modal/chips、I 全流程）設 DM Sans，無 CJK fallback；連 Design System Reference 自己都寫「Noto Sans TC (zh-TW body)」 | 高 | 含中文 text node 改 `Noto Sans TC`；DM Sans 留純英文標題/logo |
| H1-D | `heroDesc`(2itAU) 未設 textGrowth，單行 681px 溢出 600 的 heroContent 被裁 | 高 | `textGrowth:"fixed-width"` + `width:"fill_container"` |
| H1-D | `badgeDemo`（Availability Badges 標註）塞 H1 mockup 底部，違反 spec 獨立；H2-M「←水平捲動」、H4-D「animate-pulse…」同類 inline 標註 | 中 | 移出併 spec 框(H5/J1)或自成小框 |
| I1-D | 結果網格第 5 張卡片被裁 72px（x944+220>1092） | 中 | 調欄數/卡寬讓欄位整除 |
| H1-D | explore block 每排第 6 張只裁 24px，太小像對位誤差（H5-D spec 寫「最右露半表示可捲」） | 中 | 明確裁半張或整除排列 |
| 元件庫 | `EmptyLibrary-NoQBT`(fSKuT) 按鈕(oJnQS 96×40) 子節點(AHW4u 27×54) 垂直溢出，DS Reference + Component Library 兩處重現 | 中 | 修元件本體（縮圖示/文字或加大按鈕高） |
| H2-M | 區塊標題帶 emoji 🎬📺，但 H5-D spec AC3 已確認「emoji→lucide Film/Tv」，畫稿落後決策 | 中 | H2-M 移除 emoji，與 H1-D/AC3 對齊 |
| H/I/J 標註層 | caption 與 frame 間距僅 30px(blockY+90)，慣例 45px(blockY+75)，會撞 Pencil 圖框名 chrome | 中 | caption y 上移 15px（Update 改 y） |
| H2-M | 缺第三區塊（近期台灣院線），D/M 不同步 | 低 | caption 註明「行動版僅示意前兩區塊」 |
| I1-D/I4-M/H1-D | 半形標點：「快速篩選:」「篩選條件:」「套用篩選 (48 部結果)」、heroDesc「夢...」（應「……」）；I3 卻正確用全形「：」 | 低 | 統一全形「，。：（）……」 |
| H2-M/H3/I3 | 觸控偏小：tabs 40px、close 18–20px、carousel dots 8px 無文字替代 | 低 | ≥44px；dots 加 aria-label |
| I1-D | `s11main`(M2mm3) 是 reusable 卻無 `Component/` 前綴，混進元件清單 | 低 | 改 `Component/AdvSearchMain` |

### 設計系統偏移
`#1a1a3e` hero 背景(H1-D VSORG、H2-M j5EVJ) 非 token → image fill 或掛 token；`#ffffff` 按鈕/chip 文字 → 補 `text-on-accent`（⚠️ `styles.css` 的 `--text-inverse` 是 #1b2336 深色，**不適用**藍底按鈕）；`#ffffff1a`(H1-D heroBtn2)/`#00000000`(H3 cancelBtn、I1 presetAdd)/`#00000080`(I4-M overlay) → 集中定義；非 scale 數值：cornerRadius 20(H3/I1)、16(I1)；gap 6(H3/I3)、20(H3)、44(H5/J1)；padding[5,14]/[3,8]。

### 覆蓋缺口
- H 缺「探索區塊外部資料失敗」fail-soft 錯誤態（H5-D AC2 有文字規格 isError→return null，但無 mockup）。
- I 缺「搜尋無結果」態：I2 建議下拉只有快樂路徑，缺「載入中」「查無結果」；I1 結果區缺空態。
- H3/I2/I3 caption 無平台後綴（共用 overlay 可接受，但與命名格式不一致）。

### 做得好（保留）
- H4-D 載入骨架完整，「每區塊獨立 skeleton，非全頁 spinner」對齊 fail-soft。
- H5-D / J1-D spec 獨立成框，符合畫布慣例 §5⑤ 與 Rule 21 設計合約格式。
- H/I/J 標註層樣式（DM Sans 24/700 #222222 標題、Noto Sans TC 14 #666666 描述、#888888 caption）與 1540px 欄距完全符合慣例。
- I4-M 套用按鈕 48px、I2 列高 ≥44px，觸控合格。

---

*記錄者：BMAD party（A+C / B / D-G / Sally H-J）。未修改任何 `.pen` 節點，無需重產截圖。下一步：作為 Phase 0 `00-redesign-brief.md` 的證據輸入。*
