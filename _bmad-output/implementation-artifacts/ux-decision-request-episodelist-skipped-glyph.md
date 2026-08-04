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
