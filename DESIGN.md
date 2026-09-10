---
name: Vido
description: Dark, dense NAS media manager whose readouts never flatter.
colors:
  bg-primary: '#0c1512'
  bg-secondary: '#132320'
  bg-tertiary: '#1b302b'
  border-subtle: '#274039'
  accent-primary: '#c9a24b'
  accent-hover: '#e0be72'
  accent-pressed: '#a8853c'
  accent-subtle: '#c9a24b26'
  accent-tint: '#c9a24b1f'
  accent-text: '#e0be72'
  success: '#6fbfa8'
  success-tint: '#6fbfa81f'
  success-text: '#8fd3be'
  error: '#c0392b'
  error-pressed: '#9c3a2b'
  error-tint: '#c0392b1f'
  error-text: '#e08a76'
  warning: '#d4763f'
  warning-pressed: '#d97706'
  warning-tint: '#d4763f1f'
  warning-text: '#e8b04b'
  info: '#1391b2'
  info-tint: '#1391b21f'
  info-text: '#5bc4dd'
  text-primary: '#eae4d6'
  text-secondary: '#a8b3ac'
  text-muted: '#8fa096'
  text-disabled: '#5e6e66'
  text-inverse: '#0c1512'
  text-on-accent: '#14161a'
  text-on-scrim: '#faf6ea'
  overlay-scrim: '#000000b3'
  focus-ring: '#c9a24b'
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
  chrome:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.6875rem'
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
  pill: '999px'
spacing:
  xs: '4px'
  sm: '8px'
  md: '12px'
  lg: '16px'
  xl: '24px'
  2xl: '32px'
components:
  button-primary:
    backgroundColor: '{colors.accent-primary}'
    textColor: '{colors.text-on-accent}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
    typography: '{typography.body}'
  button-primary-hover:
    backgroundColor: '{colors.accent-hover}'
  button-primary-active:
    backgroundColor: '{colors.accent-pressed}'
  button-secondary:
    backgroundColor: '{colors.bg-tertiary}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  button-outline:
    backgroundColor: 'transparent'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  button-ghost:
    backgroundColor: 'transparent'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  button-destructive:
    backgroundColor: '{colors.error}'
    textColor: '{colors.text-on-scrim}'
    rounded: '{rounded.md}'
    padding: '8px 16px'
    height: '36px'
  card:
    backgroundColor: '{colors.bg-secondary}'
    rounded: '{rounded.lg}'
    padding: '24px'
  input:
    backgroundColor: '{colors.bg-secondary}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 12px'
  badge-running:
    backgroundColor: '{colors.success-tint}'
    textColor: '{colors.success-text}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  badge-asked:
    backgroundColor: '{colors.warning-tint}'
    textColor: '{colors.warning-text}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  badge-fault:
    backgroundColor: '{colors.error-tint}'
    textColor: '{colors.error-text}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  badge-note:
    backgroundColor: '{colors.info-tint}'
    textColor: '{colors.info-text}'
    rounded: '{rounded.sm}'
    padding: '2px 8px'
    typography: '{typography.label}'
  nav-item-active:
    backgroundColor: '{colors.accent-subtle}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 10px'
---

# Design System: Vido

> 章節標題與 frontmatter 的 token 名稱保持英文——DESIGN.md 規格靠精確標題解析，token 名稱等同程式碼識別碼。其餘內文為繁體中文，與 PRODUCT.md 一致。

> **權威來源**：色彩與間距的真值在 `apps/web/src/styles.css`；對比度由 `apps/web/src/styles-contrast.spec.ts` 守門；動態由 `styles-motion.spec.ts` 守門。設計稿 `ux-design.pen` 的變數已於 2026-09-10 與本文件、與 styles.css 三方對齊，其顏色與圓角變數名與這裡的 token 名一字不差。

## Overview

**Creative North Star：「誠實的讀數」（The Honest Readout）**

Vido 會在使用者不在的時候，於別人的 NAS 上持續跑上好幾分鐘，而且花的是真的錢。這個介面上的每一塊，本質都是「別處正在發生的事」的讀數——而**會諂媚的讀數比沒有讀數更糟**。一個不知道進度卻顯示 90% 的儀表不是體貼，是說謊。整套視覺系統存在的理由，就是讓背景工作的真實狀態在昏暗房間裡一眼可讀，而使用者不需要「相信」任何東西。

這導致一個刻意不華麗的介面。文字小而密——全 app `text-sm` 出現 623 次、`text-xs` 279 次，對比 `text-4xl` 只有 6 次——因為它的任務是把一整座真實片庫的狀態塞進一個畫面，不是製造第一印象。深度幾乎全靠三階背景色而非陰影，所以眼睛讀得出結構，卻不會有任何東西感覺被墊高或裝飾過。強調色被配給到近乎稀有：金色從不填滿導覽項目，只以淡洗帶過。

這套系統唯一願意花成本的地方是**誠實**。狀態色是資料不是裝飾，而且語意固定：綠＝正在發生、琥珀＝你要求了但沒發生、不出現＝你沒要求。這組詞彙比任何程度的精緻都值錢，任何「視覺改善」都不准把它弄模糊。

**Key Characteristics：**

- **兩個主題，同一個世界**：夜行（Nightwalk，深色，預設）與日巡（Daywalk，淺色，opt-in）。兩者共用同一組 token 名稱與同一組語意，只有值不同
- 刻意密集：內文 14px、標籤 12px，這就是工作區間
- 深度來自色階，不是陰影
- 強調色配給制；語意色只保留給狀態，絕不用於強調
- 對比度是硬性關卡（WCAG AA 4.5:1），兩個主題都被同一支測試守著，全系統只有一個有記錄的例外

## Colors

武俠世界的顏料，不是 UI 色票：墨綠的地、宣紙的字、泥金的「你在這裡」、硃砂只留給壞掉與毀滅。每一個飽和色都有明確任務，沒有任何顏色是為了好看而存在。

**兩個主題的機制**：`:root` 就是夜行，淺色寫成 `[data-theme='light']`。**深色不寫任何屬性**——切到夜行是「移除 attribute」，不是寫 `data-theme="dark"`，否則預設值會有兩個互相矛盾的來源。

### Primary

- **泥金 Gold**（`--accent-primary`，夜行 `#c9a24b` ／日巡 `#886208`）：唯一的品牌色，而且用得很省。主要按鈕、焦點框、active 淡洗。它只在「使用者要按的控制項」上以實心出現——**絕不當作閱讀區的背景**。
- **Hover**（`--accent-hover`，`#e0be72` ／ `#725205`）／**Pressed**（`--accent-pressed`，`#a8853c` ／ `#5b4103`）：**這是唯一一個方向會隨主題翻轉的 token**。夜行 hover 變亮，日巡 hover 變**暗**——泥金在紙上必須往深處走才讀得出來。任何自己 `brightness()` 加亮的元件在日巡會走錯方向。
- **可讀泥金 Accent Text**（`--accent-text`，`#e0be72` ／ `#654804`）：當強調色需要**被閱讀**而不是被按的時候用。`--accent-primary` 當內文在自身 tint 疊底上，日巡只有 3.57:1，過不了 AA；這一階兩個主題最差 5.31:1，**與 `--accent-primary` 不可互換**。

### Secondary — 狀態詞彙

這四個不是色盤，是一組**語意固定的詞彙**，而且它們的意思是承重的。

- **青碧 Jade**（`--success`，`#6fbfa8` ／ `#0b7352`）：**現在正在發生**。絕不用於「做完了、已結束」，也絕不當作一般的肯定。
- **赭 Ochre**（`--warning`，`#d4763f` ／ `#a6510c`）：使用者要求了某件事，而它**沒有**在發生。這是整套系統裡最重要的顏色——它是唯一能夠說出「系統與你的意願不一致」而不把它藏起來的方式。
- **硃砂 Cinnabar**（`--error`，兩個主題都是 `#c0392b`）：壞掉了。**整組色盤裡唯一不隨主題翻轉的顏色。** 當文字讀時用 `--error-text`（`#e08a76` ／ `#87251b`）。
- **靛青 Indigo-teal**（`--info`，`#1391b2` ／ `#0b657d`）：說明性的，不帶評價。用於「告知但不暗示好壞」的通知。

每個都有一個 `*-tint` 夥伴作為徽章與藥丸的底色。**tint 的 alpha 會隨主題上升**（夜行 0x1f ≈ 12%，日巡 0x33 ≈ 20%），因為 12% 的顏料鋪在宣紙上幾乎看不見——那些 alpha 是對比度的輸入值，不是裝飾。**顏料本身兩個主題相同**：淺色主題若把泥金也調暗，一層薄洗會變成濁卡其。

### Neutral

- **地 Ground**（`--bg-primary` `#0c1512` ／ `#faf6ea`）：頁面本身。所有東西都坐在這上面。日巡永遠不用純白——臨床白會瞬間打破這個世界。
- **抬升面**（`--bg-secondary` `#132320` ／ `#f0e7d3`）：卡片、輸入框，任何「是一個獨立物件而非頁面本身」的東西。
- **互動面**（`--bg-tertiary` `#1b302b` ／ `#e5d9c3`）：hover 填色、次要按鈕、chip 底色。⚠️ **日巡的 `--bg-tertiary` 是最深的一階**，所以它是所有文字 token 的最壞情況，整組色盤由它綁定。
- **髮絲線**（`--border-subtle` `#274039` ／ `#cdbe9b`）：分隔線與物件外框。兩個主題都不受對比度守門——它是裝飾，不承載資訊。
- **三個閱讀權重**：`--text-primary`（`#eae4d6` ／ `#16231d`）、`--text-secondary`（`#a8b3ac` ／ `#32493e`）、`--text-muted`（`#8fa096` ／ `#41554c`）。
- **停用文字**（`--text-disabled` `#5e6e66` ／ `#7a8980`）：**有記錄的唯一例外**——刻意低於 AA，**只准用在停用的控制項上**（WCAG 1.4.3 對 inactive UI component 有豁免）。它不是第四個閱讀權重，絕不可承載說明文字。

### 對比度實測

以下是 `styles-contrast.spec.ts` 的量法：對 `--bg-primary` / `--bg-secondary` / `--bg-tertiary` 三個底色取最差值，`*-text` 另含自身 tint 疊底、`--text-primary` 與 `--accent-text` 另含 `--accent-subtle` 疊底，夜行與日巡再取更差的一邊。

| Token                     | 最差對比 | 判定                   |
| ------------------------- | -------- | ---------------------- |
| `--text-primary`          | 8.50:1   | AA ✓                   |
| `--text-secondary`        | 6.45:1   | AA ✓                   |
| `--text-muted`            | 5.08:1   | AA ✓                   |
| `--text-disabled`         | 2.59:1   | 刻意非 AA（TC-1 豁免） |
| `--accent-text`           | 5.31:1   | AA ✓                   |
| `--error-text`            | 5.02:1   | AA ✓                   |
| `--accent-primary` 當內文 | 3.57:1   | ✗ 改用 `--accent-text` |
| `--error` 當內文          | 2.42:1   | ✗ 改用 `--error-text`  |

### Named Rules

**固定詞彙規則（The Fixed Vocabulary Rule）** 綠＝正在發生。琥珀＝你要求了但沒發生。不出現＝你沒要求。狀態色不得被挪用為強調、裝飾或分類——它一旦出現，就是在對「狀態」做出主張，而那個主張必須為真。

**配給強調規則（The Rationed Accent Rule）** 泥金只填滿「要被按的控制項」。僅僅是「目前所在」的狀態——active 導覽項、選取的列——用 `--accent-subtle` 淡洗，絕不用實心填滿。

**兩種金規則（The Two Golds Rule）** `--accent-primary` 給人按，`--accent-text` 給人讀。換過來會無聲地破壞對比度。硃砂也有同一組分工（`--error` / `--error-text`）。

**主題平價規則（The Theme Parity Rule）** 任何新的顏色 token 必須**同時**寫進兩個區塊，並在 `styles-contrast.spec.ts` 補上淺色的對應案例。只加一半會讓 parity 斷言直接失敗——這是刻意的。

## Typography

**內文字體：** Noto Sans TC（後備 `-apple-system`、`BlinkMacSystemFont`、`Segoe UI`、`Roboto`、sans-serif）
**讀數字體：** JetBrains Mono（後備 Consolas、Monaco、monospace）

**個性：** 一套人文主義無襯線扛下所有閱讀工作，加一套等寬字專門留給「必須對齊或互相比較的數字」。**沒有 display 字體、沒有編輯式的字體配對**——字體系統的任務是在昏暗房間裡小尺寸也讀得清楚，第二種個性只會礙事。

CJK 一律走 Noto Sans TC。設計稿上的 DM Sans 只用於**畫布註記**（流程標題、色票標籤那些不會被實作的東西），不是產品字體。

### Hierarchy

- **Display**（700, 36px, 1.1）：罕見。詳情頁 hero 標題與海報 placeholder 的首字——程式碼 6 次。**不是一般的標題層級。**
- **Headline**（700, 24px, 1.25）：頁面標題（`設定`、`活動`）。一個畫面一個。
- **Title**（600, 18px, 1.4）：區段與卡片標題。
- **Body**（400, 14px, 1.6）：預設值，而且是全 app 用量遙遙領先的尺寸。說明、列表列、表單標籤，幾乎所有東西。
- **Label**（500, 12px, 1.4）：徽章、藥丸、metadata、表頭、次要註記。
- **Chrome**（500, 11px, 1.4）：**只給殼層本身的標籤** —— 底部分頁列、側軌群組標題、側軌計數、環境狀態帶。**它的界線很硬：11px 只用在「不是內容的東西」上。任何使用者要閱讀的字都不得低於 12px，任何可以被點擊的目的地或動作都是 14px。**
- **Readout**（400, 13px, 等寬）：計數、進度數字、位元組大小、ID、檔案路徑——任何使用者可能跨列比較、或當成資料讀的東西。

### 設計稿與程式碼的字級對照

設計稿 `ux-design.pen` 的字級變數以角色命名，與這裡的層級對應如下。**兩邊目前尚未完全重疊**，差異列在最後一列，是已知待收斂項而不是筆誤。

| 設計稿變數            | px      | 程式碼對應              | 用途                                       |
| --------------------- | ------- | ----------------------- | ------------------------------------------ |
| `Type/Display/L/Size` | 36      | `text-4xl`              | Display                                    |
| `Type/Display/M/Size` | 32      | 無                      | 設計稿專屬（spec 頁大標）                  |
| `Type/Title/XL/Size`  | 28      | 無                      | 設計稿專屬（頁面大標）                     |
| `Type/Title/L/Size`   | 24      | `text-2xl`              | Headline                                   |
| `Type/Title/M/Size`   | 22      | 無                      | 設計稿專屬（手機頁標）                     |
| `Type/Title/S/Size`   | 20      | `text-xl`               | 區塊標題                                   |
| `Type/Heading/L/Size` | 18      | `text-lg`               | Title                                      |
| `Type/Heading/M/Size` | 16      | `text-base`             | 次級標題                                   |
| `Type/Heading/S/Size` | 15      | `text-[15px]`           | 卡片標題                                   |
| `Type/Body/L/Size`    | 14      | `text-sm`               | Body                                       |
| `Type/Body/M/Size`    | 13      | `text-[13px]`           | Readout／次要內文                          |
| `Type/Body/S/Size`    | 12      | `text-xs`               | Label                                      |
| `Type/Meta/M/Size`    | 11      | `text-[11px]`           | Chrome                                     |
| —                     | 30 / 48 | `text-3xl` / `text-5xl` | **程式碼有、設計稿沒有**（各 5 次 / 1 次） |
| —                     | 10      | `text-[10px]`           | **已廢除**，見下                           |

### Named Rules

**預設小字規則（The Small-By-Default Rule）** 14px 是內文、12px 是標籤。伸手拿更大的尺寸，等於主張「這段文字比頁面的實際內容更重要」——而它通常不是。實測 623 次 `text-sm` 對 6 次 `text-4xl` 是**預期的形狀，不是待修正的意外**。

**比較才用等寬規則（The Mono-For-Comparison Rule）** 如果同一個值的兩個實例可能被沿著欄位上下對照，就用等寬。散文永遠不用等寬；使用者只掃一眼、不會拿去比的數字也不需要。

**11px 是地板（The 11px Floor Rule）** 10px 這一階**已於 2026-09-10 廢除**（⚖️ Alexyu 裁定，PR #410）。它從來沒有授權來源：2026-09-01 的 critique 就裁掉過 `InFlightBadge` 的 10px，但那一階仍靠著沒人管的一次性寫法活了下來，設計稿裡累積到 49 個節點、程式碼裡 6 處。廢除的理由是產品自己的北極星——**讀數看不清等於沒有讀數**，而那 6 處裡有兩處是錯誤訊息。設計稿的 49 個節點已全數改為 11px；程式碼那 6 處另案處理（會動到視覺基準）。

新的地板是兩層：**11px 只給殼層 chrome（不是內容）；使用者要閱讀的任何文字都不得低於 12px。** 設計稿現在沒有任何低於 11px 的字級變數可用，這是刻意的——要用更小的字，得先改這條規則。

## Layout

App 是「固定左側軌 ＋ 流動內容欄」。左側軌展開 240px、收合 64px，狀態存在 `localStorage`；在 `sm` 斷點以下改為底部分頁列加 More sheet。

內容頁以 `max-w-7xl`（1280px）容器搭配 `mx-auto` 置中，這是**頁面內容**的規則。**它不是「本身就有側欄的頁面」的規則。** 把「包含側欄的整塊」加上寬度限制再置中，會讓那個側欄脫離 App 側軌，在兩個導覽之間留下一道死掉的垂直空白；設定區在 1920px 下實地踩過這個坑——子導覽整整離開它該貼著的側軌 200px。**巢狀側欄的版型，根層不設限也不置中，寬度限制搬到內容格並且靠左對齊。**

間距走 Tailwind 的 4px 基準：4 / 8 / 12 / 16 / 24 / 32。卡片內距 24px，輸入框 8px 垂直、12px 水平，列表列垂直 8–12px。行動裝置觸控目標維持最小 44px。

設計稿的間距階梯比這條長一截，因為它同時要描述已經畫出來的版面：2 / 4 / 6 / 8 / 10 / 12 / 14 / 16 / 20 / 24 / 28 / 32 / 40 / 48 / 64 / 80（`Space/*`）。半階（6、10、14、28）對應 Tailwind 的 `.5` 級距，不是隨手畫出來的數字。

### Named Rules

**貼齊側欄規則（The Flush Sidebar Rule）** 側欄要貼齊它旁邊的東西。只有內容拿閱讀寬度限制，而且靠左對齊，讓每一欄共用同一條左邊線——多出來的寬度收在右邊，讀起來是頁面留白，不是一個洞。

## Elevation & Depth

**這套系統近乎扁平，用色調分層而非陰影。** 三階背景幾乎包辦所有結構工作。**一張卡片之所以是卡片，不是因為它有陰影，而是因為它比頁面亮一階**（日巡則是暗一階）。

陰影確實存在也有在用，但只用在「真的浮在頁面之上」而非「坐在頁面上」的東西。**兩個主題的陰影是兩種材質**：夜行用煤黑（alpha 0.3–0.6），因為深色底上柔和的陰影等於看不見；日巡用墨（`--text-primary` 的低 alpha，總墨量 ≤15.4%），因為黑色 30–60% 打在暖紙上會讀成一塊瘀青，而且從 md 起改用兩層——紙上的抬升是一道銳利近邊加一片柔投影。

### Shadow Vocabulary

- **`--shadow-sm`**：有填色、可按的控制項——主要／次要／破壞性按鈕。剛好夠讀成一個實體控制項。
- **`--shadow-md`**：`Card` 基本元件。
- **`--shadow-lg`**：罕見的中間階。
- **`--shadow-xl`**：**只給真正的覆蓋層**——Dialog、Sheet、詳情頁 hero。這個上限是刻意的。

### Named Rules

**色調優先規則（The Tone-First Rule）** 伸手拿陰影之前先伸手拿背景階。如果一個面需要讀起來與頁面分離，它就是 `--bg-secondary`。只有當這個面真的浮在它所覆蓋的內容之上時，才加陰影。

## Motion

**動的東西 ＝ 正在發生的事。靜的東西 ＝ 已經定案的事實。**

這不是風格偏好，是色彩「固定詞彙」在時間軸上的同一條規則。綠色不能宣稱一件沒在發生的事；同理，**會動的元件就是在宣稱「現在有工作在跑」**。一個畫面不可以把一個真相穿兩種顏色，也不可以讓一件靜止的事實自己動起來。

因此**只有三張動的許可證**，其餘一律是動畫債，該刪不該調：

| 許可證       | 條件                                       | Token                                                                                                                                    |
| ------------ | ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| ① 回應手勢   | 使用者碰了它                               | `--motion-touch` 120ms                                                                                                                   |
| ② 解釋改變   | 某個值／位置真的變了                       | `--motion-state` 200ms（值）· `--motion-move` 320ms（位移、覆蓋層）· `--motion-arrive` 640ms ／ `--motion-leave` 240ms（成對的交叉淡入） |
| ③ 它真的在跑 | 後端說現在有工作在進行，**而且只在進行時** | `--breath` 2400ms                                                                                                                        |

③ 是全 app 唯一可以「沒人碰它也會動」的東西，目前只有兩個持有者：殼層的活動徽章（`InFlightBadge`）與首頁讀數帶的「進行中」格。兩者都綁在真實的 in-flight 數字上，歸零就立刻靜止。**不要把這個樣式套到靜態數字上**——那等於在第二個頻道上說謊，跟給一個沒跑過的任務掛綠色徽章是同一種錯。

### 曲線：輕功

`--ease-settle`（`cubic-bezier(0.16, 1, 0.3, 1)`）快速離地、緩緩落定、**沒有回彈**。`--ease-leave`（`cubic-bezier(0.4, 0, 0.9, 0.4)`）直接走。彈簧與 elastic 曲線會衝過 1，等於把重量放回一個以輕盈為前提的世界——`styles-motion.spec.ts` 會擋下來。

**離場永遠比入場快。** 入場是資訊，離場只是讓路。

### 兩條軸：時間與距離

`prefers-reduced-motion` 把**時間**壓到 1ms、把**距離**歸零，這兩件事必須分開做。只壓時間不會移除一個 hover 抬升，只會把它變成瞬間**跳位**——所以 `--motion-lift` / `--motion-rise` / `--motion-turn` 是獨立的 token，在 reduced motion 下變成 `1` / `0px` / `0deg`。顏色與狀態轉場則刻意保留：**要求少一點動態的人，仍然需要看見自己按到了。**

### Named Rules

**一個手勢一個秒數（One Gesture, One Duration）** 一次 hover 底下的所有圖層——縮放、徽章退場、kebab 出現、漸層——共用同一個 token。各部件在不同時間到位的卡片，會讀成四張卡片。

**秒數不寫死（No Literal Durations）** `duration-300` 對 `prefers-reduced-motion` 的 token 覆寫是隱形的，只有 `styles.css` 那張粗網能兜住。`local/no-hardcoded-duration` 會擋；真的是「期限」而非動態（例如自動關閉倒數）就寫 eslint-disable 並說明理由。

**JS 的平滑捲動 CSS 管不到** `scrollBy({ behavior: 'smooth' })` 的顯式參數會蓋過 `scroll-behavior: auto`。一律走 `lib/motion.ts` 的 `scrollByMotionSafe()`。

## Shapes

> ⚖️ **2026-08-26 修正（Alexyu 裁定，critique homepage R2）**：「絕不做成藥丸形」新增一條例外——**壓在海報/圖片上的微型覆蓋元件**（狀態小徽章、評分藥丸、輪播圓點、圖上圓形圖示鈕這類高度 ≤ ~22px 的讀數級元素）**得用藥丸形**：在任意畫面上，全圓角讓微小元件讀成一枚「印」而非一塊「面」。**可按的按鈕與頁面層的 chip 不在例外內**，維持 8px（`--radius-md`）。

柔和圓角，絕不做成藥丸形，也絕不銳利。四個圓角階存在，但一個獨大：**8px（`--radius-md`）涵蓋按鈕、輸入框、導覽項與多數互動面**。12px（`--radius-lg`）屬於卡片與較大的容器。4px（`--radius-sm`）給小色標——徽章、chip、標籤。16px（`--radius-xl`）幾乎未使用。上面那條例外用的全圓角在設計稿裡是 `radius-pill`（999px）。

邊框是 1px 髮絲線，**只出現在色階本身不足以分開兩樣東西的地方**——外框按鈕、靜止狀態的輸入框、表格與列表分隔線。有填色的面不會同時再給邊框。

### Named Rules

**單一圓角規則（The One Radius Rule）** 沒有理由就用 8px。卡片用 12px 是因為它夠大、8px 會讀起來偏銳；徽章用 4px 是因為它夠小、8px 會讀起來偏圓。其他一律 8px。

## Components

### Buttons

- **形狀：** 柔和圓角（8px），預設高 36px，內距 8px／16px。小尺寸 32px、大尺寸 40px、純圖示為 36px 正方。
- **Primary：** 泥金填色、`--text-on-accent` 深墨字、`--shadow-sm`。hover 到 `--accent-hover`、按下到 `--accent-pressed`。**注意這兩個狀態在日巡是往暗走**。
- **Secondary：** `--bg-tertiary` 填色、主要文字色、`--shadow-sm`。
- **Outline：** 透明底加髮絲線邊框；hover 填成 `--bg-tertiary`。
- **Ghost：** 無填色無邊框；hover 填成 `--bg-tertiary`。密集區域中純圖示動作的預設選擇。
- **Destructive：** 硃砂填色配 `--text-on-scrim`（**不是** `--text-on-accent`——硃砂不隨主題翻轉，它的標籤也不翻）。**只保留給不可逆的動作**。
- **Disabled：** 所有變體一律 50% 不透明度並移除指標事件。**停用的控制項要留在畫面上**：一個存在但不能用的控制項必須說明原因，絕不可消失。

### Badges and Pills

- **樣式：** 4px 圓角、12px 標籤字、2px／8px 內距。
- **狀態變體**以語意 tint 為底、`*-text` 階為字——青碧、赭、硃砂、靛青。**底用 tint、字用 `*-text`，不要用飽和色當字。**
- **中性變體：** `secondary` 用 `--bg-tertiary`；`outline` 是髮絲線邊框加次要文字色，**用於分類而非狀態**。

### Cards and Containers

- **圓角：** 12px。
- **底色：** `--bg-secondary`——**色階本身就是「它是一張卡片」的理由**。
- **陰影：** `--shadow-md`。
- **邊框：** 預設無。同時有色階又有邊框的卡片是過度指定。
- **內距：** 24px，header／content／footer 共用這個內縮，footer 去掉上內距。

### Inputs and Fields

- **樣式：** `--bg-secondary` 填色、髮絲線邊框、8px 圓角、8px／12px 內距、14px 文字。placeholder 用 `--text-muted`。
- **Focus：** `--focus-ring`（＝ `--accent-primary`）。全域 `:focus-visible` 是 2px 加 2px 外偏移——**鍵盤焦點永遠可見，絕不移除**。
- **Disabled：** 50% 不透明度、`not-allowed` 游標。

### Navigation

- **側軌項目：** 8px 圓角、14px medium 標籤配 16px 圖示，靜止為 `--text-muted`。
- **Hover：** `--bg-tertiary` 填色，標籤提升到主要文字色。
- **Active：** `--accent-subtle` 淡洗，標籤為主要文字色加 semibold——**絕不實心填滿**。側軌收合時 active 標籤色改為 `--accent-hover`，因為沒有文字可以承載那個重量。
- **行動裝置：** 底部分頁列加 More sheet；全程最小 44px 觸控目標。

### Status Rows（signature）

活動中心與所有進度面背後的反覆模式：圖示晶片、標題、原因說明行、右側槽，其下為選用的進度條。右槽在**有真實計數時**顯示 `current / total`，在**有真實分數進度時**顯示百分比，兩者皆無時顯示純文字 **進行中**——**沒有可量測進度的工作完全不渲染進度條，而不是渲染一條空的**。這是「誠實的讀數」最字面的體現。

### 設計稿的元件清冊

`ux-design.pen` ▸ `Design System · 設計系統` ▸ `Components · 元件` 底下有 35 個母版，分五類。母版住在分類裡，Component Library 那頁放的是 instance。

| 分類                | 元件                                                                                                                                                                              |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 01 · 導覽與外框     | SidebarNavItem、SidebarGroupParent、SidebarGroupLabel、SidebarFooterStatus、MobileTabItem、HomeSidebar-v2                                                                         |
| 02 · 基礎控件       | ButtonPrimary、ButtonSecondary、SearchInput、SortDropdown、FilterChip、GenreTag、Checkbox（含 Empty／Indeterminate／DisabledChecked／DisabledEmpty 四態）、TabActive、TabInactive |
| 03 · 媒體卡片與標籤 | PosterCard、PosterCardHover、PosterCard-v2、TechBadge-Video／Audio／Subtitle／HDR                                                                                                 |
| 04 · 列表與進度     | ActivityRow-v2、RequestRow-v2、DownloadCard-v2、GenerationProgress-v2、GlossaryRow-v2、GenQueueRow-v2                                                                             |
| 05 · 空狀態         | EmptyLibrary-NoQBT、EmptyLibrary-NoFolder、EmptyLibrary-ReadyForScan                                                                                                              |

TechBadge 的四個分類對應程式碼 `TechBadge.tsx` 的語意：video → accent、audio → info、hdr → warning、subtitle → success。

## Do's and Don'ts

### Do：

- **Do** 嚴格照狀態詞彙用色：綠＝正在發生、琥珀＝要求了但沒發生、不出現＝沒要求。
- **Do** 先伸手拿背景階（`--bg-primary` → `--bg-secondary` → `--bg-tertiary`），再考慮陰影。
- **Do** 顏色**被讀**時用 `--accent-text` / `--error-text`，**被按或填色**時用 `--accent-primary` / `--error`。
- **Do** 新增顏色 token 時兩個主題一起寫，並補上 `styles-contrast.spec.ts` 的淺色案例。
- **Do** 內文維持 14px、標籤 12px；**密度就是設計**。
- **Do** 任何使用者可能跨列比較的數字都用等寬字。
- **Do** 側欄保持貼齊，寬度限制搬到內容格並靠左對齊。
- **Do** 不可用的控制項留在畫面上、標為停用，並說明原因。

### Don't：

- **Don't** 渲染系統根本沒在量測的百分比。靜態的 **進行中** 是誠實的；整段執行期間都顯示 `0%` 會被讀成卡住。
- **Don't** 把 `--text-disabled`（最差 2.59:1）用在任何使用者需要讀的東西上。它只給停用控制項，而且是全系統唯一有記錄的低於 AA 的值。
- **Don't** 用實心泥金填滿導覽項目。「目前所在」用 `--accent-subtle` 淡洗。
- **Don't** 給已經有色階的填色面再加邊框。
- **Don't** 對「包含側欄的版型根層」設寬度限制再置中——那會讓側欄脫離，在兩個導覽之間開一個洞。
- **Don't** 用 `filter: brightness()` 或 `hover:brightness-110` 做 hover 加亮。日巡的 `--accent-hover` 是往**暗**走的，加亮會走錯方向；一律吃 token。
- **Don't** 假設 `--text-inverse` 等於「深色的地」。它在日巡是紙色——任何把它當「暗底」用的呼叫點在淺色主題會無聲翻錯。
- **Don't** 引入 display 字體或第二種個性。一套人文無襯線加一套等寬，就是全部。
