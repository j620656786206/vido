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
  error-pressed: '#9c3a2b'
  success: '#6fbfa8'
  error: '#c0392b'
  warning: '#d4763f'
  info: '#1391b2'
  text-primary: '#eae4d6'
  text-secondary: '#a8b3ac'
  text-muted: '#8fa096'
  text-inverse: '#0c1512'
  accent-subtle: '#c9a24b26'
  accent-tint: '#c9a24b1f'
  accent-text: '#e0be72'
  success-tint: '#6fbfa81f'
  success-text: '#8fd3be'
  error-tint: '#c0392b1f'
  error-text: '#e08a76'
  warning-tint: '#d4763f1f'
  warning-text: '#ff8d29'
  warning-pressed: '#c26a36'
  info-tint: '#1391b21f'
  info-text: '#5bc4dd'
  text-on-accent: '#14161a'
  text-on-scrim: '#faf6ea'
  text-disabled: '#5e6e66'
  overlay-scrim: '#000000b3'
  focus-ring: '#c9a24b'
typography:
  display:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '2.25rem'
    fontWeight: 700
    lineHeight: 1.25
  h1:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.875rem'
    fontWeight: 700
    lineHeight: 1.25
  h2:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.5rem'
    fontWeight: 700
    lineHeight: 1.375
  h3:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.25rem'
    fontWeight: 600
    lineHeight: 1.375
  h4:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1.125rem'
    fontWeight: 600
    lineHeight: 1.5
  body-lg:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '1rem'
    fontWeight: 400
    lineHeight: 1.625
  body:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.875rem'
    fontWeight: 400
    lineHeight: 1.625
  label:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.75rem'
    fontWeight: 500
    lineHeight: 1.5
  button-solid:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.875rem'
    fontWeight: 600
    lineHeight: 1.625
  button-hollow:
    fontFamily: 'Noto Sans TC, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif'
    fontSize: '0.875rem'
    fontWeight: 500
    lineHeight: 1.625
  readout:
    fontFamily: 'JetBrains Mono, Consolas, Monaco, monospace'
    fontSize: '0.875rem'
    fontWeight: 400
    lineHeight: 1.625
  readout-column:
    fontFamily: 'JetBrains Mono, Consolas, Monaco, monospace'
    fontSize: '0.75rem'
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sm: '4px'
  md: '8px'
  lg: '12px'
  xl: '16px'
  pill: '999px'
spacing:
  none: '0px'
  2xs: '2px'
  xs: '4px'
  xs-plus: '6px'
  sm: '8px'
  sm-plus: '10px'
  md: '12px'
  md-plus: '14px'
  lg: '16px'
  lg-plus: '20px'
  xl: '24px'
  xl-plus: '28px'
  2xl: '32px'
  3xl: '40px'
  4xl: '48px'
  5xl: '64px'
  6xl: '80px'
components:
  button-primary:
    backgroundColor: '{colors.accent-primary}'
    textColor: '{colors.text-on-accent}'
    rounded: '{rounded.md}'
    padding: '8px 20px'
    height: '36px'
    typography: '{typography.button-solid}'
  button-primary-hover:
    backgroundColor: '{colors.accent-hover}'
  button-primary-active:
    backgroundColor: '{colors.accent-pressed}'
  button-secondary:
    backgroundColor: '{colors.bg-tertiary}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 20px'
    height: '36px'
    typography: '{typography.button-hollow}'
  button-outline:
    backgroundColor: 'transparent'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 20px'
    height: '36px'
    typography: '{typography.button-hollow}'
  button-ghost:
    backgroundColor: 'transparent'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '8px 20px'
    height: '36px'
    typography: '{typography.button-hollow}'
  button-destructive:
    backgroundColor: '{colors.error}'
    textColor: '{colors.text-on-scrim}'
    rounded: '{rounded.md}'
    padding: '8px 20px'
    height: '36px'
    typography: '{typography.button-solid}'
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

> **權威來源**：色彩與間距的真值在 `apps/web/src/styles.css`；對比度由 `apps/web/src/styles-contrast.spec.ts` 守門；動態由 `styles-motion.spec.ts` 守門。設計稿 `ux-design.pen` 的顏色與圓角變數名與這裡的 token 名一字不差。
>
> ⚖️ **2026-09-10：設計稿正式進入守門範圍。** 在那之前這裡寫的是「已於某日三方對齊」——那是一次性的人工斷言，沒有任何機制維持它為真，而設計稿正是漂移了半年沒人發現的那一份。現在 `export-pen-screenshots.py` 每次都會把設計稿的變數 dump 成 `_bmad-output/pen-tokens.json`，`check-design-tokens.py` 比對那份快照（CI 讀不到 `.pen`，它是加密的、只有跑著的 Pencil.app 讀得到）。快照裡帶著 `.pen` 的 sha256，改了設計稿卻沒重跑匯出腳本，雜湊對不上，CI 直接紅。
>
> 守門範圍：**顏色**（夜行＋日巡，對 styles.css）、**圓角**（對 styles.css）、**間距階梯**（對本文件 frontmatter）、**字級與行高**（對本文件 frontmatter）、**裸數字**（gap／padding／fontSize／lineHeight 一個都不准不吃變數）。**上線第一次就抓到一個真的漂移**：設計稿的 `warning-pressed` 還是舊的 `#d97706`，比靜止的赭色更亮——正是 P0-4 裁定要修掉的那個「按下去變亮」的 bug，styles.css 與另外兩份文件早就改成 `#c26a36` 了，只有設計稿沒跟上。
>
> **漂移檢查**：同一組色彩 token 寫在四個地方（`styles.css`、本文件的 frontmatter、`_bmad-output/design-context-pack.md`、`.impeccable/design.json`）。`scripts/check-design-tokens.py` 會比對四份、不一致就讓 CI 紅（`Lint & Format Check` 的 `Design token drift check`）。**它刻意不自動覆寫**——不一致時要先判斷哪一邊才是對的設計，可能是文件過期，也可能是實作值該改。確定 `styles.css` 是想要的設計時，`--apply` 只會改本文件。

## Overview

**Creative North Star：「誠實的讀數」（The Honest Readout）**

Vido 會在使用者不在的時候，於別人的 NAS 上持續跑上好幾分鐘，而且花的是真的錢。這個介面上的每一塊，本質都是「別處正在發生的事」的讀數——而**會諂媚的讀數比沒有讀數更糟**。一個不知道進度卻顯示 90% 的儀表不是體貼，是說謊。整套視覺系統存在的理由，就是讓背景工作的真實狀態在昏暗房間裡一眼可讀，而使用者不需要「相信」任何東西。

這導致一個刻意不華麗的介面。文字小而密——全 app `text-sm` 出現 623 次、`text-xs` 279 次，對比 `text-4xl` 只有 6 次——因為它的任務是把一整座真實片庫的狀態塞進一個畫面，不是製造第一印象。深度幾乎全靠三階背景色而非陰影，所以眼睛讀得出結構，卻不會有任何東西感覺被墊高或裝飾過。強調色被配給到近乎稀有：金色從不填滿導覽項目，只以淡洗帶過。

這套系統唯一願意花成本的地方是**誠實**。狀態色是資料不是裝飾，而且語意固定：金＝正在跑、青碧＝有答案了、赭＝你要求了但沒發生、硃砂＝壞了、不出現＝你沒要求。這組詞彙比任何程度的精緻都值錢，任何「視覺改善」都不准把它弄模糊。

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

- **青碧 Jade**（`--success`，`#6fbfa8` ／ `#0b7352`）：**有答案了，而且是好的那個答案**——做完了、成功了、已入庫。它是一組**終局狀態**的一半，另一半是硃砂（壞掉了）。絕不用於「正在跑」（那是泥金），也絕不當作一般的肯定。
- **赭 Ochre**（`--warning`，`#d4763f` ／ `#a6510c`）：使用者要求了某件事，而它**沒有**在發生。這是整套系統裡最重要的顏色——它是唯一能夠說出「系統與你的意願不一致」而不把它藏起來的方式。

  ⚖️ **2026-09-11 裁定：赭色說的是「現在的世界」，不是「你按下去會怎樣」。** 事前警語（「這會覆蓋你的資料」「這會刪掉檔案」）**不給語意色**——用 `--bg-tertiary` 底、`--text-primary` 字、保留警告圖示，份量押在**動作按鈕**上（見 §Buttons 的 Destructive：硃砂實心，只保留給不可逆的動作）。
  這條線判得動的例子：金鑰頁「目前連線未加密（HTTP）」**合法**（連線現在真的沒加密，你要的安全儲存沒有發生）；備份頁「還原會覆蓋你的資料」**不合法**（還沒發生，畫面上沒有任何狀態可以被它描述）；設計稿上寫給設計師看的後設註記框**不合法**（那不是系統狀態）。
  理由不只是分類學。實測夜行下 `--warning-text` 與 `--error-text` 的 ΔE00 只有 **17.34**，是五個語意色十組兩兩比較裡**最小的一對**；它們的 tint 底色 ΔE00 更只有 **4.8**（對比 1.066:1），也就是**底色完全不承載這個區別**。新增第六格詞彙在色相上也無處可放——0–42° 已被硃砂、赭、泥金塞滿，剩下的空隙只有黃綠（讀作成功）與紫（讀作 AI）。**赭色的價值來自它零誤報，不是來自它涵蓋得廣。**

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

> ⚠️ **語意基色不可以當文字。** `--success` / `--warning` / `--error` / `--info` / `--accent-primary` 是**填色**用的；當文字時最差對比分別是 4.19（日巡）、3.96（日巡，夜行 4.30 也不過）、2.57、3.80、3.96——**五個裡有四個至少在一個主題不過 AA**。文字一律用 `-text` 那一階。2026-09-10 修正了設計稿裡 379 個違反這條的文字節點（accent-primary 138、success 128、warning 72、info 26、error 15）。

### Named Rules

**固定詞彙規則（The Fixed Vocabulary Rule）**

| 顏料   | 意思                                         | 用哪個 token 當文字 |
| ------ | -------------------------------------------- | ------------------- |
| 泥金   | **正在跑**（真的有工作在進行）               | `--accent-text`     |
| 青碧   | **有答案了，好的那個**（完成、成功、已入庫） | `--success-text`    |
| 赭     | **你要求了，但它沒發生**                     | `--warning-text`    |
| 硃砂   | **壞了**                                     | `--error-text`      |
| 靛青   | **純告知，不帶評價**                         | `--info-text`       |
| 不出現 | 你沒要求                                     | —                   |

狀態色不得被挪用為強調、裝飾或分類——它一旦出現，就是在對「狀態」做出主張，而那個主張必須為真。

> ⚖️ **2026-09-10 修正（Alexyu 裁定）**：本文件從 2026-06 起寫的是「綠＝正在發生」，但程式碼裡有一條更早的相反裁定——`EpisodeList.tsx:76`「`--success` = done, `--accent-text` = in progress（accent stays reserved for in-progress — Sally 2026-07-05）」。設計稿跟著程式碼走，128 個「完成」用青碧。兩份正典互相矛盾了三個月沒人發現。裁定採用程式碼那套：**泥金＝正在跑、青碧＝有答案了**。理由不是改得少，是語意更好——正在跑的東西是「你在這裡、正在花你的資源」，那正是泥金的角色；青碧留給「有答案了」，與硃砂形成一對終局狀態。

**配給強調規則（The Rationed Accent Rule）** 泥金只填滿「要被按的控制項」。僅僅是「目前所在」的狀態——active 導覽項、選取的列——用 `--accent-subtle` 淡洗，絕不用實心填滿。

**兩種金規則（The Two Golds Rule）** `--accent-primary` 給人按，`--accent-text` 給人讀。換過來會無聲地破壞對比度。硃砂也有同一組分工（`--error` / `--error-text`）。

**金錢是事實，不是狀態（The Money-Is-A-Fact Rule）** 金額不穿狀態色。`$25.80` 不會「發生」也不會「失敗」——它是一個事實，不是一個主張。給它顏色等於把它拉進狀態詞彙，而每多一個詞，每個詞的辨識度就下降一點。金額一律 `--text-primary`，次要位置可用 `--text-secondary` / `--text-muted`；識別靠 `$` 符號本身，那是最強的識別符號。

**一格同時要說狀態與金額時，狀態押在標籤與圖示上，數字保持中性。** 首頁讀數帶的「需要注意」格就是這樣：標籤文字與三角圖示是赭色，底下的「2 部失敗 · $1.2/$5」是中性。⚖️ 2026-09-10 修正前是反過來的——標籤是 `--text-muted`、顏色全押在數字上，於是同一串文字裡前半是「壞掉了」、後半是「花了多少錢」，中間只有一個間隔點。設計稿 46 個穿狀態色的金額已全部轉中性。

**會花錢的動作要有記號（The Cost-Bearing Action Rule）** 一個會產生實際費用的動作，必須在控制項上帶著**預估金額**，而不是靠顏色。金額開頭的 `$` 就是那個固定記號——**不另加硬幣圖示**（⚖️ 2026-09-10 Alexyu 裁定：A 案，圖示多餘）。免費的動作不帶金額也不帶記號：**沒有數字就是免費的訊號**。

**沒有金額，就沒有可按的按鈕。** 記號之所以「固定」，是因為它和動作的**可用性**綁在一起：只要按鈕可以按，金額（含 `$`）就一定在場。算不出金額時**停用按鈕**，把原因寫在說明列，而不是讓它退回成一顆看不出要錢的按鈕。這不是視覺潔癖——2026-08-07 的事故裁定寫的是「付費生成必須在**先顯示金額的畫面**上明示選擇」（那次一按掃描 enqueue 1,026 項、約 2/3 走付費語音辨識、估算 ~US$200，使用者全程沒看到數字）；沒有金額，就沒有那個畫面。停用的是**控制項**不是**說明**：按鈕降 `--text-disabled`，說明列維持 `--text-secondary`。

**因此不存在「記號平常不出現、異常時才出現」的變體。** 那會讓同一個符號同時說「要花錢」與「我們不知道多少」，而這套系統的成本全花在「一個符號只說一件事」上（見技術標籤中性化裁定）。它還會反向破壞免費訊號：金額一旦可以缺席，「沒有數字」就同時意味著免費和未知。

**估不出金額是實作缺口，不是要設計的狀態。** 估價是 `時長 × 每分鐘費率` 的純函式，費率是編譯進二進位檔的常數，時長有三階梯（ffprobe 實測 → TMDb runtime → 45 分鐘假設），後端從型別上就無法回傳「未知」。每一個「不知道」都要降級成**一個講出來的假設**，不是空值：片長未知就用 45 分鐘估、`≈` 加在金額前、說明列寫「片長未知（估 45 分）」；路線或模型未知就報**貴的那一邊**，絕不低估。畫面上唯一合法的無金額狀態只有兩個：**估價請求還在路上**（金額位置放固定寬度骨架、暫時不可點）與**估價請求失敗**（停用＋原因）。

**金額的寫法。** 同意面（按鈕、候選清單、確認框）一律兩位小數，走 `usd()` 這一個 formatter：`$0.18`、`$0.00`、`$4.50`。`≈` 只有一個意思——「這個數字建立在一個假設的片長上」——**一列最多一個**（⚖️ Sally 2026-09-05）。`$0.00` 照樣寫成 `$0.00`，**不准寫「免費」**（⚖️ §5-sexies 2026-08-11）：0 是讀數，不是省略；為什麼是 0 交給說明列（「語音辨識：自架（不另計費）」）。首頁讀數帶那套精度折疊（`$1.2` / `$12` / `$1.2k`，見 H8 spec 與 `utils/formatUsdShort.ts`）**只適用於已經花掉的錢的密集讀數**，不得套用到任何同意面——在使用者按下去的那一刻把 `$0.18` 收成 `$0.2`，是在承諾的瞬間動了數字。

**按鈕上不畫金額範圍。** 系統不產生範圍（每列每模型都是單一數字，`min_usd`/`max_usd` 前後端皆不存在）；不確定性由 `≈` 承載，模型差價由確認框的模型選擇器承載。

六個狀態的定稿文案與可點性見設計稿 `J9-D · 會花錢的按鈕（金額即記號）`，母版是 `Component/ButtonCost` / `-Loading` / `-Disabled`。

⚠️ **現況（2026-09-10）**：程式碼裡**沒有任何一顆付費按鈕帶金額**。`MediaDetailPanel.tsx:280-292` 只有免費的「搜尋字幕」；付費入口是「管理字幕」→「生成字幕」（`ManageSubtitleDialogV2.tsx:495-509`），說明列只講「約需數分鐘」，全檔零金額字串。而且**沒有單項估價端點**——`GET /api/v1/subtitles/generation-candidates` 是整庫掃描後的快照，所以單片路徑目前是 100% 無金額，不是偶發。`Button.tsx` 也還沒有對應的變體。追蹤於 `disc-2026-09-single-item-cost-estimate` 與 `disc-2026-09-no-cost-bearing-component`。

⚠️ 本規則原先舉的例子（詳情面板三顆按鈕：搜尋字幕／AI 校正／轉錄英文音軌）出自 v1 的 `b3-d`，那張稿已於 2026-09-10 與其他 38 張 v1 過時稿一併從設計稿移除。現行的 `b3p-d` 字幕區只有一顆「管理字幕」，而「AI 校正」「轉錄中」在程式碼裡只是 SSE 進度階段字串。舉例已更新為實際的付費入口。

**主題平價規則（The Theme Parity Rule）** 任何新的顏色 token 必須**同時**寫進兩個區塊，並在 `styles-contrast.spec.ts` 補上淺色的對應案例。只加一半會讓 parity 斷言直接失敗——這是刻意的。

## Typography

**內文字體：** Noto Sans TC（後備 `-apple-system`、`BlinkMacSystemFont`、`Segoe UI`、`Roboto`、sans-serif）
**讀數字體：** JetBrains Mono（後備 Consolas、Monaco、monospace）

**個性：** 一套人文主義無襯線扛下所有閱讀工作，加一套等寬字專門留給「必須對齊或互相比較的數字」。**沒有 display 字體、沒有編輯式的字體配對**——字體系統的任務是在昏暗房間裡小尺寸也讀得清楚，第二種個性只會礙事。

CJK 一律走 Noto Sans TC。設計稿上的 DM Sans 只用於**畫布註記**（流程標題、色票標籤那些不會被實作的東西），不是產品字體。

### Hierarchy

**八階，全部偶數，每一階都帶配對行高，而且行高是為繁體中文訂的。** 行高不是可選的——它是這套系統最久的一個洞（2026-09-10 之前，設計稿 5134 個文字節點裡有 4902 個沒有設行高，全部吃字型預設）。

**行高用比例，不用 px，而且每一階都落在 Tailwind 的具名階上**（`leading-tight` 1.25／`leading-snug` 1.375／`leading-normal` 1.5／`leading-relaxed` 1.625）。這是刻意的：廢除 11／13／15 的理由就是「任意值只設字級、不帶行高」，如果行高自己變成 `leading-[22px]` 這種任意值，等於把同一個病搬到另一邊。

⚖️ **2026-09-10 修正**：先前那組行高（40／36／32／28／28／24／20／16）是 Tailwind 的拉丁文預設一字不差，內文只有 1.429、低於本文件自己規定的 1.6，而且 Heading(18) 的 1.556 比 Text(14) 的 1.429 更鬆——標題行距比內文寬，是反的。現在改成**字愈大行愈緊、給人讀的字最鬆**。

| 角色           | 桌機 px / 行高         | 手機 px | 字重 | Tailwind    | 用途                                                              |
| -------------- | ---------------------- | ------- | ---- | ----------- | ----------------------------------------------------------------- |
| **Display**    | 36 / 1.25 (45)         | **30**  | 700  | `text-4xl`  | 詳情頁 hero 標題、海報 placeholder 首字。罕見，不是一般標題層級。 |
| **H1**         | 30 / 1.25 (37.5)       | **24**  | 700  | `text-3xl`  | 頁面大標。一個畫面一個。                                          |
| **H2**         | 24 / 1.375 (33)        | **20**  | 700  | `text-2xl`  | 區段大標。                                                        |
| **H3**         | 20 / 1.375 (27.5)      | **18**  | 600  | `text-xl`   | 區塊標題。                                                        |
| **H4**         | 18 / 1.5 (27)          | 18      | 600  | `text-lg`   | 卡片與區段標題。                                                  |
| **Body Large** | 16 / 1.625 (26)        | 16      | 400  | `text-base` | 大內文、次級標題。                                                |
| **Body**       | **14 / 1.625 (22.75)** | 14      | 400  | `text-sm`   | **預設內文**，也是按鈕標籤與並排讀數。全系統用量最大的一階。      |
| **Label**      | **12 / 1.5 (18)**      | 12      | 500  | `text-xs`   | 標籤、徽章、殼層 chrome、純數字欄位。**地板。**                   |

**行高是按角色訂的，不是按尺寸算的。** Label(12) 的 1.5 比 Text(14) 的 1.625 緊，不是筆誤——徽章與 chrome 是單行、掃一眼就過，把行框撐開只會讓元件變高而讀感不變；Text 是會排成整段被閱讀的字，繁中需要那個 1.625。

**手機只縮標題，不縮內文。** Display／Headline／Title／Subtitle 各降一階，Heading 以下維持原尺寸。理由是手機上真正的問題是「大標吃掉半個螢幕」，不是「內文太大」——把 14px 內文再縮小只會讓它更難讀。行高比例兩個斷點相同：1.625 已經是繁中的目標值，不需要在手機再放寬。

**等寬**（JetBrains Mono）不是獨立的字級階，是同一階換字體：與中文並排、要被讀的讀數用 **Text（14）**；只做上下比較的純數字欄位用 **Label（12）**。

### 為什麼沒有 11 / 13 / 15

這三階在 2026-09-10 一併廢除（PR #410）。**理由不是「奇數不好看」**——那個說法站不住腳：字級本身不產生半像素（半像素來自行高與容器餘數），而且偶數也救不了行高，14 × 1.6 = 22.4 一樣是小數。

真正的理由有兩個，都可驗證：

1. **11 / 13 / 15 是唯三沒有 Tailwind 具名階的尺寸。** `text-sm` 是 14px **配 20px 行高**、`text-xs` 是 12 配 16——字級與行高一起給，且行高落在 4px 網格。但 `text-[13px]` 這類任意值**只設字級、不帶行高**，而 `styles.css` 全檔沒有宣告 base line-height。所以那 125 個站點（72 個 13px + 53 個 11px）的行框全部落到瀏覽器預設的 ≈1.2——**比這份文件給繁中訂的 1.4–1.6 更緊**。
2. **決策成本。** 廢除前設計稿的用量是 13px 1583 次、12px 1370 次、14px 784 次——三階互相搶位，沒人知道該選哪個。拿掉 13，等於拿掉一次擲硬幣。

另外 11px 有一個獨立理由：繁體中文一個字身塞 15–25 筆畫，11px 在 1x 會糊、2x 筆畫沾黏。**先前「Chrome 是殼層不是內容，所以可以 11px」的豁免已經取消**——底部分頁列標籤與側軌群組標題是手機上最主要的導覽，是使用者最需要讀清楚的字，不是裝飾。

15px 則是因為它與 16px 差 6.7%，人眼看不出來。**一個看不出差別的階不該存在。**

**地板是 12px，沒有例外。** 10px 同日廢除（它從來沒有授權來源）。

### 設計稿的字級變數

設計稿 `ux-design.pen` 用同一套角色名，每階兩個變數：

```
                       桌機  手機        行高（比例）
Type/Display/Size       36    30    Type/Display/Line   1.25
Type/H1/Size            30    24    Type/H1/Line        1.25
Type/H2/Size            24    20    Type/H2/Line        1.375
Type/H3/Size            20    18    Type/H3/Line        1.375
Type/H4/Size            18    18    Type/H4/Line        1.5
Type/BodyLg/Size        16    16    Type/BodyLg/Line    1.625
Type/Body/Size          14    14    Type/Body/Line      1.625
Type/Label/Size         12    12    Type/Label/Line     1.5
```

⚖️ **2026-09-10 改名（Alexyu 裁定）。** 原本是 Display／Headline／Title／Subtitle／Heading／BodyLarge／Text／Label——
Headline(30) > Title(24) > Subtitle(20) > Heading(18) 是四個近義詞，沒有母語者說得出誰比誰大；而 `BodyLarge` 暗示存在一個
`Body`，但不存在，真正的內文叫 `Text`，於是「以 body 命名的 token」比「真正的 body」更大。
改成 **H1–H4**（普世認知，不用學）＋ **Body(14) 是預設、BodyLg(16) 是變大的那個**。2689 個文字節點的引用已同步。

**斷點是一個變數軸，不是第二套變數。** `.pen` 支援多軸主題，檔案裡現在有兩軸：`mode`（dark／light）與 `bp`（desktop／mobile）。同一個 `$Type/H2/Size` 在標了 `theme:{bp:"mobile"}` 的畫面上解析成 20，在沒標的畫面上解析成 24。**沒標的一律吃桌機值**，所以加這個軸不會動到任何既有畫面——這點與日巡那次一樣，加軸前已用暫時變數實測過四種組合。目前 56 張手機稿已全部標上 `bp:"mobile"`。

⚠️ **`.pen` 的 `lineHeight` 是比例不是 px。** 填 20 代表 20 倍行高，不是 20px——實測會讓整份檔案的裁切警告從 150 暴增到 2164。

**程式碼尚未跟上**：`apps/web` 還有 125 處 `text-[13px]`／`text-[11px]`、6 處 `text-[10px]`、4 處 `text-[15px]`，另有 5 處 `text-3xl`(30) 與 1 處 `text-5xl`(48)。48px 不在這張表裡，要嘛補一階要嘛改掉。**行高也還沒跟上**：Tailwind 的 `text-sm` 自帶 1.429，要拿到這張表的 1.625 必須明寫 `leading-relaxed`；`text-lg` 自帶 1.556，要 1.5 必須明寫 `leading-normal`。而**手機的字級降階目前在程式碼裡完全不存在**——沒有任何 `sm:text-*` 的響應式字級。追蹤於 `disc-2026-09-type-scale-even-migration`。

### Named Rules

**預設小字規則（The Small-By-Default Rule）** 14px 是內文、12px 是標籤。伸手拿更大的尺寸，等於主張「這段文字比頁面的實際內容更重要」——而它通常不是。實測 623 次 `text-sm` 對 6 次 `text-4xl` 是**預期的形狀，不是待修正的意外**。

**比較才用等寬規則（The Mono-For-Comparison Rule）** 如果同一個值的兩個實例可能被沿著欄位上下對照，就用等寬。散文永遠不用等寬；使用者只掃一眼、不會拿去比的數字也不需要。

**12px 是地板（The 12px Floor Rule）** 10px 與 11px 都已於 2026-09-10 廢除（⚖️ Alexyu 裁定，PR #410）。

10px 從來沒有授權來源：2026-09-01 的 critique 就裁掉過 `InFlightBadge` 的 10px，但那一階仍靠著沒人管的一次性寫法活了下來。廢除的理由是產品自己的北極星——**讀數看不清等於沒有讀數**，而程式碼那 6 處裡有兩處是使用者必須讀的錯誤訊息。

11px 隨後一併廢除。先前的豁免寫的是「Chrome 是殼層不是內容，所以可以 11px」，**那條豁免已經取消**：底部分頁列標籤與側軌群組標題是手機上最主要的導覽，是使用者最需要讀清楚的字，不是裝飾。另外繁體中文一個字身塞 15–25 筆畫，11px 在 1x 會糊、2x 筆畫沾黏。

**地板是 12px，沒有例外，也沒有兩層。** 設計稿現在沒有任何低於 12px 的字級變數可用，這是刻意的——要用更小的字，得先改這條規則。

⚠️ 程式碼那 6 處 `text-[10px]` 尚未改（會動到視覺基準），追蹤於 `disc-2026-09-code-still-uses-10px`。

## Layout

App 是「固定左側軌 ＋ 流動內容欄」。左側軌展開 240px、收合 64px，狀態存在 `localStorage`；在 `sm` 斷點以下改為底部分頁列加 More sheet。

內容頁以 `max-w-7xl`（1280px）容器搭配 `mx-auto` 置中，這是**頁面內容**的規則。**它不是「本身就有側欄的頁面」的規則。** 把「包含側欄的整塊」加上寬度限制再置中，會讓那個側欄脫離 App 側軌，在兩個導覽之間留下一道死掉的垂直空白；設定區在 1920px 下實地踩過這個坑——子導覽整整離開它該貼著的側軌 200px。**巢狀側欄的版型，根層不設限也不置中，寬度限制搬到內容格並且靠左對齊。**

間距走 **Tailwind 的完整預設階梯**：0 / 2 / 4 / 6 / 8 / 10 / 12 / 14 / 16 / 20 / 24 / 28 / 32 / 40 / 48 / 64 / 80（設計稿的 `Space/*` 一階不多一階不少，含 `Space/none`＝0）。**半階不是漂移，是這套系統真的在用的階**——程式碼實測用了 451 次（`gap-1.5` 212、`py-0.5` 112、`py-2.5` 100、`px-3.5`/`py-3.5` 27）。

> 本文件在 2026-09-10 之前寫的是「4 / 8 / 12 / 16 / 24 / 32」。那是一個想像出來的子集，設計稿與程式碼**都沒有**遵守它——是文件錯，不是兩邊漂移。

半階各有明確用途——⚖️ 2026-09-10 依實測用量重寫，原本寫的兩條理由是錯的：

- **2px（`Space/2xs`）**：徽章的垂直內距（badge 規格本身就是 `2px 8px`）。實測 padV 前幾名全是 `countChip`、`grade-badge`、`route-badge`，與規格一致。⚖️ 原名 `Space/hairline` 已改名——「髮絲線」在本文件指的是 1px 邊框，同一個詞不該同時是 2px 的間距階。
- **6px**：徽章／藥丸裡圖示與文字的間隔。12px 文字配 4px 太黏、8px 太散。實測 510 次並排 gap，是這一階最主要的身分。
- **10px**：**側軌導覽項、分頁列與徽章的內距**（實測 padH 前幾名：`tbb` 分頁列 44、`nav-電影/影集/動畫` 各 25、`route-badge`、`statusbadge`、`countChip`）。⚠️ 原本寫的是「要湊出 32–36px 高度的 chip／pill 垂直內距」——**實測推翻**：padV=10 的容器有 64% 是 **43px** 高，落在 30–38px 區間的只有 4%。
- **14px**：14px 文字的列內距，12 太緊、16 太鬆。
- **20px**：**CJK 可按控制項的水平內距**。拉丁字母左右自帶 side bearing，CJK 字身框直接貼到內距邊——同樣 16px，「立即掃描」看起來會比 `Scan Now` 擠。中文排版的通行補償約 0.25em，14px × 0.25 ≈ 4px，所以是 16 + 4。實測 padH 前幾名全是按鈕（`btn-cancel`、`act-edit`、`查看詳情按鈕`、`addlist`），**這條理由成立**。
- **28px**：**頁面 header／toolbar 的水平內距**（實測 padH：`header` 8、`toolbar` 8）。⚠️ 原本寫的是「區段間距，24 太近、32 太遠」——**實測推翻**：它當垂直堆疊 gap 只用過 6 次。

卡片內距 24px，輸入框 8px 垂直、12px 水平，列表列垂直 8–12px。

### 間距選擇規則（Spacing Selection Rules）

⚖️ 2026-09-10 補上（`disc-2026-09-spacing-scale-grouping` ＋ `disc-2026-09-spacing-has-no-selection-rules`，兩張單子合併）。在那之前 16 階沒有任何一條選擇規則，結果 `sm`(8) 用 1791 次、`sm-plus`(10) 用 1044 次——同一個數量級，代表沒人知道什麼時候該選 10。

**規則零：可按控制項的高度是寫死的，不是用內距算出來的。**

這條先講，因為它讓「8 還是 10」這個問題在最容易撞的地方直接消失。按鈕 36px（小 32／大 40／手機觸控 44）、輸入框 36px、分頁列 48px——母版上直接給 `height`，內容置中，垂直內距不參與計算。

為什麼不能用內距算：內文行高改成 1.625 之後，14px 文字的行框是 22.75px。要湊到 36px 需要 (36 − 22.75) ÷ 2 = 6.6px，**階梯上沒有這個數**。實測也證實了——padV=8 有 67% 產生 **39px** 高的容器、padV=10 有 64% 產生 **43px**，兩個都不是規格裡的任何一個高度。**用內距追高度，追不到。**

**其餘三個角色，各一句話：**

| 角色                                    | 可用階           | 預設 | 什麼時候不用預設                                             |
| --------------------------------------- | ---------------- | ---- | ------------------------------------------------------------ |
| **Inline** 並排（圖示↔文字、chip 之間） | 4 / 6 / 8        | 8    | 文字是 12px Label 用 **6**；數字＋單位要讀成一個東西用 **4** |
| **Stack** 垂直堆疊（列與列）            | 8 / 12 / 16 / 24 | 12   | 密集清單用 **8**；子區段之間 **16**；主要區段之間 **24**     |
| **Inset** 容器內距                      | 見下             | —    | 水平永遠比垂直大一階                                         |

**Inset 沒有單一預設，因為它跟「這是什麼東西」綁在一起：**

| 東西                                    | 垂直 | 水平 |
| --------------------------------------- | ---- | ---- |
| 徽章／chip                              | 2    | 8    |
| 側軌導覽項、分頁列、狀態徽章            | 10   | 10   |
| 內容列（表格列、清單列）                | 8    | 12   |
| 可按的 CJK 控制項（水平；高度見規則零） | —    | 20   |
| 卡片                                    | 24   | 24   |
| 對話框                                  | 24   | 24   |
| 頁面 header／toolbar                    | 12   | 28   |
| 頁面／面板外框                          | 32   | 32   |

**「水平比垂直大一階」是實測出來的，不是美學主張**：padH 集中在 12(566)／16(373)，padV 集中在 8(463)／10(304)。橫向的字比縱向的行密，所以橫向需要更多空氣。

⚠️ **這套規則目前只寫在文件裡，還沒有守門測試。** 間距與圓角、字級一樣不在 `check-design-tokens.py` 的檢查範圍內（它只比對顏色），追蹤於 `disc-2026-09-drift-checker-blind-to-pen-and-non-color`。

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

⚖️ **2026-09-10 重寫（Alexyu 裁定，PR #410）。舊規則「絕不做成藥丸形」作廢。**

舊規則從來不是一個決定。它是 2026-08-25 的 `/impeccable document` 在「把現有視覺系統記錄下來」時寫的**觀察**，原句後面帶著數字：8px 實測 98 次、12px 36 次、4px 32 次、16px 只用了一次。它數的是當時的 `apps/web`，不是設計稿。2026-08-26 又為它補了一條「高度 ≤ ~22px 的圖上微型元件可以是藥丸」的例外。

兩年都不到，兩邊都對不上了。今天的 `apps/web`：

| 圓角                   | 次數 |
| ---------------------- | ---- |
| `rounded-lg`（12px）   | 237  |
| `rounded-full`（藥丸） | 167  |
| `rounded-md`（8px）    | 69   |
| `rounded-xl`（16px）   | 13   |
| `rounded-sm`（4px）    | 1    |

規則說「8px 獨大」，實際上 12px 才是。規則說「絕不藥丸」，藥丸是第二名。規則說「徽章用 4px」，4px 全專案用了一次。設計稿那邊藥丸用了 727 次，而且**規則親自點名要排除的 FilterChip、GenreTag、四個 TechBadge，六個母版本身全是藥丸**。`styles.css` 裡也根本沒有 `--radius-pill` 這個 token。

而且「用高度判斷」這件事沒人能一致執行：同一個徽章放海報上是藥丸、放清單列裡變 4px——同一個元件因為背景而換形狀，保證會被畫錯。

### 新規則：看語意，不看高度

| 圓角                      | 給誰                                             | 例子                                      |
| ------------------------- | ------------------------------------------------ | ----------------------------------------- |
| **藥丸**（`radius-pill`） | **可摘除或可切換的資料標記**                     | FilterChip、GenreTag、TechBadge、狀態徽章 |
| **8px**（`radius-md`）    | **可按的動作控制項**                             | Button、輸入框、下拉、側軌項目            |
| **4px**（`radius-sm`）    | **不可互動的小色標**                             | 純裝飾的色塊、進度條端點                  |
| **12px**（`radius-lg`）   | **卡片與較大的容器**                             | PosterCard、對話框、面板                  |
| **16px**（`radius-xl`）   | 大型容器的特例。實測 95 次，不是「幾乎未使用」。 | Sheet 頂角、大型 modal                    |

判斷法只有一句：**這個東西是「一枚可以被拿掉的標記」還是「一個可以被按下去的動作」？** 前者藥丸，後者 8px。標記做成藥丸是近乎普世的模式，使用者一眼就知道它可以被摘掉。

**邊框是 1px 髮絲線，只出現在色階本身不足以分開兩樣東西的地方**——外框按鈕、靜止狀態的輸入框、表格與列表分隔線。有填色的面不會同時再給邊框。

### Named Rules

**單一圓角規則（The One Radius Rule）** 動作控制項沒有理由就用 8px。這條仍然成立，但它管的範圍縮小了：它只管「可按的動作」那一類，不再是全域預設。標記類一律藥丸，卡片類一律 12px，兩邊都不需要理由。

## Responsive

> PRODUCT.md §Operating Context 寫的是「手機與桌機同等重要……手機上要能完成完整任務，不只是查看進度」。在 2026-09-10 之前，這份文件給手機的全部篇幅是兩句話（「sm 斷點以下改底部分頁列加 More sheet」＋「觸控目標最小 44px」），而且那兩句在 §Components 又重複了一次。以下是補上的規格。

### 兩個畫布，三個斷點

設計稿只畫兩種尺寸：**桌機 1440×900**、**手機 390×844**（iPhone 14 的邏輯像素）。中間尺寸不畫稿，用規則推導。

程式碼實際用到的斷點（Tailwind 預設值）：

| 斷點  | 寬度   | 用量   | 意義                                                  |
| ----- | ------ | ------ | ----------------------------------------------------- |
| `sm`  | 640px  | 209 次 | **主分界**。以下是手機版面：底部分頁列、單欄、Sheet。 |
| `lg`  | 1024px | 72 次  | 側軌從覆蓋式改為常駐、篩選 rail 出現。                |
| `md`  | 768px  | 46 次  | 網格欄數的中繼調整。                                  |
| `xl`  | 1280px | 11 次  | 罕用。網格再加一欄。                                  |
| `2xl` | 1536px | 1 次   | 實質未使用。                                          |

**新畫面只需要決定 `sm` 兩側長什麼樣。** `md`／`xl` 是網格欄數的插值，不是新版面。

### 手機上什麼會變、什麼不會變

**會變的只有三件事：**

1. **標題字級降一階。** Display 36→30、Headline 30→24、Title 24→20、Subtitle 20→18。Heading 以下不變。見 §Typography。
2. **導覽從側軌換成底部分頁列。** 四個主要目的地上分頁列，其餘進 More sheet。
3. **覆蓋層從 Dialog 換成 Sheet。** 桌機置中的對話框，在手機一律改成從底部升起的 Sheet。

**刻意不變的：**

- **間距只有一套階梯。** `Space/*` 沒有手機版。密度差異靠版面決定（單欄、更少同時可見的區塊），不靠第二套 token——兩套間距等於每個節點都要做一次選擇，而那個選擇沒有人能一致執行。
- **內文字級不變。** 14px 在手機仍是 14px。手機的問題是大標吃掉螢幕，不是內文太大。
- **行高比例不變。** 1.625 已經是繁中的目標值。

### 觸控目標

**任何可按的東西，命中區最小 44×44px。** 視覺尺寸可以更小（12px 的徽章仍然是 12px），但命中區要靠內距或透明外框補到 44。

⚠️ **這條目前是空頭支票**：設計稿的按鈕最大 40px，程式碼的 `default` 是 36px，**沒有任何尺寸做得到 44**。兩邊都要補一個 `touch` 尺寸，追蹤於 `disc-2026-09-missing-component-masters`。

### Sheet

手機的主要覆蓋層。規格：

- **從底部升起**，頂角 16px（`radius-xl`），左右下三邊貼齊螢幕。
- **頂部有一條 36×4px 的拖曳把手**，`--text-muted`，置中。它同時是「這東西可以拖下去關掉」的唯一提示。
- **最大高度是螢幕的 80%**（844 × 0.8 ≈ 675）。內容更長時內部捲動，Sheet 本身不長高。
- **背後是 `--overlay-scrim`**，點擊關閉。
- 內距 16px 水平、頂部 12px、底部留出安全區。

### 現況缺口

⚖️ **2026-09-10 已補**：`Component/BottomSheet` 與 `Component/Button/Touch`（44px）都有了，母版總數 38 → 68。在那之前 20 幾張手機稿各自手畫同一個 Sheet 外殼，因為沒有母版可以 instance。

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

- **樣式：** 藥丸形（`radius-pill`）、12px 標籤字、4px／10px 內距、12px 圖示。⚖️ 2026-09-10 從 4px 改為藥丸——見 §Shapes：徽章是「一枚可以被拿掉的標記」，不是「一個可以被按下去的動作」。
- **狀態變體**以語意 tint 為底、`*-text` 階為字——青碧、赭、硃砂、靛青。**底用 tint、字用 `*-text`，不要用飽和色當字。**
- **中性變體：** `secondary` 用 `--bg-tertiary`；`outline` 是髮絲線邊框加次要文字色，**用於分類而非狀態**。

### Cards and Containers

- **圓角：** 12px。
- **底色：** `--bg-secondary`。
- **邊框：** `--border-subtle` 髮絲線，**1px，預設有**。
- **陰影：** 無。陰影保留給真的浮在頁面之上的東西（Dialog／Sheet／Popover／Toast），見 §Elevation。
- **內距：** 24px，header／content／footer 共用這個內縮，footer 去掉上內距。

⚖️ **2026-09-11 裁定：「邊框預設無、同時有色階又有邊框是過度指定」這一條刪除。**
原規則假設陰影在做事。實測它在夜行沒有在做事：

| 夜行的卡片邊界線索                                                | 對比        |
| ----------------------------------------------------------------- | ----------- |
| `--shadow-md`（`0 4px 8px rgba(0,0,0,.4)`）疊在 `--bg-primary` 上 | **1.056:1** |
| 色階 `--bg-secondary` 對 `--bg-primary`                           | 1.139:1     |
| **髮絲線 `--border-subtle` 對頁面底**                             | **1.660:1** |

陰影帶來的亮度差是 0.0057，髮絲線是它的 **12 倍**；而且陰影是 4px 下偏移，**卡片上緣連一個陰影像素都沒有**。把 alpha 拉到規格上限 0.6 也只到 1.082:1。同一個 token 在日巡是 **1.235:1**，比日巡自己的色階（1.138:1）還強——因為日巡的陰影後來被認真重做過（改墨色、兩層、總墨量 ≤15.4%），而沒有人回頭為夜行驗算。**原規則只在較晚加入的那個主題成立。**

實際採用率也證實了這件事：`shadow-[var(--shadow-md)]` 全 app 出現 **2 次**，`border border-[var(--border-subtle)]` 出現 **151 次**。唯一照原規則寫的元件是 `apps/web/src/components/ui/Card.tsx`，而它在 production 零使用——唯一 import 它的是視覺回歸的測試夾具。**一條沒有任何實作者、且在主要主題裡物理上不可觀測的規則，是這份文件裡最鬆的一塊。**

**配給規則（同時採用，否則邊框會累積）：同一層級只准一道框。** 容器有框，其內的列用 `--border-subtle` 分隔線而不是各自加框；巢狀第二層以上的容器靠間距分隔，不再加框。

**鑑別責任移交**：卡片與輸入框現在共用同一種邊框，兩者的區分改由**圓角**承擔——卡片 12px、輸入框 8px。

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

`ux-design.pen` ▸ `Design System · 設計系統` ▸ `Components · 元件` 底下有 **68 個母版**，分五類。母版住在分類裡，Component Library 那頁放的是 instance。

⚖️ **2026-09-10 補了 30 個**（`disc-2026-09-missing-component-masters`）。在那之前是 38 個，而 DESIGN.md 自己規格化的元件有 14 個只有 3 個真的存在母版——元件庫空狀態區那顆 outline 按鈕是**手畫的**，因為沒有 ButtonOutline 可以 instance。

| 分類                      | 元件                                                                                                                                                                                                                                                                                                                          |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 01 · 導覽與外框（9）      | SidebarNavItem、SidebarGroupParent、SidebarGroupLabel、SidebarFooterStatus、MobileTabItem、HomeSidebar-v2、**DialogFrame**、**BottomSheet**、**Pagination**                                                                                                                                                                   |
| 02 · 基礎控件（34）       | ButtonPrimary／Secondary／**Outline**／**Ghost**／**Destructive**／**Touch**／**Disabled**、ButtonCost（含 Loading／Disabled）、SearchInput、**TextField**／**TextFieldFocus**、SortDropdown、FilterChip、GenreTag、Checkbox 五態、TabActive／TabInactive、**SwitchOn**／**SwitchOff**、**Tooltip**、**Text/\* 八階字級母版** |
| 03 · 媒體卡片與標籤（12） | PosterCard、PosterCardHover、PosterCard-v2、TechBadge-Video／Audio／Subtitle／HDR、**StatusBadge-Running／Done／Asked／Fault／Note**                                                                                                                                                                                          |
| 04 · 列表與進度（10）     | ActivityRow-v2、RequestRow-v2、DownloadCard-v2、GenerationProgress-v2、GlossaryRow-v2、GenQueueRow-v2、**ProgressBar**、**TableRow**、**Toast**、**Skeleton**                                                                                                                                                                 |
| 05 · 空狀態（3）          | EmptyLibrary-NoQBT、EmptyLibrary-NoFolder、EmptyLibrary-ReadyForScan                                                                                                                                                                                                                                                          |

**`Text/*` 是純文字母版，一階一個。** 新畫面的文字一律 instance 它們，不要新建 text node——那是「95.5% 的文字沒有行高」那個洞唯一補得起來的辦法：母版帶著 Size／Line／Weight 三個變數，instance 只換內容。

**`ButtonTouch` 是 44px 的行動尺寸**，補上 §Responsive 那張空頭支票。桌機仍用 36px 的 `ButtonPrimary`。

**`StatusBadge` 五態對應五個狀態詞**：Running 泥金＝正在跑、Done 青碧＝有答案了、Asked 赭＝要求了但沒發生、Fault 硃砂＝壞了、Note 靛青＝純告知。底用 `*-tint`、字與圖示用 `*-text`，永遠成對。

**TechBadge 一律中性**（`--bg-tertiary` 底、`--text-secondary` 字），四個分類只靠文字與圖示區分。

> ⚖️ **2026-09-10 修正（Alexyu 裁定）**：這裡原本寫「video → accent、audio → info、hdr → warning、subtitle → success」，也就是**本文件親自把四個狀態色指派成一組媒體規格分類法**——而 §Colors 的固定詞彙規則明文禁止狀態色用於分類。後果在畫面上可見：海報左上角的青碧「繁中」徽章是分類（這個檔案有中文字幕軌），詳情面板的青碧「繁體中文字幕已就緒」是狀態（有答案了）——**同一個顏色在相鄰畫面有兩個意思**。
>
> 裁定：技術規格（H.265／DTS／HDR10／繁中）是這個檔案的**屬性**，不是「發生了什麼」，本來就不該穿狀態色。它們已經有文字，顏色沒有承載額外資訊。改成中性之後，使用者看到青碧就只有一個意思。設計稿四個母版已改；程式碼 `TechBadge.tsx:15-18` 仍是舊映射，追蹤於 `disc-2026-09-techbadge-uses-status-colors-as-taxonomy`。

## Do's and Don'ts

### Do：

- **Do** 嚴格照狀態詞彙用色：金＝正在跑、青碧＝有答案了、赭＝要求了但沒發生、硃砂＝壞了、不出現＝沒要求。
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

## 怎麼新增一張設計稿（SOP）

> 這一節在 2026-09-10 補上（`disc-2026-09-design-sop`）。在那之前沒有任何文件講「元件與 token 怎麼挑」——`design-context-pack.md` 是給 Pencil 內建 AI 的產品 primer，memory 的 flow-layout-convention 只講座標與命名。結果是每張新畫面都在重新發明一次選擇。

按順序做完七步。**每一步都有一個「不准」，那才是這份 SOP 的重點。**

### 1. 先決定殼層 — 只有一個答案

**v2 側軌是唯一正典。** 程式碼 `routes/__root.tsx:173` 掛的是 `AppShellV2`，底下是 `AppSidebar` ＋ `MobileTabBar` ＋ `MobileMoreSheet`；**全專案沒有 TopBar 元件**。

設計稿裡仍有 v1 頂欄的舊畫面（A1-D、B1-D、C1-D 那一批），它們是遺留，留著只為了對照歷史。**新畫面一律不得使用 v1 頂欄。**

桌機用 `Component/HomeSidebar-v2`；手機用底部分頁列（`Component/MobileTabItem`）加 More sheet。

### 2. 決定它落在哪個 Flow、叫什麼名字

- **Flow group**：A 瀏覽／B 詳情與互動／C 搜尋・篩選・設定／D 下載管理／E 媒體庫掃描／F 字幕搜尋與批次／H 首頁／I 進階搜尋與探索／J 設計決策 Spec／K 活動中心／L 想要與請求系統／M 登入與密碼閘。**沒有 Flow G**——它的六張稿在 2026-09-10 隨 v1 清理刪除（早已被 flow-f-subtitle-v2 取代），空群組一併移除。找不到歸屬就是這張稿的定位有問題，先想清楚再畫。
- **圖框名稱**用短碼：`B3-D`（桌機）、`B3-M`（手機）、v2 改版加後綴 `B3-D-v2`。
- **畫布可見標題**另開一個 text node：`B3 · 詳情面板・電影（桌面）`，Noto Sans TC 14／600／`#888888`，放在 frame **上方 45px**——貼太近會撞到 Pencil 自己的圖框名 chrome。
- **位置**用 `FindEmptySpace({nodeId: 同流程最後一張})` 錨定，不要自己挑座標。
- **群組內部再分三層**：`X · 桌面 Desktop`／`X · 手機 Mobile`／`X · 規格 Spec`。流程標題與描述留在最外層，不進子群組。

#### 畫布版面（2026-09-10 重排）

⚠️ **memory 裡那個「各流程間距 2600」的座標範本已作廢。** 流程長高之後全部互相穿插——重排前實測有 **34 組最外層群組彼此重疊**，Flow J 疊在 Flow A 上、Flow F 疊在 Design System 上。現行規則：

- **所有最外層群組靠左對齊 `x = 17000`。**
- **垂直堆疊，群組之間固定留 2000px。** 順序：Design System → A → B → C → D → E → F → H → I → J → K → L → M。
- **群組有可用的 `x`／`y` 位移屬性**（bounds = 子節點最小座標 ＋ 群組的 x/y）。要搬一整個流程，`Update(groupId, {x, y})` 就夠，不必逐一搬子節點。
- **Design System 內部同樣垂直堆疊**：`Docs · 設計文件`（五張說明頁橫排一列，間距 200）→ `Components · 元件`（五個分類）→ `Themes · 主題證據`。
- **新增流程或畫面之後要重跑一次對齊**，並確認最外層與同層畫面的重疊組合都是 0。這是機器檢查得出來的：兩兩比對 `ctx.bounds` 的矩形交集即可。

### 3. 必須 instance 母版，不准手畫

**任何在別的畫面出現過的東西，一律 `Copy` 母版或插 `ref`，不得重畫。** 手畫出來的東西不會跟著母版更新，就是漂移的來源——元件庫空狀態區那顆手畫的 outline 按鈕就是這樣來的。

**沒有母版怎麼辦：先補母版，再畫畫面。** 順序反過來就永遠補不上——2026-09-10 之前 38 個母版裡只有 1 個行動專用，20 幾張手機稿各自手畫同一個 Sheet 外殼，就是順序反了的結果。

新母版一律放進 `Components · 元件` 底下對應分類（01 導覽與外框／02 基礎控件／03 媒體卡片與標籤／04 列表與進度／05 空狀態），並在 Component Library 頁補一格。

**命名有兩個軸，用兩種分隔符號，不要混：**

| 寫法                            | 意思                     | 例子                                                            |
| ------------------------------- | ------------------------ | --------------------------------------------------------------- |
| `Component/<名字>`              | 一個獨立元件             | `Component/SearchInput`、`Component/Pagination`                 |
| `Component/<名字>/<狀態或變體>` | **同一個元件的不同狀態** | `Component/Checkbox/Indeterminate`、`Component/Button/Ghost`    |
| `Component/<名字>-v2`           | **同一個位置的下一代**   | `Component/PosterCard-v2`（與 `PosterCard` 並存，不是它的狀態） |
| `Text/<字級角色>`               | 純文字母版，一階一個     | `Text/H2`、`Text/Body`                                          |

⚖️ 2026-09-10 從黏字命名改過來（`CheckboxDisabledChecked` → `Component/Checkbox/DisabledChecked`，33 個母版）。理由是黏字再加兩個狀態就會變成 `CheckboxDisabledIndeterminateReadonly`；斜線讓 Pencil 的圖層面板把同前綴收在一起，未來若支援 variant 也能無痛遷移。

**變數命名**（82 個，全部符合）：顏色與圓角沿用 `styles.css` 的 CSS 變數名一字不差（`bg-primary`、`accent-tint`、`radius-md`）；程式碼沒有的階梯才用語意化群組名（`Space/lg-plus`、`Type/H2/Size`、`Type/Family/Mono`）。

### 4. 顏色只准用 `$` 變數

- **不准寫死 hex。** 唯一例外是畫布註記（流程標題、色票標籤那些不會被實作的東西）。
- **`accent-primary` 給人按，`accent-text` 給人讀。** 換過來會無聲地破壞對比度。
- **徽章底用 `*-tint`，字用 `*-text`，永遠成對。** 語意基色本身（success／warning／error／info／accent-primary）**不可以當文字**。
- **壓在圖片或深色遮罩上的字用 `text-on-scrim`**，不要用 `text-primary`——後者會隨主題翻轉，日巡下變成黑字壓黑底（追蹤於 `disc-2026-09-text-on-artwork-flips-with-theme`）。
- **金額一律中性色**，狀態押在標籤與圖示上。會花錢的控制項要帶預估金額。

### 5. 字級選 `Type/*`，三個變數一起設

每一個 text node 都要同時設 **`$Type/<角色>/Size`、`$Type/<角色>/Line`、`$Type/<角色>/Weight`** 三個。只設 Size 就是這套系統最久的那個洞（2026-09-10 之前 95.5% 的文字沒有行高）。

**手機稿必須在 root frame 加 `theme:{bp:"mobile"}`**，標題四階才會降級。忘了加就會拿到桌機的 36px 大標。

### 6. 間距一律用 `Space/*` 變數，一個裸數字都不准

**`gap` 與 `padding` 只能填 `$Space/*`，沒有例外**——連 0 都有變數（`Space/none`）。這樣「間距有沒有漂移」才是機器檢查得出來的，不是靠人眼掃。⚖️ 2026-09-10 已把設計稿裡最後 437 個裸數字全部轉成變數，現在是 0。

階梯有 17 階：0／2／4／6／8／10／12／14／16／20／24／28／32／40／48／64／80。**半階（6／10／14／20／28）不是「將就」，是有明確用途的正式階**。挑哪一階不用猜——§Layout 的〈間距選擇規則〉有三個角色（Inline／Stack／Inset）與一張 Inset 對照表，照著查。

**現有的階都不合用時，不要自己發明數字，也不要在旁邊寫一段理由把它合理化。** 那是把設計系統的決定塞進一張畫面裡，下一個人看不到、檢查器也抓不到。正確做法是**停下來送裁決**：

1. **UI/UX designer、CD、Agent** 先對「這個位置到底需要什麼」做設計裁決。
2. **PM** 決定這件事要不要動到設計系統本身——是新增一個 token、改掉一個既有 token，還是判定現有階其實夠用。
3. 裁決下來之後再改設計稿，並同步 `DESIGN.md`、`styles.css` 與漂移檢查器。

⚠️ 目前**還缺的是選擇規則**，不是階數：實測 `Space/sm`(8) 用 1791 次、`Space/sm-plus`(10) 用 1044 次，用量同一個數量級，代表沒人知道什麼時候該選 10。按用途分組（Inset 容器內距／Stack 垂直堆疊／Inline 並排）是待辦項，追蹤於 `disc-2026-09-spacing-scale-grouping`。

### 7. 收工檢查清單

按順序跑完，一項都不能跳：

1. **存檔**（Pencil 的 MCP 讀的是記憶體不是磁碟）：AppleScript 點 File ▸ Save，然後確認 `git status --porcelain ux-design.pen` 出現 ` M`。
2. **驗版面**：`Get((n,c)=>c.problems && ...)` 數全檔裁切警告。**基準值是 74**（2026-09-11 補上 12 張手機設定分頁、11 張狀態稿與首次啟動精靈後，從 59 上升——那 12 個警告全是分頁列刻意的橫向捲動裁切；在那之前是 2026-09-10 逐一修正 Flow A–M 內容從 116 降到 59。剩下的多半是刻意的橫向捲動出血與長頁面的折線下內容），超過就是新的破版，要修到回來為止。
3. **驗日巡**：在 root frame 加 `theme:{mode:"light"}` 截一張，確認沒有黑字壓黑底或濁色。看完把屬性拿掉（除非這張本來就是日巡證據畫面）。
4. **註冊截圖**：新畫面要加進 `scripts/export-pen-screenshots.py` 的 `SCREENS`（key = node ID，value = `(flow-folder, code)`）。加之前先數一次重複 key。
5. **重產截圖**：`python3 scripts/export-pen-screenshots.py`。
6. **只 commit 真改動的 PNG**：`git checkout` 掉沒有真的改變設計的那些——重產是非決定性的，全部 stage 會混進一堆 re-render 雜訊。判準是「改變的張數 == 預期被改到的畫面」，數字對不上就是有東西被誤傷。
7. **跑 `python3 scripts/check-design-tokens.py`**。它現在比對五份來源（styles.css、本文件、context-pack、design.json、**設計稿本身**），守門顏色、圓角、間距階梯、字級行高與裸數字。第 5 步的匯出腳本會順便更新 `_bmad-output/pen-tokens.json`——那份快照要跟 `.pen` 一起 commit，否則 CI 會判定過期。

### 已知地雷

- **`.pen` 的 `lineHeight` 是比例不是 px。** 填 20 代表 20 倍行高。
- **新 `Insert` 的節點在存檔前不會被截圖畫出來**——bounds 正確、fill 正確，就是一片空白。存檔後再截。
- **`Copy` 的 `descendants` 用名稱當 key 只對 reusable 元件有效。** 複製一般節點時名稱 key 會被靜默忽略，要 `Copy` 之後 `Get` 建一張「名稱 → 新 id」表再逐一 `Update`。
- **色票／示範一定要有明確底色。** tint 是半透明的，沒有底板時會疊到不確定的背景，匯出看起來像實心色。
- **`f24-d-v2` 的 `b4DwH` 是壞節點**（一個 text 與一個 frame 共用 id），所有批次腳本都要跳過。
