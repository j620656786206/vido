---
target: 登入 gate + app shell（含新增的 LogoutButton）
total_score: 16
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 3
timestamp: 2026-09-01T06-43-36Z
slug: apps-web-src-components-shell
---
Method: dual-agent (A: design_review · B: detector_evidence)

Target: V0.1.1 shared-password login gate + app shell（含本次新增的 LogoutButton）
Live env: http://localhost:8090（seeded test env, VIDO_AUTH_PASSWORD=vido1234）

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|---|---:|---|
| 1 | Visibility of System Status | 2 | 未登入開 `/` 會先渲染完整片庫殼層約 60ms 才被登入卡取代（實測）；60 秒鎖定從不倒數；全 app 無 401 攔截器 |
| 2 | Match System / Real World | 2 | 「請輸入密碼以繼續」從不說是哪一個密碼——`VIDO_AUTH_PASSWORD` 是這個受眾唯一擁有的詞彙 |
| 3 | User Control and Freedom | 2 | 無顯示密碼、gate 無出路；登出清空整個 query cache 卻無確認、無回饋 |
| 4 | Consistency and Standards | 2 | 登入卡違反圓角／深度／wordmark 三條，而三像素外的側欄三條都對；shell 每個觸控目標精準 44px，登入頁兩個主控制項都不到 |
| 5 | Error Prevention | 1 | 無顯示密碼、無 Caps Lock 提示、無剩餘次數；登出旁 44px 就是外觀完全相同的主題列 |
| 6 | Recognition Rather Than Recall | 1 | 復原路徑（改 `.env` 重啟容器）以字串存在於 `auth_gate.go:135`，從未被渲染 |
| 7 | Flexibility and Efficiency | 2 | 單欄位 gate 無 autofocus；錯誤插在欄位上方，實測把輸入框往下推 31px |
| 8 | Aesthetic and Minimalist Design | 3 | 殼層真的密而簡；gate 的「簡」是漏掉而非編輯出來的 |
| 9 | Error Recovery | 1 | `authService.ts:37` 只讀 `error.message`，後端的 `suggestion` 與 `Retry-After: 60` 全部丟棄 |
| 10 | Help and Documentation | 0 | gate 上零說明：沒有連結、沒有提示、沒提 `.env`、沒說這是哪台 NAS |
| **Total** | | **16/40** | 需要注意：**單獨評 shell 約 29–30/40**。這個分數幾乎全部由 gate 拉下來 |

## Design Specificity Verdict

**登入畫面：category-interchangeable。** 實測卡片底色 `#FAF6EA` vs 頁面底色 `#FAF6EA`，對比 **1.00** — 它與頁面的唯一分隔是 `shadow-2xl` 加一條 50% 透明髮絲線，違反 DESIGN.md「一張卡片之所以是卡片，是因為它比頁面亮一階」。圓角 16px（`--radius-xl`，文件註明實質未使用）。Wordmark 是大寫 `Vido`，而 PRODUCT.md 承諾小寫 `vido` + 副標「NAS 媒體庫」，側欄做對了 — 同一個產品，兩種拼法，相隔三個像素。畫面上零字提到 NAS、片庫或繁中字幕。

**App shell：authored，而且強。** active nav 底色對側欄 1.17:1（15% 淡洗而非實心填滿）；「需要注意」顯示「一切正常」而不是 0；`0/18` 用等寬；TMDb 降級同時交代原因、範圍與修法。這是「誠實的讀數」，做到了。

**最鋒利的對照：** `DatabaseUnavailableBanner` 為資料庫掛掉寫了 42 個字（原因＋影響＋去 NAS 檢查哪裡＋系統每 30 秒重試）。同一個專案對密碼錯誤寫了 4 個字。

## Deterministic scan

`detect.mjs` → exit 2，**11 findings，全部同一條 `design-system-font-size`（advisory），零 contrast／零 touch-target／零 a11y 命中。**

- 11 個命中裡 10 個是 11px chrome 標籤 = **FALSE POSITIVE**：11px 在 `_bmad-output/planning-artifacts/home-v3-pen-prompt.md` 被明文指定，只是沒進 DESIGN.md 的 type ramp。**要修的是 DESIGN.md，不是元件。**
- `InFlightBadge.tsx:51` 的 **10px = GENUINE**（advisory）：找不到任何授權來源，屬既有漂移，不在本次 diff 內。
- 本次新增檔案只佔 1 個命中（`LogoutButton.tsx:69`），且與它旁邊的 `ThemeToggle.tsx:99` 完全一致 — 新按鈕遵守了現有房規。
- `--text-disabled` 在受檢範圍內唯一用法是空心圈邊框，**未違規**，detector 也沒誤報。

## Priority Issues

### [P0] 後端已經算好的三個真實數字，前端全部丟掉

1. `authService.ts:37` 只讀 `data.error?.message`。後端 `ErrorResponse` 的第四個參數 `suggestion`（`response.go:22,62`）落地：「請確認密碼後再試一次。」與「密碼錯誤太多次,請稍後再試。」抓下來就扔。
2. `auth_handler.go:54-55` 回 429 並設 `Retry-After: 60`。`grep -rn "Retry-After\|TOO_MANY" apps/web/src` → **各 0 命中**。60 秒鎖定在 UI 上不存在，登入按鈕 `disabled` 仍為 false。
3. 復原路徑（「密碼是部署時設定的 `VIDO_AUTH_PASSWORD`」）一字不差寫在 `auth_gate.go:135` 的 401 detail 裡，從未被渲染。

**Why it matters.** 北極星是「誠實的讀數」，硬規則是「不渲染系統沒在量測的數字」。**「量到了卻不渲染」是同一條規則的另一半**，而這裡三個都量到了。對 NAS 自架玩家而言，`docker compose down && vi .env && up -d` 是他百分之百做得到的事，而他絕不會從「密碼錯誤」四個字猜到那就是官方答案。

**Fix.** 帶出 `suggestion` 當第二行；讀 `Retry-After` 並在鎖定期間 disable 按鈕＋跑真實倒數（遞減中的真數字，完全符合既有紀律）；第 3 次失敗後顯示剩餘次數（`auth_ratelimit.go` 已經在數）；按鈕下方常駐一行 12px 說明，env 名稱用等寬。

**Suggested command:** `/impeccable clarify`

### [P1] 手機上登入框和登入鈕都碰不滿 44px；disabled 登入鈕 2.54:1

實測 390px viewport：密碼輸入框 **318×42**、送出鈕 **318×40**。同一份 PR 的 shell 側每一個目標都精準守住 44（`min-h-[44px]` / `h-11 w-11`）。**沒守住的兩個都在 `LoginForm.tsx`，而那是手機使用者第一個要點的東西。**

disabled 送出鈕：`#FDFAF2` @ opacity 0.5 on `#886208` = **2.54:1**（門檻 4.5）。空密碼是這頁的**起始狀態**，所以第一眼看到的主 CTA 就不合格。這不受 `--text-disabled` 那條已文件化例外保護（該 token 沒用在這裡），是 `disabled:opacity-50` 把一個合格的 5.30:1 壓下去的結果。

**Fix.** `py-2.5` → `min-h-[44px]`；disabled 態改用實色 token 而非 opacity。

**Suggested command:** `/impeccable adapt`

### [P1] 未登入開 `/` 會先閃過整個片庫殼層

實測（1440×900，cold load `/`）：

| t | body length | shell | login | 可見文字 |
|---|---|---|---|---|
| 0–30ms | 31 | — | — | （空） |
| **60ms** | **27475** | **yes** | no | `vido NAS 媒體庫 內容 首頁 媒體庫 電影 影集 探索 任務 活動 下載 設定 切換到夜行 儲存空間 — 首頁 繁中字幕 00/00` |
| 100ms+ | 1221 | no | yes | `Vido 請輸入密碼以繼續 密碼 登入` |

**未通過驗證的訪客，第一個看到的畫面是一座他還沒證明自己擁有的片庫的骨架** — 側欄、導覽、讀數格全都畫出來了。成因是 `__root.tsx:61-62`：`isBareRoute` 不含 `authLoading`。在硬碟剛喚醒的 NAS 上這個窗口遠不只 40ms。

（註：直接開 `/login` 這條路徑是乾淨的，實測 31 → 1221 一次到位、無閃爍。問題只在 `/` 與其他深連結。）

**Fix.** 把 `authLoading` 納入 `isBareRoute`，解析期間渲染極簡等待面（**不要 spinner** — 沒有東西的進度可量測）。更好：把 gate 搬進 router `beforeLoad`。

**Suggested command:** `/impeccable harden`

### [P1] 登出和切換主題在畫面上是同一個東西

實測兩者字色 `#41554C`、字級 **11px**、列高 44px、圖示尺寸、hover 全部相同。一個翻轉外觀偏好；另一個終結 session 並 `queryClient.clear()`，無確認、無 undo、無回饋。

而在手機「更多」sheet 裡，這兩個**會做事**的控制項比兩個**只是換頁**的目的地更小更淡：登出 11px / `--text-muted`（6.50:1） vs 探索・設定 14px / `--text-secondary`（7.91:1）。**視覺重量沒有追隨後果，它追隨的是旁邊那個元件長什麼樣。**

`SidebarFooter.tsx:3` 自己把這區定義為「Ambient status strip」（讀數區）。把一個會開火的東西放進讀數區，在這套語彙裡是類別錯誤。

**Fix（由便宜到貴）：**
1. 分開 — 主題切換確實是可瞄的狀態，登出不是。移到磁碟／健康點**下方**、自帶髮絲線；或給它 14px / `--text-secondary`，跟它真正的同儕對齊。
2. 確認 — 兩行對話框：「要登出嗎？下次進來要再輸入一次密碼。」
3. 承認 — 登入頁落地時帶一句「已登出」，讓動作有結尾。

**Suggested command:** `/impeccable layout`

### [P2] 這張登入卡片不屬於 Vido

卡片底 `#FAF6EA` 對頁面底 `#FAF6EA` = **1.00**；16px 圓角；`shadow-2xl`（Tailwind 字面值 25% 純黑，而日巡的陰影規則是「墨不是煤」、總墨量上限 15.4%）；大寫 `Vido`。`SetupWizard.tsx:123` 帶著逐字元相同的配方 — 一個活在設計系統外面、成員數為 2 的迷你設計系統。手機上卡片左緣距 viewport 僅 3px，等於沒有安全邊距。

**Fix.** `bg-[var(--bg-secondary)]` + `rounded-[var(--radius-lg)]` + `shadow-[var(--shadow-md)]`，拿掉邊框的 `/50`（有色階又有邊框是過度指定）；wordmark 改小寫 `vido` + `--accent-text` + 副標「NAS 媒體庫」，照側欄寫法；手機加左右 gutter。兩處一起修，或抽一個 `GateCard`。

**Suggested command:** `/impeccable polish`

### [P2] 沒有 autoFocus、沒有顯示密碼、錯誤訊息把輸入框推走

- 全頁唯一的欄位沒有 `autoFocus`（`grep autoFocus` 命中 6 個檔案，LoginForm 不在其中）。
- 沒有顯示／隱藏切換，而這個欄位的值是 `.env` 裡一長串隨機字串，在 390px 上要盲打。
- alert 在 DOM 中插在 `<form>` 之前，卡片又垂直置中，所以送出失敗時輸入框**往下跳 31px**（卡片頂 y 295→264），正好在使用者伸手要重打的瞬間。失敗後焦點沒回欄位（實測 `document.activeElement === input` 為 false）。
- 錯誤只在送出時清除，打字時不清，所以「密碼錯誤」會掛在欄位上方整個重打過程。

**Fix.** 加 `autoFocus`；欄位內加 44px 眼睛按鈕；預留 alert 高度讓版面不動；失敗時 re-focus + select；補 `aria-invalid` / `aria-describedby`。

**Suggested command:** `/impeccable clarify`

## What's Working

**① 兩種顏色規則在最草率的元件裡也活了下來。** 錯誤文字 `#87251B`（`--error-text`，可讀紅）**8.41:1**；啟用的登入填色 `#886208`（`--accent-primary`，可按金）配 `#FDFAF2` **5.30:1**。一張連圓角、深度、wordmark 都做錯的卡片，把最難的那條規則做對了 — 設計系統是真的被內化了。

**② reduced-motion 是完美的。** 首頁 343 個帶 animation/transition 的可見元素**全部**被夾到 0.001s；`animationName !== 'none'` 者 0 個；`infinite` 者 0 個。`/login` 22 個元素中超過 10ms 者 0 個。

**③ 健康點用形狀而不只是顏色。** 實心／實心＋光暈／空心環三形態，8px 上給低視力**視覺**使用者三種可區分形態。而「未設定」是**空心環**不是灰色實心點 — 「你沒要求過」在視覺上是不同**種類**，不是「壞掉」的弱化版。

**④ `authEnabled === false` 時 LogoutButton 什麼都不渲染。** 做不到自己工作的控制項不被留在畫面上假裝。

## Persona Red Flags

**繁中圈 NAS 自架玩家（PRODUCT.md 目標受眾）**
- 「密碼錯誤」不提 env var 名稱 — 他是唯一能對「改 `VIDO_AUTH_PASSWORD` 再重啟」採取行動的人，而這是唯一沒給他看的一句話。
- 靜默的 60 秒鎖定 — 他的直覺是去翻 container log，而 log 裡先有 `TOO_MANY_ATTEMPTS`。**產品輸給了自己的 stdout。**
- gate 不說這是哪一台安裝 — NAS 上跑正式、筆電上跑測試，兩張畫面一模一樣。
- `Vido` vs `vido` — 讀過 README 的人第一個畫面就會注意到。

**首次安裝的非作者使用者（PRODUCT.md「成功長什麼樣」）**
- disabled 登入鈕 2.54:1，讀起來是「比較淡的普通按鈕」不是停用，而 DESIGN.md 要求「存在但不能用的控制項必須說明原因」。
- `/` 上的殼層閃現讀起來像「載入好了、然後掉回登入頁」= 壞了。
- 零說明文字，`.env` 沒生效時沒有任何一條線可以拉。

**共用螢幕上用手機的人（PRODUCT.md「手機與桌機同等重要」）**
- 登出只能走「更多」，而 sheet 開場是主題切換、儲存空間、五顆健康點 — 三樣他沒要的，才輪到他要的。
- 登出是 sheet 裡最小最淡的項目：安全關鍵動作在視覺上最不重要。
- 無確認，44px 之外就是外觀相同、只會換主題的列。
- **主題切換在 390px 出現兩次**（header + sheet），而 `ThemeToggle.tsx:8-11` 的註解明文宣稱「任何斷點都不會渲染兩次」。破壞它的是 `MobileMoreSheet.tsx:22` 的 `<SidebarFooter />` — 而 LogoutButton 引用的正是同一句作為存在正當性。

## Minor Observations

- 🔐 **`auth_gate.go:26-27` 的註解是錯的。** 它宣稱「Rotating the password (or VIDO_SESSION_SECRET) invalidates every outstanding token at once」，但 token payload 只有 `<expUnix>`、以 `sessionSecret` 簽章，而 `sessionSecret` 獨立持久化在 `.session_secret`。**改 `VIDO_AUTH_PASSWORD` 什麼都不會失效，既有 session 活滿 30 天。** 這正好打中 LogoutButton 自己援引的場景：有人看到你共用螢幕上的密碼、你因此改密碼 — 他還能繼續用一個月。
- ⏱ **鎖定其實發生在第 6 次，不是第 5 次。** `recordFailure` 在第 5 次失敗後才設 `lockedUntil`，而 `allow()` 在請求開頭檢查 — 所以 429 只在第 6 次出現。第 5 次的畫面與第 1 次逐字相同（「密碼錯誤」）。
- 🎯 **focus 環有 ~80ms 是隱形的。** 送出鈕吃全域 `:focus-visible`，而 Tailwind v4 的 `transition-colors` **包含 `outline-color`**（實測 `transition-duration: 0.15s`），所以 outline 從 `currentColor` 淡入：light `@0ms` = **1.04:1**、`@80ms` = 4.05、`@600ms` = 5.12。密碼框沒這問題（用 `box-shadow` ring、duration 0s）。reduced-motion 下全域 `!important` 夾到 1ms，所以只有一般使用者踩得到。
- `LoginForm.tsx` **沒有 spec 檔**，而 `LogoutButton.spec.tsx` 有。分支失敗狀態最多的畫面是沒被測的那個。
- `ux-design.pen` 的 162 張設計稿裡**沒有任何一張是登入畫面**。這解釋了為什麼它讀起來像任何產品的登入頁 — 它從來沒被設計過。
- 一個 76 行的檔案裡有 5 個臨時 alpha 修飾子（`/50`、`/30`、`/10`、`/60`），而 `--error-tint`、`--accent-tint` 早就存在且受 `styles-contrast.spec.ts` 把關。那個 `/10` 錯誤底色 8.41:1 沒問題 — 但它**在受測面之外**，靠運氣過關，不是靠閘門。
- 輸入框 `bg-[var(--bg-secondary)]/60` 實測 `#F4EDDD`，對卡片只有 **1.08:1**。這個稀釋主動削弱了唯一在說「這裡可以打字」的可供性；不稀釋是 1.15:1，也就是 app 裡其他每個欄位拿到的待遇。
- `useAuthStatus` 的 `staleTime: 5 分鐘` + `src/` 零 401 處理：**session 中途失效的人會待在一個完整渲染、什麼都載不出來、也沒有解釋的殼層裡**，最長五分鐘。資料庫有 `DatabaseUnavailableBanner`，驗證什麼都沒有。
- `/login` 冷載入會發一次 `GET /api/v1/setup/status` → **401**（console error），因為 `useSetupStatus` 在 auth gate 解析前就發查詢。無害但污染 console。
- 收合軌上，`LogOut`（方括號＋箭頭）與軌道頂端的 `PanelLeftOpen`（方括號＋箭頭）同軌、同 20px、僅靠 tooltip 區分；而紅色「異常」健康點就緊貼在登出圖示正下方，一眼掃過會被讀成那個控制項的狀態。
- `DatabaseUnavailableBanner` 只在 `AppShellV2` 內掛載，所以 `/login` 上永遠不渲染。驗證不碰資料庫，所以**使用者可以成功登入一個完全壞掉的 app，而 gate 上沒有預警**。
- gate 是全 app 唯一沒有主題控制的畫面。第一印象取決於 OS，而在日巡下那顆 disabled CTA 正好是它最不可讀的時候。
- 390px 無水平溢出（`/login`、首頁、更多 sheet 皆 390/390）。

## Questions to Consider

**① 「系統量到了卻不渲染」為什麼不是同一條規則的另一半？** 後端已經量到三個真實數字：還剩幾次、還要鎖幾秒、密碼在哪個 env var。三個都在 `authService.ts:37` 被丟掉。如果「不渲染系統沒在量測的數字」是硬規則，它的鏡像為什麼不是？

**② `DatabaseUnavailableBanner`（42 字）和「密碼錯誤」（4 字），是同一套標準寫出來的嗎？** 差距不是資源問題 — 是不是有一個「gate 不算產品」的假設，從來沒有被說出口地存在著？

**③ 這個介面裡，還有哪個動作的視覺重量是照它的後果決定的，而不是照它旁邊那個元件長什麼樣決定的？**

**④ 手機的登出路徑是設計出來的，還是 `<SidebarFooter />` 這一行的副產品？** 如果是副產品，那它現在承擔的是共用螢幕上的安全責任 — 這個責任是誰決定交給它的？
