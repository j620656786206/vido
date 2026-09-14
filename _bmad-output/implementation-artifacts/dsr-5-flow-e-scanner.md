# Story DSR.5: Flow E 媒體庫掃描——程式碼追回 E1–E5 設計稿

Status: review

## Story

As a self-hoster who just pressed 掃描媒體庫,
I want 掃描中的數字是真的、掃完之後通知上的連結真的帶我去看那些東西、設定頁畫的是我實際在用的媒體庫,
so that 我不用自己猜哪個按鈕有用、哪個數字是抄來的。

## Context

`epic-dsr` 的第 6 張（建議開跑順序 12 → 10 → 11 → **5**）。範圍：`components/scanner/**`（3 檔）、`components/settings/ScannerSettings.tsx`、`MediaLibraryManager.tsx`、`LibraryCard.tsx`、`LibraryEditModal.tsx`，對應十張稿 E1–E5（桌機＋手機）。

立案時記的三個缺口，查證後**三個都要更正**：

1. **「E1 的媒體資料夾區塊與程式碼是兩套資料模型，要先裁定」——不需要裁定。** 多媒體庫的 IA 在 Epic 7b 出貨時就定了（`prd-multi-library-amendment.md`＋`multi-library-ux-spec.md`＋`MediaLibraryManager`／`LibraryCard`）。E1 只是改版時沒回頭畫。本張照程式碼重畫 E1-D／E1-M，新增母版 `Component/LibraryCard`（ZbfJR），`drift-e1-scanner-multi-library` 關閉。
2. **「E1-D 的掃描頻率下拉前端沒有實作」——不成立。** `ScannerSettings.tsx` 早就有「每小時／每天／僅手動」下拉（`useScanSchedule`／`useUpdateScanSchedule`），是檔頭註解寫錯。
3. **「ScanProgressCard／Sheet 有 raw shadow」——dsr-9 已轉 token。** 但 DESIGN.md §Shadow Vocabulary 明文把「懸浮進度卡」列在 `--shadow-lg`，程式碼用的是 xl，改回 lg（Sheet 維持 xl）。

**真正最大的發現不在長相，在兩個壞掉的連結與一個抄來的數字：**

> **「查看未比對項目」按了只會回首頁。** 它導去 `/` 並帶 `search: { status: 'unmatched' }`，但首頁路由沒有 `validateSearch`，這個參數被丟掉。未比對篩選住在媒體庫：`/library?unmatched=true`。
>
> **「查看錯誤」按了什麼都沒有。** 它導去 `/?status=error`，全 app 沒有任何「錯誤」篩選。掃描錯誤只寫進系統日誌（`slog` → `system_logs`），`/settings/logs` 是唯一讀得到的地方。
>
> **掃描中的「比對 524」是「解析 524」的複製。** 後端 `ScanProgress` 沒有比對數（TMDb 比對在掃描之後才發生），前端的 `filesProcessed` 本身是估算值，卡片把它印了兩次。spec 甚至寫著 `// 524 appears twice: 解析 and 比對 both show filesProcessed`——看到了、接受了。設計稿 E2 畫的「解析 524 · 比對 498」也是編的數字（dsr-10 的「12.4 MB/s 是編的」同一類）。

另外手機的完成通知**完全沒有**這兩個連結（只有「產生字幕」），手機上掃完之後，剛剛數給你看的未比對項目與錯誤沒有入口。

⚖️ **Alexyu 2026-09-14 裁定：「全部說真話」**（對抗式 CR 往下挖之後）。第一版只把兩個連結改到「有落點」，CR 指出更深的問題：通知上的「**未比對 42**」數的是掃描器 `FilesUnmatched`——**抓不到集數、根本沒匯入**的影集檔（`scanner_service.go:549`），它們不在媒體庫裡，所以媒體庫的「未比對」篩選（`tmdb_id IS NULL`）**永遠看不到那 42 個**；「**比對成功 1,198**」則是「找到 − 未比對」，把錯誤也算成成功。而 `scan_complete` 其實一直帶著 `files_created`／`files_updated`，前端沒讀。裁定後：通知改成「找到 · 新增 · 更新 · 無法匯入 · 錯誤」，只留一個連結「**查看無法匯入與錯誤**」→ `/settings/logs`（兩者都以 `SCANNER_UNMATCHED`／錯誤寫進系統日誌），拿掉「查看未比對項目」。另一個選項（只換字、連結不動）未採用。

⚠️ 驗收基準是 `.pen` 節點值（Pencil MCP 直接讀出），`flow-e-scanner` 已在 `READABLE_FLOWS`。

## Acceptance Criteria

### 設計稿 vs 程式碼對照表

| 畫面 | 元素 | `.pen`（改前） | 程式碼（改前） | 判定 |
| --- | --- | --- | --- | --- |
| E1-D／M | 媒體資料夾區 | 扁平資料夾列＋鉛筆／硃砂垃圾桶＋「新增資料夾」 | `MediaLibraryManager`：具名媒體庫卡片＋⋮選單＋路徑狀態點＋頁尾計數 | ❌ **稿錯**（落後一個世代）→ 照碼重畫 |
| E1-D／M | 掃描按鈕文字 | 金底配 `$text-primary` | `--text-on-accent` | ❌ 稿錯 |
| E1-M | 上次掃描 | 「… · 1,247 檔案」（截掉耗時） | 完整一句 | ❌ 稿錯（假截斷） |
| E1 | 掃描排程 | 下拉「每小時」 | 已實作 | ✅（檔頭註解錯） |
| E1 | 掃描按鈕 hover／掃描中 | — | hover 設成跟靜止同色；掃描中是金字配半透明金底 | ❌ 碼錯 |
| E2-D／M | 統計 | 找到 · 解析 · **比對** · 錯誤 | 同，但比對＝解析的複製 | ❌ **兩邊都錯** → 拿掉比對 |
| E2-D／M | 錯誤圖示 | `$error`（填色階） | `--error-text` | ❌ 稿錯（圖示讀的是文字階） |
| E2-D | 取消掃描 | 框線按鈕 | 純文字 | ❌ 碼錯 |
| E2-D | 冒號 | 「正在處理：」「預估剩餘：」全形 | 半形 `:` | ❌ 碼錯 |
| E2-D | 卡片 | 髮絲框、radius-lg | 無框、`--shadow-xl` | ❌ 碼錯（§Shadow：懸浮進度卡＝lg） |
| E2-M | Sheet | `$bg-primary`、把手 `$text-muted`、標題 16 | bg-secondary、把手 bg-tertiary、標題 14 | ❌ 碼錯 |
| E3-D／M | 查看未比對項目 | 有 | 桌機導去 `/`（無效）；手機沒有 | ❌ 碼錯（真 bug） |
| E3-D／M | 查看錯誤 | 有 | 桌機導去 `/?status=error`（無落點）；手機沒有 | ❌ 碼錯（真 bug） |
| E3-D／M | 標題圖示（稿上錯誤 7） | 青碧勾勾 | 有錯誤時是赭色三角 | ❌ 稿錯（稿畫的狀態有錯誤） |
| E3-D／M | 倒數條 | 泥金 | `--text-muted`（程式碼註解說明刻意中性） | ❌ 稿錯（倒數不是「正在跑」） |
| E3-M | 位置 | 頂端 | 分頁列上方 | ❌ 稿錯（程式碼的位置有 Alexyu 2026-08-27 裁定） |
| E3-M | 形狀 | 內縮浮卡 | 全寬貼底 | ❌ 碼錯 |
| E4-M | 未比對卡片 | v1 PosterCard（有海報） | — | ❌ 稿錯（未比對不會有海報；桌機早已改 Unmatched 母版） |
| E5-D／M | 儲存按鈕 | 「儲存」＋`$text-primary` | 「儲存變更」 | ❌ 稿錯（`drift-e5d`） |
| E5-D／M | 新增路徑 | 只有輸入列 | 輸入列＋「＋」按鈕 | ❌ 稿錯（沒有按鈕就加不了路徑） |
| E5-M | 說明文字 | Label 12 | — | ❌ 稿錯（§Responsive：內文不降階） |
| E5-D | 視窗／欄位 | bg-secondary 視窗、bg-primary 欄位、路徑列同輸入框、說明 14 | bg-primary 視窗、半透明欄位、小路徑列、說明 12、`text-[13px]` | ❌ 碼錯 |
| E5 | 移除路徑 | 中性 × | hover 變硃砂 | ❌ 碼錯（移除是編輯不是壞了） |

**節點 ID：** `E1-D (KvZSc)` · `E1-M (uABWl)` · `E2-D (wyuhF)` · `E2-M (yezIo)` · `E3-D (szzaW)` · `E3-M (ZjoEI)` · `E4-D (QTqcC)` · `E4-M (n7jVF)` · `E5-D (hUVYm)` · `E5-M (P0P82x)` · 新母版 `Component/LibraryCard (ZbfJR)`

---

1. **完成通知只說掃描器量到的事。** 「找到 · 新增 · 更新 · 無法匯入 · 錯誤」（新增／更新沒回報時省略，不印 0）；只有一個連結「查看無法匯入與錯誤」→ `/settings/logs`，有無法匯入或錯誤時才出現。桌機與手機共用 `scanSummaryParts`／`scanHadProblems`／`SCAN_PROBLEMS_DESTINATION`，不可能再各寫各的。
2. **掃描中不再印抄來的數字。** 拿掉「比對」統計（碼與稿），找到 · 解析 · 錯誤抽成共用的 `ScanStats`。
3. **手機完成通知補上同一個入口**（原本一個都沒有），形狀改成內縮浮卡，仍在分頁列上方、仍會自己消失（Alexyu 2026-08-27 的兩條裁定原封不動）。
4. **E1 設計稿重畫成媒體庫卡片**，新增母版 `Component/LibraryCard`，E1-D／E1-M 各 instance 兩張（我的電影 2 條路徑、我的影集 1 條），DESIGN.md 元件清冊補上。
5. **程式碼與 E1／E2／E3／E5 對齊**（見對照表的「碼錯」列）。
6. **檔頭誠實化**：scanner 三檔從不存在的「H2／H5」代號改成 E2／E3；`ScannerSettings` 拿掉「下拉未實作」的錯誤警告；`MediaLibraryManager`／`LibraryCard` 拿掉「E1 已過時不可依此實作」。
7. **`drift-e1-scanner-multi-library`、`drift-e5d-save-button-label` 關閉。**
8. **視覺回歸**：7 個受影響夾具的 `-darwin` 重拍，對應 `-linux` 刪除交給 CI bootstrap。
9. **CI 全綠。**

## Tasks / Subtasks

- [x] **Task 1 — 兩個壞連結與抄來的數字（AC #1, #2, #3）**
  - [x] `ScanProgressCard.tsx`：落點常數、拿掉比對、`ScanStats`、E2／E3 版面與 token
  - [x] `ScanProgressSheet.tsx`：共用常數與 `ScanStats`、手機完成浮卡＋兩連結、E2-M 版面
  - [x] `ScanProgress.tsx`：檔頭與註解代號
  - [x] spec：導航斷言改新落點（附原因）、「524 appears twice」改成斷言沒有比對、手機兩連結＋乾淨掃描時隱藏查看錯誤

- [x] **Task 2 — 設定頁與編輯視窗（AC #5, #6）**
  - [x] `ScannerSettings.tsx`：檔頭、掃描按鈕 hover／active／停用、手機 `p-4`、上次掃描 `text-muted`、下拉 focus token
  - [x] `MediaLibraryManager.tsx`：檔頭、新增按鈕實線（稿沒有虛線可畫，也與 N3 精靈同款）
  - [x] `LibraryCard.tsx`：檔頭、`bg-primary`（在 bg-secondary 表單卡裡）、類型 chip 改藥丸、⋮ 按鈕補 `aria-label`／`aria-expanded`
  - [x] `LibraryEditModal.tsx`：視窗 bg-secondary、欄位 bg-primary、路徑列 44px、移除鈕中性、說明 14px、`text-[13px]`→`text-sm`、按鈕列（桌機靠右／手機平分）、手機邊距

- [x] **Task 3 — 設計稿（AC #2, #4, #7）**
  - [x] E1-D／E1-M：刪扁平資料夾列、建 `Component/LibraryCard`、各 instance 兩張＋新增媒體庫按鈕、掃描按鈕 `$text-on-accent`、E1-M 上次掃描補耗時、畫布高度 900→1024／844→1048（卡片比資料夾列高）
  - [x] E2-D／E2-M：刪比對與分隔點（手機兩列併一列）、錯誤圖示 `$error-text`
  - [x] E3-D／E3-M：圖示改赭色三角、倒數條 `$text-muted`、手機浮卡移到分頁列上方
  - [x] E4-M：四張 v1 PosterCard 換成 `PosterCard-v2/Unmatched`
  - [x] E5-D／E5-M：「儲存變更」＋`$text-on-accent`、新增路徑列旁加「＋」按鈕、E5-M 說明改 Body
  - [x] 存檔（AppleScript）→ 匯出 188/188，只有 flow-e 9 張與 `pen-tokens.json` 有差 → `check-design-tokens.py` 一致（73 母版）

- [x] **Task 4 — 驗證（AC #8, #9）**
  - [x] `pnpm nx test web` 3365／3365、typecheck、lint 0 errors
  - [x] 視覺：`--update-snapshots=all` 後只保留 7 個夾具的 15 張 `-darwin`，`git rm` 15 張 `-linux`
  - [x] 互動實測（`/test/gallery?fixture=<id>`）：掃描按鈕 靜止／hover／active 三值互異、新增媒體庫 hover 有變化、取消掃描外框

- [x] **Task 5 — CR 與裁定（AC #1）**
  - [x] `useScanProgress.ts`／`scannerService.ts`：讀 `scan_complete` 的 `filesCreated`／`filesUpdated`（輪詢退路沒有這兩個數字 → `undefined`，不印 0）
  - [x] `ScanProgressCard`：`scanSummaryParts`／`scanHadProblems`／`SCAN_PROBLEMS_DESTINATION`、單一連結、滑過後離開重新倒數
  - [x] `ScanProgressSheet`：同一套摘要（手機兩行）、單一連結、44px、`pb-2 sm:pb-4`
  - [x] 觸控目標：`LibraryEditModal`、`LibraryCard`
  - [x] E3-D／E3-M 稿：摘要文字、「查看無法匯入與錯誤」、拿掉查看錯誤
  - [x] e2e TC070、gallery 夾具、spec 更新；視覺重拍一次（仍是同樣 7 個夾具 15 張）

## Dev Notes

### 不要做的事

- **不要把「查看錯誤」做成一個新的錯誤篩選頁。** 那是新功能；本張只讓連結去唯一真的有錯誤可看的地方。
- **不要把「比對」數字改成「找到 − 未比對」之類的推算。** 掃描中根本沒有比對這件事；推一個數字填進去，就是把抄來的換成算出來的假數字。
- **不要動媒體庫的「未比對」／「未匹配」兩個詞。** E4 稿說「未比對」、程式碼篩選說「未匹配」，選哪個是文案裁定，而且牽動字幕同意清單 Sally 裁定過的徽章 → `disc-2026-09-unmatched-two-words`。掃描通知已改成「無法匯入」，那是另一件事。
- **不要重畫 E4 的版面。** E4 是瀏覽頁（`LibraryBrowseV2`）的一個篩選狀態，版面歸 `dsr-1`，已在該條目加耦合條款。本張只修了連結與 E4-M 的卡片。
- **不要改手機完成通知的位置到頂端。** 分頁列上方、會自己消失、不能按住暫停，都是 Alexyu 2026-08-27 在程式碼註解裡留下的裁定；稿改過來配合碼。
- **`LibraryCard` 頁尾「自動處理免費字幕」的青碧**沒有動：那是 J5-D 規格頁的裁定範圍（dsr-9 已結案），而且規則是「有在發生＝綠」；在 2026-09-10 詞彙改版後它是否該改泥金，需要另外判斷，不在 Flow E 範圍。

### 固定詞彙檢查

| 位置 | 用色 | 結論 |
| --- | --- | --- |
| 掃描進度條 | 泥金 | ✅ 正在跑 |
| 錯誤計數（>0） | 硃砂 `--error-text` | ✅ 壞了 |
| 完成圖示（有錯誤） | 赭三角 | ✅ 你要求掃描，有檔案沒進來 |
| 完成圖示（無錯誤） | 青碧勾勾 | ✅ 有答案了 |
| 倒數條 | 中性 | ✅ 倒數不是工作在跑 |
| 路徑狀態點 | 青碧已連線／硃砂無法存取 | ✅ |
| 移除路徑、⋮ 選單 | 中性 | ✅ 編輯不是壞了 |

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **YES，但沒有新增讀時鐘的程式。** `ScannerSettings.tsx` 的 `formatLastScan` 原本就用 `new Date(lastAt)` 格式化**傳進來的時間戳**（不讀現在時間），本張只改了該段的 class；`settings-scanner-settings` 夾具的 lastAt 是固定值，基準線不隨時間變。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` E1-D (KvZSc) … E5-M (P0P82x)、`Component/LibraryCard (ZbfJR)`]
- [Source: `apps/web/src/routes/library.tsx#validateSearch`] — `unmatched: search.unmatched === true`
- [Source: `apps/api/internal/services/scanner_service.go#ScanProgress`] — 沒有比對欄位
- [Source: `apps/web/src/hooks/useScanProgress.ts`] — `filesProcessed` 的估算
- [Source: `apps/api/cmd/api/main.go`] — `slog` 經 `logger.NewDBHandler` 寫進 `system_logs`
- [Source: DESIGN.md#Shadow Vocabulary] — 懸浮進度卡＝`--shadow-lg`
- [Source: `_bmad-output/planning-artifacts/multi-library-ux-spec.md` §2] — 媒體庫卡片的原始規格

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，2026-09-14（SM create-story 與 dev 同一個 session）

### Debug Log References

- `pnpm nx test web`：**3367 / 3367 passed**（261 檔，CR 修正後重跑）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `pnpm run lint`：0 errors、128 warnings（本 story 改動的檔案零 warning）
- `python3 scripts/check-design-tokens.py`：一致（82 變數／188 畫面／**73 母版**，+1 LibraryCard）
- `export-pen-screenshots.py`：188/188，flow-e 9 張有差（E4-D 沒改，零差異）
- `CI=1 npx playwright test --project=visual --update-snapshots=all`：1 passed（8.3m）；保留 7 個夾具 15 張 `-darwin`

### Completion Notes List

- 🔗 **AC Drift: N/A**——`7-4`（掃描進度）與 `7b-4`（媒體庫管理）都是 done；本張沒有推翻它們的 AC，是修掉它們留下的兩個死連結與一個複製數字。
- 🎭 **A11y Pre-Flight: PASS**——`LibraryCard` 的 ⋮ 按鈕原本**沒有無障礙名稱**，補 `aria-label="{名稱} 的操作"`＋`aria-expanded`；手機完成通知的關閉鈕與取消掃描鈕補到 44px；統計列的分隔點與圖示 `aria-hidden`。
- 🖱️ **互動實測**：掃描按鈕 `136,98,8` / `114,82,5` / `91,65,3`（日巡的靜止／hover／active）三值互異——改前 hover 與靜止同色；新增媒體庫 hover 從透明變 `bg-tertiary`。編輯視窗的按鈕在 gallery 裡量不到 hover（夾具同時渲染三份 `fixed inset-0` 視窗互相蓋住），沿用已驗證過的同一組 token。
- 🔍 **對抗式 code review（獨立 agent）：0 HIGH / 3 MED / 4 LOW。**
  - **MED 比對成功是推算數字**、**MED 未比對與點進去的清單不是同一批** → 送 Alexyu 裁定「全部說真話」，見 Context 與 AC #1。
  - **MED 手機觸控目標不到 44px**（完成通知連結、編輯視窗移除／新增路徑鈕與輸入框、LibraryCard ⋮ 與選單項目）→ 全部補到 `min-h-11`／`size-11`，桌機維持原尺寸。
  - **LOW 滑過完成卡片後永遠不會自己消失**（原本就有）→ 離開時重新開始倒數，倒數條同步重來；補 spec。
  - **LOW 640–767px 手機通知貼齊螢幕底**→ 外層 `pb-2 sm:pb-4`，E3-M 稿同步上移 8px。
  - **LOW 註解與事實不符**（卡片檔頭寫右下 400px、查看錯誤「原本哪裡都沒去」）→ 更正。
  - **LOW 同一個錯誤數兩種顏色**（標題赭、統計硃砂）→ **沒有改**：標題圖示說的是整次掃描「有你要的沒發生」（含無法匯入），統計列的錯誤數說的是「壞了」，是兩個主張。
- ✅ Task 1–5 全數完成。

### Discovery Triage

- **YES**，三項：
  - **① 兩個死連結、一個複製數字、手機缺連結（expand in place）**——同一個元件、同一張稿，就地修掉（AC #1–#3）。
  - **③ 「未比對」與「未匹配」兩個詞（FILE）**→ `disc-2026-09-unmatched-two-words`。
  - **③ E4 的版面仍是舊瀏覽頁（FILE → 耦合條款）**→ 寫進 `dsr-1-flow-a-browse-v2` 條目。
  - **① 滑過桌機通知後永遠不消失、手機觸控目標（expand in place）**→ CR 抓到，就地修掉。

### File List

- `apps/web/src/components/scanner/ScanProgress.tsx`（M）
- `apps/web/src/hooks/useScanProgress.ts`（M）· `useScanProgress.spec.ts`（M）
- `apps/web/src/services/scannerService.ts`（M）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（M，完成通知夾具帶新增／更新）
- `tests/e2e/scan-progress.spec.ts`（M，TC070 改斷言新連結）
- `apps/web/src/components/scanner/ScanProgressCard.tsx`（M）· `ScanProgressCard.spec.tsx`（M）
- `apps/web/src/components/scanner/ScanProgressSheet.tsx`（M）· `ScanProgressSheet.spec.tsx`（M）
- `apps/web/src/components/settings/ScannerSettings.tsx`（M）
- `apps/web/src/components/settings/MediaLibraryManager.tsx`（M）
- `apps/web/src/components/settings/LibraryCard.tsx`（M）
- `apps/web/src/components/settings/LibraryEditModal.tsx`（M）
- `ux-design.pen`（M）· `_bmad-output/pen-tokens.json`（M）
- `_bmad-output/screenshots/flow-e-scanner/{e1-d,e1-m,e2-d,e2-m,e3-d,e3-m,e4-m,e5-d,e5-m}.png`（M）
- `DESIGN.md`（M，元件清冊）
- `tests/visual/components.visual.spec.ts-snapshots/components/{scanner-scan-complete-toast,scanner-scan-progress-card,scanner-scan-progress-sheet,settings-library-card,settings-library-edit-modal,settings-media-library-manager,settings-scanner-settings}/*-darwin.png`（M ×15）· `*-linux.png`（D ×15）
- `_bmad-output/implementation-artifacts/dsr-5-flow-e-scanner.md`（A）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（M）

## Change Log

- 2026-09-14 — create-story ＋ dev 同 session 完成，狀態 → review。
- 2026-09-14 — 對抗式 CR 3M／4L；Alexyu 裁定完成通知「全部說真話」；其餘 CR 項目就地修正（1 項說明不改）。
