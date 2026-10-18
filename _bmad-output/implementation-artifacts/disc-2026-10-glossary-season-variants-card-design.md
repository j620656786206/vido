# Disc：名詞對照表「各季寫法不同」改成一張卡片（設計稿）

Status: ready-for-dev

**Source:** 2026-10-18 Alexyu 在正式機看《末日光明》名詞對照表：「人名有多個翻譯，比如 Baba 就有三種翻譯，Baba Voss 也有多個翻譯。」查證後確認這是 `disc-2026-10-glossary-season-scope-b` 刻意做的：官方字幕第 1 季寫「巴巴禾斯」、第 2 季寫「巴霸沃斯」，Vido 每季照官方。面板卻把全劇、S1、S2 攤成三列，看起來像重複。

⚖️ **2026-10-18 Alexyu 裁定：選 1「每季照官方」。** 面板改成「一個原文一張卡片」：大字是全劇預設譯名；只有某一季官方寫法**不一樣**時，卡片下方才多一行「↳ 第 N 季官方寫法：○○」。（Alexyu 看過 ASCII 草圖後回「可以」。）

## 範圍（只動 `.pen`）

1. 新元件 `Component/GlossaryRow-v2/Seasons`（桌面）與 `Component/GlossaryRow-v2/Seasons/Mobile`（手機）。
2. 新稿 `F6s-D-v2 · 名詞對照表 — 各季寫法不同`（桌面）與 `F6s-M-v2`（手機），內容用《末日光明》的真實資料。
3. 獨立的規格說明稿 `F6s Note`（不塞進畫面裡，照 [[feedback-pencil-spec-screens-stand-alone]]）。
4. `SCREENS` 加三筆、重出截圖。

## 設計規則（Sally 定稿）

- **卡片**＝原本的 GlossaryRow-v2 第一行不變（原文 → 全劇譯名、來源徽章、未確認、確認／編輯／刪除），下面接 0～N 行季別子列。
- **季別子列**：只在「該季官方寫法 ≠ 全劇譯名」時出現；跟全劇一樣的季不列。多季時照季數排序。
- 子列內容：`↳`（corner-down-right）＋「第 N 季官方寫法：」（次要字色）＋譯名（主要字色）＋右側「編輯」「刪除」。**子列沒有「確認」，也沒有徽章**——確認是整張卡片一起確認。
- 頁尾「共 N 條」算**卡片數**（原文數），不是列數。
- 刪除子列的確認文字沿用 scope-b：「這是第 N 季官方字幕的寫法。刪掉後，補第 N 季的集數會改用全劇的寫法「○○」。」
- 原本的 `S1`／`S2` 藥丸徽章退場。

## 不做

- 程式實作：另開 `disc-2026-10-glossary-season-variants-card`（blocked-by 本單）。

## Acceptance Criteria

1. `.pen` 有兩個新元件、兩張新稿、一張說明稿，命名照畫布規範，位置不跟其他稿重疊。
2. 桌面稿：《末日光明》5 張卡片，其中 3 張有第 1 季子列（Baba、Baba Voss、Jerlamarel），Haniwa、Alfre Woodard 沒有子列。
3. 手機稿：同一組資料，卡片在 358 寬內不溢出。
4. Sally 用 MCP 唯讀複審：文案逐字對、`ctx.problems` 為空、全部用變數不寫死數值。
5. 截圖重出，只 stage 新增的三張；`check-design-refs.py` 與 token 快照通過。

## Inline Agent 提示詞（Sally，2026-10-18）

把下面整段貼給 Pencil 的 Inline AI Agent。節點 id 都是 2026-10-18 用 MCP 讀出來的。

```text
在 ux-design.pen 做「名詞對照表 — 各季寫法不同」。全部用既有變數，不寫死顏色、字級、間距（0 用 $Space/none）。

【A. 新元件 Component/GlossaryRow-v2/Seasons（桌面）】
1. 複製元件 nDSEd（Component/GlossaryRow-v2），放在元件 t28C2s 右邊的空位，改名 Component/GlossaryRow-v2/Seasons，設為 reusable。
2. 版面改成 vertical，寬 720，gap $Space/none；fill／stroke／cornerRadius／padding 跟 nDSEd 一樣。
3. 把原本 15 個子節點（gr-src … gr-act-delete）包進一個新 frame「gs-main」：horizontal、width fill_container、height 44、alignItems center、gap $Space/md。內容改成：gr-src「Baba Voss」、gr-zh「巴霸沃斯」、gr-badge-lb「官方字幕」、保留未確認＋確認＋編輯＋刪除。
4. gs-main 下面加 frame「gs-season」：horizontal、width fill_container、height 44、alignItems center、gap $Space/sm、padding [$Space/none, $Space/none, $Space/none, $Space/lg]。子節點依序：
   - icon「gs-season-icon」：lucide corner-down-right，14×14，fill $text-muted
   - text「gs-season-label」：「第 1 季官方寫法：」，$Type/Family/Primary、$Type/Body/Size、$Type/Body/Line，fill $text-secondary
   - text「gs-season-zh」：「巴巴禾斯」，樣式同 gr-zh（fill $text-primary）
   - frame「gs-season-spacer」：width fill_container、height 1
   - 複製 gr-act-edit、gr-act-delete 各一個，改名 gs-act-edit、gs-act-delete（文字「編輯」「刪除」，樣式不變）
   gs-season 沒有徽章、沒有「確認」。

【B. 新元件 Component/GlossaryRow-v2/Seasons/Mobile（手機）】
1. 複製 t28C2s（Component/GlossaryRow-v2/Mobile），放在 A 元件右邊，改名 Component/GlossaryRow-v2/Seasons/Mobile，reusable。
2. 內容：gm-src「Baba Voss」、gm-zh「巴霸沃斯」、gm-badge 文字「官方字幕」。
3. 在 gm-line2 下面加 frame「gsm-season」：horizontal、width fill_container、height 44、alignItems center、gap $Space/sm、padding 同 A-4。子節點同 A-4（icon、「第 1 季官方寫法：」、「巴巴禾斯」、spacer、編輯、刪除），編輯／刪除複製自 pV4nI 旁的 I6r5j、hMoYD。
4. 整個元件寬 358，不得溢出。

【C. 桌面稿 F6s-D-v2】
1. 複製畫面 dlfMR（F6-D-v2 · 名詞對照表）到 x=62868, y=31353，改名「F6s-D-v2 · 名詞對照表 — 各季寫法不同」。在 x=62868, y=31308 加 caption 文字，樣式複製 caption JGTcS，內容「F6s-D-v2 · 名詞對照表 — 各季寫法不同」。
2. dialog-title 改「名詞對照表 — 末日光明」。
3. glossary-list 清空，依序放 5 張卡片，gap 8：
   ① GlossaryRow-v2（nDSEd 實例）：Alfre Woodard → 阿爾法·伍達德，徽章「中繼資料」，有未確認＋確認
   ② Seasons 實例：Baba → 巴霸，「官方字幕」，未確認；子列「第 1 季官方寫法：」「巴巴」
   ③ Seasons 實例：Baba Voss → 巴霸沃斯，「官方字幕」，未確認；子列「巴巴禾斯」
   ④ GlossaryRow-v2 實例：Haniwa → 哈妮娃，「官方字幕」，已確認（隱藏 gr-unconf 與 gr-act-confirm）
   ⑤ Seasons 實例：Jerlamarel → 傑拉馬瑞，「官方字幕」，未確認；子列「謝拉馬威」
   實例寬度都 fill_container（832）。
4. 頁尾：count-total「5」、count-unconfirmed「4」。
5. 對話框高度跟著內容長（約 642），垂直置中於 900 高的畫面（y 約 129）。不得溢出、不得被裁切。

【D. 手機稿 F6s-M-v2】
1. 複製畫面 buepS（F6-M-v2 · 名詞對照表（手機））到 x=22920, y=32693，改名「F6s-M-v2 · 名詞對照表 — 各季寫法不同（手機）」。caption 複製 f8u6l 放在 x=22920, y=32648，內容同畫面名。
2. sheet-title 改「名詞對照表 — 末日光明」。
3. glossary-list 清空，放 3 張，gap 8：Seasons/Mobile（Baba → 巴霸／子列 巴巴）、Seasons/Mobile（Baba Voss → 巴霸沃斯／子列 巴巴禾斯）、GlossaryRow-v2/Mobile（Haniwa → 哈妮娃，官方字幕，已確認：隱藏未確認與確認）。
4. 清單、按鈕、匯出匯入列都不得超出 844 高的畫面。

【E. 說明稿 F6s Note】
在 x=64408, y=31353 放一個 360 寬的說明 frame，樣式複製 PLdXp（F6c-D Note），名稱「F6s Note」，內容（逐字）：
「各季寫法不同（⚖️ 2026-10-18）
① 一個原文一張卡片。大字是全劇預設譯名。
② 只有某一季官方字幕寫法跟全劇不同，才多一行「↳ 第 N 季官方寫法：○○」；一樣的季不列，多季照季數排。
③ 子列沒有徽章、沒有「確認」：確認是整張卡片一起確認（連季別寫法）。
④ 子列「編輯」只改那一季；「刪除」後那一季改用全劇寫法，確認文字：這是第 N 季官方字幕的寫法。刪掉後，補第 N 季的集數會改用全劇的寫法「○○」。
⑤ 卡片的「刪除」＝整張刪（含各季）。
⑥ 頁尾「共 N 條」數卡片，不數列。
⑦ 原本的 S1／S2 藥丸徽章退場。」

做完存檔（⌘S）。
```
