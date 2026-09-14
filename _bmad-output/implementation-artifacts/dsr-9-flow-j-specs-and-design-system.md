# Story DSR.9: 設計系統與規格稿——先把陰影規則裁乾淨，後面九張才判得動

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who has to judge "這道陰影該不該在" nine more times,
I want DESIGN.md 對陰影只說一種話，而且抬升尺規只有一種方言,
so that 後面每一張 dsr story 不用各自重新發明判準。

## Context

`epic-dsr` 的第四張（dsr-10／11／12 已 done）。**它被提前，因為它擋著另外九張。**

2026-09-11 的卡片改寫（`disc-2026-09-flow-a-m-screen-content`）把 §Cards 從「有陰影」改成「髮絲邊框、無陰影」，並附了實測數字。那次改寫**只改了一節**，於是 DESIGN.md 現在對陰影同時說三種話。dsr-1／3／5／7／8 的條目都寫著「有 raw shadow-*，要逐一判定是不是真的浮層」——**在這三種話收斂成一種之前，那個判定做不了**。

⚠️ **Flow J 的九張稿不是 UI mockup，是裁定書。** J1–J9 記的是 PosterCard 資訊密度、字幕徽章規格、分集列入口、免費／付費界線、未啟用時的誠實狀態、在地化確認、設定頁表單寬度、政策分工、金額記號。對齊的問題是**「這些裁定在程式碼裡還成立嗎」**，不是像素比對。用錯方法會把裁定書當成畫面去改。

## Acceptance Criteria

### 設計稿節點

`J1-D (XlFIq)` · `J2-D (ZpQaw)` · `J3-D (Z54xAd)` · `J4-D (sPzZT)` · `J5-D (alrIw)` · `J6-D (zMYsL)` · `J7-D (JBKis)` · `J8-D (pbB6P)` · `J9-D (Ls4GO)`
`Design System Reference (8SSzc)` · `Design Language v2 (V2Kez)` · `Navigation Shell v2 (CLo58)` · `Component Library (sJzat)` · `Component Anatomy (wrjOF)`

---

### 陰影普查（本 story 的事實基礎，全部重新量過）

| 方言 | 處數 | 檔數 |
| --- | --- | --- |
| raw Tailwind（`shadow-lg` 這種） | **46** | 35 |
| token（`shadow-[var(--shadow-lg)]`） | **10** | 7 |

比例 **4.6 : 1**。⚠️ 立案時寫的「51 vs 12」不準，那次的 grep 把 `shadow-[var(--shadow-sm)]` 裡的 `shadow-sm` 也算成 raw，也沒有排除註解。本 story 的數字是排除註解與 token 內文之後重數的。

**把 46 個 raw 逐一分類之後，結論跟立案時的假設相反：**

| 類別 | 處數 | 判定 |
| --- | --- | --- |
| 真的浮層（Dialog／Sheet／Popover／Toast／Modal／側板／懸浮進度卡／下拉） | **約 42** | 位置合法，只是用了裸值不是 token |
| 真的違規（不浮的東西掛了陰影） | **4** | 要修 |

**不是「大家在卡片上亂加陰影」，是「抬升尺規有兩種方言，裸值那種贏了 4.6 倍」。**

四個真違規：

| 位置 | 是什麼 |
| --- | --- |
| `components/dashboard/RecentMediaPanel.tsx:130` | 一顆按鈕掛 `shadow-lg` |
| `components/media/MediaDetailPanel.tsx:141` | 一個圖示容器掛 `shadow-lg` |
| `components/media/MediaDetailPanel.tsx:159` | 一張 `<img>` 海報掛 `shadow-lg` |
| `components/media/PosterCard.tsx:152` | 卡片 hover 掛 `shadow-2xl` |

📌 後三者住在 v1 元件（`MediaDetailPanel` 沒有任何 route 掛載、`PosterCard` 是 v1），屬 dsr-2 的範圍。

---

### DESIGN.md 對陰影同時說三種話

| # | 位置 | 說了什麼 | 跟誰打架 |
| --- | --- | --- | --- |
| 1 | §Shadow Vocabulary（L491） | **「`--shadow-md`：`Card` 基本元件」** | §Cards（L650）說卡片**無陰影**。這一條現在指向一個已經不存在的用途 |
| 2 | §Cards（L650） | 陰影白名單＝**Dialog／Sheet／Popover／Toast** | §Buttons（L632）與 §Shadow Vocabulary（L490）都說**按鈕**帶 `--shadow-sm`，按鈕不在白名單裡 |
| 3 | §Shadow Vocabulary（L492） | 「`--shadow-lg`：**罕見**的中間階」 | 實測 `shadow-lg` 是 46 個 raw 裡**最常用**的一階 |

三者都是 2026-09-11 只改一節的副作用。

---

1. ✅ **裁定一：按鈕到底有沒有陰影？** 兩個選項，兩邊都要改文件：
   - **(a) 按鈕保留 `--shadow-sm`** → §Cards 的白名單補上「有填色、可按的控制項」。程式碼零變更（`ui/Button.tsx` ×3、`RequestButton.tsx` ×2 目前就是這樣）。
   - **(b) 按鈕也拿掉陰影** → §Buttons 與 §Shadow Vocabulary 的 `--shadow-sm` 條目刪除，`ui/Button.tsx` 與 `RequestButton.tsx` 共 5 處拿掉，並重生視覺基準。
   - 📌 供參考：§Elevation 的開場白是「**這套系統近乎扁平，用色調分層而非陰影**」，而 §Tone-First Rule 說「只有當這個面真的浮在它所覆蓋的內容之上時，才加陰影」。一顆坐在表單裡的按鈕不浮。但 `--shadow-sm` 是 `0 1px 2px rgba(0,0,0,.3)`，在夜行下幾乎不可見，拿掉的視覺代價接近零、收益是規則少一個例外。

2. ✅ **裁定二：海報磚要不要陰影？** `components/library/PosterCardV2.tsx:100` 的海報磚掛 `shadow-[var(--shadow-md)]`。
   - 它是**全 app 重複次數最多的元素**（每一格、每一頁）。
   - 同一個檔案第 95–98 行的註解自己寫著：「DESIGN.md 的 Tone-First Rule 把陰影配給真的浮起來的東西……把整格海報抬起來等於把預算花在重複最多的元素上」。註解在論證不該抬，程式碼卻有一道 `--shadow-md` 底陰影——註解講的是「hover 時不加碼」，不是「本來就沒有」。
   - 而 §Cards 現在說卡片無陰影，`--shadow-md` 的唯一指定用途（`Card` 元件）也已經被移除。
   - 📌 註解裡「11 uses app-wide」這個數字也過期了，實測是 46 raw ＋ 10 token ＝ **56**。

3. ✅ **裁定三：raw 裸值要不要禁掉？** 42 個合法浮層用的是 Tailwind 裸值而不是 `--shadow-*` token。後果是**抬升沒有單一調校點**：日巡的陰影 2026 年被認真重做過（改墨色、兩層、總墨量 ≤15.4%），但那次重做只改到 token，42 個裸值完全沒跟上——它們在日巡下仍然是 Tailwind 預設的黑色陰影。
   - **(a) 收斂成 token** → 42 處換寫法，並加一條 ESLint 規則擋住新的裸值。工程量大但一次解決。
   - **(b) 明文允許裸值** → §Elevation 補一句，並說明日巡下的落差是已知且接受的。
   - 建議 **(a)**，理由是「日巡那半邊現在是壞的」不是風格問題。

4. **裁定落地之後，DESIGN.md 的三處自相矛盾一次改乾淨。** §Shadow Vocabulary 的 `--shadow-md` 條目重寫（指向裁定二的結果）、§Cards 白名單與 §Buttons 對齊（裁定一）、`--shadow-lg` 的「罕見」改成實測後的描述。

5. **`Design Language v2 (V2Kez)` 補上 Elevation 段。** 那份 `.pen` 參考文件**完全沒有陰影規格**——掃過整個 frame，一個提到陰影的文字節點都沒有。DESIGN.md 有整節 §Elevation & Depth 加 Shadow Vocabulary，設計稿那邊卻沒有，所以在 Pencil 裡畫圖的人沒有陰影規格可循（Flow M 的登入卡就是這樣掛上一道不該有的陰影的）。補一段，內容以裁定後的 DESIGN.md 為準。

6. **J1–J9 九張裁定書逐一查「裁定還成立嗎」。** 不是像素比對。每一張確認它記的裁定在程式碼裡是否仍然為真，不成立的**不要改稿**——那是歷史記錄——改為在該裁定旁加一行「⚠️ 已被 XXX 取代」。
   - 📌 J2 那張自己就寫著「含……一個既有不一致（searching 顏色）的裁決」——而 `searching` 的顏色在 2026-09-13 被 dsr-11 改成泥金了。**J2 需要這樣的一行註記**，這是本條最明確的一筆。

7. **不要動 `components/ui/` 的檔頭。** 立案寫「ui/ 16 檔只有 2 檔有 Design ref」——**不準**。16 含 spec，非 spec 的 12 個檔**全部都有**標頭（Badge／Button／Card／Dialog／FilterRailShell／HighlightText／Pagination／Sheet／SidePanel／Skeleton／TmdbAttribution／Tooltip）。這一項撤銷。

8. **CI 全綠。** lint 0 errors、typecheck、format:check、check-design-tokens、`nx test web`、`nx test api`。裁定一與二若動到按鈕或海報磚，視覺基準走功能分支的四步流程（見 `project_visual_baseline_intentional_change`）。

## Tasks / Subtasks

- [x] **Task 1 — 取得三個裁定（AC: #1, #2, #3）**
  - [x] 把三題連同證據一次呈給 Alexyu，不要邊做邊問
  - [x] 裁定寫進 `sprint-status.yaml` 與本 story 的 Change Log，註明日期

- [x] **Task 2 — DESIGN.md 收斂（AC: #4）**
  - [x] §Shadow Vocabulary 的 `--shadow-md` 條目重寫
  - [x] §Cards 白名單與 §Buttons 對齊
  - [x] `--shadow-lg` 的「罕見」改成實測後的描述
  - [x] 每一處都附上量到的數字，不要只寫結論

- [x] **Task 3 — 依裁定改程式碼（AC: #1, #2, #3）**
  - [x] 裁定一：`ui/Button.tsx` ×3、`RequestButton.tsx` ×2
  - [x] 裁定二：`PosterCardV2.tsx:100`，並把第 95–98 行那段過期的註解一起更新（「11 uses」→ 實測值）
  - [x] 裁定三 (a)：42 處裸值換 token ＋ 一條 ESLint 規則；(b) 則只補文件
  - [x] 四個真違規照 AC 的表修（`MediaDetailPanel` 與 `PosterCard` 屬 v1，確認 dsr-2 是否已接手，避免重工）

- [x] **Task 4 — `.pen` 補 Elevation 段（AC: #5）**
  - [x] `Design Language v2 (V2Kez)` 新增一節，內容以裁定後的 DESIGN.md 為準
  - [x] 用 Pencil MCP，不要用 Read/Grep 碰 `.pen`

- [x] **Task 5 — J1–J9 逐張查裁定（AC: #6）**
  - [x] 九張各確認一次；不成立的加「⚠️ 已被 XXX 取代」而不是改內容
  - [x] J2 的 searching 顏色註記必補（dsr-11 已改成泥金）

- [x] **Task 6 — 收尾驗證（AC: #8）**
  - [x] `pnpm run lint` → 0 errors
  - [x] `pnpm nx run web:typecheck --skip-nx-cache`
  - [x] `pnpm run format:check`
  - [x] `python3 scripts/check-design-tokens.py`
  - [x] `pnpm nx test web` 與 `pnpm nx test api`（⛔ 不用 `run_in_background`）
  - [x] 視覺基準若有變動，走四步流程

## Dev Notes

### 這張為什麼要提前

dsr-1／3／5／7／8 的條目都有一句「有 raw shadow-*，要逐一判定是不是真的浮層」。那個判定要有判準，而判準現在自相矛盾。先跑這張，後面九張的陰影題就變成查表。

### 量測方法（可重現）

```bash
# 排除註解與 token 內文之後數 raw 裸值
python3 - <<'PY'
import re, pathlib
raw=tok=0
for p in pathlib.Path('apps/web/src').rglob('*.tsx'):
    if p.name.endswith('.spec.tsx'): continue
    t=p.read_text(encoding='utf-8')
    t=re.sub(r'/\*.*?\*/','',t,flags=re.S); t=re.sub(r'^\s*//.*$','',t,flags=re.M)
    tok+=len(re.findall(r'shadow-\[var\(--shadow-[a-z]+\)\]',t))
    raw+=len(re.findall(r'(?<![\w\-\[])shadow-(?:sm|md|lg|xl|2xl|inner|none)\b',t))
print(raw, tok)
PY
```

⚠️ **不要用天真的 `grep shadow-md`**——它會把 `shadow-[var(--shadow-md)]` 裡的 `shadow-md` 也算進去，還會數到註解。立案時的「51 vs 12」就是這樣來的。

### 不要做的事

- **不要把 J1–J9 當畫面改。** 它們是裁定書，記的是當時的決定與理由。裁定過期要加註記，不是改寫歷史。
- **不要在裁定之前動任何一處陰影。** 三題互相牽動：裁定一決定白名單、裁定二決定 `--shadow-md` 還有沒有用途、裁定三決定寫法。先做任何一項都會回頭重做。
- **不要碰 `components/ui/` 的檔頭**（AC #7，立案誤判）。

### Source tree

```
DESIGN.md                                       ← Task 2（主要）
apps/web/src/components/ui/Button.tsx           ← Task 3（裁定一）
apps/web/src/components/requests/RequestButton.tsx ← Task 3（裁定一）
apps/web/src/components/library/PosterCardV2.tsx   ← Task 3（裁定二）
apps/web/src/components/dashboard/RecentMediaPanel.tsx ← Task 3（真違規）
apps/web/src/components/media/MediaDetailPanel.tsx     ← Task 3（真違規，v1）
apps/web/src/components/media/PosterCard.tsx           ← Task 3（真違規，v1）
apps/web/src/styles.css                         ← 只讀，token 定義在 118-121 / 282-285
ux-design.pen（V2Kez 與 J1–J9）                  ← Task 4, 5，只能用 Pencil MCP
```

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads the wall clock?**
  - **NO** — 這張只碰陰影 class 與文件。`Button`／`RequestButton`／`PosterCardV2` 都不讀時鐘。
  - 不新增視覺夾具；但裁定一或二若執行，**既有**夾具的基準會變（`ui-button`、`request-button/*`、海報相關），要走四步流程。
- Reference: `project-context.md` Rule 23；`project_visual_baseline_intentional_change`。

### References

- [Source: DESIGN.md#Elevation & Depth L484-500] — §Shadow Vocabulary 與 §Tone-First Rule
- [Source: DESIGN.md#Buttons L632] — Primary／Secondary 帶 `--shadow-sm`
- [Source: DESIGN.md#Cards and Containers L650] — 白名單 Dialog／Sheet／Popover／Toast
- [Source: `apps/web/src/styles.css:118-121`（夜行）／`:282-285`（日巡）] — 四個 token 的實值；日巡是兩層墨、夜行是單層煤黑
- [Source: `apps/web/src/components/library/PosterCardV2.tsx:95-100`] — 註解與程式碼不一致，且「11 uses」已過期
- [Source: `ux-design.pen` `Design Language v2 (V2Kez)`] — 掃過整個 frame，**零個**提到陰影的文字節點
- [Source: `sprint-status.yaml` → `dsr-1`／`dsr-3`／`dsr-5`／`dsr-7`／`dsr-8`] — 五張都在等這個判準
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-14

### Debug Log References

- `pnpm nx test web`：**3354 / 3354 passed**（+7 新規則的 spec）
- `pnpm nx test api`：PASS
- `pnpm run lint`：**0 errors**、128 warnings（改動前 127，+1 是新規則檔自己的 `no-var-requires` disable，與三個同門規則的 spec 同款）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `python3 scripts/check-design-tokens.py`：一致（188 畫面／72 母版／clipping 67）
- 匯出：188 張，**只有 2 張**有 byte 差異（`design-language-v2`、`j2-d`）

### Completion Notes List

- ⚖️ **Alexyu 2026-09-14 三個裁定，全部照建議走**：按鈕拿掉陰影／海報磚拿掉陰影／裸值全面改 token。
- 🔗 **AC Drift: FOUND（但這張就是來解它的）**。`dsr-11` 與 `dsr-12` 都撞到過 §Buttons 與 §Cards 的矛盾並把它路由到這裡；`dsr-12` 移除 `ui/Card.tsx` 的陰影時也留下了 §Shadow Vocabulary 的 `--shadow-md` 條目變成孤兒。本 story 一次收斂，沒有新的下游漂移。
- 📎 **Contract Stamps: NONE**（DESIGN.md 不帶 `[@contract-v*]` 標記）。
- 🎭 **A11y Pre-Flight: PASS**。觸到的元件只改 class 字串與註解，沒有新增圖片／aria-modal／非同步揭露內容／自訂 widget。⚠️ 陰影拿掉**不影響對比**——陰影從來不承擔文字對比，邊界由髮絲線與色階承擔，兩者都沒動。
- **最終狀態**：裸值 **0**、token **47**（`xl` 32、`lg` 15）。`sm`／`md` 在程式碼中零使用，與新的 §Shadow Vocabulary 一致。
- **普查更正**：立案寫「51 vs 12」。那個 grep 把 `shadow-[var(--shadow-sm)]` 裡的 `shadow-sm` 也算成裸值、也數到註解。重數後是 **46 vs 10**。更重要的是分類結果與假設相反——46 個裡**約 42 個位置本來就合法**，只是寫法錯。
- **四個真違規全部移除**（不是轉 token）：`RecentMediaPanel.tsx:130` 按鈕、`MediaDetailPanel.tsx:141` 圖示容器與 `:159` `<img>` 海報、`PosterCard.tsx:152` 卡片 hover。後三者住在 v1 元件，dsr-2 之後不需要再處理陰影。
- **新規則 `local/no-raw-shadow`** 接上 `eslint.config.mjs`，並附 7 條 spec（含「ring 與 drop-shadow 不管」「variant 前綴要抓得到」兩條邊界）。上線前用暫時檔實測會擋，測完刪除。
- ✅ **J1–J9 逐張查過裁定是否仍成立**：九張都沒有引用被取代的陰影規則。J2-4 的「searching 顏色」裁定**與 dsr-11 同向**（它早就裁定 `--accent-text`），不是被取代——但那張的 After 標籤寫著「（藍）」而值是金色 `#e0be72`，是 signal-blue 時期的殘留，已改成「（金）」，並在裁定註記補一行說明 dsr-11 已把請求管線一併改齊。
- ✅ **`.pen` 的 Design Language v2 補上 §9 抬升與陰影**。原本掃過整個 frame 一個提到陰影的文字節點都沒有——在 Pencil 裡畫圖的人沒有陰影規格可循，Flow M 的登入卡就是這樣掛上一道不該有的陰影的。
- 📝 **踩到的坑**：`.pen` 的 text 節點要換行必須同時給 `textGrowth: "fixed-width"`，只給 `width` 會被忽略（節點讀回來 `width` 是 `undefined`）。三個新文字節點因此一開始溢出，補上之後 DL-v2 溢出 0。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，一項，lane ③：
    - **③→① 日巡的陰影對 42 個裸值是失效的。** 裁定三選了 (a) 收斂成 token，**這一項因此就地解決**，不需要獨立立案。原文保留供追溯： `styles.css:282-285` 那次重做（墨色、兩層、總墨量 ≤15.4%）只改到 token，裸值完全沒跟上，所以日巡下那 42 個地方仍是 Tailwind 預設的黑色陰影——正是 DESIGN.md 說「會讀成一塊瘀青」的那種。裁定三 (a) 會順帶解決；若裁定 (b)，這一項必須獨立立案。**待立案：裁定 (b) 時開 `disc-2026-09-daywalk-shadow-broken-on-raw-values: backlog` 並在此列回填 ID。**
- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/web/src/eslint-rules/no-raw-shadow.js` — 擋住裸值的規則
- `apps/web/src/eslint-rules/no-raw-shadow.spec.ts` — 7 條 spec

**修改（文件與設定）：**
- `DESIGN.md` — §Shadow Vocabulary 重寫（三階＋三個裁定＋普查數字）、§Buttons 兩條改「無陰影」、§Cards 白名單補齊並註明無例外
- `eslint.config.mjs` — 接上新規則

**修改（程式碼，共 52 處）：**
- 42 處裸值 → token（橫跨 33 檔）
- 4 處真違規移除：`dashboard/RecentMediaPanel.tsx`、`media/MediaDetailPanel.tsx`（×2）、`media/PosterCard.tsx`
- 6 處依裁定移除：`ui/Button.tsx`（×3）、`requests/RequestButton.tsx`（×1，底部 toast 的 `xl` 保留）、`library/PosterCardV2.tsx`（×1，同時更新那段過期的註解）

**修改（設計）：**
- `ux-design.pen` — `Design Language v2` 新增 §9 抬升與陰影；J2-D 的 After 標籤「藍」→「金」並補 dsr-11 註記
- `_bmad-output/screenshots/design-system/design-language-v2.png`、`flow-j-specs/j2-d.png`
- `_bmad-output/pen-tokens.json`、`sprint-status.yaml`

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-14 | ⚖️ Alexyu 三個裁定全部照建議：按鈕無陰影／海報磚無陰影／裸值改 token。 |
| 2026-09-14 | Task 2 — DESIGN.md 三處自相矛盾一次收斂，並把普查數字與裁定理由寫進 §Shadow Vocabulary。 |
| 2026-09-14 | Task 3 — 42 處裸值改 token、4 處真違規移除、6 處依裁定移除。裸值歸零，token 47（xl 32／lg 15）。 |
| 2026-09-14 | Task 3 — 新增 `local/no-raw-shadow` 並接上 eslint.config.mjs，附 7 條 spec；上線前用暫時檔實測會擋。 |
| 2026-09-14 | Task 4 — `.pen` 的 Design Language v2 補 §9 抬升與陰影（原本完全沒有陰影規格）。 |
| 2026-09-14 | Task 5 — J1–J9 逐張查裁定，九張都仍成立；J2-4 的 After 標籤「藍」→「金」（signal-blue 殘留），並補 dsr-11 同向的註記。 |
| 2026-09-14 | Task 6 — 閘門：web 3354/3354、api PASS、lint 0 errors、typecheck PASS、token 一致、prettier 通過。 |
| 2026-09-14 | ✅ 收單 —— PR #421 合併進 main（commit 5805b189），CI 17 項全綠。📌 **視覺回歸全數通過，包括 `ui-button` 與 `request-button` 的基準線**——拿掉 `--shadow-sm` 之後像素比對量不出差別，這是對「那道陰影幾乎不可觀測」的獨立佐證，不是我們自己的測量。 |
