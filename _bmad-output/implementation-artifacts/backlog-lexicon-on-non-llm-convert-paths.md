# Story: 每一條「轉成繁體」的路都套台灣用語——收斂成同一個收尾步驟

Status: review

**Source:** `backlog-lexicon-on-non-llm-convert-paths`（sub-7-4 CR 抓到，2026-09-07）。Alexyu 2026-10-05 從候選中選 1。
**相關：** `backlog-mainland-rule-three-predicates`（本單吸收其中「判斷式收斂」那一半；「大陸內容的字幕到底要不要轉繁體」那一半是產品裁定，留在原單，見 Discovery Triage）。

## Story

身為 Vido 的使用者（台灣觀眾），
我希望不管字幕是 AI 翻的、片子內嵌轉出來的、還是線上下載後轉繁體的，用語都一樣是台灣的（品質、軟體、影片），
這樣同一部片換一條路拿字幕，不會一下「品質」一下「質量」。

## 查到的事（2026-10-05，Bob 逐一開檔確認）

**「先轉字（OpenCC s2twp）再換台灣用語（詞庫）」現在散在 7 個地方，各寫各的：**

| # | 路徑 | 位置 | 有沒有套詞庫 |
|---|---|---|---|
| 1 | 內嵌簡體軌轉繁後直接交付 | `apps/api/internal/subtitle/process_item.go:375-383` | ✅ 有（`lexiconFor`） |
| 2 | 翻譯結果最後一關 | `apps/api/internal/subtitle/pipeline.go:1270-1293` | ✅ 有 |
| 3 | 翻譯時收成的詞彙 | `apps/api/internal/subtitle/process_item.go:1009-1015` | ✅ 有 |
| 4 | 語音辨識翻譯（舊路線） | `apps/api/internal/services/transcription_service.go:1970-1987` | ✅ 有（自己再寫一次 `IsMainlandContent`） |
| 5 | 舊路線收成詞彙 | `apps/api/internal/services/transcription_service.go:2037-2050` | ✅ 有 |
| 6 | **自動找線上字幕**（下載完成觸發＋批次） | `apps/api/internal/subtitle/engine.go:325-353`（`convertIfNeeded`） | ❌ **沒有** |
| 7 | **手動下載線上字幕**（開關「轉繁體」） | `apps/api/internal/handlers/subtitle_handler.go:259-271` | ❌ **沒有** |
| 8 | 手動「轉繁體」API | `apps/api/internal/handlers/subtitle_handler.go:426` | ❌ **沒有** |

- 另外兩個評估工具也各抄了一份同樣的收尾：`apps/api/cmd/grade/main.go:241-260`、`apps/api/internal/preview/service.go:309-321`。
- **#6、#7 是使用者真的會走到的路。** #6：`request_trigger.go:121-139`（下載完成自動找字幕）與 `batch.go:545-551`（批次）都呼叫 `Engine.Process`。#7：前端 `SubtitleSearchDialog.tsx:62,128` 與 `ManageSubtitleDialogV2.tsx:440` 送 `convert_to_traditional`。
- **#8 目前沒有前端呼叫**（全 repo grep `subtitles/convert`：只有 `ManageSubtitleDialogV2.tsx:40` 的註解，說明「no client wires it yet」，已立 `disc-2026-09-dialog-track-convert-not-wired`）。單子原文說「走手動轉繁得到 質量」——那條路使用者現在按不到；仍要修，因為那張單接上前端時會直接繼承這裡的行為。
- **「是不是大陸片」有兩種寫法：** `engine.go:155-163` 用 `strings.Contains(ProductionCountry, "CN")`（呼叫端先把國碼用逗號串起來，`request_trigger.go:123-129`、`batch.go:545-551`）；其餘用 `prompts.IsMainlandContent`（`apps/api/internal/ai/prompts/lexicon.go:216-229`，逐個比對、不分大小寫）。兩者對 ISO 二碼結果相同，但是兩份定義。
- 手動兩條路（#7、#8）拿不到國家：`SubtitleHandler` 只有 `SubtitleStatusUpdater`（`engine.go:22-24`，只有 `UpdateSubtitleStatus`），`main.go:1063-1066` 傳進去的是 `repos.Movies`、`repos.Series`（有 `FindByID`）。請求都帶 `media_id`＋`media_type`（`subtitle_handler.go:112-121,131-136`）。`Movie.ProductionCountries` 讀出時就填好（`models/movie.go:293-295`）。
- **套件邊界：** `subtitle` import `services`，`services` 不得 import `subtitle`（`apps/api/internal/boundaries_test.go:24-31`，Rule 19）。兩邊共用的東西前例是 `internal/segkey`，有專屬測試釘住它只能依賴 `models`、`ai/prompts`（`boundaries_test.go:110-150`）。`ai/prompts` 沒有任何 internal 依賴（`go list -deps` 已確認）。
- 詞庫 `Apply` 對 nil Lexicon、空字串、沒有漢字的字串原樣返回（`lexicon.go:137-148`）。

## Acceptance Criteria

1. **一個收尾函式，一個大陸判斷。** 新增 `apps/api/internal/zhtw` 套件，提供：
   - `IsMainland(countries []string) bool`——取代 `prompts.IsMainlandContent`（行為不變：去空白、不分大小寫比 `CN`）。
   - `Finalize(conv, text string, countries []string) (string, error)`——先 OpenCC s2twp（`conv` 為 nil 時略過），再套台灣詞庫（大陸片略過詞庫）。OpenCC 失敗時：回傳「原文經過詞庫」的結果＋錯誤，由呼叫端決定要不要用。`conv` 用最小介面（只要 `ConvertS2TWP([]byte) ([]byte, error)`）。
   - `boundaries_test.go` 比照 segkey 加一支測試：`zhtw` 只准依賴 `ai/prompts`。
2. **8 條路全部改呼叫 `zhtw.Finalize`**（表格 #1–#8），刪掉各自的 `lexiconFor`／手寫順序。每條路原本的「失敗時怎麼辦」不變：
   - #1、#8：OpenCC 失敗＝這次失敗（照舊）。
   - #2、#3、#4、#5：OpenCC 失敗＝照舊交付（文字已過品質關或是 LLM 輸出）。
   - #6、#7：OpenCC 失敗＝交付原始下載內容、語言標記不改成繁體（照舊；不可把沒轉成功的簡體標成 zh-Hant）。
3. **#6 自動找線上字幕：** 非大陸片的簡體字幕轉繁後套詞庫。大陸片照舊完全不轉（`ConvertNever`，簡體保留）。原本已是繁體的字幕照舊不動（不套詞庫——與 #1 的「已是繁體直接交付」一致）。`deriveConversionPolicy` 改用 `zhtw.IsMainland`（呼叫端仍可傳逗號串，函式內拆開；或改傳 `[]string`，dev 擇一並寫在 Completion Notes）。
4. **#7 手動下載：** 使用者開了「轉繁體」才轉；轉的時候，非大陸片套詞庫、大陸片只轉字不換詞（與 #1–#5 的大陸規則一致）。國家從 `media_id`＋`media_type` 查；查不到（找不到片、DB 錯）就當作非大陸片＋Warn log——不可讓下載失敗。
5. **#8 手動轉繁 API：** 同 #7 的國家規則（OpenCC 照舊一律做，因為按下去就是使用者的意思——`subtitle_handler.go:349-352` 的既有說明保留；只有「換詞」看國家）。
6. **評估工具一起收：** `cmd/grade/main.go` 與 `internal/preview/service.go` 的收尾改用 `zhtw.Finalize`（countries 傳 nil，行為不變）。
7. **測試：**
   - `zhtw`：轉字＋換詞的順序（`这个软件的质量很好` → 含「軟體」「品質」）；大陸片只轉字（含「質量」）；`conv` nil 只換詞；OpenCC 失敗回傳錯誤且結果是「原文經過詞庫」；`IsMainland` 搬過去的案例。
   - Engine：非大陸簡體 → 交付內容含「品質」；大陸片 → 原簡體不動；繁體輸入 → 不動。
   - Handler：#7 開轉繁＋非大陸 → 含「品質」；大陸 → 「質量」；國家查詢失敗 → 仍成功且套詞庫。#8 同兩案。
   - 既有 pipeline／transcription 測試全綠（行為不該變）。
8. `go build ./...`、`go vet ./...`、`go test ./...`、`~/go/bin/staticcheck ./...` 全綠（staticcheck 要跑：刪 `lexiconFor`／`IsMainlandContent` 後容易留 U1000）。

## Tasks / Subtasks

- [x] T1 新增 `internal/zhtw`（AC #1）＋測試＋boundaries 測試
- [x] T2 subtitle 套件：#1 #2 #3 改用 `zhtw.Finalize`，刪 `lexiconFor`；#6 engine 套詞庫＋`deriveConversionPolicy` 改用 `zhtw.IsMainland`（AC #2、#3）
- [x] T3 services：#4 #5 改用 `zhtw.Finalize`；刪 `prompts.IsMainlandContent`（先 grep 全 repo 確認沒有其他呼叫者）（AC #2）
- [x] T4 handlers：`SubtitleHandler` 加國家查詢（建議 setter，如 `SetCountryResolver`，避免改動十幾處 `NewSubtitleHandler(...)` 測試呼叫）；`main.go` 接上；#7 #8 改用 `zhtw.Finalize`（AC #4、#5）
- [x] T5 評估工具改用 `zhtw.Finalize`（AC #6）
- [x] T6 測試與檢查（AC #7、#8）

全部是後端（前端 0 項）→ 不需要拆單。

## Dev Notes

### 不要做的事

- **不要改「大陸片要不要轉繁體」的行為。** 今天 #6 大陸片完全不轉、#1（內嵌簡體軌）大陸片照樣轉字只是不換詞——這個不一致是 `backlog-mainland-rule-three-predicates` 的產品問題，要 Alexyu 裁定，本單不碰。
- **不要把 HK／MO 加進大陸判斷。** 單子提過，但沒人裁定過；行為維持「只有 CN」。
- **不要讓已是繁體的字幕過詞庫**（#1 直接交付、#6 繁體分支都不套）。改了會動到所有官方繁中字幕，範圍外。
- 不改前端、不改 API 契約（請求／回應欄位都不變）。

### 已知陷阱

- `transcription_service.go` 的 OpenCC 是 `services.OpenCCConverter`（`transcription_service.go:79-82`，含 `IsAvailable`）；engine／handler 是 `*subtitle.Converter`；pipeline 是 `VariantConverter`（`pipeline.go:304`）。`zhtw` 的介面只要 `ConvertS2TWP`，三種都能直接傳；「不可用就當 nil」的判斷留在呼叫端。
- `Lexicon` 的版本會進 PromptVersion／GlossaryVersion（`lexicon.go:25-26`），本單不改詞庫內容，所以不會讓翻譯快取失效。
- `convertAndStitch`（#2）是逐句呼叫；改用 `Finalize` 時保持逐句，不要改成整段一次（失敗計數依賴逐句）。

### Time-dependent visual coverage

N/A — 不碰前端元件。

### References

- `_bmad-output/implementation-artifacts/sub-7-4-builtin-zhtw-lexicon.md`（詞庫由來）
- `_bmad-output/planning-artifacts/sprint-change-proposal-2026-03-24.md:116,131-132`（大陸片保留簡體的原始規則）
- project-context.md Rule 19（套件邊界）、Rule 24（Discovery Triage）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

### Completion Notes List

- 2026-10-05 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: NONE (checked: 'IsMainlandContent|lexiconFor|ZhTWLexicon().Apply' across _bmad-output/implementation-artifacts/*.md — 3 hits: sub-7-4、sub-7-2b、本單；全是 REUSE：sub-7-4 AC #2 的「OpenCC 之後套詞庫、大陸片略過」原封不動，只是多套到三條路；tests/e2e 與 routes/test 無相關斷言)
- 📎 Contract Stamps: NONE (no [@contract-v*] stamps in this story or sub-7-4 — API 請求／回應欄位都沒變)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- T1：新 `internal/zhtw`（`Finalize`、`IsMainland`、`Converter` 介面）。OpenCC 失敗時回「原文經過詞庫」＋錯誤——這正是 #2、#3 原本的行為（失敗時仍套詞庫），所以不需要第二個函式。`boundaries_test.go` 加 `TestZhtwDependsOnlyOnPrompts`。
- T2：#1 #2 #3 改呼叫 `zhtw.Finalize`，刪 `lexiconFor`。#6 engine：`convertIfNeeded` 多收 countries，轉繁走 `zhtw.Finalize`；`deriveConversionPolicy` 改用 `zhtw.IsMainland`。**AC #3 的二選一：保留逗號串**（`ProcessOptions.ProductionCountry` 不變，engine 內用 `productionCountries` 拆開），因為 request_trigger／batch 兩個呼叫端與 `BatchItem` 都用字串，改型別會擴散到批次那一層。附帶行為差異：舊的 `strings.Contains(…, "CN")` 分大小寫，`"cn"` 不算大陸；現在不分大小寫——國碼來自 TMDb，一律大寫，實際上不會遇到。
- T3：#4 #5 改呼叫 `zhtw.Finalize`（新 `availableOpenCC()` 把「沒裝 OpenCC」轉成 nil）；刪 `prompts.IsMainlandContent` 與其測試（全 repo grep 已無呼叫者，案例搬到 `zhtw`）。
- T4：`SubtitleHandler.SetCountryResolver` ＋ `NewRepoCountryResolver(repos.Movies, repos.Series)`（新檔 `subtitle_countries.go`），`main.go` 接上。查不到國家（錯誤）→ Warn＋當非大陸片，下載照常成功。#7 #8 改呼叫 `zhtw.Finalize`。
- T5：`cmd/grade`、`internal/preview` 改用 `zhtw.Finalize`（countries nil）。用 `zhtw.Converter` 介面變數承接，避免把 typed-nil `*subtitle.Converter` 塞進介面。
- 測試替身：`subtitle/testdata/opencc-helper.sh`（假 OpenCC）多兩條 `个→個`、`质量→質量`，讓測試句「这个软件的质量很好」能走完「轉字→換詞」。
- /ship 對抗式自審（2026-10-05）：主要缺口是 #4 語音辨識翻譯那條路沒有任何測試證明它有套詞庫、大陸片有跳過——補 `TestTranslateSRT_TaiwanVocabularyWithMainlandExemption`（DE→品質、CN→質量）。其餘檢查都過：三種 converter 都不會以 typed-nil 進 `Finalize`（pipeline 建構時保證非 nil；engine／handler 先檢查 `IsAvailable`；評估工具用介面變數）；OpenCC 失敗時各路的原行為一一對照不變；影集讀出時有填 `ProductionCountries`（`series_repository.go:761`）。已知：找不到片時 repo 回的是包著 `sql.ErrNoRows` 的錯誤，所以手動下載一部 DB 沒有的片會留一行 Warn——行為正確（當非大陸片），只是 log 多一行，不改。
- 驗證：`go build ./...`、`go vet ./...`、`go test ./...`（全綠）、`staticcheck-2026.1 ./...`（零輸出）、`pnpm nx test web`（297 檔／4569 測試全綠，前端無改動）；`test:cleanup` 無殘留。

### Discovery Triage

- **③ backlog-with-carry-forward-link** — 「大陸片的字幕要不要轉成繁體」三條路做法不同（自動找字幕：不轉；內嵌軌與 AI 翻譯：轉字不換詞），還有 HK／MO 要不要算。→ 留在 `backlog-mainland-rule-three-predicates`，該條目補記「判斷式已由本單收斂，剩產品裁定」。
- **③** — 手動轉繁 API 沒有前端呼叫 → 已有 `disc-2026-09-dialog-track-convert-not-wired`，不另立。

### File List

- apps/api/internal/zhtw/finalize.go（新）
- apps/api/internal/zhtw/finalize_test.go（新）
- apps/api/internal/boundaries_test.go
- apps/api/internal/ai/prompts/lexicon.go
- apps/api/internal/ai/prompts/lexicon_test.go
- apps/api/internal/subtitle/engine.go
- apps/api/internal/subtitle/engine_test.go
- apps/api/internal/subtitle/pipeline.go
- apps/api/internal/subtitle/process_item.go
- apps/api/internal/subtitle/media_store.go（註解）
- apps/api/internal/subtitle/media_store_test.go（註解）
- apps/api/internal/subtitle/testdata/opencc-helper.sh
- apps/api/internal/services/transcription_service.go
- apps/api/internal/services/transcription_translation_test.go
- apps/api/internal/handlers/subtitle_handler.go
- apps/api/internal/handlers/subtitle_countries.go（新）
- apps/api/internal/handlers/subtitle_handler_lexicon_test.go（新）
- apps/api/internal/preview/service.go
- apps/api/cmd/grade/main.go
- apps/api/cmd/api/main.go
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-10-05 建立（Bob create-story）＋實作 T1–T6（Opus 5.5 dev-story），status → review。
