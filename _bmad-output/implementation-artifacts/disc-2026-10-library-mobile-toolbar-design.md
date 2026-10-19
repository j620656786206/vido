# disc-2026-10-library-mobile-toolbar-design：手機媒體庫稿補上工具列

- 狀態：done（2026-10-19 Alexyu 跑完 inline agent；Sally 用匯出截圖複審 a2p-m／a3p-m／a4p-m 通過：三張 390×844、工具列 60 高、選取與切換鈕 44 高、tabbar 貼底、A4p-M 亮列表、A3p-M／A2p-M 亮格狀）
- 來源：2026-10-19 b2（#826）dev agent 回報「A4p-M 沒畫排序／選取／切換那一列，app 一直都有」。⚖️ Alexyu 裁定 A：補進設計稿。
- 範圍：只改 `ux-design.pen` 的三張手機稿，不動程式。

## Sally 的裁定

1. **補三張**：A4p-M 列表（`EkB8g`）、A3p-M 格狀（`h1v1U6`）、A2p-M 載入中（`qBWQC`）。app 在這三種狀態都會顯示工具列。A1p-M 空片庫不補（app 空狀態沒有工具列可用的東西）。
2. **位置**：標題列（topbar）正下方。A3p-M 的「已套用篩選」那排（`V81p1`）放在工具列下面，跟 app 一樣（chips 在手機是工具列的最後一行）。
3. **內容，照 app 的順序**：左邊排序鈕；右邊「選取」鈕＋格狀／列表切換。
4. **元件**：排序用既有元件 `Component/SortDropdown`（`955EZ`）；切換鈕複製桌機稿 A3p-D 的 `view-toggle`（`k8FW61`），兩顆按鈕放大成 44×44（b2 已把 app 改成 44）。
5. **選取鈕**：桌機稿目前也沒畫，但功能是真的（批次選取）。手機先畫；桌機稿缺「選取」另記，不在這張。
6. **高度**：整張仍是 390×844。工具列 60 高，下面的列表／格狀改成填滿剩下空間（會被裁切，沒關係）。

## 已知的 app 差異（畫完再決定要不要開實作單）

- app 切換鈕選中時是**實心金底**（`--accent-primary`），設計稿是**淡金底**（`$accent-subtle`）。照設計系統畫淡金；app 要不要改，等稿子定了再看。
- app 排序鈕左邊有 ⇅、右邊是方向箭頭；元件 `955EZ` 只有右邊的 chevron。照元件畫。

## Inline agent 提示詞（Alexyu 在 Pencil 裡貼上執行，跑完按 ⌘S）

```
在 ux-design.pen 改三張手機稿，補上媒體庫工具列。規則：padding／gap 的 0 一律用 $Space/none，不要寫數字 0；每個新節點都要 name；不要動標題列、tabbar、列表內容與畫布上的說明文字。

先做一個工具列（第一張做好後，後兩張用 Copy 複製它）：
- frame，name「toolbar-m」，width fill_container，height 60，fill $bg-primary，padding [$Space/sm, $Space/lg]，gap $Space/sm，alignItems center，layout 水平。
- 子節點依序：
  1. 「sort」：ref 元件 955EZ（Component/SortDropdown），height 44，label 文字維持「新增日期」。
  2. 「spacer」：frame，width fill_container，height 1，無填色。
  3. 「select-btn」：frame，height 44，fill $bg-secondary，cornerRadius $radius-md，padding [$Space/none, $Space/md]，gap $Space/xs-plus，alignItems center；裡面 icon（lucide「square-check」，16×16，fill $text-secondary，name「i」）＋ text「選取」（name「label」，fill $text-secondary，字型用 $Type/Family/Primary、$Type/Body/Size、$Type/Body/Weight、$Type/Body/Line）。
  4. 「view-toggle」：Copy 節點 k8FW61（桌機 A3p-D 的切換鈕），把 grid-btn 和 list-btn 都改成 width 44、height 44（用 Copy 的 descendants 改，不要事後 Update）。

一、A4p-M 列表（EkB8g）：
- 把 toolbar-m 插在 topbar（R55dM）後面，也就是 EkB8g 的第 2 個子節點。
- 這張是列表檢視：view-toggle 裡 list-btn 改 fill $accent-subtle、它的 icon fill $accent-text；grid-btn 改 fill #00000000、它的 icon fill $text-secondary。
- list（NY0LJ）維持 height fill_container。

二、A3p-M 格狀（h1v1U6）：
- Copy 一份 toolbar-m 插在 topbar（jT97n）後面、active-chips（V81p1）前面。
- 這張是格狀檢視：grid-btn 選中（$accent-subtle／icon $accent-text），list-btn 不選中（#00000000／icon $text-secondary）。
- grid（wW2oF）改 height fill_container，clip true。

三、A2p-M 載入中（qBWQC）：
- Copy 一份 toolbar-m 插在 topbar（f7xQ0i）後面。格狀選中，同第二張。
- sk-grid-m（rqVRM）改 height fill_container，clip true。

做完檢查：三張 frame 仍是 390×844；工具列裡沒有節點超出、沒有被截掉；選取鈕和兩顆切換鈕都是 44 高；tabbar 仍貼齊底部。最後對三張各截一張圖。
```

## 跑完之後（Sally／我）

1. MCP 唯讀複審三張（Get＋TakeScreenshot），用 `git hash-object ux-design.pen` 對 `HEAD:` 確認已存檔。
2. `python3 scripts/export-pen-screenshots.py`，只留 `flow-a-browse-v2/a2p-m.png`、`a3p-m.png`、`a4p-m.png` 和兩份 snapshot，其他重繪雜訊 `git checkout --`。
3. 這份檔改 done、`sprint-status.yaml` 加條目，跟 .pen、截圖同一個 PR。

## 複審結果（2026-10-19）

- 通過。格狀／骨架改 clip 後最下面一排海報被切在畫面外，是手機要捲動才看得到的部分，不是壞掉。
- 已知 app 差異（切換鈕實心金底 vs 稿子淡金底；排序鈕的 ⇅／方向箭頭）不在這張，等 Alexyu 決定要不要開實作單。
