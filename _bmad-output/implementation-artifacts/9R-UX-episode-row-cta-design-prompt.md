# 9R-UX 分集列字幕入口 —— Pencil Inline Agent 提示詞

> **給 Alexyu**：以下 `---` 之間整段複製進 Pencil 的 Inline AI Agent。
> 跑完後回來跟我說一聲，我用 MCP 逐條 review（story AC #1–#5）。
>
> 節點 id、幾何、文案都是我 MCP 勘查過的實況，agent 不需要自行探索也**不應**改寫。

---

你正在編輯 `ux-design.pen`。這一輪要為「影集分集列」補上逐集字幕入口，並同步兩處過期文案。
**四個任務，全部照做，不要自由發揮。**

## 絕對不要碰

- `t4tTRq` / `v5pjv` / `YdwGD` 三列裡的 `th`（縮圖）、`inf`（標題＋日期）、`st`（`繁中` 狀態藥丸）——
  一個像素都不要動。狀態藥丸與程式碼目前的圖示版本不一致，那是既有議題，**不在這一輪**。
- `ZpQaw`（J2-D 徽章規格頁）整頁不動。
- `r1EY9` 裡除了我下面明講的節點以外，全部不動——尤其 `NS2ae`（生成 CTA 按鈕本體）、
  `kZKPP`（`搜尋線上字幕（成功率低）`）、`tKGGf`（`轉為繁中（簡轉繁）`）。
- 不新增任何 design token、不新增任何 reusable component、不改任何既有元件。

## 這一輪的設計裁定（已定案，照做即可）

**分集列的動作是「開啟管理字幕對話框」，不是「直接生成」，而且十種字幕狀態一律相同、不做分歧。**

理由：J2-D 已經裁定「字形＝是否定局、顏色＝急迫度」，狀態資訊由同列的狀態指示器承載。
若動作也隨狀態改變，等於把同一件事編碼兩次，而且 25 集的清單會因為狀態不同而讓按鈕忽隱忽現。
狀態差異一律留在對話框內處理（F1 已核定「CTA 永遠可按、助語依狀態切換」的模式）。

---

## 任務 A —— `N2fmG6` 分集列加入逐集動作

對 `t4tTRq`、`v5pjv`、`YdwGD` **三列各做一次**：

在該列的 `inf` 與 `st` **之間**插入一個新的子節點（`ep` 是 horizontal flex、gap 14、
`inf` 是 `fill_container`，所以插入後會自動重排，**不要手算座標**）。

節點資料完全比照 `RequestRow` 已核定的純文字按鈕 `yTntT`（零新元件、零新 token）：

```json
{
  "type": "frame",
  "name": "btn-subtitle",
  "height": 44,
  "cornerRadius": "$radius-md",
  "padding": [0, 14],
  "justifyContent": "center",
  "alignItems": "center",
  "children": [
    {
      "type": "text",
      "name": "subtitle-label",
      "fill": "$text-secondary",
      "content": "管理字幕",
      "fontFamily": "Noto Sans TC",
      "fontSize": 13,
      "fontWeight": "normal"
    }
  ]
}
```

插入順序務必是 `th` → `inf` → **`btn-subtitle`** → `st`。

## 任務 B —— `N2fmG6` 補一則行為註記

在 `VNeL9`（`sec-eps`）的**最後**插入一個註記區塊，寬 `fill_container`、
`fill` 用 `$bg-tertiary`、`cornerRadius` 8、`padding` 12、vertical layout、gap 6。
內含四行文字（`Noto Sans TC`，第一行 12px `$text-primary` `600`，其餘 12px `$text-secondary` normal）：

1. `⚙ 分集字幕入口規格（9R-UX 定稿）`
2. `管理字幕：僅在該集有本地檔案時出現；十種 subtitle_status 一律相同樣態，狀態差異全部留在對話框內。`
3. `可及性名稱必須含集號，例如「管理 S01E03 的字幕」——同頁十餘列文案相同，沒有集號無法區辨。`
4. `行動版（<sm）：整列改為上下堆疊（沿用 12-2 的堆疊規則），管理字幕移到第二行、靠左、維持 44px 觸控高度；不另出 TV 行動版畫面。`

## 任務 C —— `r1EY9` 對話框：兩處文案

**C-1** 在既有條件文案規格區塊的 `W3o1f9`（`手機版（F1-M-v2）行為相同，不另出圖`）**之前**，
補兩行同款式的規格行（比照 `q8kia8`／`k4bp6` 的字級與顏色）：

- `助語・untranslated（已有英文 SRT）：僅需翻譯，不再重跑語音辨識——這次很快也很便宜`
- `助語・生成進行中：本集正在生成字幕——開啟即接續顯示進度，不會重複啟動`

**C-2** 在同一個規格區塊最後補一行，記錄影集層級的文案裁定：

- `影集層級（非單集）：CTA 維持不可按，助語改為「請於下方分集清單逐集生成」——原文案「影集字幕生成即將推出」已過期`

## 任務 D —— 新增一張獨立規格畫面

**不要**把這張塞進任何既有 mockup（設計決策說明一律獨立成頁）。

在畫布空白處新增一個 frame，名稱 `J3-D`，寬 1000、`fill` `$bg-secondary`、`cornerRadius` 12、
`padding` 32、vertical layout、gap 16。內容：

- 標題（18px `$text-primary` `600`）：`J3 · 分集列字幕入口規格 — Episode Row Subtitle Entry`
- 副標（13px `$text-secondary`）：`9R-UX 定稿。消費者：9R-10c（前端實作依此頁，不得自行發明）。`
- 裁定框（`$bg-tertiary`、`cornerRadius` 8、`padding` 16、vertical、gap 8）：
  - 14px `$text-primary` `600`：`裁定：動作與狀態解耦`
  - 13px `$text-secondary`：`分集列的動作是「開啟管理字幕對話框」，不是直接生成。十種 subtitle_status 一律呈現相同動作，不做任何分歧。`
  - 13px `$text-secondary`：`理由：J2-D 已定「字形＝是否定局、顏色＝急迫度」，狀態由同列指示器承載。動作再隨狀態改變＝同一事實編碼兩次，且 25 集清單會出現按鈕忽隱忽現的雜訊。`
  - 13px `$text-secondary`：`唯一的閘門是 has_local_file：沒有本地檔案的集不出現任何動作（TMDb 有、NAS 沒有的集不可誤導使用者）。`
- 一張表（表頭 12px `$text-secondary` `600`；內容 12px `$text-primary`／`$text-secondary`），
  欄位：`subtitle_status` ｜ `列上出現動作？` ｜ `對話框開啟後的樣態`。十列：

  | found | 是 | 完成態；重新生成會覆蓋既有繁中字幕 |
  | not_found | 是 | 缺字幕態，可生成 |
  | not_searched | 是 | 缺字幕態，可生成 |
  | searching | 是 | 進度態（線上搜尋中） |
  | probing | 是 | 進度態（偵測字幕軌中） |
  | extracting | 是 | 進度態（抽取內嵌字幕中） |
  | translating | 是 | 進度態（翻譯字幕中） |
  | no_text_source | 是 | 缺字幕態；僅語音辨識可解 |
  | skipped | 是 | 缺字幕態；軌道語言不符已略過 |
  | untranslated | 是 | 缺字幕態＋助語「僅需翻譯，不再重跑語音辨識」 |

- 收尾註記（12px `$text-secondary`）：
  `「一律出現」不是偷懶——是刻意讓入口穩定：使用者不必先讀懂狀態才知道能不能點。可行性判斷在對話框內做，那裡有空間解釋。`

## 完成後

確認所有新增文字沒有與既有內容重疊（標籤與標題不得相互覆蓋），然後**存檔**。

---

## 給 Sally 的 review 清單（Alexyu 不用看）

- AC #1：三列都有 `btn-subtitle`、順序在 `inf` 與 `st` 之間、44px、零新 token
- AC #2：`J3-D` 十列表格齊全且與裁定一致
- AC #3：C-1 兩行 + C-2 一行；`Aodey`／`tO72N` 未被更動（標題規則已存在，不重畫）
- AC #4：任務 B 第 4 行涵蓋行動版
- AC #5：獨立規格頁、無重疊、已存檔
