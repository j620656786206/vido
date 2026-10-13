# Story golden-v2-exam-with-name-glossary: 200 句考卷附上角色名對照表，考法跟正式版一樣

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-14。來源：spike-local-translation-golden-eval 的發現（disc-2026-10-golden-name-rule-calibration）。
     Alexyu 2026-10-14 在兩個做法中選 A：「考試時附上角色名對照表，跟正式版一樣」，不選 B「放寬人名規則」。
     行號為分支 spike/local-translation-golden-eval d306afae（PR #777，尚未合併；本單的 grade 程式碼依賴它）。 -->

## Story

身為要靠 200 句考卷決定「哪個翻譯模型能用」的 Alexyu，
我要考卷在考試時跟正式版一樣，附上一份角色名對照表，
這樣考出來的是「模型有沒有照對照表翻人名」（使用者實際會遇到的情況），
而不是「沒給對照表時模型自己怎麼音譯」——後者讓連 Claude Sonnet 都被人名規則拖到 19.5% 翻壞。

## 背景（為什麼要做）

- 2026-10-13 第一次正式用 golden-v1 考 Sonnet：200 句裡 38 句被 `name_mismatch` 判 0，翻壞 19.5%，等級只有 B。兩個本機模型也各有 36～44 句同樣的失敗（`eval-local-translation-golden-2026-10.md`「看到的事」第 1 點）。
- 原因有兩個：
  1. 考卷每個人名只收一種譯法（例如 Celia 只收「希莉亞」）。
  2. 評分程式送給翻譯服務的對照表是空的（`cmd/grade/main.go:289`，`TranslateWithGlossary(ctx, blocks, nil, nil, …)` 的第三個參數是 `nil`）。Sonnet 沒有對照表時會把人名留英文，本機模型則每次自己音譯、前後不一。
- 正式版會先用 TMDb 演員表和詞彙表把人名定好，再交給模型（`prompts.BuildGlossarySection`，`internal/ai/prompts/subtitle_translator.go:119-135`：「Glossary — MANDATORY fixed renderings」）。所以考試應該照這個方式考。

## Acceptance Criteria

1. **考試時附上對照表（正式版提示詞模式）。** `cmd/grade` 從考卷每一句的 `names` 欄位（`internal/eval/golden.go:40-43`：`map[原文人名][]可接受的譯法`）建出對照表，用每個人名的**第一個**譯法當標準答案，透過 `TranslateWithGlossary` 的 `glossary` 參數送進正式版提示詞（`[]services.GlossaryPair`，`internal/services/translation_service.go:49-52`）。
   - 對照表以 `source`（`tos`／`sita`／`trap`）為單位，模擬正式版「一部片一份詞彙表」：一批句子只帶它所屬來源的人名。
   - 一批（目前一次 20 句，`main.go:46` 的 `translateChunk`）如果跨兩個來源，就把兩邊的人名合在一起送。dev 也可以改成按來源切批，挑比較簡單的做法並寫測試。
2. **報告記下用了哪份對照表。** `Report.GlossaryVersion` 改成 `subtitle.GlossaryVersionHash(…)`（`internal/subtitle/segment_cache.go:144`），用全部三個來源的對照表合併後的結果算。現在固定是 `""`（`main.go:318`，註解寫「the sample carries no show glossary」）。
3. **考卷版本升成 `golden-v2`。** 把 `eval.SampleVersion` 改成 `"golden-v2"`（`internal/eval/golden.go:24`），定義改成「同一份 200 句，考試時附上 `names` 對照表」。資料檔 `golden-v1.jsonl` 內容不變，不用改名。
   - `--merge` 本來就會拒收版本不同的報告（`cmd/grade/merge.go:28`），所以 v1 的報告不會被誤合併。
   - `internal/eval/golden/README.md` 要補一節「v2 的考法」，說明對照表從哪裡來，以及這會改變 `sita` 來源原本「人名一致性」的考題意義（見 Dev Notes）。
4. **保留舊考法當對照組。** 新旗標 `--no-glossary`：照 v1 的方式考，不送對照表，報告的 `sample_version` 蓋 `golden-v1`、`glossary_version` 留空。用來量「沒有 TMDb 對照表的片子，模型自己能不能前後一致」。`--merge` 必須拒收 `--no-glossary` 的報告（因為版本不同，現有檢查會自動擋下，要寫測試證明）。
5. **本機模型的「模型自己的格式」也帶對照表。** `--prompt-style native` 搭配 `hymt` 模板時，改用 Hy-MT2 官方的「術語」模板：先列「`{原文}` translates to `{譯文}`」，再接翻譯指令（模型卡 Prompt Templates → Terminology，英文版）。只列**這一句和它的上下文裡真的出現的人名**，沒有就照現在的模板。
   - `translategemma` 的官方格式沒有放對照表的位置，所以維持原樣。報告要註明「這個模板不支援對照表」：可以寫在 `JudgeNote` 的附註，或加一個 `glossary_supported=false` 欄位，dev 二選一。
6. **重考兩個 Claude 模型當新基準。** 用 golden-v2 各考一次 `claude-sonnet-5`、`claude-haiku-4-5`，開 `--also-judge simplified_leak,name_mismatch --trace`，`--budget-usd 0.80`（0.50 不夠考完，見 `disc-2026-10-model-grade-budget-and-judge-reliability`）。
   - 報告存到 `eval/golden-v2-2026-10/`。
   - 把分數寫進本 story 的 Completion Notes，並跟 golden-v1 的 Sonnet 結果比較（v1：翻壞 19.5%／自然 76.0%）。
   - **不要**合併進 `model_ratings.json`。那是 `disc-2026-10-model-ratings-judge-mislabeled`（#775）的範圍，要等它一起處理。
7. **測試**：
   - 對照表從 `names` 建出來是對的：每個來源各自建，一個人名有多個譯法時取第一個。
   - 正式版提示詞裡真的出現 `## Glossary — MANDATORY fixed renderings:` 和對應的那幾行。用假的 completer 抓下送出去的提示詞來斷言。
   - `--no-glossary` 時，送出去的提示詞跟改動前一樣，沒有對照表段落，報告版本是 `golden-v1`。
   - native `hymt` 的術語模板只列出這一句真的出現的人名。
   - `--merge` 拒收 `golden-v1`（`--no-glossary`）的報告。
   - 既有測試不改，全部通過。

## Tasks / Subtasks

- [ ] **Task 1 — 從 `names` 建對照表並送進翻譯（AC: #1, #2）**
- [ ] **Task 2 — 考卷版本 v2、`--no-glossary`、README（AC: #3, #4）**
- [ ] **Task 3 — native `hymt` 術語模板（AC: #5）**
- [ ] **Task 4 — 測試（AC: #7）**
- [ ] **Task 5 — 重考 Sonnet 與 Haiku，記錄結果（AC: #6）**

（後端 4、維運 1、前端 0 —— 不觸發拆分。）

## Dev Notes

### 查到的事（2026-10-14，Bob 開檔確認）

- **送對照表的入口已經有了**：`TranslateWithGlossary(ctx, blocks, glossary []GlossaryPair, progressFn, opts...)`（`internal/services/translation_service.go:358`）。`GlossaryPair{Source, Target}` 在 `:49-52`。評分程式現在傳 `nil`。
- **提示詞裡的對照表段落**：`prompts.BuildGlossarySection`（`internal/ai/prompts/subtitle_translator.go:119-135`）。對照表是空的時候回 `""`，所以「不帶對照表」的提示詞跟改動前一個字都不差。這就是 AC #4 的保證來源。
- **對照表版本雜湊**：`subtitle.GlossaryVersionHash`（`internal/subtitle/segment_cache.go:144`）會先複製一份再排序，不會改呼叫端的順序。
- **考卷裡的人名**：21 個人名，分布在 `tos`（Thom、Celia、Barley）、`sita`（Sita、Rama／Ram、Ravana、Hanuman、Lanka…）、`trap`（Alvarez、Sawyer、Elena、Marcus）。目前每個人名都只有一種譯法（2026-10-14 用 python 掃 `golden-v1.jsonl`）。注意 `Ram` 和 `Rama` 都對到「羅摩」。
- **版本檢查**：`ai.ParseRatings`（`internal/ai/ratings.go:88`）要求 golden 列一定要有 `sample_version`（`:123-125`）。`ModelRating` 沒有記對照表的欄位（`ratings.go:29-46`），所以「有沒有帶對照表」只能靠 `SampleVersion` 分辨。這就是 AC #3 選擇升版本、不另加欄位的原因。
- **CI 的 Model Grade workflow** 預設預算是 `'0.50'`（`.github/workflows/model-grade.yml:23-26`）。本單不改它，那是另一張 disc 的事；重考時手動帶 `--budget-usd 0.80`。

### 已知取捨

- **`sita` 來源原本是「人名一致性」的考題**（README 來源表：「美式閒聊＋印度神話人名（人名一致性的考題）」）。帶了對照表之後，它考的變成「模型有沒有照對照表翻」。模型自己能不能前後一致，改由 `--no-glossary` 來量（AC #4）。這是 Alexyu 選 A 時接受的取捨。
- **`trap` 的人名題**（例如 trap-10「Alvarez thinks Sawyer leaked the file」）也會變簡單。這些人名是虛構的，正式版遇到 TMDb 沒有的角色時也不會有對照表。所以 `--no-glossary` 這組分數，對「沒比對到 TMDb 的片子」比較有代表性，報告要兩組並列。
- **對照表只有人名**：考卷沒有地名、術語的對照表，正式版有時會有，這裡不模擬。

### 不要做的事

- 不改考卷內容（200 句、refs、names 都不動），只改考法。
- 不合併進 `model_ratings.json`（#775）。
- 不改 CI workflow 的預算預設值（另一張 disc）。
- 不重考本機模型。2027-04 重考時直接用 v2（`backlog-local-translation-retest-2027-04` 要補一句「用 golden-v2」）。

### 依賴

- **依賴 PR #777**：`--also-judge`、`--rejudge`、`--prompt-style native`、本機模式都在那個 PR 裡。#777 合併之後再開工。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（不動前端）。

### References

- `eval-local-translation-golden-2026-10.md`（發現的來源、分數）
- `sub-7-8a-golden-sample-and-grader.md`（考卷設計與授權）
- `disc-2026-10-golden-name-rule-calibration`（本單取代它）
- Hy-MT2 模型卡 Prompt Templates：<https://huggingface.co/tencent/Hy-MT2-7B-GGUF>

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created

### Discovery Triage

- （dev 填）

### File List
