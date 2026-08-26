# H1-v3 首頁改版 — Pencil Inline AI Agent 提示詞（貼進 Pencil 用）

在 ux-design.pen 新增三個 frame，畫「首頁 v3」設計稿。全部用絕對座標放在
document root，不要動任何既有節點。

## 版位（新的一列，在 flow-h v2 列正下方）

1. frame `H1-D-v3 · 首頁 v3（桌面）`：x=17040, y=34600, width=1440，
   vertical layout，高度自動（預估 ~2300）。
2. frame `H7-D-v3 · 首頁 v3（降級態）`：x=18600, y=34600, width=1440。
3. frame `H2-M-v3 · 首頁 v3（手機）`：x=20160, y=34600, width=390, height=844,
   clip=true。

## 色盤（夜行——直接用這些 hex，不要用文件裡的 $ 色彩變數，它們是舊版藍色系）

- 頁面底 #0c1512、卡片面 #132320、互動面 #1b302b、hairline #274039
- 主文字 #eae4d6、次要 #a8b3ac、弱化 #8fa096
- 金（強調/可按）#c9a24b、金 hover #e0be72、金上墨字 #14161a
- 琥珀字 #e8b04b、琥珀 tint #d4763f1f、綠字 #8fd3be、綠 tint #6fbfa81f
- 錯誤字 #e08a76、遮罩 #000000b3
- 字體一律 Noto Sans TC；數字用 JetBrains Mono

## 桌面版內容（H1-D-v3，由上而下）

左側直接引用元件 Component/HomeSidebar-v2 當側欄；右側內容欄（其餘寬度，
頁面底 #0c1512）依序：

### 1. 讀數帶（新設計，本稿主角）
一條橫帶，內容寬度 1152 置中，上下 padding 16。單行 4 格等寬
（justifyContent: space_between 或 4 格 fill_container），每格是可點的門：
圖示 16px ＋ 11px 標籤（#8fa096）＋ 20px mono 數字（#eae4d6）。格與格之間
用 1px hairline #274039 分隔。四格內容：
- 圖示 captions，標籤「繁中字幕」，數字「42/55」
- 圖示 check-check，標籤「今天處理」，數字「3 部」
- 圖示 triangle-alert，標籤「需要注意」，數字「2 部失敗 · $1.2/$5」
  ——這格的數字改用琥珀字 #e8b04b
- 圖示 activity，標籤「進行中」，數字「2 個任務」
格高至少 44。帶的底色 #132320、圓角 12。

### 2. 自家片庫 hero（靜止，無自轉）
高 400、滿內容欄寬。底是一張深色電影劇照（用 stock 圖：dark cinematic
mountain landscape），上面由下往上壓 #0c1512 → 透明的漸層。左下資訊區
（內縮：左右 pad 對齊 1152 內容欄）：
- 11px「最新入庫」標籤（#8fa096，字距寬鬆）
- 36px/700 片名「沙丘：第二部」（#eae4d6）
- 一列狀態徽章：藥丸「繁中字幕 ✓ 已就緒」（綠 tint 底 #6fbfa81f、綠字
  #8fd3be、11px）＋「2024 · 電影」次要文字
- 金色按鈕「查看詳情」（#c9a24b 底、#14161a 字、圓角 8、高 40）
右下角：一枚 #000000b3 遮罩藥丸，內含 chevron-left、三顆 8px 圓點
（第一顆 #eae4d6 寬 24、其餘 #eae4d6 半透明）、chevron-right——手動切換，
不是自動輪播。

### 3. 最近新增列
20px/600 標題「最近新增」＋右側琥珀藥丸「整理中 · 3」（tint 底
#d4763f1f、字 #e8b04b、11px）。下面一排 6 張 Component/PosterCard-v2
引用（ref），卡寬 160。

### 4. TMDb 探索（頁尾、已濾掉已有）
18px/600 標題「熱門電影」＋一排 6 張 PosterCard-v2 ref。標題下加一行
11px 弱化說明「已擁有的作品不會出現在這裡」。

## 降級態（H7-D-v3）
複製桌面版結構但：hero 用一張自家劇照照常顯示（降級不影響自家內容）；
第 4 區整塊不存在，改為頁尾一條琥珀說明列：tint 底 #d4763f1f 圓角 8、
文字「探索內容目前無法載入（TMDb 未設定或無法連線） 前往連線設定」
（#e8b04b 14px，「前往連線設定」加底線）。讀數帶、hero、最近新增照常
——降級態的頁面必須看起來一樣完整。

## 手機版（H2-M-v3，390×844）
無側欄。由上而下：讀數帶改 2×2 田字格（每格高 64）；hero 高 250（同樣
靜止＋圓點）；最近新增橫向捲動一排（露 2.5 張卡）；底部固定 tab bar
（面 #132320、上緣 hairline，五格：house 首頁[金色 active]、library
媒體庫、activity 活動、download 下載、ellipsis 更多，24px 圖示＋11px
標籤 #8fa096）。TMDb 區在手機稿裡省略（超出 844 高度即可不畫）。

## Schema 注意事項（重要，照做不要猜）

- 對齊值只有 start / center / end；justifyContent 另有 space_between。
  沒有 stretch——子元素要撐滿用它自己的 width: "fill_container"。
- width 為 fill_container 的 text 節點必須同時設 textGrowth: "fixed-width"。
- 顏色只吃 hex（含 8 位透明度），沒有 transparent 關鍵字。
- 圖示名用上面指定的（captions / check-check / triangle-alert / activity /
  chevron-left / chevron-right / house / library / download / ellipsis）——
  這些都已存在於本文件。
- 每個節點都要設有意義的 name。

## 淘汰舊稿（第一段：標記，不刪除）

三張 v3 畫完之後，把下列五個既有 frame **改名**加上淘汰標記（只改 name，
不要動內容、位置或刪除——它們要留著當對照，等 v3 實作上線後才整批刪）：

- id `yixu1` → `〔已淘汰→v3〕H1-D-v2 · 首頁 v2（桌面）`
- id `nnGs6` → `〔已淘汰→v3〕H4-D-v2 · 首頁載入骨架 v2`
- id `Z7OJB` → `〔已淘汰→v3〕H5-D-v2 · 首頁 v2（最近新增空狀態）`
- id `xCQA7` → `〔已淘汰→v3〕H6-D-v2 · 首頁 v2（區塊載入失敗）`
- id `uCfjb` → `〔已淘汰→v3〕H2-M-v2 · 首頁 v2（手機）`
