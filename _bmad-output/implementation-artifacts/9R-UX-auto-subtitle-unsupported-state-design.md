# 9R-UX-auto-subtitle-unsupported-state-design

**Agent:** Sally（UX Designer） ｜ **類型:** design-only，零程式碼
**來源:** 9R-10b 補審 M4（Rule 24 ②） ｜ **日期:** 2026-08-21
**裁定者:** Alexyu（2026-08-21，兩項）

---

## 這是什麼狀態

一個 NAS 使用者打開「編輯媒體庫」，看到最下面那個 checkbox：
**「新檔入庫後，自動完成免費的字幕處理」**。他想要。他勾下去、按儲存、成功。

然後什麼都沒有發生。永遠不會發生。

因為 `VIDO_SUBTITLE_PIPELINE_MODE` 的出貨預設是 `legacy`（`config.go:160`），
而真正會去做這件事的 `AutoGenerator` 只在 `main.go` 的 `if cfg.SubtitlePipelineEnabled()`
區塊裡才被建構。**欄位寫得進資料庫，執行的人不存在。**

PR #250 的處理是保守的：legacy 模式下**整個欄位隱藏**。不說謊了，但也不再告訴他
「有這個功能、你可以開」。這份設計是那個選擇的下一步。

---

## 兩項裁定（Alexyu, 2026-08-21）

### ⚖️ 裁定 1：走停用態，且文案**要顯示 env var 名稱**

不是隱藏，是**顯示但停用，並附上可執行的下一步**。

env var 名稱會出現在使用者眼前。理由很直接：那是**唯一可行的下一步**。
`VIDO_SUBTITLE_PIPELINE_MODE` 沒有設定頁入口、改完要重啟容器
（`docs/deployment.md:134` 明列它是無 UI 的環境變數之一）。
把名字藏起來，等於請使用者去猜一個他猜不到的字串。

**文案沿用已出貨的那一句，一字不差** —— `subtitle_pipeline_handler.go:112-113`
的 409 回應早就在講同一件事：

```
"字幕生成管線尚未啟用"
"請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。"
```

同一個動作用同一組字，一個透過 API 撞過 409 的使用者，在 modal 裡讀到的會是他認得的句子。
這是 sub-2-1a 的先例（兩個 409 共用 error code、但因為**下一步動作不同**而給不同訊息）。

### ⚖️ 裁定 2：`LibraryCard` 也要有文案，不是靜靜藏起來

`LibraryCard.tsx:115` 現在只讀 `library.autoSubtitle`，**沒有 capability 閘門**。
legacy 模式下只要 DB 還留著 `auto_subtitle=1`（在 pipeline 模式勾過、或經 API 寫入），
卡片就會亮著綠字說「· 自動處理免費字幕」——
**宣稱一件沒在發生的事**。PR #250 擋了 modal 沒擋卡片。

裁定：這一格要有自己的文案，不是消失。使用者表達過意願，系統要說明白它沒在做。

---

## 決策矩陣（四格全覆蓋）

| 伺服器模式 | 媒體庫是否勾選 | Modal 欄位 | 卡片 footer |
|---|---|---|---|
| `pipeline` | 已勾 | 正常，可變更 | 「· 自動處理免費字幕」**綠** `$success` |
| `pipeline` | 未勾 | 正常，可變更 | 整段不出現 |
| `legacy` | 已勾 | **停用 ＋ 通知列** | **「· 自動處理免費字幕（伺服器未啟用）」琥珀** `$warning` |
| `legacy` | 未勾 | **停用 ＋ 通知列** | 整段不出現 |

**顏色是一套規則，不是三個決定：**

- 🟢 綠 `$success` = **正在發生**
- 🟠 琥珀 `$warning` = **你要求了，但沒在發生**（系統與使用者意願不一致的**唯一一格**）
- ⬜ 不出現 = **你沒要求**

沒勾又沒支援的那一格刻意留白：使用者沒表達過意願，卡片沒有義務替他擔心。
他要是想開，會去打開 modal —— 那裡才是解釋的地方。

---

## 文案定稿

### Modal 停用態通知列（新增，兩句）

```
字幕生成管線尚未啟用，這個選項無法變更。
請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。
```

第一句回答「為什麼是灰的」，第二句回答「那我要做什麼」。
第二句與 `subtitle_pipeline_handler.go:113` **逐字相同**。

### 卡片 footer 未啟用態（新增，一段）

```
自動處理免費字幕（伺服器未啟用）
```

括號裡指名**誰**沒啟用。少了它會被讀成「你沒勾」——
而使用者明明勾了，那是最糟的誤讀。

### 凍結不動的三句

E5-D 區塊 D/F 的三句（開關標籤 ＋ 兩段說明）**一字不改**。
本頁是**第四種狀態**的新文案，不是既有文案的修訂版。

**停用的是「控制項」，不是「說明」**（2026-08-21 修訂，見下）：

| 元素 | 停用態 fill | 對比（對 `$bg-primary` `#1b2336`） | AA |
|---|---|---|---|
| checkbox | `opacity-60` | —— | 停用元件，1.4.3 豁免 |
| 開關標籤 | `$text-disabled` | **3.55:1** | 停用元件的**可及名稱**，1.4.3 豁免 |
| 兩句說明 | **`$text-secondary`（不降色）** | **7.47:1** | ✅ 通過 |
| 通知列兩句 | `$text-primary` / `$text-secondary` | 14.00 / 7.47 | ✅ 通過 |

### ⚖️ 修訂：兩句說明不降色（2026-08-21，Sally 自我更正）

**初版此處裁定兩句說明也降 `$text-disabled` —— 那是錯的。**

`apps/web/src/styles.css:47` 對該 token 的註解白紙黑字寫著
`intentionally sub-AA (TC-1)` —— 設計系統自己標記它讀不清。
實測 `#6e7891` 對 `#1b2336` 是 **3.55:1**，12px 內文適用 4.5:1 門檻，**不通過**。

而那兩句正是**唯一能說服使用者去改 `VIDO_SUBTITLE_PIPELINE_MODE` 的東西**。
把「你為什麼該開這個功能」印成讀不清的顏色，跟一開始「整段隱藏」犯的是同一個錯，
只是換了個手法：使用者一樣得不到他需要的資訊。

WCAG 1.4.3 的豁免寫得很精確 ——
「**inactive user interface components**」。豁免的是**控制項**，
不是控制項旁邊的說明散文。開關標籤是那個 checkbox 的**可及名稱**，屬於控制項，
降色正確；兩句說明不屬於，不該降。

**為什麼不改用 `$text-muted`（6.71:1，也通過）**：
它與 `$text-secondary`（7.47:1）的視覺差幾乎看不出來 ——
那會是一個「程式碼裡有、眼睛看不到」的區別，在規格裡顯得慎重，在產品裡毫無意義。
整塊「已停用」的訊號由**灰掉的 checkbox ＋ 灰掉的標籤 ＋ 那條 info 通知列**承擔，
已經足夠；不需要再犧牲一段必須讀懂的文字來加強它。

### 四條硬性要求（沿用 J4-D 區塊 F，逐條可指認）

1. ✅ 全文不出現「掃描」二字（sub-4-3 AC #6 / 2026-08-07 誤解的原產地）
2. ✅ 不作金額保證
3. ✅ 不暗示自動化會花錢
4. ✅ 每一句都對應一個使用者可以採取的動作，或一個他需要知道的事實

---

## 交付：新增 spec 畫面 `J5-D`

**不改 E5-D／E5-M。** 控制項的各種狀態放 spec 頁的 specimen，
mockup 只畫主要狀態 —— 這是 J4-D 區塊 D 已經建立的慣例（關閉／開啟兩個 specimen）。

**也不併進 J4-D。** J4-D 回答的是「錢的界線」；本頁回答的是「這個部署有沒有這個能力」。
兩個不同的軸，擠在一起會讓 J4-D 同時回答兩個問題。

| 項目 | 值 |
|---|---|
| 畫面 | `J5-D` · 部署未啟用時的誠實狀態規格（桌面） |
| 位置 | x=21060, y=24300, w=1240（J4-D 右側，J-spec 列對齊 y=24300） |
| Caption | x=21060, y=24270 —— `J5 · 入庫自動生成 · 部署未啟用狀態規格` |
| 區塊 | A 標題／B 三條事實／C 決策矩陣／D Modal specimen／E 卡片 specimen／F 文案理由／G dev 落地備註 |
| 可複用元件 | `Fn5MZ` CheckboxDisabledChecked、`VSXl5` CheckboxDisabledEmpty（**已存在，不需新建**） |

---

## 給 dev 的落地備註（寫進 J5-D 區塊 G）

- **Modal**：`LibraryEditModal.tsx` 現在是 `autoSubtitleSupported && (...)` 整段隱藏（PR #250）
  → 改為**永遠渲染**，不支援時走本頁的停用態。
- **卡片**：`LibraryCard.tsx:115` 需要拿到 `autoSubtitleSupported`
  （目前沒傳進去 —— 卡片的 props 要加，或由呼叫端 `MediaLibraryManager` 供給）。
- **資料來源已經有了**：`GET /api/v1/libraries` 的 `auto_subtitle_supported`，PR #250 已出貨。
  **不需要新 endpoint、不需要新欄位。**
- 停用時 update payload 仍**省略** `autoSubtitle`（PR #250 的既有行為），
  以免清掉使用者在 pipeline 模式下做過的選擇。

---

## 流程

1. ✅ Sally 出裁定與提示詞 ← **本文件 ＋ `-prompt.md`**
2. ✅ Alexyu 跑 Pencil Inline AI Agent
3. ✅ Sally MCP review —— **PASS（一輪）**，修正一處底色（見下）
4. ✅ `SCREENS` dict：`alrIw` → `("flow-j-specs", "j5-d")`
5. ✅ 匯出截圖、與 `.pen` 同 commit（PR **#252** merged 2026-08-21）
6. ✅ 落地 → `9R-10b-m4-unsupported-state-frontend`（PR #255）
7. ⚖️ **修訂 2026-08-21** —— 由該 story 的 CR 帶回：兩句說明的 `$text-disabled` 是 sub-AA。
   Sally 複核後**自我更正**：只有控制項與其標籤降色，說明維持 `$text-secondary`。
   `J5-D` 區塊D 兩個 specimen 的 `說明1`/`說明2` 已改（4 個節點），區塊F 新增理由 ⑥。

### MCP 複驗結果（2026-08-21）

| 檢查 | 結果 |
|---|---|
| 通知列兩句 vs `subtitle_pipeline_handler.go:112-113` | **逐字相同**（行動版斷行處不留空格＝正確，非漏字） |
| `ctx.problems` | **0 筆** |
| 卡片三態 | `$success`／`$warning`／整段不出現，齊全 |
| 元件實例 | `Fn5MZ`・`VSXl5` 為真 ref，非複製品 |
| 觸控目標 | 桌面 21（＝E5-D）／行動 **45** |
| `E5-D`／`E5-M`／`J4-D` | **零改動** |
| 「掃描」逐畫面 | E5-D **0** ／ E5-M **0** ／ J4-D **2** ／ J5-D **1** |

J5-D 那 1 處是區塊 F 稽核清單的**規則本身**，與 J4-D 的 2 處同性質
（規格頁後設敘述，上一輪已審定為正確必要）。使用者真的會看到的 mockup 仍是 0。
⚠️ 自檢腳本的正確規則是「**只掃 mockup 畫面**」，不是「排除區塊 F」。

**複驗修正一處**：J5-D 的內外底色與 J4-D 相反（外深內淺）。兩張 spec 頁並排在同一個 y，
內外反過來會看起來像出錯 → 對調為外淺內深（frame `$bg-secondary`／強調區塊 `$bg-primary`），
三個單屬性 `Update`。
⚠️ **`-prompt.md` 的底色段寫反了** —— 日後複用 J-spec 頁提示詞，
要先讀 J4-D 的實際 fill（frame `#24304A` 外淺、強調區塊 `#1B2336` 內深）。

## 驗收標準

- [ ] `J5-D` 存在，七個區塊齊全
- [ ] 兩句 Modal 文案與 `subtitle_pipeline_handler.go:112-113` 逐字比對相同
- [ ] 卡片三態 specimen（綠／琥珀／不出現）並陳
- [ ] 決策矩陣四格全覆蓋
- [ ] 全畫面「掃描」零命中
- [ ] `E5-D`／`E5-M`／`J4-D` 的 `.pen` diff 為 **0 行**
- [ ] 全樹 `ctx.problems` 零筆
