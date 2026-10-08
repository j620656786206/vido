# Disc：英文的 you 看不出男女時，翻譯一律用「你」，確定是女性才用「妳」

Status: done

**Source:** `eval-see-s01e02-asr-vs-official.md` 討論會清單第 7 項（2026-10-06 See S01E02 party mode）；⚖️ 規則已定：「不確定是男是女就用『你』，只有確定是女性才用『妳』」。⚖️ 2026-10-08 Alexyu 選 A 開工。

## Story

身為看 AI 字幕的使用者，
我要翻譯在看不出對方是男是女時用「你」，
這樣對男性說的話不會被翻成「妳」（See S01E02 第一次實測 8 次、第二次 12 次，例：約 23:44 *you, of all people, might know them* →「妳大概是最有可能認識他們的人」，官方用「你」）。

## Acceptance Criteria

1. 翻譯的系統提示多一條規則：第二人稱預設用「你」（複數「你們」）；只有**字幕本身**（這批或前面送去當參考的幾句）看得出對方是女性時才用「妳」（例：稱呼 ma'am／Mom／sister／girl、叫女性名字，或用 she／her 指稱**被說話的那個人**）。只靠角色名單、劇情猜、語氣猜都不算確定。「妳們」只在每個被說話的人都看得出是女性時用。「您」照原文的禮貌程度用；第三人稱照英文（he → 他、she → 她）。
2. 規則附一組正確／錯誤例子（沿用「逐句對齊」那節的寫法），錯誤例就用 See 那句。
3. 提示版本 `SubtitleTranslatorPromptVersion` 從 `m1-v4` 升到 `m1-v5`，`PinsPromptText` 的指紋同步更新（P11：改提示不升版，重跑會拿到舊譯文）。升版的後果：已快取的逐句翻譯換新鍵，之後重跑或續跑會重新翻（同 m1-v2／v3／v4 的前例）；預覽快取與 Claude 的提示快取前綴也一起換新。已翻好的字幕檔不會自動重翻，只有之後重新生成的才會套用新規則。
4. 測試：系統提示含這條規則的關鍵字（「你」預設、「妳」只在確定時）。
5. 不做：用聽障版字幕的說話者標記（`BABA VOSS:`）判斷性別——story 原文說先不做。
6. 檢查：`go build／vet`、`go test ./...`、`staticcheck` 全綠。

## Tasks / Subtasks

- [x] T1（AC #1、#2）：`apps/api/internal/ai/prompts/subtitle_translator.go` 加 rule 8 與例子。
- [x] T2（AC #3）：升版、版本歷史註解、更新指紋。
- [x] T3（AC #4）：測試。
- [x] T4（AC #6）：全套檢查。

## Dev Notes

- 所有翻譯（抽字幕再翻、聽聲音再翻、批次、單集）都走同一份 `SubtitleTranslatorSystemPrompt`（`localization.go` 的 `ComposeSystemPrompt` 把它接在最前面），只改一處。
- 效果要靠 NAS 實測量：See S01E02 重新生成後數「妳」的次數（目前 12），門檻：對男性說話的「妳」= 0。這是 prompt 規則，不是保證，所以量測寫在交付後待辦。

## Dev Agent Record

**Agent:** Opus 5.5（create-story／dev）；CR：Sonnet（換模型慣例）。

### Completion Notes

- 系統提示加 rule 8＋三組例子（See 那句的正確／錯誤、「Mom」在前一句當證據的跨句例），格式沿用逐句對齊那節（AC #1、#2）。
- `m1-v4` → `m1-v5`，版本歷史註解、指紋、三個釘版本字串的測試同步（AC #3）。
- 新測試 `TestSubtitleTranslatorSystemPrompt_SecondPersonDefaultsToNi`（含 rule 8 在「逐句對齊」之前）（AC #4）。
- 檢查：`go build／vet` 綠；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅（main 一樣紅，雲端容器 root 執行）；`staticcheck 2026.1` 0 項（AC #6）。

### CR（Sonnet，0H／4M／5L）

- M1／M2：只看「同一句」會讓「妳」幾乎用不到，也沒說參考句算不算 → 改成「這批或參考句裡任何一句都算」，加跨句正確例。
- M3：補「妳們」規則。M4：she／her 要指被說話的人；第三人稱照英文。
- L1：錯誤例改成 `[1] …` 格式。L2：測試補「rule 8 在逐句對齊之前」。L3：story 補提示快取／預覽快取也換新。L4：`quality_gate.go` 註解不再寫死 m1-v4。
- L5（未做）：評分工具沒有「妳」的自動指標——量測照 Dev Notes 放到 NAS 實測。

### 交付後待辦

- NAS 重新生成 See S01E02，數對男性說話的「妳」（目前 12 次），門檻 0；也看「妳」有沒有被壓到該用時不用（例：對母親、對女性角色）。

### File List

- `apps/api/internal/ai/prompts/subtitle_translator.go`
- `apps/api/internal/ai/prompts/subtitle_translator_test.go`
- `apps/api/internal/ai/prompts/lexicon_test.go`
- `apps/api/internal/services/transcription_translation_test.go`
- `apps/api/internal/subtitle/lexicon_pipeline_test.go`
- `apps/api/internal/subtitle/quality_gate.go`
- `_bmad-output/implementation-artifacts/disc-2026-10-translation-gender-default.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
