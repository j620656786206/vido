# Disc：連續同句規則不該把「已經被別條規則刪掉的句子」算進連續數

Status: review

**Source:** `disc-2026-10-asr-repeated-lines-dropped` 的 CR L1／L2；2026-10-07 實測看到配樂幻聽每 30 秒一句、剛好 4 句時留下來（`eval-see-s01e02-asr-vs-official.md` 第三次實測「新映像・人名提示＋字級時間」那列：`repeat_run:20`，門檻 5 讓 4 句假台詞留下來）。

## Story

身為只能靠「聽聲音」生成字幕的使用者，
我要 Vido 刪掉配樂被聽成的「每 30 秒一句、一模一樣」的假台詞，同時不要把真的連喊幾聲的台詞誤刪，
這樣字幕裡不會出現沒人講的句子，也不會少掉真的對白。

## 問題

- R2b（連續同句）用文字算連續，已被 R0（music_only）／R1（silence）／R2（repetition）標掉的同文字段落也算在內；4 句真喊＋1 句被標 silence 的同句 → 5 連 → 3 句真喊被當 `repeat_run` 刪掉。
- `filterPromptEcho` 在 `filterHallucinations` **之前把段落移除**，兩段喊叫中間若夾一句名單回音，移掉後接成一條。（人名提示 2026-10-07 起預設關，但開關還在。）
- 配樂幻聽是「每 30 秒一句、文字一模一樣」（Whisper 一窗一句）；真的喊叫差幾秒（「Face me!」×2、「轉過來看我」×3 在 16 秒內）。**時間間隔可當判準。**

## Acceptance Criteria

1. **名單回音不再先移除**：名單回音（`prompt_echo`）改在幻聽過濾裡跟其他規則一起「標記」，段落留在原位；被標的句子仍然不會出現在字幕裡，log 的刪除原因仍是 `prompt_echo`。兩組同句喊叫中間夾一句回音時，不會被接成一條長串。
2. **連續同句只數沒被標的句子**：一串同文字的段落裡，已被別條規則標掉的不算數。4 句真喊＋1 句同文字但被判 silence → 只算 4 句，不觸發 `repeat_run`（4 句真喊全留）。5 句以上沒被標的同句仍照舊只留第一句。
3. **隔很久的重複是迴圈**：同文字、相鄰兩句開口時間都相隔 ≥ 20 秒、且（沒被標的）達 3 句以上 → 只留第一句，其餘刪除原因 `repeat_spaced`（新的穩定字串，log 分得出是哪條規則）。
   - 「Thank you.」每 30 秒一句共 4 句 → 留 1 句。
   - 只有 2 句、相隔 25 秒（例「Yes.」兩次）→ 都留（**刻意比原草稿的「兩句起」保守**：短回答隔半分鐘再說一次很常見，2 句的漏網代價只是多 1 句假台詞）。
   - 一串裡只要有一個間隔 < 20 秒（真的連喊）→ 不套這條（仍可能被 AC #2 的 5 句規則抓到）。
4. **既有行為不變**：music_only／silence／repetition／tail 的判斷與現有測試全部照舊；「Face me!」×2、「轉過來看我」×3 這類真連喊仍保留。
5. **夾具**：用「配樂幻聽每 30 秒一句」的形狀（模仿 10 分鐘片段 run1-prompt：`repeat_run:20` 那串＋剛好 4 句留下的那串）寫測試；NAS 的原始 SRT 在 `/mnt/cache/appdata/vido-eval-clip/`，repo 裡沒有，所以用合成段落重現形狀。
6. **檢查**：`go build／vet`、`go test ./...`、`staticcheck` 全綠。

## Tasks / Subtasks

- [x] T1（AC #1）：`filterPromptEcho` 改成判斷函式，由 `filterHallucinationsWith` 接 prompt 參數、在 R0 前先標 `prompt_echo`；`whisper.go` 呼叫端改一次呼叫。
- [x] T2（AC #2）：R2b 數連續時只數 `reasons[k] == ""` 的段落。
- [x] T3（AC #3）：新增 `repeat_spaced`：常數 `hallucinationSpacedRepeatGapSeconds = 20`、`hallucinationSpacedRepeatRun = 3`。
- [x] T4（AC #1–#5）：測試——回音不再橋接、被標的同句不算數、30 秒一句 4 句被刪、2 句隔 25 秒保留、間隔混合不套、原回音測試改走新入口。
- [x] T5（AC #6）：全套檢查。

## Dev Notes

- 檔案：`apps/api/internal/ai/whisper_segments.go`（規則）、`apps/api/internal/ai/whisper.go`（呼叫端，約 366 行）、`whisper_segments_test.go`、`asr_prompt_test.go`。
- 分段跑（chunk-at-silence）時每段各自過濾，時間是段內時間；跨段的迴圈不會被串起來——可接受，配樂段多半被 music_only 整段丟掉。
- 「間隔」量的是相鄰兩個（沒被標的）同句的 `Start` 差。
- 不動門檻 5（`hallucinationRepeatRun`）。

## Dev Agent Record

**Agent:** Opus 5.5（create-story 補 AC／dev／自審）；CR：Sonnet（換模型慣例）。

### Completion Notes

- `filterPromptEcho` → `promptEchoMatcher`（判斷函式）；`filterHallucinationsWith(segs, applyTail, prompt)` 在 R0 前就地標 `prompt_echo`；`whisper.go` 改一次呼叫（AC #1）。
- R2b 抽成 `markRepeatRun`：一串同文字裡只數沒被標的；≥5 → `repeat_run`；≥3 且相鄰（沒被標的）開口間隔全 ≥20 秒 → `repeat_spaced`（AC #2、#3）。
- 新測試 13 個（含 CR 補的 6 個）；原回音測試改走新入口（AC #4、#5）。
- 檢查：`go build／vet` 綠；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅——在 `main` 一樣紅，是雲端容器以 root 執行、模擬不了「沒有寫入權限」（CI 非 root）；`staticcheck 2026.1 ./...` 0 項（AC #6）。

### CR（Sonnet，0H／1M／4L，全修）

- **M**：回音留在原位後會參與片尾規則（R3）的倒數——結尾一句回音會擋住片尾 3 句假台詞被刪，或把 2 句湊成 3 句。修：片尾倒數把 `prompt_echo` 當透明（不擋、不算數），等同舊的「先移除」；補測試。
- **L**：R2b 註解改寫（同文字被標的不斷串、不同文字的回音會斷串）；常數註解講清楚分段跑時跨切點的迴圈抓不到；補 19.9 秒邊界、5 句間隔大報 `repeat_run`、第一句被標時留第一個沒被標的、`applyTail=false` 仍套用等測試。
- **未改**：`dropped` 的順序從「回音先」變成依時間——只有 log 逐筆印與依原因計數，沒有消費者依賴順序。

### File List

- `apps/api/internal/ai/whisper_segments.go`
- `apps/api/internal/ai/whisper.go`
- `apps/api/internal/ai/whisper_segments_test.go`
- `apps/api/internal/ai/asr_prompt_test.go`
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-repeat-run-bridging.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
