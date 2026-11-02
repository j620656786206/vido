# Story disc-2026-11-burned-in-subs-c2-ui：前端——「畫面字幕」出現在季清單、管理字幕彈窗，可以改判

Status: done

**Source:** 設計決定 `disc-2026-11-burned-in-subs-c-design.md`（Sally）；後端 -c1 同分支（`feat/burned-in-subs-c1`），**兩張一起上線**——-c1 的來源種類 `burned` 讓舊前端的 `SOURCE_LABEL[s.kind]` 變成呼叫 undefined，季清單會壞。

## 做法（照設計決定 1～3）

1. **季清單**（`components/media/EpisodeList.tsx`）：`SOURCE_LABEL` 加 `burned: () => '畫面字幕'`；`describeSources` 遇到不認得的種類要跳過，不能讓整個清單壞掉（防未來再加種類）。型別 `ChineseSubtitleSource['kind']` 加 `'burned'`。
2. **管理字幕彈窗**（`components/subtitle/ManageSubtitleDialogV2.tsx`）：集數的 `burnedInSubtitle` 是 `zh-Hans`／`zh-Hant`／`zh` 時，「現有字幕」多一列：語言徽章（簡中／繁中／中文，用既有 descriptor）、來源「畫面字幕」、細節「燒在畫面上，不能關掉」。列尾文字按鈕「判斷錯了？」（44px 觸控、`--text-secondary`），點開兩個選項：
   - 「這季沒有畫面字幕」→ `PUT /api/v1/subtitles/burned-in/season` `{series_id, season, value:"none"}`，成功後重新抓季集數（react-query invalidate）。
   - 「重新檢查這季」→ `POST /api/v1/subtitles/burned-in/check` `{series_id, season}`；選項下方小字「會花約 US$0.002」；完成後重新抓。檢查中顯示忙碌狀態；失敗用既有錯誤樣式白話說明。
   - `none` 或沒有值 → 不出現這一列。
3. **生成字幕按鈕下方**（同彈窗）：有畫面字幕時多一行 `--text-muted` 小字「畫面上已經有中文字幕，再生成的字幕會跟它疊在一起」；按鈕照常可按。
4. 前端型別／service 照既有 snake→camel 慣例（`burned_in_subtitle` → `burnedInSubtitle`）。電影沒有這欄，不顯示。
5. 測試：vitest 覆蓋三種值的列、`none` 不出現、兩個選項打對的 API、生成按鈕小字、季清單來源文字含「畫面字幕」、未知種類不會壞。gallery：新增 fixture（新 id）展示有畫面字幕的管理字幕彈窗（桌機＋手機），**不改既有 fixture 的畫面**。

## Acceptance Criteria

1. [x] 季清單來源含「畫面字幕」；未知來源種類跳過不崩。
2. [x] 管理字幕彈窗的畫面字幕列（三種值）、`none` 不出現。
3. [x] 「判斷錯了？」兩個選項＋忙碌／錯誤狀態＋重新抓資料。
4. [x] 生成按鈕下方小字。
5. [x] vitest、typecheck、lint（0 errors）、Prettier；新 gallery fixture；不產生視覺基準。

## Dev Agent Record

### 改了什麼

- **季清單**（`components/media/EpisodeList.tsx`）：`SOURCE_LABEL` 加 `burned: () => '畫面字幕'`；`describeSources` 先確認這個種類有對應文字才用，不認得的種類直接跳過，整個清單不會因為後端多一種來源就壞掉。型別 `ChineseSubtitleSource['kind']` 加 `'burned'`，`MergedEpisode` 加 `burnedInSubtitle`（新型別 `BurnedInSubtitle`）。
- **管理字幕彈窗**：新元件 `components/subtitle/BurnedInSubtitleRow.tsx`（彈窗本身只多接三個 props，不把邏輯塞進 1200 行的檔案）。
  - 列的樣式跟其他「現有字幕」列一樣（徽章・來源・細節），排在最後；徽章用彈窗既有的 `inventoryLangDescriptor`：`zh-Hans`→簡中、`zh-Hant`→繁中、`zh`→中文（走 `zh-unknown` 那條，跟「繁簡未知」的片內字幕同一個灰色徽章）。細節是一句話不是檔名，所以不用等寬字。
  - 「判斷錯了？」是文字按鈕（`--text-secondary`、`min-h-11`＝44px），選單用下載頁已經在用的 Radix `DropdownMenu`（`modal={false}`，理由同下載頁），選項樣式沿用那邊的 `MENU_ITEM`。
  - 「這季沒有畫面字幕」→ `subtitleService.clearBurnedInSeason` → `PUT /subtitles/burned-in/season {value:"none"}`；「重新檢查這季」→ `subtitleService.checkBurnedInSeason` → `POST /subtitles/burned-in/check`，選項下方小字「會花約 US$0.002」。成功後 invalidate `detailKeys.seasonEpisodes(seriesId, season)`——彈窗本來就是從這個 query 讀集數，所以改成沒有之後這一列會自己消失。
  - 進行中：按鈕變「檢查中…」／「更新中…」加轉圈、停用、`aria-busy`。失敗：用彈窗既有的錯誤樣式（`--error-tint` 底、`CircleAlert`、`role="alert"`）寫「沒辦法重新檢查：…」／「沒辦法改成沒有畫面字幕：…」，後面接伺服器的白話原因（例如沒設 Claude 金鑰）；按另一個選項時舊錯誤會清掉。重新檢查回 `result: null`（這次判斷不出來）時，列下面用小字說「這次沒看出結果，先維持原本的判斷」。
  - 只有集數才有；電影就算傳了值也不顯示。`none` 或沒有值不出現這一列。沒有 seriesId／season 時列照樣顯示，但不給「判斷錯了？」（沒有對象可以改）。
  - 這集沒有任何字幕檔、片內也沒字幕，但畫面有中文時，顯示這一列，不再顯示「尚無字幕」；讀字幕清單失敗時，這一列也照樣保留（它不是從那個清單來的）。
- **生成字幕按鈕下方**：有畫面字幕時多一行 `--text-muted` 小字「畫面上已經有中文字幕，再生成的字幕會跟它疊在一起」，自己佔一行；按鈕照常可按，並用 `aria-describedby` 連到這行字。
- **SeasonAccordion**：把集數的 `burnedInSubtitle`、`seriesId`、`seasonNumber` 傳給彈窗。
- **Service**：`subtitleService` 加 `checkBurnedInSeason`／`clearBurnedInSeason` 和 `BurnedInCheckResult` 型別，照既有 `fetchApi`（snake→camel、錯誤帶伺服器訊息）。
- **Gallery**：新增 `subtitle-manage-subtitle-dialog-v2-burned-in`（桌機）與 `…-burned-in-mobile`（390×844），《仙逆》S01E01、簡中燒在畫面上、沒有字幕檔；用自己的集數 id，所有 query 都先塞好，不打後端。既有 fixture 一行都沒改（gallery 檔只有新增）。

### 決定

- 畫面字幕的「判斷錯了？」選單放在列尾，元件獨立成檔；沒有新的顏色、圖示或基本元件。
- 元件檔頭用 `// Design ref: ux-design.pen Screen F1-D-v2 (r1EY9) + Screen F1-M-v2 (JkdfH)`：沒有自己的 .pen 稿（Pencil 轉付費後的做法），它是 F1「現有字幕」區塊的延伸；`check-design-refs.py` 通過。
- ⚠️ **待 Sally／Alexyu 決定**：季清單上 `zh_hans` 的提示文字是「只有簡中字幕，可以在『管理字幕』轉成繁中」，而且不列來源。對《仙逆》這種**只有畫面上燒了簡中**的集數，這句話不對——燒在畫面上的字沒辦法轉。設計決定說圖示與顏色規則不變，所以本張沒改這句；建議另開一張小單決定 `zh_hans` 而來源只有 `burned` 時要說什麼。
- 沒有在瀏覽器實際看畫面（沒有 .pen 稿可對照）；畫面樣式沿用既有元件，gallery 兩個新 fixture 可以拿來看。

### 測試證據

- `BurnedInSubtitleRow.spec.tsx`（6）：選單兩個選項、重新檢查帶價格；「這季沒有畫面字幕」打 PUT（對的 series/season）並重新抓季集數；「重新檢查」打 POST、進行中顯示「檢查中…」且停用、`aria-busy`，完成後重新抓；判斷不出來顯示說明；失敗顯示白話錯誤＋伺服器原因、換選項會清掉；改判失敗說是哪個動作失敗。
- `ManageSubtitleDialogV2.spec.tsx`（+9）：三種值各自的徽章、來源、細節、列在最後；沒字幕時顯示這列而不是「尚無字幕」；`none`／沒有值不出現、也沒有生成鈕小字；電影不顯示；生成鈕可按、小字內容、`aria-describedby`；沒有 season 不給改判；讀字幕清單失敗時這列還在。
- `EpisodeList.spec.tsx`（+4）：只有畫面字幕（繁）、看不出簡繁、旁邊有 zh-TW 檔＋畫面字幕（照 API 順序）；不認得的來源種類跳過、清單照常畫出來。
- `pnpm nx run web:test`：305 個檔案、5027 個測試全過；`pnpm nx run web:typecheck` 通過；`pnpm nx lint web` 0 errors（168 個 warning 都是既有的，改到的檔案單獨跑 eslint 沒有任何訊息）；改到的檔案 Prettier 檢查通過；`python3 scripts/check-design-refs.py` 通過。
- 後端沒有改；同分支 -c1 的 `go build ./... && go test ./...` 重跑一次，44 個套件全 ok。
- 沒有產生任何視覺基準圖；兩個新 fixture 在 CI 會缺 `-linux` 基準，照慣例等 Visual Regression 自動開的 bootstrap PR。

### Opus 補充（2026-11-02）

- 依 dev 提出的待決問題，Sally 決定補進設計檔：`zh_hans` 且中文來源**只有**畫面字幕時，季清單提示改成綠色「有簡中字幕（燒在畫面上）」＋來源，不再顯示紅色「可以在管理字幕轉成繁中」（燒在畫面上不能轉；2A 也算有中文）。`EpisodeList.tsx`＋2 條測試。
- 用 gallery 單一 fixture 模式在瀏覽器實際截圖（桌機 1280、手機 390）：列、徽章、「判斷錯了？」選單兩個選項與費用小字、生成按鈕下方小字都照設計顯示。

## Change Log

- 2026-11-02：建立。
- 2026-11-02：Amelia 實作完成（季清單來源、管理字幕彈窗的畫面字幕列與改判、生成鈕小字、gallery 兩個新 fixture），跟 -c1 同分支送 review。
