# Story（前端）：季清單每一集的字幕圖示改看真實結論，圖示都有提示框

Status: backlog

**Source:** 傘狀單 `disc-2026-10-episode-list-subtitle-badge`。**依賴 `-a`**（季清單 API 的 `chinese_subtitle`／`chinese_subtitle_sources`，AC #1 `[@contract-v1]`）。
**設計稿：** `J11-D`（`w2Opax`）規則 3–6、`J2-D`（圖示語法）。

> ⚖️ SM 裁定（Bob 2026-10-07）：從傘狀單拆出，切分線與理由見 `-a`。

## 開工前要先問 Sally 的事

- `-a` 的結論有五個值，J11-D 只畫了四種有結論的樣子＋生成中。多出來的 **`zh`（有中文、分不出繁簡，例：片內 `chi` 軌沒有標題）** 要用什麼圖示、提示框寫什麼，要 Sally 定（或補進 J11-D）。

- **CR L1（-a）**：片內字幕還沒讀過、但 Vido 紀錄是「線上找不到」（`not_found`）的集數，結論會是 `none`（電影同規則），背景工作讀完片內字幕後才可能變成「有」。提示框文案要不要對這種情況講清楚，或要不要改規則（需 Alexyu 裁定），開工時一起問。

## Acceptance Criteria（沿用傘狀單 AC #4、#5、#7、#8，結論改用 `-a` 的五值）

1. **圖示照 J11-D**：`zh_hant` → `circle-check`＋成功色；`zh_hans` → `circle-alert`＋錯誤色；`none` → `circle-x`＋錯誤色；`unknown` → `minus`＋淡色；`zh` → 依 Sally 裁定；生成中照舊（四個 spinner 狀態優先）。沒有本地檔的集數不顯示圖示。
2. **提示框照 J11-D 規則 3–4**：用 `components/ui/Tooltip`；第一行結論、第二行依據（`chinese_subtitle_sources`：片內字幕／旁邊的 zh-TW 檔／Vido 生成，用「・」連接）或下一步；滑鼠移過立刻出現、Tab 到出現、Esc 收起、手機點一下出現、點別處收起；點圖示不會打開「管理字幕」；螢幕閱讀器讀到同一句話。拿掉 `title`。`Tooltip` 不支援觸控就在元件裡補，不在 `EpisodeList` 自己寫一套。
3. **前端 spec 用《末日光明》的形狀**（同 `-a` AC #7）。
4. 拿掉或改名 `data-testid` 前先 grep `tests/e2e`；E2E 至少一支：季清單圖示＋鍵盤 Tab 出提示。
5. `pnpm nx test web`、`pnpm run lint:all` 全綠；視覺基準照 /ship 的 `-linux` 流程。

## Tasks / Subtasks

- [ ] T1 `libraryService` 型別加兩個欄位；`EpisodeList` 圖示改看 `chineseSubtitle`
- [ ] T2 `Tooltip` 觸控行為＋提示框文字
- [ ] T3 spec＋E2E＋視覺基準

## Dev Agent Record

### Completion Notes List

- 2026-10-07 Bob（SM）：從傘狀單拆出。
