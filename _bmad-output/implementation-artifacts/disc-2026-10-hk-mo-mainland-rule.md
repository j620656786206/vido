# Story: 港澳片也保留原本的說法；「只轉字」真的只轉字

Status: review

**Source:** `disc-2026-10-hk-mo-mainland-rule`（由 `backlog-mainland-rule-three-predicates`／PR #670 立案）。
**⚖️ 裁定：** Alexyu 2026-10-05——「港片、澳門片也保留原本的說法」。延伸同日的 B 裁定（陸劇：轉成繁體字、保留原意，「只是做字的轉換，不轉換成台灣用語」）。

## Story

身為 Vido 的使用者，
我希望陸劇、港片、澳門片的字幕都是繁體字，但台詞的說法保持原樣（不被換成台灣用語），
這樣看港陸片時，字是我習慣的，說法是原本的。

## 查到的事（2026-10-05，Bob 逐一開檔／實測確認）

- **判斷式只認 CN：** `apps/api/internal/zhtw/finalize.go:25-35`（`IsMainland`）。用到它的地方：`zhtw.Finalize`（`finalize.go:50`）、engine 的 AI 術語校正（`apps/api/internal/subtitle/engine.go:199`）。
- **🚨 #670 的「只轉字」其實還會換詞：** 「保留原用語」的片目前跑的仍是 OpenCC `s2twp`（`finalize.go` 的 `conv.ConvertS2TWP`），而 `s2twp` 本身就帶台灣片語表。實測（Python `opencc` 套件，同一份 OpenCC 設定）：

  | profile | 「这个软件的质量很好，视频信息在网络上，出租车」 |
  |---|---|
  | `s2twp` | 這個軟**體**的質量很好，**影片資訊**在**網路**上，**計程車** |
  | `s2tw` | 這個軟件的質量很好，視頻信息在網絡上，出租車 |

  所以 #670 之後，陸劇的「軟件」「視頻」「信息」「出租車」仍被換成台灣說法——只有詞庫那一層（質量→品質）有跳過。裁定是「只做字的轉換」，**保留原用語的片要改用 `s2tw`**（台灣字形、不帶片語）。
- **測試用的假 OpenCC 也有這個盲點：** `apps/api/internal/subtitle/testdata/opencc-helper.sh` 預設分支把「软件→軟體」當成字的轉換，所以 #670 的測試（`engine_test.go`、`subtitle_handler_lexicon_test.go` 期待「這個軟體的質量很好」）沒抓到。
- **profile 怎麼選：** `subtitle.Converter.Convert(content, profile)`（`converter.go:62-95`）非 `s2twp` 時讀 `<設定目錄>/<profile>.json`；正式映像檔整份 `/usr/share/opencc/` 都有（`apps/api/Dockerfile:78`）。測試的 `TestMain`：`subtitle` 套件設 `VIDO_OPENCC_CONFIG=testdata/s2twp.json`（`converter_test.go:28-32`），`handlers` 套件設 `../subtitle/testdata/s2twp.json`（`subtitle_handler_test.go:24-28`）——兩邊都指向 `subtitle/testdata/`，所以只要在那裡放 `s2tw.json` 佔位檔。
- **會傳進 `Finalize` 的三種 converter：** `*subtitle.Converter`（engine、handler、評估工具）、`subtitle.VariantConverter`（`pipeline.go:71-73`）、`services.OpenCCConverter`（`transcription_service.go:79-82`，同時被 `GlossarySeeder` 使用，`glossary_seeder.go:34`）。測試替身：`pipeline_test.go:77`、`zhtw/finalize_test.go:19`、`glossary_seeder_test.go:67`、`transcription_service_test.go:144`、`transcription_translation_resume_test.go:41`。

## Acceptance Criteria

1. **判斷式：** `IsMainland` 改名 `KeepsOwnWording`，CN、HK、MO 都算（去空白、不分大小寫）。所有呼叫端跟著改名。
2. **只轉字：** `Finalize` 對 `KeepsOwnWording` 的片改用 OpenCC `s2tw`（字形轉換、不帶台灣片語），其他片照舊 `s2twp`＋詞庫。`zhtw.Converter` 介面多 `ConvertS2TW`；`*subtitle.Converter` 實作它；`VariantConverter` 與 `services.OpenCCConverter` 介面同步加上；所有測試替身補上。
3. **AI 術語校正：** 港澳片也跳過（跟陸劇一樣）。
4. **測試用假 OpenCC：** 新增 `s2tw` 分支，只換字不換詞；`subtitle/testdata/s2tw.json` 佔位檔。
5. **測試：** `zhtw`：CN／HK／MO 走 `s2tw` 不套詞庫、其他走 `s2twp`＋詞庫；engine、handler、transcription 的陸劇案例期待值改成「這個軟件的質量很好」；HK 案例至少一條路（engine）。
6. **文件：** `project-context.md` CN 規則、記憶 `project_cn_subtitle_policy.md`、設計稿 `v16pVI`（寫明港澳同陸劇、只轉字）；改 `.pen` 後重跑 `scripts/export-pen-screenshots.py`，只提交 `pen-tokens.json` 與真的有設計變動的截圖。
7. `go build／vet／test ./...`、`staticcheck-2026.1`、`pnpm run lint:all`、`python3 scripts/check-design-tokens.py` 全綠。

## Tasks / Subtasks

- [x] T1 `zhtw`：改名、HK／MO、`s2tw`；介面與實作（AC #1、#2）
- [x] T2 engine 術語校正改名；測試替身與假 OpenCC（AC #3、#4）
- [x] T3 測試期待值與新案例（AC #5）
- [x] T4 文件、設計稿、token 快照（AC #6）
- [x] T5 檢查（AC #7）

全部後端＋文件 → 不需要拆單。

## Dev Notes

### 不要做的事

- 合拍片規則（清單裡有 CN 就算）不在這張——未裁定，另立 `disc-2026-10-coproduction-wording-rule`。
- 不碰 AI 翻譯的提示詞（它仍以台灣說法翻譯）；本單只管翻完／下載完之後的轉換。
- 不碰 `GlossarySeeder` 的行為（它照舊用 `s2twp`），只因介面變寬而補替身方法。

### 已知陷阱

- 假 OpenCC 的 case 比對：`s2twp.json` 也包含字串 `s2tw`，分支要比對 `*s2tw.json*`（含 `.json`）。
- 已是繁體的字幕仍不經過 `Finalize`（直接交付），港片繁中字幕原樣出貨。

### Time-dependent visual coverage

N/A — 不碰前端。

### References

- `.claude/memory/project_cn_subtitle_policy.md`
- PR #669（`zhtw`）、PR #670（陸劇轉繁體字）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

### Completion Notes List

- 2026-10-05 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: FOUND — `backlog-mainland-rule-three-predicates` AC #1「陸劇轉成繁體字（OpenCC，不套詞庫）」實際用的是 `s2twp`，仍會換台灣片語 → 改成 `s2tw`（只轉字），這才符合同一個裁定的原話「只是做字的轉換」。期待值「這個軟體的質量很好」→「這個軟件的質量很好」（engine／handler／transcription 測試）。
- 📎 Contract Stamps: NONE (no [@contract-v*] stamps；API 不變)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes；設計稿只改規格註記 `v16pVI` 文字（不在截圖清單）。
- T1：`zhtw.IsMainland` → `KeepsOwnWording`（CN／HK／MO，查表比對）；`Finalize` 對這些片呼叫 `ConvertS2TW`、不套詞庫，其他片 `ConvertS2TWP`＋詞庫。`zhtw.Converter`、`subtitle.VariantConverter`、`services.OpenCCConverter` 都多 `ConvertS2TW`；`*subtitle.Converter.ConvertS2TW` 走 `Convert(…, "s2tw")`（正式映像檔 `/usr/share/opencc/s2tw.json` 隨整份 share 複製進去，`Dockerfile:116`、`apps/api/Dockerfile:78`）。
- T2：engine 術語校正改用 `KeepsOwnWording`；四個測試替身補 `ConvertS2TW`；假 OpenCC 新增 `*s2tw.json*` 分支（只換字）＋ `testdata/s2tw.json` 佔位檔。
- T3：zhtw 測試驗 profile 選擇（一般片 s2twp、CN／HK／MO s2tw）與輸出；engine 加 HK 案例、handler 加 MO（下載）與 HK（轉繁）案例；transcription 測試接上假 OpenCC，模型回「這個软件」→ 一般片「軟體＋品質」、CN／HK「軟件＋質量」。
- T4：`project-context.md`、記憶 `project_cn_subtitle_policy.md`、`.pen` `v16pVI`（選單存檔、grep 磁碟確認）、重跑匯出：只提交 `pen-tokens.json`，213 張重繪截圖沒有設計變動，全部還原。
- 驗證：`go build／vet／test ./...` 全綠、`staticcheck-2026.1` 零輸出、`check-design-tokens.py` 一致、`pnpm run lint:all` 0 errors、`pnpm nx test web` 297 檔／4566 測試。
- 實測依據：Python `opencc` 套件（同一份 OpenCC 設定）`s2twp` 把「这个软件…视频信息…网络…出租车」轉成「軟體…影片資訊…網路…計程車」，`s2tw` 只轉字。

### Discovery Triage

- **①** #670 的「只轉字」其實用 `s2twp`、會換台灣片語 → AC #2。
- **③** 合拍片（清單裡有 CN 就算）→ `disc-2026-10-coproduction-wording-rule`。

### File List

- apps/api/internal/zhtw/finalize.go
- apps/api/internal/zhtw/finalize_test.go
- apps/api/internal/subtitle/converter.go
- apps/api/internal/subtitle/pipeline.go
- apps/api/internal/subtitle/pipeline_test.go
- apps/api/internal/subtitle/engine.go
- apps/api/internal/subtitle/engine_test.go
- apps/api/internal/subtitle/batch.go（註解）
- apps/api/internal/subtitle/testdata/opencc-helper.sh
- apps/api/internal/subtitle/testdata/s2tw.json（新）
- apps/api/internal/services/transcription_service.go
- apps/api/internal/services/transcription_service_test.go
- apps/api/internal/services/transcription_translation_test.go
- apps/api/internal/services/transcription_translation_resume_test.go
- apps/api/internal/services/glossary_seeder_test.go
- apps/api/internal/handlers/subtitle_handler.go（註解）
- apps/api/internal/handlers/subtitle_countries.go（註解）
- apps/api/internal/handlers/subtitle_handler_lexicon_test.go
- ux-design.pen（v16pVI）
- _bmad-output/pen-tokens.json
- project-context.md
- _bmad-output/implementation-artifacts/backlog-mainland-rule-three-predicates.md（AC drift reference — see Completion Notes）
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-10-05 建立（Bob create-story，依 Alexyu 裁定「港澳片也保留原本的說法」）＋實作 T1–T5（Opus 5.5 dev-story），status → review。
