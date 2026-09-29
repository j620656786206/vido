# Story: disc-2026-09-instant-search-tmdb-outage-silent — TMDb 斷線時，頂欄搜尋說實話

Status: done

<!-- SM Bob create-story 2026-09-29。Alexyu 從推薦清單選 A。
     這張需要新畫面（下拉裡多一列），依 .pen 協作模式（memory: feedback_pen_inline_agent_workflow）：
     Sally 出節點錨定的提示詞 → Alexyu 貼給 Pencil 的 Inline AI Agent 執行並存檔、匯出、commit → Sally 用 MCP 唯讀 review → Dev 實作。
     行號為 main `6bfedd5c`；.pen 節點 id 由 SM 本次用 Pencil MCP 唯讀讀取。 -->

## Story

身為在頂欄搜尋框打片名的人，
當 TMDb 斷線或沒設金鑰時，
我要看到「TMDb 暫時無法連線，只顯示媒體庫結果」，而不是「找不到「X」的結果」——後者讓我以為這部片不存在或我打錯字。

## 背景（查到的事）

- 後端 `apps/api/internal/services/search_service.go:98-104` 的註解與 `:105-200` 的實作：5 個 TMDb 呼叫（中英電影、中英影集、人物）＋媒體庫並行；**任一失敗就當空清單**，只有全部（含媒體庫）都失敗才回錯誤（`:167-170`）。這是刻意的：TMDb 壞了，媒體庫結果照常（testsprite TC092，`cmd/api/main.go:922` 註解）。**不可**改成回錯誤。
- 回應型別 `UnifiedSearchResult`（`search_service.go:49-57`）沒有任何「TMDb 失敗」的訊號；前端型別 `apps/web/src/types/tmdb.ts:64-72` 同形。
- 前端 `apps/web/src/components/search/SearchSuggestions.tsx`：
  - `:117-125` 整個請求失敗 →「搜尋暫時無法使用」（dsr-8 AC #5）；`:61-66` 註解明寫 TMDb 單獨斷線走不到這裡。
  - `:127-134` 沒結果 →「找不到「{query}」的結果」← TMDb 斷線且媒體庫沒結果時的**誤導**文案。
  - `:136-225` 有結果時列出 媒體庫／電影／影集／人物，最後「按 Enter 查看所有結果 →」。
  - 唯一使用者：`InstantSearchBar.tsx:188`（桌面浮動、手機全螢幕共用，`floating` 切換）。
- 設計稿 `I3-D-v2 · 即時搜尋建議`（`m0Zew`，`flow-i-discover-v2/i3-d.png`）只有正常狀態；**沒有 TMDb 斷線的稿**。既有的「部分失敗」視覺語言是 `I8-D-v2 · 區段 fail-soft`（`KdnVw`）的 `error-panel`（`R8t8Uy`：`$error-tint` 底、`triangle-alert` 圖示、`$error-text` 字）。

## Acceptance Criteria

1. **後端 `[@contract-v1]`**：`UnifiedSearchResult` 新增 `tmdb_unavailable`（bool，JSON `tmdb_unavailable`）。**5 個 TMDb 呼叫全部失敗**時為 `true`，其餘（全成功或部分失敗）為 `false`。媒體庫仍成功時照常回 200；全部失敗時照舊回錯誤。
2. **前端**：`UnifiedSearchResult` 型別加 `tmdbUnavailable?: boolean`。`tmdbUnavailable` 為真時：
   - 下拉**最上方**顯示一列提示（依 Task 0 設計稿）：「TMDb 暫時無法連線，只顯示媒體庫結果」，`role="status"`，`data-testid="search-suggestions-tmdb-down"`。
   - 媒體庫有結果 → 提示列＋媒體庫區段＋「按 Enter 查看所有結果 →」。
   - 媒體庫沒結果 → 提示列＋「媒體庫裡沒有「{query}」」（取代「找不到「{query}」的結果」）。
   - `tmdbUnavailable` 為假或沒有 → 行為完全不變。
3. 桌面浮動與手機全螢幕都適用（同一元件）。
4. 測試：Go — TMDb 全掛＋媒體庫有／無結果 → `TmdbUnavailable=true`；TMDb 部分失敗 → false；全成功 → false。前端 — 兩種 tmdb-down 狀態的文案與 testid、`tmdbUnavailable` 為假時舊行為不變。先紅後綠。
5. 視覺：新狀態有畫廊夾具（`routes/test/-gallery.fixtures.tsx`）與 `-darwin` 基準；`-linux` 由 CI bootstrap。與 Task 0 設計稿逐項比對。

## Tasks / Subtasks

- [x] **Task 0（設計，Alexyu 執行）**— 把下方「Sally 的 Pencil 提示詞」貼給 Pencil 的 Inline AI Agent；⌘S 存檔；跑 `python3 scripts/export-pen-screenshots.py`；Dev 會補 `SCREENS` 兩筆（`i9-d`、`i10-d`）。完成後 Sally 用 MCP 唯讀 review。
- [x] Task 1 — 後端 `TmdbUnavailable` 與測試（AC #1, #4）
- [x] Task 2 — 前端型別＋`SearchSuggestions` 兩種狀態＋測試（AC #2, #3, #4）
- [x] Task 3 — 畫廊夾具＋`-darwin` 基準；`pnpm nx test api`、`pnpm nx test web`、`lint:all` 全綠（AC #5）

## Sally 的 Pencil 提示詞（Task 0，整段貼給 Inline AI Agent）

```
在 ux-design.pen 裡新增兩張桌面稿，示範「頂欄即時搜尋在 TMDb 斷線時」的下拉。兩張都從現有的
「I3-D-v2 · 即時搜尋建議」（節點 m0Zew）複製而來，放在同一個群組「I · 桌面 Desktop」（vWdb4）裡，
接在「I8-D-v2 · 區段 fail-soft」（KdnVw）右邊：第一張 x = KdnVw 的 x + 1540，第二張 x = KdnVw 的 x + 3080，
y 都跟 KdnVw 一樣。不要動 m0Zew 本身。

【第一張】名稱：I9-D-v2 · 即時搜尋・TMDb 斷線（桌面）
1. 在複本的下拉「suggestion-popover」（原節點 LCHfC 的複本）裡，刪除這些子節點的複本：
   divider（ObRa5）、section-hdr-電影（d3K6A0）、tmdb-你的名字。（o5dmG）、tmdb-你的鳥兒會唱歌（teDYf）、
   section-hdr-影集（rekhI）、tmdb-你的情歌（aLrvg）、section-hdr-人物（qOhQL）、person-新海誠（dn71t）。
   保留：section-hdr-媒體庫、result-你的名字、result-你的婚禮、footer（按 Enter 查看所有結果 →）。
2. 在下拉的**第一個子節點位置**（section-hdr-媒體庫 之前）插入一列提示，名稱 tmdb-down-notice：
   - frame，水平排列，width = fill_container，alignItems = center，gap = $Space/sm，
     padding = [$Space/sm, $Space/lg]，fill = $error-tint；底部加 1px 分隔線 $border-subtle（stroke 只畫下邊即可，
     做不到就在它下面插一個名稱 divider、高 1、fill $border-subtle、width fill_container 的 rectangle）。
   - 子節點 1：icon，名稱 icon，icon = triangle-alert，16×16，fill = $error-text。
   - 子節點 2：text，名稱 msg，內容「TMDb 暫時無法連線，只顯示媒體庫結果」，
     fontFamily = $Type/Family/Primary，fontSize = $Type/Body/Size，lineHeight = $Type/Body/Line，
     fontWeight = 500，fill = $error-text，width = fill_container，textGrowth = fixed-width。
3. 下拉高度改成隨內容（fit_content）；確認沒有任何子節點被裁切。

【第二張】名稱：I10-D-v2 · 即時搜尋・TMDb 斷線＋媒體庫無結果（桌面）
1. 從第一張複製，放在上面說的位置。
2. 把頂部搜尋框（search-input，原節點 Fgp2j 的複本）裡的查詢文字改成「星際效應」。
3. 在下拉裡刪除 section-hdr-媒體庫、result-你的名字、result-你的婚禮、footer 的複本，只留 tmdb-down-notice（和它的 divider）。
4. 在 tmdb-down-notice 下面插入一個 text，名稱 empty-msg，內容「媒體庫裡沒有「星際效應」」，
   fontFamily = $Type/Family/Primary，fontSize = $Type/Body/Size，lineHeight = $Type/Body/Line，
   fill = $text-muted，textAlign = center，width = fill_container，textGrowth = fixed-width；
   用一個 frame 包起來（名稱 empty-row，width fill_container，padding = [$Space/lg, $Space/lg]）。
5. 下拉高度 fit_content；確認沒有裁切。

注意：.pen 的對齊值只有 start/center/end；顏色只用上面寫的變數；圖示名是 triangle-alert。
做完後存檔（File ▸ Save）。
```

## Dev Notes

- 「TMDb 全掛」的判準：`zhMovieErr, enMovieErr, zhTVErr, enTVErr, peopleErr` 五個都 `!= nil`（`search_service.go:154` 那組）。部分失敗不亮提示（部分結果仍然是 TMDb 的真結果）。
- 沒設金鑰時 TMDb client 會回錯誤（401），五個都失敗 → `true`，正確。
- 不要動 `isError` 分支（整個請求失敗仍是「搜尋暫時無法使用」）。
- `/search` 全頁結果不走這個端點（`git grep useInstantSearch` 只有 `InstantSearchBar.tsx:53`），不在本張。

### Project Structure Notes

- 後端：`internal/services/search_service.go`＋測試。前端：`types/tmdb.ts`、`components/search/SearchSuggestions.tsx`＋spec、畫廊夾具。設計：`ux-design.pen`＋兩張截圖＋`scripts/export-pen-screenshots.py` 的 `SCREENS`。後端 1、前端 2 → 單張。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 原條目：`sprint-status.yaml` `disc-2026-09-instant-search-tmdb-outage-silent`（dsr-8 立案）
- 同類：`disc-2026-09-manual-search-hides-source-errors`（#590）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia；Sally review 同一 session）

### Debug Log References

- RED：Go `TestSearch_TmdbUnavailableFlag` 先加欄位不設值 → 2 條「全掛」紅、2 條「部分／正常」綠；前端 `SearchSuggestions — TMDb unavailable` 4 條中 2 條紅（另 2 條守舊行為）。
- GREEN：`internal/services` 全綠；`components/search/` 154 條綠。

### Completion Notes List

- **Task 0（設計）**：Alexyu 以 Inline AI Agent 執行提示詞並 commit `2f95d542`（`ux-design.pen`、`i9-d.png`、`i10-d.png`、`SCREENS` 兩筆 `wnolW`／`PcNux`、`pen-tokens.json`）。`git hash-object ux-design.pen` 與 HEAD 相同（已落盤）；`SCREENS` 無重複 key。
- **Sally MCP 唯讀 review**：兩張位置 x=28440／29980、y 同 `KdnVw`；`tmdb-down-notice` fill `$error-tint`、padding [$Space/sm=8, $Space/lg=16]、gap 8、底線 1px `$border-subtle`、`triangle-alert` 16 `$error-text`、文案逐字「TMDb 暫時無法連線，只顯示媒體庫結果」500；I10 `empty-msg` 逐字「媒體庫裡沒有「星際效應」」`$text-muted`、`empty-row` padding 16。唯一 problem 是繼承自 `m0Zew` 的 `poster-6 fully clipped`（原稿就有，非本次）。→ 追認。
- 後端：`UnifiedSearchResult.TmdbUnavailable`（`tmdb_unavailable`），5 個 TMDb 呼叫全錯才為真。
- 前端：`tmdbDown = !isLoading && !isError && result?.tmdbUnavailable`；提示列放第一個子節點（`role="status"`、`search-suggestions-tmdb-down`）；空狀態改「媒體庫裡沒有「X」」且依 I10 padding 16（`py-4`，一般空狀態維持 `py-6`）。
- 畫廊：`search-suggestions-tmdb-down`、`search-suggestions-tmdb-down-empty`（`TmdbDownSuggestions` 包一個 0 高的 relative 錨點讓浮動下拉有地方掛）；`-darwin` 基準已生成（本機後端因 AI 設定起不來，改以 `CI=1` 只起前端跑 `update-snapshots=missing`；另一張 `parse-floating-parse-progress-card` 的本機差異與本張無關、未更新）。`-linux` 由 CI bootstrap。
- 🎨 UX Verification: PASS — 提示列（底色、圖示、字級字重、內距、底線）與 I9／I10 一致；媒體庫列與底部「按 Enter」沿用既有實作（與 I3 的差異為既有，非本張）。
- 全量回歸：`pnpm nx test web` 291 檔／4,359 條綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- /ship 對抗式 CR：0 HIGH／1 MED／6 LOW＋2 NIT。確認後端判準、無後端快取、401 會亮旗標、snake→camel 轉換、其他消費者不受影響。已修：MED-1（「TMDb 斷線」的回應會被快取 5 分鐘，TMDb 恢復後打同一個字仍顯示斷線 → `useInstantSearch` 的 `staleTime` 對 `tmdbUnavailable` 回應設 0，hook 測試先紅後綠）、LOW-2（「載入中不顯示」測試模擬的是不會發生的情況 → 改成「狀態區域先存在、再填字」測試）、LOW-3（`role="status"` 帶著文字才掛上，VoiceOver 可能不念 → 狀態區域永遠掛著、只換內容）、LOW-4（行高：設計 1.625 → 加 `leading-relaxed`，重出 `-darwin` 基準）、LOW-5（夾具高度加到 300／150 留餘裕）、LOW-6 的測試面（加一條 `floating=false` 手機版測試；設計稿缺口立案）、NIT（Go 驗 JSON 鍵名 `tmdb_unavailable`；前端驗 API 回應轉成 `tmdbUnavailable`）。
- 🔗 AC Drift: FOUND — dsr-8 AC #5（TMDb 單獨斷線走不到錯誤分支，`SearchSuggestions.tsx:61-66` 註解）→ 本張補上 TMDb-down 狀態；`isError` 分支不變。
- 📎 Contract Stamps: FOUND（本張 AC #1 `[@contract-v1]` 定義 `tmdb_unavailable`；新增欄位，無上游 stamp 需 ack）。
- 🎭 A11y Pre-Flight: PASS（1 component checked；提示列 `role="status"`、圖示 `aria-hidden`；0 jsx-a11y warnings introduced）。

### Discovery Triage

- ③ backlog-with-carry-forward-link — CR LOW-6：手機全螢幕搜尋沒有對應設計稿，提示列會隨結果捲走 → `disc-2026-09-instant-search-tmdb-down-mobile-design`（P4）。
- 不立案：commit `2f95d542` 訊息把提示寫成「橘色」，實際是 `$error-tint`／`$error-text`（紅）——只是文字，PR 說明已更正。

### File List

- ux-design.pen（Alexyu，`2f95d542`）
- _bmad-output/screenshots/flow-i-discover-v2/i9-d.png、i10-d.png（Alexyu，`2f95d542`）
- scripts/export-pen-screenshots.py（Alexyu，`2f95d542`）
- _bmad-output/pen-tokens.json（Alexyu，`2f95d542`）
- apps/api/internal/services/search_service.go
- apps/api/internal/services/search_service_test.go
- apps/web/src/types/tmdb.ts
- apps/web/src/components/search/SearchSuggestions.tsx
- apps/web/src/components/search/SearchSuggestions.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/search-suggestions-tmdb-down/default-visual-darwin.png
- tests/visual/components.visual.spec.ts-snapshots/components/search-suggestions-tmdb-down-empty/default-visual-darwin.png
- _bmad-output/implementation-artifacts/disc-2026-09-instant-search-tmdb-outage-silent.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

| Date       | Change                                                        |
| ---------- | ------------------------------------------------------------- |
| 2026-09-29 | create-story（SM Bob）＋ Sally 的 Pencil 提示詞；等 Alexyu 執行 Task 0。 |
| 2026-09-29 | Alexyu 執行 Task 0（`2f95d542`）；Sally MCP review 追認；dev-story（Amelia）完成 Task 1–3 → review。 |
| 2026-09-29 | /ship 對抗式 CR：修 1 中 6 低；重出 `-darwin` 基準；立案手機設計稿缺口。 |
