# Story DSR.12: Flow M 登入閘——程式碼追回 M1–M7 設計稿

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a self-hoster who is turned away at the password gate,
I want 登入畫面的字級、卡片形狀與說明文字跟設計稿是同一份事實,
so that 我第一眼看到的這個畫面不是全 app 唯一沒有人在守的那一張。

## Context

`epic-dsr` 的第一張 story，也是規模最小的一張（2 個 tsx 檔）。目的有兩個：**把 Flow M 的程式碼追回設計稿**，同時**驗證 dsr 系列的驗收格式**——後面 12 張 story 會照這張的形狀寫。

PR #411（2026-09-11）逐一修完 187 張設計稿之後，設計稿變成新事實、程式碼還停在舊事實。Flow M 的具體落差在下面的「設計稿 vs 程式碼對照表」，每一行都是從 `.pen` 節點屬性與程式碼原文直接讀出來的，不是印象。

⚠️ **這張 story 的驗收基準是 `.pen` 的節點值，不是 `_bmad-output/screenshots/flow-m-auth-gate/` 底下的 PNG。** 那 7 張 PNG 現在是 400px 縮圖，12px 的字渲染成 3.3px，人眼讀不到——Task 4 就是修這件事（`flow-e-scanner` 在 2026-09-11 因為同一個理由被加進 `READABLE_FLOWS`，這裡照抄那個判例）。

## Acceptance Criteria

### 設計稿 vs 程式碼對照表（本 story 的事實基礎）

| 元素 | `.pen` 值 | 程式碼現況 | 判定 |
| --- | --- | --- | --- |
| Login Card 形狀 | `$bg-secondary` + `$border-subtle` 描邊 + `$radius-lg`，**`effects` 為空** | `shadow-[var(--shadow-md)]` | ❌ 多一道陰影 |
| Wordmark `vido` | `$Type/H2/Size` = 桌機 24 / 手機 20 | `text-2xl`（24 固定） | ❌ 手機沒降級 |
| Subtitle `NAS 媒體庫` | `$Type/Label/Size` = **12** | `text-[11px]` | ❌ 差 1px 且是任意值 |
| 錯誤建議行（M2 Err Line2） | `$Type/Label/Size` = **12** | `text-[11px]` | ❌ 同上 |
| 說明行 Help1/EnvVar/Help2 | `$Type/Label/Size` = **12** | `text-[11px]` | ❌ 同上 |
| 標籤「密碼」 | `$Type/Body/Size` = 14 | `text-sm` | ✅ |
| Placeholder「輸入密碼」 | Body 14 | `text-sm` | ✅ |
| 錯誤主行（M2 Err Line1） | Body 14、`$error-text` | `text-sm text-[var(--error-text)]` | ✅ |
| 按鈕文字 | Body 14 | `text-sm` | ✅ |
| M3 鎖定按鈕底色 | `$bg-tertiary` + `$text-muted` | `disabled:bg-[var(--bg-tertiary)] disabled:text-[var(--text-muted)]` | ✅ |
| M4 對話框文案 | 「要登出嗎？」／「下次進來要再輸入密碼。」／取消／登出 | 逐字相同 | ✅ |
| M7 sheet 順序 | 導覽在上、狀態帶在中、登出在下 | `MobileMoreSheet` 先 `MORE_DESTS` 後 `SidebarFooter` | ✅ |
| M5 等待面 | 只有 wordmark，**刻意沒有 spinner** | `__root.tsx:147-170` 同樣只有 wordmark | ✅ |
| `LoginForm.tsx` Rule 21 檔頭 | M1-D / M2-D / M3-D / M6-M 都存在 | `// Implements: <utility — no .pen counterpart>` | ❌ 說謊 |
| `LogoutButton.tsx` Rule 21 檔頭 | M4-D / M7-M 都存在 | `// Implements: <utility — no .pen counterpart>` | ❌ 說謊 |

**節點 ID（Task 1 要逐字抄進檔頭，不要重查）：**
`M1-D (I6UAK)` · `M2-D (ihLvG)` · `M3-D (bfIZd)` · `M4-D (aJYv6)` · `M5-D (fMJFJ)` · `M6-M (e2CuFg)` · `M7-M (YKYhH)`

---

1. **Rule 21 檔頭誠實化。** `apps/web/src/components/auth/LoginForm.tsx` 與 `apps/web/src/components/shell/LogoutButton.tsx` 的第一行不得再宣稱 `<utility — no .pen counterpart>`；兩者各自指向真正畫了它的 screen frame。`MobileMoreSheet.tsx` 的 `// Implements: Component/MobileMoreSheet (mfDKV)` **已驗證該節點仍存在，不要動它**。

2. **字級改用字階。** `LoginForm.tsx` 的 3 處 `text-[11px]` 全部改成 `text-xs`（Tailwind v4 預設 12px，等於 `$Type/Label/Size`）。全 app 已有 99 個 component 檔在用 `text-xs`，這是既有慣例不是新發明。改完後 `grep -c 'text-\[11px\]' apps/web/src/components/auth/LoginForm.tsx` 必須是 0。

3. **Wordmark 跟著斷點降級。** `text-2xl` → `text-xl sm:text-2xl`（手機 20、桌機 24），對上 `$Type/H2/Size` 的 `{desktop:24, mobile:20}`。`__root.tsx` 等待面的 wordmark **不在本 story 範圍**（`routes/` 免 Rule 21，且 M5-D 只驗「沒有 spinner」，已通過）。

4. **卡片不再有陰影。** 移除 `LoginForm.tsx` 卡片容器的 `shadow-[var(--shadow-md)]`。依據是 DESIGN.md §Cards and Containers 2026-09-11 裁定：卡片＝12px 圓角＋`--bg-secondary`＋1px 髮絲邊框＋**無陰影**；夜行下 `--shadow-md` 對底色只有 1.056:1，髮絲線是 1.660:1。`.pen` 的 Login Card 節點 `effects` 為空，兩邊一致。

5. **Flow M 的設計稿要讀得到。** `scripts/export-pen-screenshots.py` 的 `READABLE_FLOWS` 加入 `"flow-m-auth-gate"`，重跑匯出，`_bmad-output/screenshots/flow-m-auth-gate/` 7 張 PNG 從 400px 長邊變成 2x／最小寬 1400。**只 stage 這 7 張**，其餘重匯的 PNG 用 `git checkout` 丟掉（全量重匯是非決定性的）。

6. **登入卡進視覺回歸。** `apps/web/src/routes/test/-gallery.fixtures.tsx` 新增 `auth-login-form` 夾具（`default` 態即可）。這是全 app 第一眼會看到的畫面，目前視覺覆蓋率為 0——AC #4 改了它的形狀卻沒有任何基準線會察覺。
   ⚠️ **`-linux` 基準線不能在本機產生**。照 CLAUDE.md 的四步流程：本機只跑 `-darwin`，推上去讓 CI 的 `Visual Regression` workflow 自動開 `chore(visual): bootstrap N missing -linux baselines` PR，合併它，不要自己造 `-linux.png`。

7. **既有測試全數保留通過。** `LoginForm.spec.tsx` 現有 11 個 `it()` 一個都不能刪或改行為斷言，特別是 `disables the submit button with tokens rather than opacity` 與 `does not re-announce the countdown every second`——那兩條是 Sally 的無障礙裁定，不是實作細節。

8. **CI 全綠。** `pnpm run lint`（0 errors）、`pnpm nx run web:typecheck`、`pnpm run format:check`、`python3 scripts/check-design-tokens.py` 四關都過。

## Tasks / Subtasks

- [ ] **Task 1 — 兩個檔頭改成誠實的（AC: #1）**
  - [ ] `LoginForm.tsx` 第一行改為 `// Design ref: ux-design.pen Screen M1-D (I6UAK) · M2-D (ihLvG) · M3-D (bfIZd) · M6-M (e2CuFg)`
  - [ ] `LogoutButton.tsx` 第一行改為 `// Design ref: ux-design.pen Screen M4-D (aJYv6) · M7-M (YKYhH)`
  - [ ] 🚨 **檔頭那一行必須以 `(節點ID)` 結尾。** `local/implements-pen-node-id` 的 regex 是 `^Design ref:\s*ux-design\.pen\s+Screen\s+.+\s+\([A-Za-z0-9]+\)$`——在節點 ID 後面接任何中文說明都會讓整行不匹配，8 個檔案就是這樣在 2026-09-11 把 PR #411 的 Lint 弄紅的。要補充說明就開第二行 `//` 註解。
  - [ ] `LoginForm.tsx` 現有的 `// Time-bomb-exempt:` 區塊在檔頭之上，**保留原位**（Rule 23 標記，與 Rule 21 檔頭並存）

- [ ] **Task 2 — 字級與斷點（AC: #2, #3）**
  - [ ] 3 處 `text-[11px]` → `text-xs`：Subtitle（`NAS 媒體庫`）、錯誤建議行、`password-help` 說明行
  - [ ] Wordmark `text-2xl` → `text-xl sm:text-2xl`
  - [ ] 跑 `pnpm nx test web` 確認 `LoginForm.spec.tsx` 的 11 個既有測試仍過（要只跑這一支：`pnpm nx test web -- LoginForm`，vitest 吃位置參數當檔名過濾，沒有 `--testPathPattern`）

- [ ] **Task 3 — 卡片去陰影（AC: #4）**
  - [ ] 移除卡片容器 class 裡的 `shadow-[var(--shadow-md)]`，其餘 class 不動
  - [ ] 順手確認 `sm:p-8` 與 `p-6` 保留——DESIGN.md 說卡片內距 24px，`p-6` 正是 24px

- [ ] **Task 4 — 讓 Flow M 的稿讀得到（AC: #5）**
  - [ ] `scripts/export-pen-screenshots.py` 的 `READABLE_FLOWS` set 加入 `"flow-m-auth-gate"`
  - [ ] 執行 `python3 scripts/export-pen-screenshots.py`（需 Pencil.app 在跑）
  - [ ] `git add` 只加 `_bmad-output/screenshots/flow-m-auth-gate/*.png` 這 7 張，其餘 `git checkout -- _bmad-output/screenshots/`
  - [ ] 順手確認 `_bmad-output/pen-tokens.json` 是否被腳本更新；若有變動一起 commit（`check-design-tokens.py` 會擋 stale 快照）

- [ ] **Task 5 — 登入卡進視覺回歸（AC: #6）**
  - [ ] `-gallery.fixtures.tsx` 新增這個夾具，**四個欄位一個都不能少**（`penNode` 是 `GalleryFixture` 的必填欄位，漏了會 typecheck 紅）：
    ```ts
    {
      id: 'auth-login-form',
      label: 'auth/LoginForm',
      component: LoginForm,
      penNode: 'screen-section',
      width: 420,
      statesOnly: ['default'],
    },
    ```
  - [ ] `statesOnly: ['default']` **是必要的，不是省事**。預設會渲染 `['default','hover','focus']` 三態，而 `focus` 態會踩到 Rule 23 的豁免前提（見下方 Time-dependent 段），且 `LoginForm` 本身有 `autoFocus`，`focus` 態的基準線會與 `default` 難以區分。
  - [ ] **不要**設 `routePath`。`StubRoutePath` 目前只接受 `/library` `/downloads` `/pending` `/settings`，沒有 `/login`；而 `LoginForm` 只在**登入成功後**才呼叫 `useNavigate`，`default` 態的截圖不會走到那條路徑，gallery 本身又已經在真正的 router 裡面。加 `routePath` 等於要先擴充那個 union，是沒有必要的改動。
  - [ ] `penNode` 填 `'screen-section'`（全檔已有 122 個夾具這樣填）。M1-D 是 screen frame 不是 Reusable Component，不適合填裸節點 ID。
  - [ ] 本機產 `-darwin` 基準線：先 `npx nx serve web`，另一個終端 `CI=1 npx playwright test --project=visual --update-snapshots=missing`（`CI=1` 會清空 playwright 的 `webServer` 清單，避免它去啟動需要 `GEMINI_API_KEY` 的 Go 後端）
  - [ ] ⛔ **不要 commit 任何 `-linux.png`**；推上去後等 CI 自動開 bootstrap PR 再合併

- [ ] **Task 6 — 收尾驗證（AC: #7, #8）**
  - [ ] `pnpm run lint` → 0 errors
  - [ ] `pnpm nx run web:typecheck`
  - [ ] `pnpm run format:check`（本機若有 `apps/api/coverage/` 或 `testsprite_tests/` 的未追蹤檔會噪音報 warn，那些 CI 不會有，看的是**有被 git 追蹤的檔**）
  - [ ] `python3 scripts/check-design-tokens.py`
  - [ ] `pnpm nx test web`（全套回歸；⛔ 絕不用 `run_in_background` 跑測試，會留下孤兒 vitest worker）

## Dev Notes

### 為什麼這張 story 存在

`epic-dsr` 的來源是 PR #411。那個 PR 只動設計稿不動程式碼——它把 187 張稿修對了，於是製造出一整批「稿是對的、碼是舊的」的落差。這張 story 是 13 張 flow 對齊 story 的第一張，選 Flow M 是因為它最小（2 個檔），**用來驗證驗收格式**。後面 12 張會照這張的「對照表 + 節點 ID 逐字列出」形狀寫。

### 不要做的事

- **不要重畫 `.pen`。** 這張 story 的方向是單向的：程式碼追設計稿。唯一動到設計產物的是 Task 4，而那是渲染解析度不是設計內容。
- **不要改 `LoginForm` 的行為。** 倒數計時、focus/select 回填、`aria-hidden` 的倒數行、disabled 用 token 不用 opacity——每一條都有 spec 在守，且註解裡寫了為什麼。這張 story 只碰**形狀與字級**。
- **不要碰 `MobileMoreSheet.tsx`。** 它的 Rule 21 檔頭指向 `Component/MobileMoreSheet (mfDKV)`，該節點已用 Pencil MCP 驗證仍然存在；順序也已對上 M7 Note。
- **不要動 `__root.tsx` 的等待面。** M5-D 只有一個 wordmark、刻意沒有 spinner（M5 Note 原文：「這裡沒有任何可量測的進度，spinner 會宣稱有」），程式碼已符合。`routes/` 檔案本來就免 Rule 21。

### 固定詞彙檢查（本 story 無新增用色）

M2 的錯誤主行用 `$error-text`（硃砂＝壞了），建議行用 `$text-secondary`（中性）。程式碼一致。**不要**把建議行改成赭色——赭色的定義是「你要求了但沒發生」，而「還可以再試 2 次」是純告知。

### Source tree

```
apps/web/src/components/auth/LoginForm.tsx        ← Task 1,2,3（主要）
apps/web/src/components/auth/LoginForm.spec.tsx   ← 只讀，不改
apps/web/src/components/shell/LogoutButton.tsx    ← Task 1（只改第一行）
apps/web/src/routes/test/-gallery.fixtures.tsx    ← Task 5
scripts/export-pen-screenshots.py                 ← Task 4（READABLE_FLOWS）
_bmad-output/screenshots/flow-m-auth-gate/*.png   ← Task 4 產出（7 張）
```

### Project Structure Notes

- `LogoutButton.tsx` 住在 `components/shell/` 不是 `components/auth/`，但它畫的是 M4-D／M7-M，所以屬於 Flow M 的範圍。`sprint-status.yaml` 的 `dsr-12` 條目原本只寫「components/auth/**（只有 2 檔）」，**那個範圍少算了 `LogoutButton.tsx`**——本 story 把它納入，這是唯一一處與立案時描述不同的地方。
- `components/auth/` 的兩個檔案裡，`LoginForm.spec.tsx` 是 spec，Rule 21 本來就豁免。所以「兩個檔案都沒有 Design ref 標頭」實際上是「一個該有的沒有」。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **YES** — `LoginForm.tsx` 讀 `Date.now()`（鎖定倒數）。但它**已經**帶著三種合法標記之一：檔案最上方的 `// Time-bomb-exempt:` 區塊，理由是那個 deadline 來自伺服器的 `Retry-After` header、只在 429 之後渲染，而 429 是沒有任何視覺基準線會捕捉到的狀態。
  - Task 5 新增的 `auth-login-form` 夾具是 `default` 態（空密碼、無錯誤、無鎖定），**不會進入讀時鐘的分支**，所以 `Time-bomb-exempt` 這個標記在夾具加入後仍然成立，不需要升級成 `Clock-mocked`，也不需要 ≥2 個 fixture 狀態基準線。
  - ⚠️ 如果 dev 決定額外加 `locked` 或 `error` 狀態的夾具（AC #6 只要求 `default`），那個豁免就失效，必須改用 AC #4 的 `withFixedClock(page, iso)` helper 並補 `clockTime` 夾具欄位。**建議不要加**，留在 `default`。
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`; canonical migration precedent: story 19-9 AC #5。

### References

- [Source: `ux-design.pen` Screen M1-D (I6UAK) / M2-D (ihLvG) / M3-D (bfIZd) / M4-D (aJYv6) / M5-D (fMJFJ) / M6-M (e2CuFg) / M7-M (YKYhH)] — 節點屬性經 Pencil MCP `execute` 直接讀出，非依賴截圖
- [Source: DESIGN.md#Cards and Containers] — 2026-09-11 裁定：卡片無陰影，附夜行 1.056:1 vs 1.660:1 實測表
- [Source: project-context.md#Rule 21] — Component-to-Design Node Traceability，Phase 2 已由 ESLint 強制
- [Source: project-context.md#Rule 23] — Time-Dependent Component Fixture Stability
- [Source: project-context.md#Rule 24] — Discovery Triage（本 story 的 triage 見下方 Dev Agent Record）
- [Source: `apps/web/src/eslint-rules/implements-pen-node-id.js`] — `DESIGN_REF_RE` 的完整 regex，Task 1 的地雷
- [Source: CLAUDE.md#UX Design Screenshots Workflow] — 全量重匯非決定性、只 stage 真的改了的 PNG
- [Source: CLAUDE.md#Git Workflow] — `-linux` 基準線只能由 CI 產生
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml` → `epic-dsr` / `dsr-12-flow-m-auth-gate`]

## Dev Agent Record

### Agent Model Used

（dev agent 填寫）

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，兩項，都走 lane ③（backlog、非阻塞、雙向連結）：
    - **③ `dsr-12` 條目的範圍少算了 `LogoutButton.tsx`。** 立案時寫「components/auth/**（只有 2 檔）」，但 M4-D／M7-M 是 `components/shell/LogoutButton.tsx` 畫的。本 story 已就地吸收（見 AC #1、Task 1），**不需要**另開條目——這其實是 lane ① expand-scope-in-place，由 AC #1 追蹤。
    - **③ `flow-m-auth-gate` 之外還有 9 個 flow 資料夾仍是 400px 縮圖。** Task 4 只修 Flow M 這一個。其餘 flow（a/b/d/h/i/k/l 等）在各自的 `dsr-*` story 做同一件事時會逐一補上；若想一次做完，請在 `sprint-status.yaml` 另開條目。**目前無獨立條目——dev 若決定不一次做完（建議不要），此列即為記錄。**
- Reference: `project-context.md` Rule 24

### File List

（dev agent 填寫）
