# UX Decision Request — `EpisodeList` 上 `skipped` 與 `not_searched` 的視覺碰撞

**For:** Sally（ux-designer） · **From:** Amelia（dev） · **Raised:** 2026-08-04
**Origin:** sub-1-7b 出貨前自審發現；Alexyu 裁定此類問題須由 Sally 決策，不得由 dev 自選或以 backlog 帶過。
**Tracking:** `sprint-status.yaml` → `backlog-episodelist-skipped-vs-not-searched-glyph: blocked`（⛔ awaiting Sally）
**Affects:** `apps/web/src/components/media/EpisodeList.tsx`（已 merge，#194）· `ux-design.pen` 規格畫面 `J2-D`（node `ZpQaw`）

---

## 問題

劇集列的字幕狀態圖示上，**兩個語意完全不同的狀態渲染成同一個畫面**：

| 狀態 | 圖示 | 顏色 | 使用者看到 |
|---|---|---|---|
| `not_searched` | `Minus` | `--text-muted` `#a0aabe` | 一條灰色短橫 |
| `skipped` | `Minus` | `--text-muted` `#a0aabe` | **一模一樣的灰色短橫** |

唯一的差別在無障礙名稱／tooltip：`尚未搜尋字幕` vs `已略過（字幕軌語言不符）`。**滑鼠停留才看得出差別；掃視列表看不出來。**

這正面牴觸 sub-1-7b 自己的 story statement：

> 「…讓**被管線永久拒絕**的項目，能**視覺上**區分於**只是還沒輪到**的項目。」

## 為什麼這是設計決策而不是實作問題

`J2-D` 規格畫面（sub-1-7a AC #3）明訂 `skipped` → `Minus` + `--text-muted`，而 `not_searched` 的 `Minus` + `--text-muted` 是 story 12-2 就有的既有值。規格沒有算到這個碰撞。要解就得**選一個新字形**，而「哪個字形讀起來像『刻意略過』而不是『壞掉』或『還沒做』」正是設計判斷。

## 兩個狀態的語意差異（決策依據）

| | `not_searched` | `skipped` |
|---|---|---|
| 發生了什麼 | 管線還沒處理到這個檔案 | 管線**處理過了**，並依規則**主動拒絕** |
| 為什麼 | 排隊中／尚未掃描 | 字幕軌語言不是英文（P0：`und` 永不視為英文） |
| 會變嗎 | **會** —— 下次掃描就會處理 | **不會** —— 除非換檔案或手動觸發 |
| 使用者該做什麼 | 什麼都不用做，等就好 | 知道這檔案不會自動有字幕；想要的話得手動處理 |
| 是不是壞掉 | 不是 | **不是**（這是正確行為，不可讀成錯誤） |

## 既有設計約束（請在這些之內裁決）

- **色票**：只能用 `libraryStatus.ts:30-37` 既有六個 tint／語意色，**不新增色票**。
- **accent 保留給進行中狀態**（Sally 2026-07-05 裁決）—— 終局狀態不可用 accent。
- **不可用 error 紅**：`skipped` 沒有出錯，用紅色會讓使用者對正確行為開 bug（sub-1-7a AC #4 已裁定「已略過必須讀起來像刻意」）。
- **圖示詞彙**：目前用 `lucide-react` 的 `CheckCircle2` / `XCircle` / `Loader2` / `Minus`。
- 同一張表上 `no_text_source` 已經用掉 `XCircle` + `--text-muted`（與 `not_found` 的 `XCircle` + `--error` 靠顏色區分，且 aria-label 不同，資訊未僅靠顏色傳達）。

## 候選方案（不是推薦，是選項）

| # | 方案 | 讀感 | 代價 |
|---|---|---|---|
| A | `skipped` 改用 `CircleSlash` | 「被規則排除」—— 有邊界感，不像錯誤 | 與 `XCircle`（`no_text_source`）形狀相近，小尺寸下可能難分 |
| B | `skipped` 改用 `Ban` | 「不做這個」—— 意圖明確 | 偏否定／禁止，可能讀成「被封鎖」 |
| C | `skipped` 改用 `SkipForward` | 「跳過，往下一個」—— 最貼近字面 | 播放控制的語彙，在字幕狀態欄可能誤讀成播放操作 |
| D | 保留 `Minus`，改用不同灰階或加虛線／半透明處理 | 維持同一家族、僅降低權重 | 兩個都是灰，區分度可能仍不足 |
| E | 判定「不需要區分」 | 兩者都是「這裡沒有字幕、你不用動作」，tooltip 足矣 | 需明確推翻 story statement 的那句話，並記錄理由 |

**E 是正當選項。** 如果你認為在劇集列這個次要介面上，兩者的使用者行動相同（都是「不用做什麼」），而海報徽章已經區分了（`已略過` vs `缺字幕`/`有字幕`/無徽章），那就裁定 E，把 story statement 的適用範圍限縮在海報／列表徽章即可 —— 但請把這個判斷寫進 `J2-D`，否則下一個讀到 story statement 的人會再提一次。

## 裁定後的落地成本

- **`J2-D`（`ux-design.pen` node `ZpQaw`）**：更新 J2-2 表格那一列 + 樣本圖示；同一次 Pencil session 順便修 `backlog-pen-j2-accent-token-correction`（規格寫 `--accent-primary`，程式實際用 `--accent-text`，理由是 3.04:1 卡在 WCAG 1.4.11 門檻）。
- **程式**：`EpisodeList.tsx` 的 `SUBTITLE_STATUS` map 改一行 + 一個測試（現有測試已斷言 icon／顏色，改動會立刻紅）。
- **截圖**：重跑 `export-pen-screenshots.py`，只 stage `j2-d.png`。

方案 E 則只需 `J2-D` 上補一段說明，程式零改動。

---

# ✅ RULING — Sally（ux-designer），2026-08-04

## 我看到的不是圖示碰撞，是分類錯誤

Amelia 問的是「該給 `skipped` 哪個字形」。但把整組圖示攤開來看，會發現這套圖示裡藏著一條**沒人寫下來的規則**，而它一直在正確運作：

| 圖示 | 有沒有圓框 | 狀態 | 這是什麼 |
|---|---|---|---|
| `CheckCircle2` | **有** | `found` | 一個**定局** |
| `XCircle` | **有** | `not_found` · `no_text_source` | 一個**定局** |
| `Minus` | 無 | `not_searched` | **還不是**結果 |
| `Loader2` | 無 | 進行中四態 | **還不是**結果 |

**圓框＝這件事有結論了；沒圓框＝還沒到結論。** 這條規則從 story 12-2 就在那裡，只是從來沒被寫下來。

`skipped` 是一個**定局** —— 管線看過這個檔案、依規則拒絕了，除非換檔案否則不會再變。但它拿到的是 `Minus`，一個「還沒到結論」的字形。它不是「跟 `not_searched` 撞在一起」，它是**被歸錯了家族**，然後才撞上。

這也解釋了為什麼碰撞讓人不舒服卻說不出所以然：使用者的眼睛其實已經在讀那條規則了。

## 裁定

### ① `skipped` → `CircleSlash` + `--text-muted`

```
skipped: { Icon: CircleSlash, color: 'text-[var(--text-muted)]', label: '已略過（字幕軌語言不符）' },
```

- **圓框** → 回到「定局」家族，這是修正的核心。
- **斜槓** → 「依規則不適用／被排除」，正是 `skipped` 的語意（P0：`und` 永不視為英文）。不是叉、不是驚嘆號，沒有「壞掉」的味道。
- **`--text-muted`** → 不是錯誤，不需要使用者做任何事。**不可用 error 紅**（sub-1-7a AC #4 已裁定「已略過必須讀起來像刻意」）。

### ② `not_searched` 維持裸 `Minus` —— 不要動它

它是整組裡唯一「還沒到結論」的靜態狀態，裸字形正是它該有的樣子。碰撞的錯不在它身上。

### ③ `no_text_source` 維持 `XCircle` + `--text-muted`

**我知道它和 `skipped` 會長得有點像 —— 那是對的，不是缺陷。**

兩者都是「這個檔案不會自動有字幕了」，使用者的下一步完全相同（在 M1 裡：沒有下一步）。它們**應該**讀起來像同一個家族。真正必須一眼分開的是**定局 vs 還沒輪到**，那個現在分開了。至於「為什麼是定局」的細節，本來就是 tooltip 的工作 —— 那也是為什麼 sub-1-7b 把長版說明放進無障礙名稱是對的。

Amelia 把「與 `XCircle` 形狀相近」列為 A 案的代價。我把它改判為 A 案的**優點**。

### ④ 兩個終局狀態不合併 —— 海報徽章已經區分了

海報／列表徽章顯示 `無字幕源` 與 `已略過` 兩個不同標籤。如果劇集列把它們合併成同一個圖示，兩個介面就會對同一個檔案講不同的故事。跨介面一致優先。

### ⑤ 追認 Amelia 的 `--accent-text` 偏離 —— 她是對的，規格是錯的

`--accent-primary` `#3b82f6` 在 `--bg-tertiary` 上量到 **3.04:1**，WCAG 1.4.11 非文字門檻是 3.0。那不叫通過，那叫剛好沒掉下去 —— 而劇集列不帶自身背景，任何 hover／抬升表面就會讓它掉下去。`--accent-text` `#60a5fa` 最差 4.40:1，而且它的 token 註解本來就寫著 AA-safe 前景色。

**規格照實作改，不是實作照規格改。** `backlog-pen-j2-accent-token-correction` 併入同一次 `.pen` 修改。

## 要寫進 `J2-D` 的那條原則（這才是真正的交付物）

字形選擇會被忘記，原則不會。請在 `J2-D` 上新增一節：

> **圖示語法：圓框＝定局，裸字形＝尚未有結論。**
> `CheckCircle2`／`XCircle`／`CircleSlash` 都有圓框 —— 管線對這個檔案已經有答案。
> `Minus`／`Loader2` 沒有圓框 —— 還沒有答案（還沒輪到／正在做）。
> **顏色是急迫度，不是結果**：紅＝使用者可以做點什麼（重新搜尋）；灰＝沒有可做的事；綠＝成功；accent＝進行中（accent 保留給進行中，Sally 2026-07-05）。
> 新增任何狀態前，先回答這兩個問題：它是定局嗎？使用者能做什麼嗎？答案就決定了字形與顏色。

## 落地成本

| 項目 | 變更 |
|---|---|
| `ux-design.pen` `J2-D`（node `ZpQaw`） | J2-2 表格 `skipped` 那列改 `CircleSlash`＋換樣本圖示；三個進行中圖示的 token 由 `--accent-primary` 改為 `--accent-text`；新增上方「圖示語法」原則段落 |
| `EpisodeList.tsx` | import 加 `CircleSlash`；map 改一行 |
| `EpisodeList.spec.tsx` | 既有測試已斷言 icon／顏色，改動會立刻紅 —— 補一個「`skipped` 與 `not_searched` 不得渲染成相同字形」的回歸測試 |
| 截圖 | 重跑匯出，只 stage `j2-d.png` |

---

## 裁定補充 — 畫布位置（Sally，2026-08-04，Pencil 落地回報後）

改動 C 讓 J2-D 由 2435 長到 2907，撞上下方 P2-A 區塊。Pencil agent 只把 J2-D 自己上移 400px 避開，並正確地拒絕在未授權下動 J1-D，把列對齊的取捨交上來。

**裁定：整列一起移，J1-D 與 J2-D 的 y 都設為 24300。**

- **理由一：移動 frame 的 y 不影響匯出的 PNG。** `get_screenshot` 抓的是節點本身而非畫布區域，所以重新對齊是零風險、零截圖 diff 的整理動作 —— 沒有理由為了「少動一個節點」而留下參差的列。
- **理由二：參差會複利。** flow-J 之後還會長出 j3、j4。若每張新規格單各自上移避讓，這一列會退化成階梯，而 spec 畫面正是最需要「掃一眼就知道有幾張」的地方。
- **理由三：49px 淨空太薄。** 只移 J2-D 的方案下方僅剩 49px；兩張都放到 24300 後，J2-D 底部 27207、距 P2-A 的 27456 有 **249px**，上方距 I1-D 底部 23370 仍有 930px。

**追認 Pencil agent 的一處自主判斷：** 顏色圖例改用畫面既有的 10px circle 色票取代 🔴⚪🟢🔵 emoji。理由（emoji 在 Pencil 會被 text fill 統一上色，四色會塌成同一色）是對的 —— 一個會讓自己失效的色彩圖例比沒有更糟。文字一字未改，正確。

**沿革教訓：** spec 畫面是變動高度的長條，不吃 flow 區塊的 2600 間距。加內容後必須檢查與下一個區塊的淨空，且同列一起移。已記入 `project_pen_flow_layout_convention`。
