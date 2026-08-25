---
name: Vido
description: Dark, dense NAS media manager whose readouts never flatter.
colors:
  midnight: '#1b2336'
  slate-raised: '#24304a'
  slate-hover: '#2e3b56'
  hairline: '#374461'
  signal-blue: '#3b82f6'
  signal-blue-hover: '#60a5fa'
  signal-blue-pressed: '#2563eb'
  signal-blue-wash: '#3b82f626'
  signal-blue-tint: '#3b82f61f'
  signal-blue-text: '#60a5fa'
  running-green: '#22c55e'
  running-green-tint: '#22c55e1f'
  fault-red: '#ef4444'
  fault-red-tint: '#ef44441f'
  fault-red-text: '#f87171'
  asked-amber: '#f59e0b'
  asked-amber-tint: '#f59e0b1f'
  note-cyan: '#06b6d4'
  note-cyan-tint: '#06b6d41f'
  read-white: '#f2f2f2'
  read-grey: '#b3b3b3'
  read-dim: '#a0aabe'
  read-disabled: '#6e7891'
  ink-inverse: '#1b2336'
  on-signal: '#ffffff'
  scrim: '#000000b3'
typography:
  display:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '2.25rem'
    fontWeight: 700
    lineHeight: 1.1
  headline:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.5rem'
    fontWeight: 700
    lineHeight: 1.25
  title:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.125rem'
    fontWeight: 600
    lineHeight: 1.4
  body:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.875rem'
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.75rem'
    fontWeight: 500
    lineHeight: 1.4
  readout:
    fontFamily: 'JetBrains Mono, Consolas, Monaco, monospace'
    fontSize: '0.8125rem'
    fontWeight: 400
    lineHeight: 1.4
rounded:
  sm: '4px'
  md: '8px'
  lg: '12px'
  xl: '16px'
spacing:
  xs: '4px'
  sm: '8px'
  md: '12px'
  lg: '16px'
  xl: '24px'
  2xl: '32px'
components:
  button-primary:
    backgroundColor: '{colors.signal-blue}'
    textColor: '{colors.on-signal}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
    typography: '{typography.body}'
  button-primary-hover:
    backgroundColor: '{colors.signal-blue-hover}'
  button-primary-active:
    backgroundColor: '{colors.signal-blue-pressed}'
  button-secondary:
    backgroundColor: '{colors.slate-hover}'
    textColor: '{colors.read-white}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  button-outline:
    backgroundColor: 'transparent'
    textColor: '{colors.read-white}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  button-ghost:
    backgroundColor: 'transparent'
    textColor: '{colors.read-white}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  card:
    backgroundColor: '{colors.slate-raised}'
    rounded: '{rounded.lg}'
    padding: '24px'
  input:
    backgroundColor: '{colors.slate-raised}'
    textColor: '{colors.read-white}'
    rounded: '{rounded.md}'
    padding: '8px 12px'
  badge-running:
    backgroundColor: '{colors.running-green-tint}'
    textColor: '{colors.running-green}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  badge-asked:
    backgroundColor: '{colors.asked-amber-tint}'
    textColor: '{colors.asked-amber}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  badge-fault:
    backgroundColor: '{colors.fault-red-tint}'
    textColor: '{colors.fault-red-text}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  nav-item-active:
    backgroundColor: '{colors.signal-blue-wash}'
    textColor: '{colors.read-white}'
    rounded: '{rounded.md}'
    padding: '8px 10px'
---

# Design System: Vido

> 章節標題與 frontmatter 的 token 名稱保持英文——DESIGN.md 規格靠精確標題解析，token 名稱等同程式碼識別碼。其餘內文為繁體中文，與 PRODUCT.md 一致。

## Overview

**Creative North Star：「誠實的讀數」（The Honest Readout）**

Vido 會在使用者不在的時候，於別人的 NAS 上持續跑上好幾分鐘，而且花的是真的錢。這個介面上的每一塊，本質都是「別處正在發生的事」的讀數——而**會諂媚的讀數比沒有讀數更糟**。一個不知道進度卻顯示 90% 的儀表不是體貼，是說謊。整套視覺系統存在的理由，就是讓背景工作的真實狀態在昏暗房間裡一眼可讀，而使用者不需要「相信」任何東西。

這導致一個刻意不華麗的介面。文字小而密——全 app `text-sm` 出現 589 次、`text-xs` 254 次，對比 `text-4xl` 只有 7 次——因為它的任務是把一整座真實片庫的狀態塞進一個畫面，不是製造第一印象。深度幾乎全靠三階背景色而非陰影（色階 525 次 vs 陰影 11 次），所以眼睛讀得出結構，卻不會有任何東西感覺被墊高或裝飾過。強調色被配給到近乎稀有：訊號藍從不填滿導覽項目，只以 15% 淡洗帶過。

這套系統唯一願意花成本的地方是**誠實**。狀態色是資料不是裝飾，而且語意固定：綠＝正在發生、琥珀＝你要求了但沒發生、不出現＝你沒要求。這組詞彙比任何程度的精緻都值錢，任何「視覺改善」都不准把它弄模糊。

**Key Characteristics：**

- 只有暗色單一主題——沒有淺色模式，連骨架都沒有，只留了一行註解
- 刻意密集：內文 14px、標籤 12px，這就是工作區間
- 深度來自色階，不是陰影
- 強調色配給制；語意色只保留給狀態，絕不用於強調
- 對比度是硬性關卡（WCAG AA 4.5:1），全系統只有一個有記錄的例外

## Colors

冷調、低彩度的午夜色盤，每一個飽和色都有明確任務，沒有任何顏色是為了好看而存在。

### Primary

- **訊號藍 Signal Blue**（`--accent-primary`, #3b82f6）：唯一的品牌色，而且用得很省。主要按鈕、焦點框、active 淡洗。它只在「使用者要按的控制項」上以實心出現——**絕不當作閱讀區的背景**。
- **訊號藍 Hover**（`--accent-hover`, #60a5fa）／**Pressed**（`--accent-pressed`, #2563eb）：hover 變亮、按下變暗。方向是刻意的：滑過去是往光的方向抬起，按下去是往下沉。
- **可讀訊號藍 Signal Blue Text**（`--accent-text`, #60a5fa）：當強調色需要**被閱讀**而不是被按的時候用。500 權重的藍在這些底色上當內文會過不了 AA；這個較亮的階是 AA-safe 的替代品，**與 `--accent-primary` 不可互換**。

### Secondary — 狀態詞彙

這四個不是色盤，是一組**語意固定的詞彙**，而且它們的意思是承重的。

- **執行綠 Running Green**（`--success`, #22c55e）：**現在正在發生**。絕不用於「做完了、已結束」，也絕不當作一般的肯定。
- **待命琥珀 Asked Amber**（`--warning`, #f59e0b）：使用者要求了某件事，而它**沒有**在發生。這是整套系統裡最重要的顏色——它是唯一能夠說出「系統與你的意願不一致」而不把它藏起來的方式。
- **故障紅 Fault Red**（`--error`, #ef4444）：壞掉了。當文字讀時用 **可讀故障紅**（`--error-text`, #f87171），那是 AA-safe 的階。
- **提示青 Note Cyan**（`--info`, #06b6d4）：說明性的，不帶評價。用於「告知但不暗示好壞」的通知。

每個都有一個 `*-tint` 夥伴（約 12% 不透明度）作為徽章與藥丸的底色。飽和值給字或圖示，tint 給它背後的面。實心用量約為 tint 的 4 倍——它們讀起來是深底上的小色標，不是大片彩色面板。

### Neutral

- **午夜 Midnight**（`--bg-primary`, #1b2336）：頁面本身。所有東西都坐在這上面。
- **抬升板岩 Slate Raised**（`--bg-secondary`, #24304a）：卡片、輸入框，任何「是一個獨立物件而非頁面本身」的東西。225 次使用——這是主力。
- **互動板岩 Slate Hover**（`--bg-tertiary`, #2e3b56）：互動層。hover 填色、次要按鈕、chip 底色。300 次使用。
- **髮絲線 Hairline**（`--border-subtle`, #374461）：分隔線與物件外框。比互動板岩高一階，所以在三種底色上都讀得出來。
- **主要文字 Read White**（`--text-primary`, #f2f2f2）／**次要 Read Grey**（`--text-secondary`, #b3b3b3）／**弱化 Read Dim**（`--text-muted`, #a0aabe）：三個閱讀權重。Read Dim 是從 #808080 改過來的，因為那個值在 3.3–4.0:1 過不了 AA；#a0aabe 在三種底色上任何尺寸都清得過 4.5:1。
- **停用文字 Read Disabled**（`--text-disabled`, #6e7891）：**有記錄的唯一例外**——實測 3.55:1，刻意低於 AA，**只准用在停用的控制項上**（WCAG 1.4.3 對 inactive UI component 有豁免）。它不是第四個閱讀權重，絕不可承載說明文字。

### Named Rules

**固定詞彙規則（The Fixed Vocabulary Rule）** 綠＝正在發生。琥珀＝你要求了但沒發生。不出現＝你沒要求。狀態色不得被挪用為強調、裝飾或分類——它一旦出現，就是在對「狀態」做出主張，而那個主張必須為真。

**配給強調規則（The Rationed Accent Rule）** 訊號藍只填滿「要被按的控制項」。僅僅是「目前所在」的狀態——active 導覽項、選取的列——用 15% 淡洗（`--accent-subtle`），絕不用實心填滿。全 app 閱讀文字背後的實心強調色背景數量是 **0**，而那正是預期的數字。

**兩種藍規則（The Two Blues Rule）** `--accent-primary` 給人按，`--accent-text` 給人讀。換過來會無聲地破壞對比度。紅色也有同一組分工（`--error` / `--error-text`）。

## Typography

**內文字體：** Noto Sans TC（後備 `-apple-system`、`BlinkMacSystemFont`、`Segoe UI`、`Roboto`、sans-serif）
**讀數字體：** JetBrains Mono（後備 Consolas、Monaco、monospace）

**個性：** 一套人文主義無襯線扛下所有閱讀工作，加一套等寬字專門留給「必須對齊或互相比較的數字」。**沒有 display 字體、沒有編輯式的字體配對**——字體系統的任務是在昏暗房間裡小尺寸也讀得清楚，第二種個性只會礙事。

### Hierarchy

- **Display**（700, 36px, 1.1）：罕見。只用於詳情頁 hero 標題——全 app 7 次。**不是一般的標題層級。**
- **Headline**（700, 24px, 1.25）：頁面標題（`設定`、`活動`）。一個畫面一個。
- **Title**（600, 18px, 1.4）：區段與卡片標題。
- **Body**（400, 14px, 1.6）：預設值，而且是全 app 用量遙遙領先的尺寸。說明、列表列、表單標籤，幾乎所有東西。
- **Label**（500, 12px, 1.4）：徽章、藥丸、metadata、表頭、次要註記。
- **Readout**（400, 13px, 等寬）：計數、進度數字、位元組大小、ID、檔案路徑——任何使用者可能跨列比較、或當成資料讀的東西。

### Named Rules

**預設小字規則（The Small-By-Default Rule）** 14px 是內文、12px 是標籤。伸手拿更大的尺寸，等於主張「這段文字比頁面的實際內容更重要」——而它通常不是。實測分布 589 次 `text-sm` 對 7 次 `text-4xl` 是**預期的形狀，不是待修正的意外**。

**比較才用等寬規則（The Mono-For-Comparison Rule）** 如果同一個值的兩個實例可能被沿著欄位上下對照，就用等寬。散文永遠不用等寬；使用者只掃一眼、不會拿去比的數字也不需要。

## Layout

App 是「固定左側軌 ＋ 流動內容欄」。左側軌展開 240px、收合 64px，狀態存在 `localStorage`；在 `sm` 斷點以下改為底部分頁列加 More sheet。

內容頁以 `max-w-7xl`（1280px）容器搭配 `mx-auto` 置中，這是**頁面內容**的規則。**它不是「本身就有側欄的頁面」的規則。** 把「包含側欄的整塊」加上寬度限制再置中，會讓那個側欄脫離 App 側軌，在兩個導覽之間留下一道死掉的垂直空白；設定區在 1920px 下實地踩過這個坑——子導覽整整離開它該貼著的側軌 200px。**巢狀側欄的版型，根層不設限也不置中，寬度限制搬到內容格並且靠左對齊。**

間距走 Tailwind 的 4px 基準：4 / 8 / 12 / 16 / 24 / 32。卡片內距 24px，輸入框 8px 垂直、12px 水平，列表列垂直 8–12px。行動裝置觸控目標維持最小 44px。

### Named Rules

**貼齊側欄規則（The Flush Sidebar Rule）** 側欄要貼齊它旁邊的東西。只有內容拿閱讀寬度限制，而且靠左對齊，讓每一欄共用同一條左邊線——多出來的寬度收在右邊，讀起來是頁面留白，不是一個洞。

## Elevation & Depth

**這套系統近乎扁平，用色調分層而非陰影。** 三階背景——午夜、抬升板岩、互動板岩——幾乎包辦所有結構工作：全 app 實測 525 次色階使用對 11 次陰影使用。**一張卡片之所以是卡片，不是因為它有陰影，而是因為它比頁面亮一階。**

陰影確實存在也有在用，但只用在「真的浮在頁面之上」而非「坐在頁面上」的東西。它們比淺色主題該有的更重（alpha 0.3 到 0.6），因為在深色底上，柔和的陰影等於看不見。

### Shadow Vocabulary

- **`--shadow-sm`**（`0 1px 2px rgba(0,0,0,0.3)`）：有填色、可按的控制項——主要／次要／破壞性按鈕。剛好夠讀成一個實體控制項。
- **`--shadow-md`**（`0 4px 8px rgba(0,0,0,0.4)`）：`Card` 基本元件。
- **`--shadow-lg`**（`0 8px 16px rgba(0,0,0,0.5)`）：罕見的中間階。
- **`--shadow-xl`**（`0 12px 24px rgba(0,0,0,0.6)`）：**只給真正的覆蓋層**——Dialog、Sheet、詳情頁 hero。4 次使用，這個上限是刻意的。

### Named Rules

**色調優先規則（The Tone-First Rule）** 伸手拿陰影之前先伸手拿背景階。如果一個面需要讀起來與頁面分離，它就是抬升板岩。只有當這個面真的浮在它所覆蓋的內容之上時，才加陰影。

## Shapes

柔和圓角，絕不做成藥丸形，也絕不銳利。四個圓角階存在，但一個獨大：**8px（`--radius-md`）涵蓋按鈕、輸入框、導覽項與多數互動面，實測 98 次**。12px（`--radius-lg`）屬於卡片與較大的容器（36 次）。4px（`--radius-sm`）給小色標——徽章、chip、標籤（32 次）。16px（`--radius-xl`）**只用了一次**，應當視為實質未使用，而非「可選階」。

邊框是 1px 髮絲線，**只出現在色階本身不足以分開兩樣東西的地方**——外框按鈕、靜止狀態的輸入框、表格與列表分隔線。有填色的面不會同時再給邊框。

### Named Rules

**單一圓角規則（The One Radius Rule）** 沒有理由就用 8px。卡片用 12px 是因為它夠大、8px 會讀起來偏銳；徽章用 4px 是因為它夠小、8px 會讀起來偏圓。其他一律 8px。

## Components

### Buttons

- **形狀：** 柔和圓角（8px），預設高 36px，內距 8px／16px。小尺寸 32px、大尺寸 40px、純圖示為 36px 正方。
- **Primary：** 訊號藍填色、白字、`--shadow-sm`。hover 亮到 Signal Blue Hover，按下暗到 Signal Blue Pressed。
- **Secondary：** 互動板岩填色、主要文字色、`--shadow-sm`。hover 時填色降到 80%。
- **Outline：** 透明底加髮絲線邊框；hover 填成互動板岩。
- **Ghost：** 無填色無邊框；hover 填成互動板岩。密集區域中純圖示動作的預設選擇。
- **Destructive：** 故障紅填色、白字。**只保留給不可逆的動作**——不用於任何僅僅是「要小心」的情況。
- **Disabled：** 所有變體一律 50% 不透明度並移除指標事件。**停用的控制項要留在畫面上**：一個存在但不能用的控制項必須說明原因，絕不可消失。

### Badges and Pills

- **樣式：** 4px 圓角、12px 標籤字、2px／8px 內距。
- **狀態變體**以語意 tint 為底、飽和色為字——執行綠、待命琥珀、故障紅（用可讀故障紅）、提示青。
- **中性變體：** `secondary` 用互動板岩；`outline` 是髮絲線邊框加次要文字色，**用於分類而非狀態**。

### Cards and Containers

- **圓角：** 12px。
- **底色：** 抬升板岩——**色階本身就是「它是一張卡片」的理由**。
- **陰影：** `--shadow-md`。
- **邊框：** 預設無。同時有色階又有邊框的卡片是過度指定。
- **內距：** 24px，header／content／footer 共用這個內縮，footer 去掉上內距。

### Inputs and Fields

- **樣式：** 抬升板岩填色、髮絲線邊框、8px 圓角、8px／12px 內距、14px 文字。placeholder 用 Read Dim。
- **Focus：** 邊框轉為訊號藍，外加同色 1px 環。全域 `:focus-visible` 是 2px 訊號藍、2px 外偏移——**鍵盤焦點永遠可見，絕不移除**。
- **Disabled：** 50% 不透明度、`not-allowed` 游標。

### Navigation

- **側軌項目：** 8px 圓角、14px medium 標籤配 16px 圖示，靜止為 Read Dim。
- **Hover：** 互動板岩填色，標籤提升到主要文字色。
- **Active：** 15% 訊號藍淡洗（`--accent-subtle`），標籤為主要文字色加 semibold——**絕不實心填滿**。側軌收合時 active 標籤色改為 Signal Blue Hover，因為沒有文字可以承載那個重量。
- **行動裝置：** 底部分頁列加 More sheet；全程最小 44px 觸控目標。

### Status Rows（signature）

活動中心與所有進度面背後的反覆模式：圖示晶片、標題、原因說明行、右側槽，其下為選用的進度條。右槽在**有真實計數時**顯示 `current / total`，在**有真實分數進度時**顯示百分比，兩者皆無時顯示純文字 **進行中**——**沒有可量測進度的工作完全不渲染進度條，而不是渲染一條空的**。這是「誠實的讀數」最字面的體現。

## Do's and Don'ts

### Do：

- **Do** 嚴格照狀態詞彙用色：綠＝正在發生、琥珀＝要求了但沒發生、不出現＝沒要求。
- **Do** 先伸手拿背景階（午夜 → 抬升板岩 → 互動板岩），再考慮陰影。
- **Do** 顏色**被讀**時用 `--accent-text` / `--error-text`，**被按或填色**時用 `--accent-primary` / `--error`。
- **Do** 內文維持 14px、標籤 12px；**密度就是設計**。
- **Do** 任何使用者可能跨列比較的數字都用等寬字。
- **Do** 側欄保持貼齊，寬度限制搬到內容格並靠左對齊。
- **Do** 不可用的控制項留在畫面上、標為停用，並說明原因。

### Don't：

- **Don't** 渲染系統根本沒在量測的百分比。靜態的 **進行中** 是誠實的；整段執行期間都顯示 `0%` 會被讀成卡住。
- **Don't** 把 `--text-disabled`（#6e7891, 3.55:1）用在任何使用者需要讀的東西上。它只給停用控制項，而且是全系統唯一有記錄的低於 AA 的值。
- **Don't** 用實心訊號藍填滿導覽項目。「目前所在」用 15% 淡洗。
- **Don't** 給已經有色階的填色面再加邊框。
- **Don't** 對「包含側欄的版型根層」設寬度限制再置中——那會讓側欄脫離，在兩個導覽之間開一個洞。
- **Don't** 隨手加上淺色主題。這裡每一個 token 都是為深色底調過的，包括 0.3–0.6 的陰影 alpha——那是一個獨立專案。
- **Don't** 引入 display 字體或第二種個性。一套人文無襯線加一套等寬，就是全部。
