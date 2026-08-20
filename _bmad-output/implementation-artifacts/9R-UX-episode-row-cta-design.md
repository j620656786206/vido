# Story 9R-UX: 分集列的逐集字幕入口（design）

Status: done

**Epic:** epic-9R-subtitle-route-c · **Risk: 🟢 UX/DESIGN-ONLY（Sally ux-designer，NOT dev）· 零程式碼**
**Created:** 2026-08-19（SM Bob, create-story）
**Source:** ⚖️ Alexyu 裁定 2026-08-19 —— 9R-10a authoring 原採 function-first，**Alexyu 推翻，改走 design-first**。
本 story 因此由 lane ③ backlog 條目**升格為 lane ② blocking story**。
**Depends on:** nothing（只需 Pencil.app 執行中）。
**Blocks:** `9R-10c-episode-row-subtitle-cta-frontend`（硬阻斷 —— 那支 story 的 Status 是 `blocked`，
本 story done 且經 Sally MCP review 後才解鎖）。

---

## Story

As Alexyu（在影集詳情頁看到某一集缺字幕的人），
I want 那一列本身就能通往字幕生成，
so that 我不必離開分集清單、也不必猜「管理字幕」按鈕到底管的是整部劇還是這一集。

---

## 🔎 Findings（2026-08-19 create-story 逐檔驗證 —— 設計前必讀的現況事實）

1. **分集列今天零動作 affordance。** `EpisodeList.tsx` 每列只渲染 `SxxExx`／集名／播出日／片長／
   `SubtitleStatusIcon`。整份清單唯一的 `<button>` 在 `:163`，是清單層級的 retry。
   ⇒ 這是**淨新增**的 affordance，不是既有元件的修改。

2. **影集層級的 CTA 是死的，而且文案已過期。** `ManageSubtitleDialogV2.tsx:443` 對 series
   `disabled={!isMovie ...}`，`:457` helper 寫「影集字幕生成即將推出」。
   ⇒ 這句話在 sub-4-2（批次吃 episode id）與 sub-5-3（F15 series 群組勾選）出貨後**已是謊話**：
   影集**可以**生成，只是走批次。文案必須一起處理，否則新舊兩個入口會互相矛盾。

3. **十種字幕狀態的圖示語彙已由 Sally 定稿。** `EpisodeList.tsx:36-52` 的 J2-D icon grammar
   （CIRCLED = 已定局／BARE = 尚未定局；顏色 = 急迫度不是結果）＋ `flow-j-specs/j2-d`。
   ⇒ 新 affordance **必須與這套語彙共存而不打架**：一列上同時有「狀態圖示」與「動作」時，
   兩者的視覺權重誰高、是否所有狀態都該出現動作、`已略過`／`無字幕源` 這類**定局但可用 ASR 翻盤**的
   狀態要不要出現動作 —— 這些正是本 story 要裁的。

4. **不是每一列都有本地檔。** `MergedEpisode.hasLocalFile` 為 false 時（TMDb 有這集但 NAS 沒有），
   `SubtitleStatusIcon` 整個不渲染（`EpisodeList.tsx:115`）。動作 affordance 顯然也不該出現，
   但**那一列會不會因此看起來「壞掉」**（有些列有東西、有些列空白）是設計要回答的。

5. **既有可重用的語彙**：電影詳情頁的「管理字幕」按鈕（`LocalDetailV2.tsx:13` 檔頭稱其為
   the subtitle differentiator）。SM 的 authoring 傾向是重用它，**但這是傾向不是裁定** ——
   Sally 可以判定列內尺度需要不同型態（圖示鈕／hover 顯現／整列可點）。

---

## 目標畫面（`.pen`）

| 畫面 | node id | 檔名 | 角色 |
|---|---|---|---|
| TV Detail v2（含季選擇器＋分集清單） | `N2fmG6` | `flow-b-detail-v2/b4p-d.png` | **主戰場** —— 逐集 affordance 畫在這裡 |
| F1-D-v2 管理字幕 dialog | `r1EY9` | `flow-f-subtitle-v2/f1-d-v2.png` | Finding 2 的過期文案；並確認 dialog 在 episode 情境的標題／副標怎麼指認「哪一集」 |
| J2-D 字幕狀態徽章規格 | — | `flow-j-specs/j2-d.png` | Finding 3 的語彙來源；若新 affordance 影響狀態列的讀法，此規格頁需附註 |

⚠️ **行動裝置**：`flow-b-detail-v2` 只有 `SzNRb`（b3p-m，**電影**行動版）——
**沒有 TV 行動版畫面**。Sally 需裁定：補一張 TV 行動版、或在 `N2fmG6` 上以註記說明行動版行為。
（列內動作在窄螢幕是經典難題：44px 觸控目標 vs 一列塞不下。這一項不可略過。）

---

## Acceptance Criteria

### AC #1 — `N2fmG6` 畫出逐集字幕入口

**Given** Finding 1，**then** TV detail v2 的分集列出現通往「這一集」字幕生成的 affordance。Sally 裁定其形態，約束：

- **只在 `has_local_file = true` 的列出現**（Finding 4）。沒有本地檔的列**不得**出現任何動作。
- 與同列的 `SubtitleStatusIcon` **視覺權重明確分工** —— 不得出現「兩個小圖示並排、看不出誰是狀態誰是按鈕」。
- 觸控目標 ≥ 44px（專案既有 a11y 底線，dialog CTA 亦然）。
- 可及性名稱**必須含 `SxxExx`**：同一頁會有十幾列相同文案的按鈕，沒有集號就無法區辨。
- **零新 design token、零新元件類**優先。若 Sally 判定必須新增，於 Completion Notes 記錄理由。

### AC #2 — 裁定「哪些字幕狀態該出現動作」

**Given** Finding 3 的十值語彙，**then** 明確裁定每一種 `subtitle_status` 下這個 affordance 的存在與樣態。
至少必須回答：

- `found`（已有繁中字幕）—— 還要不要給入口？（重生／取代的語意）
- `untranslated`（已生成英文、尚未翻譯）—— 這是 **translate-only 續跑**，成本遠低於全跑；
  要不要有不同的文案／樣態，讓使用者知道這次很便宜？
- `skipped`／`no_text_source` —— 定局但 ASR 可翻盤；動作的存在會不會與「已定局」的圖示語彙互相矛盾？
- 四種 in-flight（`probing`／`extracting`／`translating`／`searching`）—— 動作應為 disabled 還是消失？

裁定寫進 Completion Notes 的表格（狀態 → 動作存在？→ 文案／樣態），**該表就是 `9R-10c` 的實作契約**。

### AC #3 — `r1EY9` 的影集文案與 episode 情境

**Given** Finding 2，**then**：

- 影集層級 CTA 的過期文案「影集字幕生成即將推出」被取代。Sally 裁定新文案與 CTA 狀態
  （維持 disabled ＋ 指路？改成通往批次的連結？還是整塊改寫？）。
- dialog 在 **episode 情境**開啟時，標題／副標如何指認「這是第幾季第幾集」——
  今天 `mediaTitle` 是單一字串，設計需給出組合規則（例：`集名` 為標題、`劇名 · SxxExx` 為副標，或反之）。

### AC #4 — 行動裝置行為有答案

**Given** 上方 ⚠️，**then** TV 行動版的逐集動作有明確設計：補一張 `-m` 畫面，
或在 `N2fmG6` 上以 Pencil 註記寫清楚（哪一種由 Sally 裁定並記錄理由）。

### AC #5 — 交付與門檻

- `.pen` 修改**只能**經 Pencil MCP（`.pen` 是加密檔，禁止 Read／Grep）。
- 工作流依 `feedback_pen_inline_agent_workflow`：**Sally 出提示詞 → Alexyu 跑 Pencil Inline AI Agent
  → Sally MCP review**。Sally **不直接編輯**。
- 標籤／標題不得與其他內容重疊（`feedback_pencil_label_overlap`）。
- 若需要獨立的規格／決策說明畫面，**另開 standalone screen**，不得塞進既有 mockup
  （`feedback_pencil_spec_standalone_screen`）。
- **存檔驗證**：commit 截圖前必須確認 `.pen` 已存檔 —— MCP 匯出讀的是 app 記憶體不是磁碟
  （`feedback_verify_pen_saved_before_commit`）。
- 重產截圖：`python3 scripts/export-pen-screenshots.py`；新畫面須更新該腳本的 `SCREENS` dict。
  ⚠️ **full regen 非決定性** —— 只 stage 設計真正改動的 PNG，其餘 `git checkout` 還原。
- `.pen` 與 `_bmad-output/screenshots/` 同一個 commit。

---

## Tasks / Subtasks

- [x] **Task 1 — 現況勘查（AC: #1, #2）**
  - [x] Pencil MCP `get_app_state`（含 schema ＋ canvas design）
  - [x] 檢視 `N2fmG6`／`r1EY9`／`j2-d` 現況，確認 Findings 1–5 與畫布一致
  - [x] 盤點可重用的按鈕型與 token

- [x] **Task 2 — 狀態×動作矩陣裁定（AC: #2）**
  - [x] 十個 `subtitle_status` 逐一裁定，產出 Completion Notes 表格
  - [x] 與 J2-D icon grammar 的相容性檢查（不得產生「已定局圖示 + 大聲的動作」的矛盾）

- [x] **Task 3 — 出提示詞交 Alexyu（AC: #1, #3, #4）**
  - [x] 撰寫 Inline AI Agent 提示詞（`9R-UX-episode-row-cta-design-prompt.md`）
  - [x] 涵蓋 `N2fmG6` 逐集 affordance、`r1EY9` 文案與 episode 標題規則、行動版裁定

- [x] **Task 4 — MCP review（AC: #1–#4）**
  - [x] Alexyu 跑完 agent 後，Sally 以 MCP 逐條對照 AC #1–#4
  - [x] 標籤重疊檢查、spec 畫面 standalone 檢查

- [x] **Task 5 — 截圖與交付（AC: #5）**
  - [x] 確認 `.pen` 已存檔 → `python3 scripts/export-pen-screenshots.py`
  - [x] 新畫面更新 `SCREENS` dict；只 stage 真正改動的 PNG
  - [x] Completion Notes 寫齊：狀態×動作矩陣、node ids、文案字串表（`9R-10c` 逐字實作的來源）

---

## Dev Notes

### 這支 story 的產出物就是 `9R-10c` 的規格

`9R-10c` 的 Status 是 `blocked`，其 STOP GATE 明訂「必須繼承本 story 的 node ids、狀態×動作矩陣、
文案字串表，**不得自行發明**」。因此 Completion Notes 的完整度**就是**交付品質本身 ——
spec 畫面的 PNG 匯出解析度偏低（`backlog-pen-spec-screen-readable-export` 尚未解），
**字串與矩陣必須以文字形式寫在 Completion Notes**，不能只存在於圖裡。
先例：sub-2-2c（γ）以 Completion Notes 的字串表交付給 sub-2-2d。

### Rule 20 / Rule 7 / Rule 23

- **Rule 20：** 本 story 不定義也不消費 wire contract ⇒
  `📎 Contract Stamps: NONE (design-only story; defines no wire contracts and references none)`
- **Rule 7：** 零程式碼 ⇒ 不適用。
- **Rule 23：** 零程式碼 ⇒ `N/A — design-only, no components touched`。

### References

- [Source: `apps/web/src/components/media/EpisodeList.tsx:36-52,115,163`] — J2-D icon grammar／`hasLocalFile` 閘門／唯一既有按鈕
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:443,457`] — 死 CTA 與過期文案
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:13`] — 「管理字幕」按鈕型（可重用候選）
- [Source: `_bmad-output/implementation-artifacts/ux2-3-detail-v2.md:20,61`] — `N2fmG6` = TV detail v2 含 `SeasonAccordion`
- [Source: `_bmad-output/implementation-artifacts/sub-2-2c-f5-asr-copy-design.md`] — design-story 先例（字串表交付模式）
- [Source: `CLAUDE.md#UX Design Screenshots Workflow`] — 匯出腳本、`SCREENS` dict、非決定性 regen 告誡
- [Source: memory `feedback_pen_inline_agent_workflow` / `feedback_verify_pen_saved_before_commit` / `feedback_pencil_label_overlap` / `feedback_pencil_spec_standalone_screen`]

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (Sally, ux-designer)

### Debug Log References

### Completion Notes List

> ⚠️ **回填說明（Amelia, 2026-08-20）**：本區塊在設計輪當下未寫回檔案 —— 產出當時記在
> chat、`sprint-status.yaml` 條目與 PR #236，但 story 檔的 Completion Notes 留空。
> 9R-10c 的 STOP GATE 指名這裡為規格來源，因此開工前依規回填。
> **內容全部從已合併的 `.pen`（Pencil MCP 逐節點讀出）取得，非憑記憶重述**；
> `sprint-status` 條目與 PR #236 內文為交叉佐證。

#### ⚖️ 核心裁定 —— 動作與狀態解耦

> 分集列的動作是「**開啟管理字幕對話框**」，不是直接生成。
> **十種 `subtitle_status` 一律呈現相同動作，不做任何分歧。**

理由（J3-D `ruling-line-2` 逐字）：J2-D 已定「字形＝是否定局、顏色＝急迫度」，狀態由同列指示器
承載。動作再隨狀態改變＝同一事實編碼兩次，且 25 集清單會出現按鈕忽隱忽現的雜訊。

唯一的閘門是 `has_local_file`：沒有本地檔案的集不出現任何動作（TMDb 有、NAS 沒有的集不可誤導使用者）。

#### 📐 node ids（9R-10c 實作依據）

| 用途 | node id | 位置 |
|---|---|---|
| 獨立規格頁 **J3-D** | `Z54xAd` | 截圖 `flow-j-specs/j3-d.png`；`SCREENS` dict 已登記 |
| 列內動作按鈕 | `Uj2bq` / `A5Sm6G` / `JIvv3` | `N2fmG6` > `VNeL9` > `aNNxp` > 各 `ep` 列 |
| 按鈕文字 | `CFSFm` / `zLvLX` / `umnnf` | 各 `btn-subtitle` 之下 |
| 對話框標題（劇名） | `Aodey` | `r1EY9` header |
| 對話框標題（集號 chip） | `tO72N` | `r1EY9` header |
| 新增條件文案三行 | `jjKhO` / `oTTwd` / `Nqfd9` | `r1EY9` > `x1OAW` |

#### 🎨 按鈕規格（逐欄位，直接照抄）

```json
{
  "type": "frame", "name": "btn-subtitle", "height": 44,
  "cornerRadius": "$radius-md", "padding": [0, 14],
  "justifyContent": "center", "alignItems": "center",
  "children": [{
    "type": "text", "name": "subtitle-label", "fill": "$text-secondary",
    "content": "管理字幕", "fontFamily": "Noto Sans TC",
    "fontSize": 13, "fontWeight": "normal"
  }]
}
```

零新 token、零新元件類 —— 等同 `RequestRow` 已核定的純文字按鈕 `yTntT`。
插入位置：`ep` 列的 `th → inf → **btn-subtitle** → st`（`ep` 是 horizontal flex，`inf` 為
`fill_container`，插入後自動重排）。

#### 📊 狀態 × 動作矩陣（J3-D 表格逐列）

| `subtitle_status` | 列上出現動作？ | 對話框開啟後的樣態 |
|---|---|---|
| `found` | 是 | 完成態；重新生成會覆蓋既有繁中字幕 |
| `not_found` | 是 | 缺字幕態，可生成 |
| `not_searched` | 是 | 缺字幕態，可生成 |
| `searching` | 是 | 進度態（線上搜尋中） |
| `probing` | 是 | 進度態（偵測字幕軌中） |
| `extracting` | 是 | 進度態（抽取內嵌字幕中） |
| `translating` | 是 | 進度態（翻譯字幕中） |
| `no_text_source` | 是 | 缺字幕態；僅語音辨識可解 |
| `skipped` | 是 | 缺字幕態；軌道語言不符已略過 |
| `untranslated` | 是 | 缺字幕態＋助語「僅需翻譯，不再重跑語音辨識」 |

**十列皆「是」不是偷懶** —— 是刻意讓入口穩定：使用者不必先讀懂狀態才知道能不能點。
可行性判斷在對話框內做，那裡有空間解釋。

#### 📝 文案字串表（逐字採用）

| 位置 | 字串 |
|---|---|
| 列內動作標籤 | `管理字幕` |
| a11y 名稱格式 | `管理 S01E03 的字幕`（**必須含集號**） |
| 助語・`untranslated` | `僅需翻譯，不再重跑語音辨識——這次很快也很便宜` |
| 助語・生成進行中 | `本集正在生成字幕——開啟即接續顯示進度，不會重複啟動` |
| 影集層級助語 | `請於下方分集清單逐集生成`（CTA **維持不可按**） |

#### 📱 行動版裁定（AC #4）

**補註記，不畫新畫面。** `flow-b-detail-v2` 沒有 TV 行動版（`SzNRb` 是電影行動版），
而補一張需連 hero／meta 整組重做，超出本 story。行為本身可推導，故以 J3-D `ruling-line-5`
明文規範：

> 行動版（<sm）：整列改為上下堆疊（沿用 12-2 的堆疊規則），管理字幕移到第二行、靠左、
> 維持 44px 觸控高度；不另出 TV 行動版畫面。

#### 🏷️ 對話框標題規則（AC #3）—— 不需重畫，畫布上已存在

`Aodey`「管理字幕 — 怪奇物語」＋ `tO72N`「S04E07」。
設計從一開始就假設**逐集**操作，只是後端一直沒有路由（9R-10a 已於 #234 補上）。
⇒ 規則：**劇名為標題、`SxxExx` 為獨立 code chip**。

#### ⚠️ 設計未涵蓋（9R-10c 需自行處理並回頭追認）

**hover 態全檔的列內動作皆未繪**（不只本輪）。9R-10c 實作時需自訂一個合理的 hover，
並在 UX 驗證時交 Sally 追認。

#### 🚦 Sally MCP Gate

**PASS-WITH-MUST-FIX** — 1 MUST-FIX（已修並複驗）、2 NOTE、1 lane ③ 立案。

- **MUST-FIX（MEDIUM）**：第一輪的 in-frame 規格註記把 `N2fmG6` 從 900 撐到 1000，
  但全檔桌面畫面皆 900（含四個同族兄弟 `uRGu2`/`Z42zy`/`Tqy3E`/`UH0sk`；更高的首頁 1714／
  下載 1560／活動 1360 是刻意的整頁捲動例外）。**根因是提示詞要錯了**（設計決策說明應獨立成頁），
  已把內容移入 J3-D `ruling-line-4/5`、刪除註記、高度復原 900。複驗：0 clipping、最深內容剛好 900、
  B 系列五個畫面高度全部 900。
- **ACCEPT（不改）**：對話框 `gD99f` y 110→36 是正確置中（(900−827)÷2＝36.5；827 高擠不進
  兄弟的 140）。
- **NOTE（LOW）**：新增三行的節點命名為 `note-*`，打斷原區塊 `dn-head`＋`dn-line-1..5` 的序號；
  順序正確，非阻斷。
- **NOTE（LOW）**：「管理字幕」為 `$text-secondary` 純文字，視覺上比同列的 `繁中` 藥丸安靜 ——
  符合「安靜、恆在」的裁定與 `yTntT` 先例，**不改**；但這正是 hover 態缺口需要補上的原因。

#### 📎 其他

- **Contract Stamps: NONE**（design-only；不定義也不消費 wire contract）
- **Rule 7 / Rule 23**：N/A（零程式碼）
- 匯出：只 commit 真的變了的三張（`b4p-d` 400×277→400×250 對應 1000→900、
  `j3-d` 新增、`f1-d-v2`），其餘 153 張重繪雜訊已還原
- 出貨：PR #236（`6c75a432`）；狀態流轉 PR #237（`b91dbccf`）

### Discovery Triage

- **YES — 1 筆，lane ③（已於發現當下立案，雙向）：**
  - `backlog-episodelist-status-pill-vs-icon-drift` —— `.pen` 的分集列狀態畫的是**文字藥丸**
    「繁中」（`$success-tint` 底／`$success` 字／11px 600），但出貨的 `EpisodeList.tsx` 渲染的是
    J2-D icon grammar 的**純圖示**。同一個元素，設計與程式碼自 12-2 起分頭演進未曾對齊。
    **非本輪範圍**（提示詞明文保護 `st` 節點，避免一輪混兩件事）。條目附三個處理選項與 SM 預判傾向。

### File List

- `ux-design.pen` — `N2fmG6` 三列 `btn-subtitle`；新增 `Z54xAd`（J3-D）；`r1EY9` 條件文案 +3 行；
  MUST-FIX：刪除 in-frame 註記、高度復原 900、J3-D `ruling-line-4/5`
- `_bmad-output/screenshots/flow-b-detail-v2/b4p-d.png`
- `_bmad-output/screenshots/flow-j-specs/j3-d.png`（新增）
- `_bmad-output/screenshots/flow-f-subtitle-v2/f1-d-v2.png`
- `scripts/export-pen-screenshots.py` — `SCREENS` 登記 `Z54xAd`；Pen 1.2.5 改走 `execute`+`Export`；
  自審修正：匯出不完整時非零離開 + 檢查 `sips` 回傳碼
- `_bmad-output/implementation-artifacts/9R-UX-episode-row-cta-design-prompt.md`（新增）
- `_bmad-output/implementation-artifacts/9R-UX-episode-row-cta-fix-prompt.md`（新增）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-08-19 | Story 建檔（SM Bob, create-story）。⚖️ Alexyu 裁定 9R-10a 的 FE 半邊改走 design-first，本 story 由 lane ③ backlog 條目升格為 lane ② blocking 設計 story。5 AC／5 task，UX/DESIGN-ONLY。硬阻斷 `9R-10c-episode-row-subtitle-cta-frontend`。 |
| 2026-08-20 | **Completion Notes 回填（Amelia，9R-10c Task 0 STOP GATE）。** 設計輪產出當下只寫進 chat／sprint-status／PR #236，story 檔的 Dev Agent Record 留空；而 9R-10c 的 STOP GATE 指名此處為規格來源，缺漏就必須停工。依規回填：核心裁定、6 組 node ids、按鈕逐欄位規格、十列狀態×動作矩陣、5 條文案字串、行動版裁定、標題規則、hover 缺口、Sally gate 結論、lane ③ discovery、File List。**全部由 Pencil MCP 從已合併的 `.pen` 逐節點讀出**，非憑記憶重述。Status → done、17 個 checkbox 補齊（工作早已完成並 merged，只是未回寫檔案）。 |
