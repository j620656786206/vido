# Story: 大陸片字幕一律轉成繁體字，但保留原用語（像 Netflix／Disney+ 的陸劇）

Status: review

**Source:** `backlog-mainland-rule-three-predicates`（sub-7-4 CR，2026-09-07；判斷式收斂已由 PR #669 做完）。
**⚖️ 裁定：** Alexyu 2026-10-05 選 B——「像 Disney+ 或者是 Netflix 的陸劇，它們的應該也是繁體中文，但是是保留原意，只是做字的轉換，不轉換成台灣用語。」**取代** `_bmad-output/planning-artifacts/sprint-change-proposal-2026-03-24.md` 的「大陸片保留簡體不轉換」。

## Story

身為 Vido 的使用者，
我希望大陸片的字幕不管從哪來，都是繁體字、但保留大陸的說法（「質量」不改成「品質」），
這樣讀起來是我習慣的字，又不會改掉台詞原本的意思。

## 查到的事（2026-10-05，Bob 逐一開檔確認）

**大陸片今天三條路三種結果（「这个软件的质量很好」）：**

| 路 | 位置 | 今天 | 要改成 |
|---|---|---|---|
| 線上自動找（下載完成觸發＋批次） | `apps/api/internal/subtitle/engine.go:157-165`（`deriveConversionPolicy` → `ConvertNever`）、`engine.go:340-343`（`ConvertNever` 原樣交付） | 简体原樣 | 這個軟體的質量很好 |
| 片內簡體軌 | `apps/api/internal/subtitle/process_item.go:374-383`（不看國家，一律轉字） | 這個軟體的質量很好 | 不變 |
| AI 翻譯 | `pipeline.go` `convertAndStitch`、`transcription_service.go` `translateSRT` | 繁體＋大陸用語 | 不變 |
| 手動下載（管理字幕對話框） | `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:440`（`convertToTraditional: !isCNContent`） | 大陸片不轉 | 一律轉 |

- 詞彙步驟對大陸片已經全部跳過：`zhtw.Finalize` 在 `IsMainland` 時不套詞庫（`apps/api/internal/zhtw/finalize.go:41-53`，PR #669）。AI 術語校正（`engine.go:221-224`）是把大陸用語改台灣用語（`apps/api/internal/ai/prompts/terminology_corrector.go:8,37`），屬於詞彙步驟，**大陸片照樣跳過**。
- `ConversionPolicy`（`ConvertAuto`／`ConvertAlways`／`ConvertNever`）與 `ProcessOptions.ConversionOverride`：全 repo grep `ConvertNever|ConversionOverride|ConvertAlways`，產品程式碼只有 `engine.go` 自己；`ConversionOverride` **沒有任何呼叫端設定**，`ConvertAlways` 只在 `convertIfNeeded` 的繁體分支出現且行為與 `ConvertAuto` 相同（`engine.go:346-350`）。拿掉 `ConvertNever` 後整個 enum 沒有作用。
- 前端「陸劇保留簡體字幕（對白一致）」政策行：`ManageSubtitleDialogV2.tsx:402-411`；測試 `ManageSubtitleDialogV2.spec.tsx:447-463`。全 repo grep `cn-policy-note|陸劇`：`tests/`、`routes/test` 沒有其他斷言。
- `SubtitleSearchDialog`（轉繁開關，大陸片預設關）：`SubtitleSearchDialog.tsx:61-72`；只掛在 `MediaDetailPanel.tsx:301`，而 `MediaDetailPanel` 已無產品掛載點（`disc-2026-09-unmounted-v1-components`），只活在 `routes/test/-gallery.fixtures.tsx:5006`（productionCountry 'US'）。測試 `SubtitleSearchDialog.spec.tsx:97-107`。
- 兩個對話框的 `productionCountry` prop 只拿來算 `isCNContent`（grep 確認）；`LocalDetailV2.tsx:82-86,455` 與 `MediaDetailPanel.tsx:104,306` 只是為了傳這個 prop 才組字串。
- 後端手動下載：`subtitle_handler.go` `shouldConvert`（開關沒送時預設轉）；註解寫「前端依 production_countries 預設 OFF for CN」——要改。
- 設計稿：`ux-design.pen` 的 `v16pVI`（`note-f1-cn-variant`，在 `JzmvC`「F · 規格 Spec」裡）寫「簡中軌的轉為繁中鈕換成政策行（政策=保留簡體，可覆寫）」＋示範列 `YE3SJ`（「陸劇保留簡體字幕（對白一致）」／「仍要轉換」）。`JzmvC` 不在 `scripts/export-pen-screenshots.py` 的 SCREENS 內（grep 確認），**沒有截圖要重產**。

## Acceptance Criteria

1. **線上自動找：** 大陸片的簡體字幕轉成繁體字（OpenCC），不套台灣詞庫、不跑 AI 術語校正。非大陸片行為不變。已是繁體的不動。
2. **拿掉 `ConversionPolicy`：** 刪 `ConvertAuto`／`ConvertAlways`／`ConvertNever`、`ConversionOverride`、`deriveConversionPolicy`；`convertIfNeeded` 只依偵測結果決定。AI 術語校正的大陸片跳過改用 `zhtw.IsMainland(productionCountries(opts))`。
3. **管理字幕對話框：** 手動下載一律送 `convertToTraditional: true`；拿掉「陸劇保留簡體字幕（對白一致）」政策行（大陸片的簡中軌跟一般片一樣顯示）；拿掉 `productionCountry` prop 與 `LocalDetailV2` 傳它的那段。
4. **舊搜尋對話框（只剩夾具）：** 轉繁開關一律預設開；拿掉 `productionCountry` prop 與 `MediaDetailPanel` 傳它的那段。開關保留（使用者仍可關）。
5. **後端手動下載註解**改成新規則；`shouldConvert` 行為不變。
6. **設計稿：** `v16pVI` 改寫成新規則（大陸片簡中軌跟一般片一樣；轉出來只轉字、保留原用語；註明 2026-10-05 裁定取代舊規則），刪示範列 `YE3SJ`。
7. **規劃文件：** `sprint-change-proposal-2026-03-24.md` 開頭加一行「⚖️ 2026-10-05 已被取代」指向本單。
8. **測試：** engine 大陸片 → 「這個軟體的質量很好」；AI 術語校正大陸片仍跳過；對話框下載送 `convertToTraditional: true`（大陸片也是）；政策行不再出現；搜尋對話框大陸片開關預設開。
10. **（lane ① 實作時吸收）批次找字幕的影集與單集也帶國家：** `RepoCollector.CollectSeriesNeedingSubtitles` 原本對影集一律傳空國家（`batch.go` 註解寫「Series model does not have production_countries」——sub-7-2b 之後已不成立），`CollectEpisodesBySeasonID` 對單集也傳空。結果陸劇用「批次找字幕」會被套台灣詞庫＋AI 術語校正。改成影集傳自己的國家、單集傳所屬影集的國家（每部劇查一次；查不到只 Warn、當非大陸片，不讓批次失敗）。`SeriesSubtitleFinder` 多 `FindByID`。
9. `go build／vet／test ./...`、`staticcheck-2026.1`、`pnpm nx test web`、`pnpm run lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 後端：engine 拿掉 `ConversionPolicy`，術語校正改用 `zhtw.IsMainland`；handler 註解；批次影集／單集帶國家（AC #1、#2、#5、#10）
- [x] T2 前端：`ManageSubtitleDialogV2`＋spec＋`LocalDetailV2`（＋spec stub）（AC #3）
- [x] T3 前端：`SubtitleSearchDialog`＋spec＋`MediaDetailPanel`（AC #4）
- [x] T4 設計稿 `v16pVI` 與規劃文件註記（AC #6、#7）
- [x] T5 測試與檢查（AC #8、#9）

後端 1 項、前端 2 項 → 不需要拆單。

## Dev Notes

### 不要做的事

- 不要動片內簡體軌與 AI 翻譯兩條路——它們已經是新規則。
- 不要把 HK／MO 加進大陸判斷（未裁定，另立 `disc-2026-10-hk-mo-mainland-rule`）。
- 不要拿掉手動下載 API 的 `convert_to_traditional` 欄位——API 契約不變，只改前端送的值。

### 已知陷阱

- 拿掉 `isCNContent` 後，`ManageSubtitleDialogV2` 的 `useCallback` 依賴陣列（`:459`）要一起拿掉；`Info` icon 若沒有其他使用處要從 import 移除（lint 會抓）。
- `LocalDetailV2.spec.tsx:114-123` 的對話框 stub 會渲染 `productionCountry`——stub 與相關斷言要一起改。
- 視覺夾具：gallery 的 SubtitleSearchDialog 夾具傳 'US'，開關本來就是開——預期不會有視覺差異；若有，照 /ship 的 `-linux` 基準線流程處理。
- 改 `.pen` 後要確認 `git status` 有 ` M ux-design.pen`（`.claude/memory/feedback_verify_pen_saved_before_commit.md`）。

### Time-dependent visual coverage

N/A — 不碰讀時鐘的元件。

### References

- `.claude/memory/project_cn_subtitle_policy.md`（裁定紀錄）
- PR #669（`zhtw.Finalize`／`IsMainland`）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

### Completion Notes List

- 2026-10-05 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: FOUND — see below。
  - 🔗 AC Drift: Story 8-8／8-9（sprint-change-proposal-2026-03-24）AC #9 — 「CN 內容保留簡體（`ConvertNever`、前端開關預設關、政策行）」→「CN 內容轉繁體字、保留原用語」。Alexyu 2026-10-05 裁定，規劃文件與 `project-context.md` 已加註取代。
  - 🔗 AC Drift: ux3-subtitle-v2（§9b 政策行 v16pVI）— 「簡中軌在陸劇顯示政策行」→「不顯示」；設計稿 v16pVI 已改寫。
- 📎 Contract Stamps: NONE (no [@contract-v*] stamps — API 欄位 `convert_to_traditional` 保留，只改前端送的值)
- 🎭 A11y Pre-Flight: PASS (3 components checked, lint 172 warnings 與改動前相同, 0 introduced by this story) — 只刪除元素與 prop，沒有新增互動元件。
- 🎨 UX Verification: PASS — 唯一的畫面差異是陸劇簡中軌下不再有政策行，與改寫後的 v16pVI 一致；`JzmvC` 不在截圖清單，無截圖需重產。視覺夾具（SubtitleSearchDialog 傳 'US'、開關本來就開）預期無像素差。
- T1：`engine.go` 刪 `ConversionPolicy`／`ConvertAuto`／`ConvertAlways`／`ConvertNever`／`ConversionOverride`／`deriveConversionPolicy`；`convertIfNeeded(data, countries)` 只依偵測；AI 術語校正改用 `!zhtw.IsMainland(countries)`。批次（AC #10）：影集用 `joinCountryCodes(s.ProductionCountries)`、單集用 `seriesCountries`（同一季的單集只查一次）。電影那段改用同一個 `joinCountryCodes`（會順手去掉空白與空碼）。
- T2：`ManageSubtitleDialogV2` 刪政策行、`productionCountry` prop、`Info` import，下載送 `convertToTraditional: true`；`LocalDetailV2` 刪組國家字串那段；spec 換成「簡中軌沒有政策行」「線上下載一律要繁體」，`LocalDetailV2.spec` 刪兩個傳國家的測試與 stub 欄位。
- T3：`SubtitleSearchDialog` 開關一律預設開、刪 `productionCountry` prop；`MediaDetailPanel` 與 gallery 夾具不再傳它。
- T4：`.pen` `v16pVI` 標題改寫、刪示範列 `YE3SJ`；用選單存檔後 grep 磁碟確認新字串落盤、`YE3SJ` 已不在。`sprint-change-proposal-2026-03-24.md` 開頭加取代註記；`project-context.md` 的 CN 規則與「合拍片」那行改成實際行為。
- 驗證：`go build／vet／test ./...` 全綠、`staticcheck-2026.1` 零輸出、`pnpm nx test web`（297 檔／4566 測試）、`pnpm run lint:all`（0 errors、警告數不變、Prettier 過）、`test:cleanup` 無殘留。

### Discovery Triage

- **③** HK／MO 算不算「大陸」——未裁定 → `disc-2026-10-hk-mo-mainland-rule`（建單時立）。
- **③** 合拍片只要國家清單有 CN 就算大陸（`zhtw.IsMainland` 是「任一為 CN」）；`project-context.md` 原本寫「合拍片預設轉」與程式不符 → 已改文件描述實際行為，要不要改規則併入 `disc-2026-10-hk-mo-mainland-rule`。
- **①** 批次找字幕的影集／單集沒帶國家（陸劇被套台灣用語）→ AC #10。

### File List

- apps/api/internal/subtitle/engine.go
- apps/api/internal/subtitle/engine_test.go
- apps/api/internal/subtitle/batch.go
- apps/api/internal/subtitle/batch_test.go
- apps/api/internal/subtitle/request_trigger.go（註解）
- apps/api/internal/handlers/subtitle_handler.go（註解）
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx
- apps/web/src/components/subtitle/SubtitleSearchDialog.tsx
- apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx
- apps/web/src/components/media/LocalDetailV2.tsx
- apps/web/src/components/media/LocalDetailV2.spec.tsx
- apps/web/src/components/media/MediaDetailPanel.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- ux-design.pen（v16pVI）
- project-context.md
- _bmad-output/planning-artifacts/sprint-change-proposal-2026-03-24.md（AC drift reference — see Completion Notes）
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-10-05 建立（Bob create-story，依 Alexyu 裁定 B）＋實作 T1–T5（Opus 5.5 dev-story），status → review。
