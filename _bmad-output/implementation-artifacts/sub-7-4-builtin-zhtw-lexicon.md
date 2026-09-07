# Story 7.4: 內建台灣用語詞庫 + OTT 風格在地化程度（後端為主，設定頁一格）

Status: review

## Story

As a Taiwanese viewer,
I want translations that read like Netflix or Apple TV subtitles — 影片 not 視頻, 全聯 not「超市」when the scene calls for it —
so that the output never feels like a machine transliterating a mainland phrasebook.

## Context

party-mode 2026-09-03（Mary／Alexyu）：這是 A 路線裡**唯一不用等累積**的部分，出貨即生效。兩種性質不同的東西，**不要混成一張表**：

| | (a) 查表型詞庫 | (b) OTT 風格在地化 |
| --- | --- | --- |
| 例 | 視頻→影片、質量→品質、信息→資訊、屏幕→螢幕；Life360（被翻成「360 號公路」）、常見品牌／App 名 | "the grocery store" → 全聯／家樂福／喜互惠；"the convenience store" → 超商／小七 |
| 機制 | 後處理 replace（OpenCC s2twp 之後）＋ prompt glossary 注入 | prompt **風格規則 + 範例**，不是查表 |
| 風險 | 低（可逆、可測） | **口味**——有人討厭美劇裡出現全聯 → 必須是**使用者可調的開關** |

## Acceptance Criteria

1. **(a) 詞庫檔。** `apps/api/internal/ai/prompts/lexicon/zh-tw.yaml`：兩段——`replacements`（簡→台，含 OpenCC s2twp 漏掉的常見詞，先收 ≥ 60 筆，來源註明）與 `terms`（品牌／App／機構 ≥ 40 筆，如 Life360、Venmo、Costco、7-Eleven、CVS、DMV、IRS）。embed 進 binary；版本號進 `PromptVersion` 的組成（改詞庫 = 新版本 → cache 語意誠實）。
2. **(a) 注入與後處理。** `terms` 併入 `BuildGlossarySection` 的**全域**段（在 show 詞彙表之前、show 詞彙表優先覆蓋）；`replacements` 在 OpenCC 之後、quality gate 之前做整詞替換（Unicode 邊界，不碰英文）。`GlossaryVersion` 含詞庫版本。
3. **(b) 在地化程度開關。** 設定 `SUBTITLE_LOCALIZATION_LEVEL`（env + settings 表，預設 `standard`）三檔：`literal`（不在地化，超市就是超市）／`standard`（台灣用語，但不換品牌）／`ott`（Netflix／Apple 風格，可用在地品牌與俚語）。prompt 依檔位插入一段風格規則＋ 3–5 個範例（`ott` 檔附「只在場景是日常生活且原文是泛稱時才替換；專有品牌名不換」的護欄）。檔位進 `PromptVersion`。
4. **設定頁。** 字幕設定區一格 radio（三檔，各一句說明＋一個例子）；Rule 21 header 循 `ApiKeysForm` 先例 ride 設定 shell。
5. **評測掛鉤。** `standard` 與 `ott` 各跑 sub-7-8 的黃金樣本一次（$0.05 內），把差異記在 story Completion Notes；**不得**因為分數改預設檔位，那是 Alexyu 的產品裁定。
6. **測試。** 詞庫 YAML schema 驗證測試（重複鍵、空值）；replace 邊界（「視頻」在「電視頻道」裡**不得**被換——需詞邊界或白名單）；三檔 prompt 快照；設定讀寫；FE radio spec。

## Tasks / Subtasks

- [x] **Task 1 — 詞庫檔 + 載入 + 版本（AC: #1）**
  - [x] `prompts/lexicon/zh-tw.yaml`（`version: zh-tw-lex-1`；replacements 109 筆、terms 100 筆，來源註在檔頭）`go:embed` 進 binary，`ParseLexicon` 在 package init 驗證（壞檔直接不能啟動）
  - [x] 版本進 `PromptVersionFor(level)` = `m1-v3+zh-tw-lex-1+<level>`；`SubtitleTranslatorPromptVersion` m1-v2 → m1-v3（P11 pin digest 同步更新、新 surface 進指紋）
- [x] **Task 2 — 注入與後處理（AC: #2）**
  - [x] terms 進 `ComposeInvariantSystemPrompt`（翻譯 prompt + 風格段 + 全域詞彙段），兩條腿 block[0]／system prompt 共用；文字明講「show 詞彙表覆蓋本段」，show 詞彙表照舊在 per-show block
  - [x] replacements 在 OpenCC 之後做詞級替換：extract 腿在 `convertAndStitch`（gate 之後——這條腿本來就是 gate 先、OpenCC 後，見裁量 1）、ASR 腿在 safety net 之後；`production_countries` 含 CN 一律跳過
  - [x] `GlossaryVersionHash` 含詞庫版本（空 show 詞彙表不再是 `""`）
- [x] **Task 3 — 在地化檔位 + prompt 段（AC: #3）**
  - [x] `SUBTITLE_LOCALIZATION_LEVEL` env（開機驗證，打錯直接不啟動）→ settings 表 `subtitle.localization_level` 覆蓋 → 預設 standard；`LocalizationSettingsService` 每次 run 讀（不快取，因為它進 PromptVersion）
  - [x] 三檔各一段風格規則＋範例；`ott` 附「只在日常場景且原文是泛稱才換；專有品牌名不換」護欄
  - [x] `GET/PUT /api/v1/subtitles/localization`（回 level／source／levels，未知值 400）
- [x] **Task 4 — 設定頁一格（AC: #4）**
  - [x] 新頁 `/settings/subtitle`「字幕設定」（側欄媒體庫組，掃描之後）；`LocalizationLevelForm` 三檔 radio 各一句說明＋一個例子，選了就存；來源是 env 時明講「這裡儲存會蓋過」；h1 在 `max-w-3xl` 之外（J7-D spec 加了 subtitle.tsx）
- [x] **Task 5 — 評測與測試（AC: #5, #6）**
  - [ ] AC #5 **延後**（見 Completion Notes）：sub-7-8 的黃金樣本還不存在，本機也沒有 Claude key；預設檔位是 Alexyu 裁定，不由分數改
  - [x] AC #6：詞庫 schema 測試 9 例（缺版本／重複 from／空值／from=to／拉丁 from／except 不含 from／重複 source／未知欄位）；replace 反例表 17 例（「電視頻道」不換、「視訊通話」不換、質量守恆、信息素、大數據、朝鮮半島、最長優先、冪等）；三檔 prompt 段內容測試＋ P11 pin；設定讀寫優先序 6 例＋ Set 驗證；handler GET/PUT；config env；pipeline 4 例（後處理、CN 跳過、檔位進 block[0]、檔位進 run row）；ASR 腿 2 例；FE radio spec 8 例

（後端 4 task、前端 1 —— 不觸發拆分。）

## Dev Notes

- 「視頻／電視頻道」這類誤換是這個 story 最容易翻車的地方；replace 必須是**詞級**不是子字串級，測試要有反例表。
- 詞庫是**跨片**資產（scope=`global:zh-TW` 的概念），但本 story 先做 embed 檔，不進 `show_glossary` 表——sub-7-1 的 scope 命名空間留了位子，將來要讓使用者編輯再搬進 DB。
- PRD 既有規則：大陸出品內容保留簡體不轉換（`production_countries` 含 CN）→ 詞庫替換也要**跳過**這類內容。

### Time-dependent visual coverage

- N/A — 設定頁 radio 無時鐘。

### References

- party-mode 2026-09-03（喜互惠／全聯 討論）；eval-1 零分例 Life360；`subtitle/converter.go`（s2twp）；`prompts/subtitle_translator.go:21-71`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-09-07

### Completion Notes List

- **AC #5 延後，明講原因**：黃金樣本是 sub-7-8 的產出（還是 ready-for-dev），本機沒有 Claude key，NAS 上跑要花錢且沒有可重複的評分器。三檔差異目前只能從 prompt 段本身看（`TestBuildLocalizationSection_ThreeLevels` 釘住各檔的關鍵字：literal 不出現全聯、standard 明寫 never 全聯、ott 有護欄句）。sub-7-8 落地後補跑 standard vs ott 一次，結果記回這裡；預設維持 standard（Alexyu 裁定）。
- **裁量 1（AC #2「OpenCC 之後、quality gate 之前」）**：extract 腿的既有順序是 gate 先、OpenCC 後（`convertAndStitch` 的註解：早轉會把 gate 要抓的簡體漏洞修掉）。詞庫替換是繁→繁，不影響 gate 的兩個檢查（簡體字、英文殘留），所以掛在 `convertAndStitch` 裡 OpenCC 之後、gate 之後。效果同 AC 的意圖（輸出一定經過替換），順序字面上不同。ASR 腿沒有 gate，順序照 AC。
- **裁量 2（全域詞彙段放哪）**：AC 說「併入 BuildGlossarySection 的全域段」。實作把它放進 **block[0]（install-wide 不變前綴）**，不放進 per-show block——這樣 provider 端 prompt cache 的前綴對整台機器所有片都一樣；show 詞彙表照舊在 per-show block，段內文字明講「下方 per-show glossary 覆蓋本段」。沒做「同名時從全域段剔除」——那會讓不變前綴隨片而變，得不償失。
- **裁量 3（「詞邊界」）**：中文沒有詞邊界，用 AC 提到的白名單路線：每條 `from` 可帶 `except`（含它的更長片語），比對時先在原字串標出 except 的範圍，落在範圍內的不換。`ParseLexicon` 拒絕「except 不含 from」的打錯。另外**最長 from 先做**（打印機→印表機 先於 打印→列印）。
- **裁量 4（OpenCC 的 視訊）**：官方 s2twp 把 视频 轉成 視訊；台灣的「視訊」只指 video call，一般「影片」不叫視訊。詞庫加 `視訊→影片`，except 視訊通話／會議／電話／聊天／鏡頭／軟體／面試。
- **詞庫取捨**：刻意不收的高風險詞（記在這裡免得下次又收）：默認（默認了＝承認）、登錄以外的「用戶」（台灣也用）、地鐵（台灣講外國地鐵也用地鐵）、程序／文件（OpenCC 已處理且台灣「程序」＝procedure）、水平（水平線）、一次性（一次性付清）、橡皮（橡皮擦）、幼兒園（台灣現行官方用語）。
- **GlossaryVersion 語意變更**：空 show 詞彙表以前 hash 成 `""`（相容 M1 快取）；現在 hash 詞庫版本。反正 m1-v3 的 prompt bump 已把整庫 re-key，相容性本來就沒了；測試與註解同步改。
- 驗證：`go test ./...` 全綠、`nx lint api` 乾淨；web 340 specs 全綠（settings 目錄）、typecheck 0、eslint 只有既有 warning、prettier 乾淨；`routeTree.gen.ts` 由 build 重生。

### Discovery Triage

- ① 沿用（不立案）：`SettingsService.Set` 對任何 key 都收任何值（generic `POST /settings`）。這個 story 給在地化程度開了專用端點做驗證；若有人用 generic 端點寫壞值，`LocalizationSettingsService.Resolve` 會 warn 並退到 env／預設，不會壞。
- ② 記錄：`gofmt -w internal/ai/prompts/*.go` 會動到三個本來就沒 format 的既有檔（fansub_parser_test、keyword_generator*），已 revert 不帶進本 PR；repo 裡還有一批這種檔（`gofmt -l` 列 20 個），CI 沒擋是因為 lint 只跑 vet+staticcheck。要不要一次清掉是另一件事。

### File List

- apps/api/internal/ai/prompts/lexicon/zh-tw.yaml（新）、lexicon.go（新，+ _test）、localization.go（新）、subtitle_translator.go（+ _test：m1-v3、pin digest）
- apps/api/internal/subtitle/pipeline.go、process_item.go、segment_cache.go（+ 各 _test）、lexicon_pipeline_test.go（新）
- apps/api/internal/services/translation_service.go、transcription_service.go（+ _test）、localization_settings_service.go（新，+ _test）
- apps/api/internal/handlers/localization_handler.go（新，+ _test）
- apps/api/internal/config/config.go（+ _test）
- apps/api/cmd/api/main.go
- apps/web/src/services/subtitleLocalizationService.ts、hooks/useSubtitleLocalization.ts、components/settings/LocalizationLevelForm.tsx（+ .spec）、routes/settings/subtitle.tsx、components/settings/SettingsLayout.tsx（+ .spec）、routes/settings/settings-page-header-width.spec.ts、routeTree.gen.ts
- _bmad-output/implementation-artifacts/sub-7-4-builtin-zhtw-lexicon.md、sprint-status.yaml
