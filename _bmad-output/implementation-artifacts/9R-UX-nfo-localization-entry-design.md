# Story 9R-UX: .nfo 在地化的入口與覆寫確認（design）

Status: done  <!-- Sally 2026-08-21：五段提示詞全數執行完畢，MCP 唯讀 review **PASS**（逐字比對 23 個文字節點零改動、全樹零裁切）。截圖已產出並只 stage 真變更的 4 張。 -->

**Epic:** epic-9R-subtitle-route-c · **Risk: 🟡 UX/DESIGN-ONLY（Sally ux-designer，NOT dev）· 零程式碼 —— 但文案要讓人敢按「覆寫我的檔案」**
**Created:** 2026-08-21（SM Bob, create-story）
**Source:** Rule 24 lane ③ `backlog-nfo-localization-frontend-entry`（由 `9R-13a-tv-nfo-localization` dev-story 立案）。
⚖️ **依 Alexyu 2026-08-20 於 `9R-10b` 的裁定先例**：不走 `// Design ref: PENDING` 暫掛，**設計先行**。
**Depends on:** nothing（只需 Pencil.app 執行中）。
**Blocks:** `9R-13b-nfo-localization-frontend`（硬阻斷 —— 該 story Status = `blocked`，本 story `done` 且經 Sally MCP review 後才轉 `ready-for-dev`）。

---

## Story

As Alexyu（庫裡都是英文中繼資料、播放器上一片英文的 NAS 擁有者），
I want 在媒體詳情頁有一個看得懂的入口，把這部片／這部劇的中繼資料翻成繁中，
so that 我按得下去 —— 而且**影集那次我知道自己正在覆寫一個檔案，也知道原檔去了哪裡**。

---

## 🚨 為什麼這張要先做：後端已經出貨兩輪，使用者一次都沒碰到

| 出貨 | 後端 | 前端 |
|---|---|---|
| 9R-13（2026-07） | `POST /movies/:id/localize-nfo` | ❌ 零呼叫者 |
| 9R-13a（2026-08-21） | `POST /series/:id/localize-nfo[?include_episodes]`<br>`POST /episodes/:id/localize-nfo` | ❌ 零呼叫者 |

**實證**：`grep -rn "localize-nfo" apps/web/src` → **空**。
三條路由**只能用 curl 打**。ADR 自封的「category-level differentiator」，使用者從來沒看過。

---

## 🔎 Findings（2026-08-21 create-story 逐檔驗證 —— 設計前必讀）

### 1. 🔴 **不要設計到 v1 詳情頁 —— 那是死程式碼**

`MediaDetailPanel.tsx` + `DetailPanelMenu.tsx`（`// Design ref: Screen 4c Detail Panel Context Menu (7mdTJ)`，
三個項目：重新解析／匯出／刪除）**沒有任何正式路由在用**。

實證：`src/routes/media/$type.$id.tsx:77` **無旗標、無條件**渲染 `LocalDetailV2`：
```tsx
return <LocalDetailV2 type={type} id={id} />;
```
`grep -rn "MediaDetailPanel" src/` 只命中它自己、`LocalDetailV2` 裡的一行註解、以及 gallery fixture。

⚠️ **精確一點**：它並非「無人引用」，而是**無正式路由引用** —— 仍被 (a) `src/routes/test/-gallery.fixtures.tsx` 的元件展示廊、(b) `src/routes/media/-$type.$id.spec.tsx` 這支測試引用。兩者的檔名都以 `-` 開頭，在 TanStack Router 中**被排除在路由之外**（前者是 dev-only 的 `/test` 展示頁）。⇒ 它會出現在展示廊裡看起來「活著」，但**真實使用者永遠走不到**。

⇒ **入口的家是 `LocalDetailV2`**，設計錨點是 `Component/Detail-Movie-v2 (uRGu2)` 與 `Component/Detail-TV-v2 (N2fmG6)`
（截圖：`_bmad-output/screenshots/flow-b-detail-v2/`）。往 `Screen 4c` 加項目＝把設計加到不會被渲染的地方。

### 2. v2 的動作列現在**剛好三個**，加第四個是設計決策

`LocalDetailV2.tsx:147-184` 的 `actions`：

| 順序 | 動作 | 樣式 | 條件 |
|---|---|---|---|
| 1 | **管理字幕** | primary（`--accent-primary`） | 僅 `filePath` 存在 |
| 2 | 修改資訊 | secondary（`--bg-secondary`） | 恆顯示 |
| 3 | 複製檔案路徑 | icon-only 44×44 | 僅 `filePath` 存在 |

**沒有 overflow 選單**（v1 的 `⋮` 沒有被帶進 v2）。

⚖️ **Sally 要裁定**：第四個動作怎麼放？
- (a) 再加一顆 secondary 按鈕 → 動作列變四個，手機寬度會不會擠？
- (b) 引入 overflow `⋮`（把「複製路徑」「在地化」收進去）→ 但那是把 v1 的語彙帶回來，需要 v2 的設計語言。
- (c) 收進「**修改資訊**」（Metadata Editor）→ 語意上最貼近（都是改中繼資料），但那是個 modal，入口更深。

### 3. 🔴 電影和影集是**兩種不同風險等級的同一個按鈕**

| | 電影 | 影集 |
|---|---|---|
| 檔名槽位 | **兩個**（`<basename>.nfo` + `movie.nfo`） | **一個**（`tvshow.nfo`／`<集名>.nfo`） |
| 寫入語意 | 有空槽 → **additive，不動原檔** | **一定覆寫**（原檔先備份 `.nfo.orig`） |
| 後端要求 | 不需確認 | 🔴 **強制 `confirm_replace: true`**，缺席回 **409 `NFO_REPLACE_NOT_CONFIRMED`** |

⚖️ **Sally 要裁定**：同一個入口在兩種媒體型別上要不要**長得不一樣**？
- 文案要不要不同（電影「補上繁中資訊」vs 影集「覆寫成繁中資訊」）？
- 影集要不要有警示色／圖示？
- 還是入口一致、差異全部收在確認對話框裡？

**紅線**：使用者按下去之前**必須知道影集那次會覆寫檔案**。這是產品承諾，不是實作細節。

### 4. 影集還有第二個選擇：**只做劇集檔 vs 整劇每一集**

後端 `?include_episodes` 決定要不要連每一集的 `.nfo` 一起做。

⚖️ **Sally 要裁定**：這個選擇放哪？
- 確認對話框裡的一個 checkbox（「連同 24 集的集名與劇情」）？
- 兩顆不同的按鈕？
- 預設值是什麼？（勾／不勾 —— 一集一次 LLM 呼叫，24 集就是 24 次翻譯費）

⚠️ **成本相關**：這會花錢。文案不能讓使用者以為是免費的本地操作。

### 5. 可抄的確認對話框語彙

- `src/components/ui/Dialog.tsx`（v2 通用 Dialog）
- `src/components/library/BatchConfirmDialog.tsx`（既有的「批次操作確認」先例）
- ⚠️ `DetailPanelMenu` 的刪除確認是 **inline**（不是 modal）—— 但那整個元件是死程式碼，**不要當先例**。

### 6. 結果要怎麼給回饋？

後端回傳：
- 單一：`{path, backup_path, replaced}` —— `replaced=true` 時 `backup_path` 非空
- 整劇：`{show, episodes[], succeeded, failed, skipped}` —— `skipped` = 沒有檔案路徑的集數

⚖️ **Sally 要裁定**：toast？inline 狀態？整劇要不要進度條？
`skipped` 這個概念要怎麼對使用者講（「有 3 集在資料庫裡但硬碟上找不到檔案」）？

### 7. Rule 21 是 ESLint 強制的

`local/implements-pen-node-id` 會在 `pnpm lint:all` ⇒ CI 擋下沒有 `// Implements:` / `// Design ref:` 標頭的
`apps/web/src/components/**` 檔案。⇒ **沒有 .pen 節點就沒有合法的 FE 檔案**（除非用 coverage-gap 逃生門，
而那正是本裁定要避免的）。

---

## Acceptance Criteria

### AC #1 — 入口位置定案
- 在 `Component/Detail-Movie-v2 (uRGu2)` 與 `Component/Detail-TV-v2 (N2fmG6)` 上定出在地化入口的**確切位置與樣式**（Findings #2 的 a／b／c 擇一並寫下理由）。
- 手機寬度（`-m` 畫面）必須一併驗證 —— 動作列是最容易在窄螢幕爆掉的地方。

### AC #2 — 電影 vs 影集的差異裁定
- 明文裁定兩種媒體型別的入口**是否**在文案／樣式上不同（Findings #3）。
- 若相同，必須說明為什麼「覆寫」的風險在確認對話框裡講就夠。

### AC #3 — 覆寫確認對話框（影集專用）
- 新增一個 spec 畫面，定稿**逐字文案**，至少涵蓋：
  - 這會**覆寫**既有的 `tvshow.nfo`
  - 原檔會備份成 `tvshow.nfo.orig`（且**再執行也不會覆蓋那份備份**）
  - 這會**花錢**（LLM 翻譯）
  - `include_episodes` 的選擇（Findings #4）
- 主要按鈕文案不得是含糊的「確定」——要說出它要做什麼。

### AC #4 — 結果回饋
- 定出成功／部分成功／失敗的呈現方式（Findings #6），含整劇的 `succeeded / failed / skipped` 三個數字要不要露、怎麼講。

### AC #5 — 交付形式（依 `.pen` inline-agent 協作模式）
- Sally **不直接** `mcp__pencil__execute` 寫入。
- 產出**節點錨定的提示詞檔** `9R-UX-nfo-localization-entry-design-prompt.md`（含確切 node ID、定稿字串、樣式規格、**自包含**——inline agent 沒有對話上下文）。
- Alexyu 執行 Inline AI Agent → ⌘S 存檔 → `python3 scripts/export-pen-screenshots.py` → **只 stage 真變更的 PNG** → commit。
- Sally 以 Pencil MCP **唯讀 review**（`Get` 逐字比對文案、`ctx.bounds`／`ctx.problems` 驗對齊與溢出）後追認或退回。
- 若新增畫面：更新 `scripts/export-pen-screenshots.py` 的 `SCREENS` dict（flow 資料夾 = `flow-b-detail-v2`，spec 畫面可放 `flow-j-specs`）。

### AC #6 — 範圍紅線
- ❌ **不寫任何程式碼**（本 story 零 `apps/` 變更）。
- ❌ **不動 v1**（`Screen 4c` / `MediaDetailPanel` / `DetailPanelMenu`）—— 死程式碼。
- ❌ **不設計後端沒有的能力**（例如「還原備份」按鈕 —— 後端沒有這個端點；要就另立 story）。
- ❌ **不改後端契約**（`confirm_replace` / 409 碼 / 回傳形狀已出貨）。
- 📌 spec 類標註依 `feedback_pencil_spec_standalone_screen`：**獨立畫面，不塞進既有 mockup**。

---

## Tasks / Subtasks

- [x] **Task 1（AC #1 / #2）** —— 讀 `LocalDetailV2.tsx:147-184` 與 v2 詳情設計節點，裁定入口位置與電影/影集差異
- [x] **Task 2（AC #3）** —— 覆寫確認對話框：畫面 + 逐字文案定稿（含成本與備份說明、`include_episodes` 選擇與預設值）
- [x] **Task 3（AC #4）** —— 結果回饋的呈現（單一 / 整劇三數字）
- [x] **Task 4（AC #5）** —— 產出自包含的節點錨定提示詞檔，交 Alexyu 執行
- [x] **Task 5（AC #5）** —— Alexyu 回報完成後，Sally MCP 唯讀 review：逐字比對 + `ctx.bounds`/`ctx.problems`
- [x] **Task 6** —— 更新 `SCREENS` dict（若有新畫面）、確認截圖已產出且只 stage 真變更

---

## Dev Notes

### 給 Sally 的定位

這張的難點**不是畫面**，是**兩句話**：
1. 讓使用者知道影集那次是**覆寫**，而且原檔沒有消失。
2. 讓使用者知道這**會花錢**（LLM 翻譯），而且整劇會花更多。

兩句都寫得太輕 → 使用者事後發現檔案被改，信任沒了。
兩句都寫得太重 → 沒人敢按，這個 differentiator 繼續零使用者（現在已經這樣兩個月了）。

### 後端契約（已出貨，不可更動）

| | |
|---|---|
| 電影 | `POST /api/v1/movies/:id/localize-nfo` · 無 body 要求 · additive |
| 影集 | `POST /api/v1/series/:id/localize-nfo[?include_episodes]` · body `{"confirm_replace": true}` |
| 單集 | `POST /api/v1/episodes/:id/localize-nfo` · body `{"confirm_replace": true}` |
| 未確認 | **409** `NFO_REPLACE_NOT_CONFIRMED` |
| 服務不可用 | **503** `NFO_LOCALIZE_DISABLED`（沒有翻譯 provider key） |
| 無檔案路徑 | **400** `VALIDATION_REQUIRED_FIELD` |

`?include_episodes` 的解析是**寬鬆**的：有帶就是要，除非值明確是 `false`／`0`／`no`。

### Time-dependent visual coverage

- **N/A —— 本 story 零 `apps/web/src/components/**` 變更**（design-only）。Rule 23 不適用。

### Project Structure Notes

- 產出：`ux-design.pen`（Pencil）＋ `_bmad-output/screenshots/flow-b-detail-v2/`（與視需要 `flow-j-specs/`）＋提示詞 md。
- 零 `apps/` 變更。

### References

- [Source: `apps/web/src/routes/media/$type.$id.tsx#77`] — 🔴 詳情路由無旗標、直接渲染 `LocalDetailV2`（v1 是死程式碼的實證）
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx#1,147-184`] — v2 設計錨點與現有三個動作
- [Source: `apps/web/src/components/media/DetailPanelMenu.tsx#1`] — v1 的 `Screen 4c` 錨點（**不要用**）
- [Source: `apps/web/src/components/ui/Dialog.tsx`、`library/BatchConfirmDialog.tsx`] — 可抄的確認對話框語彙
- [Source: `apps/api/internal/handlers/nfo_localizer_handler.go`] — 三條路由、`confirm_replace`、錯誤碼、回傳形狀
- [Source: `_bmad-output/implementation-artifacts/9R-13a-tv-nfo-localization.md`] — 單槽/覆寫/備份語意的來源
- [Source: `project-context.md#Rule 21`] — ESLint 強制的 design-node traceability
- [Source: `.claude/memory/feedback_pen_inline_agent_workflow.md`] — `.pen` 修改的協作模式（AC #5）
- [Source: `.claude/memory/feedback_pencil_spec_standalone_screen.md`] — spec 畫面獨立成頁
- [Source: `.claude/memory/project_pen_flow_layout_convention.md`] — canvas 版面慣例
- [Source: `_bmad-output/implementation-artifacts/9R-UX-auto-generation-toggle-design.md`] — 同型 UX story 的先例與交付節奏

---

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- **Pencil MCP 唯讀查證**（`get_app_state` + `execute` 的 `Get`）：
  - `uRGu2` = frame **`B3p-D`**、`N2fmG6` = frame **`B4p-D`**、`SzNRb` = frame **`B3p-M`** —— 三者都是 **screen frame，不是 Reusable Component**（36 個 Reusable Components 清單裡查無這兩個 id）。
  - 動作列節點：`D2HOt`（B3p-D）／`KgifB`（B4p-D）／`vWLjV`（B3p-M），皆為 auto-layout（`gap`/`padding`/`alignItems`）。
  - 🔴 **三個動作列的內容全是舊的**：`播放`(play)／`加入片單`(plus)／`⋯`(ellipsis)。
    程式碼渲染的是 `管理字幕`／`修改資訊`／`複製檔案路徑`（`LocalDetailV2.tsx:147-184`）。**零重疊。**
  - 交叉查證：`grep -rln "片單|watchlist" apps/web/src` → **空**；`LocalDetailV2.tsx:10-14` 檔頭明文
    「Vido has NO playback path … there is NO 播放 button」。⇒ **設計稿在設計兩個不存在的功能**。
  - `GetVariables()` 36 個變數：**`text-tertiary` 與 `border-default` 不存在**（初稿提示詞誤用），
    已改為 `text-muted` / `border-subtle` 後才交付 —— inline agent 沒有對話上下文，錯的變數名會讓它自行猜測。
  - Toast：`apps/web/src/components/ui/` **沒有 Toast 元件**；既有語彙是 `RequestButton.tsx` 的
    inline pill（`role="status"` + `aria-live="polite"` + rounded-full + tinted）。
  - J-spec 版面：J4-D `x=19720`／J5-D `x=21060`（`y=24300`，寬 1240，步進 1340），
    caption 命名 `Caption J{N}-D`、`y = frame.y - 30`、fontSize 14 / weight 600 / `#888888`
    ⇒ **J6-D 落在 `x=22400, y=24300`**。

### Completion Notes List

## ✅ Sally MCP 唯讀 review：**PASS**（2026-08-21）

五段提示詞由 Alexyu 以 Pencil Inline AI Agent 執行完畢，Sally 逐段獨立驗證：

| 段 | 產出 | 驗證結果 |
|---|---|---|
| 1 | `D2HOt`（B3p-D）四顆動作 | ✅ 順序／圖示／文字全對，新節點 `hR1BD` 逐項合規，寬 484／容器 962，零裁切 |
| 2 | `KgifB`（B4p-D）四顆動作 | ✅ 與電影版**逐項相同**，新節點 `k8YyiI`，寬 484，零裁切 |
| 3 | `vWLjV`（B3p-M）兩排四顆 | ✅ 四顆皆 174×46，`174+10+174 = 358 = 390−32(padding)` 剛好填滿，零裁切 |
| 4 | `zMYsL`（J6-D）兩個對話框 | ✅ **17 個文字節點逐字全對**，座標 `22400,24300` 精確，區域無重疊，零裁切 |
| 5 | `Ki1Uc`（sec-result）四態 pill | ✅ **6 個文字節點逐字全對**，四顆 `h=40 r=999`，零裁切 |

**逐字比對合計 23 個文字節點，零改動、零潤飾。** 定稿文案是產品對使用者的承諾，這點必須零容忍。

### 🔁 inline agent 回報的 3 次偏離 —— **全部追認，且全部反饋回未執行的段落**

| # | 偏離 | Sally 裁決 | 連帶修正 |
|---|---|---|---|
| 1 | `subtitles` 圖示不存在 → 改用 `captions` | ✅ **追認，且是我寫錯** —— `captions` 在本檔**已用 14 次**（既有字幕語彙），`subtitles` 0 次 | 提示詞 2、3 同步改 |
| 2 | `alignItems: "stretch"` schema 不吃 → 改用 `start` | ✅ **追認** —— 實查全檔 `alignItems` 只有 `center`×2797 / `end`×101；撐滿本就該靠 `width: "fill_container"` | 🔴 **連帶發現提示詞 4、5 有 6 處 CSS 寫法**（`flex-end`×2 / `flex-start`×4）會全部噴錯，已預先修正 |
| 3 | `fill_container` 的 text 需 `textGrowth: "fixed-width"`；`transparent` 不存在需用 `#00000000` | ✅ **追認** —— 沒有 `textGrowth`，長中文會排成不換行單行、爆出對話框 | 提示詞 5 的 `note` 同步補上 |

> **這個回報迴路的價值**：三次偏離讓提示詞檔在**尚未執行**的段落先修好，估計省下至少 9 次失敗的 execute。

### 截圖交付

`python3 scripts/export-pen-screenshots.py` 產出 161 張；依慣例**只 stage 設計真變更的 4 張**，
其餘 151 張 re-render 雜訊以 `git checkout` 丟棄（全量重跑非決定性，每張都有 byte diff）：

| 檔案 | 變化 |
|---|---|
| `flow-b-detail-v2/b3p-d.png` | 91,563 → 95,224 |
| `flow-b-detail-v2/b4p-d.png` | 92,959 → 93,148 |
| `flow-b-detail-v2/b3p-m.png` | 72,585 → 77,468 |
| `flow-j-specs/j6-d.png` | **新增** 37,667 |

存檔已驗證落盤（`hR1BD`／`k8YyiI`／`w1nnr`／`zMYsL`／`Ki1Uc` 五個新節點皆在磁碟上的 `.pen` 內）。
`scripts/export-pen-screenshots.py` 的 `SCREENS` 已登記 `"zMYsL": ("flow-j-specs", "j6-d")`。

⚖️ **Alexyu 裁定 2026-08-21：順手校正整排**（不在三顆假按鈕旁邊加第四顆）。

**七項設計裁定**（完整理由表見 `9R-UX-nfo-localization-entry-design-prompt.md` 末段）：

1. **AC #1 入口** —— 選 (a)：直接加第四顆**有標籤的** secondary 按鈕，排在「修改資訊」後、「複製路徑」前。
   **不引入 overflow `⋯`** —— 把一個會覆寫檔案的動作藏進選單，風險與可見度不對等。
2. **AC #1 手機** —— `B3p-M` 改成**兩排、每排兩個、四個都有文字標籤**。390 寬放不下四顆有標籤的按鈕，
   而壓成無標籤 icon 更糟：會覆寫檔案的按鈕不該是猜謎圖示。
3. **AC #2 電影 vs 影集** —— 按鈕**完全一致**（同位置、同 `languages` 圖示、同文字「在地化資訊」），
   差異全部收在確認對話框。在按鈕上分岔會讓人以為是兩個不同功能。
4. **AC #3 ⚠️ 推翻 story 的預設：電影也要確認對話框。**
   story AC #3.3 說電影 additive 所以免確認，但**電影一樣會花錢**（LLM 翻譯）。
   2026-08-19「花錢須同意」的精神是付費動作要有明確同意動作 —— 一鍵無提示就開始計費不可接受。
   電影版對話框講「不會覆寫」＋「會花額度」兩件事（安心 ＋ 誠實）。
5. **AC #4 `include_episodes` 預設不勾** —— 勾了等於第一次嘗試就可能花 24 倍的錢。
   「先做一部、滿意再做整劇」是自然的學習路徑。
6. **AC #3 主鍵文案** —— 電影「**開始在地化**」／影集「**備份並覆寫**」。不得是含糊的「確定」；
   影集那顆要把兩個動作都說出來（先備份、才覆寫）。
7. **AC #4 結果回饋** —— **inline 狀態 pill 就地取代按鈕**，不是浮動 toast（repo 沒有共用 Toast 系統，
   沿用 `RequestButton` 既有語彙）。四態：電影成功／影集覆寫成功／整劇部分成功／未設定金鑰。
   `skipped` 對使用者說「**略過**」，spec 附註解釋「DB 有這集但硬碟沒檔案」。

**交付**：`9R-UX-nfo-localization-entry-design-prompt.md` —— 五段自包含提示詞（節點 id、定稿字串、樣式規格全部寫死）。

### Discovery Triage

- **本 story 是否發現任何超出當前範圍的工作？**
  - 若 **NO**：寫 `N/A — no out-of-scope work discovered`。
  - 若 **YES**：每項一列，歸入**恰好一條** lane（① / ② / ③）。

**YES —— 3 項（Sally 於 2026-08-21 裁定期間發現）：**

| Lane | 發現 | 說明 |
|---|---|---|
| ① expand-scope-in-place | **設計稿在設計兩個不存在的功能**（播放／加入片單），三個 v2 詳情 frame 皆然 | ⚖️ Alexyu 裁定「順手校正整排」⇒ 由《提示詞 1-3》就地吸收 |
| ③ backlog | **`LocalDetailV2.tsx` 的 Rule 21 檔頭語法與現實不符** —— 寫 `Implements: Component/Detail-Movie-v2 (uRGu2)`，但 `uRGu2` 是 **screen frame 不是 Reusable Component**；依 Rule 21 應為 `// Design ref: ux-design.pen Screen {Name} ({id})` | 需開條目。本 story 不改（不在範圍） |
| ③ backlog | **v2 影集詳情沒有手機版畫面** —— 只有 `B3p-M`（電影），**沒有 `B4p-M`**。影集手機的季集手風琴與本案的兩排動作列在手機上無設計覆蓋 | 需開條目 |

**其餘候選：**

- **v1 詳情頁（`MediaDetailPanel` / `DetailPanelMenu` / `Screen 4c`）是死程式碼** —— 本 story 只是「不去動它」。
  若 Sally 認為 v2 遷移已足以刪除 v1，那是**獨立的清理案**（lane ③），需連同 gallery fixture 與 `Screen 4c` 設計節點一併處理。

### File List

<!-- 預期：
  ux-design.pen
  _bmad-output/screenshots/flow-b-detail-v2/*.png（與視需要 flow-j-specs/）
  _bmad-output/implementation-artifacts/9R-UX-nfo-localization-entry-design-prompt.md
  scripts/export-pen-screenshots.py（若有新畫面）
-->

### Change Log
