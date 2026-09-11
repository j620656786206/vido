# Story DSR.12: Flow M 登入閘——程式碼追回 M1–M7 設計稿

Status: review

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

- [x] **Task 1 — 兩個檔頭改成誠實的（AC: #1）**
  - [x] `LoginForm.tsx` 第一行改為 `// Design ref: ux-design.pen Screen M1-D (I6UAK) · M2-D (ihLvG) · M3-D (bfIZd) · M6-M (e2CuFg)`
  - [x] `LogoutButton.tsx` 第一行改為 `// Design ref: ux-design.pen Screen M4-D (aJYv6) · M7-M (YKYhH)`
  - [x] 🚨 **檔頭那一行必須以 `(節點ID)` 結尾。** `local/implements-pen-node-id` 的 regex 是 `^Design ref:\s*ux-design\.pen\s+Screen\s+.+\s+\([A-Za-z0-9]+\)$`——在節點 ID 後面接任何中文說明都會讓整行不匹配，8 個檔案就是這樣在 2026-09-11 把 PR #411 的 Lint 弄紅的。要補充說明就開第二行 `//` 註解。
  - [x] `LoginForm.tsx` 現有的 `// Time-bomb-exempt:` 區塊在檔頭之上，**保留原位**（Rule 23 標記，與 Rule 21 檔頭並存）

- [x] **Task 2 — 字級與斷點（AC: #2, #3）**
  - [x] 3 處 `text-[11px]` → `text-xs`：Subtitle（`NAS 媒體庫`）、錯誤建議行、`password-help` 說明行
  - [x] Wordmark `text-2xl` → `text-xl sm:text-2xl`
  - [x] 跑 `pnpm nx test web` 確認 `LoginForm.spec.tsx` 的 11 個既有測試仍過（要只跑這一支：`pnpm nx test web -- LoginForm`，vitest 吃位置參數當檔名過濾，沒有 `--testPathPattern`）

- [x] **Task 3 — 卡片去陰影（AC: #4）**
  - [x] 移除卡片容器 class 裡的 `shadow-[var(--shadow-md)]`，其餘 class 不動
  - [x] 順手確認 `sm:p-8` 與 `p-6` 保留——DESIGN.md 說卡片內距 24px，`p-6` 正是 24px

- [x] **Task 4 — 讓 Flow M 的稿讀得到（AC: #5）**
  - [x] `scripts/export-pen-screenshots.py` 的 `READABLE_FLOWS` set 加入 `"flow-m-auth-gate"`
  - [x] 執行 `python3 scripts/export-pen-screenshots.py`（需 Pencil.app 在跑）
  - [x] `git add` 只加 `_bmad-output/screenshots/flow-m-auth-gate/*.png` 這 7 張，其餘 `git checkout -- _bmad-output/screenshots/`
  - [x] 順手確認 `_bmad-output/pen-tokens.json` 是否被腳本更新；若有變動一起 commit（`check-design-tokens.py` 會擋 stale 快照）

- [x] **Task 5 — 登入卡進視覺回歸（AC: #6）**
  - [x] `-gallery.fixtures.tsx` 新增這個夾具，**四個欄位一個都不能少**（`penNode` 是 `GalleryFixture` 的必填欄位，漏了會 typecheck 紅）：
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
  - [x] `statesOnly: ['default']` **是必要的，不是省事**。預設會渲染 `['default','hover','focus']` 三態，而 `focus` 態會踩到 Rule 23 的豁免前提（見下方 Time-dependent 段），且 `LoginForm` 本身有 `autoFocus`，`focus` 態的基準線會與 `default` 難以區分。
  - [x] **不要**設 `routePath`。`StubRoutePath` 目前只接受 `/library` `/downloads` `/pending` `/settings`，沒有 `/login`；而 `LoginForm` 只在**登入成功後**才呼叫 `useNavigate`，`default` 態的截圖不會走到那條路徑，gallery 本身又已經在真正的 router 裡面。加 `routePath` 等於要先擴充那個 union，是沒有必要的改動。
  - [x] `penNode` 填 `'screen-section'`（全檔已有 122 個夾具這樣填）。M1-D 是 screen frame 不是 Reusable Component，不適合填裸節點 ID。
  - [x] 本機產 `-darwin` 基準線：先 `npx nx serve web`，另一個終端 `CI=1 npx playwright test --project=visual --update-snapshots=missing`（`CI=1` 會清空 playwright 的 `webServer` 清單，避免它去啟動需要 `GEMINI_API_KEY` 的 Go 後端）
  - [x] ⛔ **不要 commit 任何 `-linux.png`**；推上去後等 CI 自動開 bootstrap PR 再合併

- [x] **Task 6 — 收尾驗證（AC: #7, #8）**
  - [x] `pnpm run lint` → 0 errors
  - [x] `pnpm nx run web:typecheck`
  - [x] `pnpm run format:check`（本機若有 `apps/api/coverage/` 或 `testsprite_tests/` 的未追蹤檔會噪音報 warn，那些 CI 不會有，看的是**有被 git 追蹤的檔**）
  - [x] `python3 scripts/check-design-tokens.py`
  - [x] `pnpm nx test web`（全套回歸；⛔ 絕不用 `run_in_background` 跑測試，會留下孤兒 vitest worker）

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

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-11

### Debug Log References

- `pnpm nx test web`（RED，Task 2/3 實作前）：3 failed / 3339 passed — 新寫的 3 個測試如預期全紅
- `pnpm nx test web`（GREEN，Task 1–3 實作後）：**3342 / 3342 passed**
- `pnpm nx test api`：PASS
- `pnpm run lint`：**0 errors**、127 warnings（與改動前同數，本 story 沒有引入新的 warning）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `python3 scripts/check-design-tokens.py`：一致（82 變數／187 畫面／72 母版）
- `CI=1 npx playwright test --project=visual`：`auth-login-form` 綠；另有 3 個既有紅，見下方 Completion Notes

### Completion Notes List

- 🔗 **AC Drift: N/A**（`grep -rln "LoginForm|密碼閘|VIDO_AUTH_PASSWORD" _bmad-output/implementation-artifacts/*.md` 只命中本 story 自己。密碼閘是 PR #365 直接出貨的，沒有走 story 流程，所以沒有前置 AC 可以漂移。）
- 📎 **Contract Stamps: NONE**（本 story 沒有 `[@contract-v*]` 標記，Dev Notes 也沒有引用任何上游標記 AC——這對不定義也不消費 wire contract 的 story 是正常的。）
- 🎭 **A11y Pre-Flight: PASS**（2 個 component 檢查：`LoginForm.tsx`、`LogoutButton.tsx`。這兩個檔案的 jsx-a11y warning 數 0，本 story 引入 0 個。四類回歸項逐一確認：本 story 沒有新增圖片、沒有新增 aria-modal、沒有新增非同步揭露內容、沒有新增自訂 widget——只改了 class 字串與檔頭註解。`LoginForm` 既有的 `autoFocus` eslint-disable 與 `aria-hidden` 倒數行**原封不動**，兩條各自的 spec 仍綠。）
- ⚠️ **Pre-existing failure — FILED**（不是 fix）：`CI=1 npx playwright test --project=visual` 在 darwin 上有 3 個夾具紅（`parse-floating-parse-progress-card`、`retry-retry-notifications`、`glossary-panel-v2/seeded`）。**用 `git stash` 對照驗證過**：把本 story 的四個原始碼改動全部 stash 掉重跑，仍然是同樣 3 個紅，所以與本 story 無關。三條 `-darwin` 基準線最後更新是 PR #168（兩個）與 PR #146。CI 走 `-linux` 且 PR #411 四個 shard 全綠，產品沒壞。選 FILE 不選 FIX 的理由：diff 顯示側軌「儲存空間」讀數是 `–`（本機沒跑 Go 後端），直接 `--update-snapshots` 會把降級狀態烤進基準線，讓 darwin 跟 linux 差更多——要先裁定基準線該在哪種環境生成。立案：`preexisting-fail-visual-darwin-three-stale-baselines`。
- ✅ Task 1：兩個檔頭改成指向真正畫了它們的 screen frame。踩到 story 已預警的地雷零次——節點 ID 後面的中文說明一律另起第二行。
- ✅ Task 2：3 處 `text-[11px]` → `text-xs`；wordmark `text-2xl` → `text-xl sm:text-2xl`。
- ✅ Task 3：卡片的 `shadow-[var(--shadow-md)]` 移除。DESIGN.md 說全 app 只有 2 個地方用這個 token，這是其中一個。
- ✅ Task 4：`flow-m-auth-gate` 進 `READABLE_FLOWS`，重匯 187 張，**只有 Flow M 那 7 張有 byte 差異**，零重匯噪音，`pen-tokens.json` 未變動。
- ✅ Task 5：`auth-login-form` 夾具加入（`statesOnly: ['default']`），`-darwin` 基準線已生成並確認內容正確（無陰影、髮絲邊框、灰色停用按鈕、12px 說明行）。**沒有** commit 任何 `-linux.png`。
- 📝 **順手修正一條會過期的註解**：`LoginForm.tsx` 的 `Time-bomb-exempt` 標記原文寫「there is no login gallery fixture」，Task 5 之後那句話就不成立了。改寫成說明「夾具存在但只有 default 態，讀時鐘的分支仍然摸不到基準線，所以豁免仍成立；加 error/locked 態就會失效」。這是讓 Rule 23 標記維持誠實，不是擴大範圍。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，四項。實作後的最終分類如下（前兩項是 create-story 時就預見的，後兩項是實作中撿到的）：
    - **③ `dsr-12` 條目的範圍少算了 `LogoutButton.tsx`。** 立案時寫「components/auth/**（只有 2 檔）」，但 M4-D／M7-M 是 `components/shell/LogoutButton.tsx` 畫的。本 story 已就地吸收（見 AC #1、Task 1），**不需要**另開條目——這其實是 lane ① expand-scope-in-place，由 AC #1 追蹤。
    - **③ `flow-m-auth-gate` 之外還有 9 個 flow 資料夾仍是 400px 縮圖。** Task 4 只修 Flow M 這一個。其餘 flow（a/b/d/h/i/k/l 等）在各自的 `dsr-*` story 做同一件事時會逐一補上，每一張 dsr story 都帶著自己的那一份——所以**追蹤載體是 `epic-dsr` 底下那 12 筆既有 backlog 條目**，不需要另開。
    - **③ darwin 視覺基準線有 3 張是舊的。** 已用 `git stash` 對照確認與本 story 無關。立案 `preexisting-fail-visual-darwin-three-stale-baselines: backlog`（雙向：該條目寫明由 dsr-12 立案）。
    - **③ M1-D／M6-M 的 Input 沒有畫出 focus ring，但實機預設態一定是聚焦的**（`autoFocus`，有 spec 與 eslint-disable 在守）。方向是**稿要補、碼不動**——拿掉 autoFocus 會退掉一個無障礙決定。立案 `disc-2026-09-m1d-missing-autofocus-ring: backlog`（雙向：該條目寫明由 dsr-12 立案）。
- Reference: `project-context.md` Rule 24

### File List

**修改：**
- `apps/web/src/components/auth/LoginForm.tsx` — Rule 21 檔頭、Rule 23 標記措辭、卡片去陰影、wordmark 斷點、3 處字級
- `apps/web/src/components/auth/LoginForm.spec.tsx` — 新增 3 個測試（卡片形狀、字階、wordmark 斷點）
- `apps/web/src/components/shell/LogoutButton.tsx` — Rule 21 檔頭（只有第一行＋一行說明）
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 新增 `auth-login-form` 夾具＋`LoginForm` import
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` 加入 `flow-m-auth-gate`
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — dsr-12 狀態流轉＋2 筆新立案
- `_bmad-output/screenshots/flow-m-auth-gate/{m1-d,m2-d,m3-d,m4-d,m5-d,m6-m,m7-m}.png` — 7 張改用可讀解析度重匯

**新增：**
- `tests/visual/components.visual.spec.ts-snapshots/components/auth-login-form/default-visual-darwin.png`

**待 CI 產生（不得本機 commit）：**
- `tests/visual/components.visual.spec.ts-snapshots/components/auth-login-form/default-visual-linux.png`

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-11 | Task 1 — `LoginForm.tsx` / `LogoutButton.tsx` 的 Rule 21 檔頭從 `<utility — no .pen counterpart>` 改成指向 M1/M2/M3/M6 與 M4/M7。lint 0 errors。 |
| 2026-09-11 | Task 2 — 3 處 `text-[11px]` → `text-xs`（＝`$Type/Label/Size` 12）；wordmark `text-2xl` → `text-xl sm:text-2xl`（＝`$Type/H2/Size` 手機 20／桌機 24）。 |
| 2026-09-11 | Task 3 — 卡片移除 `shadow-[var(--shadow-md)]`，對齊 DESIGN.md §Cards 2026-09-11 裁定與 `.pen` Login Card 的空 `effects`。 |
| 2026-09-11 | Task 4 — `flow-m-auth-gate` 進 `READABLE_FLOWS`；7 張 Flow M 設計稿從 400px 縮圖改為 2x／最小寬 1400，終於讀得到 12px 的字。 |
| 2026-09-11 | Task 5 — 新增 `auth-login-form` 視覺夾具（`default` 態）與 `-darwin` 基準線；同步把 `Time-bomb-exempt` 註解改成夾具存在之後仍然成立的版本。 |
| 2026-09-11 | Task 6 — 全套閘門：web 3342/3342、api PASS、lint 0 errors、typecheck PASS、token 一致、prettier 通過。 |
| 2026-09-11 | 立案 2 筆：`preexisting-fail-visual-darwin-three-stale-baselines`（既有、已用 stash 驗證）、`disc-2026-09-m1d-missing-autofocus-ring`（設計稿側）。 |
