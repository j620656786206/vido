# Disc：只有圖示的地方，提示要用 Vido 自己的提示框（a：靠瀏覽器 title 的地方＋共用的 ✕）

Status: done

**Source:** `disc-2026-10-icon-only-native-title-audit`（2026-10-05 Alexyu 裁定，J11-D 規則 6）。2026-10-13 Alexyu 從 backlog 選 1。

## 查到的事（2026-10-13，唯讀稽核，main `1abdb61c`）

規則（J11-D `FiD2R`）：「只有圖示、沒有文字的狀態，一律用這個提示框，不可以只靠瀏覽器的 title。」提示框是 `components/ui/Tooltip`。正確的範例有：側欄（`AppSidebar`／`SidebarNavItem`／`LogoutButton`／`ThemeToggle`／`SidebarFooter`）、季清單字幕圖示（`EpisodeList.tsx` `SubtitleStatusIcon`）。

全站掃出約 45 處不符合，分三類：

1. **還靠瀏覽器 `title` 的圖示**（這張做）：
   - 設定 › 備份，每列四個圖示按鈕：還原／驗證完整性／下載／刪除（`BackupTable.tsx:117-150`）。
   - 下載頁的「清單／表格」切換（`DownloadsBrowseV2.tsx:580`）。
   - 詳情頁「觀看平台」的平台 logo（`StreamingAvailability.tsx:49`）。只有圖，名字只在 title 裡。
   - 名詞表的「S2」季標（`GlossaryRowV2.tsx:167`）、下載表格的「qB」標（`DownloadsTableV2.tsx:158`）。縮寫的意思只在 title 裡。
2. **共用對話框的 ✕**（這張做）：`ui/Dialog.tsx:76-83`。12 個對話框共用，沒有提示，而且唯一的名字是英文的 `Close`。改一個元件就全部修好。
3. **只有 aria-label、沒有任何看得到的提示的圖示按鈕**（約 40 處，拆到 `disc-2026-10-icon-only-controls-tooltip`）：返回箭頭、輪播左右、分頁、排序、篩選、批次動作列、各種清除 ✕、7 個自己畫的 ✕……

另外有幾個**有字、但原因只藏在 title 裡**的標籤（選片清單「未匹配／資料夾無法寫入／無法估價」、模型品質、NZBGet「尚未支援」、篩選預設「右鍵或長按可刪除」）。它們不是只有圖示，不在規則 6 的範圍；要不要也改，交 Sally（記在 b 單）。

## 設計裁定（Sally，2026-10-13）

- **狀態類**（logo、S2、qB）：照季清單的做法——觸發點是真的按鈕（鍵盤到得了，按了不做別的事）、`delay={0}`、`openOnPress`（手機點一下就出現），名字跟提示框同一句。
- **控制類**（備份動作、檢視切換、✕）：滑鼠移過、鍵盤切到時出現提示（沿用側欄的預設延遲）。手機點下去就是執行動作，不另外開提示。
- **對話框 ✕**：提示框與名字都寫「關閉視窗」。不寫「關閉」，是因為很多對話框底部已經有一顆叫「關閉」的按鈕，同名的兩顆會讓螢幕閱讀器分不清。提示框出現在 ✕ 下方。
- 不畫新設計稿：提示框本身就是現成元件（J11-D），這張只是把它套到該用的地方。

## Story

身為用滑鼠、鍵盤或手機的人，
我希望只有圖示的地方，移過去、切到、點一下都看得到它是什麼，
這樣不用猜，也不用等瀏覽器慢吞吞的小黃框。

## Acceptance Criteria

1. 上面第 1 類的每一處都改用 `Tooltip`，拿掉 `title`。提示文字跟原本 title 一樣；名字（aria-label）不變。
   - 平台 logo 包一顆按鈕當觸發點；名字＝平台名。
   - S2 改成按鈕觸發點；名字＝提示句（「第 2 季官方字幕的寫法」）。
   - ~~qB~~：dev 時移到 b 單（見 Completion Notes）。
2. `ui/Dialog.tsx` 的 ✕：包 `Tooltip`（`side="bottom"`），名字從英文 `Close` 改成「關閉視窗」。`closeClassName="hidden"` 的對話框照舊藏起來。
3. 所有用 `Close` 找這顆按鈕的測試（vitest 與 `tests/e2e`）改成「關閉視窗」。
4. `DESIGN.md` 寫進規則：只有圖示的狀態與控制一律用 `components/ui/Tooltip`，不可只靠 `title`；狀態類 `delay={0}`＋`openOnPress`；控制類用預設；觸發點必須能用鍵盤到。
5. 測試：每一處至少一條「滑鼠移過（或鍵盤切到）會出現提示、沒有 title 屬性」；狀態類多一條「點一下會出現」。
6. 檢查：vitest、`tsc`、`pnpm run lint:all`、`format:check`、visual（提示框預設是關的，基準圖不應該變）。

## Tasks / Subtasks

- [x] T1 `ui/Dialog.tsx` ✕＋所有找 `Close` 的測試（AC #2、#3）
- [x] T2 備份動作、下載檢視切換（AC #1）
- [x] T3 平台 logo、S2、qB（AC #1）
- [x] T4 `DESIGN.md`（AC #4）
- [x] T5 測試與檢查（AC #5、#6）

## Dev Notes

- 拿掉 data-testid 或改名前先 grep `tests/e2e`。
- `Tooltip` 的觸發點用 `render={children}` 合併，子元素必須是單一個可以接 ref 的元素。

### Time-dependent visual coverage

N/A — 不讀現在時間。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- `ui/Dialog.tsx` 的 ✕：包 `Tooltip`（`side="bottom"`），名字從英文 `Close` 改成「關閉視窗」，圖示加 `aria-hidden`。className 一字不改（原本的 byte-identical 守門測試照樣綠）。
  - 8 個 vitest spec 和 3 個 e2e spec 原本用 `Close` 找這顆按鈕，改成用角色＋名字「關閉視窗」找。
  - `ParseFailureCard.spec` 那顆 `Close` 是它自己 mock 的按鈕，維持原樣。
- 設定 › 備份四個列動作、下載頁「清單／表格」切換：拿掉 `title`，改包 `Tooltip`。
- 觀看平台 logo：外面包一顆按鈕（名字＝平台名，圖片改成裝飾 `alt=""`）＋`Tooltip`（`delay={0}`、`openOnPress`）。
- 名詞表「S2」：改成按鈕＋`Tooltip`（`delay={0}`、`openOnPress`），名字＝「第 2 季官方字幕的寫法」。
- **qB 標沒做，移到 b 單**：下載表格每一列都有一顆，全部變成按鈕的話，表格每列多一個 Tab 停點（50 列就多 50 下），而且它在每一列都一樣。要不要改成表頭一次說明，交 Sally。
- `DESIGN.md` 新增 §Tooltips：狀態類與控制類的做法、✕ 叫「關閉視窗」的理由。
- 檢查：
  - vitest 全部 288 檔、4,455 個測試全綠。
  - `tsc`、`pnpm run lint:all`（0 errors）、`format:check` 綠。
  - 本機 visual 綠，基準圖沒有變（提示框預設是關的）。
  - e2e：三個改過的 spec 在 chromium 25 條全過。webkit 的 11 條在本機沒裝瀏覽器，跑不起來，交給 CI。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／2 Low，皆接受。

- **L1（接受）**：如果一個對話框裡唯一能用鍵盤到的就是 ✕，用鍵盤打開它時，焦點會落在 ✕ 上，「關閉視窗」提示會跟著出現。用滑鼠打開不會（不算 `:focus-visible`）。這是給鍵盤使用者的提示，可以接受。
- **L2（接受）**：✕ 原本的 `focus:outline-none` 沒有動（全域 `:focus-visible` 規則另外處理）。對話框 ✕ 的大小問題是 `disc-2026-09-dialog-close-target-44px`，不在這張。

### File List

- `apps/web/src/components/ui/Dialog.tsx`、`.spec.tsx`
- `apps/web/src/components/settings/BackupTable.tsx`、`.spec.tsx`
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx`、`.spec.tsx`
- `apps/web/src/components/media/StreamingAvailability.tsx`、`.spec.tsx`
- `apps/web/src/components/subtitle/GlossaryRowV2.tsx`、`.spec.tsx`
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`（只改註解）
- 改用「關閉視窗」找 ✕ 的 spec：`RestoreConfirmDialog`、`LibraryEditModal`、`ManageSubtitleDialogV2`、`GlossaryPanelV2`、`GenerationBatchDialogV2`、`ConfirmGenerationDialog`、`GenerationConsentView`
- `tests/e2e/manage-subtitle-mobile.spec.ts`、`batch-consent-mobile.spec.ts`、`glossary-mobile.spec.ts`
- `DESIGN.md`
- `_bmad-output/implementation-artifacts/disc-2026-10-icon-only-native-title-audit.md`、`sprint-status.yaml`
