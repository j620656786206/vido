# Story: 季清單每一集標出「有沒有中文字幕」，圖示都有提示

Status: ready-for-dev

**Source:** `disc-2026-10-episode-list-subtitle-badge`（Alexyu 2026-10-05 在 NAS 上看《末日光明》時提出）。
**⚖️ 裁定（Alexyu 2026-10-05，party mode）：**
1. **片內字幕也要算**（接受「掃描媒體庫會慢一點」的代價）。
2. **只放圖示可以，但每個圖示都要有提示**：滑鼠移過去、鍵盤切到、手機點一下都會出現。
3. **開工時機：Alexyu 看完 S01E02（Vido 生成的字幕）之後。** 單子與設計稿先做好。
**設計稿：** `ux-design.pen` `J11-D`（`w2Opax`）→ `_bmad-output/screenshots/flow-j-specs/j11-d.png`。

## Story

身為 Vido 的使用者，
我希望打開一部影集時，季清單的每一集旁邊就看得出這集有繁中、只有簡中、缺中文、還沒檢查、或正在生成，滑鼠移過去（或點一下）會告訴我原因，
這樣我不用一集一集打開「管理字幕」，就知道要先處理哪幾集。

## 查到的事（2026-10-05，Bob 逐一開檔確認）

- **現在每集已有一個字幕狀態圖示**，但只反映 Vido 自己的紀錄（`subtitle_status`），不看檔案：`apps/web/src/components/media/EpisodeList.tsx:147-165`（`SubtitleStatusIcon`），十種狀態與圖示語法在 `:62-95`（J2-D：實心圈＝有結論、無圈＝還沒結論；顏色＝要不要處理）。《末日光明》S01E02–E08 旁邊明明有 `.zh-TW.srt`，全部顯示「—」（`not_searched`）。
- **提示只靠瀏覽器內建 `title`**（`EpisodeList.tsx:158-160`）：要停約一秒才出現、手機點了不會出現——Alexyu 看不出「—」是什麼意思。
- **App 已有提示元件** `apps/web/src/components/ui/Tooltip.tsx:1-40`（Base UI，目前給收合側欄用；觸發元素必須是可互動的元素，預設出現在右側）。要確認它在觸控裝置「點一下出現」的行為；不支援就在這個元件裡補，不要在 EpisodeList 自己寫一套。
- **讀「旁邊的字幕檔」已有現成的便宜做法**（PR #674）：`apps/api/internal/subtitle/inventory.go:127`（`ListSidecars`，一次讀資料夾、中文看內容前 100KB、不跑 ffprobe）；完整版 `BuildInventory`（`:87`）含 ffprobe，只給單集對話框用。
- **單集沒有存片內字幕資訊**：`apps/api/internal/models/episode.go:23-39` 只有 `duration_seconds` 與 `subtitle_status／path／language`，沒有 `subtitle_tracks`。單集的 ffprobe 目前只在估價／候選分析時「懶惰地」量時長（`apps/api/internal/services/generation_candidates.go:1287` `ProbeWithDuration`、`:1322` 寫回時長）。
- **季標頭的「劇集數未知」與影集標頭的「缺字幕」是另外兩個問題**，各自立單（見 Discovery Triage），本單不處理。

## Acceptance Criteria

1. **每集一個結論（後端算）。** 季的分集清單 API 對每個有本地檔的集數回一個字幕摘要：
   - `zh_hant`：片內字幕、旁邊的字幕檔、Vido 生成，任一個是繁中。
   - `zh_hans_only`：沒有繁中，但有簡中（片內或旁邊）。
   - `no_chinese`：讀過了，只有其他語言（或沒有字幕）。
   - `unknown`：片內字幕還沒讀過，且旁邊沒有中文檔。
   - 生成中照舊用既有的進行中狀態（`EpisodeList.tsx` 的四個 spinner 狀態）。
   - 摘要同時帶「依據」：哪幾個來源成立（片內／旁邊的檔名／Vido 生成），給提示框第二行用。
2. **不逐集開檔。** 旁邊的字幕檔：一季讀一次資料夾（同資料夾的集數共用一次讀取）。片內字幕：讀**資料庫裡存好的**結果，不在打開清單時跑 ffprobe。
3. **片內字幕存起來。** 單集新增「片內字幕摘要」欄位（migration），在下列時機寫入：掃描媒體庫時（新檔與檔案大小／修改時間有變的檔）；以及一次性的背景補齊，把現有單集讀一遍（要能中斷、續跑，不擋 API 啟動，進度寫 log）。估價／候選分析已經跑的 ffprobe 也順手寫入，不要重複探測。
4. **畫面照 J11-D。** 圖示沿用 J2 語法：有繁中 `circle-check`＋成功色；只有簡中 `circle-alert`＋錯誤色；缺中文 `circle-x`＋錯誤色；還沒檢查 `minus`＋淡色；生成中照舊。沒有本地檔的集數不顯示圖示（照舊）。
5. **提示框照 J11-D 規則 3–4。** 用 `components/ui/Tooltip`；第一行結論、第二行依據或下一步；滑鼠移過立刻出現、鍵盤 Tab 到出現、Esc 收起、手機點一下出現、點別處收起；點圖示不會打開「管理字幕」；螢幕閱讀器讀到同一句話。拿掉 `title` 屬性。
6. **驗收用《末日光明》的真實形狀寫成測試**（後端 fixture＋前端 spec）：S01E01＝缺中文（只有英文片內字幕）；S01E02＝有繁中（旁邊 zh-TW 檔＋Vido 生成）；S01E03–E08＝有繁中（旁邊 zh-TW 檔）；S02 全部＝有繁中（片內中文）——其中 S02 的片內 `chi` 軌若分不出繁簡，照 PR #674 的規則判斷，並寫清楚結論。
7. 拿掉或改名 `data-testid` 前先 grep `tests/e2e`；E2E 至少一支：季清單圖示＋鍵盤 Tab 出提示。
8. `go build／vet／test ./...`、`staticcheck-2026.1`、`pnpm nx test web`、`pnpm run lint:all` 全綠；視覺基準照 /ship 的 `-linux` 流程。

## Tasks / Subtasks

- [ ] T1 後端：單集片內字幕摘要欄位＋migration＋掃描時寫入＋背景補齊（AC #3）
- [ ] T2 後端：分集清單 API 回字幕摘要（一季一次讀資料夾、讀存好的片內結果）（AC #1、#2、#6）
- [ ] T3 後端測試（AC #6、#8）
- [ ] T4 前端：`EpisodeList` 圖示改用摘要＋`Tooltip`；必要時補 `Tooltip` 的觸控行為（AC #4、#5）
- [ ] T5 前端測試＋E2E（AC #6、#7）
- [ ] T6 視覺基準與檢查（AC #8）

後端 3 項、前端 3 項 → 不需要拆單。

## Dev Notes

### 不要做的事

- 不要在打開季清單時跑 ffprobe（AC #2）。
- 不要改影集標頭的「缺字幕」、季標頭的「劇集數未知」——各自有單。
- 不要動「管理字幕」對話框（PR #674 已完成）。

### 已知陷阱

- 補齊既有單集時，NAS 的影片在 Unraid FUSE 路徑上，ffprobe 對 4K 檔要數秒；補齊要限流（沿用 FFprobeService 的併發上限），並能在重啟後從上次的位置續跑。
- 中文繁簡一律看內容（project-context.md「Language detection MUST analyze subtitle file content」）。

### 下游依賴（2026-10-06 Bob 加註）

- `disc-2026-10-subtitle-filter-disagrees-with-badges`（媒體庫「有／缺中文字幕」篩選）的影集部分＝`disc-2026-10-subtitle-filter-series-phase-2`，要等本單的每集摘要。**那邊的需求：** 媒體庫篩選要在 SQL 裡跨所有影集篩，所以每集的結論（含「旁邊的字幕檔」那一半）必須**存進資料庫**，不能只在打開季清單時現讀（本單 AC #2 的做法）。設計本單的資料欄位時請一起考慮，例如掃描時把 `ListSidecars` 的中文結論也存起來。
- 每集的「有沒有中文」請用同一個判斷函式 `models.ChineseSubtitleVerdict`（由上面那張單新增；若本單先做，就由本單照那張 story「判斷規則」建立它），不要另寫一份。

### Time-dependent visual coverage

N/A — 不碰讀時鐘的元件。

### References

- `ux-design.pen` `J11-D`（`w2Opax`）、`J2-D`（圖示語法）
- PR #674（`subtitle.ListSidecars`／`BuildInventory`）
- `.claude/memory/feedback_new_feature_needs_party_mode_first.md`（本單的需求是 party mode 討論後才開的）

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- 2026-10-05 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created
- 2026-10-05 Sally（UX）：J11-D 規格畫面已畫（狀態六列＋提示框範例＋六條規則），截圖 `flow-j-specs/j11-d.png`。

### Discovery Triage

- **③** 季標頭寫「劇集數未知」但清單列了 8 集 → `disc-2026-10-season-episode-count-unknown`
- **③** 影集標頭「缺字幕」只看影集那一列的狀態，不看各集 → `disc-2026-10-series-header-subtitle-badge-stale`
- **③** 全站「只有圖示、沒文字」的地方還有哪些只靠 `title` → `disc-2026-10-icon-only-native-title-audit`

### File List
