# Story DSR.13: Flow N 首次啟動精靈——程式碼追回 N1–N5 設計稿

Status: review

## Story

As a self-hoster who just installed Vido on a NAS,
I want 精靈五步的長相、用詞跟設計稿是同一份事實，而且我在精靈裡填的東西真的會被用到,
so that 我對這個產品的第一印象不是一張會說謊的摘要頁。

## Context

`epic-dsr` 的第 13 張，Flow N（2026-09-11 新畫的六張稿：N1-D～N5-D、N3-M）。範圍是 `apps/web/src/components/setup/**`，外加 `routes/setup.tsx`。

立案時記的三個已知缺口，create-story 查證後**三個都要更正**：

1. **「精靈的資料模型與 E5-D／LibraryCard 不一致，要先裁定」——不成立。** `setup_service.go` 的 `CompleteSetup` 把精靈的每一列建成**一個媒體庫**：資料夾名稱當庫名、`Paths: []string{entry.Path}`。這就是設定頁「具名媒體庫＋多路徑」那個模型，只是精靈刻意不問名字、一庫只收一條路徑。沒有東西要裁定，**dsr-5 不受影響**。`MediaLibrarySetupStep.tsx` 標頭的 ⚠️ 警告改寫成這段事實。
2. **「SetupWizard.tsx 有 raw shadow-*」——dsr-9 已轉成 `--shadow-xl` token。** 但依 2026-09-14「陰影只屬於浮層，一個例外都沒有」，精靈卡是坐在空白頁面上的卡、不是浮層（與 dsr-12 登入卡同一個判例），**陰影整個拿掉，稿與碼都拿**。
3. **「16 檔中 8 檔有 Design ref 標頭」——不準。** 16 檔＝8 個 spec（Rule 21 豁免）＋8 個非 spec，**非 spec 的 8 個全都有標頭**。此項撤銷。其中 `MediaFolderStep.tsx` 的標頭寫「no current screen frame」——因為它根本沒有被 `WIZARD_STEPS` 掛載，是 7b-3 換成多媒體庫之後留下的死碼，本張刪除。

**真正最大的發現不在長相，在資料：**

> 精靈第 4 步收的 TMDb 金鑰存成 secret `tmdb_api_key`、AI 金鑰存成 `ai_api_key`、提供者存成設定 `ai_provider`。**全 repo 沒有任何程式讀這三個名字。** 執行中的伺服器透過 `KeyResolver` 讀的是 `tmdb.api_key`／`claude.api_key`／`openai.api_key`。所以精靈裡填的金鑰從來沒被用過，第 5 步還會寫「已設定」。而且 resolver 只認得 Claude 這一把文字 AI 金鑰——精靈提供的「Google Gemini」選項只能靠環境變數，選了也沒用。

⚖️ **Alexyu 2026-09-14 裁定：「只留 Claude」。** 拿掉 AI 提供者選單與 Gemini，欄位改叫「Claude 金鑰」，後端存到設定頁用的同一個名字。另一個選項（精靈完全不收 AI 金鑰）未採用。

同一頁還有第二個讀錯欄位的 bug：第 5 步「媒體資料夾」讀的是 `data.mediaFolderPath`——精靈改成多媒體庫之後就沒人填這個欄位了，**真實跑一次永遠顯示「未設定」**。設計稿畫的是「2 個（電影・影集）」。

⚠️ **這張 story 的驗收基準是 `.pen` 的節點值**（Pencil MCP 直接讀出），不是 PNG。`flow-n-setup-wizard` 早已在 `READABLE_FLOWS`，六張稿是 2x 可讀截圖。

## Acceptance Criteria

### 設計稿 vs 程式碼對照表（本 story 的事實基礎）

**共用殼層（N1–N5 的 `wizard-card`）**

| 元素 | `.pen` 值 | 程式碼（改前） | 判定 |
| --- | --- | --- | --- |
| 卡片頭 | 字標 `vido`（`$Type/H2/Size` 24/20、`$accent-text`、700）＋「NAS 媒體庫」（Label 12、`$text-muted`），底部對齊置中 | `<h1>Vido 設定精靈</h1>` ＋「步驟 N / 5」 | ❌ 碼錯 |
| 卡片形狀 | `$radius-xl`＋`$border-subtle` 1px、padding `$Space/2xl`、gap `$Space/xl` | `rounded-2xl`＋`border-subtle/50`、區塊用 `mb-8` | ❌ 碼錯 |
| 卡片陰影 | `effect` = `#00000066` y12 blur40 | `shadow-[var(--shadow-xl)]` | ❌ **兩邊都錯**（§Shadow Vocabulary 2026-09-14：不是浮層） |
| step-progress | 10px 點、24×2 線、gap 8、泥金／`$bg-tertiary` | 同 | ✅ |
| 按鈕列 | 上一步（`$bg-tertiary` 實底、`$text-primary`、600）· 跳過（無底、`$text-secondary`）· 下一步（泥金、撐滿）；高 44 | 上一步是框線、順序是 上一步·下一步·跳過；高 ~42 | ❌ 碼錯 |
| 輸入框 | `$bg-secondary`＋`$border-subtle`、`$radius-md`、高 44、值用 Mono | `bg-secondary/60`＋`border/50`、`rounded-lg`、~42、非 Mono | ❌ 碼錯 |

**逐張**

| 畫面 | 元素 | `.pen` 值 | 程式碼（改前） | 判定 |
| --- | --- | --- | --- | --- |
| N1-D | 語言下拉 | 右側 `chevron-down` `$text-muted` | 原生箭頭 | ❌ 碼錯 |
| N2-D | 位址欄標籤 | 「伺服器位址」 | 「WebUI 網址」 | ❌ **兩邊都錯**：設定頁 `QBittorrentForm` 與 C 系列稿都叫「主機位址」（spec 鎖定），同一個設定三個名字 → 統一「主機位址」 |
| N3-D | 類型選擇 | 兩顆切換鈕：選中＝`$accent-subtle` 底＋`$accent-primary` 框＋`$accent-text`；未選＝`$bg-tertiary`＋`$text-secondary`；Label 12/600、高 32 | `<select>` | ❌ 碼錯 |
| N3-D | 路徑列 | 列本身就是欄位（無內框）、Mono 14、右側 32×32 刪除鈕 | 內框輸入＋「資料夾路徑」「類型」可見標籤 | ❌ 碼錯（標籤改 sr-only 保留無障礙名稱） |
| N3-D | 刪除圖示 | `trash-2` 穿 `$error-text` | `X` 靜止 muted、hover 硃砂 | ❌ **兩邊都錯**：刪掉一列還沒存的資料不是「壞了」，固定詞彙不准挪用硃砂；依 §Buttons Ghost → `$text-muted` |
| N3-D | 新增媒體庫 | 實線 `$border-subtle`、高 44、600 | 虛線 `/50` | ❌ 碼錯 |
| N3-M | 標題／說明／路徑字級 | 標題 Body 14、說明 Label 12 且只剩「至少需要一個媒體庫。」、路徑 Label 12 | （碼不分斷點） | ❌ **稿錯**：§Responsive「Heading 以下不變、內文字級不變」，且截短文案是 dsr-10 抓過的假截斷 → 稿改回 H4／Body／完整句 |
| N3-M | 觸控 | 切換鈕與刪除鈕 44 高、切換鈕各佔一半 | — | ✅ 碼照做（`h-11 sm:h-8`） |
| N4-D | AI 欄位 | 「AI 金鑰」單一欄位、無提供者 | 提供者下拉（含 Gemini）＋條件式金鑰 | ❌ **裁定後兩邊都改**：「Claude 金鑰」＋說明列「用於字幕翻譯與 AI 檔名解析」 |
| N4-D | TMDb 標籤 | 「TMDb 金鑰」 | 「TMDb API 金鑰」 | ❌ 碼錯 |
| N4-D | 跳過警語 | `$warning-tint` 底＋`$warning-text` 字與圖示 | `warning-tint`＋`warning-text`、無圖示、14px | ❌ **兩邊都錯**：2026-09-11 裁定「赭色說的是現在的世界，不是你按下去會怎樣」，事前警語用 `$bg-tertiary` 底、`$text-primary` 字、保留圖示 |
| N5-D | 完成圖示 | 56px `$success-tint` 圓底＋26px `circle-check` | 48px 裸圖示 | ❌ 碼錯 |
| N5-D | 摘要順序與值 | 語言「繁體中文」· 媒體資料夾「2 個（電影・影集）」· qBittorrent「已設定」· TMDb 金鑰 · AI 服務 | `zh-TW` · qBT 印網址 · 媒體資料夾讀死欄位 · 「TMDb API」· AI 印 `gemini` | ❌ 碼錯（且媒體資料夾是真 bug） |
| N5-D | AI 列標籤 | 「AI 服務」 | — | ❌ 隨裁定改「Claude 金鑰」，稿同步 |
| N5-D | 補充說明 | 「未設定的項目之後都可以在「設定」裡補上。」 | 無 | ❌ 碼錯 |
| N5-D | 完成設定鈕 | `$accent-primary` | `bg-[var(--success)]` | ❌ 碼錯（青碧是狀態不是按鈕色） |
| 全部 | Design ref 檔頭 | — | 8/8 非 spec 檔都有 | ✅ |

**節點 ID（逐字抄進檔頭，不要重查）：**
`N1-D (dzgq9)` · `N2-D (CP7AX)` · `N3-D (TyjL0)` · `N4-D (D990CP)` · `N5-D (CWh3E)` · `N3-M (YyaqL)`

---

1. **精靈填的金鑰要真的被用到。** `CompleteSetup` 透過設定頁自己的 `KeySettingsService.Save` 存 TMDb／Claude 金鑰（落在 `tmdb.api_key`／`claude.api_key`，就是 `KeyResolver` 讀的名字），精靈與設定頁從此只有一條儲存路徑，「寫的名字」與「讀的名字」不可能再分岔。`ai_provider` 設定不再寫入。`models.SetupConfig` 的 `AIProvider`／`AIApiKey` 換成 `ClaudeApiKey`（JSON `claude_api_key`）。
2. **第 4 步只收 Claude。** 沒有提供者選單、沒有 Gemini；「Claude 金鑰」是密碼欄位、有說明列並以 `aria-describedby` 連結。
3. **第 5 步說真話。** 語言顯示名稱；媒體資料夾由 `libraries` 算出「N 個（類型・類型）」（空白路徑不算、同類型只列一次）；qBittorrent／TMDb／Claude 一律「已設定／未設定」；有補充說明行；完成鈕用泥金。
4. **殼層與五步對齊上表**，按鈕列抽成 `StepNav`（順序固定 上一步 · 跳過 · 下一步），文字欄位抽成 `WizardTextField`，語言清單抽成 `setupLanguages.ts` 讓第 1 步與第 5 步共用同一份。
5. **卡片無陰影。** 程式碼移除 `shadow-[var(--shadow-xl)]`；`.pen` 六張 `wizard-card` 的 `effect` 清空。
6. **設計稿修正 7 處**（每一處都有規則依據，見對照表的「兩邊都錯／稿錯」）：卡片陰影 ×6、N2 標籤、N3 刪除圖示 ×4、N3-M 字級與文案、N4 Claude 欄位＋說明列、N4 跳過警語、N5 列標籤。改完存檔、重匯截圖，**只 stage Flow N 那 6 張＋`pen-tokens.json`**。
7. **死碼刪除。** `MediaFolderStep.tsx`／`.spec.tsx`、它的 gallery 夾具與 6 張視覺基準線；前端 `SetupConfig.mediaFolderPath` 一併移除（後端相容欄位保留，`tests/support/global-setup.ts` 仍在用）。
8. **無障礙不退步。** 「步驟 N / 5」改 sr-only 保留；路徑欄的「資料夾路徑」與類型的「類型」仍是可查詢的無障礙名稱（retro-11-AI1b 的 spec 不改斷言）；類型切換是原生 radio（方向鍵可切換、鍵盤焦點有環）。
9. **視覺回歸。** 6 個 setup 夾具的 `-darwin` 本機重生；對應 16 張 `-linux` 用 `git rm` 轉成「缺少」，交給 CI bootstrap（`project_visual_baseline_intentional_change` 四步流程）。⛔ 不 commit 任何 `-linux.png`。
10. **CI 全綠。** lint 0 errors、typecheck、format:check、`check-design-tokens.py`、web／api 單元測試。
11. **（CR 補）沒有 ENCRYPTION_KEY 就不收金鑰，而且要早說。** 設定頁在沒有 `ENCRYPTION_KEY` 時拒絕存金鑰（`ErrKeysNotWritable` → 409）；精靈共用同一條路徑，所以也拒絕：第 4 步按「下一步」時就回 409 並在卡片上說「請按「跳過」，設定好之後再到「設定 › API 金鑰」填寫」；`CompleteSetup` 則在**寫入任何東西之前**就拒絕，避免重試時把媒體庫建兩次。
12. **（CR 補）「跳過」會丟掉那一步填了一半的內容。** 跳過從不驗證，所以 qBittorrent 三格與兩把金鑰在按「跳過」時清空，不會帶著沒驗證的值送出、也不會在摘要上顯示「已設定」。
13. **（CR 補）金鑰欄位不讓瀏覽器當成登入表單。** 精靈的文字欄位一律 `autoComplete="off"`、`autoCapitalize="off"`、`spellCheck={false}`（與設定頁金鑰欄位同一個做法），避免 Chrome 把 Claude 金鑰存成本站密碼、之後自動填進登入閘。

## Tasks / Subtasks

- [x] **Task 1 — 後端：金鑰存到 resolver 讀的名字（AC #1）**
  - [x] `setup_service.go`：金鑰改走 `SetupKeyWriter`（＝`KeySettingsService`，`main.go` 注入），拿掉 `ai_provider` 寫入，log 欄位 `has_ai_key` → `has_claude_key`
  - [x] `models/settings.go`：`ClaudeApiKey`（gofmt）
  - [x] `setup_service_test.go`：成功案例改期望 `tmdb.api_key`／`claude.api_key`（**刻意寫字面值**：secret 名稱是落盤資料，改名等於遷移）；刪 `ai_provider` 失敗案例；`ai_api_key` 失敗案例改 Claude

- [x] **Task 2 — 共用元件（AC #4）**
  - [x] `StepNav.tsx`（`ui/Button` 的 secondary／ghost／default，`h-11`）
  - [x] `WizardTextField.tsx`（label · input · hint，Mono、`aria-describedby`）
  - [x] `setupLanguages.ts`

- [x] **Task 3 — 殼層（AC #4, #5, #8）**
  - [x] `SetupWizard.tsx`：字標、`rounded-[var(--radius-xl)]`、`gap-6`、無陰影、sr-only 步驟數、`claudeApiKey` 送出
  - [x] `StepProgress.tsx`：拿掉 `mb-8`（改由卡片 gap 控制）
  - [x] `routes/setup.tsx`：`px-6 py-10`（N3-M 卡片左右 24px）

- [x] **Task 4 — 五步（AC #2, #3, #4, #8）**
  - [x] N1 語言下拉加 chevron
  - [x] N2 「主機位址」、按鈕列順序
  - [x] N3 列即欄位（`has-[input[type=text]:focus]` 讓整列描邊轉焦點色）、原生 radio 切換鈕（`useId` 讓群組名每次掛載唯一）、Ghost 刪除鈕、實線新增鈕、手機 44px
  - [x] N4 TMDb／Claude 兩欄、中性跳過警語
  - [x] N5 圓底圖示、摘要五列、補充說明、泥金完成鈕

- [x] **Task 5 — 設計稿（AC #6）**
  - [x] Pencil MCP 修改 7 處 → AppleScript 點 File ▸ Save → `git hash-object` 確認落盤
  - [x] `python3 scripts/export-pen-screenshots.py` 188/188；只 stage Flow N 6 張＋`pen-tokens.json`

- [x] **Task 6 — 死碼與夾具（AC #7, #9）**
  - [x] `git rm` MediaFolderStep 兩檔與 `setup-media-folder-step/` 6 張基準
  - [x] gallery 夾具：`setup-api-keys-step` 改 `claudeApiKey`、`setup-complete-step` 改成 N5-D 同一組資料（兩個媒體庫、qBT＋TMDb 已設定、Claude 未設定）
  - [x] `CI=1 npx playwright test --project=visual --update-snapshots=all` → 只保留 6 個 setup 夾具的 16 張 `-darwin`，其餘 `git checkout`；`git rm` 對應 16 張 `-linux`

- [x] **Task 7 — 測試與驗證（AC #8, #10）**
  - [x] spec：ApiKeysStep 重寫（13）、CompleteStep 重寫（16）、MediaLibrarySetupStep 擴充（1 → 7）、QBittorrentStep ＋2、SetupWizard ＋2 並改 `zh-TW` → `繁體中文`
  - [x] CDP／真鍵盤互動檢查（`/test/gallery?fixture=<id>`，桌機 1280 與手機 390 各一輪）

- [x] **Task 8 — 對抗式 code review 的修正（AC #11, #12, #13）**
  - [x] `KeySettingsService.Writable()`；`SetupService.SetKeyWriter`；`validateApiKeysStep` 與 `CompleteSetup` 的 ENCRYPTION_KEY 閘門（後者在任何寫入之前）；`setup_handler.go` 把 `ErrKeysNotWritable` 對應成 409 `SETUP_KEYS_NOT_WRITABLE`
  - [x] `TestSetupService_KeysNotWritable`（5 個子測試：第 4 步拒絕、沒填金鑰照常通過、完成時零寫入、沒注入 writer 也拒絕、沒金鑰的完成不受影響）＋ handler 兩個 409 案例
  - [x] `SetupWizard.tsx`：第 4 步驗證也送 `claudeApiKey`；`clearOnSkip`
  - [x] `WizardTextField.tsx`：`autoComplete`／`autoCapitalize`／`spellCheck` 關閉
  - [x] `setupService.spec.ts`：原本還在送已刪除的 `mediaFolderPath`（`tsconfig.spec.json` 會報 TS2353，但 CI 的 typecheck 只看 app 設定所以沒擋），改成斷言 `claudeApiKey` 真的變成 `claude_api_key`
  - [x] TestSprite `TC106`／`TC107`：用 xpath 抓路徑輸入框，版面一改就抓不到 → 改 `get_by_test_id`

## Dev Notes

### 不要做的事

- **不要把精靈的媒體庫改成「具名＋多路徑」。** 查證後兩邊是同一個資料模型，精靈是刻意簡化，要改名／加路徑去設定頁。
- **不要把 Gemini 加回精靈。** Alexyu 2026-09-14 裁定只留 Claude；resolver 目前不支援 Gemini 的 secret，加回去就是再做一個存了沒人讀的欄位。
- **不要動「設定完成！」與青碧勾勾。** 按下「完成設定」之前其實什麼都還沒存，這個主張不成立——但改標題是文案／詞彙裁定，已立案 `disc-2026-09-setup-complete-claims-done-before-submit`，不在對齊範圍。
- **不要搬舊安裝留下的 `tmdb_api_key`／`ai_api_key` secret。** 那兩個名字從來沒被讀過，裡面的值也從來沒生效過；精靈只在第一次啟動跑一次，已經裝好的人早就用設定頁或環境變數設過金鑰。搬過去反而可能蓋掉他們現在真正在用的值。它們就留在 secrets 表裡當孤兒。
- **不要在精靈的 TMDb 說明列加「需重啟」。** TMDb client 目前只讀 env，連重啟都不會吃到 `tmdb.api_key`；那是 `sub-7-7` 的範圍（它 lane ① 吸收了 `backlog-tmdb-runtime-key-resolution`），已在該 story 加耦合條款。

### 固定詞彙檢查

| 位置 | 用色 | 結論 |
| --- | --- | --- |
| step-progress、下一步、完成設定 | 泥金實底 | ✅ 要被按的控制項 |
| 類型切換選中 | `--accent-subtle` 淡洗＋框 | ✅ 配給強調規則：「目前所在」用淡洗不用實心 |
| 完成圖示 | 青碧 | ⚠️ 見上方立案 |
| 跳過警語 | 中性（改前赭） | ✅ 事前警語不給語意色 |
| 刪除圖示 | 中性（改前稿是硃砂） | ✅ 可逆動作不穿「壞了」 |
| 錯誤橫幅 | 硃砂 | ✅ 真的壞了（驗證失敗） |

### Source tree

```
apps/api/internal/services/setup_service.go        ← Task 1
apps/api/internal/services/setup_service_test.go   ← Task 1
apps/api/internal/models/settings.go               ← Task 1
apps/web/src/components/setup/StepNav.tsx          ← Task 2（新）
apps/web/src/components/setup/WizardTextField.tsx  ← Task 2（新）
apps/web/src/components/setup/setupLanguages.ts    ← Task 2（新）
apps/web/src/components/setup/SetupWizard.tsx      ← Task 3
apps/web/src/components/setup/StepProgress.tsx     ← Task 3
apps/web/src/routes/setup.tsx                      ← Task 3
apps/web/src/components/setup/*Step.tsx            ← Task 4
apps/web/src/services/setupService.ts              ← Task 1/7（型別）
apps/web/src/routes/test/-gallery.fixtures.tsx     ← Task 6
ux-design.pen                                      ← Task 5
_bmad-output/screenshots/flow-n-setup-wizard/*.png ← Task 5（6 張）
tests/visual/.../setup-*/                          ← Task 6
```

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `components/setup/` 10 個非 spec 檔（7 個原有＋3 個新增）都不讀時鐘。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` Screen N1-D (dzgq9) / N2-D (CP7AX) / N3-D (TyjL0) / N4-D (D990CP) / N5-D (CWh3E) / N3-M (YyaqL)] — 節點屬性經 Pencil MCP `execute` 直接讀出
- [Source: `apps/api/internal/services/key_resolver.go`] — `SecretNameClaude`／`SecretNameTMDb`／`SecretNameOpenAI` 與 `[@contract-v1]` 的「使用者在 UI 打的就是 pipeline 用的」
- [Source: `apps/api/internal/services/setup_service.go#CompleteSetup`] — 每列建一個媒體庫的證據
- [Source: DESIGN.md#Shadow Vocabulary] — 2026-09-14 陰影只屬於浮層
- [Source: DESIGN.md#Secondary — 狀態詞彙] — 2026-09-11 赭色說的是現在的世界
- [Source: DESIGN.md#Responsive] — 內文字級不變、Heading 以下不降階
- [Source: `apps/web/src/components/settings/QBittorrentForm.spec.tsx`] — 「主機位址」的 spec 鎖定
- [Source: `.claude/memory/project_visual_baseline_intentional_change.md`] — 功能分支上 `-linux` 的四步流程

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，2026-09-14（SM create-story 與 dev 同一個 session）

### Debug Log References

- `pnpm nx test web`：**3363 / 3363 passed**（261 檔，CR 修正後重跑）
- `pnpm nx test api`：PASS；CR 修正後 `go build ./...`、`go vet ./internal/services/ ./internal/handlers/`、`go test ./internal/services/ -run 'Setup|KeySettings|KeyResolver'`、`go test ./internal/handlers/ -run Setup` 全 ok
- `pnpm run lint`：**0 errors**、128 warnings；`npx eslint apps/web/src/components/setup apps/web/src/routes/setup.tsx` 零輸出（本 story 零 warning）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `pnpm run format:check`：只剩 `apps/api/coverage/*` 與 `testsprite_tests/*.html` 兩類本機未追蹤檔（CI 不會有）
- `python3 scripts/check-design-tokens.py`：一致（82 變數／188 畫面／72 母版）
- `export-pen-screenshots.py`：第一次 168/188（一個 chunk 失敗、腳本自己擋下），重跑 **188/188**；全部重匯裡**只有 Flow N 6 張與 `pen-tokens.json` 有位元差異**
- `CI=1 npx playwright test --project=visual --update-snapshots=all`：1 passed（8.4m），保留 6 個 setup 夾具的 16 張 `-darwin`

### Completion Notes List

- 🔗 **AC Drift: N/A**——`7b-3-setup-wizard-multi-library` 與 `sub-2-1a-key-resolution-api` 都是 done，本 story 沒有改它們的 AC，而是補上兩者之間漏接的那一段（精靈寫的名字 vs resolver 讀的名字）。
- 📎 **Contract Stamps:** 讀取 `KeyResolver [@contract-v1]`，**沒有改動**它（名稱、解析順序都不變），只是讓精靈成為它的合法寫入方。
- 🎭 **A11y Pre-Flight: PASS**——(1) 「步驟 N / 5」保留為 sr-only（新 spec 斷言 `sr-only` class）；(2) 路徑欄的 label 改 sr-only、`htmlFor` 不變；(3) 類型切換是 `role="radiogroup" aria-label="類型"` 內的原生 radio，**實測**：Tab 進入落在已選的那顆、方向鍵切換到另一顆、`:focus-visible` 時外框有 ring；(4) 金鑰說明列用 `aria-describedby`，spec 用 `toHaveAccessibleDescription` 斷言；(5) 所有觸控目標手機 44px（實測 390 寬：切換鈕 44×146、刪除鈕 44×44、按鈕列 44）。
- 🖱️ **互動狀態實測**（`/test/gallery?fixture=<id>`，1280 與 390 兩輪，數值為日巡）：列描邊 靜止 `rgb(205,190,155)` → 路徑聚焦 `rgb(136,98,8)`，焦點移到 radio 時**不變**（刻意）；下一步 靜止／hover／active = `136,98,8` / `114,82,5` / `91,65,3` 三值互異；刪除鈕、新增鈕、跳過、未選切換鈕的 hover 都有變化。
  - 🪤 **踩到一次**：第一次在完整 gallery 上量，焦點與 hover 全都「沒反應」——因為另一個夾具開著 Dialog，全頁覆蓋層攔截指標、焦點被 trap 回去。改用 `?fixture=<id>` 隔離後正常。
  - 🪤 **因此抓到一個真問題**：隔離後 Tab 仍會跳過類型切換。原因是 gallery 對同一夾具渲染 default／hover／focus 三份，三份的 radio `name` 都是 `library-type-fixture-lib-1`，瀏覽器把它們併成同一組、Tab 停在別份的已選 radio。實機只掛一份所以不會發生，但這是「name 只在列內唯一、不在掛載內唯一」的潛在碰撞 → 改用 `useId()` 當前綴，重測 Tab 正確落點。
- 🔍 **對抗式 code review（獨立 agent）：0 HIGH / 1 MED / 5 LOW，全部處理。**
  - **MED-1 精靈繞過了設定頁的 ENCRYPTION_KEY 閘門。** 修正前精靈寫的是沒人讀的名字，所以沒差；本 story 讓它寫進真正的名字之後，沒有 `ENCRYPTION_KEY` 的 NAS 會用機器 ID 衍生的金鑰加密——容器重建後解不開，resolver 只記 Debug log 就退回 env，AI 功能安靜地停掉，設定頁還會 409 拒絕重存。→ 改成走 `KeySettingsService`（AC #11）。
  - **LOW-2** `setupService.spec.ts` 還在送已刪除的欄位、且沒有任何測試鎖住 `claude_api_key` 這個線上格式 → 修。
  - **LOW-3** TestSprite TC106／TC107 的 xpath 會壞、白燒月額度 → 改 test id。
  - **LOW-4** 金鑰欄位會被 Chrome 當登入表單 → AC #13。
  - **LOW-5** 「跳過」不驗證卻保留填了一半的值（修正前就存在，但現在金鑰真的會生效，後果變大）→ AC #12，lane ① 就地吸收。
  - **LOW-6** story 內兩處事實錯（「11 個非 spec 檔」其實是 10、沒寫舊 secret 成為孤兒）→ 已更正。
  - 另刪掉兩條永遠不會失敗的斷言（`not.toHaveProperty('aiProvider')`——payload 是型別封閉的物件字面值）。
- ✅ Task 1–8 全數完成，見上方 Tasks。
- 📝 設計稿截圖逐張看過：N4-D 兩個欄位＋中性警語、N3-M 完整兩行說明與 Body 路徑、N5-D「Claude 金鑰」列。視覺基準 `setup-media-library-setup-step`、`setup-complete-step`、`setup-api-keys-step` 逐張對過稿。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，四項：
    - **① 精靈金鑰是死寫入（expand in place）。** 不是對齊題，但同一個表單、同一個畫面；Alexyu 當場裁定「只留 Claude」後就地修掉，由 AC #1–#2 追蹤。
    - **① `MediaFolderStep` 是死碼（expand in place）。** 同時是第 5 步讀錯欄位的源頭，由 AC #7 追蹤。
    - **③ 「設定完成！」在存檔前就宣告完成（FILE）。** 青碧的主張在按下去之前不成立；改標題是文案裁定。立案：`disc-2026-09-setup-complete-claims-done-before-submit`。
    - **③ TMDb 金鑰寫對了名字但仍不生效（FILE → 耦合條款）。** TMDb client 在 `main.go` 直接吃 `cfg.TMDbAPIKey`（env），從不經過 resolver——精靈（以及設定頁）存的 `tmdb.api_key` 連重啟都不會生效，設定頁的「儲存後需重啟伺服器才會生效」目前也不成立。`sub-7-7-bundled-tmdb-key` 已 lane ① 吸收 `backlog-tmdb-runtime-key-resolution`，**不另開條目**，在 sub-7-7 的 sprint 條目與 story 檔 Dev Notes 加耦合條款。

### File List

- `apps/api/internal/models/settings.go`（M）
- `apps/api/internal/services/setup_service.go`（M）
- `apps/api/internal/services/setup_service_test.go`（M）
- `apps/api/internal/services/key_settings_service.go`（M，`Writable()`）
- `apps/api/internal/handlers/setup_handler.go`（M）· `setup_handler_test.go`（M）
- `apps/api/cmd/api/main.go`（M，`SetKeyWriter`）
- `apps/web/src/components/setup/ApiKeysStep.tsx`（M）· `ApiKeysStep.spec.tsx`（M）
- `apps/web/src/components/setup/CompleteStep.tsx`（M）· `CompleteStep.spec.tsx`（M）
- `apps/web/src/components/setup/MediaFolderStep.tsx`（D）· `MediaFolderStep.spec.tsx`（D）
- `apps/web/src/components/setup/MediaLibrarySetupStep.tsx`（M）· `MediaLibrarySetupStep.spec.tsx`（M）
- `apps/web/src/components/setup/QBittorrentStep.tsx`（M）· `QBittorrentStep.spec.tsx`（M）
- `apps/web/src/components/setup/SetupWizard.tsx`（M）· `SetupWizard.spec.tsx`（M）
- `apps/web/src/components/setup/StepProgress.tsx`（M）
- `apps/web/src/components/setup/WelcomeStep.tsx`（M）
- `apps/web/src/components/setup/StepNav.tsx`（A）
- `apps/web/src/components/setup/WizardTextField.tsx`（A）
- `apps/web/src/components/setup/setupLanguages.ts`（A）
- `apps/web/src/routes/setup.tsx`（M）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（M）
- `apps/web/src/services/setupService.ts`（M）· `setupService.spec.ts`（M）
- `testsprite_tests/TC106_Setup_wizard_skip_on_qBittorrent_step_reaches_the_media_library_step.py`（M）· `TC107_Setup_wizard_back_and_forward_navigation_preserves_entered_values.py`（M）
- `ux-design.pen`（M）
- `_bmad-output/pen-tokens.json`（M）
- `_bmad-output/screenshots/flow-n-setup-wizard/{n1-d,n2-d,n3-d,n3-m,n4-d,n5-d}.png`（M）
- `tests/visual/components.visual.spec.ts-snapshots/components/setup-{api-keys-step,complete-step,media-library-setup-step,qbittorrent-step,step-progress,welcome-step}/*-darwin.png`（M ×16）· `*-linux.png`（D ×16）
- `tests/visual/components.visual.spec.ts-snapshots/components/setup-media-folder-step/*`（D ×6）
- `_bmad-output/implementation-artifacts/dsr-13-flow-n-setup-wizard.md`（A）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（M）
- `_bmad-output/implementation-artifacts/sub-7-7-bundled-tmdb-key.md`（M，耦合條款）

## Change Log

- 2026-09-14 — create-story ＋ dev 同 session 完成；Alexyu 裁定「只留 Claude」。狀態 → review。
- 2026-09-14 — 對抗式 CR：1 MED／5 LOW 全數修正（ENCRYPTION_KEY 閘門、跳過清空、autocomplete、線上格式測試、TestSprite locator、story 事實更正）。
