# Story（前端）：季清單每一集的字幕圖示改看真實結論，圖示都有提示框

Status: done

**Source:** 傘狀單 `disc-2026-10-episode-list-subtitle-badge`。**依賴 `-a`**（季清單 API 的 `chinese_subtitle`／`chinese_subtitle_sources`，AC #1 `[@contract-v1]`）。
**設計稿：** `J11-D`（`w2Opax`）規則 3–6、`J2-D`（圖示語法）。

> ⚖️ SM 裁定（Bob 2026-10-07）：從傘狀單拆出，切分線與理由見 `-a`。

## 開工前要先問 Sally 的事

- `-a` 的結論有五個值，J11-D 只畫了四種有結論的樣子＋生成中。多出來的 **`zh`（有中文、分不出繁簡，例：片內 `chi` 軌沒有標題）** 要用什麼圖示、提示框寫什麼，要 Sally 定（或補進 J11-D）。

- **CR L1（-a）**：片內字幕還沒讀過、但 Vido 紀錄是「線上找不到」（`not_found`）的集數，結論會是 `none`（電影同規則），背景工作讀完片內字幕後才可能變成「有」。提示框文案要不要對這種情況講清楚，或要不要改規則（需 Alexyu 裁定），開工時一起問。

- ⚖️ **Alexyu 2026-10-07 裁定：1A 2A。** 1A：`zh` 跟有繁中一樣用綠色打勾，提示框寫「有中文字幕（繁簡未知）」（-c 合併後只剩圖片字幕會這樣）。2A：不改規則；片內字幕還沒讀過時，「缺中文」的提示框第二行寫「還在讀片內字幕，讀完可能會變」。為此後端加一個欄位 `embedded_subtitles_read`（additive）。

## Acceptance Criteria（沿用傘狀單 AC #4、#5、#7、#8，結論改用 `-a` 的五值）

1. **圖示照 J11-D**：`zh_hant` → `circle-check`＋成功色；`zh_hans` → `circle-alert`＋錯誤色；`none` → `circle-x`＋錯誤色；`unknown` → `minus`＋淡色；`zh` → 依 Sally 裁定；生成中照舊（四個 spinner 狀態優先）。沒有本地檔的集數不顯示圖示。
2. **提示框照 J11-D 規則 3–4**：用 `components/ui/Tooltip`；第一行結論、第二行依據（`chinese_subtitle_sources`：片內字幕／旁邊的 zh-TW 檔／Vido 生成，用「・」連接）或下一步；滑鼠移過立刻出現、Tab 到出現、Esc 收起、手機點一下出現、點別處收起；點圖示不會打開「管理字幕」；螢幕閱讀器讀到同一句話。拿掉 `title`。`Tooltip` 不支援觸控就在元件裡補，不在 `EpisodeList` 自己寫一套。
3. **前端 spec 用《末日光明》的形狀**（同 `-a` AC #7）。
4. 拿掉或改名 `data-testid` 前先 grep `tests/e2e`；E2E 至少一支：季清單圖示＋鍵盤 Tab 出提示。
5. `pnpm nx test web`、`pnpm run lint:all` 全綠；視覺基準照 /ship 的 `-linux` 流程。

## Tasks / Subtasks

- [x] T1 `libraryService` 型別加兩個欄位；`EpisodeList` 圖示改看 `chineseSubtitle`
- [x] T2 `Tooltip` 觸控行為＋提示框文字
- [x] T3 spec＋E2E＋視覺基準

## Dev Agent Record

- 2026-10-07 /ship：PR #719；`-linux` focus 基準由 CI bootstrap PR #720 補進分支。CI 第一輪 Lint 紅：React Compiler 規則不准把讀 ref 的函式塞進 `cloneElement`（本機 `lint:all` 我只看了「可自動修」那行的 0 errors，漏看了總數）→ 改成把處理函式直接交給 Base UI 的 Trigger（它會跟子元素自己的處理函式串起來），不再 cloneElement。E2E ×2、Tooltip 單元測試照舊綠。

### File List

- apps/web/src/components/ui/Tooltip.tsx、Tooltip.spec.tsx（新）
- apps/web/src/components/media/EpisodeList.tsx、EpisodeList.spec.tsx
- apps/web/src/types/library.ts
- tests/e2e/episode-subtitle-badge.spec.ts（新）
- apps/api/internal/services/series_season.go、episode_chinese_subtitle.go、series_season_subtitle_test.go


### Completion Notes List

- 2026-10-07 Bob（SM）：從傘狀單拆出。
- 2026-10-07 Amelia（dev，Opus 5.5）：T1–T3 完成。
  - 後端：`MergedEpisode.embedded_subtitles_read`（`*bool`，有本地檔才有）。
  - `Tooltip` 加 `delay`、`openOnPress`：觸控點一下打開；Base UI 的懸停邏輯會在點完時送一個「滑鼠離開」把它關掉，所以觸控打開的提示框忽略 `trigger-hover` 關閉，等點別處／Esc／失焦才關。滑鼠點擊交給懸停處理；鍵盤 Enter（`detail === 0`）不算觸控。既有的側欄等呼叫者不受影響。
  - `EpisodeList`：有 `chineseSubtitle` 就照 J11-D 顯示（繁中／中文繁簡未知＝綠勾；只有簡中＝紅驚嘆；缺中文＝紅叉；還沒檢查＝淡色橫線）；生成中的四種狀態優先；舊後端沒送這欄就照舊看 `subtitle_status`。圖示改成 24px 的按鈕（`-m-1 p-1`，版面不動），拿掉 `title`，aria-label＝提示框兩行合成一句。提示框放右邊（放上面會蓋住上一集的片名）。
  - 實機截圖對照 J11-D：桌機懸停、手機點一下都跟稿一致。
- 2026-10-07 對抗式 CR（Sonnet 5.5）：0H／3M／4L。修 M1（鍵盤 Enter 被當成觸控）、M3（E2E 懸停等待 500→1500ms）、L4（不認得的結論改退回舊邏輯）、L5（加 `Tooltip.spec.tsx`，突變測試確認拿掉觸控保護會紅）。不修：M2（打開時螢幕閱讀器會念兩次同一句——J11-D 規則 3 本來就要求同一句）、L6、L7。
  - 閘門：web 單元 300 檔全綠、E2E chromium／firefox／mobile-chrome ×3 全綠（本機沒裝 WebKit，交給 CI）、`lint:all` 0 errors、prettier 綠、Go services 測試綠。
  - **視覺基準有一張真的變了**：`media-episode-list-subtitle-entry/focus`——Tab 現在先停在字幕圖示（新的可聚焦按鈕）並打開提示框，不再直接停在「管理字幕」。這是 AC 要的行為。⚖️ Alexyu 2026-10-07 選 A：更新 darwin 基準、刪掉舊的 linux 基準，由 CI bootstrap 補。
