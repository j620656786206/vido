# Story 9R-UX: 分集列的逐集字幕入口（design）

Status: ready-for-dev

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

- [ ] **Task 1 — 現況勘查（AC: #1, #2）**
  - [ ] Pencil MCP `get_app_state`（含 schema ＋ canvas design）
  - [ ] 檢視 `N2fmG6`／`r1EY9`／`j2-d` 現況，確認 Findings 1–5 與畫布一致
  - [ ] 盤點可重用的按鈕型與 token

- [ ] **Task 2 — 狀態×動作矩陣裁定（AC: #2）**
  - [ ] 十個 `subtitle_status` 逐一裁定，產出 Completion Notes 表格
  - [ ] 與 J2-D icon grammar 的相容性檢查（不得產生「已定局圖示 + 大聲的動作」的矛盾）

- [ ] **Task 3 — 出提示詞交 Alexyu（AC: #1, #3, #4）**
  - [ ] 撰寫 Inline AI Agent 提示詞（`9R-UX-episode-row-cta-design-prompt.md`）
  - [ ] 涵蓋 `N2fmG6` 逐集 affordance、`r1EY9` 文案與 episode 標題規則、行動版裁定

- [ ] **Task 4 — MCP review（AC: #1–#4）**
  - [ ] Alexyu 跑完 agent 後，Sally 以 MCP 逐條對照 AC #1–#4
  - [ ] 標籤重疊檢查、spec 畫面 standalone 檢查

- [ ] **Task 5 — 截圖與交付（AC: #5）**
  - [ ] 確認 `.pen` 已存檔 → `python3 scripts/export-pen-screenshots.py`
  - [ ] 新畫面更新 `SCREENS` dict；只 stage 真正改動的 PNG
  - [ ] Completion Notes 寫齊：狀態×動作矩陣、node ids、文案字串表（`9R-10c` 逐字實作的來源）

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

<!-- 必填：狀態×動作矩陣表、node ids、文案字串表、行動版裁定與理由 -->

### Discovery Triage

<!-- 設計期間若發現超出範圍的事項，照 lane ①/②/③ 分流並在發現當下寫進 sprint-status.yaml -->

### File List

## Change Log

| Date | Change |
|---|---|
| 2026-08-19 | Story 建檔（SM Bob, create-story）。⚖️ Alexyu 裁定 9R-10a 的 FE 半邊改走 design-first，本 story 由 lane ③ backlog 條目升格為 lane ② blocking 設計 story。5 AC／5 task，UX/DESIGN-ONLY。硬阻斷 `9R-10c-episode-row-subtitle-cta-frontend`。 |
