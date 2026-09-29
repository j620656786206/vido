# Story: disc-2026-09-ime-enter-submits-mid-composition — 打注音按 Enter 選字，不會再被當成「送出」

Status: review

<!-- SM Bob create-story 2026-09-29。Alexyu 從推薦清單選定（「做注音」）。
     行號為 main `d4cec789`，改動前；每條「查到的事」都是本次親自讀過的位置（retro-dsr-AI4）。 -->

## Story

身為用注音（或倉頡、拼音）打字的人，
我要在搜尋框、名稱欄、演員欄裡按 Enter 選字時，**只是選字**，
而不是把還沒打完的字（例如「ㄓㄨˋ」）直接送出去搜尋或存檔。

## 背景（查到的事，附親自讀過的位置）

**共用工具已經有了：**

- `apps/web/src/utils/keyboard.ts:12-14` — `isImeComposing(e)`：`e.nativeEvent.isComposing || e.keyCode === 229`（Chrome／Firefox 用前者，Safari 用後者）。測試 `utils/keyboard.spec.ts:18-33`。
- 已套用的先例（dsr-6c）：`components/subtitle/GlossaryPanelV2.tsx:159-162`（`onAddFormKeyDown` 開頭 `if (isImeComposing(e)) return;`，Esc 也擋）、`:185`、`:200`（`e.key === 'Enter' && !isImeComposing(e)`）；`GlossaryRowV2.tsx:128`。測試寫法：`GlossaryRowV2.spec.tsx:229-238`（`it.each` 兩種：`{ key: 'Enter', isComposing: true }`、`{ key: 'Enter', keyCode: 229 }`，`fireEvent.keyDown`）、`GlossaryPanelV2.spec.tsx:455-462`（IME 的 Esc）。

**還沒擋的地方**（`git grep -n "'Enter'" -- apps/web/src ':!*.spec.*' ':!*.test.*'` 全部命中逐一打開看過）：

| # | 位置 | 按 Enter 會做什麼 | 產品上有掛嗎 |
|---|------|------------------|-------------|
| 1 | `components/search/InstantSearchBar.tsx:102-108` | `preventDefault` 後跳到建議項目或 `submitAll()`（導到 `/search?q=…`，`:80-85`）。**同一個 switch 的 `:110-116` Esc 會清空整個輸入框** | ✅ 頂欄搜尋，`shell/AppShellV2.tsx` |
| 2 | `components/search/SavePresetDialog.tsx:139-141` | `handleSave()` 存篩選預設（`:50-71`） | ✅ `search/DiscoverBrowseV2.tsx` |
| 3 | `components/metadata-editor/CastEditor.tsx:68-79` | Enter → `add()` 加演員；**Esc → 取消並關掉輸入框** | ✅ `metadata-editor/MetadataEditorDialog.tsx` |
| 4 | `components/settings/LogFilters.tsx:41-45` | `onKeywordChange(inputValue)` 用關鍵字篩日誌 | ✅ `settings/LogsViewer.tsx` |
| 5 | `components/dashboard/QuickSearchBar.tsx:100-116` | 下拉開著時 Enter 選最近搜尋；Esc 收下拉 | ❌ 只在 `routes/test/-gallery.fixtures.tsx:105` |
| 6 | `components/subtitle/SubtitleSearchDialog.tsx:209` | `handleSearch()` 搜字幕 | ❌ 只經 `media/MediaDetailPanel.tsx`，而它只在夾具（`-gallery.fixtures.tsx:195`） |

掛載點查法：`git grep -ln "<元件名\b" -- apps/web/src ':!*.spec.*' ':!routes/test/*' ':!*fixtures*'`。

**與 sprint-status 原條目的差異（建單時更正）：**

- 原條目列的 `MetadataEditorDialog.tsx:421` 已不存在——poster-upload-a 改寫後整個檔只有 419 行，檔內沒有任何 `onKeyDown`／`'Enter'`（`grep -n "onKeyDown\|'Enter'"` 無結果）。它是 `<form onSubmit>`（`:242-246`），Enter 由瀏覽器的隱式送出處理，不在本單範圍。⚠️ 未查證：Safari 選字的 Enter 會不會觸發表單隱式送出（jsdom 測不到，dev 不需處理；只在 Discovery Triage 記錄）。
- 原條目**漏了 `InstantSearchBar`**（`switch` 寫法 `case 'Enter':`，原條目用 `e.key === 'Enter'` 的樣式找，所以沒抓到；本單的 `'Enter'` grep 有抓到）——它是最常用的頂欄搜尋，本單最重要的一處。
- `MediaFileCard.tsx:70,158` 的 Enter／Space 是按鈕語意，不是文字框，不算（原條目已註明）。

## Acceptance Criteria

1. 上表 6 個文字框：選字用的 Enter（`isComposing: true` 或 `keyCode 229`）**什麼都不做**——不搜尋、不存、不加演員、不篩選、不跳頁，也不 `preventDefault`（讓輸入法自己把字選好）。
2. 同一個處理函式裡的 Esc 也一樣：選字中的 Esc 不清空頂欄搜尋（#1）、不取消演員輸入框（#3）、不收 QuickSearchBar 下拉（#5）。
3. 一般的 Enter／Esc（沒有在選字）行為**完全不變**——現有測試全部照過，不改既有斷言。
4. 一律用 `utils/keyboard.ts` 的 `isImeComposing`，不另寫判斷。寫法跟先例一致：整個 handler 開頭 `if (isImeComposing(e)) return;`（有 Esc 的 #1、#3、#5），或單行 `e.key === 'Enter' && !isImeComposing(e)`（只有 Enter 的 #2、#4、#6）。
5. 測試：6 個元件各自的 spec 各加 IME Enter 兩種情況（`isComposing: true`、`keyCode: 229`）斷言「沒送出」；#1、#3 再各加一條 IME Esc 斷言「沒清空／沒取消」。每條新測試都要先在改動前跑一次看它**紅**（證明測試真的抓得到）。
6. 視覺零變動（沒有任何外觀改動）。
7. （CR MED-1 併入，Rule 24 lane ①）只有 Esc、沒有 Enter 的搜尋框也一樣：`/search` 頁的 `search/SearchBar.tsx:43-50`（產品：`routes/search.tsx`）與 `library/LibrarySearchBar.tsx:49-56`（只在夾具），選字中的 Esc 不清空框、不觸發 `onSearch('')`。

## Tasks / Subtasks

- [x] Task 1 — 產品上有掛的 4 處（AC #1–#4）
  - [x] 1.1 `InstantSearchBar.tsx` `handleKeyDown` 開頭加 guard（擋 Enter 與 Esc）
  - [x] 1.2 `SavePresetDialog.tsx:140`
  - [x] 1.3 `CastEditor.tsx` `onKeyDown` 開頭加 guard（擋 Enter 與 Esc）
  - [x] 1.4 `LogFilters.tsx:42`
- [x] Task 2 — 只在夾具的 2 處，一行改法相同（AC #1, #2, #4）
  - [x] 2.1 `QuickSearchBar.tsx` `handleKeyDown` 開頭加 guard（放在 `:101` 的 early return 之前）
  - [x] 2.2 `SubtitleSearchDialog.tsx:209`
- [x] Task 3 — 測試（AC #5）：先寫測試看紅，再改碼看綠
- [x] Task 4 — `pnpm nx test web`、`lint`、typecheck、`format:check` 全綠（AC #3, #6）
- [x] Task 5 — CR 追加：`SearchBar`、`LibrarySearchBar` 的 Esc 加 guard ＋ 各一條測試（先紅後綠）；`InstantSearchBar`／`CastEditor` 的 IME Enter 測試加斷言「沒有 `preventDefault`」（AC #1, #7）

## Dev Notes

- **guard 要放在 `preventDefault` 之前。** `InstantSearchBar.tsx:103` 與 `CastEditor.tsx:71` 在 Enter 分支一進去就 `preventDefault()`。選字的 Enter 是輸入法的，頁面不該插手（⚠️ 未查證：各瀏覽器對 composing 中 `preventDefault` 的反應不一，不作為 AC 依據，只是 guard 放最前面最保險）。所以 guard 在整個 handler 最前面 `return`。
- **CastEditor 的 Esc 不必動對話框那一側。** `MetadataEditorDialog.tsx:110-116` 的 `onEscapeKeyDown` 已經看 `data-escape-local`（`CastEditor.tsx:65`）不讓對話框關；本單只讓 CastEditor 自己在選字時不取消。
- **型別：** `isImeComposing` 吃 React 的 `KeyboardEvent`（`keyboard.ts:1`）。`QuickSearchBar.tsx:100` 是 `React.KeyboardEvent`、`InstantSearchBar.tsx:88` 是 `React.KeyboardEvent<HTMLInputElement>`，都相容。
- **測試寫法照抄** `GlossaryRowV2.spec.tsx:229-238` 的 `it.each`。jsdom 的 `fireEvent.keyDown(el, { key: 'Enter', isComposing: true })` 會把 `isComposing` 帶進原生事件（先例已證實可用）。
- **不要做：** 不要改 `keyboard.ts`；不要順手修對話框層級的 Esc（見 Discovery Triage）；不要動 `MediaFileCard`。

### Project Structure Notes

- 只動 `apps/web/src/components/**` 的 6 個元件與各自 spec；純前端，沒有後端工作（Cross-stack split：後端 0、前端 4 → 單張）。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 原條目：`sprint-status.yaml` `disc-2026-09-ime-enter-submits-mid-composition`（dsr-6c 立案）
- 先例：dsr-6c AC #7（`isImeComposing` 的由來）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- RED：先加 15 條 IME 測試，改碼前跑 6 個 spec → 15 failed／65 passed（新增的 3 條「一般 Enter 照常」測試在改動前就綠，證明它們守的是舊行為）。
- GREEN：改碼後同 6 個 spec ＋ `utils/keyboard.spec.ts` → 84 passed。

### Completion Notes List

- 6 處全部改用 `isImeComposing`：有 Esc 的三處（`InstantSearchBar`、`CastEditor`、`QuickSearchBar`）在 handler 最前面 `return`；只有 Enter 的三處（`SavePresetDialog`、`LogFilters`、`SubtitleSearchDialog`）用 `e.key === 'Enter' && !isImeComposing(e)`，與 dsr-6c 先例一致。
- 新增測試 18 條：每處 IME Enter 兩種（`isComposing`、`keyCode 229`）＝12；IME Esc 3 條（`InstantSearchBar` 不清空、`CastEditor` 不取消、`QuickSearchBar` 下拉不收）；一般 Enter 照常 3 條（`SavePresetDialog`、`QuickSearchBar`、`SubtitleSearchDialog`——這三處原本沒有直接測 keydown Enter 的測試）＋ `InstantSearchBar`／`CastEditor`／`LogFilters` 原有的 Enter 測試照過。
- 全量回歸：`pnpm nx test web` 291 檔／4,344 條全綠；`pnpm nx test api` 綠（本單沒動後端）。`web:typecheck` 綠；eslint 對 12 個改動檔 0 errors（3 條 `no-explicit-any` warning 都在既有行，不是本單新增）；prettier 綠。
- 🔗 AC Drift: NONE（checked: `CastEditor|InstantSearchBar|SavePresetDialog` across `_bmad-output/implementation-artifacts/*.md` — 21 hits；一般 Enter／Esc 行為不變，全是 REUSE。E2E 掃描 `instant-search-input|preset-name-input|log-keyword-input` in `tests/`：Playwright `fill` + `press('Enter')` 送的是一般 Enter（非 composing），不受影響）。
- 📎 Contract Stamps: NONE（本單不定義也不引用任何 `[@contract-v*]`，沒有 wire contract）。
- 🎭 A11y Pre-Flight: PASS（6 components checked, 0 jsx-a11y warnings on touched files, 0 introduced by this story；只改 keydown 分支，沒動 ARIA、焦點或結構）。
- 🎨 UX Verification: SKIPPED — no UI changes in this story（沒有任何外觀改動，只改按鍵判斷）。
- 對抗式 CR（fresh-context 子代理）：0 HIGH／1 MED／6 LOW。已修：MED-1（`/search` 頁 `SearchBar` 的 Esc）＋ LOW-2（`LibrarySearchBar`）→ AC #7；LOW-4／5（`InstantSearchBar` 測試靠 50ms 等待、沒測 `preventDefault`）→ `InstantSearchBar`、`CastEditor` 的 IME Enter 測試改成斷言 `fireEvent.keyDown(...)` 回傳 `true`（沒被 `preventDefault`），並暫時拿掉 guard 驗證 6 條都會紅。LOW-3（媒體庫頁的 document Esc 快捷鍵）→ 放寬 `disc-2026-09-dialog-esc-closes-mid-composition` 的範圍。LOW-6／7 未查證，見下。
- 最終新增測試 20 條（原 18 ＋ `SearchBar`、`LibrarySearchBar` 各 1）。CR 修正後相關 21 個 spec 檔／271 條全綠，改動檔 eslint 0 errors。
- ⚠️ 未查證（CR LOW-6）：`CastEditor` 原本對**所有** Enter（含選字）都 `preventDefault`，順帶擋住了「修改資訊」表單的隱式送出；現在選字的 Enter 不再被攔。Chrome／WebKit 的隱式送出走 `keypress`，而 keyCode 229 不會發 `keypress`，理論上安全——請 Alexyu 在 Safari 實機試一次。
- ⚠️ 未查證（CR LOW-7）：部分 Android 鍵盤在英文字還有底線建議時，按「前往」可能送 keyCode 229 → 手機頂欄搜尋第一下被忽略、要再按一次。名詞面板早就用同一個 guard；手機實測再決定要不要處理。
- 限制（誠實記錄）：jsdom 只能模擬事件旗標，無法跑真的注音輸入法；真機驗證要 Alexyu 在瀏覽器實際用注音打字按 Enter。

### Discovery Triage

- ③ backlog-with-carry-forward-link — **對話框層級的 Esc 不看輸入法**：Radix 的 `useEscapeKeydown`（`node_modules/.pnpm/@radix-ui+react-use-escape-keydown@1.1.1…/dist/index.mjs:7-11`）只看 `event.key === "Escape"`，沒有看 `isComposing`；自製的 `document` keydown（如 `SavePresetDialog.tsx:32-38`）也一樣。在對話框的文字框裡用注音打字、按 Esc 放棄選字 → 整個對話框被關掉（「修改資訊」會丟掉整份編輯）。CR LOW-3 補：頁面層級的 document Esc 快捷鍵同理（`LibraryBrowseV2.tsx:522-534`：批次選取時在頂欄用注音按 Esc → 選取被清掉），一併歸入該條目。→ `disc-2026-09-dialog-esc-closes-mid-composition`（P3，建單時立案）。
- ③ backlog-with-carry-forward-link — `QuickSearchBar` 沒有產品掛載點（只在夾具），不在 `disc-2026-09-unmounted-v1-components` 的清單上 → 已補進該條目。
- 未查證、不立案：Safari 選字 Enter 是否觸發 `<form>` 隱式送出（`MetadataEditorDialog`）——jsdom 無法驗；若 Alexyu 實機遇到再立案。

### File List

- apps/web/src/components/search/InstantSearchBar.tsx
- apps/web/src/components/search/InstantSearchBar.spec.tsx
- apps/web/src/components/search/SavePresetDialog.tsx
- apps/web/src/components/search/SavePresetDialog.spec.tsx
- apps/web/src/components/metadata-editor/CastEditor.tsx
- apps/web/src/components/metadata-editor/CastEditor.spec.tsx
- apps/web/src/components/settings/LogFilters.tsx
- apps/web/src/components/settings/LogFilters.spec.tsx
- apps/web/src/components/dashboard/QuickSearchBar.tsx
- apps/web/src/components/dashboard/QuickSearchBar.spec.tsx
- apps/web/src/components/subtitle/SubtitleSearchDialog.tsx
- apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx
- apps/web/src/components/search/SearchBar.tsx
- apps/web/src/components/search/SearchBar.spec.tsx
- apps/web/src/components/library/LibrarySearchBar.tsx
- apps/web/src/components/library/LibrarySearchBar.spec.tsx
- _bmad-output/implementation-artifacts/disc-2026-09-ime-enter-submits-mid-composition.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

| Date | Change |
|------|--------|
| 2026-09-29 | create-story（SM Bob）：建單，更正原條目（`MetadataEditorDialog:421` 已不存在、補上 `InstantSearchBar`），立案 `disc-2026-09-dialog-esc-closes-mid-composition`。 |
| 2026-09-29 | dev-story（Amelia）：6 處 Enter（＋3 處同 handler 的 Esc）改用 `isImeComposing`；18 條新測試；全量回歸綠 → review。 |
| 2026-09-29 | /ship 對抗式 CR：併入 AC #7（`SearchBar`、`LibrarySearchBar` 的 Esc），IME Enter 測試改斷言「沒被 `preventDefault`」，共 20 條新測試。 |
